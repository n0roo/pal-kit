package path

import (
	"fmt"
	"sort"
	"strings"

	"github.com/n0roo/pal-kit/internal/env"
)

// Path variable prefixes
const (
	VarWorkspace  = "$workspace"
	VarClaudeData = "$claude_data"
	VarHome       = "$home"
)

// Resolver handles logical <-> absolute path conversions
type Resolver struct {
	envService *env.Service
	paths      *env.PathVariables
}

// NewResolver creates a new path resolver
func NewResolver(envService *env.Service) *Resolver {
	return &Resolver{
		envService: envService,
	}
}

// loadPaths loads current environment paths (cached)
func (r *Resolver) loadPaths() (*env.PathVariables, error) {
	if r.paths != nil {
		return r.paths, nil
	}

	current, err := r.envService.Current()
	if err != nil {
		// Fallback to default paths if no environment configured
		defaults := env.DefaultPaths()
		r.paths = &defaults
		return r.paths, nil
	}

	r.paths = &current.Paths
	return r.paths, nil
}

// Reset clears cached paths (call when environment changes)
func (r *Resolver) Reset() {
	r.paths = nil
}

// ToLogical converts absolute path to logical path
// Example: /Users/n0roo/playground/project -> $workspace/project
func (r *Resolver) ToLogical(absPath string) (string, error) {
	if absPath == "" {
		return "", nil
	}

	// Already logical path
	if strings.HasPrefix(absPath, "$") {
		return absPath, nil
	}

	paths, err := r.loadPaths()
	if err != nil {
		return absPath, err
	}

	// Try to match paths in order of specificity (longest first)
	mappings := r.getSortedMappings(paths)

	for _, m := range mappings {
		if m.absPath == "" {
			continue
		}
		if strings.HasPrefix(absPath, m.absPath) {
			// Handle exact match
			if absPath == m.absPath {
				return m.varName, nil
			}
			// Handle path with suffix
			suffix := strings.TrimPrefix(absPath, m.absPath)
			if strings.HasPrefix(suffix, "/") {
				return m.varName + suffix, nil
			}
		}
	}

	// No match - return original path
	return absPath, nil
}

// ToAbsolute converts logical path to absolute path
// Example: $workspace/project -> /Users/n0roo/playground/project
func (r *Resolver) ToAbsolute(logicalPath string) (string, error) {
	if logicalPath == "" {
		return "", nil
	}

	// Not a logical path
	if !strings.HasPrefix(logicalPath, "$") {
		return logicalPath, nil
	}

	paths, err := r.loadPaths()
	if err != nil {
		return logicalPath, err
	}

	// Parse variable and suffix
	varName, suffix := r.parseLogicalPath(logicalPath)

	// Resolve variable
	var basePath string
	switch varName {
	case VarWorkspace:
		basePath = paths.Workspace
	case VarClaudeData:
		basePath = paths.ClaudeData
	case VarHome:
		basePath = paths.Home
	default:
		return logicalPath, fmt.Errorf("알 수 없는 경로 변수: %s", varName)
	}

	if basePath == "" {
		return logicalPath, fmt.Errorf("경로 변수가 설정되지 않음: %s", varName)
	}

	return basePath + suffix, nil
}

// IsLogical checks if path is a logical path
func (r *Resolver) IsLogical(path string) bool {
	return strings.HasPrefix(path, "$")
}

// IsResolvable checks if logical path can be resolved in current environment
func (r *Resolver) IsResolvable(logicalPath string) bool {
	if !r.IsLogical(logicalPath) {
		return true // Absolute paths are always "resolvable"
	}

	absPath, err := r.ToAbsolute(logicalPath)
	if err != nil {
		return false
	}

	// Check if still contains unresolved variable
	return !strings.HasPrefix(absPath, "$")
}

