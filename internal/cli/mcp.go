package cli

import (
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP Server 실행",
	Long: `Model Context Protocol (MCP) 서버를 실행합니다.
Claude Desktop과 연동하여 PAL Kit 도구를 사용할 수 있습니다.

Claude Desktop 설정 (claude_desktop_config.json):
{
  "mcpServers": {
    "pal-kit": {
      "command": "pal",
      "args": ["mcp", "--db", "~/.pal/pal.db", "--project", "/path/to/project"]
    }
  }
}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := GetDBPath()
		projectRoot, _ := cmd.Flags().GetString("project")

		if projectRoot == "" {
			projectRoot = GetProjectRoot()
		}

		server, err := mcp.NewServer(dbPath, projectRoot)
		if err != nil {
			return fmt.Errorf("MCP 서버 생성 실패: %w", err)
		}
		defer server.Close()

		return server.Run()
	},
}

var mcpConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "MCP 설정 출력",
	Long:  "Claude Desktop에서 사용할 MCP 설정을 출력합니다.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, _ := cmd.Flags().GetString("project")
		if projectRoot == "" {
			projectRoot = GetProjectRoot()
		}

		dbPath := GetDBPath()

		// Get the pal binary path
		palPath, err := os.Executable()
		if err != nil {
			palPath = "pal"
		}

		config := fmt.Sprintf(`{
  "mcpServers": {
    "pal-kit": {
      "command": "%s",
      "args": ["mcp", "--db", "%s", "--project", "%s"]
    }
  }
}`, palPath, dbPath, projectRoot)

		fmt.Println("Claude Desktop 설정 (claude_desktop_config.json):")
		fmt.Println()
		fmt.Println(config)
		fmt.Println()
		fmt.Println("설정 파일 위치:")
		fmt.Println("  macOS: ~/Library/Application Support/Claude/claude_desktop_config.json")
		fmt.Println("  Windows: %APPDATA%\\Claude\\claude_desktop_config.json")
		fmt.Println("  Linux: ~/.config/Claude/claude_desktop_config.json")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().StringP("project", "p", "", "프로젝트 루트 경로")

	mcpCmd.AddCommand(mcpConfigCmd)
	mcpConfigCmd.Flags().StringP("project", "p", "", "프로젝트 루트 경로")
}
