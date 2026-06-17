package unified

import "context"

// AgentExecutor واجهة لتنفيذ المهام
// هذه الواجهة تسمح لـ SessionManager بتنفيذ المهام دون الاعتماد المباشر على UnifiedAgent
type AgentExecutor interface {
	// ExecuteTask ينفذ مهمة
	ExecuteTask(ctx context.Context, task string) (*UnifiedTaskResult, error)
}
