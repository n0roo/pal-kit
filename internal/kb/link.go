package kb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// LinkService handles link and tag analysis
type LinkService struct {
	vaultPath string
}

// Link represents a wikilink
type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Alias  string `json:"alias,omitempty"`
	Line   int    `json:"line"`
}

// BrokenLink represents a broken link
type BrokenLink struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Line       int    `json:"line"`
	Suggestion string `json:"suggestion,omitempty"`
}

// LinkCheckResult represents link check results
type LinkCheckResult struct {
	TotalLinks  int           `json:"total_links"`
	ValidLinks  int           `json:"valid_links"`
	BrokenLinks []*BrokenLink `json:"broken_links,omitempty"`
}

// LinkGraph represents the link graph
type LinkGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode represents a node in the link graph
type GraphNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Type     string `json:"type,omitempty"`
	InLinks  int    `json:"in_links"`
	OutLinks int    `json:"out_links"`
}

// GraphEdge represents an edge in the link graph
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// TagInfo represents tag information
type TagInfo struct {
	Name      string   `json:"name"`
	Count     int      `json:"count"`
	Documents []string `json:"documents,omitempty"`
}

// TagCheckResult represents tag check results
type TagCheckResult struct {
	TotalTags   int        `json:"total_tags"`
	UsedTags    int        `json:"used_tags"`
	OrphanTags  []string   `json:"orphan_tags,omitempty"`
	UnknownTags []*TagInfo `json:"unknown_tags,omitempty"`
}

// NewLinkService creates a new link service
func NewLinkService(vaultPath string) *LinkService {
	return &LinkService{
		vaultPath: vaultPath,
	}
}

// WikilinkRegex matches [[target]] or [[target|alias]]
var WikilinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// HashtagRegex matches #tag (but not in code blocks or URLs)
var HashtagRegex = regexp.MustCompile(`(?:^|[^&\w])#([\w가-힣/-]+)`)

// CheckLinks checks all links in the vault
func (s *LinkService) CheckLinks() (*LinkCheckResult, error) {
	result := &LinkCheckResult{}

	// Build file index for quick lookup
	fileIndex := s.buildFileIndex()

	// Scan all markdown files
	sections := []string{TaxonomyDir, SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}

	for _, section := range sections {
		sectionPath := filepath.Join(s.vaultPath, section)
		if _, err := os.Stat(sectionPath); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			relPath, _ := filepath.Rel(s.vaultPath, path)
			links, err := s.extractLinks(path)
			if err != nil {
				return nil
			}

			for _, link := range links {
				result.TotalLinks++

				// Resolve link target
				targetPath := s.resolveLink(link.Target, relPath, fileIndex)
				if targetPath == "" {
					result.BrokenLinks = append(result.BrokenLinks, &BrokenLink{
						Source:     relPath,
						Target:     link.Target,
						Line:       link.Line,
						Suggestion: s.suggestTarget(link.Target, fileIndex),
					})
				} else {
					result.ValidLinks++
				}
			}

			return nil
		})
	}

	return result, nil
}

func (s *LinkService) buildFileIndex() map[string]string {
	index := make(map[string]string)

	filepath.Walk(s.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, _ := filepath.Rel(s.vaultPath, path)

		// Index by full path (without .md)
		pathNoExt := strings.TrimSuffix(relPath, ".md")
		index[pathNoExt] = relPath

		// Index by filename only
		name := strings.TrimSuffix(info.Name(), ".md")
		if _, exists := index[name]; !exists {
			index[name] = relPath
		}

		return nil
	})

	return index
}

func (s *LinkService) extractLinks(path string) ([]*Link, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var links []*Link
	content := string(data)
	lines := strings.Split(content, "\n")

	inCodeBlock := false
	for lineNum, line := range lines {
		// Track code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Find wikilinks
		matches := WikilinkRegex.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				target := line[match[2]:match[3]]
				var alias string
				if match[4] != -1 && match[5] != -1 {
					alias = line[match[4]:match[5]]
				}

				links = append(links, &Link{
					Target: target,
					Alias:  alias,
					Line:   lineNum + 1,
				})
			}
		}
	}

	return links, nil
}

