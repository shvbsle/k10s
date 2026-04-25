package k8s

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makePod(nodeName string, containers []map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"nodeName":   nodeName,
				"containers": toSlice(containers),
			},
		},
	}
}

func toSlice(items []map[string]interface{}) []interface{} {
	s := make([]interface{}, len(items))
	for i, v := range items {
		s[i] = v
	}
	return s
}

func container(gpuReq int, cpuReq string) map[string]interface{} {
	req := map[string]interface{}{}
	if gpuReq > 0 {
		req[NvidiaGPUResource] = int64(gpuReq)
	}
	if cpuReq != "" {
		req["cpu"] = cpuReq
	}
	return map[string]interface{}{
		"resources": map[string]interface{}{
			"requests": req,
		},
	}
}

func TestPodGPURequests(t *testing.T) {
	tests := []struct {
		name string
		pod  *unstructured.Unstructured
		want int
	}{
		{
			name: "no containers",
			pod:  &unstructured.Unstructured{Object: map[string]interface{}{"spec": map[string]interface{}{}}},
			want: 0,
		},
		{
			name: "single container with 2 GPUs",
			pod:  makePod("node-a", []map[string]interface{}{container(2, "")}),
			want: 2,
		},
		{
			name: "two containers with GPUs",
			pod:  makePod("node-a", []map[string]interface{}{container(4, ""), container(4, "")}),
			want: 8,
		},
		{
			name: "container with no GPU request",
			pod:  makePod("node-a", []map[string]interface{}{container(0, "500m")}),
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PodGPURequests(tt.pod)
			if got != tt.want {
				t.Errorf("PodGPURequests() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPodCPURequests(t *testing.T) {
	tests := []struct {
		name string
		pod  *unstructured.Unstructured
		want int64
	}{
		{
			name: "no containers",
			pod:  &unstructured.Unstructured{Object: map[string]interface{}{"spec": map[string]interface{}{}}},
			want: 0,
		},
		{
			name: "single container 500m",
			pod:  makePod("node-a", []map[string]interface{}{container(0, "500m")}),
			want: 500,
		},
		{
			name: "single container 4 cores",
			pod:  makePod("node-a", []map[string]interface{}{container(0, "4")}),
			want: 4000,
		},
		{
			name: "two containers",
			pod:  makePod("node-a", []map[string]interface{}{container(0, "250m"), container(0, "750m")}),
			want: 1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PodCPURequests(tt.pod)
			if got != tt.want {
				t.Errorf("PodCPURequests() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildPodAllocMaps(t *testing.T) {
	pods := []unstructured.Unstructured{
		*makePod("gpu-node-1", []map[string]interface{}{container(4, "1000m")}),
		*makePod("gpu-node-1", []map[string]interface{}{container(2, "500m")}),
		*makePod("cpu-node-1", []map[string]interface{}{container(0, "2000m")}),
		*makePod("gpu-node-2", []map[string]interface{}{container(8, "4000m")}),
	}

	gpuMap, cpuMap := BuildPodAllocMaps(pods)

	// GPU map checks
	if gpuMap["gpu-node-1"] != 6 {
		t.Errorf("gpuMap[gpu-node-1] = %d, want 6", gpuMap["gpu-node-1"])
	}
	if gpuMap["gpu-node-2"] != 8 {
		t.Errorf("gpuMap[gpu-node-2] = %d, want 8", gpuMap["gpu-node-2"])
	}
	if _, exists := gpuMap["cpu-node-1"]; exists {
		t.Error("cpu-node-1 should not appear in GPU map")
	}

	// CPU map checks
	if cpuMap["gpu-node-1"] != 1500 {
		t.Errorf("cpuMap[gpu-node-1] = %d, want 1500", cpuMap["gpu-node-1"])
	}
	if cpuMap["cpu-node-1"] != 2000 {
		t.Errorf("cpuMap[cpu-node-1] = %d, want 2000", cpuMap["cpu-node-1"])
	}
	if cpuMap["gpu-node-2"] != 4000 {
		t.Errorf("cpuMap[gpu-node-2] = %d, want 4000", cpuMap["gpu-node-2"])
	}
}

func TestBuildPodAllocMaps_EmptyPods(t *testing.T) {
	gpuMap, cpuMap := BuildPodAllocMaps(nil)
	if len(gpuMap) != 0 {
		t.Errorf("expected empty GPU map, got %v", gpuMap)
	}
	if len(cpuMap) != 0 {
		t.Errorf("expected empty CPU map, got %v", cpuMap)
	}
}

func TestBuildPodAllocMaps_UnscheduledPods(t *testing.T) {
	// Pods without spec.nodeName should be skipped
	pod := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"containers": toSlice([]map[string]interface{}{container(4, "1000m")}),
			},
		},
	}
	gpuMap, cpuMap := BuildPodAllocMaps([]unstructured.Unstructured{*pod})
	if len(gpuMap) != 0 {
		t.Errorf("expected empty GPU map for unscheduled pod, got %v", gpuMap)
	}
	if len(cpuMap) != 0 {
		t.Errorf("expected empty CPU map for unscheduled pod, got %v", cpuMap)
	}
}
