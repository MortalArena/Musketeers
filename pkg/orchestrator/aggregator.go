package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/verification"
	"go.uber.org/zap"
)

// AggregationStrategy استراتيجية التجميع
type AggregationStrategy string

const (
	StrategyFirstValid AggregationStrategy = "first_valid" // أول نتيجة صحيحة
	StrategyMajority   AggregationStrategy = "majority"    // الأغلبية
	StrategyWeighted   AggregationStrategy = "weighted"    // مرجح
	StrategyConsensus  AggregationStrategy = "consensus"   // إجماع
)

// AggregationResult نتيجة التجميع
type AggregationResult struct {
	Results      []*agent.TaskExecutionResult       `json:"results"`
	FinalResult  *agent.TaskExecutionResult         `json:"final_result"`
	Strategy     AggregationStrategy                `json:"strategy"`
	Confidence   float64                            `json:"confidence"`
	Verification []*verification.VerificationResult `json:"verification"`
	Metadata     map[string]interface{}             `json:"metadata"`
}

// ResultAggregator مجمع النتائج
type ResultAggregator struct {
	verifier *verification.MultiStageVerifier
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewResultAggregator ينشئ مجمع نتائج جديد
func NewResultAggregator(verifier *verification.MultiStageVerifier) *ResultAggregator {
	return &ResultAggregator{
		verifier: verifier,
		logger:   zap.NewNop(),
	}
}

// SetLogger يضبط logger
func (ra *ResultAggregator) SetLogger(logger *zap.Logger) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.logger = logger
}

// AggregateResults يجمع نتائج متعددة
func (ra *ResultAggregator) AggregateResults(ctx context.Context, results []*agent.TaskExecutionResult, strategy AggregationStrategy) (*AggregationResult, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	if len(results) == 0 {
		return nil, fmt.Errorf("no results to aggregate")
	}

	var finalResult *agent.TaskExecutionResult
	var confidence float64

	switch strategy {
	case StrategyFirstValid:
		finalResult, confidence = ra.aggregateFirstValid(results)
	case StrategyMajority:
		finalResult, confidence = ra.aggregateMajority(results)
	case StrategyWeighted:
		finalResult, confidence = ra.aggregateWeighted(results)
	case StrategyConsensus:
		finalResult, confidence = ra.aggregateConsensus(results)
	default:
		return nil, fmt.Errorf("unknown aggregation strategy: %s", strategy)
	}

	// التحقق من النتيجة النهائية
	verificationResults := ra.verifyResult(ctx, finalResult)

	ra.logger.Info("Results aggregated",
		zap.String("strategy", string(strategy)),
		zap.Float64("confidence", confidence),
		zap.Int("result_count", len(results)),
	)

	return &AggregationResult{
		Results:      results,
		FinalResult:  finalResult,
		Strategy:     strategy,
		Confidence:   confidence,
		Verification: verificationResults,
		Metadata:     map[string]interface{}{},
	}, nil
}

// aggregateFirstValid يجمع باستخدام أول نتيجة صحيحة
func (ra *ResultAggregator) aggregateFirstValid(results []*agent.TaskExecutionResult) (*agent.TaskExecutionResult, float64) {
	for _, result := range results {
		if result.Success {
			return result, 1.0
		}
	}
	// إذا لم تكن هناك نتيجة صحيحة، نرجع النتيجة الأخيرة
	return results[len(results)-1], 0.5
}

// aggregateMajority يجمع باستخدام الأغلبية
func (ra *ResultAggregator) aggregateMajority(results []*agent.TaskExecutionResult) (*agent.TaskExecutionResult, float64) {
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	confidence := float64(successCount) / float64(len(results))

	// نرجع النتيجة الأكثر شيوعاً
	if successCount > len(results)/2 {
		// نرجع أول نتيجة ناجحة
		for _, result := range results {
			if result.Success {
				return result, confidence
			}
		}
	}

	// نرجع النتيجة الأخيرة
	return results[len(results)-1], confidence
}

// aggregateWeighted يجمع باستخدام الترجيح
func (ra *ResultAggregator) aggregateWeighted(results []*agent.TaskExecutionResult) (*agent.TaskExecutionResult, float64) {
	totalWeight := 0.0
	weightedOutput := ""

	for i, result := range results {
		weight := float64(len(results)-i) / float64(len(results)) // وزن أعلى للنتائج الأولى
		totalWeight += weight

		if result.Success {
			weightedOutput += result.Output
		}
	}

	confidence := 0.5 // افتراضي

	// نرجع نتيجة مدمجة
	return &agent.TaskExecutionResult{
		Success:  true,
		Output:   weightedOutput,
		Duration: results[0].Duration,
		Metrics:  results[0].Metrics,
	}, confidence
}

// aggregateConsensus يجمع باستخدام الإجماع
func (ra *ResultAggregator) aggregateConsensus(results []*agent.TaskExecutionResult) (*agent.TaskExecutionResult, float64) {
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	confidence := float64(successCount) / float64(len(results))

	// للإجماع، نحتاج إلى جميع النتائج متطابقة
	if successCount == len(results) {
		return results[0], 1.0
	}

	// إذا لم يكن هناك إجماع، نرجع النتيجة الأكثر شيوعاً مع الثقة المحسوبة
	result, _ := ra.aggregateMajority(results)
	return result, confidence
}

// verifyResult يتحقق من نتيجة
func (ra *ResultAggregator) verifyResult(ctx context.Context, result *agent.TaskExecutionResult) []*verification.VerificationResult {
	if ra.verifier == nil {
		return []*verification.VerificationResult{}
	}

	request := &verification.VerificationRequest{
		TaskID:  "aggregated",
		AgentID: "aggregator",
		Output:  result.Output,
		Stages:  []verification.VerificationStage{},
	}

	results, err := ra.verifier.Verify(ctx, request)
	if err != nil {
		ra.logger.Error("Verification failed",
			zap.Error(err),
		)
		return []*verification.VerificationResult{}
	}

	return results
}

// GetBestResult يحصل على أفضل نتيجة من مجموعة
func (ra *ResultAggregator) GetBestResult(results []*agent.TaskExecutionResult) *agent.TaskExecutionResult {
	if len(results) == 0 {
		return nil
	}

	bestResult := results[0]
	bestScore := 0.0

	for _, result := range results {
		score := ra.calculateResultScore(result)
		if score > bestScore {
			bestScore = score
			bestResult = result
		}
	}

	return bestResult
}

// calculateResultScore يحسب نتيجة نتيجة
func (ra *ResultAggregator) calculateResultScore(result *agent.TaskExecutionResult) float64 {
	score := 0.0

	if result.Success {
		score += 0.5
	}

	// نتيجة أعلى لوقت تنفيذ أقل
	if result.Duration.Seconds() > 0 {
		score += 1.0 / result.Duration.Seconds() * 0.3
	}

	return score
}
