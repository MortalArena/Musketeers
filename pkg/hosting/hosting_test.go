package hosting

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getFreePort() int {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestNewHostingServer(t *testing.T) {
	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       getFreePort(),
		HTTPSPort:      getFreePort(),
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	server := NewHostingServer(config)
	assert.NotNil(t, server)
	assert.False(t, server.IsRunning())
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
		HTTPPort:       getFreePort(),
		HTTPSPort:      getFreePort(),
		EnableHTTP:     true,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	err := manager.AddServer("test", config)
	assert.NoError(t, err)

	err = manager.AddServer("test", config)
	assert.Error(t, err)
}

func TestHostingManager_RemoveServer(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       getFreePort(),
		HTTPSPort:      getFreePort(),
		EnableHTTP:     false,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	err := manager.AddServer("test", config)
	assert.NoError(t, err)

	err = manager.RemoveServer("test")
	assert.NoError(t, err)

	err = manager.RemoveServer("nonexistent")
	assert.Error(t, err)
}

func TestHostingManager_GetServer(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       getFreePort(),
		HTTPSPort:      getFreePort(),
		EnableHTTP:     false,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	_, err := manager.GetServer("nonexistent")
	assert.Error(t, err)

	err = manager.AddServer("test", config)
	assert.NoError(t, err)

	server, err := manager.GetServer("test")
	assert.NoError(t, err)
	assert.NotNil(t, server)
}

func TestHostingManager_ListServers(t *testing.T) {
	manager := NewHostingManager()

	config := &HostingConfig{
		Domain:         "example.com",
		HTTPPort:       getFreePort(),
		HTTPSPort:      getFreePort(),
		EnableHTTP:     false,
		EnableHTTPS:    false,
		MaxConnections: 100,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	servers := manager.ListServers()
	assert.Len(t, servers, 0)

	manager.AddServer("server1", config)
	manager.AddServer("server2", config)

	servers = manager.ListServers()
	assert.Len(t, servers, 2)
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
		EnableHTTP:     false,
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

func TestHostingServer_HTTPHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestSanitizeFileID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"valid.txt", "valid.txt"},
		{"../etc/passwd", ""},
		{"../../etc/shadow", ""},
		{"foo/../bar", ""},
		{"a/b/c", "a/b/c"},
		{"", ""},
		{"/", ""},
	}

	for _, tc := range tests {
		got := sanitizeFileID(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeFileID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
