package kb

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ClassifierService handles document classification
type ClassifierService struct {
	vaultPath string
	taxonomy  *Taxonomy
}

// Taxonomy represents the classification system
type Taxonomy struct {
	Domains  map[string]DomainDef  `yaml:"domains"`
	DocTypes map[string]DocTypeDef `yaml:"doc_types"`
	Tags     map[string]TagDef     `yaml:"tags"`
}

// DomainDef defines a domain
type DomainDef struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
	SubDomains  []string `yaml:"sub_domains,omitempty"`
}

// DocTypeDef defines a document type
type DocTypeDef struct {
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	RequiredFields  []string `yaml:"required_fields"`
	OptionalFields  []string `yaml:"optional_fields,omitempty"`
	Keywords        []string `yaml:"keywords"`
	FilePatterns    []string `yaml:"file_patterns,omitempty"`
}

// TagDef defines a tag
type TagDef struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
	Children    []string `yaml:"children,omitempty"`
}

// ClassifyResult represents classification suggestions
type ClassifyResult struct {
	FilePath          string              `json:"file_path"`
	CurrentMeta       map[string]any      `json:"current_meta,omitempty"`
	SuggestedType     []TypeSuggestion    `json:"suggested_type"`
	SuggestedDomain   []DomainSuggestion  `json:"suggested_domain"`
	SuggestedTags     []TagSuggestion     `json:"suggested_tags"`
	Confidence        float64             `json:"confidence"`
}

// TypeSuggestion represents a document type suggestion
type TypeSuggestion struct {
	Type       string  `json:"type"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
}

// DomainSuggestion represents a domain suggestion
type DomainSuggestion struct {
	Domain     string  `json:"domain"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
}

