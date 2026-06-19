package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BackupManager مدير النسخ الاحتياطي
type BackupManager struct {
	backups      map[string]*Backup
	schedules    map[string]*BackupSchedule
	logger       *zap.Logger
	mu           sync.RWMutex
	storage      BackupStorage
	eventBus     EventBus
	config       *BackupConfig
}

// BackupStorage واجهة تخزين النسخ الاحتياطي
type BackupStorage interface {
	StoreBackup(backup *Backup) error
	RetrieveBackup(backupID string) (*Backup, error)
	ListBackups(filter BackupFilter) ([]*Backup, error)
	DeleteBackup(backupID string) error
}

// EventBus واجهة ناقل الأحداث
type EventBus interface {
	Publish(event string, data interface{}) error
	Subscribe(event string, handler func(data interface{})) error
}

// BackupConfig تكوين النسخ الاحتياطي
type BackupConfig struct {
	BackupDir      string        `json:"backup_dir"`
	RetentionDays  int           `json:"retention_days"`
	MaxBackups     int           `json:"max_backups"`
	Compression    bool          `json:"compression"`
	Encryption     bool          `json:"encryption"`
	EncryptionKey  string        `json:"encryption_key"`
	CheckInterval  time.Duration `json:"check_interval"`
}

// Backup نسخة احتياطية
type Backup struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         BackupType             `json:"type"`
	Source       string                 `json:"source"`
	Destination  string                 `json:"destination"`
	Size         int64                  `json:"size"`
	Status       BackupStatus           `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  time.Time              `json:"completed_at,omitempty"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Checksum     string                 `json:"checksum"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// BackupType نوع النسخ الاحتياطي
type BackupType string

const (
	BackupTypeFull     BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeDifferential BackupType = "differential"
)

// BackupStatus حالة النسخ الاحتياطي
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusInProgress BackupStatus = "in_progress"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
	BackupStatusExpired    BackupStatus = "expired"
)

// BackupSchedule جدولة النسخ الاحتياطي
type BackupSchedule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        BackupType             `json:"type"`
	Source      string                 `json:"source"`
	Schedule    string                 `json:"schedule"` // cron expression
	Enabled     bool                   `json:"enabled"`
	LastRun     time.Time              `json:"last_run"`
	NextRun     time.Time              `json:"next_run"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// BackupFilter فلتر النسخ الاحتياطي
type BackupFilter struct {
	Type       BackupType
	Status     BackupStatus
	Source     string
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
}

// NewBackupManager ينشئ مدير نسخ احتياطي جديد
func NewBackupManager(logger *zap.Logger, storage BackupStorage, eventBus EventBus, config *BackupConfig) *BackupManager {
	return &BackupManager{
		backups:   make(map[string]*Backup),
		schedules: make(map[string]*BackupSchedule),
		logger:    logger,
		storage:   storage,
		eventBus:  eventBus,
		config:    config,
	}
}

// CreateBackup ينشئ نسخة احتياطية جديدة
func (bm *BackupManager) CreateBackup(name, source string, backupType BackupType) (*Backup, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	backupID := fmt.Sprintf("backup_%d", time.Now().UnixNano())

	backup := &Backup{
		ID:          backupID,
		Name:        name,
		Type:        backupType,
		Source:      source,
		Destination: filepath.Join(bm.config.BackupDir, backupID),
		Status:      BackupStatusPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().AddDate(0, 0, bm.config.RetentionDays),
		Metadata:    make(map[string]interface{}),
	}

	bm.backups[backupID] = backup

	bm.logger.Info("تم إنشاء نسخة احتياطية جديدة",
		zap.String("backup_id", backupID),
		zap.String("name", name),
		zap.String("type", string(backupType)),
		zap.String("source", source))

	// بدء عملية النسخ الاحتياطي في الخلفية
	go bm.performBackup(backup)

	return backup, nil
}

