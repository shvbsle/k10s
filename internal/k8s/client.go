package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
}

type ResourceType string

const (
	ResourcePods       ResourceType = "pods"
	ResourceNodes      ResourceType = "nodes"
	ResourceNamespaces ResourceType = "namespaces"
)

type Resource struct {
	Name      string
	Namespace string
	Status    string
	Age       string
	Extra     string // For additional info like node IP, pod IP, etc.
}

func NewClient() (*Client, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Client) ListPods(namespace string) ([]Resource, error) {
	ctx := context.Background()
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	pods, err := c.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, len(pods.Items))
	for i, pod := range pods.Items {
		status := string(pod.Status.Phase)
		if pod.DeletionTimestamp != nil {
			status = "Terminating"
		}

		resources[i] = Resource{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    status,
			Age:       formatAge(pod.CreationTimestamp.Time),
			Extra:     pod.Status.PodIP,
		}
	}

	return resources, nil
}

func (c *Client) ListNodes() ([]Resource, error) {
	ctx := context.Background()

	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, len(nodes.Items))
	for i, node := range nodes.Items {
		status := "NotReady"
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				status = "Ready"
				break
			}
		}

		// Get internal IP
		var nodeIP string
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				nodeIP = addr.Address
				break
			}
		}

		resources[i] = Resource{
			Name:      node.Name,
			Namespace: "",
			Status:    status,
			Age:       formatAge(node.CreationTimestamp.Time),
			Extra:     nodeIP,
		}
	}

	return resources, nil
}

func (c *Client) ListNamespaces() ([]Resource, error) {
	ctx := context.Background()

	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		status := string(ns.Status.Phase)

		resources[i] = Resource{
			Name:      ns.Name,
			Namespace: "",
			Status:    status,
			Age:       formatAge(ns.CreationTimestamp.Time),
			Extra:     "",
		}
	}

	return resources, nil
}
