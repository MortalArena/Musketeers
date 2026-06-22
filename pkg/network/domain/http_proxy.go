package domain

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// HTTPProxy يعترض طلبات HTTP ويجلب المحتوى من P2P
type HTTPProxy struct {
	p2pHost    host.Host
	resolver   *P2PDNSResolver
	server     *http.Server
	listenAddr string
}

// NewHTTPProxy ينشئ HTTP Proxy جديد
func NewHTTPProxy(p2pHost host.Host, listenAddr string) *HTTPProxy {
	proxy := &HTTPProxy{
		p2pHost:    p2pHost,
		resolver:   NewP2PDNSResolver(p2pHost),
		listenAddr: listenAddr,
	}

	proxy.server = &http.Server{
		Addr:    listenAddr,
		Handler: proxy,
	}

	return proxy
}

// Start يشغّل HTTP Proxy
func (hp *HTTPProxy) Start() error {
	log.Printf("HTTP Proxy started on %s", hp.listenAddr)
	return hp.server.ListenAndServe()
}

// Stop يوقف HTTP Proxy
func (hp *HTTPProxy) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return hp.server.Shutdown(ctx)
}

// ServeHTTP يعالج طلبات HTTP
func (hp *HTTPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// استخراج اسم النطاق من Host header
	domain := r.Host
	if strings.Contains(domain, ":") {
		domain = strings.Split(domain, ":")[0]
	}

	// إذا كان نطاق .musketeers أو .mskt
	if strings.HasSuffix(domain, ".musketeers") || strings.HasSuffix(domain, ".mskt") {
		hp.handleP2PRequest(w, r, domain)
		return
	}

	// إذا لم يكن نطاق P2P، مرر الطلب
	hp.handleRegularRequest(w, r)
}

// handleP2PRequest يعالج طلب لنطاق P2P
func (hp *HTTPProxy) handleP2PRequest(w http.ResponseWriter, r *http.Request, domain string) {
	// حل النطاق إلى Peer ID
	peerID, err := hp.resolver.ResolvePeerID(domain)
	if err != nil {
		http.Error(w, fmt.Sprintf("Site not found: %s", domain), http.StatusNotFound)
		log.Printf("Failed to resolve %s: %v", domain, err)
		return
	}

	// جلب المحتوى من P2P
	content, contentType, err := hp.fetchContentFromP2P(peerID, r.URL.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch content: %v", err), http.StatusInternalServerError)
		log.Printf("Failed to fetch content from %s: %v", peerID, err)
		return
	}

	// إرسال المحتوى
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// fetchContentFromP2P يجلب المحتوى من شبكة P2P
func (hp *HTTPProxy) fetchContentFromP2P(peerID peer.ID, path string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// فتح stream إلى peer
	stream, err := hp.p2pHost.NewStream(ctx, peerID, "/musketeers/content/1.0.0")
	if err != nil {
		return nil, "", fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// إرسال طلب المحتوى
	request := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\n\r\n", path, peerID)
	_, err = stream.Write([]byte(request))
	if err != nil {
		return nil, "", fmt.Errorf("failed to send request: %w", err)
	}

	// قراءة الاستجابة
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, err := stream.Read(buf)
	if err != nil && err != io.EOF {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	// تحليل الاستجابة (مبسط)
	content := buf[:n]
	contentType := "text/html" // افتراضي

	// في الإنتاج، يجب تحليل HTTP Response بشكل صحيح
	// هذا مثال مبسط

	return content, contentType, nil
}

// handleRegularRequest يعالج طلب عادي (ليس P2P)
func (hp *HTTPProxy) handleRegularRequest(w http.ResponseWriter, r *http.Request) {
	// إنشاء client جديد
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// إعادة توجيه الطلب
	url := fmt.Sprintf("http://%s%s", r.Host, r.URL.Path)
	proxyReq, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// نسخ Headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// إرسال الطلب
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to fetch resource", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// نسخ Response Headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// نسخ Status Code
	w.WriteHeader(resp.StatusCode)

	// نسخ Body
	io.Copy(w, resp.Body)
}
