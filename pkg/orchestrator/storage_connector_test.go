package orchestrator

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestStorageConnectorCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	if storageConnector == nil {
		t.Fatal("فشل إنشاء StorageConnector")
	}

	t.Log("تم إنشاء StorageConnector بنجاح")
}

func TestStorageConnectorStartStop(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}

	// إيقاف StorageConnector
	if err := storageConnector.Stop(); err != nil {
		t.Fatalf("فشل إيقاف StorageConnector: %v", err)
	}

	t.Log("تم بدء وإيقاف StorageConnector بنجاح")
}

func TestStorageConnectorStoreFile(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}
	defer storageConnector.Stop()

	// إنشاء ملف
	file := &StorageFile{
		Name:     "test.txt",
		Size:     1024,
		Type:     "text/plain",
		Content:  []byte("test content"),
		OwnerDID: "did:example:123",
		Metadata: map[string]interface{}{
			"description": "Test file",
		},
	}

	// تخزين الملف
	if err := storageConnector.StoreFile(file); err != nil {
		t.Fatalf("فشل تخزين الملف: %v", err)
	}

	t.Log("تم تخزين الملف بنجاح")
}

func TestStorageConnectorRetrieveFile(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}
	defer storageConnector.Stop()

	// إنشاء ملف
	file := &StorageFile{
		Name:     "test.txt",
		Size:     1024,
		Type:     "text/plain",
		Content:  []byte("test content"),
		OwnerDID: "did:example:123",
		Metadata: map[string]interface{}{
			"description": "Test file",
		},
	}

	// تخزين الملف
	if err := storageConnector.StoreFile(file); err != nil {
		t.Fatalf("فشل تخزين الملف: %v", err)
	}

	// استرجاع الملف
	retrievedFile, err := storageConnector.RetrieveFile(file.ID)
	if err != nil {
		t.Fatalf("فشل استرجاع الملف: %v", err)
	}

	if retrievedFile.Name != "test.txt" {
		t.Errorf("اسم الملف غير صحيح: got %s, want test.txt", retrievedFile.Name)
	}

	t.Log("تم استرجاع الملف بنجاح")
}

func TestStorageConnectorDeleteFile(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}
	defer storageConnector.Stop()

	// إنشاء ملف
	file := &StorageFile{
		Name:     "test.txt",
		Size:     1024,
		Type:     "text/plain",
		Content:  []byte("test content"),
		OwnerDID: "did:example:123",
		Metadata: map[string]interface{}{
			"description": "Test file",
		},
	}

	// تخزين الملف
	if err := storageConnector.StoreFile(file); err != nil {
		t.Fatalf("فشل تخزين الملف: %v", err)
	}

	// حذف الملف
	if err := storageConnector.DeleteFile(file.ID); err != nil {
		t.Fatalf("فشل حذف الملف: %v", err)
	}

	t.Log("تم حذف الملف بنجاح")
}

func TestStorageConnectorListFiles(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}
	defer storageConnector.Stop()

	// إنشاء ملفات
	file1 := &StorageFile{
		Name:     "test1.txt",
		Size:     1024,
		Type:     "text/plain",
		Content:  []byte("test content 1"),
		OwnerDID: "did:example:123",
		SessionID: "session-123",
	}

	file2 := &StorageFile{
		Name:     "test2.txt",
		Size:     2048,
		Type:     "text/plain",
		Content:  []byte("test content 2"),
		OwnerDID: "did:example:456",
		SessionID: "session-123",
	}

	// تخزين الملفات
	if err := storageConnector.StoreFile(file1); err != nil {
		t.Fatalf("فشل تخزين الملف 1: %v", err)
	}
	if err := storageConnector.StoreFile(file2); err != nil {
		t.Fatalf("فشل تخزين الملف 2: %v", err)
	}

	// الحصول على قائمة الملفات
	files := storageConnector.ListFiles("", "")

	if len(files) == 0 {
		t.Error("يجب أن يكون هناك ملفات")
	}

	t.Logf("عدد الملفات: %d", len(files))
}

func TestStorageConnectorListFilesByOwner(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}
	defer storageConnector.Stop()

	// إنشاء ملفات
	file1 := &StorageFile{
		Name:     "test1.txt",
		Size:     1024,
		Type:     "text/plain",
		Content:  []byte("test content 1"),
		OwnerDID: "did:example:123",
	}

	file2 := &StorageFile{
		Name:     "test2.txt",
		Size:     2048,
		Type:     "text/plain",
		Content:  []byte("test content 2"),
		OwnerDID: "did:example:456",
	}

	// تخزين الملفات
	if err := storageConnector.StoreFile(file1); err != nil {
		t.Fatalf("فشل تخزين الملف 1: %v", err)
	}
	if err := storageConnector.StoreFile(file2); err != nil {
		t.Fatalf("فشل تخزين الملف 2: %v", err)
	}

	// الحصول على قائمة الملفات حسب المالك
	files := storageConnector.ListFiles("did:example:123", "")

	if len(files) == 0 {
		t.Error("يجب أن يكون هناك ملفات من المالك did:example:123")
	}

	t.Logf("عدد ملفات did:example:123: %d", len(files))
}

func TestStorageConnectorListFilesBySession(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}
	defer storageConnector.Stop()

	// إنشاء ملفات
	file1 := &StorageFile{
		Name:     "test1.txt",
		Size:     1024,
		Type:     "text/plain",
		Content:  []byte("test content 1"),
		OwnerDID: "did:example:123",
		SessionID: "session-123",
	}

	file2 := &StorageFile{
		Name:     "test2.txt",
		Size:     2048,
		Type:     "text/plain",
		Content:  []byte("test content 2"),
		OwnerDID: "did:example:456",
		SessionID: "session-456",
	}

	// تخزين الملفات
	if err := storageConnector.StoreFile(file1); err != nil {
		t.Fatalf("فشل تخزين الملف 1: %v", err)
	}
	if err := storageConnector.StoreFile(file2); err != nil {
		t.Fatalf("فشل تخزين الملف 2: %v", err)
	}

	// الحصول على قائمة الملفات حسب الجلسة
	files := storageConnector.ListFiles("", "session-123")

	if len(files) == 0 {
		t.Error("يجب أن يكون هناك ملفات من الجلسة session-123")
	}

	t.Logf("عدد ملفات session-123: %d", len(files))
}

func TestStorageConnectorGetMetrics(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء StorageConnector
	storageConnector := NewStorageConnector(eventBus, nil, zap.NewNop())

	// بدء StorageConnector
	if err := storageConnector.Start(); err != nil {
		t.Fatalf("فشل بدء StorageConnector: %v", err)
	}
	defer storageConnector.Stop()

	// إنشاء ملف
	file := &StorageFile{
		Name:     "test.txt",
		Size:     1024,
		Type:     "text/plain",
		Content:  []byte("test content"),
		OwnerDID: "did:example:123",
	}

	// تخزين الملف
	if err := storageConnector.StoreFile(file); err != nil {
		t.Fatalf("فشل تخزين الملف: %v", err)
	}

	// الحصول على المقاييس
	metrics := storageConnector.GetMetrics()

	if metrics == nil {
		t.Error("يجب أن تكون هناك مقاييس")
	}

	if metrics.FilesStored == 0 {
		t.Error("يجب أن يكون هناك ملفات مخزنة")
	}

	t.Logf("المقاييس: %+v", metrics)
}