// performBackup ينفذ عملية النسخ الاحتياطي
func (bm *BackupManager) performBackup(backup *Backup) {
	bm.mu.Lock()
	backup.Status = BackupStatusInProgress
	bm.mu.Unlock()

	// إنشاء دليل الوجهة
	if err := os.MkdirAll(backup.Destination, 0755); err != nil {
		bm.mu.Lock()
		backup.Status = BackupStatusFailed
		bm.mu.Unlock()
		bm.logger.Error("فشل إنشاء دليل الوجهة",
			zap.String("backup_id", backup.ID),
			zap.Error(err))
		return
	}

	// نسخ الملفات
	size, err := bm.copyDirectory(backup.Source, backup.Destination)
	if err != nil {
		bm.mu.Lock()
		backup.Status = BackupStatusFailed
		bm.mu.Unlock()
		bm.logger.Error("فشل نسخ الملفات",
			zap.String("backup_id", backup.ID),
			zap.Error(err))
		return
	}

	// حساب Checksum
	checksum, err := bm.calculateChecksum(backup.Destination)
	if err != nil {
		bm.logger.Warn("فشل حساب Checksum",
			zap.String("backup_id", backup.ID),
			zap.Error(err))
	}

	bm.mu.Lock()
	backup.Size = size
	backup.Status = BackupStatusCompleted
	backup.CompletedAt = time.Now()
	backup.Checksum = checksum
	bm.mu.Unlock()

	bm.logger.Info("تم إكمال النسخ الاحتياطي",
		zap.String("backup_id", backup.ID),
		zap.Int64("size", size),
		zap.String("checksum", checksum))

	// تخزين النسخ الاحتياطي
	if bm.storage != nil {
		if err := bm.storage.StoreBackup(backup); err != nil {
			bm.logger.Error("فشل تخزين النسخ الاحتياطي",
				zap.String("backup_id", backup.ID),
				zap.Error(err))
		}
	}

	// نشر حدث إكمال النسخ الاحتياطي
	if bm.eventBus != nil {
		bm.eventBus.Publish("backup.completed", map[string]interface{}{
			"backup_id": backup.ID,
			"name":      backup.Name,
			"size":      size,
		})
	}
}

// copyDirectory ينسخ دليل
func (bm *BackupManager) copyDirectory(src, dst string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return bm.copyFile(path, dstPath, &totalSize)
	})

	return totalSize, err
}

// copyFile ينسخ ملف
func (bm *BackupManager) copyFile(src, dst string, totalSize *int64) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	size, err := io.Copy(destination, source)
	if err != nil {
		return err
	}

	*totalSize += size
	return nil
}

// calculateChecksum يحسب Checksum
func (bm *BackupManager) calculateChecksum(path string) (string, error) {
	// تنفيذ بسيط - في التطبيق الحقيقي يجب استخدام SHA256
	return fmt.Sprintf("checksum_%d", time.Now().UnixNano()), nil
}

// RestoreBackup يستعيد نسخة احتياطية
func (bm *BackupManager) RestoreBackup(backupID, destination string) error {
	bm.mu.RLock()
	backup, exists := bm.backups[backupID]
	bm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	if backup.Status != BackupStatusCompleted {
		return fmt.Errorf("backup is not completed: %s", backupID)
	}

	bm.logger.Info("بدء استعادة النسخ الاحتياطي",
		zap.String("backup_id", backupID),
		zap.String("destination", destination))

	// نسخ الملفات من النسخ الاحتياطي إلى الوجهة
	size, err := bm.copyDirectory(backup.Destination, destination)
	if err != nil {
		bm.logger.Error("فشل استعادة النسخ الاحتياطي",
			zap.String("backup_id", backupID),
			zap.Error(err))
		return err
	}

	bm.logger.Info("تم استعادة النسخ الاحتياطي بنجاح",
		zap.String("backup_id", backupID),
		zap.Int64("size", size))

	// نشر حدث استعادة النسخ الاحتياطي
	if bm.eventBus != nil {
		bm.eventBus.Publish("backup.restored", map[string]interface{}{
			"backup_id":   backupID,
			"destination": destination,
			"size":        size,
		})
	}

	return nil
}

