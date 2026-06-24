package hosting

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// P2PHostingService خدمة استضافة المواقع عبر P2P
type P2PHostingService struct {
	p2pHost       host.Host
	sites         map[string]*Site
	mu            sync.RWMutex
}

// NewP2PHostingService ينشئ خدمة استضافة جديدة
func NewP2PHostingService(p2pHost host.Host) *P2PHostingService {
	service := &P2PHostingService{
		p2pHost: p2pHost,
		sites:   make(map[string]*Site),
	}

	// تسجيل handler لاستقبال طلبات المحتوى
	p2pHost.SetStreamHandler(protocol.ID("/musketeers/content/1.0.0"), service.handleContentRequest)

	return service
}

// DeploySite ينشر موقع
func (s *P2PHostingService) DeploySite(ctx context.Context, siteName string, files map[string][]byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// إنشاء كائن الموقع
	site := &Site{
		Name:      siteName,
		Files:     files,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// حفظ الموقع محلياً
	s.sites[siteName] = site

	// إعلان الموقع في الشبكة
	if err := s.announceSite(ctx, siteName); err != nil {
		return "", fmt.Errorf("failed to announce site: %w", err)
	}

	// إرجاع الرابط
	url := fmt.Sprintf("http://%s.musketeers", siteName)
	log.Printf("Site deployed: %s", url)

	return url, nil
}

// RemoveSite يحذف موقع
func (s *P2PHostingService) RemoveSite(siteName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sites[siteName]; !exists {
		return fmt.Errorf("site not found: %s", siteName)
	}

	delete(s.sites, siteName)
	log.Printf("Site removed: %s", siteName)

	return nil
}

// GetSite يحصل على موقع
func (s *P2PHostingService) GetSite(siteName string) (*Site, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	site, exists := s.sites[siteName]
	if !exists {
		return nil, fmt.Errorf("site not found: %s", siteName)
	}

	return site, nil
}

// ListSites يسرد كل المواقع
func (s *P2PHostingService) ListSites() []*Site {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sites []*Site
	for _, site := range s.sites {
		sites = append(sites, site)
	}

	return sites
}

// handleContentRequest يعالج طلبات المحتوى الواردة
func (s *P2PHostingService) handleContentRequest(stream network.Stream) {
	defer stream.Close()

	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
		return
	}

	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		log.Printf("Failed to read request: %v", err)
		return
	}

	request := string(buf[:n])
	path := extractPathFromRequest(request)
	siteName := extractSiteNameFromRequest(request)

	s.mu.RLock()
	site, exists := s.sites[siteName]
	s.mu.RUnlock()

	if !exists {
		_, _ = stream.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\nSite not found"))
		return
	}

	site.mu.RLock()
	content, exists := site.Files[path]
	if !exists {
		content, exists = site.Files["/index.html"]
	}
	site.mu.RUnlock()

	if !exists {
		_, _ = stream.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\nFile not found"))
		return
	}

	response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n", len(content))
	_, _ = stream.Write([]byte(response))
	_, _ = stream.Write(content)
}

// announceSite يعلن عن موقع في الشبكة
func (s *P2PHostingService) announceSite(ctx context.Context, siteName string) error {
	// في الإنتاج، يجب استخدام DHT للإعلان عن الموقع
	// هذا مثال مبسط

	log.Printf("Announcing site: %s", siteName)
	return nil
}

// extractPathFromRequest يستخرج المسار من طلب HTTP
func extractPathFromRequest(request string) string {
	for _, part := range strings.SplitN(request, " ", 3) {
		if strings.HasPrefix(part, "/") {
			path := strings.SplitN(part, "?", 2)[0]
			path = strings.TrimSuffix(path, "/")
			if path == "" {
				path = "/"
			}
			for _, c := range path {
				if c < 32 || c > 126 {
					return "/"
				}
			}
			if strings.Contains(path, "..") || strings.Contains(path, "//") {
				return "/"
			}
			return path
		}
	}
	return "/"
}

// extractSiteNameFromRequest يستخرج اسم الموقع من الطلب
func extractSiteNameFromRequest(request string) string {
	lines := strings.Split(request, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "host:") {
			host := strings.TrimSpace(line[5:])
			colonIdx := strings.LastIndex(host, ":")
			if colonIdx >= 0 {
				host = host[:colonIdx]
			}
			if strings.HasSuffix(host, ".musketeers") {
				name := strings.TrimSuffix(host, ".musketeers")
				if name != "" && !strings.ContainsAny(name, "./\\") {
					return name
				}
			}
		}
	}
	return ""
}
