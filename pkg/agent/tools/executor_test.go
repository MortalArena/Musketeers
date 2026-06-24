package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func newTestExecutor(t *testing.T) *ToolExecutor {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	tmpDir := t.TempDir()
	return NewToolExecutor(tmpDir, logger)
}

func newTestExecutorWithRegistry(t *testing.T, registry *ToolRegistry, role AgentRole) *ToolExecutor {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	tmpDir := t.TempDir()
	return NewToolExecutorWithRegistry(tmpDir, registry, role, logger)
}

func TestNewToolExecutorWithRegistry(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(ToolDefinition{
		Name:         "test_tool",
		RequiredRole: RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "done", nil
		},
	})

	exec := newTestExecutorWithRegistry(t, registry, RoleRegular)
	if exec.GetRegistry() == nil {
		t.Fatal("expected registry to be set")
	}
	if exec.GetAgentRole() != RoleRegular {
		t.Fatalf("expected RoleRegular, got %s", exec.GetAgentRole())
	}

	tools := exec.GetAvailableTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
}

func TestExecuteToolWithRegistry(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(ToolDefinition{
		Name:         "greet",
		Category:     CategoryChannel,
		Action:       ActionWrite,
		RequiredRole: RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			name, _ := params["name"].(string)
			return "Hello, " + name, nil
		},
	})

	exec := newTestExecutorWithRegistry(t, registry, RoleRegular)

	result, err := exec.ExecuteTool(context.Background(), "task1", "greet", map[string]interface{}{
		"name": "Agent",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	r, ok := result.(*ToolResult)
	if !ok {
		t.Fatalf("expected *ToolResult, got %T", result)
	}
	if !r.Success {
		t.Fatal("expected success")
	}
	if r.Data != "Hello, Agent" {
		t.Fatalf("expected 'Hello, Agent', got '%v'", r.Data)
	}
}

func TestExecuteToolPermissionDenied(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(ToolDefinition{
		Name:         "admin_only",
		RequiredRole: RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "secret", nil
		},
	})

	exec := newTestExecutorWithRegistry(t, registry, RoleRegular)

	_, err := exec.ExecuteTool(context.Background(), "task1", "admin_only", nil)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
}

