package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.agents)
	assert.NotNil(t, registry.metadata)
	assert.NotNil(t, registry.stats)
}

func TestAgentRegistry_Register(t *testing.T) {
	registry := NewAgentRegistry()

	// إنشاء mock agent
	mockAgent := NewMockAgent()
	mockAgent.info.ID = "agent_123"
	mockAgent.info.Name = "Test Agent"
	mockAgent.info.Provider = "claude"
	mockAgent.info.Model = "claude-3-opus"

	metadata := &AgentMetadata{
		Name:     "Test Agent",
		Type:     AgentTypeAPI,
		Provider: "claude",
		Model:    "claude-3-opus",
		Tags:     []string{"test", "api"},
		Config:   map[string]interface{}{"key": "value"},
	}

	err := registry.Register(mockAgent, metadata)
	require.NoError(t, err)

	// التحقق من التسجيل
	agent, err := registry.Get("agent_123")
	require.NoError(t, err)
	assert.Equal(t, "agent_123", agent.GetInfo().ID)

	// التحقق من البيانات الوصفية
	retrievedMetadata, err := registry.GetMetadata("agent_123")
	require.NoError(t, err)
	assert.Equal(t, "Test Agent", retrievedMetadata.Name)
	assert.Equal(t, AgentTypeAPI, retrievedMetadata.Type)

	// التحقق من الإحصائيات
	stats, err := registry.GetStats("agent_123")
	require.NoError(t, err)
	assert.Equal(t, "agent_123", stats.AgentID)
	assert.Equal(t, 0, stats.TotalTasks)
	assert.Equal(t, 1.0, stats.SuccessRate)
}

