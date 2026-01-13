package marker

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Marker represents a PAL marker found in code
type Marker struct {
	Port      string   `json:"port"`       // @pal-port value (e.g., "L1-InventoryCommandService")
	Layer     string   `json:"layer"`      // @pal-layer value (e.g., "l1", "lm", "l2")
	Domain    string   `json:"domain"`     // @pal-domain value (e.g., "inventories")
	Adapter   string   `json:"adapter"`    // @pal-adapter value (optional)
	Depends   []string `json:"depends"`    // @pal-depends values
	Generated bool     `json:"generated"`  // @pal-generated presence
	FilePath  string   `json:"file_path"`  // Source file path
	Line      int      `json:"line"`       // Line number where marker starts
}

// Service handles marker operations
type Service struct {
	projectRoot string
}

// NewService creates a new marker service
func NewService(projectRoot string) *Service {
	return &Service{projectRoot: projectRoot}
}

// Regular expressions for marker parsing
var (
	portRe      = regexp.MustCompile(`@pal-port\s+(\S+)`)
	layerRe     = regexp.MustCompile(`@pal-layer\s+(\S+)`)
	domainRe    = regexp.MustCompile(`@pal-domain\s+(\S+)`)
	adapterRe   = regexp.MustCompile(`@pal-adapter\s+(\S+)`)
	dependsRe   = regexp.MustCompile(`@pal-depends\s+(.+)`)
	generatedRe = regexp.MustCompile(`@pal-generated`)
)

// ScanFile scans a single file for PAL markers
func (s *Service) ScanFile(filePath string) ([]Marker, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var markers []Marker
	var currentMarker *Marker
	var inComment bool
	lineNum := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Detect comment blocks based on file extension
		ext := filepath.Ext(filePath)

		switch ext {
		case ".kt", ".java", ".ts", ".tsx", ".js", ".jsx":
			// KDoc/JSDoc style: /** ... */
			if strings.Contains(line, "/**") {
				inComment = true
				currentMarker = &Marker{FilePath: filePath, Line: lineNum}
			}
			if strings.Contains(line, "*/") && inComment {
				inComment = false
				if currentMarker != nil && currentMarker.Port != "" {
					markers = append(markers, *currentMarker)
				}
				currentMarker = nil
			}
		case ".go":
			// Go style: // comments (consecutive lines)
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				if currentMarker == nil {
					currentMarker = &Marker{FilePath: filePath, Line: lineNum}
				}
				inComment = true
			} else if inComment {
				inComment = false
				if currentMarker != nil && currentMarker.Port != "" {
					markers = append(markers, *currentMarker)
				}
				currentMarker = nil
			}
		case ".py":
			// Python docstring: """ ... """
			if strings.Contains(line, `"""`) {
				if !inComment {
					inComment = true
					currentMarker = &Marker{FilePath: filePath, Line: lineNum}
				} else {
					inComment = false
					if currentMarker != nil && currentMarker.Port != "" {
						markers = append(markers, *currentMarker)
					}
					currentMarker = nil
				}
			}
		case ".yaml", ".yml":
			// YAML: # comments
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				if currentMarker == nil {
					currentMarker = &Marker{FilePath: filePath, Line: lineNum}
				}
				inComment = true
			} else if inComment {
				inComment = false
				if currentMarker != nil && currentMarker.Port != "" {
					markers = append(markers, *currentMarker)
				}
				currentMarker = nil
			}
		}

		// Parse marker tags if in comment
		if inComment && currentMarker != nil {
			s.parseMarkerLine(line, currentMarker)
		}
	}

	// Handle unclosed comment at end of file
	if currentMarker != nil && currentMarker.Port != "" {
		markers = append(markers, *currentMarker)
	}

	return markers, scanner.Err()
}

// parseMarkerLine extracts marker information from a comment line
func (s *Service) parseMarkerLine(line string, marker *Marker) {
	if match := portRe.FindStringSubmatch(line); len(match) > 1 {
		marker.Port = match[1]
	}
	if match := layerRe.FindStringSubmatch(line); len(match) > 1 {
		marker.Layer = match[1]
	}
	if match := domainRe.FindStringSubmatch(line); len(match) > 1 {
		marker.Domain = match[1]
	}
	if match := adapterRe.FindStringSubmatch(line); len(match) > 1 {
		marker.Adapter = match[1]
	}
	if match := dependsRe.FindStringSubmatch(line); len(match) > 1 {
		deps := strings.Split(match[1], ",")
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep != "" {
				marker.Depends = append(marker.Depends, dep)
			}
		}
	}
	if generatedRe.MatchString(line) {
		marker.Generated = true
	}
}

