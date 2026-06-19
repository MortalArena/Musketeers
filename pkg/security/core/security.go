package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SecurityManager مدير الأمان
type SecurityManager struct {
	users         map[string]*UserSecurity
	sessions      map[string]*SessionSecurity
	apiKeys       map[string]*APIKeySecurity
	rateLimits    map[string]*RateLimit
	logger        *zap.Logger
	mu            sync.RWMutex
	encryptionKey []byte
	eventBus      EventBus
}

// EventBus واجهة ناقل الأحداث
type EventBus interface {
	Publish(event string, data interface{}) error
	Subscribe(event string, handler func(data interface{})) error
}

// UserSecurity أمان المستخدم
type UserSecurity struct {
	UserID           string                 `json:"user_id"`
	PasswordHash     string                 `json:"password_hash"`
	TwoFactorEnabled bool                   `json:"two_factor_enabled"`
	TwoFactorSecret  string                 `json:"two_factor_secret"`
	Permissions      []string               `json:"permissions"`
	Roles            []string               `json:"roles"`
	LastLogin        time.Time              `json:"last_login"`
	FailedAttempts   int                    `json:"failed_attempts"`
	LockedUntil      time.Time              `json:"locked_until"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// SessionSecurity أمان الجلسة
type SessionSecurity struct {
	SessionID    string                 `json:"session_id"`
	UserID       string                 `json:"user_id"`
	Token        string                 `json:"token"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	ExpiresAt    time.Time              `json:"expires_at"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	IsValid      bool                   `json:"is_valid"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// APIKeySecurity أمان مفتاح API
type APIKeySecurity struct {
	KeyID       string                 `json:"key_id"`
	KeyHash     string                 `json:"key_hash"`
	UserID      string                 `json:"user_id"`
	Name        string                 `json:"name"`
	Permissions []string               `json:"permissions"`
	RateLimit   int                    `json:"rate_limit"`
	ExpiresAt   time.Time              `json:"expires_at"`
	CreatedAt   time.Time              `json:"created_at"`
	LastUsed    time.Time              `json:"last_used"`
	IsActive    bool                   `json:"is_active"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RateLimit حد المعدل
type RateLimit struct {
	Key          string        `json:"key"`
	Requests     int           `json:"requests"`
	Window       time.Duration `json:"window"`
	LastReset    time.Time     `json:"last_reset"`
	BlockedUntil time.Time     `json:"blocked_until"`
}

// SecurityEvent حدث أمان
type SecurityEvent struct {
	ID          string                 `json:"id"`
	Type        SecurityEventType      `json:"type"`
	Severity    SecuritySeverity       `json:"severity"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityEventType نوع حدث الأمان
type SecurityEventType string

const (
	SecurityEventTypeLogin        SecurityEventType = "login"
	SecurityEventTypeLogout       SecurityEventType = "logout"
	SecurityEventTypeAuthFailed   SecurityEventType = "auth_failed"
	SecurityEventTypeAuthSuccess  SecurityEventType = "auth_success"
	SecurityEventTypeRateLimit    SecurityEventType = "rate_limit"
	SecurityEventTypeSuspicious   SecurityEventType = "suspicious"
	SecurityEventTypeDataAccess   SecurityEventType = "data_access"
	SecurityEventTypeDataModified SecurityEventType = "data_modified"
)

// SecuritySeverity خطورة حدث الأمان
type SecuritySeverity string

const (
	SecuritySeverityLow      SecuritySeverity = "low"
	SecuritySeverityMedium   SecuritySeverity = "medium"
	SecuritySeverityHigh     SecuritySeverity = "high"
	SecuritySeverityCritical SecuritySeverity = "critical"
)

// NewSecurityManager ينشئ مدير أمان جديد
func NewSecurityManager(logger *zap.Logger, encryptionKey string, eventBus EventBus) (*SecurityManager, error) {
	key, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key: %w", err)
	}

	return &SecurityManager{
		users:         make(map[string]*UserSecurity),
		sessions:      make(map[string]*SessionSecurity),
		apiKeys:       make(map[string]*APIKeySecurity),
		rateLimits:    make(map[string]*RateLimit),
		logger:        logger,
		encryptionKey: key,
		eventBus:      eventBus,
	}, nil
}

// RegisterUser يسجل مستخدم جديد
func (sm *SecurityManager) RegisterUser(userID, password string, permissions, roles []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.users[userID]; exists {
		return fmt.Errorf("user already registered: %s", userID)
	}

	// تشفير كلمة المرور
	passwordHash, err := sm.hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// توليد سر 2FA
	twoFactorSecret, err := sm.generateTwoFactorSecret()
	if err != nil {
		return fmt.Errorf("failed to generate 2FA secret: %w", err)
	}

	sm.users[userID] = &UserSecurity{
		UserID:           userID,
		PasswordHash:     passwordHash,
		TwoFactorEnabled: false,
		TwoFactorSecret:  twoFactorSecret,
		Permissions:      permissions,
		Roles:            roles,
		LastLogin:        time.Time{},
		FailedAttempts:   0,
		LockedUntil:      time.Time{},
		Metadata:         make(map[string]interface{}),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	sm.logger.Info("تم تسجيل مستخدم جديد",
		zap.String("user_id", userID))

	return nil
}

// AuthenticateUser يصادق المستخدم
func (sm *SecurityManager) AuthenticateUser(userID, password string) (*SessionSecurity, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	user, exists := sm.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	// التحقق من الحساب المقفل
	if time.Now().Before(user.LockedUntil) {
		return nil, fmt.Errorf("account locked until: %s", user.LockedUntil)
	}

	// التحقق من كلمة المرور
	passwordHash, err := sm.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if passwordHash != user.PasswordHash {
		user.FailedAttempts++
		if user.FailedAttempts >= 5 {
			user.LockedUntil = time.Now().Add(30 * time.Minute)
			sm.logger.Warn("تم قفل الحساب بسبب محاولات فاشلة متعددة",
				zap.String("user_id", userID))
		}
		return nil, fmt.Errorf("invalid password")
	}

	// إعادة تعيين المحاولات الفاشلة
	user.FailedAttempts = 0
	user.LastLogin = time.Now()

	// إنشاء جلسة جديدة
	sessionID, err := sm.generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	token, err := sm.generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	session := &SessionSecurity{
		SessionID:    sessionID,
		UserID:       userID,
		Token:        token,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		IsValid:      true,
		Metadata:     make(map[string]interface{}),
	}

	sm.sessions[sessionID] = session

	sm.logger.Info("تم مصادقة المستخدم بنجاح",
		zap.String("user_id", userID),
		zap.String("session_id", sessionID))

	// نشر حدث المصادقة الناجحة
	if sm.eventBus != nil {
		sm.eventBus.Publish("security.auth_success", map[string]interface{}{
			"user_id":    userID,
			"session_id": sessionID,
		})
	}

	return session, nil
}

// ValidateSession يتحقق من صحة الجلسة
func (sm *SecurityManager) ValidateSession(sessionID, token string) (*SessionSecurity, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// التحقق من صحة الجلسة
	if !session.IsValid {
		return nil, fmt.Errorf("session is invalid")
	}

	// التحقق من انتهاء الصلاحية
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// التحقق من التوكن
	if session.Token != token {
		return nil, fmt.Errorf("invalid token")
	}

	return session, nil
}

// InvalidateSession يبطل الجلسة
func (sm *SecurityManager) InvalidateSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.IsValid = false

	sm.logger.Info("تم إبطال الجلسة",
		zap.String("session_id", sessionID),
		zap.String("user_id", session.UserID))

	return nil
}

// RegisterAPIKey يسجل مفتاح API جديد
func (sm *SecurityManager) RegisterAPIKey(userID, name string, permissions []string, rateLimit int, expiresAt time.Time) (*APIKeySecurity, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key, err := sm.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	keyHash, err := sm.hashAPIKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	apiKey := &APIKeySecurity{
		KeyID:       fmt.Sprintf("key_%d", time.Now().UnixNano()),
		KeyHash:     keyHash,
		UserID:      userID,
		Name:        name,
		Permissions: permissions,
		RateLimit:   rateLimit,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		LastUsed:    time.Time{},
		IsActive:    true,
		Metadata:    make(map[string]interface{}),
	}

	sm.apiKeys[apiKey.KeyID] = apiKey

	sm.logger.Info("تم تسجيل مفتاح API جديد",
		zap.String("key_id", apiKey.KeyID),
		zap.String("user_id", userID),
		zap.String("name", name))

	return apiKey, nil
}

// ValidateAPIKey يتحقق من صحة مفتاح API
func (sm *SecurityManager) ValidateAPIKey(key string) (*APIKeySecurity, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	keyHash, err := sm.hashAPIKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	for _, apiKey := range sm.apiKeys {
		if apiKey.KeyHash == keyHash {
			// التحقق من النشاط
			if !apiKey.IsActive {
				return nil, fmt.Errorf("API key is inactive")
			}

			// التحقق من انتهاء الصلاحية
			if !apiKey.ExpiresAt.IsZero() && time.Now().After(apiKey.ExpiresAt) {
				return nil, fmt.Errorf("API key expired")
			}

			return apiKey, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// CheckRateLimit يتحقق من حد المعدل
func (sm *SecurityManager) CheckRateLimit(key string, limit int, window time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	rateLimit, exists := sm.rateLimits[key]
	if !exists {
		sm.rateLimits[key] = &RateLimit{
			Key:       key,
			Requests:  0,
			Window:    window,
			LastReset: time.Now(),
		}
		return nil
	}

	// التحقق من الحظر
	if time.Now().Before(rateLimit.BlockedUntil) {
		return fmt.Errorf("rate limit blocked until: %s", rateLimit.BlockedUntil)
	}

	// إعادة تعيين النافذة
	if time.Since(rateLimit.LastReset) > rateLimit.Window {
		rateLimit.Requests = 0
		rateLimit.LastReset = time.Now()
	}

	// التحقق من الحد
	if rateLimit.Requests >= limit {
		rateLimit.BlockedUntil = time.Now().Add(window)
		sm.logger.Warn("تم الوصول إلى حد المعدل",
			zap.String("key", key))
		return fmt.Errorf("rate limit exceeded")
	}

	rateLimit.Requests++
	return nil
}

// LogSecurityEvent يسجل حدث أمان
func (sm *SecurityManager) LogSecurityEvent(event *SecurityEvent) error {
	event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	event.Timestamp = time.Now()

	sm.logger.Info("حدث أمان",
		zap.String("event_type", string(event.Type)),
		zap.String("severity", string(event.Severity)),
		zap.String("description", event.Description))

	// نشر حدث الأمان
	if sm.eventBus != nil {
		sm.eventBus.Publish("security.event", event)
	}

	return nil
}

// hashPassword يشفير كلمة المرور
func (sm *SecurityManager) hashPassword(password string) (string, error) {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:]), nil
}

// hashAPIKey يشفير مفتاح API
func (sm *SecurityManager) hashAPIKey(key string) (string, error) {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:]), nil
}

// generateSessionID يولد معرف الجلسة
func (sm *SecurityManager) generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// generateToken يولد التوكن
func (sm *SecurityManager) generateToken() (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// generateAPIKey يولد مفتاح API
func (sm *SecurityManager) generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk_" + base64.URLEncoding.EncodeToString(b), nil
}

// generateTwoFactorSecret يولد سر 2FA
func (sm *SecurityManager) generateTwoFactorSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(b), nil
}
