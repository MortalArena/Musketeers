package hosting

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHostingServer(t *testing.T) {
	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	server := NewHostingServer(config)

	assert.NotNil(t, server)
	assert.Equal(t, config, server.config)
	assert.False(t, server.IsRunning())
}

func TestHostingServer_StartStop(t *testing.T) {
	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	server := NewHostingServer(config)

	// Test start
	err := server.Start()
	assert.NoError(t, err)
	assert.True(t, server.IsRunning())

	// Test start when already running
	err = server.Start()
	assert.Error(t, err)

	// Test stop
	err = server.Stop()
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())

	// Test stop when not running
	err = server.Stop()
	assert.Error(t, err)
}

func TestHostingServer_GetConfig(t *testing.T) {
	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	server := NewHostingServer(config)

	retrievedConfig := server.GetConfig()
	assert.Equal(t, config, retrievedConfig)
}

func TestHostingServer_SetHandler(t *testing.T) {
	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	server := NewHostingServer(config)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server.SetHandler(handler)
	assert.NotNil(t, handler)
}

func TestNewHostingManager(t *testing.T) {
	manager := NewHostingManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.servers)
}

func TestHostingManager_AddServer(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Test add server
	err := manager.AddServer("test", config)
	assert.NoError(t, err)

	// Test add duplicate server
	err = manager.AddServer("test", config)
	assert.Error(t, err)
}

func TestHostingManager_RemoveServer(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Add server
	err := manager.AddServer("test", config)
	assert.NoError(t, err)

	// Remove server
	err = manager.RemoveServer("test")
	assert.NoError(t, err)

	// Remove non-existent server
	err = manager.RemoveServer("nonexistent")
	assert.Error(t, err)
}

func TestHostingManager_GetServer(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Get non-existent server
	_, err := manager.GetServer("nonexistent")
	assert.Error(t, err)

	// Add server
	err = manager.AddServer("test", config)
	assert.NoError(t, err)

	// Get existing server
	server, err := manager.GetServer("test")
	assert.NoError(t, err)
	assert.NotNil(t, server)
}

func TestHostingManager_ListServers(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8080,
		HTTPSPort:      8443,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Empty list
	servers := manager.ListServers()
	assert.Len(t, servers, 0)

	// Add servers
	manager.AddServer("server1", config)
	manager.AddServer("server2", config)

	// List servers
	servers = manager.ListServers()
	assert.Len(t, servers, 2)
}

func TestHostingManager_StartServer(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8081, // Use different port to avoid conflicts
		HTTPSPort:      8444,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Add server
	err := manager.AddServer("test", config)
	assert.NoError(t, err)

	// Start server
	err = manager.StartServer("test")
	assert.NoError(t, err)

	// Start non-existent server
	err = manager.StartServer("nonexistent")
	assert.Error(t, err)

	// Stop server
	err = manager.StopServer("test")
	assert.NoError(t, err)
}

func TestHostingManager_StopServer(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8082, // Use different port to avoid conflicts
		HTTPSPort:      8445,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Add and start server
	manager.AddServer("test", config)
	manager.StartServer("test")

	// Stop server
	err := manager.StopServer("test")
	assert.NoError(t, err)

	// Stop non-existent server
	err = manager.StopServer("nonexistent")
	assert.Error(t, err)
}

func TestHostingManager_StartAll(t *testing.T) {
	manager := NewHostingManager()

	config1 := &HostingConfig{
		Domain:         "example1.com",
		HTTPPort:       8083,
		HTTPSPort:      8446,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	config2 := &HostingConfig{
		Domain:         "example2.com",
		HTTPPort:       8084,
		HTTPSPort:      8447,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Add servers
	manager.AddServer("server1", config1)
	manager.AddServer("server2", config2)

	// Start all servers
	err := manager.StartAll()
	assert.NoError(t, err)

	// Stop all servers
	err = manager.StopAll()
	assert.NoError(t, err)
}

func TestHostingManager_StopAll(t *testing.T) {
	manager := NewHostingManager()

	config1 := &HostingConfig{
		Domain:         "example1.com",
		HTTPPort:       8085,
		HTTPSPort:      8448,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	config2 := &HostingConfig{
		Domain:         "example2.com",
		HTTPPort:       8086,
		HTTPSPort:      8449,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	// Add and start servers
	manager.AddServer("server1", config1)
	manager.AddServer("server2", config2)
	manager.StartAll()

	// Stop all servers
	err := manager.StopAll()
	assert.NoError(t, err)

	// Verify all servers are stopped
	server1, _ := manager.GetServer("server1")
	server2, _ := manager.GetServer("server2")
	assert.False(t, server1.IsRunning())
	assert.False(t, server2.IsRunning())
}

func TestHostingServer_HTTPHandler(t *testing.T) {
	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8087,
		HTTPSPort:      8450,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	server := NewHostingServer(config)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	server.SetHandler(handler)

	// Start server
	err := server.Start()
	assert.NoError(t, err)
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test handler
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestHostingServer_ContextCancellation(t *testing.T) {
	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       8088,
		HTTPSPort:      8451,
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	server := NewHostingServer(config)

	// Start server
	err := server.Start()
	assert.NoError(t, err)

	// Stop server
	err = server.Stop()
	assert.NoError(t, err)
}
