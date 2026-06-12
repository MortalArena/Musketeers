package node

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pproto "github.com/libp2p/go-libp2p/core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	nrcrypto "github.com/neuroroot/core/pkg/crypto"
	"github.com/neuroroot/core/pkg/acp"
	"github.com/neuroroot/core/pkg/channel"
	"github.com/neuroroot/core/pkg/content"
	"github.com/neuroroot/core/pkg/identity"
	"github.com/neuroroot/core/pkg/naming"
	nrproto "github.com/neuroroot/core/pkg/protocol"
	"github.com/neuroroot/core/pkg/search"
	"github.com/sirupsen/logrus"
)

// Node عقدة NeuroRoot
type Node struct {
	cfg      *Config
	host     host.Host
	dht      *dht.IpfsDHT
	ps       *pubsub.PubSub
	keyPair  *nrcrypto.KeyPair
	identity *identity.IdentityRecord
	log      *logrus.Logger

	db           *badger.DB
	blockStore   content.BlockStore
	provider     *content.ProviderManager
	fetcher      *content.Fetcher
	nonceStore   *NonceStore
	crl          *identity.CRLCache
	rateLimiter  *search.TokenBucket
	validators   *DHTValidators
	founderPub   ed25519.PublicKey

	keyCache     map[string]ed25519.PublicKey
	keyCacheMu   sync.RWMutex
	chunkAsm     *ChunkAssembler
	acpRouter    *acp.Router
	acpTransport *acp.Transport
	topics       map[string]*pubsub.Topic
	topicsMu     sync.RWMutex
	cancel       context.CancelFunc
}

// New ينشئ عقدة جديدة
func New(ctx context.Context, cfg *Config, kp *nrcrypto.KeyPair, idRec *identity.IdentityRecord) (*Node, error) {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("فشل إنشاء مجلد البيانات: %w", err)
	}

	// BadgerDB
	opts := badger.DefaultOptions(cfg.DataDir + "/badger").
		WithLogger(nil).
		WithValueLogFileSize(16 * 1024 * 1024) // 16MB instead of 1GB to support low disk space
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("فشل فتح BadgerDB: %w", err)
	}

	// libp2p host
	privKey, err := crypto.UnmarshalEd25519PrivateKey(kp.Private)
	if err != nil {
		db.Close()
		return nil, err
	}

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.ListenPort)
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Muxer(yamux.ID, yamux.DefaultTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.NATPortMap(),
	)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("فشل إنشاء libp2p host: %w", err)
	}

	crl := identity.NewCRLCache(24 * time.Hour)

	var founderPub ed25519.PublicKey
	if cfg.FounderPubHex != "" {
		founderPub, err = nrcrypto.PubKeyFromHex(cfg.FounderPubHex)
		if err != nil {
			db.Close()
			h.Close()
			return nil, fmt.Errorf("مفتاح المؤسس غير صالح: %w", err)
		}
	}

	validators := NewDHTValidators(founderPub, crl)

	// DHT
	kad, err := dht.New(ctx, h,
		dht.Mode(dht.ModeAutoServer),
		dht.ProtocolPrefix("/neuroroot"),
		validators.ValidatorOption(),
	)
	if err != nil {
		db.Close()
		h.Close()
		return nil, fmt.Errorf("فشل إنشاء DHT: %w", err)
	}

	if err := kad.Bootstrap(ctx); err != nil {
		log.WithError(err).Warn("فشل bootstrap DHT جزئياً")
	}

	// Bootstrap peers
	for _, addr := range cfg.BootstrapPeers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}
		if err := h.Connect(ctx, *info); err != nil {
			log.WithError(err).WithField("peer", info.ID).Debug("فشل اتصال bootstrap")
		}
	}

	// PubSub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		db.Close()
		h.Close()
		return nil, fmt.Errorf("فشل إنشاء PubSub: %w", err)
	}

	blockStore := content.NewBadgerBlockStore(db, cfg.StorageQuotaMB)
	provider := content.NewProviderManager(h, kad, blockStore, log)
	fetcher := content.NewFetcher(h, provider, blockStore, log)
	nonceStore := NewNonceStore(db, time.Hour)
	rateLimiter := search.NewTokenBucket(cfg.MaxPutPerMinute, cfg.MaxPutPerMinute*2)

	ctx, cancel := context.WithCancel(ctx)

	n := &Node{
		cfg:         cfg,
		host:        h,
		dht:         kad,
		ps:          ps,
		keyPair:     kp,
		identity:    idRec,
		log:         log,
		db:          db,
		blockStore:  blockStore,
		provider:    provider,
		fetcher:     fetcher,
		nonceStore:  nonceStore,
		crl:         crl,
		rateLimiter: rateLimiter,
		validators:  validators,
		founderPub:  founderPub,
		keyCache:    make(map[string]ed25519.PublicKey),
		chunkAsm:    NewChunkAssembler(),
		topics:      make(map[string]*pubsub.Topic),
		cancel:      cancel,
	}

	// تسجيل بروتوكول Bitswap
	h.SetStreamHandler(libp2pproto.ID(nrproto.ProtocolBitswap), provider.ServeBitswap)

	// تسجيل بروتوكول المراسلة المباشرة
	h.SetStreamHandler(libp2pproto.ID(nrproto.ProtocolDirect), n.handleDirectStream)

	n.initACP()

	return n, nil
}

