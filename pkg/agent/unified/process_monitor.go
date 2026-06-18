package unified

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ProcessMonitor يراقب العمليات النشطة
type ProcessMonitor struct {
	sessionID string
	logger    *zap.Logger

	// حالة المراقبة
	monitoring         bool
	monitoredProcesses map[string]*ProcessInfo
	processesMu        sync.RWMutex

	// قنوات الأحداث
	processEvents chan *ProcessEvent

	// WaitGroup
	wg sync.WaitGroup
}

// ProcessInfo معلومات العملية
type ProcessInfo struct {
	PID         int
	Name        string
	CPUUsage    float64
	MemoryUsage float64
	StartTime   time.Time
	Status      ProcessStatus
}

// ProcessStatus حالة العملية
type ProcessStatus string

const (
	ProcessStatusRunning ProcessStatus = "running"
	ProcessStatusStopped ProcessStatus = "stopped"
	ProcessStatusFailed  ProcessStatus = "failed"
	ProcessStatusUnknown ProcessStatus = "unknown"
)

// ProcessEvent حدث عملية
type ProcessEvent struct {
	Type      ProcessEventType
	Process   *ProcessInfo
	Timestamp time.Time
}

// ProcessEventType نوع حدث العملية
type ProcessEventType string

const (
	ProcessEventTypeStarted ProcessEventType = "started"
	ProcessEventTypeStopped ProcessEventType = "stopped"
	ProcessEventTypeFailed  ProcessEventType = "failed"
	ProcessEventTypeUpdated ProcessEventType = "updated"
)

// NewProcessMonitor ينشئ مراقب عمليات جديد
func NewProcessMonitor(sessionID string, logger *zap.Logger) *ProcessMonitor {
	return &ProcessMonitor{
		sessionID:          sessionID,
		logger:             logger,
		monitoring:         false,
		monitoredProcesses: make(map[string]*ProcessInfo),
		processEvents:      make(chan *ProcessEvent, 100),
	}
}

// Start يبدأ مراقبة العمليات
func (pm *ProcessMonitor) Start(ctx context.Context) error {
	pm.processesMu.Lock()
	defer pm.processesMu.Unlock()

	if pm.monitoring {
		return nil
	}

	pm.monitoring = true

	// بدء معالج الأحداث
	pm.wg.Add(1)
	go pm.handleProcessEvents(ctx)

	// بدء مراقب الموارد
	pm.wg.Add(1)
	go pm.monitorResources(ctx)

	pm.logger.Info("Process monitor started",
		zap.String("session_id", pm.sessionID),
	)

	return nil
}

// Stop يوقف مراقبة العمليات
func (pm *ProcessMonitor) Stop() {
	pm.processesMu.Lock()
	defer pm.processesMu.Unlock()

	if !pm.monitoring {
		return
	}

	pm.monitoring = false

	// إغلاق قناة الأحداث
	close(pm.processEvents)

	// انتظار انتهاء جميع goroutines
	pm.wg.Wait()

	pm.logger.Info("Process monitor stopped",
		zap.String("session_id", pm.sessionID),
	)
}

// MonitorProcess يراقب عملية معينة
func (pm *ProcessMonitor) MonitorProcess(pid int, name string) error {
	pm.processesMu.Lock()
	defer pm.processesMu.Unlock()

	processKey := fmt.Sprintf("%d", pid)

	// التحقق مما إذا كانت العملية مراقبة بالفعل
	if _, exists := pm.monitoredProcesses[processKey]; exists {
		return nil
	}

	// إضافة العملية للمراقبة
	pm.monitoredProcesses[processKey] = &ProcessInfo{
		PID:       pid,
		Name:      name,
		StartTime: time.Now(),
		Status:    ProcessStatusRunning,
	}

	// إرسال حدث البدء
	event := &ProcessEvent{
		Type:      ProcessEventTypeStarted,
		Process:   pm.monitoredProcesses[processKey],
		Timestamp: time.Now(),
	}

	select {
	case pm.processEvents <- event:
	default:
		pm.logger.Warn("Process events channel full",
			zap.Int("pid", pid),
		)
	}

	pm.logger.Info("Process added to monitor",
		zap.Int("pid", pid),
		zap.String("name", name),
	)

	return nil
}

