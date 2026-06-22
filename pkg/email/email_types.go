package email

import "time"

// Email يمثل إيميل
type Email struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
	Encrypted bool      `json:"encrypted"`
	Attachments []string `json:"attachments,omitempty"`
}

// EmailFilter مرشح للإيميلات
type EmailFilter struct {
	From    string
	To      string
	Subject string
	Unread  bool
	Before  time.Time
	After   time.Time
}
