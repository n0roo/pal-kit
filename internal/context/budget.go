package context

import (
	"errors"
	"sort"

	"github.com/n0roo/pal-kit/internal/config"
)

// Category constants for context allocation
const (
	CategoryPortSpec      = "port_spec"
	CategoryConventions   = "conventions"
	CategoryRecentChanges = "recent_changes"
	CategoryRelatedDocs   = "related_docs"
	CategorySessionInfo   = "session_info"
)

// ErrBudgetExceeded is returned when the token budget is exceeded
var ErrBudgetExceeded = errors.New("token budget exceeded")

// BudgetItem represents a single item in the budget
type BudgetItem struct {
	ID       string
	Category string
	Name     string
	Path     string
	Content  string
	Tokens   int
	Priority int
	Loaded   bool
}

// BudgetManager manages token budget allocation across categories
type BudgetManager struct {
	Config     config.ContextConfig
	Counter    TokenCounter
	Items      []BudgetItem
	Allocation map[string]int // 카테고리별 할당된 토큰
	Used       map[string]int // 카테고리별 사용된 토큰
}

// NewBudgetManager creates a new budget manager
func NewBudgetManager(cfg config.ContextConfig) *BudgetManager {
	bm := &BudgetManager{
		Config:     cfg,
		Counter:    NewApproximateCounter(),
		Items:      make([]BudgetItem, 0),
		Allocation: make(map[string]int),
		Used:       make(map[string]int),
	}

	// 예산 할당 계산
	bm.calculateAllocation()

	return bm
}

// calculateAllocation calculates token allocation for each category
func (bm *BudgetManager) calculateAllocation() {
	total := bm.Config.TokenBudget
	alloc := bm.Config.Allocation

	bm.Allocation[CategoryPortSpec] = (total * alloc.PortSpec) / 100
	bm.Allocation[CategoryConventions] = (total * alloc.Conventions) / 100
	bm.Allocation[CategoryRecentChanges] = (total * alloc.RecentChanges) / 100
	bm.Allocation[CategoryRelatedDocs] = (total * alloc.RelatedDocs) / 100
	bm.Allocation[CategorySessionInfo] = (total * alloc.SessionInfo) / 100

	// 최소 보장 적용
	if bm.Allocation[CategoryPortSpec] < bm.Config.Minimum.PortSpec {
		bm.Allocation[CategoryPortSpec] = bm.Config.Minimum.PortSpec
	}
	if bm.Allocation[CategoryConventions] < bm.Config.Minimum.Conventions {
		bm.Allocation[CategoryConventions] = bm.Config.Minimum.Conventions
	}

	// 초기화
	for cat := range bm.Allocation {
		bm.Used[cat] = 0
	}
}

// AddItem adds an item to the budget
func (bm *BudgetManager) AddItem(item BudgetItem) error {
	// 토큰 수가 없으면 계산
	if item.Tokens == 0 && item.Content != "" {
		item.Tokens = bm.Counter.Count(item.Content)
	}

	remaining := bm.Remaining(item.Category)

	if item.Tokens > remaining {
		switch bm.Config.Strategy {
		case "priority":
			// 우선순위 낮은 항목 제거
			freed := bm.trimLowPriority(item.Category, item.Tokens-remaining, item.Priority)
			if freed < item.Tokens-remaining {
				return ErrBudgetExceeded
			}
		default:
			return ErrBudgetExceeded
		}
	}

	item.Loaded = true
	bm.Items = append(bm.Items, item)
	bm.Used[item.Category] += item.Tokens

	return nil
}

// CanFit checks if an item can fit in the budget
func (bm *BudgetManager) CanFit(category string, tokens int) bool {
	return tokens <= bm.Remaining(category)
}

// Remaining returns remaining tokens for a category
func (bm *BudgetManager) Remaining(category string) int {
	allocated := bm.Allocation[category]
	used := bm.Used[category]
	return allocated - used
}

// TotalRemaining returns total remaining tokens
func (bm *BudgetManager) TotalRemaining() int {
	total := 0
	for cat := range bm.Allocation {
		total += bm.Remaining(cat)
	}
	return total
}

// TotalUsed returns total used tokens
func (bm *BudgetManager) TotalUsed() int {
	total := 0
	for _, used := range bm.Used {
		total += used
	}
	return total
}

