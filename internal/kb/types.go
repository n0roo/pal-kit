package kb

// KBStatus represents KB status
type KBStatus struct {
	VaultPath   string         `json:"vault_path"`
	Initialized bool           `json:"initialized"`
	Version     string         `json:"version,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	Sections    map[string]int `json:"sections,omitempty"`
}

// DomainsConfig represents domains.yaml
type DomainsConfig struct {
	Version string            `yaml:"version"`
	Domains map[string]Domain `yaml:"domains"`
}

// Domain represents a domain definition
type Domain struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags,omitempty"`
	Owner       string   `yaml:"owner,omitempty"`
}

// DocTypesConfig represents doc-types.yaml
type DocTypesConfig struct {
	Version string             `yaml:"version"`
	Types   map[string]DocType `yaml:"types"`
}

// DocType represents a document type
type DocType struct {
	Name           string   `yaml:"name"`
	Template       string   `yaml:"template"`
	RequiredFields []string `yaml:"required_fields"`
}

// TagsConfig represents tags.yaml
type TagsConfig struct {
	Version   string              `yaml:"version"`
	Hierarchy map[string]TagGroup `yaml:"hierarchy"`
}

// TagGroup represents a tag group
type TagGroup struct {
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
}
