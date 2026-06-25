package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LocalCache ذاكرة مؤقتة محلية
type LocalCache struct {
	data   map[string]*cacheEntry
	mu     sync.RWMutex
	logger *zap.Logger
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// NewLocalCache ينشئ ذاكرة مؤقتة محلية جديدة
func NewLocalCache(logger *zap.Logger) *LocalCache {
	return &LocalCache{
		data:   make(map[string]*cacheEntry),
		logger: logger,
	}
}

// Set يضبط قيمة في الذاكرة المؤقتة
func (c *LocalCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &cacheEntry{
		value:      value,
		expiration: time.Now().Add(expiration),
	}

	return nil
}

// Get يحصل على قيمة من الذاكرة المؤقتة
func (c *LocalCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return fmt.Errorf("key not found")
	}

	if time.Now().After(entry.expiration) {
		return fmt.Errorf("key expired")
	}

	data, err := json.Marshal(entry.value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete يحذف قيمة من الذاكرة المؤقتة
func (c *LocalCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
	return nil
}

// Exists يتحقق من وجود مفتاح
func (c *LocalCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(entry.expiration) {
		return false, nil
	}

	return true, nil
}

// SetLLMResponse يخزن استجابة LLM
func (c *LocalCache) SetLLMResponse(ctx context.Context, prompt, model string, response interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("llm:%s:%s", model, hashPrompt(prompt))
	return c.Set(ctx, key, response, expiration)
}

// GetLLMResponse يحصل على استجابة LLM مخزنة
func (c *LocalCache) GetLLMResponse(ctx context.Context, prompt, model string, dest interface{}) error {
	key := fmt.Sprintf("llm:%s:%s", model, hashPrompt(prompt))
	return c.Get(ctx, key, dest)
}

// SetEmbedding يخزن embedding
func (c *LocalCache) SetEmbedding(ctx context.Context, text string, embedding []float64, expiration time.Duration) error {
	key := fmt.Sprintf("embedding:%s", hashPrompt(text))
	return c.Set(ctx, key, embedding, expiration)
}

// GetEmbedding يحصل على embedding مخزن
func (c *LocalCache) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	key := fmt.Sprintf("embedding:%s", hashPrompt(text))
	var embedding []float64
	err := c.Get(ctx, key, &embedding)
	return embedding, err
}

// hashPrompt يولد hash من prompt
func hashPrompt(prompt string) string {
	// Simple hash function - in production use proper hashing
	hash := 0
	for i, c := range prompt {
		hash += int(c) * (i + 1)
	}
	return fmt.Sprintf("%d", hash)
}
