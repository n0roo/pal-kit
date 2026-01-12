package worker

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkerSpec represents a worker YAML specification
type WorkerSpec struct {
	Agent WorkerAgent `yaml:"agent"`
}

// WorkerAgent is the agent section of a worker YAML
type WorkerAgent struct {
	ID               string            `yaml:"id"`
	Name             string            `yaml:"name"`
	Type             string            `yaml:"type"`
	Layer            string            `yaml:"layer"`
	Description      string            `yaml:"description"`
	Tech             WorkerTech        `yaml:"tech"`
	Responsibilities []string          `yaml:"responsibilities"`
	ConventionsRef   string            `yaml:"conventions_ref"`
	PortTypes        []string          `yaml:"port_types"`
	Dependencies     WorkerDependency  `yaml:"dependencies"`
	Checklist        []string          `yaml:"checklist"`
	Escalation       []EscalationRule  `yaml:"escalation"`
	Tools            []string          `yaml:"tools"`
	Prompt           string            `yaml:"prompt"`
}

// WorkerTech represents worker technology stack
type WorkerTech struct {
	Language   string   `yaml:"language"`
	Frameworks []string `yaml:"frameworks"`
}

// WorkerDependency represents worker dependency rules
type WorkerDependency struct {
	Allowed   []string `yaml:"allowed"`
	Forbidden []string `yaml:"forbidden"`
}

// EscalationRule represents an escalation rule
type EscalationRule struct {
	Condition string `yaml:"condition"`
	Target    string `yaml:"target"`
	Action    string `yaml:"action"`
}

// PortTypeMapping maps port types to workers
type PortTypeMapping struct {
	PortType     string   `yaml:"port_type"`
	Workers      []string `yaml:"workers"`
	DefaultHint  string   `yaml:"default_hint"`  // tech hint for default selection
	SelectionBy  string   `yaml:"selection_by"`  // "tech" | "layer" | "manual"
}

// Mapper handles worker selection based on port specification
type Mapper struct {
	projectRoot string
	workers     map[string]*WorkerSpec
	mappings    []PortTypeMapping
}

// NewMapper creates a new worker mapper
func NewMapper(projectRoot string) *Mapper {
	m := &Mapper{
		projectRoot: projectRoot,
		workers:     make(map[string]*WorkerSpec),
		mappings:    defaultMappings(),
	}
	return m
}

// defaultMappings returns the default port type to worker mappings
func defaultMappings() []PortTypeMapping {
	return []PortTypeMapping{
		// Backend L1
		{
			PortType:    "tpl-server-l1-port",
			Workers:     []string{"entity-worker", "cache-worker", "document-worker"},
			SelectionBy: "tech",
		},
		// Backend LM
		{
			PortType:    "tpl-server-lm-port",
			Workers:     []string{"service-worker"},
			SelectionBy: "layer",
		},
		// Backend L2
		{
			PortType:    "tpl-server-l2-port",
			Workers:     []string{"service-worker", "router-worker"},
			SelectionBy: "tech",
		},
		// Backend L3
		{
			PortType:    "tpl-server-l3-port",
			Workers:     []string{"router-worker"},
			SelectionBy: "layer",
		},
		// Backend Test
		{
			PortType:    "tpl-server-test",
			Workers:     []string{"test-worker"},
			SelectionBy: "layer",
		},
		// Frontend Feature
		{
			PortType:    "tpl-client-feature",
			Workers:     []string{"frontend-engineer-worker"},
			SelectionBy: "layer",
		},
		// Frontend API
		{
			PortType:    "tpl-client-api-port",
			Workers:     []string{"component-model-worker"},
			SelectionBy: "layer",
		},
		// Frontend Query
		{
			PortType:    "tpl-client-query",
			Workers:     []string{"component-model-worker"},
			SelectionBy: "layer",
		},
		// Frontend Component
		{
			PortType:    "tpl-client-component-port",
			Workers:     []string{"component-ui-worker"},
			SelectionBy: "layer",
		},
		// Frontend E2E Test
		{
			PortType:    "tpl-client-e2e",
			Workers:     []string{"e2e-worker"},
			SelectionBy: "layer",
		},
		// Frontend Unit Test
		{
			PortType:    "tpl-unit-test",
			Workers:     []string{"unit-tc-worker"},
			SelectionBy: "layer",
		},
	}
}

// Load loads all worker specifications from agents directory
func (m *Mapper) Load() error {
	m.workers = make(map[string]*WorkerSpec)

	workersDir := filepath.Join(m.projectRoot, "agents", "workers")
	if _, err := os.Stat(workersDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(workersDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		worker, err := m.loadWorkerFile(path)
		if err != nil {
			return nil // skip invalid files
		}

		m.workers[worker.Agent.ID] = worker
		return nil
	})
}

