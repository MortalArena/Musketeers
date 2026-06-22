package email

import (
	"fmt"
	"sync"
	"time"
)

// EmailStore مخزن الإيميلات (في الذاكرة للتبسيط)
type EmailStore struct {
	inbox map[string][]*Email
	sent  map[string][]*Email
	mu    sync.RWMutex
}

// NewEmailStore ينشئ مخزن جديد
func NewEmailStore() (*EmailStore, error) {
	return &EmailStore{
		inbox: make(map[string][]*Email),
		sent:  make(map[string][]*Email),
	}, nil
}

// Close يغلق المخزن
func (s *EmailStore) Close() error {
	return nil
}

// SaveInbox يحفظ إيميل في صندوق الوارد
func (s *EmailStore) SaveInbox(email *Email) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inbox[email.To] = append(s.inbox[email.To], email)
	return nil
}

// SaveSent يحفظ إيميل في المرسلة
func (s *EmailStore) SaveSent(email *Email) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sent[email.From] = append(s.sent[email.From], email)
	return nil
}

// GetInbox يحصل على صندوق الوارد
func (s *EmailStore) GetInbox(userEmail string) ([]*Email, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	emails, exists := s.inbox[userEmail]
	if !exists {
		return []*Email{}, nil
	}
	return emails, nil
}

// GetSent يحصل على الرسائل المرسلة
func (s *EmailStore) GetSent(userEmail string) ([]*Email, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	emails, exists := s.sent[userEmail]
	if !exists {
		return []*Email{}, nil
	}
	return emails, nil
}

// MarkAsRead يحدد إيميل كمقروء
func (s *EmailStore) MarkAsRead(emailID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// البحث في inbox
	for _, emails := range s.inbox {
		for _, email := range emails {
			if email.ID == emailID {
				email.Read = true
				return nil
			}
		}
	}

	// البحث في sent
	for _, emails := range s.sent {
		for _, email := range emails {
			if email.ID == emailID {
				email.Read = true
				return nil
			}
		}
	}

	return fmt.Errorf("email not found: %s", emailID)
}

// Delete يحذف إيميل
func (s *EmailStore) Delete(emailID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// البحث في inbox وحذف
	for userEmail, emails := range s.inbox {
		for i, email := range emails {
			if email.ID == emailID {
				s.inbox[userEmail] = append(emails[:i], emails[i+1:]...)
				return nil
			}
		}
	}

	// البحث في sent وحذف
	for userEmail, emails := range s.sent {
		for i, email := range emails {
			if email.ID == emailID {
				s.sent[userEmail] = append(emails[:i], emails[i+1:]...)
				return nil
			}
		}
	}

	return fmt.Errorf("email not found: %s", emailID)
}

// CleanupOldEmails يحذف الإيميلات القديمة
func (s *EmailStore) CleanupOldEmails(maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	// تنظيف inbox
	for userEmail, emails := range s.inbox {
		var filtered []*Email
		for _, email := range emails {
			if email.Timestamp.After(cutoff) {
				filtered = append(filtered, email)
			}
		}
		s.inbox[userEmail] = filtered
	}

	// تنظيف sent
	for userEmail, emails := range s.sent {
		var filtered []*Email
		for _, email := range emails {
			if email.Timestamp.After(cutoff) {
				filtered = append(filtered, email)
			}
		}
		s.sent[userEmail] = filtered
	}

	return nil
}
