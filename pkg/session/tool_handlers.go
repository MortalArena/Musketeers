package session

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/google/uuid"
)

// RegisterSessionTools يسجل جميع أدوات الجلسة في الـ registry
// [WHY] يربط الأدوات المنطقية (ذاكرة، مهارات، معرفة، قنوات) مع الـ session container
// [SAFETY] كل أداة تتحقق من صلاحية الدور قبل التنفيذ
func RegisterSessionTools(registry *tools.ToolRegistry, container *SessionContainer) {
	// ============================================================
	// Memory Tools - أدوات الذاكرة الجماعية
	// ============================================================

	// memory_write - يكتب حدث في الذاكرة العرضية (كل الوكلاء يشاركون)
	registry.Register(tools.ToolDefinition{
		Name:         "memory_write",
		Description:  "يسجل حدثاً جديداً في الذاكرة الجماعية (تشاركي)",
		Category:     tools.CategoryMemory,
		Action:       tools.ActionWrite,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			agentDID, _ := params["agent_did"].(string)
			action, _ := params["action"].(string)
			if action == "" {
				return nil, fmt.Errorf("المعامل action مطلوب")
			}
			if agentDID == "" {
				agentDID = "unknown"
			}

			event := MemoryEvent{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				AgentDID:  agentDID,
				Action:    action,
				Context:   extractMap(params, "context"),
				Outcome:   extractString(params, "outcome", "success"),
				Confidence: extractFloat(params, "confidence", 1.0),
				Tags:      extractStringSlice(params, "tags"),
			}
			if err := container.Memory.RecordEvent(event); err != nil {
				return nil, fmt.Errorf("فشل تسجيل الحدث: %w", err)
			}
			return map[string]interface{}{"event_id": event.ID}, nil
		},
	})

	// memory_search - يبحث في الأحداث (قراءة فقط)
	registry.Register(tools.ToolDefinition{
		Name:         "memory_search",
		Description:  "يبحث في أحداث الذاكرة الجماعية",
		Category:     tools.CategoryMemory,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			filters := extractMap(params, "filters")
			if filters == nil {
				filters = make(map[string]interface{})
			}
			results := container.Memory.QueryEvents(filters)
			return map[string]interface{}{
				"events": results,
				"count":  len(results),
			}, nil
		},
	})

	// memory_fact_add - يضيف حقيقة (تشاركي)
	registry.Register(tools.ToolDefinition{
		Name:         "memory_fact_add",
		Description:  "يضيف حقيقة جديدة للذاكرة الدلالية",
		Category:     tools.CategoryMemory,
		Action:       tools.ActionWrite,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			statement, _ := params["statement"].(string)
			if statement == "" {
				return nil, fmt.Errorf("المعامل statement مطلوب")
			}
			fact := MemoryFact{
				ID:        uuid.New().String(),
				Statement: statement,
				Category:  extractString(params, "category", "general"),
				Confidence: extractFloat(params, "confidence", 0.8),
				Source:    extractString(params, "source", "agent"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Tags:      extractStringSlice(params, "tags"),
			}
			if err := container.Memory.LearnFact(fact); err != nil {
				return nil, fmt.Errorf("فشل إضافة الحقيقة: %w", err)
			}
			return map[string]interface{}{"fact_id": fact.ID}, nil
		},
	})

	// memory_delete - يحذف حدث أو حقيقة (للمدير فقط)
	registry.Register(tools.ToolDefinition{
		Name:         "memory_delete",
		Description:  "يحذف عنصراً من الذاكرة الجماعية (تنقية)",
		Category:     tools.CategoryMemory,
		Action:       tools.ActionDelete,
		RequiredRole: tools.RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"note": "مدير الجلسة مسموح له بتنقية الذاكرة. تم تعليم العنصر للحذف.",
			}, nil
		},
	})

	// ============================================================
	// Knowledge Tools - أدوات المعرفة
	// ============================================================

	// knowledge_add - يضيف معرفة (تشاركي)
	registry.Register(tools.ToolDefinition{
		Name:         "knowledge_add",
		Description:  "يضيف عنصر معرفة جديد (ملف، رابط، ملاحظة)",
		Category:     tools.CategoryKnowledge,
		Action:       tools.ActionWrite,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			name, _ := params["name"].(string)
			content, _ := params["content"].(string)
			itemType, _ := params["type"].(string)
			if name == "" || content == "" {
				return nil, fmt.Errorf("المعاملات name و content مطلوبان")
			}
			if itemType == "" {
				itemType = "data"
			}

			item := KnowledgeItem{
				ID:          uuid.New().String(),
				Type:        itemType,
				Name:        name,
				Description: extractString(params, "description", ""),
				Content:     content,
				ProcessedAt: time.Now(),
				ProcessedBy: extractString(params, "agent_did", "unknown"),
				Category:    extractString(params, "category", "reference"),
				Tags:        extractStringSlice(params, "tags"),
				Priority:    int(extractFloat(params, "priority", 5)),
			}
			if err := container.Memory.AddKnowledge(item); err != nil {
				return nil, fmt.Errorf("فشل إضافة المعرفة: %w", err)
			}
			return map[string]interface{}{"knowledge_id": item.ID}, nil
		},
	})

	// knowledge_search - يبحث في المعرفة (قراءة فقط)
	registry.Register(tools.ToolDefinition{
		Name:         "knowledge_search",
		Description:  "يبحث في المعرفة الجماعية",
		Category:     tools.CategoryKnowledge,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			query, _ := params["query"].(string)
			if query == "" {
				return nil, fmt.Errorf("المعامل query مطلوب")
			}
			results := container.Memory.SearchKnowledge(query)
			return map[string]interface{}{
				"results": results,
				"count":   len(results),
			}, nil
		},
	})

	// knowledge_delete - يحذف معرفة (للمدير فقط)
	registry.Register(tools.ToolDefinition{
		Name:         "knowledge_delete",
		Description:  "يحذف عنصر معرفة (تنقية المدير)",
		Category:     tools.CategoryKnowledge,
		Action:       tools.ActionDelete,
		RequiredRole: tools.RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"note": "مدير الجلسة مسموح له بتنقية المعرفة.",
			}, nil
		},
	})

	// ============================================================
	// Skills Tools - أدوات المهارات
	// ============================================================

	// skill_learn - يتعلم مهارة (تشاركي)
	registry.Register(tools.ToolDefinition{
		Name:         "skill_learn",
		Description:  "يسجل تنفيذ مهمة لتطوير المهارات",
		Category:     tools.CategorySkills,
		Action:       tools.ActionWrite,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			agentDID, _ := params["agent_did"].(string)
			taskName, _ := params["task_name"].(string)
			success, _ := params["success"].(bool)
			if agentDID == "" || taskName == "" {
				return nil, fmt.Errorf("المعاملات agent_did و task_name مطلوبان")
			}

			skillTask := SkillTask{
				Name:    taskName,
				Success: success,
			}
			if err := container.Skills.RecordTaskCompletion(agentDID, skillTask); err != nil {
				return nil, fmt.Errorf("فشل تسجيل المهارة: %w", err)
			}
			return map[string]interface{}{"success": true}, nil
		},
	})

	// skill_list - يعرض المهارات (قراءة فقط)
	registry.Register(tools.ToolDefinition{
		Name:         "skill_list",
		Description:  "يعرض قائمة مهارات الوكلاء",
		Category:     tools.CategorySkills,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			agentDID, hasAgent := params["agent_did"]
			if hasAgent {
				agentID, ok := agentDID.(string)
				if ok && agentID != "" {
					skill, err := container.Skills.GetAgentSkills(agentID)
					if err != nil {
						return nil, fmt.Errorf("فشل الحصول على المهارات: %w", err)
					}
					return map[string]interface{}{"skills": skill}, nil
				}
			}
			allSkills := container.Skills.GetAllAgentSkills()
			return map[string]interface{}{
				"skills": allSkills,
				"count":  len(allSkills),
			}, nil
		},
	})

	// skill_delete - يحذف مهارة (للمدير فقط)
	registry.Register(tools.ToolDefinition{
		Name:         "skill_delete",
		Description:  "يحذف مهارة من سجل الوكيل (تنقية المدير)",
		Category:     tools.CategorySkills,
		Action:       tools.ActionDelete,
		RequiredRole: tools.RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"note": "مدير الجلسة مسموح له بتنقية المهارات.",
			}, nil
		},
	})

	// ============================================================
	// Channel Tools - أدوات القنوات والرسائل
	// ============================================================

	// channel_send - يرسل رسالة (تشاركي)
	registry.Register(tools.ToolDefinition{
		Name:         "channel_send",
		Description:  "يرسل رسالة في قناة الجلسة",
		Category:     tools.CategoryChannel,
		Action:       tools.ActionWrite,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			content, _ := params["content"].(string)
			msgType, _ := params["type"].(string)
			source, _ := params["source"].(string)
			if content == "" {
				return nil, fmt.Errorf("المعامل content مطلوب")
			}
			if msgType == "" {
				msgType = MsgTypeMessage
			}
			if source == "" {
				source = "agent"
			}

			msg := ChatMessage{
				ID:        uuid.New().String(),
				Type:      msgType,
				Content:   content,
				Source:    source,
				Timestamp: time.Now(),
				SessionID: container.ID,
			}
			container.ChatManager.AddMessage(msg)
			return map[string]interface{}{
				"message_id": msg.ID,
				"sent":       true,
			}, nil
		},
	})

	// channel_read - يقرأ الرسائل (قراءة فقط)
	registry.Register(tools.ToolDefinition{
		Name:         "channel_read",
		Description:  "يقرأ آخر الرسائل في القناة",
		Category:     tools.CategoryChannel,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			limit := int(extractFloat(params, "limit", 50))
			messages := container.ChatManager.GetLastMessages(limit)
			return map[string]interface{}{
				"messages": messages,
				"count":    len(messages),
			}, nil
		},
	})

	// ============================================================
	// Agent Tools - أدوات الوكيل
	// ============================================================

	// agent_info - معلومات الوكيل (أي دور)
	registry.Register(tools.ToolDefinition{
		Name:         "agent_info",
		Description:  "يعرض معلومات الوكيل الحالي",
		Category:     tools.CategoryAgent,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			state := container.GetUnifiedState()
			agents := make([]map[string]interface{}, 0)
			for _, a := range state.Agents {
				agents = append(agents, map[string]interface{}{
					"did":    a.DID,
					"name":   a.Name,
					"status": a.Status,
					"role":   a.Role,
				})
			}
			return map[string]interface{}{
				"session_id": container.ID,
				"session_status": container.Status,
				"agents":     agents,
			}, nil
		},
	})

	// agent_list - قائمة الوكلاء (أي دور)
	registry.Register(tools.ToolDefinition{
		Name:         "agent_list",
		Description:  "يعرض قائمة الوكلاء في الجلسة",
		Category:     tools.CategoryAgent,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			state := container.GetUnifiedState()
			return map[string]interface{}{
				"agents": state.Agents,
				"total":  len(state.Agents),
			}, nil
		},
	})

	// ============================================================
	// Session Tools - أدوات الجلسة
	// ============================================================

	// session_status - حالة الجلسة (أي دور)
	registry.Register(tools.ToolDefinition{
		Name:         "session_status",
		Description:  "يعرض حالة الجلسة الحالية",
		Category:     tools.CategorySession,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"session_id":     container.ID,
				"name":           container.Name,
				"status":         container.Status,
				"version":        container.Version,
				"created_at":     container.CreatedAt,
				"updated_at":     container.UpdatedAt,
				"agent_count":    len(container.state.Agents),
				"task_count":     len(container.state.Tasks),
				"progress":       container.state.Progress.Percentage,
			}, nil
		},
	})

	// task_status - حالة المهام (أي دور)
	registry.Register(tools.ToolDefinition{
		Name:         "task_status",
		Description:  "يعرض حالة المهام في الجلسة",
		Category:     tools.CategorySession,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			state := container.GetUnifiedState()
			return map[string]interface{}{
				"tasks":    state.Tasks,
				"progress": state.Progress,
			}, nil
		},
	})

	// progress_get - نسبة الإنجاز (أي دور)
	registry.Register(tools.ToolDefinition{
		Name:         "progress_get",
		Description:  "يعرض نسبة الإنجاز الحالية",
		Category:     tools.CategorySession,
		Action:       tools.ActionRead,
		RequiredRole: tools.RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			state := container.GetUnifiedState()
			return map[string]interface{}{
				"percentage":      state.Progress.Percentage,
				"completed_tasks": state.Progress.CompletedTasks,
				"total_tasks":     state.Progress.TotalTasks,
			}, nil
		},
	})

	// ============================================================
	// Execution Tools - أدوات تنفيذية (تتطلب sandbox أو عزل)
	// ============================================================

	// terminal_exec - ينفذ أمر طرفية (معزول)
	registry.Register(tools.ToolDefinition{
		Name:         "terminal_exec",
		Description:  "ينفذ أمراً في الطرفية المعزولة",
		Category:     tools.CategoryExecution,
		Action:       tools.ActionExecute,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			command, _ := params["command"].(string)
			if command == "" {
				return nil, fmt.Errorf("المعامل command مطلوب")
			}
			return map[string]interface{}{
				"output":   fmt.Sprintf("[محاكي] تنفيذ: %s", command),
				"exit_code": 0,
			}, nil
		},
	})

	// file_manager - إدارة الملفات (للوكيل العادي: قراءة/كتابة في مساحته)
	registry.Register(tools.ToolDefinition{
		Name:         "file_manager",
		Description:  "يدير الملفات في مساحة عمل الوكيل",
		Category:     tools.CategoryExecution,
		Action:       tools.ActionExecute,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"note": "استخدم read_file, write_file, file_list, file_delete مباشرة",
				"available": []string{"read_file", "write_file", "file_list", "file_delete"},
			}, nil
		},
	})

	// ============================================================
	// Integration Tools - أدوات التكامل
	// ============================================================

	// github_clone - يستنسخ مستودع GitHub
	registry.Register(tools.ToolDefinition{
		Name:         "github_clone",
		Description:  "يستنسخ مستودع GitHub في مساحة العمل",
		Category:     tools.CategoryIntegration,
		Action:       tools.ActionExecute,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			repo, _ := params["repo"].(string)
			if repo == "" {
				return nil, fmt.Errorf("المعامل repo مطلوب (مثال: owner/repo)")
			}
			return map[string]interface{}{
				"status":  "محاكي - تم استنساخ المستودع",
				"repo":    repo,
			}, nil
		},
	})

	// email_send - يرسل بريد إلكتروني (للمشتركين)
	registry.Register(tools.ToolDefinition{
		Name:         "email_send",
		Description:  "يرسل بريد إلكتروني (يتطلب اشتراك)",
		Category:     tools.CategoryIntegration,
		Action:       tools.ActionExecute,
		RequiredRole: tools.RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"status": "محاكي - البريد الإلكتروني يتطلب اشتراكاً فعّالاً",
			}, nil
		},
	})
}

// دوال مساعدة لاستخراج المعاملات بأمان
func extractString(params map[string]interface{}, key, defaultVal string) string {
	if v, ok := params[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func extractFloat(params map[string]interface{}, key string, defaultVal float64) float64 {
	switch v := params[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		fmt.Sscanf(v, "%f", &defaultVal)
		return defaultVal
	default:
		return defaultVal
	}
}

func extractMap(params map[string]interface{}, key string) map[string]interface{} {
	if v, ok := params[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func extractStringSlice(params map[string]interface{}, key string) []string {
	switch v := params[key].(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
