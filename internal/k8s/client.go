package k8s

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client manages the connection to a Kubernetes cluster and provides methods
// for listing resources. It gracefully handles disconnected states and supports
// reconnection.
type Client struct {
	clientset   kubernetes.Interface
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
type ResourceType = string

const (
	// ResourcePods represents Kubernetes pods.
	ResourcePods ResourceType = "pods"
	// ResourceNodes represents Kubernetes nodes.
	ResourceNodes ResourceType = "nodes"
	// ResourceNamespaces represents Kubernetes namespaces.
	ResourceNamespaces ResourceType = "namespaces"
	// ResourceServices represents Kubernetes services.
	ResourceServices ResourceType = "services"
	// ResourceContainers represents containers within a pod.
	ResourceContainers ResourceType = "containers"
	// ResourceLogs represents logs for a specific container.
	ResourceLogs ResourceType = "logs"
)

// OrderedResourceFields represents a Kubernetes resource with common fields suitable for
// display in the TUI table view.
type OrderedResourceFields []string

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

func (c *Client) Discovery() discovery.DiscoveryInterface {
	return c.clientset.Discovery()
}

func (c *Client) Dynamic() dynamic.Interface {
	return dynamic.New(c.clientset.Discovery().RESTClient())
}

func (c *Client) testConnection() bool {
	if c.clientset == nil {
		return false
	}
	_, err := c.Discovery().ServerVersion()
	return err == nil
}

func (c *Client) markDisconnected() {
	if c.isConnected {
		log.Printf("K8s: Client disconnected from cluster")
		c.isConnected = false
	}
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
		namespace = metav1.NamespaceDefault
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
		if serverVersion, err := c.Discovery().ServerVersion(); err == nil {
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

// ListContainersForPod retrieves all containers (init and regular) for a specific pod.
// Returns an error if the client is not connected or if the API request fails.
func (c *Client) ListContainersForPod(podName, namespace string) ([]OrderedResourceFields, error) {
	if !c.isConnected || c.clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx := context.Background()

	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		c.markDisconnected()
		return nil, err
	}

	var resources []OrderedResourceFields

	// Add init containers
	for _, container := range pod.Spec.InitContainers {
		status := "Waiting"
		restarts := 0
		ready := "No"

		for _, cs := range pod.Status.InitContainerStatuses {
			if cs.Name == container.Name {
				restarts = int(cs.RestartCount)
				if cs.Ready {
					ready = "Yes"
				}
				if cs.State.Running != nil {
					status = "Running"
				} else if cs.State.Terminated != nil {
					status = "Terminated"
				} else if cs.State.Waiting != nil {
					status = fmt.Sprintf("Waiting: %s", cs.State.Waiting.Reason)
				}
				break
			}
		}

		resources = append(resources, OrderedResourceFields{
			container.Name,
			"[init]",
			container.Image,
			status,
			fmt.Sprintf("%d", restarts),
			ready,
		})
	}

	// Add regular containers
	for _, container := range pod.Spec.Containers {
		status := "Waiting"
		restarts := 0
		ready := "No"

		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == container.Name {
				restarts = int(cs.RestartCount)
				if cs.Ready {
					ready = "Yes"
				}
				if cs.State.Running != nil {
					status = "Running"
				} else if cs.State.Terminated != nil {
					status = "Terminated"
				} else if cs.State.Waiting != nil {
					status = fmt.Sprintf("Waiting: %s", cs.State.Waiting.Reason)
				}
				break
			}
		}

		resources = append(resources, OrderedResourceFields{
			container.Name,
			metav1.NamespaceAll,
			container.Image,
			status,
			fmt.Sprintf("%d", restarts),
			ready,
		})
	}

	return resources, nil
}
