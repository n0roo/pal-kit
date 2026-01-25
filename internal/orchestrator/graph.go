package orchestrator

import (
	"fmt"
	"sort"
)

// DependencyGraph represents a directed acyclic graph of port dependencies
type DependencyGraph struct {
	Nodes    map[string]*PortNode // port_id -> node
	Edges    map[string][]string  // port_id -> depends_on[]
	InDegree map[string]int       // port_id -> number of dependencies
}

// PortNode represents a port in the dependency graph
type PortNode struct {
	PortID    string
	Order     int
	Status    string // pending, running, complete, failed
	DependsOn []string
}

// NewDependencyGraph creates a new dependency graph from atomic ports
func NewDependencyGraph(ports []AtomicPort) *DependencyGraph {
	g := &DependencyGraph{
		Nodes:    make(map[string]*PortNode),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
	}

	// Initialize nodes
	for _, p := range ports {
		g.Nodes[p.PortID] = &PortNode{
			PortID:    p.PortID,
			Order:     p.Order,
			Status:    p.Status,
			DependsOn: p.DependsOn,
		}
		g.Edges[p.PortID] = p.DependsOn
		g.InDegree[p.PortID] = 0
	}

	// Calculate in-degrees
	for _, deps := range g.Edges {
		for _, dep := range deps {
			if _, exists := g.Nodes[dep]; exists {
				g.InDegree[dep]++
			}
		}
	}

	return g
}

// TopologicalLevels returns ports grouped by execution level using Kahn's algorithm
// Ports in the same level have no dependencies on each other and can run in parallel
func (g *DependencyGraph) TopologicalLevels() ([][]string, error) {
	// Work with a copy of in-degrees
	inDegree := make(map[string]int)
	for k, v := range g.InDegree {
		inDegree[k] = v
	}

	// Track remaining nodes
	remaining := make(map[string]bool)
	for portID := range g.Nodes {
		remaining[portID] = true
	}

	var levels [][]string

	for len(remaining) > 0 {
		// Find all nodes with in-degree 0
		var level []string
		for portID := range remaining {
			if inDegree[portID] == 0 {
				level = append(level, portID)
			}
		}

		if len(level) == 0 {
			// No nodes with in-degree 0 but still have remaining nodes
			// This indicates a cycle
			return nil, fmt.Errorf("순환 의존성이 발견되었습니다")
		}

		// Sort by order within level for deterministic execution
		sort.Slice(level, func(i, j int) bool {
			return g.Nodes[level[i]].Order < g.Nodes[level[j]].Order
		})

		levels = append(levels, level)

		// Remove processed nodes and update in-degrees
		for _, portID := range level {
			delete(remaining, portID)
			// Decrease in-degree of dependent nodes
			for depPortID := range remaining {
				for _, dep := range g.Edges[depPortID] {
					if dep == portID {
						inDegree[depPortID]--
					}
				}
			}
		}
	}

	return levels, nil
}

// GetReadyPorts returns ports that are ready to execute (all dependencies complete)
func (g *DependencyGraph) GetReadyPorts() []string {
	var ready []string

	for portID, node := range g.Nodes {
		if node.Status != "pending" {
			continue
		}

		// Check if all dependencies are complete
		allDepsComplete := true
		for _, depID := range node.DependsOn {
			if depNode, exists := g.Nodes[depID]; exists {
				if depNode.Status != "complete" {
					allDepsComplete = false
					break
				}
			}
		}

		if allDepsComplete {
			ready = append(ready, portID)
		}
	}

	// Sort by order
	sort.Slice(ready, func(i, j int) bool {
		return g.Nodes[ready[i]].Order < g.Nodes[ready[j]].Order
	})

	return ready
}

// MarkComplete marks a port as complete and returns newly unblocked ports
func (g *DependencyGraph) MarkComplete(portID string) []string {
	if node, exists := g.Nodes[portID]; exists {
		node.Status = "complete"
	}
	return g.GetReadyPorts()
}

// MarkRunning marks a port as running
func (g *DependencyGraph) MarkRunning(portID string) {
	if node, exists := g.Nodes[portID]; exists {
		node.Status = "running"
	}
}

