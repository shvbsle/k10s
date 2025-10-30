package tui

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shvbsle/k10s/internal/k8s"
)

const (
	resourceSchemaFileName = "resource.schema.json"
)

//go:embed resource.schema.json
var defaultResourceSchema []byte

var resourceSchemas ResoureSchemas

func init() {
	resourceSchemaJson := defaultResourceSchema
	if home, err := os.UserHomeDir(); err == nil {
		if schema, err := os.ReadFile(filepath.Join(home, resourceSchemaFileName)); err == nil {
			resourceSchemaJson = schema
		}
	}
	if err := json.Unmarshal(resourceSchemaJson, &resourceSchemas); err != nil {
		panic(err)
	}
}

type ResoureSchemas map[k8s.ResourceType]struct {
	Fields []struct {
		Weight float32 `json:"weight"`
		Name   string  `json:"name"`
	} `json:"fields"`
}

func GetColumns(totalWidth int) map[k8s.ResourceType][]table.Column {
	columnMap := map[k8s.ResourceType][]table.Column{}
	for rType, schema := range resourceSchemas {
		var columns []table.Column
		for _, field := range schema.Fields {
			columns = append(columns, table.Column{
				Title: field.Name,
				Width: int(field.Weight * float32(totalWidth)),
			})
		}
		columnMap[rType] = columns
	}
	return columnMap
}
