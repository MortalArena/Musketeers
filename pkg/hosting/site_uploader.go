package hosting

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SiteUploader مسؤول عن رفع المواقع
type SiteUploader struct {
	hostingService *P2PHostingService
}

// NewSiteUploader ينشئ uploader جديد
func NewSiteUploader(hostingService *P2PHostingService) *SiteUploader {
	return &SiteUploader{
		hostingService: hostingService,
	}
}

// UploadFromDirectory يرفع موقع من مجلد
func (u *SiteUploader) UploadFromDirectory(ctx context.Context, siteName, dirPath string) (string, error) {
	// قراءة كل الملفات من المجلد
	files := make(map[string][]byte)

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// قراءة الملف
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// حساب المسار النسبي
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// حفظ في الخريطة
		files["/"+relPath] = content

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	// نشر الموقع
	return u.hostingService.DeploySite(ctx, siteName, files)
}

// UploadFromFiles يرفع موقع من قائمة ملفات
func (u *SiteUploader) UploadFromFiles(ctx context.Context, siteName string, files map[string][]byte) (string, error) {
	return u.hostingService.DeploySite(ctx, siteName, files)
}