// loadWorkerFile loads a single worker YAML file
func (m *Mapper) loadWorkerFile(filePath string) (*WorkerSpec, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var spec WorkerSpec
	if err := yaml.Unmarshal(content, &spec); err != nil {
		return nil, fmt.Errorf("YAML 파싱 실패: %w", err)
	}

	return &spec, nil
}

// GetWorker returns a worker by ID
func (m *Mapper) GetWorker(id string) (*WorkerSpec, error) {
	if len(m.workers) == 0 {
		if err := m.Load(); err != nil {
			return nil, err
		}
	}

	worker, ok := m.workers[id]
	if !ok {
		return nil, fmt.Errorf("워커 '%s'을(를) 찾을 수 없습니다", id)
	}

	return worker, nil
}

// ListWorkers returns all loaded workers
func (m *Mapper) ListWorkers() ([]*WorkerSpec, error) {
	if len(m.workers) == 0 {
		if err := m.Load(); err != nil {
			return nil, err
		}
	}

	var workers []*WorkerSpec
	for _, w := range m.workers {
		workers = append(workers, w)
	}

	return workers, nil
}

// PortHints contains hints extracted from port specification
type PortHints struct {
	PortTypes []string          // port_types from port spec
	Tech      map[string]string // tech hints (language, framework, etc.)
	Layer     string            // layer hint (L1, L2, L3, LM)
}

// MapPortToWorker determines the appropriate worker for a port
func (m *Mapper) MapPortToWorker(hints PortHints) (string, error) {
	if len(m.workers) == 0 {
		if err := m.Load(); err != nil {
			return "", err
		}
	}

	// Try each port type
	for _, portType := range hints.PortTypes {
		worker, err := m.findWorkerForPortType(portType, hints)
		if err == nil && worker != "" {
			return worker, nil
		}
	}

	// Fallback: try to match by layer
	if hints.Layer != "" {
		worker := m.findWorkerByLayer(hints.Layer, hints.Tech)
		if worker != "" {
			return worker, nil
		}
	}

	return "", fmt.Errorf("적합한 워커를 찾을 수 없습니다")
}

// findWorkerForPortType finds a worker for a specific port type
func (m *Mapper) findWorkerForPortType(portType string, hints PortHints) (string, error) {
	for _, mapping := range m.mappings {
		if mapping.PortType != portType {
			continue
		}

		if len(mapping.Workers) == 0 {
			continue
		}

		// Single worker - return it
		if len(mapping.Workers) == 1 {
			return mapping.Workers[0], nil
		}

		// Multiple workers - select by tech hints
		if mapping.SelectionBy == "tech" {
			selected := m.selectByTech(mapping.Workers, hints.Tech)
			if selected != "" {
				return selected, nil
			}
		}

		// Default to first worker
		return mapping.Workers[0], nil
	}

	return "", fmt.Errorf("매핑 없음: %s", portType)
}

// selectByTech selects a worker based on technology hints
func (m *Mapper) selectByTech(workerIDs []string, techHints map[string]string) string {
	for _, workerID := range workerIDs {
		worker, ok := m.workers[workerID]
		if !ok {
			continue
		}

		// Check framework match
		if framework, ok := techHints["framework"]; ok {
			for _, wf := range worker.Agent.Tech.Frameworks {
				if strings.Contains(strings.ToLower(wf), strings.ToLower(framework)) {
					return workerID
				}
			}
		}

		// Check tech match
		if tech, ok := techHints["tech"]; ok {
			tech = strings.ToLower(tech)

			// JPA/ORM → entity-worker
			if strings.Contains(tech, "jpa") || strings.Contains(tech, "orm") || strings.Contains(tech, "hibernate") {
				if workerID == "entity-worker" {
					return workerID
				}
			}

			// Redis/Cache → cache-worker
			if strings.Contains(tech, "redis") || strings.Contains(tech, "valkey") || strings.Contains(tech, "cache") {
				if workerID == "cache-worker" {
					return workerID
				}
			}

			// MongoDB → document-worker
			if strings.Contains(tech, "mongo") || strings.Contains(tech, "document") {
				if workerID == "document-worker" {
					return workerID
				}
			}

			// MVC/Controller → router-worker
			if strings.Contains(tech, "mvc") || strings.Contains(tech, "controller") || strings.Contains(tech, "rest") {
				if workerID == "router-worker" {
					return workerID
				}
			}
		}
	}

	return ""
}