func TestAgentRegistry_Register_Duplicate(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "agent_123"

	// تسجيل أول
	err := registry.Register(mockAgent, nil)
	require.NoError(t, err)

	// محاولة تسجيل نفس الوكيل مرة أخرى
	err = registry.Register(mockAgent, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestAgentRegistry_Register_NoMetadata(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "agent_456"
	mockAgent.info.Name = "Auto Agent"
	mockAgent.info.Type = AgentTypeLocal
	mockAgent.info.Provider = "ollama"
	mockAgent.info.Model = "llama2"

	// تسجيل بدون بيانات وصفية
	err := registry.Register(mockAgent, nil)
	require.NoError(t, err)

	// التحقق من إنشاء البيانات الوصفية تلقائياً
	metadata, err := registry.GetMetadata("agent_456")
	require.NoError(t, err)
	assert.Equal(t, "Auto Agent", metadata.Name)
	assert.Equal(t, AgentTypeLocal, metadata.Type)
	assert.Equal(t, "ollama", metadata.Provider)
	assert.NotZero(t, metadata.RegisteredAt)
}

func TestAgentRegistry_Unregister(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "agent_789"
	mockAgent.info.Name = "To Remove"
	mockAgent.info.Type = AgentTypeCLI

	// تسجيل
	err := registry.Register(mockAgent, nil)
	require.NoError(t, err)

	// إلغاء التسجيل
	err = registry.Unregister("agent_789")
	require.NoError(t, err)

	// التحقق من الإزالة
	_, err = registry.Get("agent_789")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAgentRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewAgentRegistry()

	err := registry.Unregister("non_existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAgentRegistry_Get(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "agent_001"
	mockAgent.info.Name = "Get Test"
	mockAgent.info.Type = AgentTypeAPI

	registry.Register(mockAgent, nil)

	agent, err := registry.Get("agent_001")
	require.NoError(t, err)
	assert.Equal(t, "agent_001", agent.GetInfo().ID)

	_, err = registry.Get("non_existent")
	assert.Error(t, err)
}

func TestAgentRegistry_ListAll(t *testing.T) {
	registry := NewAgentRegistry()

	// تسجيل عدة وكلاء
	for i := 0; i < 3; i++ {
		mockAgent := NewMockAgent()
		mockAgent.info.ID = string(rune('a' + i))
		mockAgent.info.Name = "Agent"
		mockAgent.info.Type = AgentTypeAPI
		registry.Register(mockAgent, nil)
	}

	agents := registry.ListAll()
	assert.Equal(t, 3, len(agents))
}

func TestAgentRegistry_ListByType(t *testing.T) {
	registry := NewAgentRegistry()

	// تسجيل وكلاء بأنواع مختلفة
	apiAgent := NewMockAgent()
	apiAgent.info.ID = "api_agent"
	apiAgent.info.Name = "API Agent"
	apiAgent.info.Type = AgentTypeAPI

	cliAgent := NewMockAgent()
	cliAgent.info.ID = "cli_agent"
	cliAgent.info.Name = "CLI Agent"
	cliAgent.info.Type = AgentTypeCLI

	registry.Register(apiAgent, nil)
	registry.Register(cliAgent, nil)

	apiAgents := registry.ListByType(AgentTypeAPI)
	assert.Equal(t, 1, len(apiAgents))
	assert.Equal(t, "api_agent", apiAgents[0].GetInfo().ID)

	cliAgents := registry.ListByType(AgentTypeCLI)
	assert.Equal(t, 1, len(cliAgents))
	assert.Equal(t, "cli_agent", cliAgents[0].GetInfo().ID)
}

func TestAgentRegistry_ListByCapability(t *testing.T) {
	registry := NewAgentRegistry()

	// تسجيل وكلاء بقدرات مختلفة
	codeGenAgent := NewMockAgent()
	codeGenAgent.info.ID = "code_gen"
	codeGenAgent.info.Name = "Code Generator"
	codeGenAgent.info.Type = AgentTypeAPI

	testAgent := NewMockAgent()
	testAgent.info.ID = "tester"
	testAgent.info.Name = "Tester"
	testAgent.info.Type = AgentTypeAPI

	registry.Register(codeGenAgent, nil)
	registry.Register(testAgent, nil)

	codeGenAgents := registry.ListByCapability(CapabilityCodeGeneration)
	assert.Equal(t, 2, len(codeGenAgents)) // MockAgent has code_generation by default

	testingAgents := registry.ListByCapability(CapabilityTesting)
	assert.Equal(t, 0, len(testingAgents)) // MockAgent doesn't have testing
}

func TestAgentRegistry_ListAvailable(t *testing.T) {
	registry := NewAgentRegistry()

	availableAgent := NewMockAgent()
	availableAgent.info.ID = "available"
	availableAgent.info.Name = "Available"
	availableAgent.info.Type = AgentTypeAPI

	unavailableAgent := NewMockAgent()
	unavailableAgent.info.ID = "unavailable"
	unavailableAgent.info.Name = "Unavailable"
	unavailableAgent.info.Type = AgentTypeAPI
	unavailableAgent.available = false
	unavailableAgent.status.IsAvailable = false

	registry.Register(availableAgent, nil)
	registry.Register(unavailableAgent, nil)

	availableAgents := registry.ListAvailable()
	assert.Equal(t, 1, len(availableAgents))
	assert.Equal(t, "available", availableAgents[0].GetInfo().ID)
}

func TestAgentRegistry_UpdateStats(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "stats_agent"
	mockAgent.info.Name = "Stats Agent"
	mockAgent.info.Type = AgentTypeAPI

	registry.Register(mockAgent, nil)

	// تحديث الإحصائيات
	err := registry.UpdateStats("stats_agent", true, 1000, 500*time.Millisecond)
	require.NoError(t, err)

	stats, err := registry.GetStats("stats_agent")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalTasks)
	assert.Equal(t, 1, stats.CompletedTasks)
	assert.Equal(t, 0, stats.FailedTasks)
	assert.Equal(t, 1000, stats.TotalTokens)
	assert.Equal(t, 500*time.Millisecond, stats.TotalDuration)
	assert.Equal(t, 1.0, stats.SuccessRate)

	// تحديث إحصائيات مهمة فاشلة
	err = registry.UpdateStats("stats_agent", false, 500, 300*time.Millisecond)
	require.NoError(t, err)

	stats, err = registry.GetStats("stats_agent")
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalTasks)
	assert.Equal(t, 1, stats.CompletedTasks)
	assert.Equal(t, 1, stats.FailedTasks)
	assert.Equal(t, 1500, stats.TotalTokens)
	assert.Equal(t, 0.5, stats.SuccessRate)
}

func TestAgentRegistry_UpdateStats_NotFound(t *testing.T) {
	registry := NewAgentRegistry()

	err := registry.UpdateStats("non_existent", true, 100, 100*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAgentRegistry_UpdateMetadata(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "meta_agent"
	mockAgent.info.Name = "Meta Agent"
	mockAgent.info.Type = AgentTypeAPI

	registry.Register(mockAgent, nil)

	// تحديث البيانات الوصفية
	newMetadata := &AgentMetadata{
		Name:     "Updated Agent",
		Type:     AgentTypeAPI,
		Provider: "openai",
		Model:    "gpt-4",
		Tags:     []string{"updated", "gpt"},
		Config:   map[string]interface{}{"new_key": "new_value"},
	}

	err := registry.UpdateMetadata("meta_agent", newMetadata)
	require.NoError(t, err)

	retrievedMetadata, err := registry.GetMetadata("meta_agent")
	require.NoError(t, err)
	assert.Equal(t, "Updated Agent", retrievedMetadata.Name)
	assert.Equal(t, "openai", retrievedMetadata.Provider)
	assert.Equal(t, "gpt-4", retrievedMetadata.Model)
}

func TestAgentRegistry_GetCount(t *testing.T) {
	registry := NewAgentRegistry()

	assert.Equal(t, 0, registry.GetCount())

	for i := 0; i < 5; i++ {
		mockAgent := NewMockAgent()
		mockAgent.info.ID = string(rune('a' + i))
		mockAgent.info.Name = "Agent"
		mockAgent.info.Type = AgentTypeAPI
		registry.Register(mockAgent, nil)
	}

	assert.Equal(t, 5, registry.GetCount())
}

func TestAgentRegistry_GetAvailableCount(t *testing.T) {
	registry := NewAgentRegistry()

	// تسجيل 3 وكلاء متاحين و 2 غير متاحين
	for i := 0; i < 3; i++ {
		mockAgent := NewMockAgent()
		mockAgent.info.ID = string(rune('a' + i))
		mockAgent.info.Name = "Available"
		mockAgent.info.Type = AgentTypeAPI
		registry.Register(mockAgent, nil)
	}

	for i := 0; i < 2; i++ {
		mockAgent := NewMockAgent()
		mockAgent.info.ID = string(rune('x' + i))
		mockAgent.info.Name = "Unavailable"
		mockAgent.info.Type = AgentTypeAPI
		mockAgent.available = false
		mockAgent.status.IsAvailable = false
		registry.Register(mockAgent, nil)
	}

	assert.Equal(t, 3, registry.GetAvailableCount())
}

func TestAgentRegistry_GetByProvider(t *testing.T) {
	registry := NewAgentRegistry()

	claudeAgent := NewMockAgent()
	claudeAgent.info.ID = "claude"
	claudeAgent.info.Name = "Claude"
	claudeAgent.info.Type = AgentTypeAPI
	claudeAgent.info.Provider = "claude"

	openaiAgent := NewMockAgent()
	openaiAgent.info.ID = "gpt"
	openaiAgent.info.Name = "GPT"
	openaiAgent.info.Type = AgentTypeAPI
	openaiAgent.info.Provider = "openai"

	registry.Register(claudeAgent, nil)
	registry.Register(openaiAgent, nil)

	claudeAgents := registry.GetByProvider("claude")
	assert.Equal(t, 1, len(claudeAgents))
	assert.Equal(t, "claude", claudeAgents[0].GetInfo().ID)

	openaiAgents := registry.GetByProvider("openai")
	assert.Equal(t, 1, len(openaiAgents))
	assert.Equal(t, "gpt", openaiAgents[0].GetInfo().ID)
}

func TestAgentRegistry_GetByModel(t *testing.T) {
	registry := NewAgentRegistry()

	opusAgent := NewMockAgent()
	opusAgent.info.ID = "opus"
	opusAgent.info.Name = "Opus"
	opusAgent.info.Type = AgentTypeAPI
	opusAgent.info.Provider = "claude"
	opusAgent.info.Model = "claude-3-opus"

	sonnetAgent := NewMockAgent()
	sonnetAgent.info.ID = "sonnet"
	sonnetAgent.info.Name = "Sonnet"
	sonnetAgent.info.Type = AgentTypeAPI
	sonnetAgent.info.Provider = "claude"
	sonnetAgent.info.Model = "claude-3-sonnet"

	registry.Register(opusAgent, nil)
	registry.Register(sonnetAgent, nil)

	opusAgents := registry.GetByModel("claude-3-opus")
	assert.Equal(t, 1, len(opusAgents))
	assert.Equal(t, "opus", opusAgents[0].GetInfo().ID)
}

func TestAgentRegistry_FindBestAgent(t *testing.T) {
	registry := NewAgentRegistry()

	// تسجيل وكلاء بقدرات مختلفة
	codeGenAgent := NewMockAgent()
	codeGenAgent.info.ID = "coder"
	codeGenAgent.info.Name = "Coder"
	codeGenAgent.info.Type = AgentTypeAPI

	testAgent := NewMockAgent()
	testAgent.info.ID = "tester"
	testAgent.info.Name = "Tester"
	testAgent.info.Type = AgentTypeAPI

	registry.Register(codeGenAgent, nil)
	registry.Register(testAgent, nil)

	// تحديث إحصائيات coder لجعله أفضل
	registry.UpdateStats("coder", true, 1000, 100*time.Millisecond)

	// البحث عن وكيل لإنشاء الكود
	bestAgent, err := registry.FindBestAgent([]AgentCapability{CapabilityCodeGeneration})
	require.NoError(t, err)
	assert.Equal(t, "coder", bestAgent.GetInfo().ID)
}

func TestAgentRegistry_FindBestAgent_NoMatch(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "agent"
	mockAgent.info.Name = "Agent"
	mockAgent.info.Type = AgentTypeAPI
	mockAgent.available = false // جعل الوكيل غير متاح
	mockAgent.status.IsAvailable = false

	registry.Register(mockAgent, nil)

	// البحث عن وكيل بقدرة غير موجودة - لا يوجد وكيل متاح
	_, err := registry.FindBestAgent([]AgentCapability{CapabilityDesign})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable agent found")
}

func TestAgentRegistry_SaveAndLoad(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewMockAgent()
	mockAgent.info.ID = "save_agent"
	mockAgent.info.Name = "Save Agent"
	mockAgent.info.Type = AgentTypeAPI

	metadata := &AgentMetadata{
		Name:     "Save Agent",
		Type:     AgentTypeAPI,
		Provider: "claude",
		Model:    "claude-3-opus",
		Tags:     []string{"save", "test"},
		Config:   map[string]interface{}{"key": "value"},
	}

	registry.Register(mockAgent, metadata)
	registry.UpdateStats("save_agent", true, 1000, 100*time.Millisecond)

	// حفظ
	data, err := registry.Save()
	require.NoError(t, err)
	assert.NotNil(t, data)

	// إنشاء سجل جديد وتحميل
	newRegistry := NewAgentRegistry()
	err = newRegistry.Load(data)
	require.NoError(t, err)

	// التحقق من البيانات المحملة
	loadedMetadata, err := newRegistry.GetMetadata("save_agent")
	require.NoError(t, err)
	assert.Equal(t, "Save Agent", loadedMetadata.Name)
	assert.Equal(t, "claude", loadedMetadata.Provider)

	loadedStats, err := newRegistry.GetStats("save_agent")
	require.NoError(t, err)
	assert.Equal(t, 1, loadedStats.TotalTasks)
	assert.Equal(t, 1.0, loadedStats.SuccessRate)
}

func TestAgentRegistry_CleanupInactive(t *testing.T) {
	registry := NewAgentRegistry()

	// تسجيل وكلاء
	activeAgent := NewMockAgent()
	activeAgent.info.ID = "active"
	activeAgent.info.Name = "Active"
	activeAgent.info.Type = AgentTypeAPI

	inactiveAgent := NewMockAgent()
	inactiveAgent.info.ID = "inactive"
	inactiveAgent.info.Name = "Inactive"
	inactiveAgent.info.Type = AgentTypeAPI

	registry.Register(activeAgent, nil)
	registry.Register(inactiveAgent, nil)

	// تحديث LastSeen للوكيل النشط
	registry.UpdateMetadata("active", &AgentMetadata{
		Name:     "Active",
		Type:     AgentTypeAPI,
		LastSeen: time.Now(),
	})

	// تعيين LastSeen للوكيل غير النشط قبل ساعة
	oldTime := time.Now().Add(-2 * time.Hour)
	registry.UpdateMetadata("inactive", &AgentMetadata{
		Name:     "Inactive",
		Type:     AgentTypeAPI,
		LastSeen: oldTime,
	})

	// تنظيف الوكلاء غير النشطين (أكثر من ساعة)
	removed := registry.CleanupInactive(1 * time.Hour)

	assert.Equal(t, 1, len(removed))
	assert.Contains(t, removed, "inactive")

	// التحقق من أن الوكيل النشط لا يزال موجود
	_, err := registry.Get("active")
	assert.NoError(t, err)

	// التحقق من أن الوكيل غير النشط تم إزالته
	_, err = registry.Get("inactive")
	assert.Error(t, err)
}