// trimLowPriority removes lower priority items to free up tokens
func (bm *BudgetManager) trimLowPriority(category string, needed int, minPriority int) int {
	// 같은 카테고리의 낮은 우선순위 항목 찾기
	var candidates []int
	for i, item := range bm.Items {
		if item.Category == category && item.Priority < minPriority && item.Loaded {
			candidates = append(candidates, i)
		}
	}

	// 우선순위 낮은 순으로 정렬
	sort.Slice(candidates, func(i, j int) bool {
		return bm.Items[candidates[i]].Priority < bm.Items[candidates[j]].Priority
	})

	freed := 0
	for _, idx := range candidates {
		if freed >= needed {
			break
		}
		item := &bm.Items[idx]
		item.Loaded = false
		bm.Used[category] -= item.Tokens
		freed += item.Tokens
	}

	return freed
}

// GetLoadedItems returns all loaded items
func (bm *BudgetManager) GetLoadedItems() []BudgetItem {
	var loaded []BudgetItem
	for _, item := range bm.Items {
		if item.Loaded {
			loaded = append(loaded, item)
		}
	}
	return loaded
}

// GetItemsByCategory returns loaded items for a category
func (bm *BudgetManager) GetItemsByCategory(category string) []BudgetItem {
	var items []BudgetItem
	for _, item := range bm.Items {
		if item.Category == category && item.Loaded {
			items = append(items, item)
		}
	}
	return items
}

// BudgetReport represents a budget usage report
type BudgetReport struct {
	Total          int            `json:"total"`
	Used           int            `json:"used"`
	Remaining      int            `json:"remaining"`
	UsagePercent   int            `json:"usage_percent"`
	ByCategory     map[string]int `json:"by_category"`
	LoadedItems    int            `json:"loaded_items"`
	PendingItems   int            `json:"pending_items"`
	CategoryDetail []CategoryReport `json:"category_detail"`
}

// CategoryReport represents a category's budget report
type CategoryReport struct {
	Category   string `json:"category"`
	Allocated  int    `json:"allocated"`
	Used       int    `json:"used"`
	Remaining  int    `json:"remaining"`
	ItemCount  int    `json:"item_count"`
}

// Report generates a budget report
func (bm *BudgetManager) Report() BudgetReport {
	report := BudgetReport{
		Total:        bm.Config.TokenBudget,
		Used:         bm.TotalUsed(),
		Remaining:    bm.TotalRemaining(),
		ByCategory:   make(map[string]int),
		CategoryDetail: make([]CategoryReport, 0),
	}

	if report.Total > 0 {
		report.UsagePercent = (report.Used * 100) / report.Total
	}

	// 카테고리별 상세
	categories := []string{CategoryPortSpec, CategoryConventions, CategoryRecentChanges, CategoryRelatedDocs, CategorySessionInfo}
	for _, cat := range categories {
		report.ByCategory[cat] = bm.Used[cat]

		catReport := CategoryReport{
			Category:  cat,
			Allocated: bm.Allocation[cat],
			Used:      bm.Used[cat],
			Remaining: bm.Remaining(cat),
		}

		// 항목 수 계산
		for _, item := range bm.Items {
			if item.Category == cat && item.Loaded {
				catReport.ItemCount++
			}
		}

		report.CategoryDetail = append(report.CategoryDetail, catReport)
	}

	// 로드된/대기 중 항목 수
	for _, item := range bm.Items {
		if item.Loaded {
			report.LoadedItems++
		} else {
			report.PendingItems++
		}
	}

	return report
}

// Clear clears all items and resets usage
func (bm *BudgetManager) Clear() {
	bm.Items = make([]BudgetItem, 0)
	for cat := range bm.Used {
		bm.Used[cat] = 0
	}
}

// SetCounter sets a custom token counter
func (bm *BudgetManager) SetCounter(counter TokenCounter) {
	bm.Counter = counter
}

// FormatReport formats the budget report as a string
func (bm *BudgetManager) FormatReport() string {
	report := bm.Report()

	var result string
	result += "Context Budget: " + formatTokens(report.Used) + " / " + formatTokens(report.Total) + " tokens (" + intToString(report.UsagePercent) + "%)\n"
	result += "\n"
	result += "Loaded Documents:\n"

	for _, item := range bm.GetLoadedItems() {
		icon := getCategoryIcon(item.Category)
		status := "✓"
		if !item.Loaded {
			status = "(pending)"
		}
		result += "  " + icon + " " + item.Name + " (" + item.Category + ")  " + formatTokens(item.Tokens) + " " + status + "\n"
	}

	return result
}

// getCategoryIcon returns an emoji icon for a category
func getCategoryIcon(category string) string {
	switch category {
	case CategoryPortSpec:
		return "📄"
	case CategoryConventions:
		return "📘"
	case CategoryRecentChanges:
		return "📝"
	case CategoryRelatedDocs:
		return "📚"
	case CategorySessionInfo:
		return "ℹ️"
	default:
		return "📁"
	}
}
