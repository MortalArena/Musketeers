package domain

import (
	"context"
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
)

// LocalDNSProxy وكيل DNS محلي لتوجيه طلبات DNS
type LocalDNSProxy struct {
	logger      *zap.Logger
	p2pResolver *P2PDNSResolver
	localCache  map[string]string // domain -> IP
	cacheMutex  sync.RWMutex
	listenAddr  string
	server      *net.UDPConn
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewLocalDNSProxy ينشئ وكيل DNS محلي جديد
func NewLocalDNSProxy(logger *zap.Logger, p2pResolver *P2PDNSResolver, listenAddr string) *LocalDNSProxy {
	ctx, cancel := context.WithCancel(context.Background())

	return &LocalDNSProxy{
		logger:      logger,
		p2pResolver: p2pResolver,
		localCache:  make(map[string]string),
		listenAddr:  listenAddr,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start يبدأ تشغيل وكيل DNS المحلي
func (p *LocalDNSProxy) Start() error {
	addr, err := net.ResolveUDPAddr("udp", p.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	p.server = conn
	p.logger.Info("بدء تشغيل وكيل DNS المحلي", zap.String("addr", p.listenAddr))

	go p.handleDNSRequests()

	return nil
}

// Stop يوقف تشغيل وكيل DNS المحلي
func (p *LocalDNSProxy) Stop() error {
	p.cancel()
	if p.server != nil {
		return p.server.Close()
	}
	return nil
}

// handleDNSRequests يعالج طلبات DNS
func (p *LocalDNSProxy) handleDNSRequests() {
	buf := make([]byte, 512)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			n, addr, err := p.server.ReadFromUDP(buf)
			if err != nil {
				continue
			}

			go p.processDNSRequest(buf[:n], addr)
		}
	}
}

// processDNSRequest يعالج طلب DNS واحد
func (p *LocalDNSProxy) processDNSRequest(data []byte, addr *net.UDPAddr) {
	// تحليل طلب DNS (تنفيذ مبسط)
	domain := string(data[:min(50, len(data))]) // استخراج اسم النطاق

	// البحث في الكاش المحلي
	p.cacheMutex.RLock()
	ip, found := p.localCache[domain]
	p.cacheMutex.RUnlock()

	if found {
		// إرسال الإجابة من الكاش
		p.sendDNSResponse(addr, ip)
		return
	}

	// البحث عبر P2P DNS Resolver
	if p.p2pResolver != nil {
		ip, err := p.p2pResolver.Resolve(domain)
		if err == nil && ip != "" {
			// تخزين في الكاش
			p.cacheMutex.Lock()
			p.localCache[domain] = ip
			p.cacheMutex.Unlock()

			p.sendDNSResponse(addr, ip)
			return
		}
	}

	// استخدام DNS الافتراضي
	ips, err := net.LookupHost(domain)
	if err == nil && len(ips) > 0 {
		p.cacheMutex.Lock()
		p.localCache[domain] = ips[0]
		p.cacheMutex.Unlock()

		p.sendDNSResponse(addr, ips[0])
	}
}

// sendDNSResponse يرسل إجابة DNS
func (p *LocalDNSProxy) sendDNSResponse(addr *net.UDPAddr, ip string) {
	// تنفيذ مبسط لإرسال إجابة DNS
	response := []byte(ip)
	p.server.WriteToUDP(response, addr)
}

// ClearCache يمسح كاش DNS المحلي
func (p *LocalDNSProxy) ClearCache() {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	p.localCache = make(map[string]string)
	p.logger.Info("تم مسح كاش DNS المحلي")
}

// GetCacheStats يحصل على إحصائيات الكاش
func (p *LocalDNSProxy) GetCacheStats() map[string]interface{} {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	return map[string]interface{}{
		"cached_domains": len(p.localCache),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
