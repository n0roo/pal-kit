package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "쉘 자동완성 스크립트 생성",
	Long: `지정한 쉘에 대한 자동완성 스크립트를 생성합니다.

Bash:
  # 현재 세션에서만 활성화
  $ source <(pal completion bash)

  # 영구 설정 (Linux)
  $ pal completion bash > /etc/bash_completion.d/pal

  # 영구 설정 (macOS with Homebrew)
  $ pal completion bash > $(brew --prefix)/etc/bash_completion.d/pal

Zsh:
  # 현재 세션에서만 활성화
  $ source <(pal completion zsh)

  # 영구 설정
  $ pal completion zsh > "${fpath[1]}/_pal"
  
  # 또는 Oh My Zsh 사용 시
  $ pal completion zsh > ~/.oh-my-zsh/completions/_pal

Fish:
  $ pal completion fish | source

  # 영구 설정
  $ pal completion fish > ~/.config/fish/completions/pal.fish

PowerShell:
  PS> pal completion powershell | Out-String | Invoke-Expression

  # 영구 설정
  PS> pal completion powershell > pal.ps1
  # 그 후 $PROFILE에 ". pal.ps1" 추가
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// 각 커맨드에 동적 완성 추가
	registerCompletions()
}

// registerCompletions adds dynamic completions for various commands
func registerCompletions() {
	// Session ID 완성
	if sessionEndCmd != nil {
		sessionEndCmd.ValidArgsFunction = completeSessionIDs
	}
	if sessionShowCmd != nil {
		sessionShowCmd.ValidArgsFunction = completeSessionIDs
	}
	if sessionUpdateCmd != nil {
		sessionUpdateCmd.ValidArgsFunction = completeSessionIDs
	}
	if sessionTreeCmd != nil {
		sessionTreeCmd.ValidArgsFunction = completeSessionIDs
	}

	// Port ID 완성
	if portShowCmd != nil {
		portShowCmd.ValidArgsFunction = completePortIDs
	}
	if portStatusCmd != nil {
		portStatusCmd.ValidArgsFunction = completePortIDs
	}
	if portDeleteCmd != nil {
		portDeleteCmd.ValidArgsFunction = completePortIDs
	}

	// Pipeline ID 완성
	if plShowCmd != nil {
		plShowCmd.ValidArgsFunction = completePipelineIDs
	}
	if plAddCmd != nil {
		plAddCmd.ValidArgsFunction = completePipelineAndPortIDs
	}
	if plStatusCmd != nil {
		plStatusCmd.ValidArgsFunction = completePipelineIDs
	}
	if plDeleteCmd != nil {
		plDeleteCmd.ValidArgsFunction = completePipelineIDs
	}
	if plPlanCmd != nil {
		plPlanCmd.ValidArgsFunction = completePipelineIDs
	}
	if plNextCmd != nil {
		plNextCmd.ValidArgsFunction = completePipelineIDs
	}
	if plRunCmd != nil {
		plRunCmd.ValidArgsFunction = completePipelineIDs
	}
	if plPortStatusCmd != nil {
		plPortStatusCmd.ValidArgsFunction = completePipelineIDs
	}
	if plExecCmd != nil {
		plExecCmd.ValidArgsFunction = completePipelineIDs
	}
	if plResetCmd != nil {
		plResetCmd.ValidArgsFunction = completePipelineIDs
	}

	// Agent ID 완성
	if agentShowCmd != nil {
		agentShowCmd.ValidArgsFunction = completeAgentIDs
	}
	if agentDeleteCmd != nil {
		agentDeleteCmd.ValidArgsFunction = completeAgentIDs
	}
	if agentPromptCmd != nil {
		agentPromptCmd.ValidArgsFunction = completeAgentIDs
	}

	// Lock 완성
	if lockReleaseCmd != nil {
		lockReleaseCmd.ValidArgsFunction = completeLockResources
	}

	// Escalation 완성
	if escResolveCmd != nil {
		escResolveCmd.ValidArgsFunction = completeEscalationIDs
	}
	if escDismissCmd != nil {
		escDismissCmd.ValidArgsFunction = completeEscalationIDs
	}
}

// completeSessionIDs provides session ID completion
func completeSessionIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	svc, cleanup, err := getSessionService()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer cleanup()

	sessions, err := svc.List(false, 50)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, s := range sessions {
		desc := s.Status
		if s.Title.Valid {
			desc = s.Title.String + " (" + s.Status + ")"
		}
		completions = append(completions, s.ID+"\t"+desc)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completePortIDs provides port ID completion
func completePortIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	svc, cleanup, err := getPortService()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer cleanup()

	ports, err := svc.List("", 50)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, p := range ports {
		desc := p.Status
		if p.Title.Valid {
			desc = p.Title.String + " (" + p.Status + ")"
		}
		completions = append(completions, p.ID+"\t"+desc)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completePipelineIDs provides pipeline ID completion
func completePipelineIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer cleanup()

	pipelines, err := svc.List("", 50)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, p := range pipelines {
		completions = append(completions, p.ID+"\t"+p.Name+" ("+p.Status+")")
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completePipelineAndPortIDs provides pipeline ID then port ID completion
func completePipelineAndPortIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completePipelineIDs(cmd, args, toComplete)
	}
	if len(args) == 1 {
		return completePortIDs(cmd, nil, toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// completeAgentIDs provides agent ID completion
func completeAgentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	svc, err := getAgentService()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agents, err := svc.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, a := range agents {
		completions = append(completions, a.ID+"\t"+a.Name+" ("+a.Type+")")
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeLockResources provides lock resource completion
func completeLockResources(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	svc, cleanup, err := getLockService()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer cleanup()

	locks, err := svc.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, l := range locks {
		completions = append(completions, l.Resource+"\t"+"by "+l.SessionID)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeEscalationIDs provides escalation ID completion
func completeEscalationIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	svc, cleanup, err := getEscalationService()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer cleanup()

	escalations, err := svc.List("open", 50)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, e := range escalations {
		issue := e.Issue
		if len(issue) > 30 {
			issue = issue[:27] + "..."
		}
		completions = append(completions, fmt.Sprintf("%d\t%s", e.ID, issue))
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