func (s *LinkService) resolveLink(target, sourcePath string, fileIndex map[string]string) string {
	// Remove any anchor
	target = strings.Split(target, "#")[0]
	if target == "" {
		return sourcePath // Self-reference with anchor
	}

	// Try exact match
	if path, ok := fileIndex[target]; ok {
		return path
	}

	// Try relative to source
	sourceDir := filepath.Dir(sourcePath)
	relTarget := filepath.Join(sourceDir, target)
	if path, ok := fileIndex[relTarget]; ok {
		return path
	}

	// Try with _index suffix for directories
	if path, ok := fileIndex[target+"/_index"]; ok {
		return path
	}

	return ""
}

func (s *LinkService) suggestTarget(target string, fileIndex map[string]string) string {
	targetLower := strings.ToLower(target)
	var bestMatch string
	bestScore := 0

	for name, path := range fileIndex {
		nameLower := strings.ToLower(name)

		// Exact match in path components
		if strings.Contains(nameLower, targetLower) {
			score := len(targetLower) * 10
			if score > bestScore {
				bestScore = score
				bestMatch = path
			}
		}

		// Partial match
		if strings.Contains(targetLower, strings.ToLower(filepath.Base(name))) {
			score := len(filepath.Base(name))
			if score > bestScore {
				bestScore = score
				bestMatch = path
			}
		}
	}

	if bestMatch != "" {
		return strings.TrimSuffix(bestMatch, ".md")
	}
	return ""
}

// BuildLinkGraph builds the link graph
func (s *LinkService) BuildLinkGraph() (*LinkGraph, error) {
	graph := &LinkGraph{}
	fileIndex := s.buildFileIndex()

	// Track in/out links
	inLinks := make(map[string]int)
	outLinks := make(map[string]int)
	edges := make(map[string]bool)

	// Collect all nodes and edges
	sections := []string{TaxonomyDir, SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}

	for _, section := range sections {
		sectionPath := filepath.Join(s.vaultPath, section)
		if _, err := os.Stat(sectionPath); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			relPath, _ := filepath.Rel(s.vaultPath, path)
			nodeID := strings.TrimSuffix(relPath, ".md")

			links, err := s.extractLinks(path)
			if err != nil {
				return nil
			}

			for _, link := range links {
				targetPath := s.resolveLink(link.Target, relPath, fileIndex)
				if targetPath != "" {
					targetID := strings.TrimSuffix(targetPath, ".md")

					// Create edge
					edgeKey := nodeID + "->" + targetID
					if !edges[edgeKey] {
						edges[edgeKey] = true
						graph.Edges = append(graph.Edges, GraphEdge{
							Source: nodeID,
							Target: targetID,
						})
						outLinks[nodeID]++
						inLinks[targetID]++
					}
				}
			}

			return nil
		})
	}

	// Build nodes
	nodeSet := make(map[string]bool)
	for _, edge := range graph.Edges {
		nodeSet[edge.Source] = true
		nodeSet[edge.Target] = true
	}

	for nodeID := range nodeSet {
		// Get node type from path
		nodeType := ""
		parts := strings.Split(nodeID, "/")
		if len(parts) > 0 {
			nodeType = parts[0]
		}

		// Get label from file
		label := filepath.Base(nodeID)
		if title := s.getTitle(filepath.Join(s.vaultPath, nodeID+".md")); title != "" {
			label = title
		}

		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:       nodeID,
			Label:    label,
			Type:     nodeType,
			InLinks:  inLinks[nodeID],
			OutLinks: outLinks[nodeID],
		})
	}

	return graph, nil
}

func (s *LinkService) getTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Try first heading
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}

	return ""
}

// SaveLinkGraph saves the link graph to a file
func (s *LinkService) SaveLinkGraph(graph *LinkGraph) error {
	graphPath := filepath.Join(s.vaultPath, MetaDir, "link-graph.json")

	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(graphPath, data, 0644)
}

// ListTags lists all tags in the vault
func (s *LinkService) ListTags() ([]*TagInfo, error) {
	tagMap := make(map[string]*TagInfo)

	sections := []string{TaxonomyDir, SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}

	for _, section := range sections {
		sectionPath := filepath.Join(s.vaultPath, section)
		if _, err := os.Stat(sectionPath); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			relPath, _ := filepath.Rel(s.vaultPath, path)
			tags, err := s.extractTags(path)
			if err != nil {
				return nil
			}

			for _, tag := range tags {
				if _, ok := tagMap[tag]; !ok {
					tagMap[tag] = &TagInfo{
						Name:      tag,
						Documents: []string{},
					}
				}
				tagMap[tag].Count++
				tagMap[tag].Documents = append(tagMap[tag].Documents, relPath)
			}

			return nil
		})
	}

	// Convert to slice and sort
	var tags []*TagInfo
	for _, info := range tagMap {
		tags = append(tags, info)
	}

	return tags, nil
}

