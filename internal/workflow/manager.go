package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Manager interface {
	Build(workflowName string, params map[string]interface{}) ([]byte, error)
}

type NodeMapping struct {
	NodeID   string `yaml:"node_id"`
	Property string `yaml:"property"`
}

type WorkflowConfig struct {
	Mappings map[string]NodeMapping `yaml:"node_mappings"`
}

type manager struct {
	templateDir string
	configDir   string
}

func NewManager(templateDir, configDir string) Manager {
	return &manager{templateDir: templateDir, configDir: configDir}
}

func (m *manager) Build(workflowName string, params map[string]interface{}) ([]byte, error) {
	templatePath := filepath.Join(m.templateDir, workflowName+".json")
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}
	var workflow map[string]interface{}
	if err := json.Unmarshal(templateData, &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	configPath := filepath.Join(m.configDir, workflowName+".yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	var config WorkflowConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	for key, value := range params {
		mapping, ok := config.Mappings[key]
		if !ok {
			continue
		}

		node, ok := workflow[mapping.NodeID].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("node %s not found or not an object", mapping.NodeID)
		}
		inputs, ok := node["inputs"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("inputs for node %s not found", mapping.NodeID)
		}
		inputs[mapping.Property] = value
	}

	return json.Marshal(workflow)
}