// Host يرجع libp2p host
func (n *Node) Host() host.Host { return n.host }

// DHT يرجع kad-dht
func (n *Node) DHT() *dht.IpfsDHT { return n.dht }

// KeyPair يرجع زوج المفاتيح
func (n *Node) KeyPair() *nrcrypto.KeyPair { return n.keyPair }

// Identity يرجع سجل الهوية
func (n *Node) Identity() *identity.IdentityRecord { return n.identity }

// Logger يرجع logger
func (n *Node) Logger() *logrus.Logger { return n.log }

// Fetcher يرجع content fetcher
func (n *Node) Fetcher() *content.Fetcher { return n.fetcher }

// Provider يرجع provider manager
func (n *Node) Provider() *content.ProviderManager { return n.provider }

// PublishIdentity ينشر سجل الهوية على DHT
func (n *Node) PublishIdentity(ctx context.Context) error {
	if !n.rateLimiter.Allow(n.host.ID().String()) {
		return fmt.Errorf("تجاوز حد المعدل")
	}
	data, err := json.Marshal(n.identity)
	if err != nil {
		return err
	}
	return n.dht.PutValue(ctx, n.identity.DHTKey(), data)
}

// PublishSearch ينشر إعلان بحث
func (n *Node) PublishSearch(ctx context.Context, keyword, meta string, ttl int64) error {
	if !n.rateLimiter.Allow(n.host.ID().String()) {
		return fmt.Errorf("تجاوز حد المعدل")
	}
	entry, err := search.NewIndexEntry(
		n.keyPair.DID,
		n.host.ID().String(),
		keyword, meta, ttl,
		n.keyPair.Private,
	)
	if err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return n.dht.PutValue(ctx, entry.DHTKey(), data)
}

// ResolveDomain يحل نطاق .ia
func (n *Node) ResolveDomain(ctx context.Context, name string) (*naming.DomainRecord, error) {
	normalized, err := naming.NormalizeDomainName(name)
	if err != nil {
		return nil, err
	}
	val, err := n.dht.GetValue(ctx, naming.DHTKey(normalized))
	if err != nil {
		return nil, fmt.Errorf("النطاق غير موجود: %w", err)
	}
	var rec naming.DomainRecord
	if err := json.Unmarshal(val, &rec); err != nil {
		return nil, err
	}
	ownerPub, err := n.ResolvePublicKey(rec.Owner)
	if err != nil {
		return nil, fmt.Errorf("فشل حل مفتاح المالك: %w", err)
	}
	if err := rec.Verify(n.founderPub, ownerPub); err != nil {
		return nil, fmt.Errorf("تحقق النطاق فشل: %w", err)
	}
	return &rec, nil
}

