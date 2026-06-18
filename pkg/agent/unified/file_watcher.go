package unified

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// FileWatcher يراقب التغييرات في الملفات
type FileWatcher struct {
	sessionID string
	logger    *zap.Logger

	// fsnotify watcher
	watcher *fsnotify.Watcher

	// حالة المراقبة
	watching       bool
	watchedPaths   map[string]bool
	watchedPathsMu sync.RWMutex

	// قنوات الأحداث
	fileEvents chan *FileEvent

	// WaitGroup
	wg sync.WaitGroup
}

// FileEvent حدث ملف
type FileEvent struct {
	Path      string
	EventType fsnotify.Op
	Timestamp time.Time
}

// NewFileWatcher ينشئ مراقب ملفات جديد
func NewFileWatcher(sessionID string, logger *zap.Logger) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &FileWatcher{
		sessionID:    sessionID,
		logger:       logger,
		watcher:      watcher,
		watching:     false,
		watchedPaths: make(map[string]bool),
		fileEvents:   make(chan *FileEvent, 100),
	}, nil
}

// Start يبدأ مراقبة الملفات
func (fw *FileWatcher) Start(ctx context.Context) error {
	fw.watchedPathsMu.Lock()
	defer fw.watchedPathsMu.Unlock()

	if fw.watching {
		return nil
	}

	fw.watching = true

	// بدء معالج الأحداث
	fw.wg.Add(1)
	go fw.processEvents(ctx)

	fw.logger.Info("File watcher started",
		zap.String("session_id", fw.sessionID),
	)

	return nil
}

// Stop يوقف مراقبة الملفات
func (fw *FileWatcher) Stop() {
	fw.watchedPathsMu.Lock()
	defer fw.watchedPathsMu.Unlock()

	if !fw.watching {
		return
	}

	fw.watching = false

	// إغلاق المراقب
	fw.watcher.Close()

	// إغلاق قناة الأحداث
	close(fw.fileEvents)

	// انتظار انتهاء جميع goroutines
	fw.wg.Wait()

	fw.logger.Info("File watcher stopped",
		zap.String("session_id", fw.sessionID),
	)
}

// WatchPath يراقب مسار معين
func (fw *FileWatcher) WatchPath(path string) error {
	fw.watchedPathsMu.Lock()
	defer fw.watchedPathsMu.Unlock()

	// التحقق مما إذا كان المسار مراقباً بالفعل
	if fw.watchedPaths[path] {
		return nil
	}

	// إضافة المسار للمراقبة
	if err := fw.watcher.Add(path); err != nil {
		return err
	}

	fw.watchedPaths[path] = true

	fw.logger.Info("Path added to watcher",
		zap.String("path", path),
		zap.String("session_id", fw.sessionID),
	)

	return nil
}

// UnwatchPath يلغي مراقبة مسار معين
func (fw *FileWatcher) UnwatchPath(path string) error {
	fw.watchedPathsMu.Lock()
	defer fw.watchedPathsMu.Unlock()

	// التحقق مما إذا كان المسار مراقباً
	if !fw.watchedPaths[path] {
		return nil
	}

	// إزالة المسار من المراقبة
	if err := fw.watcher.Remove(path); err != nil {
		return err
	}

	delete(fw.watchedPaths, path)

	fw.logger.Info("Path removed from watcher",
		zap.String("path", path),
		zap.String("session_id", fw.sessionID),
	)

	return nil
}

// WatchDir يراقب مجلد معين
func (fw *FileWatcher) WatchDir(dir string, recursive bool) error {
	if recursive {
		// مراقبة المجلد بشكل متكرر
		return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// مراقبة المجلدات فقط
			if info.IsDir() {
				if err := fw.WatchPath(path); err != nil {
					fw.logger.Warn("Failed to watch directory",
						zap.String("path", path),
						zap.Error(err),
					)
				}
			}

			return nil
		})
	} else {
		// مراقبة المجلد فقط
		return fw.WatchPath(dir)
	}
}

// processEvents يعالج أحداث الملفات
func (fw *FileWatcher) processEvents(ctx context.Context) {
	defer fw.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// إنشاء حدث الملف
			fileEvent := &FileEvent{
				Path:      event.Name,
				EventType: event.Op,
				Timestamp: time.Now(),
			}

			// إرسال الحدث
			select {
			case fw.fileEvents <- fileEvent:
			default:
				fw.logger.Warn("File events channel full",
					zap.String("path", event.Name),
				)
			}

			fw.logger.Debug("File event received",
				zap.String("path", event.Name),
				zap.String("operation", event.Op.String()),
			)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}

			fw.logger.Error("File watcher error",
				zap.Error(err),
			)
		}
	}
}

// GetFileEvents يرجع قناة أحداث الملفات
func (fw *FileWatcher) GetFileEvents() <-chan *FileEvent {
	return fw.fileEvents
}

// GetWatchedPaths يرجع المسارات المراقبة
func (fw *FileWatcher) GetWatchedPaths() []string {
	fw.watchedPathsMu.RLock()
	defer fw.watchedPathsMu.RUnlock()

	paths := make([]string, 0, len(fw.watchedPaths))
	for path := range fw.watchedPaths {
		paths = append(paths, path)
	}

	return paths
}

// IsWatching يرجع ما إذا كان المراقب يعمل
func (fw *FileWatcher) IsWatching() bool {
	fw.watchedPathsMu.RLock()
	defer fw.watchedPathsMu.RUnlock()

	return fw.watching
}

// GetStatus يرجع حالة المراقب
func (fw *FileWatcher) GetStatus() map[string]interface{} {
	fw.watchedPathsMu.RLock()
	defer fw.watchedPathsMu.RUnlock()

	return map[string]interface{}{
		"watching":      fw.watching,
		"watched_paths": len(fw.watchedPaths),
		"session_id":    fw.sessionID,
	}
}
