package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// PrometheusMetrics مقاييس Prometheus لمراقبة الأداء
type PrometheusMetrics struct {
	logger *zap.Logger

	// Task metrics
	taskDuration *prometheus.HistogramVec
	taskSuccess  *prometheus.CounterVec
	taskFailure  *prometheus.CounterVec

	// Agent metrics
	agentActive *prometheus.GaugeVec
	agentTotal  *prometheus.CounterVec

	// Session metrics
	sessionActive *prometheus.GaugeVec
	sessionTotal  *prometheus.CounterVec

	// LLM metrics
	llmDuration    *prometheus.HistogramVec
	llmTokens      *prometheus.CounterVec
	llmCost        *prometheus.CounterVec
	llmRateLimit   *prometheus.CounterVec

	// Memory metrics
	memoryUsage *prometheus.GaugeVec
	memoryCache *prometheus.GaugeVec

	// Concurrency metrics
	concurrentTasks *prometheus.GaugeVec
	concurrentAgents *prometheus.GaugeVec

	// Error metrics
	errorCount *prometheus.CounterVec
	errorRate  *prometheus.GaugeVec

	mu sync.RWMutex
}

// NewPrometheusMetrics ينشئ مقاييس Prometheus جديدة
func NewPrometheusMetrics(logger *zap.Logger) *PrometheusMetrics {
	return &PrometheusMetrics{
		logger: logger,

		// Task metrics
		taskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "musketeers_task_duration_seconds",
				Help:    "Duration of task execution in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"task_type", "agent_id"},
		),
		taskSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_task_success_total",
				Help: "Total number of successful tasks",
			},
			[]string{"task_type", "agent_id"},
		),
		taskFailure: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_task_failure_total",
				Help: "Total number of failed tasks",
			},
			[]string{"task_type", "agent_id", "error_type"},
		),

		// Agent metrics
		agentActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "musketeers_agent_active",
				Help: "Number of active agents",
			},
			[]string{"agent_type", "session_id"},
		),
		agentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_agent_total",
				Help: "Total number of agents created",
			},
			[]string{"agent_type", "session_id"},
		),

		// Session metrics
		sessionActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "musketeers_session_active",
				Help: "Number of active sessions",
			},
			[]string{"session_type"},
		),
		sessionTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_session_total",
				Help: "Total number of sessions created",
			},
			[]string{"session_type"},
		),

		// LLM metrics
		llmDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "musketeers_llm_duration_seconds",
				Help:    "Duration of LLM calls in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider", "model", "operation"},
		),
		llmTokens: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_llm_tokens_total",
				Help: "Total number of tokens used",
			},
			[]string{"provider", "model", "token_type"},
		),
		llmCost: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_llm_cost_total",
				Help: "Total cost of LLM calls",
			},
			[]string{"provider", "model"},
		),
		llmRateLimit: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_llm_rate_limit_total",
				Help: "Total number of rate limit errors",
			},
			[]string{"provider", "model"},
		),

		// Memory metrics
		memoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "musketeers_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"memory_type"},
		),
		memoryCache: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "musketeers_memory_cache_size",
				Help: "Cache size in bytes",
			},
			[]string{"cache_type"},
		),

		// Concurrency metrics
		concurrentTasks: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "musketeers_concurrent_tasks",
				Help: "Number of concurrent tasks",
			},
			[]string{"session_id"},
		),
		concurrentAgents: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "musketeers_concurrent_agents",
				Help: "Number of concurrent agents",
			},
			[]string{"session_id"},
		),

		// Error metrics
		errorCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "musketeers_error_total",
				Help: "Total number of errors",
			},
			[]string{"error_type", "component"},
		),
		errorRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "musketeers_error_rate",
				Help: "Error rate per component",
			},
			[]string{"component"},
		),
	}
}

// RecordTaskDuration يسجل مدة تنفيذ المهمة
func (m *PrometheusMetrics) RecordTaskDuration(taskType, agentID string, duration time.Duration) {
	m.taskDuration.WithLabelValues(taskType, agentID).Observe(duration.Seconds())
}