// GetVariable extracts the variable name from a logical path
func (r *Resolver) GetVariable(logicalPath string) string {
	varName, _ := r.parseLogicalPath(logicalPath)
	return varName
}

// ConvertPaths converts multiple paths at once
func (r *Resolver) ConvertPaths(paths map[string]string, toLogical bool) (map[string]string, error) {
	result := make(map[string]string)

	for key, path := range paths {
		var converted string
		var err error

		if toLogical {
			converted, err = r.ToLogical(path)
		} else {
			converted, err = r.ToAbsolute(path)
		}

		if err != nil {
			return nil, fmt.Errorf("경로 변환 실패 (%s): %w", key, err)
		}
		result[key] = converted
	}

	return result, nil
}

// pathMapping represents a path variable mapping
type pathMapping struct {
	varName string
	absPath string
}

// getSortedMappings returns mappings sorted by path length (longest first)
func (r *Resolver) getSortedMappings(paths *env.PathVariables) []pathMapping {
	mappings := []pathMapping{
		{VarWorkspace, paths.Workspace},
		{VarClaudeData, paths.ClaudeData},
		{VarHome, paths.Home},
	}

	// Sort by path length descending (most specific first)
	sort.Slice(mappings, func(i, j int) bool {
		return len(mappings[i].absPath) > len(mappings[j].absPath)
	})

	return mappings
}

// parseLogicalPath splits logical path into variable and suffix
func (r *Resolver) parseLogicalPath(logicalPath string) (varName, suffix string) {
	if !strings.HasPrefix(logicalPath, "$") {
		return "", logicalPath
	}

	// Find the end of variable name (next / or end of string)
	idx := strings.Index(logicalPath, "/")
	if idx == -1 {
		return logicalPath, ""
	}

	return logicalPath[:idx], logicalPath[idx:]
}

// PathInfo contains information about a path
type PathInfo struct {
	Original   string `json:"original"`
	Logical    string `json:"logical"`
	Absolute   string `json:"absolute"`
	Variable   string `json:"variable,omitempty"`
	Resolvable bool   `json:"resolvable"`
}

// Analyze returns detailed information about a path
func (r *Resolver) Analyze(path string) (*PathInfo, error) {
	info := &PathInfo{
		Original: path,
	}

	if r.IsLogical(path) {
		info.Logical = path
		info.Variable = r.GetVariable(path)

		abs, err := r.ToAbsolute(path)
		if err != nil {
			info.Resolvable = false
			return info, nil
		}
		info.Absolute = abs
		info.Resolvable = !strings.HasPrefix(abs, "$")
	} else {
		info.Absolute = path

		logical, err := r.ToLogical(path)
		if err != nil {
			info.Logical = path
		} else {
			info.Logical = logical
		}

		if strings.HasPrefix(info.Logical, "$") {
			info.Variable = r.GetVariable(info.Logical)
		}
		info.Resolvable = true
	}

	return info, nil
}

// BatchConvert converts a slice of paths
type BatchResult struct {
	Original  string `json:"original"`
	Converted string `json:"converted"`
	Error     string `json:"error,omitempty"`
}

// BatchToLogical converts multiple absolute paths to logical paths
func (r *Resolver) BatchToLogical(paths []string) []BatchResult {
	results := make([]BatchResult, len(paths))
	for i, p := range paths {
		converted, err := r.ToLogical(p)
		results[i] = BatchResult{
			Original:  p,
			Converted: converted,
		}
		if err != nil {
			results[i].Error = err.Error()
		}
	}
	return results
}

// BatchToAbsolute converts multiple logical paths to absolute paths
func (r *Resolver) BatchToAbsolute(paths []string) []BatchResult {
	results := make([]BatchResult, len(paths))
	for i, p := range paths {
		converted, err := r.ToAbsolute(p)
		results[i] = BatchResult{
			Original:  p,
			Converted: converted,
		}
		if err != nil {
			results[i].Error = err.Error()
		}
	}
	return results
}
