package session

import (
	"container/heap"
	"context"
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewTaskManager(t *testing.T) {
	tm := NewTaskManager("test_session")
	assert.NotNil(t, tm)
	assert.Equal(t, "test_session", tm.sessionID)
	assert.NotNil(t, tm.pendingQueue)
	assert.NotNil(t, tm.runningTasks)
	assert.NotNil(t, tm.completedTasks)
	assert.NotNil(t, tm.failedTasks)
	assert.NotNil(t, tm.agentStates)
}

func TestTaskManager_CreateTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	inputs := map[string]interface{}{
		"param1": "value1",
		"param2": 123,
	}

	task, err := tm.CreateTask(ctx, "Test Task", "This is a test task", PriorityHigh, inputs, 5*time.Minute)
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, "This is a test task", task.Description)
	assert.Equal(t, PriorityHigh, task.Priority)
	assert.Equal(t, TaskStatusPending, task.Status)
	assert.Equal(t, inputs, task.Inputs)
	assert.Equal(t, 5*time.Minute, task.Timeout)
}

func TestTaskManager_AssignTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// إنشاء مهمة
	task, err := tm.CreateTask(ctx, "Test Task", "Description", PriorityMedium, nil, 10*time.Minute)
	require.NoError(t, err)

	// تعيين المهمة لوكيل
	err = tm.AssignTask(ctx, task.ID, "agent_123")
	require.NoError(t, err)

	// التحقق من أن المهمة في القائمة الجارية
	runningTask, exists := tm.runningTasks[task.ID]
	assert.True(t, exists)
	assert.Equal(t, TaskStatusAssigned, runningTask.Status)
	assert.Equal(t, "agent_123", runningTask.AgentID)
}

func TestTaskManager_StartTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// إنشاء وتعيين مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityMedium, nil, 10*time.Minute)
	tm.AssignTask(ctx, task.ID, "agent_123")

	// بدء المهمة
	err := tm.StartTask(ctx, task.ID)
	require.NoError(t, err)

	// التحقق من الحالة
	runningTask, exists := tm.runningTasks[task.ID]
	assert.True(t, exists)
	assert.Equal(t, TaskStatusRunning, runningTask.Status)
	assert.NotNil(t, runningTask.StartedAt)
}

func TestTaskManager_CompleteTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// تسجيل وكيل
	tm.RegisterAgent("agent_123", []agent.AgentCapability{agent.CapabilityCodeGeneration})

	// إنشاء وتعيين وبدء مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityMedium, nil, 10*time.Minute)
	tm.AssignTask(ctx, task.ID, "agent_123")
	tm.StartTask(ctx, task.ID)

	// إكمال المهمة
	outputs := map[string]interface{}{
		"result": "success",
	}
	err := tm.CompleteTask(ctx, task.ID, outputs)
	require.NoError(t, err)

	// التحقق من أن المهمة في المهام المكتملة
	completedTask, exists := tm.completedTasks[task.ID]
	assert.True(t, exists)
	assert.Equal(t, TaskStatusCompleted, completedTask.Status)
	assert.Equal(t, outputs, completedTask.Outputs)
	assert.NotNil(t, completedTask.CompletedAt)

	// التحقق من أن المهمة ليست في القائمة الجارية
	_, exists = tm.runningTasks[task.ID]
	assert.False(t, exists)

	// التحقق من تحديث حالة الوكيل
	agentState, err := tm.GetAgentState("agent_123")
	require.NoError(t, err)
	assert.Equal(t, 1, agentState.TotalTasks)
	assert.Equal(t, "", agentState.CurrentTask)
}

func TestTaskManager_FailTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// تسجيل وكيل
	tm.RegisterAgent("agent_123", []agent.AgentCapability{agent.CapabilityCodeGeneration})

	// إنشاء وتعيين وبدء مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityMedium, nil, 10*time.Minute)
	tm.AssignTask(ctx, task.ID, "agent_123")
	tm.StartTask(ctx, task.ID)

	// فشل المهمة
	err := tm.FailTask(ctx, task.ID, "Task failed due to error")
	require.NoError(t, err)

	// التحقق من أن المهمة في المهام الفاشلة
	failedTask, exists := tm.failedTasks[task.ID]
	assert.True(t, exists)
	assert.Equal(t, TaskStatusFailed, failedTask.Status)
	assert.Equal(t, "Task failed due to error", failedTask.Metadata["error"])

	// التحقق من تحديث حالة الوكيل
	agentState, err := tm.GetAgentState("agent_123")
	require.NoError(t, err)
	assert.Equal(t, 1, agentState.TotalTasks)
	assert.Equal(t, 1, agentState.FailedTasks)
	assert.Equal(t, 0.0, agentState.SuccessRate)
}

