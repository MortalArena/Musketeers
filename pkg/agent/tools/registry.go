package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ToolRegistry السجل المركزي للأدوات - قلب النظام
// يسجل جميع الأدوات ويوزعها حسب دور الوكيل
type ToolRegistry struct {
	mu         sync.RWMutex
	definitions map[string]*ToolDefinition
	categories   map[ToolCategory][]string // category -> tool names
}

// NewToolRegistry ينشئ سجل أدوات جديد
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		definitions: make(map[string]*ToolDefinition),
		categories:  make(map[ToolCategory][]string),
	}
}

// Register يسجل أداة جديدة في السجل
// [SAFETY] يمنع تسجيل أداة بنفس الاسم مرتين
// [SAFETY] يتحقق من صحة المدخلات
func (tr *ToolRegistry) Register(def ToolDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if def.Handler == nil {
		return fmt.Errorf("tool handler is required for %s", def.Name)
	}
	if def.RequiredRole == "" {
		def.RequiredRole = RoleAny // الدور الافتراضي: أي دور
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.definitions[def.Name]; exists {
		return fmt.Errorf("tool already registered: %s", def.Name)
	}

	tr.definitions[def.Name] = &def
	tr.categories[def.Category] = append(tr.categories[def.Category], def.Name)

	return nil
}

// RegisterBatch يسجل مجموعة أدوات دفعة واحدة
// [SAFETY] إذا فشل تسجيل أداة، لا يتم تسجيل أي منها (atomic)
func (tr *ToolRegistry) RegisterBatch(defs []ToolDefinition) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for _, def := range defs {
		if _, exists := tr.definitions[def.Name]; exists {
			return fmt.Errorf("tool already registered: %s", def.Name)
		}
	}

	for _, def := range defs {
		tr.definitions[def.Name] = &def
		tr.categories[def.Category] = append(tr.categories[def.Category], def.Name)
	}

	return nil
}

// Get يحصل على تعريف أداة بالاسم
func (tr *ToolRegistry) Get(name string) (*ToolDefinition, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	def, exists := tr.definitions[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return def, nil
}

// GetAll يعيد جميع تعريفات الأدوات
func (tr *ToolRegistry) GetAll() []*ToolDefinition {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]*ToolDefinition, 0, len(tr.definitions))
	for _, def := range tr.definitions {
		result = append(result, def)
	}

	// ترتيب ثابت للاختبارات
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetToolsByRole يعيد الأدوات المسموحة لدور معين
func (tr *ToolRegistry) GetToolsByRole(role AgentRole) []ToolInfo {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]ToolInfo, 0)
	for _, def := range tr.definitions {
		if def.HasPermission(role) {
			result = append(result, ToolInfo{
				Name:         def.Name,
				Description:  def.Description,
				Category:     def.Category,
				Action:       def.Action,
				RequiredRole: def.RequiredRole,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// HasTool يتحقق من وجود أداة بالاسم
func (tr *ToolRegistry) HasTool(name string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	_, exists := tr.definitions[name]
	return exists
}

// Execute ينفذ أداة بعد التحقق من الصلاحية
// [SAFETY] يتحقق من وجود الأداة وصلاحية الدور قبل التنفيذ
func (tr *ToolRegistry) Execute(ctx context.Context, role AgentRole, toolName string, params map[string]interface{}) (*ToolResult, error) {
	def, err := tr.Get(toolName)
	if err != nil {
		return NewToolError(fmt.Errorf("tool not found: %s", toolName)), err
	}

	if !def.HasPermission(role) {
		return NewToolError(fmt.Errorf("permission denied: role %s cannot use tool %s", role, toolName)),
			fmt.Errorf("permission denied: role %s requires %s", role, def.RequiredRole)
	}

	result, err := def.Handler(ctx, params)
	if err != nil {
		return NewToolError(err), err
	}

	return NewToolResult(result), nil
}

// GetCategories يعيد جميع التصنيفات المسجلة
func (tr *ToolRegistry) GetCategories() []ToolCategory {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	categories := make([]ToolCategory, 0, len(tr.categories))
	for cat := range tr.categories {
		categories = append(categories, cat)
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i] < categories[j]
	})

	return categories
}

// GetToolsByCategory يعيد الأدوات في تصنيف معين
func (tr *ToolRegistry) GetToolsByCategory(category ToolCategory) []ToolInfo {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	names, exists := tr.categories[category]
	if !exists {
		return nil
	}

	result := make([]ToolInfo, 0, len(names))
	for _, name := range names {
		if def, ok := tr.definitions[name]; ok {
			result = append(result, ToolInfo{
				Name:         def.Name,
				Description:  def.Description,
				Category:     def.Category,
				Action:       def.Action,
				RequiredRole: def.RequiredRole,
			})
		}
	}

	return result
}

// Count يعيد عدد الأدوات المسجلة
func (tr *ToolRegistry) Count() int {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	return len(tr.definitions)
}
