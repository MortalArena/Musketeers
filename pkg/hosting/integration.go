package hosting

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MortalArena/Musketeers/pkg/orchestrator"
)

type HostingIntegrator struct {
	hostingManager   *HostingManager
	storageConnector *orchestrator.StorageConnector
}

func NewHostingIntegrator(hostingManager *HostingManager, storageConnector *orchestrator.StorageConnector) *HostingIntegrator {
	return &HostingIntegrator{
		hostingManager:   hostingManager,
		storageConnector: storageConnector,
	}
}

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

func sanitizeFileID(fileID string) string {
	clean := strings.TrimSpace(fileID)
	clean = strings.TrimPrefix(clean, "/")
	clean = strings.TrimSuffix(clean, "/")
	if clean == "" {
		return ""
	}
	for _, c := range clean {
		if c < 32 || c > 126 {
			return ""
		}
	}
	if strings.Contains(clean, "..") || strings.Contains(clean, "\\") || strings.Contains(clean, "//") {
		return ""
	}
	return clean
}

func (hi *HostingIntegrator) handleGet(w http.ResponseWriter, r *http.Request) {
	rawFileID := strings.TrimPrefix(r.URL.Path, "/files/")
	fileID := sanitizeFileID(rawFileID)
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	file, err := hi.storageConnector.RetrieveFile(fileID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(file.Content)
}

func (hi *HostingIntegrator) handlePost(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 100<<20))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read data: %v", err), http.StatusBadRequest)
		return
	}

	file := &orchestrator.StorageFile{
		Name:     r.URL.Query().Get("name"),
		Size:     int64(len(data)),
		Type:     r.URL.Query().Get("type"),
		Content:  data,
		OwnerDID: "default",
	}

	err = hi.storageConnector.StoreFile(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"file_id": "%s"}`, file.ID)
}

func (hi *HostingIntegrator) handlePut(w http.ResponseWriter, r *http.Request) {
	rawFileID := strings.TrimPrefix(r.URL.Path, "/files/")
	fileID := sanitizeFileID(rawFileID)
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, 100<<20))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read data: %v", err), http.StatusBadRequest)
		return
	}

	file, err := hi.storageConnector.RetrieveFile(fileID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve file: %v", err), http.StatusInternalServerError)
		return
	}

	file.Content = data
	file.Size = int64(len(data))
	file.UpdatedAt = time.Now()

	err = hi.storageConnector.StoreFile(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (hi *HostingIntegrator) handleDelete(w http.ResponseWriter, r *http.Request) {
	rawFileID := strings.TrimPrefix(r.URL.Path, "/files/")
	fileID := sanitizeFileID(rawFileID)
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	http.Error(w, "Delete not yet implemented in StorageConnector", http.StatusNotImplemented)
}

func (hi *HostingIntegrator) SetupHostingServer(name string, config *HostingConfig) error {
	if err := hi.hostingManager.AddServer(name, config); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	server, err := hi.hostingManager.GetServer(name)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	handler := hi.CreateHTTPHandler()
	server.SetHandler(handler)

	return nil
}

func (hi *HostingIntegrator) StartHosting(name string) error {
	return hi.hostingManager.StartServer(name)
}

func (hi *HostingIntegrator) StopHosting(name string) error {
	return hi.hostingManager.StopServer(name)
}
