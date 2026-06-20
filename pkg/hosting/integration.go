package hosting

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MortalArena/Musketeers/pkg/orchestrator"
)

// ============================================================
// Hosting Integration - تكامل نظام الاستضافة
// ============================================================

// HostingIntegrator يربط حزمة الاستضافة مع نظام التخزين في orchestrator
type HostingIntegrator struct {
	hostingManager   *HostingManager
	storageConnector *orchestrator.StorageConnector
}

// NewHostingIntegrator إنشاء مُكامل الاستضافة
func NewHostingIntegrator(hostingManager *HostingManager, storageConnector *orchestrator.StorageConnector) *HostingIntegrator {
	return &HostingIntegrator{
		hostingManager:   hostingManager,
		storageConnector: storageConnector,
	}
}

// CreateHTTPHandler إنشاء معالج HTTP يدمج الاستضافة مع التخزين
func (hi *HostingIntegrator) CreateHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			hi.handleGet(w, r)
		case http.MethodPost:
			hi.handlePost(w, r)
		case http.MethodPut:
			hi.handlePut(w, r)
		case http.MethodDelete:
			hi.handleDelete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// handleGet معالجة طلبات GET
func (hi *HostingIntegrator) handleGet(w http.ResponseWriter, r *http.Request) {
	// استخراج fileID من المسار
	fileID := r.URL.Path[len("/files/"):]
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	// استرجاع الملف من نظام التخزين
	file, err := hi.storageConnector.RetrieveFile(fileID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve file: %v", err), http.StatusInternalServerError)
		return
	}

	// إرجاع الملف
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(file.Content)
}

// handlePost معالجة طلبات POST
func (hi *HostingIntegrator) handlePost(w http.ResponseWriter, r *http.Request) {
	// قراءة البيانات من الطلب
	data := make([]byte, r.ContentLength)
	_, err := r.Body.Read(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read data: %v", err), http.StatusBadRequest)
		return
	}

	// إنشاء ملف التخزين
	file := &orchestrator.StorageFile{
		Name:     r.URL.Query().Get("name"),
		Size:     int64(len(data)),
		Type:     r.URL.Query().Get("type"),
		Content:  data,
		OwnerDID: "default",
	}

	// تخزين الملف في نظام التخزين
	err = hi.storageConnector.StoreFile(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store file: %v", err), http.StatusInternalServerError)
		return
	}

	// إرجاع fileID
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"file_id": "%s"}`, file.ID)
}

// handlePut معالجة طلبات PUT
func (hi *HostingIntegrator) handlePut(w http.ResponseWriter, r *http.Request) {
	// استخراج fileID من المسار
	fileID := r.URL.Path[len("/files/"):]
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	// قراءة البيانات من الطلب
	data := make([]byte, r.ContentLength)
	_, err := r.Body.Read(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read data: %v", err), http.StatusBadRequest)
		return
	}

	// تحديث الملف في نظام التخزين
	// (ملاحظة: StorageConnector ليس لديه طريقة UpdateFile حالياً)
	// يجب استرجاع الملف أولاً ثم تخزينه مرة أخرى
	file, err := hi.storageConnector.RetrieveFile(fileID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve file: %v", err), http.StatusInternalServerError)
		return
	}

	// تحديث المحتوى
	file.Content = data
	file.Size = int64(len(data))
	file.UpdatedAt = time.Now()

	// تخزين الملف المحدث
	err = hi.storageConnector.StoreFile(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update file: %v", err), http.StatusInternalServerError)
		return
	}

	// إرجاع النجاح
	w.WriteHeader(http.StatusOK)
}

// handleDelete معالجة طلبات DELETE
func (hi *HostingIntegrator) handleDelete(w http.ResponseWriter, r *http.Request) {
	// استخراج fileID من المسار
	fileID := r.URL.Path[len("/files/"):]
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	// حذف الملف من نظام التخزين
	// (ملاحظة: StorageConnector ليس لديه طريقة DeleteFile حالياً)
	// يجب إضافة هذه الطريقة إلى StorageConnector
	http.Error(w, "Delete not yet implemented in StorageConnector", http.StatusNotImplemented)
}

// SetupHostingServer إعداد خادم استضافة مع معالج HTTP
func (hi *HostingIntegrator) SetupHostingServer(name string, config *HostingConfig) error {
	// إضافة الخادم
	if err := hi.hostingManager.AddServer(name, config); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	// الحصول على الخادم
	server, err := hi.hostingManager.GetServer(name)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// تعيين معالج HTTP
	handler := hi.CreateHTTPHandler()
	server.SetHandler(handler)

	return nil
}

// StartHosting بدء تشغيل خادم الاستضافة
func (hi *HostingIntegrator) StartHosting(name string) error {
	return hi.hostingManager.StartServer(name)
}

// StopHosting إيقاف خادم الاستضافة
func (hi *HostingIntegrator) StopHosting(name string) error {
	return hi.hostingManager.StopServer(name)
}