// findWorkerByLayer finds a worker based on layer
func (m *Mapper) findWorkerByLayer(layer string, techHints map[string]string) string {
	layer = strings.ToUpper(layer)

	switch layer {
	case "L1":
		// Determine by tech
		if tech, ok := techHints["tech"]; ok {
			tech = strings.ToLower(tech)
			if strings.Contains(tech, "redis") || strings.Contains(tech, "cache") {
				return "cache-worker"
			}
			if strings.Contains(tech, "mongo") {
				return "document-worker"
			}
		}
		return "entity-worker"
	case "LM":
		return "service-worker"
	case "L2":
		return "service-worker"
	case "L3":
		return "router-worker"
	case "TEST":
		if tech, ok := techHints["tech"]; ok {
			if strings.Contains(strings.ToLower(tech), "e2e") {
				return "e2e-worker"
			}
		}
		return "test-worker"
	case "ORCHESTRATION":
		return "frontend-engineer-worker"
	case "LOGIC":
		return "component-model-worker"
	case "VIEW":
		return "component-ui-worker"
	}

	return ""
}

// ParsePortSpecHints extracts hints from port specification content
func ParsePortSpecHints(content string) PortHints {
	hints := PortHints{
		Tech: make(map[string]string),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))

	inPortTypes := false
	inTech := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Parse port_types section
		if strings.HasPrefix(strings.ToLower(line), "port_type") ||
			strings.HasPrefix(strings.ToLower(line), "포트 타입") ||
			strings.HasPrefix(strings.ToLower(line), "template:") {
			inPortTypes = true
			inTech = false

			// Inline value
			if idx := strings.Index(line, ":"); idx >= 0 {
				value := strings.TrimSpace(line[idx+1:])
				if value != "" && !strings.HasPrefix(value, "[") {
					hints.PortTypes = append(hints.PortTypes, value)
				}
			}
			continue
		}

		// Parse tech section
		if strings.HasPrefix(strings.ToLower(line), "tech:") ||
			strings.HasPrefix(strings.ToLower(line), "기술:") {
			inTech = true
			inPortTypes = false
			continue
		}

		// Parse layer
		if strings.HasPrefix(strings.ToLower(line), "layer:") ||
			strings.HasPrefix(strings.ToLower(line), "레이어:") {
			if idx := strings.Index(line, ":"); idx >= 0 {
				hints.Layer = strings.TrimSpace(line[idx+1:])
			}
			continue
		}

		// Collect port types
		if inPortTypes && strings.HasPrefix(line, "-") {
			value := strings.TrimPrefix(line, "-")
			value = strings.TrimSpace(value)
			if value != "" {
				hints.PortTypes = append(hints.PortTypes, value)
			}
		}

		// Collect tech hints
		if inTech && strings.HasPrefix(line, "-") {
			value := strings.TrimPrefix(line, "-")
			value = strings.TrimSpace(value)
			if value != "" {
				hints.Tech["tech"] = value
			}
		}

		// Detect tpl-* patterns anywhere
		if strings.Contains(line, "tpl-") {
			for _, word := range strings.Fields(line) {
				if strings.HasPrefix(word, "tpl-") {
					// Clean up punctuation
					word = strings.Trim(word, "[](),\"'")
					hints.PortTypes = append(hints.PortTypes, word)
				}
			}
		}

		// End of section
		if line == "" || (strings.HasPrefix(line, "#") && !inPortTypes && !inTech) {
			inPortTypes = false
			inTech = false
		}
	}

	// Deduplicate port types
	seen := make(map[string]bool)
	var unique []string
	for _, pt := range hints.PortTypes {
		if !seen[pt] {
			seen[pt] = true
			unique = append(unique, pt)
		}
	}
	hints.PortTypes = unique

	return hints
}

// GetWorkerConventionPath returns the convention file path for a worker
func (m *Mapper) GetWorkerConventionPath(workerID string) string {
	worker, err := m.GetWorker(workerID)
	if err != nil {
		return ""
	}

	if worker.Agent.ConventionsRef != "" {
		return filepath.Join(m.projectRoot, worker.Agent.ConventionsRef)
	}

	return ""
}

// GetWorkerPrompt returns the prompt for a worker
func (m *Mapper) GetWorkerPrompt(workerID string) string {
	worker, err := m.GetWorker(workerID)
	if err != nil {
		return ""
	}

	return worker.Agent.Prompt
}

// GetWorkerChecklist returns the checklist for a worker
func (m *Mapper) GetWorkerChecklist(workerID string) []string {
	worker, err := m.GetWorker(workerID)
	if err != nil {
		return nil
	}

	return worker.Agent.Checklist
}