// JoinChannel ينضم لقناة عامة
func (n *Node) JoinChannel(ctx context.Context, channelID string) (*pubsub.Topic, *pubsub.Subscription, error) {
	n.topicsMu.Lock()
	defer n.topicsMu.Unlock()

	validator := channel.NewChannelMessageValidator(n, n.log)
	topicName := channel.TopicName(channelID)

	var topic *pubsub.Topic
	var err error
	if existing, ok := n.topics[topicName]; ok {
		topic = existing
	} else {
		topic, err = n.ps.Join(topicName)
		if err != nil {
			return nil, nil, err
		}
		n.topics[topicName] = topic

		// تسجيل validator
		n.ps.RegisterTopicValidator(topicName, func(ctx context.Context, peerID peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
			return validator.Validate(channelID, msg.Data)
		})
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, nil, err
	}
	return topic, sub, nil
}

// PublishChannelMessage ينشر رسالة في قناة
func (n *Node) PublishChannelMessage(ctx context.Context, channelID, content string) error {
	n.topicsMu.Lock()
	topicName := channel.TopicName(channelID)
	var topic *pubsub.Topic
	var err error
	if existing, ok := n.topics[topicName]; ok {
		topic = existing
	} else {
		topic, err = n.ps.Join(topicName)
		if err != nil {
			n.topicsMu.Unlock()
			return err
		}
		n.topics[topicName] = topic
	}
	n.topicsMu.Unlock()

	msg, err := channel.NewChannelMessage(n.keyPair.DID, content, channelID, n.keyPair.Private)
	if err != nil {
		return err
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return topic.Publish(ctx, data)
}

// ResolvePublicKey يجلب المفتاح العام من DID (KeyResolver)
func (n *Node) ResolvePublicKey(did string) (ed25519.PublicKey, error) {
	n.keyCacheMu.RLock()
	if pub, ok := n.keyCache[did]; ok {
		n.keyCacheMu.RUnlock()
		return pub, nil
	}
	n.keyCacheMu.RUnlock()

	if n.crl.IsRevoked(did) {
		return nil, fmt.Errorf("الهوية ملغاة: %s", did)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	val, err := n.dht.GetValue(ctx, "/nr/identity/"+did)
	if err != nil {
		return nil, fmt.Errorf("الهوية غير موجودة: %w", err)
	}
	var rec identity.IdentityRecord
	if err := json.Unmarshal(val, &rec); err != nil {
		return nil, err
	}
	if err := rec.Verify(); err != nil {
		return nil, err
	}
	pub, err := rec.PublicKey()
	if err != nil {
		return nil, err
	}

	n.keyCacheMu.Lock()
	if len(n.keyCache) >= 1000 {
		for k := range n.keyCache {
			delete(n.keyCache, k)
			if len(n.keyCache) < 900 {
				break
			}
		}
	}
	n.keyCache[did] = pub
	n.keyCacheMu.Unlock()
	return pub, nil
}

// IsRevoked يتحقق من إلغاء الهوية
func (n *Node) IsRevoked(ctx context.Context, did string) bool {
	if n.crl.IsRevoked(did) {
		return true
	}
	val, err := n.dht.GetValue(ctx, "/nr/revoke/"+did)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false
		}
		n.log.Warnf("فشل التحقق من إلغاء الهوية %s بسبب عطل بالشبكة: %v. تفعيل وضع الإغلاق الآمن.", did, err)
		return true // fail-closed
	}
	var rec identity.RevocationRecord
	if err := json.Unmarshal(val, &rec); err != nil {
		return false
	}
	pub, err := n.ResolvePublicKey(did)
	if err != nil {
		return true
	}
	if err := rec.Verify(pub); err != nil {
		return false
	}
	n.crl.MarkRevoked(did, rec.RevokedAt)
	return true
}