func TestTaskManager_CancelTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// إنشاء مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityMedium, nil, 10*time.Minute)

	// إلغاء المهمة من قائمة الانتظار
	err := tm.CancelTask(ctx, task.ID)
	require.NoError(t, err)

	// التحقق من أن المهمة ليست في قائمة الانتظار
	nextTask := tm.GetNextTask()
	assert.Nil(t, nextTask)
}

func TestTaskManager_CancelRunningTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// تسجيل وكيل
	tm.RegisterAgent("agent_123", []agent.AgentCapability{agent.CapabilityCodeGeneration})

	// إنشاء وتعيين مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityMedium, nil, 10*time.Minute)
	tm.AssignTask(ctx, task.ID, "agent_123")

	// إلغاء المهمة الجارية
	err := tm.CancelTask(ctx, task.ID)
	require.NoError(t, err)

	// التحقق من أن المهمة ليست في القائمة الجارية (حالة cancelled)
	cancelledTask, err := tm.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusCancelled, cancelledTask.Status)

	// التحقق من تحديث حالة الوكيل
	agentState, err := tm.GetAgentState("agent_123")
	require.NoError(t, err)
	assert.Equal(t, "", agentState.CurrentTask)
}

func TestTaskManager_GetTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// إنشاء مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityHigh, nil, 10*time.Minute)

	// البحث عن المهمة
	foundTask, err := tm.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, foundTask.ID)
	assert.Equal(t, task.Title, foundTask.Title)
}

func TestTaskManager_GetTask_NotFound(t *testing.T) {
	tm := NewTaskManager("test_session")

	_, err := tm.GetTask("non_existent_task")
	assert.Error(t, err)
}

func TestTaskManager_GetNextTask(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// إنشاء مهام بأولويات مختلفة
	tm.CreateTask(ctx, "Low Priority Task", "Description", PriorityLow, nil, 10*time.Minute)
	tm.CreateTask(ctx, "High Priority Task", "Description", PriorityHigh, nil, 10*time.Minute)
	tm.CreateTask(ctx, "Medium Priority Task", "Description", PriorityMedium, nil, 10*time.Minute)

	// الحصول على المهمة التالية (يجب أن تكون ذات الأولوية العالية)
	nextTask := tm.GetNextTask()
	assert.NotNil(t, nextTask)
	assert.Equal(t, "High Priority Task", nextTask.Title)
}

func TestTaskManager_RegisterAgent(t *testing.T) {
	tm := NewTaskManager("test_session")

	capabilities := []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
	}

	tm.RegisterAgent("agent_123", capabilities)

	agentState, err := tm.GetAgentState("agent_123")
	require.NoError(t, err)
	assert.Equal(t, "agent_123", agentState.AgentID)
	assert.Equal(t, "idle", agentState.Status)
	assert.Equal(t, 0, agentState.Load)
	assert.Equal(t, 1.0, agentState.SuccessRate)
	assert.Equal(t, capabilities, agentState.Capabilities)
}

func TestTaskManager_UnregisterAgent(t *testing.T) {
	tm := NewTaskManager("test_session")

	tm.RegisterAgent("agent_123", []agent.AgentCapability{agent.CapabilityCodeGeneration})
	tm.UnregisterAgent("agent_123")

	_, err := tm.GetAgentState("agent_123")
	assert.Error(t, err)
}

func TestTaskManager_UpdateAgentLoad(t *testing.T) {
	tm := NewTaskManager("test_session")

	tm.RegisterAgent("agent_123", []agent.AgentCapability{agent.CapabilityCodeGeneration})

	// تحديث الحمل إلى 90 (busy)
	tm.UpdateAgentLoad("agent_123", 90)
	agentState, _ := tm.GetAgentState("agent_123")
	assert.Equal(t, 90, agentState.Load)
	assert.Equal(t, "busy", agentState.Status)

	// تحديث الحمل إلى 30 (idle)
	tm.UpdateAgentLoad("agent_123", 30)
	agentState, _ = tm.GetAgentState("agent_123")
	assert.Equal(t, 30, agentState.Load)
	assert.Equal(t, "idle", agentState.Status)
}

func TestTaskManager_GetStats(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// إنشاء بعض المهام
	tm.CreateTask(ctx, "Task 1", "Description", PriorityHigh, nil, 10*time.Minute)
	tm.CreateTask(ctx, "Task 2", "Description", PriorityMedium, nil, 10*time.Minute)

	// تسجيل وكلاء
	tm.RegisterAgent("agent_1", []agent.AgentCapability{agent.CapabilityCodeGeneration})
	tm.RegisterAgent("agent_2", []agent.AgentCapability{agent.CapabilityCodeReview})

	stats := tm.GetStats()
	assert.Equal(t, 2, stats["pending_count"])
	assert.Equal(t, 0, stats["running_count"])
	assert.Equal(t, 0, stats["completed_count"])
	assert.Equal(t, 0, stats["failed_count"])
	assert.Equal(t, 2, stats["agent_count"])
}

