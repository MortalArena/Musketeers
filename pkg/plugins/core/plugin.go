package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Plugin واجهة الإضافة
type Plugin interface {
	// معلومات الإضافة
	Name() string
	Version() string
	Description() string
	Author() string

	// دورة حياة الإضافة
	Initialize(ctx context.Context, config map[string]interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Shutdown(ctx context.Context) error

	// حالة الإضافة
	Status() PluginStatus
	Health() PluginHealth

	// التكامل
	GetDependencies() []string
	GetCapabilities() []string
}

// PluginStatus حالة الإضافة
type PluginStatus string

const (
	PluginStatusUninitialized PluginStatus = "uninitialized"
	PluginStatusInitializing   PluginStatus = "initializing"
	PluginStatusReady         PluginStatus = "ready"
	PluginStatusRunning       PluginStatus = "running"
	PluginStatusPaused        PluginStatus = "paused"
	PluginStatusStopping      PluginStatus = "stopping"
	PluginStatusStopped       PluginStatus = "stopped"
	PluginStatusError         PluginStatus = "error"
)

// PluginHealth صحة الإضافة
type PluginHealth struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	LastCheck   time.Time `json:"last_check"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// PluginMetadata بيانات وصفية للإضافة
type PluginMetadata struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	Dependencies []string               `json:"dependencies"`
	Capabilities []string               `json:"capabilities"`
	Config       map[string]interface{} `json:"config"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// PluginManager مدير الإضافات
type PluginManager struct {
	plugins      map[string]Plugin
	metadata     map[string]*PluginMetadata
	health       map[string]*PluginHealth
	logger       *zap.Logger
	mu           sync.RWMutex
	eventBus     EventBus
}

// EventBus واجهة ناقل الأحداث
type EventBus interface {
	Publish(event string, data interface{}) error
	Subscribe(event string, handler func(data interface{})) error
}

// NewPluginManager ينشئ مدير إضافات جديد
func NewPluginManager(logger *zap.Logger, eventBus EventBus) *PluginManager {
	return &PluginManager{
		plugins:  make(map[string]Plugin),
		metadata: make(map[string]*PluginMetadata),
		health:   make(map[string]*PluginHealth),
		logger:   logger,
		eventBus: eventBus,
	}
}

// Register يسجل إضافة جديدة
func (pm *PluginManager) Register(plugin Plugin, config map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := plugin.Name()

	// التحقق من عدم وجود الإضافة بالفعل
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin already registered: %s", name)
	}

	// التحقق من التبعيات
	deps := plugin.GetDependencies()
	for _, dep := range deps {
		if _, exists := pm.plugins[dep]; !exists {
			return fmt.Errorf("dependency not found: %s", dep)
		}
	}

	// تسجيل الإضافة
	pm.plugins[name] = plugin

	// إنشاء البيانات الوصفية
	pm.metadata[name] = &PluginMetadata{
		Name:         plugin.Name(),
		Version:      plugin.Version(),
		Description:  plugin.Description(),
		Author:       plugin.Author(),
		Dependencies: plugin.GetDependencies(),
		Capabilities: plugin.GetCapabilities(),
		Config:       config,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// إنشاء حالة الصحة الأولية
	pm.health[name] = &PluginHealth{
		Status:    "uninitialized",
		Message:   "Plugin registered but not initialized",
		LastCheck: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	pm.logger.Info("تم تسجيل إضافة جديدة",
		zap.String("plugin_name", name),
		zap.String("version", plugin.Version()),
		zap.String("author", plugin.Author()))

	// نشر حدث تسجيل الإضافة
	if pm.eventBus != nil {
		pm.eventBus.Publish("plugin.registered", map[string]interface{}{
			"name":    name,
			"version": plugin.Version(),
		})
	}

	return nil
}

// Unregister يلغي تسجيل إضافة
func (pm *PluginManager) Unregister(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// إيقاف الإضافة إذا كانت تعمل
	if plugin.Status() == PluginStatusRunning {
		if err := plugin.Stop(context.Background()); err != nil {
			pm.logger.Error("فشل إيقاف الإضافة",
				zap.String("plugin_name", name),
				zap.Error(err))
		}
	}

	// إغلاق الإضافة
	if err := plugin.Shutdown(context.Background()); err != nil {
		pm.logger.Error("فشل إغلاق الإضافة",
			zap.String("plugin_name", name),
			zap.Error(err))
	}

	// حذف الإضافة
	delete(pm.plugins, name)
	delete(pm.metadata, name)
	delete(pm.health, name)

	pm.logger.Info("تم إلغاء تسجيل الإضافة",
		zap.String("plugin_name", name))

	// نشر حدث إلغاء التسجيل
	if pm.eventBus != nil {
		pm.eventBus.Publish("plugin.unregistered", map[string]interface{}{
			"name": name,
		})
	}

	return nil
}

// Initialize يهيئ إضافة
func (pm *PluginManager) Initialize(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	metadata := pm.metadata[name]

	// تهيئة الإضافة
	if err := plugin.Initialize(context.Background(), metadata.Config); err != nil {
		pm.health[name] = &PluginHealth{
			Status:    "error",
			Message:   fmt.Sprintf("Initialization failed: %v", err),
			LastCheck: time.Now(),
			Metrics:   make(map[string]interface{}),
		}
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// تحديث حالة الصحة
	pm.health[name] = &PluginHealth{
		Status:    "ready",
		Message:   "Plugin initialized successfully",
		LastCheck: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	pm.logger.Info("تم تهيئة الإضافة",
		zap.String("plugin_name", name))

	// نشر حدث التهيئة
	if pm.eventBus != nil {
		pm.eventBus.Publish("plugin.initialized", map[string]interface{}{
			"name": name,
		})
	}

	return nil
}

// Start يبدأ إضافة
func (pm *PluginManager) Start(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// التحقق من حالة الإضافة
	if plugin.Status() != PluginStatusReady {
		return fmt.Errorf("plugin is not ready: %s", name)
	}

	// بدء الإضافة
	if err := plugin.Start(context.Background()); err != nil {
		pm.health[name] = &PluginHealth{
			Status:    "error",
			Message:   fmt.Sprintf("Start failed: %v", err),
			LastCheck: time.Now(),
			Metrics:   make(map[string]interface{}),
		}
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	// تحديث حالة الصحة
	pm.health[name] = &PluginHealth{
		Status:    "running",
		Message:   "Plugin is running",
		LastCheck: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	pm.logger.Info("تم بدء الإضافة",
		zap.String("plugin_name", name))

	// نشر حدث البدء
	if pm.eventBus != nil {
		pm.eventBus.Publish("plugin.started", map[string]interface{}{
			"name": name,
		})
	}

	return nil
}

// Stop يوقف إضافة
func (pm *PluginManager) Stop(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// التحقق من حالة الإضافة
	if plugin.Status() != PluginStatusRunning {
		return fmt.Errorf("plugin is not running: %s", name)
	}

	// إيقاف الإضافة
	if err := plugin.Stop(context.Background()); err != nil {
		pm.health[name] = &PluginHealth{
			Status:    "error",
			Message:   fmt.Sprintf("Stop failed: %v", err),
			LastCheck: time.Now(),
			Metrics:   make(map[string]interface{}),
		}
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	// تحديث حالة الصحة
	pm.health[name] = &PluginHealth{
		Status:    "stopped",
		Message:   "Plugin stopped",
		LastCheck: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	pm.logger.Info("تم إيقاف الإضافة",
		zap.String("plugin_name", name))

	// نشر حدث الإيقاف
	if pm.eventBus != nil {
		pm.eventBus.Publish("plugin.stopped", map[string]interface{}{
			"name": name,
		})
	}

	return nil
}

// GetPlugin يحصل على إضافة
func (pm *PluginManager) GetPlugin(name string) (Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return plugin, nil
}

// GetAllPlugins يحصل على جميع الإضافات
func (pm *PluginManager) GetAllPlugins() []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]Plugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// GetPluginMetadata يحصل على بيانات وصفية للإضافة
func (pm *PluginManager) GetPluginMetadata(name string) (*PluginMetadata, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metadata, exists := pm.metadata[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return metadata, nil
}

// GetPluginHealth يحصل على حالة صحة الإضافة
func (pm *PluginManager) GetPluginHealth(name string) (*PluginHealth, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	health, exists := pm.health[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return health, nil
}

// GetPluginsByCapability يحصل على الإضافات حسب القدرة
func (pm *PluginManager) GetPluginsByCapability(capability string) []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]Plugin, 0)
	for _, plugin := range pm.plugins {
		for _, cap := range plugin.GetCapabilities() {
			if cap == capability {
				plugins = append(plugins, plugin)
				break
			}
		}
	}

	return plugins
}

// GetSummary يحصل على ملخص الإضافات
func (pm *PluginManager) GetSummary() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	totalCount := len(pm.plugins)
	runningCount := 0
	stoppedCount := 0
	errorCount := 0

	for _, health := range pm.health {
		switch health.Status {
		case "running":
			runningCount++
		case "stopped":
			stoppedCount++
		case "error":
			errorCount++
		}
	}

	return map[string]interface{}{
		"total_plugins": totalCount,
		"running":       runningCount,
		"stopped":       stoppedCount,
		"error":         errorCount,
	}
}