func (s *LinkService) extractTags(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	tagSet := make(map[string]bool)

	// Extract inline hashtags
	matches := HashtagRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tagSet[match[1]] = true
		}
	}

	// Extract from frontmatter tags array
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			// Simple YAML parsing for tags
			lines := strings.Split(parts[1], "\n")
			inTags := false
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "tags:") {
					inTags = true
					// Handle inline array: tags: [a, b, c]
					if strings.Contains(trimmed, "[") {
						arrayStr := strings.TrimPrefix(trimmed, "tags:")
						arrayStr = strings.Trim(arrayStr, " []")
						for _, tag := range strings.Split(arrayStr, ",") {
							tag = strings.Trim(tag, " \"'")
							if tag != "" {
								tagSet[tag] = true
							}
						}
						inTags = false
					}
					continue
				}
				if inTags {
					if strings.HasPrefix(trimmed, "-") {
						tag := strings.TrimPrefix(trimmed, "-")
						tag = strings.Trim(tag, " \"'")
						if tag != "" {
							tagSet[tag] = true
						}
					} else if trimmed != "" && !strings.HasPrefix(trimmed, " ") {
						inTags = false
					}
				}
			}
		}
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags, nil
}

// CheckOrphanTags checks for orphan tags (defined but not used)
func (s *LinkService) CheckOrphanTags() (*TagCheckResult, error) {
	result := &TagCheckResult{}

	// Load defined tags from taxonomy
	definedTags, err := s.loadDefinedTags()
	if err != nil {
		return nil, err
	}

	// Get used tags
	usedTags, err := s.ListTags()
	if err != nil {
		return nil, err
	}

	usedTagSet := make(map[string]bool)
	for _, tag := range usedTags {
		usedTagSet[tag.Name] = true
	}

	result.TotalTags = len(definedTags)
	result.UsedTags = 0

	// Find orphan tags (defined but not used)
	for _, tag := range definedTags {
		if usedTagSet[tag] {
			result.UsedTags++
		} else {
			result.OrphanTags = append(result.OrphanTags, tag)
		}
	}

	// Find unknown tags (used but not defined)
	definedTagSet := make(map[string]bool)
	for _, tag := range definedTags {
		definedTagSet[tag] = true
	}

	for _, tagInfo := range usedTags {
		if !definedTagSet[tagInfo.Name] {
			result.UnknownTags = append(result.UnknownTags, tagInfo)
		}
	}

	return result, nil
}

func (s *LinkService) loadDefinedTags() ([]string, error) {
	tagsPath := filepath.Join(s.vaultPath, TaxonomyDir, "tags.yaml")
	data, err := os.ReadFile(tagsPath)
	if err != nil {
		return nil, nil // No tags.yaml, return empty
	}

	// Simple extraction of tags from YAML
	var tags []string
	content := string(data)
	lines := strings.Split(content, "\n")

	inTags := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "tags:") {
			inTags = true
			continue
		}
		if inTags && strings.HasPrefix(trimmed, "-") {
			tag := strings.TrimPrefix(trimmed, "-")
			tag = strings.Trim(tag, " \"'")
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		// Reset if we hit a new top-level key
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "-") {
			if !strings.HasPrefix(trimmed, "tags:") {
				inTags = false
			}
		}
	}

	return tags, nil
}

// GetBacklinks returns documents that link to a given path
func (s *LinkService) GetBacklinks(targetPath string) ([]*Link, error) {
	var backlinks []*Link
	fileIndex := s.buildFileIndex()

	targetPath = strings.TrimSuffix(targetPath, ".md")

	sections := []string{TaxonomyDir, SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}

	for _, section := range sections {
		sectionPath := filepath.Join(s.vaultPath, section)
		if _, err := os.Stat(sectionPath); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			relPath, _ := filepath.Rel(s.vaultPath, path)
			links, err := s.extractLinks(path)
			if err != nil {
				return nil
			}

			for _, link := range links {
				resolvedTarget := s.resolveLink(link.Target, relPath, fileIndex)
				if strings.TrimSuffix(resolvedTarget, ".md") == targetPath {
					link.Source = relPath
					backlinks = append(backlinks, link)
				}
			}

			return nil
		})
	}

	return backlinks, nil
}