// MarkFailed marks a port as failed
func (g *DependencyGraph) MarkFailed(portID string) {
	if node, exists := g.Nodes[portID]; exists {
		node.Status = "failed"
	}
}

// GetStatus returns the status of a port
func (g *DependencyGraph) GetStatus(portID string) string {
	if node, exists := g.Nodes[portID]; exists {
		return node.Status
	}
	return ""
}

// GetDependents returns ports that depend on the given port
func (g *DependencyGraph) GetDependents(portID string) []string {
	var dependents []string
	for pID, deps := range g.Edges {
		for _, dep := range deps {
			if dep == portID {
				dependents = append(dependents, pID)
				break
			}
		}
	}
	return dependents
}

// GetDependencies returns the dependencies of a port
func (g *DependencyGraph) GetDependencies(portID string) []string {
	if node, exists := g.Nodes[portID]; exists {
		return node.DependsOn
	}
	return nil
}

// HasCycle checks if the graph has any cycles
func (g *DependencyGraph) HasCycle() bool {
	_, err := g.TopologicalLevels()
	return err != nil
}

// GetCriticalPath returns the longest path in the graph (critical path)
func (g *DependencyGraph) GetCriticalPath() []string {
	levels, err := g.TopologicalLevels()
	if err != nil {
		return nil
	}

	// Build longest path ending at each node
	longestPath := make(map[string][]string)

	for _, level := range levels {
		for _, portID := range level {
			// Find the longest path among dependencies
			var longest []string
			for _, depID := range g.Edges[portID] {
				if path, exists := longestPath[depID]; exists {
					if len(path) > len(longest) {
						longest = path
					}
				}
			}

			// Create new path
			newPath := make([]string, len(longest)+1)
			copy(newPath, longest)
			newPath[len(longest)] = portID
			longestPath[portID] = newPath
		}
	}

	// Find the longest overall path
	var criticalPath []string
	for _, path := range longestPath {
		if len(path) > len(criticalPath) {
			criticalPath = path
		}
	}

	return criticalPath
}

// GetStats returns statistics about the graph
type GraphStats struct {
	TotalPorts     int      `json:"total_ports"`
	PendingPorts   int      `json:"pending_ports"`
	RunningPorts   int      `json:"running_ports"`
	CompletePorts  int      `json:"complete_ports"`
	FailedPorts    int      `json:"failed_ports"`
	MaxParallelism int      `json:"max_parallelism"`
	CriticalPath   []string `json:"critical_path"`
	Levels         int      `json:"levels"`
}

func (g *DependencyGraph) GetStats() (*GraphStats, error) {
	levels, err := g.TopologicalLevels()
	if err != nil {
		return nil, err
	}

	stats := &GraphStats{
		TotalPorts:   len(g.Nodes),
		CriticalPath: g.GetCriticalPath(),
		Levels:       len(levels),
	}

	// Count by status
	for _, node := range g.Nodes {
		switch node.Status {
		case "pending":
			stats.PendingPorts++
		case "running":
			stats.RunningPorts++
		case "complete":
			stats.CompletePorts++
		case "failed":
			stats.FailedPorts++
		}
	}

	// Find maximum parallelism
	for _, level := range levels {
		if len(level) > stats.MaxParallelism {
			stats.MaxParallelism = len(level)
		}
	}

	return stats, nil
}

// Clone creates a deep copy of the graph
func (g *DependencyGraph) Clone() *DependencyGraph {
	clone := &DependencyGraph{
		Nodes:    make(map[string]*PortNode),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
	}

	for k, v := range g.Nodes {
		clone.Nodes[k] = &PortNode{
			PortID:    v.PortID,
			Order:     v.Order,
			Status:    v.Status,
			DependsOn: append([]string{}, v.DependsOn...),
		}
	}

	for k, v := range g.Edges {
		clone.Edges[k] = append([]string{}, v...)
	}

	for k, v := range g.InDegree {
		clone.InDegree[k] = v
	}

	return clone
}
