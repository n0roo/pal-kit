package config

import (
	"os"
	"path/filepath"
)

const (
	// GlobalDirName is the name of the global PAL directory
	GlobalDirName = ".pal"
	// ProjectDirName is the name of the project-level directory
	ProjectDirName = ".claude"
)

// GlobalDir returns the global PAL directory path (~/.pal)
func GlobalDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, GlobalDirName)
}

// GlobalDBPath returns the global database path (~/.pal/pal.db)
func GlobalDBPath() string {
	return filepath.Join(GlobalDir(), "pal.db")
}

// GlobalAgentsDir returns the global agents directory (~/.pal/agents)
func GlobalAgentsDir() string {
	return filepath.Join(GlobalDir(), "agents")
}

// GlobalConventionsDir returns the global conventions directory (~/.pal/conventions)
func GlobalConventionsDir() string {
	return filepath.Join(GlobalDir(), "conventions")
}

// GlobalTemplatesDir returns the global templates directory (~/.pal/templates)
func GlobalTemplatesDir() string {
	return filepath.Join(GlobalDir(), "templates")
}

// GlobalPackagesDir returns the global packages directory (~/.pal/packages)
func GlobalPackagesDir() string {
	return filepath.Join(GlobalDir(), "packages")
}

// GlobalPalDir is an alias for GlobalDir for package compatibility
func GlobalPalDir() string {
	return GlobalDir()
}

// GlobalConfigPath returns the global config file path (~/.pal/config.yaml)
func GlobalConfigPath() string {
	return filepath.Join(GlobalDir(), "config.yaml")
}

// IsInstalled checks if PAL is globally installed
func IsInstalled() bool {
	_, err := os.Stat(GlobalDBPath())
	return err == nil
}

// ProjectDir returns the project-level PAL directory (.claude)
func ProjectDir(projectRoot string) string {
	return filepath.Join(projectRoot, ProjectDirName)
}

// ProjectSettingsPath returns the project settings.json path
func ProjectSettingsPath(projectRoot string) string {
	return filepath.Join(ProjectDir(projectRoot), "settings.json")
}

// ProjectAgentsDir returns the project agents directory
func ProjectAgentsDir(projectRoot string) string {
	return filepath.Join(projectRoot, "agents")
}

// ProjectConventionsDir returns the project conventions directory
func ProjectConventionsDir(projectRoot string) string {
	return filepath.Join(projectRoot, "conventions")
}

// ProjectPortsDir returns the project ports directory
func ProjectPortsDir(projectRoot string) string {
	return filepath.Join(projectRoot, "ports")
}

// ClaudeMDPath returns the CLAUDE.md path for a project
func ClaudeMDPath(projectRoot string) string {
	return filepath.Join(projectRoot, "CLAUDE.md")
}

// FindProjectRoot finds the project root by looking for .claude directory
func FindProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ProjectDirName)); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return cwd // fallback to current directory
}

// EnsureGlobalDirs creates all global directories
func EnsureGlobalDirs() error {
	dirs := []string{
		GlobalDir(),
		GlobalAgentsDir(),
		GlobalConventionsDir(),
		GlobalTemplatesDir(),
		GlobalPackagesDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// EnsureProjectDirs creates project-level directories
func EnsureProjectDirs(projectRoot string) error {
	dirs := []string{
		ProjectDir(projectRoot),
		filepath.Join(ProjectDir(projectRoot), "rules"),
		filepath.Join(ProjectDir(projectRoot), "hooks"),
		ProjectAgentsDir(projectRoot),
		ProjectConventionsDir(projectRoot),
		ProjectPortsDir(projectRoot),
		filepath.Join(projectRoot, "docs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