// UnmonitorProcess يلغي مراقبة عملية معينة
func (pm *ProcessMonitor) UnmonitorProcess(pid int) error {
	pm.processesMu.Lock()
	defer pm.processesMu.Unlock()

	processKey := fmt.Sprintf("%d", pid)

	// التحقق مما إذا كانت العملية مراقبة
	process, exists := pm.monitoredProcesses[processKey]
	if !exists {
		return nil
	}

	// إرسال حدث التوقف
	event := &ProcessEvent{
		Type:      ProcessEventTypeStopped,
		Process:   process,
		Timestamp: time.Now(),
	}

	select {
	case pm.processEvents <- event:
	default:
		pm.logger.Warn("Process events channel full",
			zap.Int("pid", pid),
		)
	}

	// إزالة العملية من المراقبة
	delete(pm.monitoredProcesses, processKey)

	pm.logger.Info("Process removed from monitor",
		zap.Int("pid", pid),
	)

	return nil
}

// handleProcessEvents يعالج أحداث العمليات
func (pm *ProcessMonitor) handleProcessEvents(ctx context.Context) {
	defer pm.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-pm.processEvents:
			if !ok {
				return
			}

			pm.logger.Debug("Process event received",
				zap.String("type", string(event.Type)),
				zap.Int("pid", event.Process.PID),
				zap.String("name", event.Process.Name),
			)
		}
	}
}

// monitorResources يراقب استخدام الموارد
func (pm *ProcessMonitor) monitorResources(ctx context.Context) {
	defer pm.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			pm.updateProcessResources()
		}
	}
}

// updateProcessResources يحدث معلومات الموارد للعمليات
func (pm *ProcessMonitor) updateProcessResources() {
	pm.processesMu.Lock()
	defer pm.processesMu.Unlock()

	for _, process := range pm.monitoredProcesses {
		// محاكاة تحديث الموارد
		// في التطبيق الحقيقي، سيتم قراءة الموارد الفعلية
		process.CPUUsage = 0.0
		process.MemoryUsage = 0.0

		// إرسال حدث التحديث
		event := &ProcessEvent{
			Type:      ProcessEventTypeUpdated,
			Process:   process,
			Timestamp: time.Now(),
		}

		select {
		case pm.processEvents <- event:
		default:
			pm.logger.Warn("Process events channel full",
				zap.Int("pid", process.PID),
			)
		}
	}
}

// GetProcessEvents يرجع قناة أحداث العمليات
func (pm *ProcessMonitor) GetProcessEvents() <-chan *ProcessEvent {
	return pm.processEvents
}

// GetMonitoredProcesses يرجع العمليات المراقبة
func (pm *ProcessMonitor) GetMonitoredProcesses() []*ProcessInfo {
	pm.processesMu.RLock()
	defer pm.processesMu.RUnlock()

	processes := make([]*ProcessInfo, 0, len(pm.monitoredProcesses))
	for _, process := range pm.monitoredProcesses {
		processes = append(processes, process)
	}

	return processes
}

// IsMonitoring يرجع ما إذا كان المراقب يعمل
func (pm *ProcessMonitor) IsMonitoring() bool {
	pm.processesMu.RLock()
	defer pm.processesMu.RUnlock()

	return pm.monitoring
}

// GetStatus يرجع حالة المراقب
func (pm *ProcessMonitor) GetStatus() map[string]interface{} {
	pm.processesMu.RLock()
	defer pm.processesMu.RUnlock()

	return map[string]interface{}{
		"monitoring":          pm.monitoring,
		"monitored_processes": len(pm.monitoredProcesses),
		"session_id":          pm.sessionID,
		"os":                  runtime.GOOS,
		"arch":                runtime.GOARCH,
	}
}

// ExecuteCommand ينفذ أمر ويراقب العملية
func (pm *ProcessMonitor) ExecuteCommand(ctx context.Context, command string, args []string) (int, error) {
	pm.logger.Info("Executing command",
		zap.String("command", command),
		zap.Strings("args", args),
	)

	// إنشاء الأمر
	cmd := exec.CommandContext(ctx, command, args...)

	// بدء الأمر
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start command: %w", err)
	}

	pid := cmd.Process.Pid

	// إضافة العملية للمراقبة
	if err := pm.MonitorProcess(pid, command); err != nil {
		pm.logger.Warn("Failed to monitor process",
			zap.Int("pid", pid),
			zap.Error(err),
		)
	}

	// انتظار انتهاء الأمر
	if err := cmd.Wait(); err != nil {
		// إزالة العملية من المراقبة
		pm.UnmonitorProcess(pid)
		return pid, fmt.Errorf("command failed: %w", err)
	}

	// إزالة العملية من المراقبة
	pm.UnmonitorProcess(pid)

	return pid, nil
}
