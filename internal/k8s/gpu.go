package k8s

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NodeClass represents the accelerator type of a node.
type NodeClass string

const (
	NodeClassGPU_NVIDIA NodeClass = "gpu/nvidia"
	NodeClassGPU_Neuron NodeClass = "gpu/neuron"
	NodeClassCPU        NodeClass = "cpu"
)

// GPU resource keys checked in priority order.
const (
	NvidiaGPUResource  = "nvidia.com/gpu"
	NeuronCoreResource = "aws.amazon.com/neuroncore"
)

// Common node labels for GPU model and instance type.
const (
	LabelGPUProduct   = "nvidia.com/gpu.product"
	LabelInstanceType = "node.kubernetes.io/instance-type"
)

// ClassifyNode determines the accelerator type of a node from its capacity.
func ClassifyNode(node *unstructured.Unstructured) NodeClass {
	capacity := getNestedMap(node, "status", "capacity")
	if capacity == nil {
		capacity = getNestedMap(node, "status", "allocatable")
	}
	if capacity == nil {
		return NodeClassCPU
	}

	if count, ok := capacity[NvidiaGPUResource]; ok {
		if toInt(count) > 0 {
			return NodeClassGPU_NVIDIA
		}
	}

	if count, ok := capacity[NeuronCoreResource]; ok {
		if toInt(count) > 0 {
			return NodeClassGPU_Neuron
		}
	}

	return NodeClassCPU
}

// GPUCapacity returns the total GPU count from node capacity/allocatable.
func GPUCapacity(node *unstructured.Unstructured) int {
	// Try allocatable first — it reflects what's actually schedulable
	for _, field := range []string{"allocatable", "capacity"} {
		m := getNestedMap(node, "status", field)
		if m == nil {
			continue
		}
		if v, ok := m[NvidiaGPUResource]; ok {
			if n := toInt(v); n > 0 {
				return n
			}
		}
		if v, ok := m[NeuronCoreResource]; ok {
			if n := toInt(v); n > 0 {
				return n
			}
		}
	}
	return 0
}

// GPUModel returns the GPU product name from node labels.
func GPUModel(node *unstructured.Unstructured) string {
	labels := getNestedMap(node, "metadata", "labels")
	if labels == nil {
		return ""
	}
	if model, ok := labels[LabelGPUProduct]; ok {
		if s, ok := model.(string); ok {
			return s
		}
	}
	return ""
}

// InstanceType returns the instance type from node labels.
func InstanceType(node *unstructured.Unstructured) string {
	labels := getNestedMap(node, "metadata", "labels")
	if labels == nil {
		return ""
	}
	if it, ok := labels[LabelInstanceType]; ok {
		if s, ok := it.(string); ok {
			return s
		}
	}
	return ""
}

// NodeReadyStatus returns "Ready" or "NotReady" from node conditions.
func NodeReadyStatus(node *unstructured.Unstructured) string {
	conditions, found, err := unstructured.NestedSlice(node.Object, "status", "conditions")
	if err != nil || !found {
		return "Unknown"
	}
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == "Ready" {
			if cond["status"] == "True" {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

// AllocatableCPU returns the allocatable CPU in millicores.
func AllocatableCPU(node *unstructured.Unstructured) int64 {
	m := getNestedMap(node, "status", "allocatable")
	if m == nil {
		return 0
	}
	if v, ok := m["cpu"]; ok {
		return parseCPU(v)
	}
	return 0
}

// GPUDisplayString returns a formatted string like "8× H100" for GPU nodes.
func GPUDisplayString(node *unstructured.Unstructured) string {
	count := GPUCapacity(node)
	if count == 0 {
		return "—"
	}
	model := GPUModel(node)
	if model == "" {
		return fmt.Sprintf("%d×", count)
	}
	return fmt.Sprintf("%d× %s", count, model)
}

// --- helpers ---

func getNestedMap(obj *unstructured.Unstructured, fields ...string) map[string]interface{} {
	m, found, err := unstructured.NestedMap(obj.Object, fields...)
	if err != nil || !found {
		return nil
	}
	return m
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	}
	return 0
}

func parseCPU(v interface{}) int64 {
	s, ok := v.(string)
	if !ok {
		return 0
	}
	// Handle millicores like "4000m"
	if len(s) > 0 && s[len(s)-1] == 'm' {
		n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	// Handle whole cores like "4"
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n * 1000
}

// PodGPURequests returns the total nvidia.com/gpu count requested by all
// containers in the given pod. Returns 0 if no GPU requests are found.
func PodGPURequests(pod *unstructured.Unstructured) int {
	containers, found, err := unstructured.NestedSlice(pod.Object, "spec", "containers")
	if err != nil || !found {
		return 0
	}
	total := 0
	for _, c := range containers {
		ctr, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		res, ok := ctr["resources"].(map[string]interface{})
		if !ok {
			continue
		}
		req, ok := res["requests"].(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := req[NvidiaGPUResource]; ok {
			total += toInt(v)
		}
	}
	return total
}

// PodCPURequests returns the total CPU millicores requested by all containers
// in the given pod. Uses the existing parseCPU helper for string→millicore
// conversion. Returns 0 if no CPU requests are found.
func PodCPURequests(pod *unstructured.Unstructured) int64 {
	containers, found, err := unstructured.NestedSlice(pod.Object, "spec", "containers")
	if err != nil || !found {
		return 0
	}
	var total int64
	for _, c := range containers {
		ctr, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		res, ok := ctr["resources"].(map[string]interface{})
		if !ok {
			continue
		}
		req, ok := res["requests"].(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := req["cpu"]; ok {
			total += parseCPU(v)
		}
	}
	return total
}

// BuildPodAllocMaps iterates all pods and groups GPU and CPU requests by
// spec.nodeName. Returns two maps: gpuByNode (node name → total GPU requests)
// and cpuByNode (node name → total CPU millicore requests). Nodes with no
// GPU-requesting pods do not appear in the GPU map.
func BuildPodAllocMaps(pods []unstructured.Unstructured) (map[string]int, map[string]int64) {
	gpuByNode := make(map[string]int)
	cpuByNode := make(map[string]int64)

	for i := range pods {
		pod := &pods[i]
		nodeName, found, err := unstructured.NestedString(pod.Object, "spec", "nodeName")
		if err != nil || !found || nodeName == "" {
			continue
		}

		gpuReq := PodGPURequests(pod)
		if gpuReq > 0 {
			gpuByNode[nodeName] += gpuReq
		}

		cpuReq := PodCPURequests(pod)
		if cpuReq > 0 {
			cpuByNode[nodeName] += cpuReq
		}
	}

	return gpuByNode, cpuByNode
}

// AllocatablePods returns the max pod count from node allocatable.
func AllocatablePods(node *unstructured.Unstructured) int {
	m := getNestedMap(node, "status", "allocatable")
	if m == nil {
		return 0
	}
	if v, ok := m["pods"]; ok {
		return toInt(v)
	}
	return 0
}
