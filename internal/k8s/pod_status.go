package k8s

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PodDisplayStatus computes the effective display status for a pod, mirroring
// the logic kubectl uses. It inspects container statuses for waiting/terminated
// reasons (e.g. OOMKilled, CrashLoopBackOff) and init container failures,
// returning the most relevant human-readable status string.
func PodDisplayStatus(obj *unstructured.Unstructured) string {
	content := obj.UnstructuredContent()

	phase, _, _ := unstructured.NestedString(content, "status", "phase")
	if phase == "" {
		phase = "Unknown"
	}
	if reason, _, _ := unstructured.NestedString(content, "status", "reason"); reason != "" {
		phase = reason
	}

	if status := initContainerStatus(content); status != "" {
		return status
	}
	if status := containerStatus(content); status != "" {
		return status
	}

	if ts, found, _ := unstructured.NestedString(content, "metadata", "deletionTimestamp"); found && ts != "" {
		return "Terminating"
	}

	return phase
}

// initContainerStatus returns a non-empty string when an init container is in
// a non-normal state (waiting or terminated with a non-Completed reason).
func initContainerStatus(content map[string]interface{}) string {
	statuses, found, _ := unstructured.NestedSlice(content, "status", "initContainerStatuses")
	if !found {
		return ""
	}
	// Walk backwards so the last-failing init container wins.
	for i := len(statuses) - 1; i >= 0; i-- {
		cs, ok := statuses[i].(map[string]interface{})
		if !ok {
			continue
		}
		state, _, _ := unstructured.NestedMap(cs, "state")

		if waiting, ok := state["waiting"].(map[string]interface{}); ok {
			if r, _ := waiting["reason"].(string); r != "" {
				return "Init:" + r
			}
		}
		if terminated, ok := state["terminated"].(map[string]interface{}); ok {
			if r, _ := terminated["reason"].(string); r != "" && r != "Completed" {
				return "Init:" + r
			}
			if exitCode, _, _ := unstructured.NestedInt64(terminated, "exitCode"); exitCode != 0 {
				return fmt.Sprintf("Init:ExitCode:%d", exitCode)
			}
		}
	}
	return ""
}

// containerStatus returns a non-empty string when a regular container is in a
// waiting or terminated state.
func containerStatus(content map[string]interface{}) string {
	statuses, found, _ := unstructured.NestedSlice(content, "status", "containerStatuses")
	if !found {
		return ""
	}
	for i := len(statuses) - 1; i >= 0; i-- {
		cs, ok := statuses[i].(map[string]interface{})
		if !ok {
			continue
		}
		state, _, _ := unstructured.NestedMap(cs, "state")

		if waiting, ok := state["waiting"].(map[string]interface{}); ok {
			if r, _ := waiting["reason"].(string); r != "" {
				return r
			}
		}
		if terminated, ok := state["terminated"].(map[string]interface{}); ok {
			if r, _ := terminated["reason"].(string); r != "" {
				return r
			}
			if exitCode, _, _ := unstructured.NestedInt64(terminated, "exitCode"); exitCode != 0 {
				return fmt.Sprintf("ExitCode:%d", exitCode)
			}
		}
	}
	return ""
}

// PodRestartCount returns the total restart count across all containers in a pod.
func PodRestartCount(obj *unstructured.Unstructured) string {
	content := obj.UnstructuredContent()
	statuses, found, _ := unstructured.NestedSlice(content, "status", "containerStatuses")
	if !found {
		return "0"
	}

	var total int64
	for _, cs := range statuses {
		m, ok := cs.(map[string]interface{})
		if !ok {
			continue
		}
		if n, found, _ := unstructured.NestedInt64(m, "restartCount"); found {
			total += n
		}
	}
	return strconv.FormatInt(total, 10)
}