// RevokeIdentity يلغي الهوية
func (n *Node) RevokeIdentity(ctx context.Context) error {
	rec, err := identity.NewRevocationRecord(n.keyPair.DID, n.keyPair.Private)
	if err != nil {
		return err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return n.dht.PutValue(ctx, rec.DHTKey(), data)
}

// handleDirectStream يعالج المراسلة المباشرة
func (n *Node) handleDirectStream(s network.Stream) {
	defer s.Close()
	msg, err := ReadDirect(s)
	if err != nil {
		n.log.WithError(err).Debug("فشل قراءة رسالة مباشرة")
		return
	}

	// فحص replay
	seen, err := n.nonceStore.Seen(msg.Nonce)
	if err != nil || seen {
		n.log.Debug("nonce مكرر — رفض replay")
		return
	}

	senderPub, err := n.ResolvePublicKey(msg.From)
	if err != nil {
		return
	}

	plain, err := DecryptDirectMessage(msg, n.keyPair.Private, senderPub)
	if err != nil {
		n.log.WithError(err).Debug("فشل فك تشفير رسالة مباشرة")
		return
	}

	if msg.ChunkTotal > 1 {
		complete, done, err := n.chunkAsm.Add(msg, plain)
		if err != nil {
			n.log.WithError(err).Error("فشل تجميع أجزاء الملف")
			return
		}
		if done {
			n.log.WithFields(logrus.Fields{
				"from":    msg.From,
				"file_id": msg.FileID,
				"size":    len(complete),
			}).Info("تم استلام ملف مجزأ كامل")
		}
	} else {
		n.log.WithFields(logrus.Fields{
			"from": msg.From,
			"size": len(plain),
		}).Info("تم استلام رسالة مباشرة")
	}
}

// SendDirectMessage يرسل رسالة مباشرة
func (n *Node) SendDirectMessage(ctx context.Context, toDID string, content []byte) error {
	recipientPub, err := n.ResolvePublicKey(toDID)
	if err != nil {
		return err
	}

	msgs, err := ChunkMessage(n.keyPair.DID, toDID, content, n.keyPair.Private, recipientPub)
	if err != nil {
		return err
	}

	// نحتاج peer ID للمستقبل — من identity record أو search
	// مبسّط: نبحث في DHT عن peer
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// للتبسيط نستخدم البحث
	// في الإنتاج: resolve peer من IndexEntry

	for _, msg := range msgs {
		_ = msg
	}
	_ = ctx
	return fmt.Errorf("يتطلب peer ID للمستقبل — استخدم SendDirectToPeer")
}

// SendDirectToPeer يرسل رسالة مباشرة لـ peer محدد
func (n *Node) SendDirectToPeer(ctx context.Context, pid peer.ID, toDID string, content []byte) error {
	recipientPub, err := n.ResolvePublicKey(toDID)
	if err != nil {
		return err
	}

	msgs, err := ChunkMessage(n.keyPair.DID, toDID, content, n.keyPair.Private, recipientPub)
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		s, err := n.host.NewStream(ctx, pid, libp2pproto.ID(nrproto.ProtocolDirect))
		if err != nil {
			return err
		}
		if err := SendDirect(s, msg); err != nil {
			s.Close()
			return err
		}
		s.Close()
	}
	return nil
}

// PublishContent ينشر محتوى
func (n *Node) PublishContent(ctx context.Context, data []byte) (string, error) {
	return n.provider.PublishContent(ctx, data)
}

// FetchContent يجلب محتوى
func (n *Node) FetchContent(ctx context.Context, cid string) ([]byte, error) {
	return n.fetcher.FetchContent(ctx, cid)
}

// Close يغلق العقدة
func (n *Node) Close() error {
	n.cancel()
	if n.db != nil {
		n.db.Close()
	}
	if n.host != nil {
		return n.host.Close()
	}
	return nil
}

// Addrs يرجع عناوين العقدة
func (n *Node) Addrs() []string {
	var addrs []string
	for _, addr := range n.host.Addrs() {
		full := fmt.Sprintf("%s/p2p/%s", addr, n.host.ID())
		addrs = append(addrs, full)
	}
	return addrs
}
