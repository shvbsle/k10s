package resources

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shvbsle/k10s/internal/k8s"
)

const (
	resourceSchemaFileName = "resource.views.json"
)

//go:embed resource.views.json
var defaultResourceViews []byte

var resourceViews ResourceViews

func init() {
	resourceViewsJson := defaultResourceViews
	if home, err := os.UserHomeDir(); err == nil {
		if schema, err := os.ReadFile(filepath.Join(home, resourceSchemaFileName)); err == nil {
			resourceViewsJson = schema
		}
	}
	if err := json.Unmarshal(resourceViewsJson, &resourceViews); err != nil {
		panic(err)
	}
}

func GetResourceView(resource k8s.ResourceType) ResourceView {
	if view, ok := resourceViews[resource]; ok {
		return view
	}
	return ResourceView{
		Fields: []ResourceViewField{
			ResourceViewField{
				Name:         "Name",
				PathTemplate: "{{ .metadata.name }}",
				Weight:       .5,
			},
			// TODO: this effectively assumed the resource is namespaced, but we
			// can always determine that dynamically using the API.
			ResourceViewField{
				Name:         "Namespace",
				PathTemplate: "{{ .metadata.namespace }}",
				Weight:       .5,
			},
		},
	}
}

type ResourceViews map[k8s.ResourceType]ResourceView

type ResourceView struct {
	DrillDown *struct {
		Resource          string   `json:"resource"`
		SelectorTemplates []string `json:"selectors"`
	} `json:"drill,omitempty"`
	Fields []ResourceViewField `json:"fields"`
}

type ResourceViewField struct {
	Weight       float32 `json:"weight"`
	PathTemplate string  `json:"path"`
	Name         string  `json:"name"`
}

func GetColumns(totalWidth int, resource k8s.ResourceType) []table.Column {
	var columns []table.Column
	for _, field := range GetResourceView(resource).Fields {
		columns = append(columns, table.Column{
			Title: field.Name,
			Width: int(field.Weight * float32(totalWidth)),
		})
	}
	return columns
}