// RecordTaskSuccess يسجل نجاح المهمة
func (m *PrometheusMetrics) RecordTaskSuccess(taskType, agentID string) {
	m.taskSuccess.WithLabelValues(taskType, agentID).Inc()
}

// RecordTaskFailure يسجل فشل المهمة
func (m *PrometheusMetrics) RecordTaskFailure(taskType, agentID, errorType string) {
	m.taskFailure.WithLabelValues(taskType, agentID, errorType).Inc()
}

// SetAgentActive يضبط عدد الوكلاء النشطين
func (m *PrometheusMetrics) SetAgentActive(agentType, sessionID string, count float64) {
	m.agentActive.WithLabelValues(agentType, sessionID).Set(count)
}

// IncrementAgentTotal يزيد عدد الوكلاء الإجمالي
func (m *PrometheusMetrics) IncrementAgentTotal(agentType, sessionID string) {
	m.agentTotal.WithLabelValues(agentType, sessionID).Inc()
}

// SetSessionActive يضبط عدد الجلسات النشطة
func (m *PrometheusMetrics) SetSessionActive(sessionType string, count float64) {
	m.sessionActive.WithLabelValues(sessionType).Set(count)
}

// IncrementSessionTotal يزيد عدد الجلسات الإجمالي
func (m *PrometheusMetrics) IncrementSessionTotal(sessionType string) {
	m.sessionTotal.WithLabelValues(sessionType).Inc()
}

// RecordLLMDuration يسجل مدة استدعاء LLM
func (m *PrometheusMetrics) RecordLLMDuration(provider, model, operation string, duration time.Duration) {
	m.llmDuration.WithLabelValues(provider, model, operation).Observe(duration.Seconds())
}

// RecordLLMTokens يسجل عدد الرموز المستخدمة
func (m *PrometheusMetrics) RecordLLMTokens(provider, model, tokenType string, count float64) {
	m.llmTokens.WithLabelValues(provider, model, tokenType).Add(count)
}

// RecordLLMCost يسجل تكلفة استدعاء LLM
func (m *PrometheusMetrics) RecordLLMCost(provider, model string, cost float64) {
	m.llmCost.WithLabelValues(provider, model).Add(cost)
}

// RecordLLMRateLimit يسجل خطأ حد السرعة
func (m *PrometheusMetrics) RecordLLMRateLimit(provider, model string) {
	m.llmRateLimit.WithLabelValues(provider, model).Inc()
}

// SetMemoryUsage يضبط استخدام الذاكرة
func (m *PrometheusMetrics) SetMemoryUsage(memoryType string, bytes float64) {
	m.memoryUsage.WithLabelValues(memoryType).Set(bytes)
}

// SetMemoryCache يضبط حجم الذاكرة المؤقتة
func (m *PrometheusMetrics) SetMemoryCache(cacheType string, bytes float64) {
	m.memoryCache.WithLabelValues(cacheType).Set(bytes)
}

// SetConcurrentTasks يضبط عدد المهام المتزامنة
func (m *PrometheusMetrics) SetConcurrentTasks(sessionID string, count float64) {
	m.concurrentTasks.WithLabelValues(sessionID).Set(count)
}

// SetConcurrentAgents يضبط عدد الوكلاء المتزامنين
func (m *PrometheusMetrics) SetConcurrentAgents(sessionID string, count float64) {
	m.concurrentAgents.WithLabelValues(sessionID).Set(count)
}

// RecordError يسجل خطأ
func (m *PrometheusMetrics) RecordError(errorType, component string) {
	m.errorCount.WithLabelValues(errorType, component).Inc()
}

// SetErrorRate يضبط معدل الأخطاء
func (m *PrometheusMetrics) SetErrorRate(component string, rate float64) {
	m.errorRate.WithLabelValues(component).Set(rate)
}
