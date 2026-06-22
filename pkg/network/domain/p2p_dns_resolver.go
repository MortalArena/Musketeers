package domain

import (
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// P2PDNSResolver يحل النطاقات عبر P2P DHT
type P2PDNSResolver struct {
	p2pHost   host.Host
	nameCache map[string]*NameResolution
	cacheMu   sync.RWMutex
}

// NameResolution نتيجة حل اسم
type NameResolution struct {
	IP        string
	PeerID    peer.ID
	ExpiresAt time.Time
}

// NewP2PDNSResolver ينشئ resolver جديد
func NewP2PDNSResolver(p2pHost host.Host) *P2PDNSResolver {
	return &P2PDNSResolver{
		p2pHost:   p2pHost,
		nameCache: make(map[string]*NameResolution),
	}
}

// Resolve يحل اسم النطاق إلى IP وهمي
func (r *P2PDNSResolver) Resolve(domain string) (string, error) {
	// التحقق من الكاش
	r.cacheMu.RLock()
	if resolution, exists := r.nameCache[domain]; exists {
		if time.Now().Before(resolution.ExpiresAt) {
			r.cacheMu.RUnlock()
			return resolution.IP, nil
		}
	}
	r.cacheMu.RUnlock()

	// البحث في DHT عن Peer ID الخاص بالنطاق
	peerID, err := r.findPeerByName(domain)
	if err != nil {
		return "", fmt.Errorf("failed to find peer for %s: %w", domain, err)
	}

	// الحصول على عنوان IP من Peer Info
	_, err = r.getPeerAddresses(peerID)
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for %s: %w", peerID, err)
	}

	// استخدام أول عنوان (أو إنشاء IP وهمي)
	ip := r.generateFakeIP(domain)

	// حفظ في الكاش
	r.cacheMu.Lock()
	r.nameCache[domain] = &NameResolution{
		IP:        ip,
		PeerID:    peerID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	r.cacheMu.Unlock()

	return ip, nil
}

// findPeerByName يبحث في DHT عن Peer ID
func (r *P2PDNSResolver) findPeerByName(domain string) (peer.ID, error) {
	// البحث في DHT عن المفتاح (اسم النطاق)
	providers := r.p2pHost.Network().Peerstore().Peers()

	// البحث عن peer الذي سجل هذا الاسم
	for _, p := range providers {
		// التحقق من سجلات peer
		name, err := r.p2pHost.Peerstore().Get(p, "name")
		if err == nil && name == domain {
			return p, nil
		}
	}

	return "", fmt.Errorf("peer not found for domain: %s", domain)
}

// getPeerAddresses يحصل على عناوين peer
func (r *P2PDNSResolver) getPeerAddresses(peerID peer.ID) ([]multiaddr.Multiaddr, error) {
	addrs := r.p2pHost.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses found for peer: %s", peerID)
	}
	return addrs, nil
}

// generateFakeIP يولد IP وهمي بناءً على اسم النطاق
func (r *P2PDNSResolver) generateFakeIP(domain string) string {
	// توليد IP وهمي في نطاق 10.0.0.0/8
	// هذا IP لا يستخدم فعلياً، فقط للتوجيه عبر Proxy
	hash := 0
	for _, c := range domain {
		hash = (hash*31 + int(c)) % 256
	}
	return fmt.Sprintf("10.0.%d.%d", hash/256, hash%256)
}

// RegisterName يسجل اسم نطاق في DHT
func (r *P2PDNSResolver) RegisterName(domain string) error {
	// تسجيل الاسم في Peerstore
	r.p2pHost.Peerstore().Put(r.p2pHost.ID(), "name", domain)

	// إعلان الاسم في الشبكة
	// يمكن استخدام DHT هنا للإعلان عن الاسم
	// هذا مبسط، في الإنتاج يجب استخدام DHT كامل

	return nil
}

// ResolvePeerID يحل اسم النطاق إلى Peer ID
func (r *P2PDNSResolver) ResolvePeerID(domain string) (peer.ID, error) {
	r.cacheMu.RLock()
	if resolution, exists := r.nameCache[domain]; exists {
		if time.Now().Before(resolution.ExpiresAt) {
			r.cacheMu.RUnlock()
			return resolution.PeerID, nil
		}
	}
	r.cacheMu.RUnlock()

	peerID, err := r.findPeerByName(domain)
	if err != nil {
		return "", err
	}

	return peerID, nil
}
