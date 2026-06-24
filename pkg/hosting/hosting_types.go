package hosting

import (
	"sync"
	"time"
)

// Site يمثل موقع
type Site struct {
	mu         sync.RWMutex
	Name        string            `json:"name"`
	Files       map[string][]byte `json:"files"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Visitors    int64             `json:"visitors"`
	Bandwidth   int64             `json:"bandwidth"`
}

// SiteStats إحصائيات الموقع
type SiteStats struct {
	TotalVisitors  int64     `json:"total_visitors"`
	TotalBandwidth int64     `json:"total_bandwidth"`
	LastAccessed   time.Time `json:"last_accessed"`
}