// ScanDirectory scans a directory recursively for PAL markers
func (s *Service) ScanDirectory(dir string) ([]Marker, error) {
	var allMarkers []Marker

	supportedExts := map[string]bool{
		".kt": true, ".java": true,
		".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".go": true, ".py": true,
		".yaml": true, ".yml": true,
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories and common excludes
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" || base == "build" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file extension is supported
		ext := filepath.Ext(path)
		if !supportedExts[ext] {
			return nil
		}

		markers, err := s.ScanFile(path)
		if err != nil {
			return nil // Skip files with errors
		}

		allMarkers = append(allMarkers, markers...)
		return nil
	})

	return allMarkers, err
}

// Scan scans the project root for all markers
func (s *Service) Scan() ([]Marker, error) {
	return s.ScanDirectory(s.projectRoot)
}

// ListByPort returns markers filtered by port pattern
func (s *Service) ListByPort(pattern string) ([]Marker, error) {
	markers, err := s.Scan()
	if err != nil {
		return nil, err
	}

	if pattern == "" {
		return markers, nil
	}

	var filtered []Marker
	for _, m := range markers {
		if matchPattern(m.Port, pattern) {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// ListByDomain returns markers filtered by domain
func (s *Service) ListByDomain(domain string) ([]Marker, error) {
	markers, err := s.Scan()
	if err != nil {
		return nil, err
	}

	var filtered []Marker
	for _, m := range markers {
		if m.Domain == domain {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// ListByLayer returns markers filtered by layer
func (s *Service) ListByLayer(layer string) ([]Marker, error) {
	markers, err := s.Scan()
	if err != nil {
		return nil, err
	}

	var filtered []Marker
	for _, m := range markers {
		if m.Layer == layer {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// ListGenerated returns only Claude-generated markers
func (s *Service) ListGenerated() ([]Marker, error) {
	markers, err := s.Scan()
	if err != nil {
		return nil, err
	}

	var filtered []Marker
	for _, m := range markers {
		if m.Generated {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// GetDependencyTree returns the dependency tree for a port
func (s *Service) GetDependencyTree(portID string) (map[string][]string, error) {
	markers, err := s.Scan()
	if err != nil {
		return nil, err
	}

	// Build port -> depends map
	deps := make(map[string][]string)
	for _, m := range markers {
		if len(m.Depends) > 0 {
			deps[m.Port] = m.Depends
		}
	}

	// Build tree from the given port
	tree := make(map[string][]string)
	var buildTree func(port string)
	buildTree = func(port string) {
		if _, visited := tree[port]; visited {
			return
		}
		tree[port] = deps[port]
		for _, dep := range deps[port] {
			buildTree(dep)
		}
	}

	buildTree(portID)
	return tree, nil
}

// CheckValidity checks for invalid markers (missing required fields)
func (s *Service) CheckValidity(strict bool) ([]MarkerIssue, error) {
	markers, err := s.Scan()
	if err != nil {
		return nil, err
	}

	var issues []MarkerIssue
	for _, m := range markers {
		// Port is always required
		if m.Port == "" {
			continue // Not a valid marker
		}

		if strict {
			// In strict mode, layer and domain are required
			if m.Layer == "" {
				issues = append(issues, MarkerIssue{
					Marker:  m,
					Type:    "missing_layer",
					Message: "@pal-layer is missing",
				})
			}
			if m.Domain == "" {
				issues = append(issues, MarkerIssue{
					Marker:  m,
					Type:    "missing_domain",
					Message: "@pal-domain is missing",
				})
			}
		}

		// Validate layer value
		if m.Layer != "" && m.Layer != "l1" && m.Layer != "lm" && m.Layer != "l2" {
			issues = append(issues, MarkerIssue{
				Marker:  m,
				Type:    "invalid_layer",
				Message: "@pal-layer must be l1, lm, or l2",
			})
		}
	}

	return issues, nil
}

// MarkerIssue represents a validation issue with a marker
type MarkerIssue struct {
	Marker  Marker `json:"marker"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// matchPattern matches a string against a glob-like pattern (* wildcard)
func matchPattern(s, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(s, prefix)
	}
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(s, suffix)
	}
	return s == pattern
}
