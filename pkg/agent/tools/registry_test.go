package tools

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestNewToolRegistry(t *testing.T) {
	tr := NewToolRegistry()
	if tr == nil {
		t.Fatal("NewToolRegistry() returned nil")
	}
	if tr.Count() != 0 {
		t.Fatalf("expected 0 tools, got %d", tr.Count())
	}
}

func TestRegister(t *testing.T) {
	tr := NewToolRegistry()

	err := tr.Register(ToolDefinition{
		Name:         "test_tool",
		Description:  "test tool",
		Category:     CategoryMemory,
		Action:       ActionRead,
		RequiredRole: RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tr.Count() != 1 {
		t.Fatalf("expected 1 tool, got %d", tr.Count())
	}
}

func TestRegisterDuplicate(t *testing.T) {
	tr := NewToolRegistry()
	handler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return "ok", nil
	}

	tr.Register(ToolDefinition{Name: "dup", Handler: handler})
	err := tr.Register(ToolDefinition{Name: "dup", Handler: handler})
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	tr := NewToolRegistry()
	err := tr.Register(ToolDefinition{
		Name: "",
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestRegisterNilHandler(t *testing.T) {
	tr := NewToolRegistry()
	err := tr.Register(ToolDefinition{Name: "nil_handler"})
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

func TestGet(t *testing.T) {
	tr := NewToolRegistry()
	handler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return "found", nil
	}
	tr.Register(ToolDefinition{Name: "my_tool", Handler: handler})

	def, err := tr.Get("my_tool")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if def.Name != "my_tool" {
		t.Fatalf("expected my_tool, got %s", def.Name)
	}
}

func TestGetNotFound(t *testing.T) {
	tr := NewToolRegistry()
	_, err := tr.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
}

func TestHasTool(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name: "exists",
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})

	if !tr.HasTool("exists") {
		t.Fatal("expected HasTool to return true")
	}
	if tr.HasTool("missing") {
		t.Fatal("expected HasTool to return false")
	}
}

func TestGetToolsByRole(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name:         "manager_only",
		RequiredRole: RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})
	tr.Register(ToolDefinition{
		Name:         "any_role",
		RequiredRole: RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})
	tr.Register(ToolDefinition{
		Name:         "regular_too",
		RequiredRole: RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})

	// Manager should see all 3
	managerTools := tr.GetToolsByRole(RoleManager)
	if len(managerTools) != 3 {
		t.Fatalf("expected 3 tools for manager, got %d", len(managerTools))
	}

	// Regular should see 2 (not manager_only)
	regularTools := tr.GetToolsByRole(RoleRegular)
	if len(regularTools) != 2 {
		t.Fatalf("expected 2 tools for regular, got %d", len(regularTools))
	}

	// Verify regular doesn't get manager_only
	for _, tool := range regularTools {
		if tool.Name == "manager_only" {
			t.Fatal("regular should not see manager_only tool")
		}
	}
}

func TestExecute(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name:         "greet",
		RequiredRole: RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			name, _ := params["name"].(string)
			return "Hello, " + name, nil
		},
	})

	result, err := tr.Execute(context.Background(), RoleRegular, "greet", map[string]interface{}{
		"name": "Agent",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if result.Data != "Hello, Agent" {
		t.Fatalf("expected 'Hello, Agent', got '%v'", result.Data)
	}
}

func TestExecutePermissionDenied(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name:         "secret",
		RequiredRole: RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "secret data", nil
		},
	})

	_, err := tr.Execute(context.Background(), RoleRegular, "secret", nil)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}

func TestRegistryExecuteToolNotFound(t *testing.T) {
	tr := NewToolRegistry()
	_, err := tr.Execute(context.Background(), RoleAny, "missing", nil)
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestRegisterBatch(t *testing.T) {
	tr := NewToolRegistry()
	defs := []ToolDefinition{
		{
			Name: "tool1",
			Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return 1, nil
			},
		},
		{
			Name: "tool2",
			Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return 2, nil
			},
		},
	}

	if err := tr.RegisterBatch(defs); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tr.Count() != 2 {
		t.Fatalf("expected 2 tools, got %d", tr.Count())
	}
}

func TestGetCategories(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name:     "a",
		Category: CategoryMemory,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})
	tr.Register(ToolDefinition{
		Name:     "b",
		Category: CategorySkills,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})

	cats := tr.GetCategories()
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
}

func TestGetToolsByCategory(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{
		Name:     "mem1",
		Category: CategoryMemory,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})
	tr.Register(ToolDefinition{
		Name:     "mem2",
		Category: CategoryMemory,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	})

	tools := tr.GetToolsByCategory(CategoryMemory)
	if len(tools) != 2 {
		t.Fatalf("expected 2 memory tools, got %d", len(tools))
	}
}

func TestConcurrentAccess(t *testing.T) {
	tr := NewToolRegistry()
	handler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return nil, nil
	}

	// Concurrent register
	t.Run("ParallelRegister", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 10; i++ {
			name := fmt.Sprintf("concurrent_%d", i)
			tr.Register(ToolDefinition{Name: name, Handler: handler})
		}
	})

	// Concurrent read
	t.Run("ParallelRead", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 10; i++ {
			tr.GetToolsByRole(RoleAny)
			tr.GetCategories()
			tr.Count()
		}
	})
}

func TestToolDefinitionHasPermission(t *testing.T) {
	tests := []struct {
		name         string
		requiredRole AgentRole
		agentRole    AgentRole
		expected     bool
	}{
		{"any + manager", RoleAny, RoleManager, true},
		{"any + regular", RoleAny, RoleRegular, true},
		{"manager + manager", RoleManager, RoleManager, true},
		{"manager + regular", RoleManager, RoleRegular, false},
		{"regular + regular", RoleRegular, RoleRegular, true},
		{"regular + manager", RoleRegular, RoleManager, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &ToolDefinition{RequiredRole: tt.requiredRole}
			if got := def.HasPermission(tt.agentRole); got != tt.expected {
				t.Errorf("HasPermission(%s) = %v, want %v", tt.agentRole, got, tt.expected)
			}
		})
	}
}

func TestNewToolResult(t *testing.T) {
	r := NewToolResult("data")
	if !r.Success {
		t.Fatal("expected success")
	}
	if r.Data != "data" {
		t.Fatalf("expected 'data', got '%v'", r.Data)
	}
}

func TestNewToolError(t *testing.T) {
	r := NewToolError(nil)
	if r.Success {
		t.Fatal("expected failure")
	}
	if r.Error != "unknown error" {
		t.Fatalf("expected 'unknown error', got '%s'", r.Error)
	}

	sampleErr := errors.New("sample error")
	r = NewToolError(sampleErr)
	if r.Error != sampleErr.Error() {
		t.Fatalf("expected '%s', got '%s'", sampleErr.Error(), r.Error)
	}
}
