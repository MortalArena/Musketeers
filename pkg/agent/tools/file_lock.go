package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FileLockManager يدير أقفال الملفات لمنع التعارضات في بيئة متعددة الوكلاء
type FileLockManager struct {
	locks      map[string]*FileLock
	mu         sync.RWMutex
	logger     *zap.Logger
	lockDir    string
	defaultTimeout time.Duration
	maxWaitTime   time.Duration
}

// FileLock يمثل قفل ملف
type FileLock struct {
	filePath    string
	lockFile    string
	owner       string
	acquiredAt  time.Time
	timeout     time.Duration
	mu          sync.Mutex
}

// NewFileLockManager ينشئ مدير أقفال ملفات جديد
func NewFileLockManager(lockDir string, logger *zap.Logger) *FileLockManager {
	if lockDir == "" {
		lockDir = filepath.Join(os.TempDir(), "musketeers_locks")
	}

	// إنشاء مجلد الأقفال إذا لم يكن موجوداً
	os.MkdirAll(lockDir, 0755)

	return &FileLockManager{
		locks:         make(map[string]*FileLock),
		logger:        logger,
		lockDir:       lockDir,
		defaultTimeout: 30 * time.Second,
		maxWaitTime:    5 * time.Minute,
	}
}

// Lock يحصل على قفل على ملف
func (flm *FileLockManager) Lock(ctx context.Context, filePath string, owner string) error {
	return flm.LockWithTimeout(ctx, filePath, owner, flm.defaultTimeout)
}

// LockWithTimeout يحصل على قفل على ملف مع مهلة محددة
func (flm *FileLockManager) LockWithTimeout(ctx context.Context, filePath string, owner string, timeout time.Duration) error {
	flm.mu.Lock()
	defer flm.mu.Unlock()

	// التحقق مما إذا كان القفل موجوداً بالفعل
	if existingLock, exists := flm.locks[filePath]; exists {
		// التحقق مما إذا كان القفل منتهي
		if time.Since(existingLock.acquiredAt) > existingLock.timeout {
			// القفل منتهي، حذفه
			delete(flm.locks, filePath)
			os.Remove(existingLock.lockFile)
		} else {
			// القفل لا يزال صالحاً
			return fmt.Errorf("file %s is locked by %s", filePath, existingLock.owner)
		}
	}

	// إنشاء قفل جديد
	lockFile := filepath.Join(flm.lockDir, fmt.Sprintf("%s.lock", filepath.Base(filePath)))
	ownerData := fmt.Sprintf("%s|%d", owner, time.Now().Unix())

	// طريقة القفل عبر المحتوى: اكتب بياناتك ثم اقرأها للتأكد
	// هذا يعمل حتى على أنظمة حيث O_EXCL ليس ذرياً بالكامل
	// الفكرة: اكتب claim، اقرأه — إذا كان لا يزال لك، فالقفل لك
	for retries := 0; retries < 3; retries++ {
		// اقرأ ملف القفل الموجود (إن وُجد)
		if existingData, readErr := os.ReadFile(lockFile); readErr == nil {
			parts := strings.SplitN(string(existingData), "|", 2)
			if len(parts) == 2 {
				if ts, parseErr := strconv.ParseInt(parts[1], 10, 64); parseErr == nil {
					if time.Since(time.Unix(ts, 0)) <= timeout {
						return fmt.Errorf("file %s is locked by %s (cross-executor)", filePath, parts[0])
					}
				}
			}
			// منته أو تالف — احذفه للسماح بمحاولة جديدة
			os.Remove(lockFile)
		}

		// اكتب claim
		if writeErr := os.WriteFile(lockFile, []byte(ownerData), 0644); writeErr != nil {
			return fmt.Errorf("failed to write lock file: %w", writeErr)
		}

		// اقرأه للتأكد — إذا تطابق، القفل لنا
		if readBack, readErr := os.ReadFile(lockFile); readErr == nil {
			if string(readBack) == ownerData {
				// القفل لنا — اكسر الحلقة
				break
			}
		}

		// شخص آخر كتب محتواه بعدنا — انتظر قصيراً وحاول مجدداً
		if retries < 2 {
			time.Sleep(time.Millisecond)
		}
	}

	// التحقق النهائي: اقرأ ملف القفل وتأكد أنه لا يزال لنا
	finalData, readErr := os.ReadFile(lockFile)
	if readErr != nil || string(finalData) != ownerData {
		return fmt.Errorf("failed to acquire cross-executor lock for %s", filePath)
	}

	lock := &FileLock{
		filePath:   filePath,
		lockFile:   lockFile,
		owner:      owner,
		acquiredAt: time.Now(),
		timeout:    timeout,
	}

	flm.locks[filePath] = lock

	flm.logger.Debug("File lock acquired",
		zap.String("file", filePath),
		zap.String("owner", owner),
		zap.Duration("timeout", timeout),
	)

	return nil
}