func TestTaskManager_SetLogger(t *testing.T) {
	tm := NewTaskManager("test_session")

	logger := zap.NewNop()
	tm.SetLogger(logger)

	assert.Equal(t, logger, tm.logger)
}

func TestTaskManager_SetEventBus(t *testing.T) {
	tm := NewTaskManager("test_session")

	eb := eventbus.NewEventBus()
	tm.SetEventBus(eb)

	assert.Equal(t, eb, tm.eventBus)
}

func TestTaskManager_EventBusIntegration(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	eb := eventbus.NewEventBus()
	tm.SetEventBus(eb)

	// اشتراك في الأحداث
	eventsReceived := make([]string, 0)
	eb.Subscribe("task.created", func(e eventbus.Event) {
		eventsReceived = append(eventsReceived, e.Type)
	})
	eb.Subscribe("task.assigned", func(e eventbus.Event) {
		eventsReceived = append(eventsReceived, e.Type)
	})
	eb.Subscribe("task.started", func(e eventbus.Event) {
		eventsReceived = append(eventsReceived, e.Type)
	})
	eb.Subscribe("task.completed", func(e eventbus.Event) {
		eventsReceived = append(eventsReceived, e.Type)
	})

	// إنشاء مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityHigh, nil, 10*time.Minute)
	time.Sleep(10 * time.Millisecond) // انتظار معالجة الحدث

	// تعيين وبدء وإكمال المهمة
	tm.AssignTask(ctx, task.ID, "agent_123")
	time.Sleep(10 * time.Millisecond)

	tm.StartTask(ctx, task.ID)
	time.Sleep(10 * time.Millisecond)

	tm.CompleteTask(ctx, task.ID, map[string]interface{}{"result": "success"})
	time.Sleep(10 * time.Millisecond)

	// التحقق من استلام الأحداث
	assert.Contains(t, eventsReceived, "task.created")
	assert.Contains(t, eventsReceived, "task.assigned")
	assert.Contains(t, eventsReceived, "task.started")
	assert.Contains(t, eventsReceived, "task.completed")
}

func TestTaskManager_SaveAndLoad(t *testing.T) {
	tm := NewTaskManager("test_session")
	ctx := context.Background()

	// إنشاء مهمة
	task, _ := tm.CreateTask(ctx, "Test Task", "Description", PriorityHigh, map[string]interface{}{"key": "value"}, 10*time.Minute)

	// حفظ الحالة
	data, err := tm.Save()
	require.NoError(t, err)
	assert.NotNil(t, data)

	// إنشاء مدير مهام جديد وتحميل الحالة
	tm2 := NewTaskManager("test_session_2")
	err = tm2.Load(data)
	require.NoError(t, err)

	// التحقق من أن المهمة محملة
	loadedTask, err := tm2.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, loadedTask.ID)
	assert.Equal(t, task.Title, loadedTask.Title)
	assert.Equal(t, task.Priority, loadedTask.Priority)
}

func TestTaskPriority_Values(t *testing.T) {
	assert.Equal(t, TaskPriority(1), PriorityLow)
	assert.Equal(t, TaskPriority(2), PriorityMedium)
	assert.Equal(t, TaskPriority(3), PriorityHigh)
	assert.Equal(t, TaskPriority(4), PriorityUrgent)
}

func TestTaskStatus_Values(t *testing.T) {
	assert.Equal(t, TaskStatus("pending"), TaskStatusPending)
	assert.Equal(t, TaskStatus("assigned"), TaskStatusAssigned)
	assert.Equal(t, TaskStatus("running"), TaskStatusRunning)
	assert.Equal(t, TaskStatus("completed"), TaskStatusCompleted)
	assert.Equal(t, TaskStatus("failed"), TaskStatusFailed)
	assert.Equal(t, TaskStatus("cancelled"), TaskStatusCancelled)
}

func TestTaskHeap_Ordering(t *testing.T) {
	h := &TaskHeap{}
	heap.Init(h)

	// إضافة مهام بأولويات مختلفة
	heap.Push(h, &ManagedTask{ID: "task1", Priority: PriorityLow})
	heap.Push(h, &ManagedTask{ID: "task2", Priority: PriorityHigh})
	heap.Push(h, &ManagedTask{ID: "task3", Priority: PriorityMedium})
	heap.Push(h, &ManagedTask{ID: "task4", Priority: PriorityUrgent})

	// استخراج المهام بالترتيب
	task1 := heap.Pop(h).(*ManagedTask)
	task2 := heap.Pop(h).(*ManagedTask)
	task3 := heap.Pop(h).(*ManagedTask)
	task4 := heap.Pop(h).(*ManagedTask)

	// التحقق من الترتيب (من الأعلى إلى الأدنى)
	assert.Equal(t, PriorityUrgent, task1.Priority)
	assert.Equal(t, PriorityHigh, task2.Priority)
	assert.Equal(t, PriorityMedium, task3.Priority)
	assert.Equal(t, PriorityLow, task4.Priority)
}
