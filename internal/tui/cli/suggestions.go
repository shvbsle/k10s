package cli

import (
	"slices"
	"strings"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

type Suggester interface {
	Suggestions(args ...string) []string
}

type treeNode struct {
	value string
	links []treeNode
}

type suggestionTree struct {
	nodes []treeNode
}

func (c *suggestionTree) Suggestions(args ...string) []string {
	// TODO: clean up this code some more..

	// walk the args down the tree.
	nodes := c.nodes
	var last string

	for i, arg := range args {
		last = arg
		match, ok := lo.Find(nodes, func(node treeNode) bool {
			return node.value == arg
		})
		if !ok {
			break
		}
		if ok && i == len(args)-1 {
			return []string{}
		}
		nodes = match.links
	}

	suggestions := lo.FilterMap(nodes, func(node treeNode, _ int) (string, bool) {
		return node.value, strings.HasPrefix(node.value, last)
	})

	// TODO: expose sorting configuration option
	// sorting here gets us the shortest suggestions first
	return slices.SortedFunc(slices.Values(suggestions), func(s1, s2 string) int {
		return len(s1) - len(s2)
	})
}

func ParseSuggestionTree(ast map[string]any) *suggestionTree {
	return &suggestionTree{
		nodes: lo.MapToSlice(ast, parseNode),
	}
}

func parseNode(value string, node any) treeNode {
	switch node := node.(type) {
	case map[string]any:
		return treeNode{
			value: value,
			links: lo.MapToSlice(node, parseNode),
		}
	case []string:
		return treeNode{
			value: value,
			links: lo.Map(node, func(value string, _ int) treeNode { return treeNode{value: value} }),
		}
	default:
		return treeNode{value: value}
	}
}

// GetServerGVRs fetches the preferred resources on the server and returns a
// list of their GVRs.
func GetServerGVRs(discovery discovery.DiscoveryInterface) []schema.GroupVersionResource {
	var suggestions []schema.GroupVersionResource

	apiResourceLists, err := discovery.ServerPreferredResources()
	if err != nil && len(apiResourceLists) == 0 {
		// Only bail out if we got no results at all.
		// ServerPreferredResources can return partial results alongside
		// a non-nil error (e.g. ErrGroupDiscoveryFailed) when some API
		// groups are unavailable. We still want to use whatever it returned.
		return suggestions
	}

	for _, apiResourceList := range apiResourceLists {
		// SAFETY: cannot fail on a field provided by the api server
		gv, _ := schema.ParseGroupVersion(apiResourceList.GroupVersion)

		for _, apiResource := range apiResourceList.APIResources {
			suggestions = append(suggestions, gv.WithResource(apiResource.Name))
		}
	}

	return suggestions
}
