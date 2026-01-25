package context

import (
	"os"
	"path/filepath"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/rules"
)

// BudgetService provides context budget management functionality
type BudgetService struct {
	db          *db.DB
	projectRoot string
	manager     *BudgetManager
}

// NewBudgetService creates a new budget service
func NewBudgetService(database *db.DB, projectRoot string) *BudgetService {
	// 설정 로드
	cfg := config.ContextConfig{
		TokenBudget: 15000,
		Allocation: config.ContextAllocation{
			PortSpec:      40,
			Conventions:   25,
			RecentChanges: 15,
			RelatedDocs:   15,
			SessionInfo:   5,
		},
		Strategy: "priority",
		Minimum: config.ContextMinimum{
			PortSpec:    2000,
			Conventions: 1000,
		},
	}

	// 프로젝트 설정에서 덮어쓰기
	if projectRoot != "" {
		if projectCfg, err := config.LoadProjectConfig(projectRoot); err == nil {
			if projectCfg.Context.TokenBudget > 0 {
				cfg = projectCfg.Context
			}
		}
	}

	return &BudgetService{
		db:          database,
		projectRoot: projectRoot,
		manager:     NewBudgetManager(cfg),
	}
}

// BudgetStatusReport represents the current budget status
type BudgetStatusReport struct {
	Total          int              `json:"total"`
	Used           int              `json:"used"`
	Remaining      int              `json:"remaining"`
	UsagePercent   int              `json:"usage_percent"`
	Items          []BudgetItemInfo `json:"items"`
	CategoryDetail []CategoryReport `json:"category_detail"`
}

// BudgetItemInfo represents info about a loaded item
type BudgetItemInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Tokens   int    `json:"tokens"`
	Loaded   bool   `json:"loaded"`
}

// GetCurrentStatus returns the current budget status
func (s *BudgetService) GetCurrentStatus() (*BudgetStatusReport, error) {
	// 현재 로드된 컨텍스트 수집
	s.collectCurrentContext()

	report := s.manager.Report()

	// 아이템 정보 변환
	items := make([]BudgetItemInfo, 0)
	for _, item := range s.manager.Items {
		items = append(items, BudgetItemInfo{
			ID:       item.ID,
			Name:     item.Name,
			Category: item.Category,
			Tokens:   item.Tokens,
			Loaded:   item.Loaded,
		})
	}

	return &BudgetStatusReport{
		Total:          report.Total,
		Used:           report.Used,
		Remaining:      report.Remaining,
		UsagePercent:   report.UsagePercent,
		Items:          items,
		CategoryDetail: report.CategoryDetail,
	}, nil
}

// collectCurrentContext collects all currently loaded context
func (s *BudgetService) collectCurrentContext() {
	s.manager.Clear()

	// 1. 활성 포트 명세 수집
	s.collectPortSpecs()

	// 2. 컨벤션 수집
	s.collectConventions()

	// 3. 세션 정보 수집
	s.collectSessionInfo()
}

// collectPortSpecs collects active port specifications
func (s *BudgetService) collectPortSpecs() {
	portSvc := port.NewService(s.db)
	runningPorts, err := portSvc.List("running", 10)
	if err != nil {
		return
	}

	for _, p := range runningPorts {
		if p.FilePath.Valid && p.FilePath.String != "" {
			specPath := p.FilePath.String
			if !filepath.IsAbs(specPath) {
				specPath = filepath.Join(s.projectRoot, specPath)
			}

			content, err := os.ReadFile(specPath)
			if err != nil {
				continue
			}

			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}

			s.manager.AddItem(BudgetItem{
				ID:       p.ID,
				Category: CategoryPortSpec,
				Name:     title,
				Path:     specPath,
				Content:  string(content),
				Priority: 100, // 높은 우선순위
			})
		}
	}
}

// collectConventions collects convention files from rules
func (s *BudgetService) collectConventions() {
	if s.projectRoot == "" {
		return
	}

	rulesSvc := rules.NewService(s.projectRoot)
	activeRules, err := rulesSvc.ListActiveRules()
	if err != nil {
		return
	}

	for _, ruleID := range activeRules {
		rulePath := rulesSvc.GetRulePath(ruleID)
		content, err := os.ReadFile(rulePath)
		if err != nil {
			continue
		}

		s.manager.AddItem(BudgetItem{
			ID:       ruleID,
			Category: CategoryConventions,
			Name:     ruleID + ".md",
			Path:     rulePath,
			Content:  string(content),
			Priority: 80,
		})
	}
}

// collectSessionInfo collects session information
func (s *BudgetService) collectSessionInfo() {
	// 세션 브리핑 파일
	briefingPath := filepath.Join(s.projectRoot, ".pal", "context", "session-briefing.md")
	if content, err := os.ReadFile(briefingPath); err == nil {
		s.manager.AddItem(BudgetItem{
			ID:       "session-briefing",
			Category: CategorySessionInfo,
			Name:     "session-briefing.md",
			Path:     briefingPath,
			Content:  string(content),
			Priority: 50,
		})
	}
}

// GetManager returns the underlying budget manager
func (s *BudgetService) GetManager() *BudgetManager {
	return s.manager
}
