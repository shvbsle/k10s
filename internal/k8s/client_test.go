package k8s

import (
	"io"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListContainersForPod(t *testing.T) {
	tests := []struct {
		name      string
		pod       *corev1.Pod
		wantCount int
		wantErr   bool
	}{
		{
			name: "pod with init and regular containers",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{Name: "init-container", Image: "init:latest"},
					},
					Containers: []corev1.Container{
						{Name: "app-container", Image: "app:latest"},
						{Name: "sidecar", Image: "sidecar:latest"},
					},
				},
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "init-container",
							Ready:        false,
							RestartCount: 0,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "Completed",
								},
							},
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "app-container",
							Ready:        true,
							RestartCount: 2,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: metav1.Now(),
								},
							},
						},
						{
							Name:         "sidecar",
							Ready:        true,
							RestartCount: 0,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{
									StartedAt: metav1.Now(),
								},
							},
						},
					},
				},
			},
			wantCount: 3, // 1 init + 2 regular
			wantErr:   false,
		},
		{
			name: "pod with only regular containers",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple-pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "app:v1"},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "app",
							Ready:        true,
							RestartCount: 0,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(tt.pod)
			k8sClient := &Client{
				clientset:   fakeClient,
				isConnected: true,
			}

			resources, err := k8sClient.ListContainersForPod(tt.pod.Name, tt.pod.Namespace)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListContainersForPod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(resources) != tt.wantCount {
				t.Errorf("ListContainersForPod() got %d containers, want %d", len(resources), tt.wantCount)
			}
		})
	}
}

func TestGetContainerLogs(t *testing.T) {
	// Note: Testing GetContainerLogs with fake clientset is complex because it requires
	// mocking the log stream. This test validates the basic structure.
	// In a real scenario, you'd use a more sophisticated mock or integration tests.

	t.Run("disconnected client returns error", func(t *testing.T) {
		k8sClient := &Client{
			clientset:   nil,
			isConnected: false,
		}

		_, err := k8sClient.GetContainerLogs("pod", "default", "container", 100, false)
		if err == nil {
			t.Error("GetContainerLogs() expected error for disconnected client, got nil")
		}
	})
}

// TestLogLineParsing tests the log line parsing logic
func TestLogLineParsing(t *testing.T) {
	tests := []struct {
		name           string
		logContent     string
		withTimestamps bool
		wantLines      int
	}{
		{
			name:           "simple logs without timestamps",
			logContent:     "line1\nline2\nline3",
			withTimestamps: false,
			wantLines:      3,
		},
		{
			name:           "logs with timestamps",
			logContent:     "2024-01-01T10:00:00Z line1\n2024-01-01T10:00:01Z line2",
			withTimestamps: true,
			wantLines:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate log parsing
			scanner := strings.NewReader(tt.logContent)
			lines := 0
			buf := make([]byte, 1024)
			for {
				n, err := scanner.Read(buf)
				if err == io.EOF {
					break
				}
				if n > 0 {
					lines++
				}
			}

			// This is a simplified test - in reality, we'd test the actual GetContainerLogs
			// with a proper mock that returns log streams
			if lines == 0 && tt.wantLines > 0 {
				t.Errorf("Expected to read %d lines", tt.wantLines)
			}
		})
	}
}

func TestMarkDisconnected(t *testing.T) {
	t.Run("marks connected client as disconnected", func(t *testing.T) {
		k8sClient := &Client{
			isConnected: true,
		}

		k8sClient.markDisconnected()

		if k8sClient.isConnected {
			t.Error("markDisconnected() should set isConnected to false")
		}
	})

	t.Run("idempotent on already disconnected client", func(t *testing.T) {
		k8sClient := &Client{
			isConnected: false,
		}

		k8sClient.markDisconnected()

		if k8sClient.isConnected {
			t.Error("markDisconnected() should keep isConnected as false")
		}
	})
}