func TestExecutorToolNotFound(t *testing.T) {
	registry := NewToolRegistry()
	exec := newTestExecutorWithRegistry(t, registry, RoleRegular)

	_, err := exec.ExecuteTool(context.Background(), "task1", "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestExecuteToolLimit(t *testing.T) {
	exec := newTestExecutor(t)
	exec.MaxToolCallsPerTask = 2

	for i := 0; i < 2; i++ {
		_, err := exec.ExecuteTool(context.Background(), "limited", "http_request", map[string]interface{}{
			"url":    "https://example.com",
			"method": "GET",
		})
		if err != nil {
			// Allow network errors (not rate limit)
			if err.Error() == "تجاوز الحد الأقصى لاستدعاءات الأدوات: 2" {
				t.Fatalf("unexpected rate limit on iteration %d", i)
			}
		}
	}

	// Third call should hit limit
	_, err := exec.ExecuteTool(context.Background(), "limited", "http_request", nil)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
}

func TestReadWriteFile(t *testing.T) {
	exec := newTestExecutor(t)

	// Write
	writeResult, err := exec.ExecuteTool(context.Background(), "task1", "write_file", map[string]interface{}{
		"path":    "test.txt",
		"content": "hello world",
	})
	if err != nil {
		t.Fatalf("write_file failed: %v", err)
	}
	wm, ok := writeResult.(map[string]interface{})
	if !ok || wm["success"] != true {
		t.Fatal("write_file expected success")
	}

	// Read
	readResult, err := exec.ExecuteTool(context.Background(), "task1", "read_file", map[string]interface{}{
		"path": "test.txt",
	})
	if err != nil {
		t.Fatalf("read_file failed: %v", err)
	}
	rm, ok := readResult.(map[string]interface{})
	if !ok || rm["content"] != "hello world" {
		t.Fatalf("expected 'hello world', got '%v'", rm["content"])
	}
}

func TestListFiles(t *testing.T) {
	exec := newTestExecutor(t)

	// Create a test file first
	exec.ExecuteTool(context.Background(), "task1", "write_file", map[string]interface{}{
		"path":    "list_test.txt",
		"content": "test",
	})

	result, err := exec.ExecuteTool(context.Background(), "task1", "file_list", map[string]interface{}{
		"path": ".",
	})
	if err != nil {
		t.Fatalf("file_list failed: %v", err)
	}
	rm, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	files, ok := rm["files"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected files array")
	}
	if len(files) == 0 {
		t.Fatal("expected at least 1 file")
	}
}

func TestDeleteFile(t *testing.T) {
	exec := newTestExecutor(t)

	// Create
	exec.ExecuteTool(context.Background(), "task1", "write_file", map[string]interface{}{
		"path":    "to_delete.txt",
		"content": "delete me",
	})

	// Delete
	delResult, err := exec.ExecuteTool(context.Background(), "task1", "file_delete", map[string]interface{}{
		"path": "to_delete.txt",
	})
	if err != nil {
		t.Fatalf("file_delete failed: %v", err)
	}
	dm, ok := delResult.(map[string]interface{})
	if !ok || dm["success"] != true {
		t.Fatal("file_delete expected success")
	}

	// Verify deleted
	if _, err := os.Stat(filepath.Join(exec.AllowedBasePath, "to_delete.txt")); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
}

func TestPathSecurity(t *testing.T) {
	exec := newTestExecutor(t)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"absolute path", "/etc/passwd", true},
		{"parent dir", "../../etc/passwd", true},
		{"clean path", "valid_file.txt", false},
		{"subdir path", "subdir/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := exec.ExecuteTool(context.Background(), "task1", "read_file", map[string]interface{}{
				"path": tt.path,
			})
			if tt.wantErr && err == nil {
				t.Errorf("expected error for path %s", tt.path)
			}
			if !tt.wantErr && err != nil {
				// File might not exist, but shouldn't be security error
				if err.Error() == "المسار غير مسموح: "+tt.path {
					t.Errorf("unexpected security denial for path %s", tt.path)
				}
			}
		})
	}
}

func TestSetAgentRole(t *testing.T) {
	exec := newTestExecutor(t)
	if exec.GetAgentRole() != RoleRegular {
		t.Fatalf("default role should be RoleRegular")
	}

	exec.SetAgentRole(RoleManager)
	if exec.GetAgentRole() != RoleManager {
		t.Fatalf("expected RoleManager after SetAgentRole")
	}
}

func TestResetTaskCallCount(t *testing.T) {
	exec := newTestExecutor(t)

	exec.ExecuteTool(context.Background(), "task1", "http_request", map[string]interface{}{
		"url":    "https://example.com",
		"method": "GET",
	})

	if exec.GetTaskCallCount("task1") == 0 {
		t.Fatal("expected task call count to be > 0")
	}

	exec.ResetTaskCallCount("task1")
	if exec.GetTaskCallCount("task1") != 0 {
		t.Fatal("expected task call count to be 0 after reset")
	}
}

func TestIsPrivateURL(t *testing.T) {
	tests := []struct {
		url      string
		private  bool
	}{
		{"https://example.com", false},
		{"http://example.com", true},    // http not allowed
		{"https://localhost:8080", true},
		{"https://127.0.0.1:8080", true},
		{"https://192.168.1.1", true},
		{"https://10.0.0.1", true},
		{"https://169.254.169.254", true},
		{"https://metadata.google.internal", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isPrivateURL(tt.url); got != tt.private {
				t.Errorf("isPrivateURL(%s) = %v, want %v", tt.url, got, tt.private)
			}
		})
	}
}