// DeleteBackup يحذف نسخة احتياطية
func (bm *BackupManager) DeleteBackup(backupID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	backup, exists := bm.backups[backupID]
	if !exists {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// حذف الملفات
	if err := os.RemoveAll(backup.Destination); err != nil {
		bm.logger.Error("فشل حذف ملفات النسخ الاحتياطي",
			zap.String("backup_id", backupID),
			zap.Error(err))
		return err
	}

	delete(bm.backups, backupID)

	bm.logger.Info("تم حذف النسخ الاحتياطي",
		zap.String("backup_id", backupID))

	// حذف من التخزين
	if bm.storage != nil {
		if err := bm.storage.DeleteBackup(backupID); err != nil {
			bm.logger.Error("فشل حذف النسخ الاحتياطي من التخزين",
				zap.String("backup_id", backupID),
				zap.Error(err))
		}
	}

	return nil
}

// GetBackup يحصل على نسخة احتياطي
func (bm *BackupManager) GetBackup(backupID string) (*Backup, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	backup, exists := bm.backups[backupID]
	if !exists {
		return nil, fmt.Errorf("backup not found: %s", backupID)
	}

	return backup, nil
}

// GetAllBackups يحصل على جميع النسخ الاحتياطية
func (bm *BackupManager) GetAllBackups() []*Backup {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	backups := make([]*Backup, 0, len(bm.backups))
	for _, backup := range bm.backups {
		backups = append(backups, backup)
	}

	return backups
}

// CreateSchedule ينشئ جدولة نسخ احتياطي
func (bm *BackupManager) CreateSchedule(name, source, schedule string, backupType BackupType) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	scheduleID := fmt.Sprintf("schedule_%d", time.Now().UnixNano())

	bm.schedules[scheduleID] = &BackupSchedule{
		ID:        scheduleID,
		Name:      name,
		Type:      backupType,
		Source:    source,
		Schedule:  schedule,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	bm.logger.Info("تم إنشاء جدولة نسخ احتياطي",
		zap.String("schedule_id", scheduleID),
		zap.String("name", name),
		zap.String("schedule", schedule))

	return nil
}

// GetSchedule يحصل على جدولة
func (bm *BackupManager) GetSchedule(scheduleID string) (*BackupSchedule, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	schedule, exists := bm.schedules[scheduleID]
	if !exists {
		return nil, fmt.Errorf("schedule not found: %s", scheduleID)
	}

	return schedule, nil
}

// GetAllSchedules يحصل على جميع الجداول
func (bm *BackupManager) GetAllSchedules() []*BackupSchedule {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	schedules := make([]*BackupSchedule, 0, len(bm.schedules))
	for _, schedule := range bm.schedules {
		schedules = append(schedules, schedule)
	}

	return schedules
}

// CleanupExpiredBackups ينظف النسخ الاحتياطية المنتهية
func (bm *BackupManager) CleanupExpiredBackups() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for id, backup := range bm.backups {
		if now.After(backup.ExpiresAt) {
			backup.Status = BackupStatusExpired
			if err := os.RemoveAll(backup.Destination); err != nil {
				bm.logger.Error("فشل حذف نسخة احتياطي منتهية",
					zap.String("backup_id", id),
					zap.Error(err))
			} else {
				delete(bm.backups, id)
				expiredCount++
			}
		}
	}

	bm.logger.Info("تم تنظيف النسخ الاحتياطية المنتهية",
		zap.Int("expired_count", expiredCount))

	return nil
}

// GetSummary يحصل على ملخص النسخ الاحتياطي
func (bm *BackupManager) GetSummary() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	totalBackups := len(bm.backups)
	totalSchedules := len(bm.schedules)
	totalSize := int64(0)
	completedCount := 0
	failedCount := 0

	for _, backup := range bm.backups {
		totalSize += backup.Size
		switch backup.Status {
		case BackupStatusCompleted:
			completedCount++
		case BackupStatusFailed:
			failedCount++
		}
	}

	return map[string]interface{}{
		"total_backups":   totalBackups,
		"total_schedules": totalSchedules,
		"total_size":      totalSize,
		"completed":       completedCount,
		"failed":         failedCount,
	}
}
