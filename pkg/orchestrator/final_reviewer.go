package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/verification"
	"go.uber.org/zap"
)

// ReviewCriteria معايير المراجعة
type ReviewCriteria struct {
	MinConfidence    float64  `json:"min_confidence"`    // الحد الأدنى للثقة
	RequireVerification bool   `json:"require_verification"` // هل يتطلب التحقق
	MaxDuration      int64    `json:"max_duration"`       // الحد الأقصى للمدة بالميلي ثانية
	RequiredStages   []verification.VerificationStage `json:"required_stages"` // المراحل المطلوبة
}

// ReviewDecision قرار المراجعة
type ReviewDecision struct {
	Approved     bool                         `json:"approved"`
	Reason       string                       `json:"reason"`
	Score        float64                      `json:"score"`
	Verification []*verification.VerificationResult `json:"verification"`
	Metadata     map[string]interface{}      `json:"metadata"`
}

// FinalReviewer المراجع النهائي
type FinalReviewer struct {
	verifier *verification.MultiStageVerifier
	criteria *ReviewCriteria
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewFinalReviewer ينشئ مراجع نهائي جديد
func NewFinalReviewer(verifier *verification.MultiStageVerifier) *FinalReviewer {
	return &FinalReviewer{
		verifier: verifier,
		criteria: &ReviewCriteria{
			MinConfidence:     0.7,
			RequireVerification: true,
			MaxDuration:      300000, // 5 دقائق
			RequiredStages:   []verification.VerificationStage{},
		},
		logger: zap.NewNop(),
	}
}

// SetLogger يضبط logger
func (fr *FinalReviewer) SetLogger(logger *zap.Logger) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.logger = logger
}

// SetCriteria يضبط معايير المراجعة
func (fr *FinalReviewer) SetCriteria(criteria *ReviewCriteria) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.criteria = criteria
}

// GetCriteria يحصل على معايير المراجعة
func (fr *FinalReviewer) GetCriteria() *ReviewCriteria {
	fr.mu.RLock()
	defer fr.mu.RUnlock()
	return fr.criteria
}

// Review يراجع نتيجة مهمة
func (fr *FinalReviewer) Review(ctx context.Context, taskID string, agentID string, output string, duration int64) (*ReviewDecision, error) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	decision := &ReviewDecision{
		Approved: false,
		Reason:   "",
		Score:    0.0,
		Verification: []*verification.VerificationResult{},
		Metadata: map[string]interface{}{},
	}

	// التحقق من المدة
	if duration > fr.criteria.MaxDuration {
		decision.Reason = fmt.Sprintf("Task duration %dms exceeds maximum %dms", duration, fr.criteria.MaxDuration)
		decision.Score = 0.3
		fr.logger.Warn("Task duration exceeded",
			zap.String("task_id", taskID),
			zap.Int64("duration", duration),
			zap.Int64("max_duration", fr.criteria.MaxDuration),
		)
		return decision, nil
	}

	// التحقق من الناتج
	if len(output) == 0 {
		decision.Reason = "Task output is empty"
		decision.Score = 0.0
		fr.logger.Warn("Task output is empty",
			zap.String("task_id", taskID),
		)
		return decision, nil
	}

	// التحقق متعدد المراحل إذا كان مطلوباً
	if fr.criteria.RequireVerification && fr.verifier != nil {
		request := &verification.VerificationRequest{
			TaskID:  taskID,
			AgentID: agentID,
			Output:  output,
			Stages:  fr.criteria.RequiredStages,
		}

		results, err := fr.verifier.Verify(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("verification failed: %w", err)
		}

		decision.Verification = results

		// حساب النتيجة الإجمالية
		overallScore := fr.verifier.GetOverallScore(results)
		decision.Score = overallScore

		// التحقق من الحد الأدنى للثقة
		if overallScore < fr.criteria.MinConfidence {
			decision.Reason = fmt.Sprintf("Verification score %.2f below minimum %.2f", overallScore, fr.criteria.MinConfidence)
			decision.Approved = false
			fr.logger.Warn("Verification score below minimum",
				zap.String("task_id", taskID),
				zap.Float64("score", overallScore),
				zap.Float64("min_confidence", fr.criteria.MinConfidence),
			)
			return decision, nil
		}

		// التحقق من المراحل المطلوبة
		failedStages := fr.verifier.GetFailedStages(results)
		if len(failedStages) > 0 {
			decision.Reason = fmt.Sprintf("%d verification stages failed", len(failedStages))
			decision.Approved = false
			fr.logger.Warn("Verification stages failed",
				zap.String("task_id", taskID),
				zap.Int("failed_stages", len(failedStages)),
			)
			return decision, nil
		}
	}

	// كل شيء على ما يرام
	decision.Approved = true
	decision.Reason = "Task review passed"
	if decision.Score == 0 {
		decision.Score = 1.0
	}

	fr.logger.Info("Task review passed",
		zap.String("task_id", taskID),
		zap.String("agent_id", agentID),
		zap.Float64("score", decision.Score),
	)

	return decision, nil
}

// ReviewAggregated يراجع نتيجة مجمعة
func (fr *FinalReviewer) ReviewAggregated(ctx context.Context, aggregation *AggregationResult) (*ReviewDecision, error) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	decision := &ReviewDecision{
		Approved: false,
		Reason:   "",
		Score:    0.0,
		Verification: []*verification.VerificationResult{},
		Metadata: map[string]interface{}{},
	}

	// التحقق من الثقة
	if aggregation.Confidence < fr.criteria.MinConfidence {
		decision.Reason = fmt.Sprintf("Aggregation confidence %.2f below minimum %.2f", aggregation.Confidence, fr.criteria.MinConfidence)
		decision.Score = aggregation.Confidence
		fr.logger.Warn("Aggregation confidence below minimum",
			zap.Float64("confidence", aggregation.Confidence),
			zap.Float64("min_confidence", fr.criteria.MinConfidence),
		)
		return decision, nil
	}

	// استخدام نتائج التحقق من التجميع
	if len(aggregation.Verification) > 0 {
		decision.Verification = aggregation.Verification
		overallScore := fr.verifier.GetOverallScore(aggregation.Verification)
		decision.Score = overallScore

		if overallScore < fr.criteria.MinConfidence {
			decision.Reason = fmt.Sprintf("Verification score %.2f below minimum %.2f", overallScore, fr.criteria.MinConfidence)
			decision.Approved = false
			return decision, nil
		}
	}

	// كل شيء على ما يرام
	decision.Approved = true
	decision.Reason = "Aggregated result review passed"
	if decision.Score == 0 {
		decision.Score = aggregation.Confidence
	}

	fr.logger.Info("Aggregated result review passed",
		zap.String("strategy", string(aggregation.Strategy)),
		zap.Float64("confidence", aggregation.Confidence),
		zap.Float64("score", decision.Score),
	)

	return decision, nil
}

// GetRequiredStages يحصل على المراحل المطلوبة للمراجعة
func (fr *FinalReviewer) GetRequiredStages() []verification.VerificationStage {
	fr.mu.RLock()
	defer fr.mu.RUnlock()
	return fr.criteria.RequiredStages
}

// SetRequiredStages يضبط المراحل المطلوبة للمراجعة
func (fr *FinalReviewer) SetRequiredStages(stages []verification.VerificationStage) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.criteria.RequiredStages = stages
}