// TagSuggestion represents a tag suggestion
type TagSuggestion struct {
	Tag        string  `json:"tag"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
}

// NewClassifierService creates a new classifier service
func NewClassifierService(vaultPath string) *ClassifierService {
	return &ClassifierService{
		vaultPath: vaultPath,
	}
}

// LoadTaxonomy loads the taxonomy definitions
func (c *ClassifierService) LoadTaxonomy() error {
	c.taxonomy = &Taxonomy{
		Domains:  make(map[string]DomainDef),
		DocTypes: make(map[string]DocTypeDef),
		Tags:     make(map[string]TagDef),
	}

	// Load domains
	domainsPath := c.vaultPath + "/" + TaxonomyDir + "/domains.yaml"
	if data, err := os.ReadFile(domainsPath); err == nil {
		var domains struct {
			Domains map[string]DomainDef `yaml:"domains"`
		}
		if err := yaml.Unmarshal(data, &domains); err == nil {
			c.taxonomy.Domains = domains.Domains
		}
	}

	// Load doc types
	typesPath := c.vaultPath + "/" + TaxonomyDir + "/doc-types.yaml"
	if data, err := os.ReadFile(typesPath); err == nil {
		var types struct {
			DocTypes map[string]DocTypeDef `yaml:"doc_types"`
		}
		if err := yaml.Unmarshal(data, &types); err == nil {
			c.taxonomy.DocTypes = types.DocTypes
		}
	}

	// Load tags
	tagsPath := c.vaultPath + "/" + TaxonomyDir + "/tags.yaml"
	if data, err := os.ReadFile(tagsPath); err == nil {
		var tags struct {
			Tags map[string]TagDef `yaml:"tags"`
		}
		if err := yaml.Unmarshal(data, &tags); err == nil {
			c.taxonomy.Tags = tags.Tags
		}
	}

	// Add default types if not loaded
	if len(c.taxonomy.DocTypes) == 0 {
		c.taxonomy.DocTypes = getDefaultDocTypes()
	}

	// Add default domains if not loaded
	if len(c.taxonomy.Domains) == 0 {
		c.taxonomy.Domains = getDefaultDomains()
	}

	return nil
}

// Classify analyzes a document and suggests classifications
func (c *ClassifierService) Classify(filePath string) (*ClassifyResult, error) {
	if c.taxonomy == nil {
		if err := c.LoadTaxonomy(); err != nil {
			return nil, err
		}
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}

	// Parse frontmatter
	meta, body := parseFrontmatterForClassify(string(content))

	result := &ClassifyResult{
		FilePath:    filePath,
		CurrentMeta: meta,
	}

	// Combine content for analysis
	fullText := strings.ToLower(body)
	if title, ok := meta["title"].(string); ok {
		fullText = strings.ToLower(title) + " " + fullText
	}

	// Suggest document type
	result.SuggestedType = c.suggestType(filePath, fullText, meta)

	// Suggest domain
	result.SuggestedDomain = c.suggestDomain(fullText, meta)

	// Suggest tags
	result.SuggestedTags = c.suggestTags(fullText, meta)

	// Calculate overall confidence
	result.Confidence = c.calculateConfidence(result)

	return result, nil
}

func (c *ClassifierService) suggestType(filePath, content string, meta map[string]any) []TypeSuggestion {
	suggestions := []TypeSuggestion{}
	fileName := strings.ToLower(filePath)

	for typeID, typeDef := range c.taxonomy.DocTypes {
		score := 0.0
		reasons := []string{}

		// Check file patterns
		for _, pattern := range typeDef.FilePatterns {
			if matched, _ := regexp.MatchString(pattern, fileName); matched {
				score += 0.4
				reasons = append(reasons, fmt.Sprintf("파일 패턴 '%s' 매칭", pattern))
			}
		}

		// Check keywords in content
		keywordMatches := 0
		for _, kw := range typeDef.Keywords {
			if strings.Contains(content, strings.ToLower(kw)) {
				keywordMatches++
			}
		}
		if len(typeDef.Keywords) > 0 {
			keywordScore := float64(keywordMatches) / float64(len(typeDef.Keywords)) * 0.4
			score += keywordScore
			if keywordMatches > 0 {
				reasons = append(reasons, fmt.Sprintf("키워드 %d개 매칭", keywordMatches))
			}
		}

		// Check required fields presence
		fieldMatches := 0
		for _, field := range typeDef.RequiredFields {
			if _, ok := meta[field]; ok {
				fieldMatches++
			}
		}
		if len(typeDef.RequiredFields) > 0 {
			fieldScore := float64(fieldMatches) / float64(len(typeDef.RequiredFields)) * 0.2
			score += fieldScore
			if fieldMatches > 0 {
				reasons = append(reasons, fmt.Sprintf("필드 %d개 존재", fieldMatches))
			}
		}

		if score > 0.1 {
			suggestions = append(suggestions, TypeSuggestion{
				Type:   typeID,
				Score:  score,
				Reason: strings.Join(reasons, ", "),
			})
		}
	}

	// Sort by score
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Return top 3
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

func (c *ClassifierService) suggestDomain(content string, meta map[string]any) []DomainSuggestion {
	suggestions := []DomainSuggestion{}

	for domainID, domainDef := range c.taxonomy.Domains {
		score := 0.0
		reasons := []string{}

		// Check keywords
		keywordMatches := 0
		for _, kw := range domainDef.Keywords {
			kwLower := strings.ToLower(kw)
			count := strings.Count(content, kwLower)
			if count > 0 {
				keywordMatches += count
			}
		}
		if keywordMatches > 0 {
			score = float64(keywordMatches) * 0.1
			if score > 0.8 {
				score = 0.8
			}
			reasons = append(reasons, fmt.Sprintf("키워드 %d회 매칭", keywordMatches))
		}

		// Check domain name in content
		if strings.Contains(content, strings.ToLower(domainID)) {
			score += 0.2
			reasons = append(reasons, "도메인명 직접 언급")
		}

		if score > 0.1 {
			suggestions = append(suggestions, DomainSuggestion{
				Domain: domainID,
				Score:  score,
				Reason: strings.Join(reasons, ", "),
			})
		}
	}

	// Sort by score
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Return top 3
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

func (c *ClassifierService) suggestTags(content string, meta map[string]any) []TagSuggestion {
	suggestions := []TagSuggestion{}

	// Check existing hashtags in content
	hashtagRegex := regexp.MustCompile(`#([\w가-힣/-]+)`)
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	existingTags := make(map[string]bool)
	for _, m := range matches {
		existingTags[m[1]] = true
	}

	for tagID, tagDef := range c.taxonomy.Tags {
		score := 0.0
		reasons := []string{}

		// Check if tag already exists
		if existingTags[tagID] {
			continue // Skip already used tags
		}

		// Check keywords
		keywordMatches := 0
		for _, kw := range tagDef.Keywords {
			kwLower := strings.ToLower(kw)
			if strings.Contains(content, kwLower) {
				keywordMatches++
			}
		}
		if len(tagDef.Keywords) > 0 && keywordMatches > 0 {
			score = float64(keywordMatches) / float64(len(tagDef.Keywords))
			reasons = append(reasons, fmt.Sprintf("키워드 %d개 매칭", keywordMatches))
		}

		if score > 0.2 {
			suggestions = append(suggestions, TagSuggestion{
				Tag:    tagID,
				Score:  score,
				Reason: strings.Join(reasons, ", "),
			})
		}
	}

	// Sort by score
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Return top 5
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

func (c *ClassifierService) calculateConfidence(result *ClassifyResult) float64 {
	confidence := 0.0

	// Type confidence
	if len(result.SuggestedType) > 0 && result.SuggestedType[0].Score > 0.5 {
		confidence += 0.4
	} else if len(result.SuggestedType) > 0 {
		confidence += result.SuggestedType[0].Score * 0.4
	}

	// Domain confidence
	if len(result.SuggestedDomain) > 0 && result.SuggestedDomain[0].Score > 0.5 {
		confidence += 0.4
	} else if len(result.SuggestedDomain) > 0 {
		confidence += result.SuggestedDomain[0].Score * 0.4
	}

	// Tags confidence
	if len(result.SuggestedTags) > 0 {
		confidence += 0.2
	}

	return confidence
}

func parseFrontmatterForClassify(content string) (map[string]any, string) {
	meta := make(map[string]any)
	body := content

	if !strings.HasPrefix(content, "---") {
		return meta, body
	}

	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		return meta, body
	}

	yaml.Unmarshal([]byte(parts[0]), &meta)
	body = strings.TrimSpace(parts[1])

	return meta, body
}

