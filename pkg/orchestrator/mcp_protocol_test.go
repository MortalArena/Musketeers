package orchestrator

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestMCPManagerCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	if mcpManager == nil {
		t.Fatal("فشل إنشاء MCPManager")
	}

	t.Log("تم إنشاء MCPManager بنجاح")
}

func TestMCPManagerStartStop(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// بدء MCPManager
	if err := mcpManager.Start(); err != nil {
		t.Fatalf("فشل بدء MCPManager: %v", err)
	}

	// إيقاف MCPManager
	if err := mcpManager.Stop(); err != nil {
		t.Fatalf("فشل إيقاف MCPManager: %v", err)
	}

	t.Log("تم بدء وإيقاف MCPManager بنجاح")
}

func TestMCPServerRegistration(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// إنشاء سيرفر جديد
	newServer := &MCPServer{
		ID:   "test-server",
		Name: "Test Server",
		Type: "test",
		Tools: []*MCPTool{
			{
				Name:        "test_tool",
				Description: "Test tool",
				InputSchema: map[string]interface{}{},
			},
		},
		Enabled: true,
		Config:  map[string]interface{}{},
	}

	// تسجيل السيرفر
	if err := mcpManager.RegisterServer(newServer); err != nil {
		t.Fatalf("فشل تسجيل السيرفر: %v", err)
	}

	// الحصول على السيرفر
	server, err := mcpManager.GetServer("test-server")
	if err != nil {
		t.Fatalf("فشل الحصول على السيرفر: %v", err)
	}

	if server.Name != "Test Server" {
		t.Errorf("اسم السيرفر غير صحيح: got %s, want Test Server", server.Name)
	}

	t.Log("تم تسجيل والحصول على السيرفر بنجاح")
}

func TestMCPListServers(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// بدء MCPManager
	if err := mcpManager.Start(); err != nil {
		t.Fatalf("فشل بدء MCPManager: %v", err)
	}
	defer mcpManager.Stop()

	// الحصول على قائمة السيرفرات
	servers := mcpManager.ListServers()

	if len(servers) == 0 {
		t.Error("يجب أن يكون هناك سيرفرات افتراضية")
	}

	t.Logf("عدد السيرفرات: %d", len(servers))
}

func TestMCPListTools(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// بدء MCPManager
	if err := mcpManager.Start(); err != nil {
		t.Fatalf("فشل بدء MCPManager: %v", err)
	}
	defer mcpManager.Stop()

	// الحصول على أدوات GitHub
	tools, err := mcpManager.ListTools("github")
	if err != nil {
		t.Fatalf("فشل الحصول على الأدوات: %v", err)
	}

	if len(tools) == 0 {
		t.Error("يجب أن يكون هناك أدوات GitHub")
	}

	t.Logf("عدد أدوات GitHub: %d", len(tools))
}

func TestMCPCallTool(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// بدء MCPManager
	if err := mcpManager.Start(); err != nil {
		t.Fatalf("فشل بدء MCPManager: %v", err)
	}
	defer mcpManager.Stop()

	// استدعاء أداة
	result, err := mcpManager.CallTool("github", "create_issue", map[string]interface{}{
		"repo":  "test/repo",
		"title": "Test Issue",
		"body":  "Test Body",
	})

	if err != nil {
		t.Fatalf("فشل استدعاء الأداة: %v", err)
	}

	if result == nil {
		t.Error("يجب أن يكون هناك نتيجة")
	}

	t.Log("تم استدعاء الأداة بنجاح")
}

func TestMCPListResources(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// بدء MCPManager
	if err := mcpManager.Start(); err != nil {
		t.Fatalf("فشل بدء MCPManager: %v", err)
	}
	defer mcpManager.Stop()

	// الحصول على موارد GitHub
	resources, err := mcpManager.ListResources("github")
	if err != nil {
		t.Fatalf("فشل الحصول على الموارد: %v", err)
	}

	if len(resources) == 0 {
		t.Error("يجب أن يكون هناك موارد GitHub")
	}

	t.Logf("عدد موارد GitHub: %d", len(resources))
}

func TestMCPReadResource(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// بدء MCPManager
	if err := mcpManager.Start(); err != nil {
		t.Fatalf("فشل بدء MCPManager: %v", err)
	}
	defer mcpManager.Stop()

	// قراءة مورد
	result, err := mcpManager.ReadResource("github", "repo://source_code")
	if err != nil {
		t.Fatalf("فشل قراءة المورد: %v", err)
	}

	if result == nil {
		t.Error("يجب أن يكون هناك نتيجة")
	}

	t.Log("تم قراءة المورد بنجاح")
}

func TestMCPGetMetrics(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء MCPManager
	mcpManager := NewMCPManager(eventBus, zap.NewNop())

	// بدء MCPManager
	if err := mcpManager.Start(); err != nil {
		t.Fatalf("فشل بدء MCPManager: %v", err)
	}
	defer mcpManager.Stop()

	// الحصول على المقاييس
	metrics := mcpManager.GetMetrics()

	if metrics == nil {
		t.Error("يجب أن تكون هناك مقاييس")
	}

	if metrics.ServersCount == 0 {
		t.Error("يجب أن يكون هناك سيرفرات")
	}

	t.Logf("المقاييس: %+v", metrics)
}
