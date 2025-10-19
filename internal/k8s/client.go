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

// Client manages the connection to a Kubernetes cluster and provides methods
// for listing resources. It gracefully handles disconnected states and supports
// reconnection.
type Client struct {
	clientset   *kubernetes.Clientset
	config      *rest.Config
	isConnected bool
}

// ClusterInfo contains metadata about the current Kubernetes cluster context,
// including the cluster name, namespace, server URL, and Kubernetes version.
type ClusterInfo struct {
	Context    string
	Cluster    string
	Namespace  string
	Server     string
	K8sVersion string
}

// ResourceType represents the type of Kubernetes resource being displayed.
type ResourceType string

const (
	// ResourcePods represents Kubernetes pods.
	ResourcePods ResourceType = "pods"
	// ResourceNodes represents Kubernetes nodes.
	ResourceNodes ResourceType = "nodes"
	// ResourceNamespaces represents Kubernetes namespaces.
	ResourceNamespaces ResourceType = "namespaces"
	// ResourceServices represents Kubernetes services.
	ResourceServices ResourceType = "services"
)

// Resource represents a Kubernetes resource with common fields suitable for
// display in the TUI table view.
type Resource struct {
	Name      string
	Namespace string
	Node      string // Node name (for pods) or empty
	Status    string
	Age       string
	Extra     string // For additional info like node IP, pod IP, etc.
}

// NewClient creates a new Kubernetes client by attempting to load the kubeconfig.
// It does not fail if the cluster is unavailable - instead it returns a client
// in a disconnected state that can be reconnected later.
func NewClient() (*Client, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client := &Client{
		config:      config,
		isConnected: false,
	}

	// Try to connect but don't fail if cluster is unavailable
	clientset, err := kubernetes.NewForConfig(config)
	if err == nil {
		client.clientset = clientset
		client.isConnected = client.testConnection()
	}

	return client, nil
}

func (c *Client) testConnection() bool {
	if c.clientset == nil {
		return false
	}
	_, err := c.clientset.Discovery().ServerVersion()
	return err == nil
}

// IsConnected returns true if the client is currently connected to a Kubernetes cluster.
func (c *Client) IsConnected() bool {
	return c.isConnected
}

// Reconnect attempts to re-establish connection to the Kubernetes cluster.
// It returns an error if reconnection fails or if the connection test fails.
func (c *Client) Reconnect() error {
	if c.config == nil {
		config, err := getKubeConfig()
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %w", err)
		}
		c.config = config
	}

	clientset, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	c.clientset = clientset
	c.isConnected = c.testConnection()

	if !c.isConnected {
		return fmt.Errorf("connection test failed")
	}

	return nil
}

// GetClusterInfo retrieves metadata about the current Kubernetes cluster,
// including the context name, cluster name, default namespace, server URL,
// and Kubernetes version.
func (c *Client) GetClusterInfo() (*ClusterInfo, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Load the kubeconfig file to extract context/cluster info
	configLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{},
	)

	rawConfig, err := configLoader.RawConfig()
	if err != nil {
		return nil, err
	}

	namespace, _, err := configLoader.Namespace()
	if err != nil {
		namespace = "default"
	}

	currentContext := rawConfig.CurrentContext
	if currentContext == "" {
		return nil, fmt.Errorf("no current context set")
	}

	context, exists := rawConfig.Contexts[currentContext]
	if !exists {
		return nil, fmt.Errorf("context %s not found", currentContext)
	}

	cluster, exists := rawConfig.Clusters[context.Cluster]
	server := "unknown"
	if exists {
		server = cluster.Server
	}

	// Get K8s version if connected
	k8sVersion := "n/a"
	if c.isConnected && c.clientset != nil {
		if serverVersion, err := c.clientset.Discovery().ServerVersion(); err == nil {
			k8sVersion = serverVersion.GitVersion
		}
	}

	return &ClusterInfo{
		Context:    currentContext,
		Cluster:    context.Cluster,
		Namespace:  namespace,
		Server:     server,
		K8sVersion: k8sVersion,
	}, nil
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

// ListPods retrieves all pods in the specified namespace. If namespace is empty,
// it returns pods from all namespaces. Returns an error if the client is not
// connected or if the API request fails.
func (c *Client) ListPods(namespace string) ([]Resource, error) {
	if !c.isConnected || c.clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx := context.Background()
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	pods, err := c.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.isConnected = false // Mark as disconnected on error
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
			Node:      pod.Spec.NodeName,
			Status:    status,
			Age:       formatAge(pod.CreationTimestamp.Time),
			Extra:     pod.Status.PodIP,
		}
	}

	return resources, nil
}

// ListNodes retrieves all nodes in the Kubernetes cluster. Returns an error
// if the client is not connected or if the API request fails.
func (c *Client) ListNodes() ([]Resource, error) {
	if !c.isConnected || c.clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx := context.Background()

	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.isConnected = false // Mark as disconnected on error
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

// ListNamespaces retrieves all namespaces in the Kubernetes cluster. Returns
// an error if the client is not connected or if the API request fails.
func (c *Client) ListNamespaces() ([]Resource, error) {
	if !c.isConnected || c.clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx := context.Background()

	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.isConnected = false // Mark as disconnected on error
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

// ListServices retrieves all services in the specified namespace. If namespace is empty,
// it returns services from all namespaces. Returns an error if the client is not
// connected or if the API request fails.
func (c *Client) ListServices(namespace string) ([]Resource, error) {
	if !c.isConnected || c.clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx := context.Background()
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	services, err := c.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.isConnected = false // Mark as disconnected on error
		return nil, err
	}

	resources := make([]Resource, len(services.Items))
	for i, svc := range services.Items {
		// Service type (ClusterIP, NodePort, LoadBalancer, ExternalName)
		serviceType := string(svc.Spec.Type)

		// Build port info for Extra column
		var portInfo string
		if len(svc.Spec.Ports) > 0 {
			ports := make([]string, 0, len(svc.Spec.Ports))
			for _, port := range svc.Spec.Ports {
				if port.NodePort != 0 {
					ports = append(ports, fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol))
				} else {
					ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
				}
			}
			portInfo = ports[0]
			if len(ports) > 1 {
				portInfo += fmt.Sprintf("+%d", len(ports)-1)
			}
		}

		// Add Cluster IP to port info
		clusterIP := svc.Spec.ClusterIP
		if clusterIP != "" && clusterIP != "None" {
			if portInfo != "" {
				portInfo = clusterIP + " " + portInfo
			} else {
				portInfo = clusterIP
			}
		}

		resources[i] = Resource{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Node:      "",
			Status:    serviceType,
			Age:       formatAge(svc.CreationTimestamp.Time),
			Extra:     portInfo,
		}
	}

	return resources, nil
}