// TryLock يحاول الحصول على قفل على ملف بدون انتظار
func (flm *FileLockManager) TryLock(filePath string, owner string) error {
	flm.mu.Lock()
	defer flm.mu.Unlock()

	// التحقق مما إذا كان القفل موجوداً بالفعل
	if existingLock, exists := flm.locks[filePath]; exists {
		// التحقق مما إذا كان القفل منتهي
		if time.Since(existingLock.acquiredAt) > existingLock.timeout {
			// القفل منتهي، حذفه
			delete(flm.locks, filePath)
			os.Remove(existingLock.lockFile)
		} else {
			// القفل لا يزال صالحاً
			return fmt.Errorf("file %s is locked by %s", filePath, existingLock.owner)
		}
	}

	// إنشاء قفل جديد
	lockFile := filepath.Join(flm.lockDir, fmt.Sprintf("%s.lock", filepath.Base(filePath)))
	lock := &FileLock{
		filePath:   filePath,
		lockFile:   lockFile,
		owner:      owner,
		acquiredAt: time.Now(),
		timeout:    flm.defaultTimeout,
	}

	// إنشاء ملف القفل
	if err := os.WriteFile(lockFile, []byte(fmt.Sprintf("%s|%d", owner, time.Now().Unix())), 0644); err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	flm.locks[filePath] = lock

	flm.logger.Debug("File lock acquired (try)",
		zap.String("file", filePath),
		zap.String("owner", owner),
	)

	return nil
}

// Unlock يفرغ قفل ملف
func (flm *FileLockManager) Unlock(filePath string) error {
	flm.mu.Lock()
	defer flm.mu.Unlock()

	lock, exists := flm.locks[filePath]
	if !exists {
		return fmt.Errorf("file %s is not locked", filePath)
	}

	// حذف ملف القفل
	if err := os.Remove(lock.lockFile); err != nil {
		flm.logger.Warn("Failed to remove lock file",
			zap.String("lock_file", lock.lockFile),
			zap.Error(err),
		)
	}

	// حذف القفل من الخريطة
	delete(flm.locks, filePath)

	flm.logger.Debug("File lock released",
		zap.String("file", filePath),
		zap.String("owner", lock.owner),
	)

	return nil
}

// IsLocked يتحقق مما إذا كان الملف مقفولاً
func (flm *FileLockManager) IsLocked(filePath string) bool {
	flm.mu.RLock()
	defer flm.mu.RUnlock()

	lock, exists := flm.locks[filePath]
	if !exists {
		return false
	}

	// التحقق مما إذا كان القفل منتهي
	if time.Since(lock.acquiredAt) > lock.timeout {
		return false
	}

	return true
}

// GetLockOwner يحصل على مالك قفل الملف
func (flm *FileLockManager) GetLockOwner(filePath string) (string, error) {
	flm.mu.RLock()
	defer flm.mu.RUnlock()

	lock, exists := flm.locks[filePath]
	if !exists {
		return "", fmt.Errorf("file %s is not locked", filePath)
	}

	return lock.owner, nil
}

// CleanupExpiredLocks ينظف الأقفال المنتهية
func (flm *FileLockManager) CleanupExpiredLocks() {
	flm.mu.Lock()
	defer flm.mu.Unlock()

	now := time.Now()
	expiredLocks := make([]string, 0)

	for filePath, lock := range flm.locks {
		if now.Sub(lock.acquiredAt) > lock.timeout {
			expiredLocks = append(expiredLocks, filePath)
		}
	}

	for _, filePath := range expiredLocks {
		lock := flm.locks[filePath]
		os.Remove(lock.lockFile)
		delete(flm.locks, filePath)

		flm.logger.Debug("Expired lock cleaned up",
			zap.String("file", filePath),
			zap.String("owner", lock.owner),
		)
	}

	if len(expiredLocks) > 0 {
		flm.logger.Info("Cleaned up expired locks",
			zap.Int("count", len(expiredLocks)),
		)
	}
}

// GetLockInfo يحصل على معلومات عن الأقفال النشطة
func (flm *FileLockManager) GetLockInfo() map[string]interface{} {
	flm.mu.RLock()
	defer flm.mu.RUnlock()

	locks := make([]map[string]interface{}, 0, len(flm.locks))
	for filePath, lock := range flm.locks {
		locks = append(locks, map[string]interface{}{
			"file":       filePath,
			"owner":      lock.owner,
			"acquired_at": lock.acquiredAt,
			"timeout":    lock.timeout,
			"age":        time.Since(lock.acquiredAt),
		})
	}

	return map[string]interface{}{
		"total_locks": len(flm.locks),
		"locks":       locks,
		"lock_dir":    flm.lockDir,
	}
}

// StartCleanupLoop يبدأ حلقة تنظيف دورية للأقفال المنتهية
func (flm *FileLockManager) StartCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			flm.CleanupExpiredLocks()
		}
	}
}