func getDefaultDocTypes() map[string]DocTypeDef {
	return map[string]DocTypeDef{
		"port": {
			Name:           "Port 명세",
			Description:    "작업 포트 명세 문서",
			RequiredFields: []string{"title", "status", "priority"},
			Keywords:       []string{"port", "작업", "task", "목표", "범위", "산출물"},
			FilePatterns:   []string{`ports/.*\.md$`, `port-.*\.md$`},
		},
		"adr": {
			Name:           "Architecture Decision Record",
			Description:    "아키텍처 결정 기록",
			RequiredFields: []string{"title", "status", "decision_date"},
			Keywords:       []string{"decision", "결정", "context", "배경", "alternatives", "대안"},
			FilePatterns:   []string{`decisions/.*\.md$`, `adr-.*\.md$`},
		},
		"concept": {
			Name:           "개념 문서",
			Description:    "도메인 개념 정의",
			RequiredFields: []string{"title", "domain"},
			Keywords:       []string{"개념", "정의", "concept", "definition", "설명"},
			FilePatterns:   []string{`concepts/.*\.md$`},
		},
		"guide": {
			Name:           "가이드 문서",
			Description:    "사용 가이드",
			RequiredFields: []string{"title"},
			Keywords:       []string{"가이드", "guide", "사용법", "how to", "튜토리얼"},
			FilePatterns:   []string{`guides/.*\.md$`, `.*-guide\.md$`},
		},
		"convention": {
			Name:           "컨벤션 문서",
			Description:    "코딩/문서 컨벤션",
			RequiredFields: []string{"title"},
			Keywords:       []string{"컨벤션", "convention", "규칙", "rule", "표준"},
			FilePatterns:   []string{`conventions/.*\.md$`},
		},
		"session": {
			Name:           "세션 기록",
			Description:    "작업 세션 기록",
			RequiredFields: []string{"title", "date"},
			Keywords:       []string{"session", "세션", "작업 기록", "로그"},
			FilePatterns:   []string{`sessions/.*\.md$`},
		},
	}
}

func getDefaultDomains() map[string]DomainDef {
	return map[string]DomainDef{
		"auth": {
			Name:        "인증/인가",
			Description: "인증 및 권한 관리",
			Keywords:    []string{"auth", "인증", "로그인", "token", "jwt", "oauth", "permission", "권한"},
		},
		"api": {
			Name:        "API",
			Description: "API 설계 및 구현",
			Keywords:    []string{"api", "endpoint", "rest", "graphql", "request", "response"},
		},
		"database": {
			Name:        "데이터베이스",
			Description: "데이터 저장 및 관리",
			Keywords:    []string{"database", "db", "sql", "query", "schema", "migration"},
		},
		"frontend": {
			Name:        "프론트엔드",
			Description: "UI/UX 구현",
			Keywords:    []string{"frontend", "ui", "component", "react", "css", "html"},
		},
		"backend": {
			Name:        "백엔드",
			Description: "서버 로직",
			Keywords:    []string{"backend", "server", "service", "handler"},
		},
		"infra": {
			Name:        "인프라",
			Description: "인프라 및 배포",
			Keywords:    []string{"infra", "docker", "k8s", "kubernetes", "deploy", "ci/cd"},
		},
	}
}

// GetTaxonomy returns the loaded taxonomy
func (c *ClassifierService) GetTaxonomy() *Taxonomy {
	if c.taxonomy == nil {
		c.LoadTaxonomy()
	}
	return c.taxonomy
}

// ListDomains returns all available domains
func (c *ClassifierService) ListDomains() []DomainDef {
	if c.taxonomy == nil {
		c.LoadTaxonomy()
	}
	domains := []DomainDef{}
	for id, d := range c.taxonomy.Domains {
		d.Name = id // Ensure ID is accessible
		domains = append(domains, d)
	}
	return domains
}

// ListDocTypes returns all available document types
func (c *ClassifierService) ListDocTypes() []DocTypeDef {
	if c.taxonomy == nil {
		c.LoadTaxonomy()
	}
	types := []DocTypeDef{}
	for id, t := range c.taxonomy.DocTypes {
		t.Name = id // Ensure ID is accessible
		types = append(types, t)
	}
	return types
}
