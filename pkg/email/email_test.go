package email

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEmailClient(t *testing.T) {
	config := &EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "user",
		SMTPPassword: "pass",
		UseTLS:      false,
		FromAddress:  "from@example.com",
		FromName:     "Test",
	}

	client := NewEmailClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, config, client.config)
}

func TestEmailClient_Validate(t *testing.T) {
	config := &EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "user",
		SMTPPassword: "pass",
		UseTLS:      false,
		FromAddress:  "from@example.com",
		FromName:     "Test",
	}

	client := NewEmailClient(config)

	tests := []struct {
		name    string
		msg     *EmailMessage
		wantErr bool
	}{
		{
			name: "valid email",
			msg: &EmailMessage{
				From:    "from@example.com",
				To:      []string{"to@example.com"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			wantErr: false,
		},
		{
			name: "missing from",
			msg: &EmailMessage{
				To:      []string{"to@example.com"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			wantErr: true,
		},
		{
			name: "missing to",
			msg: &EmailMessage{
				From:    "from@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			wantErr: true,
		},
		{
			name: "missing subject",
			msg: &EmailMessage{
				From: "from@example.com",
				To:   []string{"to@example.com"},
				Body: "Test Body",
			},
			wantErr: true,
		},
		{
			name: "missing body",
			msg: &EmailMessage{
				From:    "from@example.com",
				To:      []string{"to@example.com"},
				Subject: "Test Subject",
			},
			wantErr: true,
		},
		{
			name: "invalid email address",
			msg: &EmailMessage{
				From:    "from@example.com",
				To:      []string{"invalid-email"},
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Validate(tt.msg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailClient_SendAsync(t *testing.T) {
	config := &EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "user",
		SMTPPassword: "pass",
		UseTLS:      false,
		FromAddress:  "from@example.com",
		FromName:     "Test",
	}

	client := NewEmailClient(config)

	msg := &EmailMessage{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	callbackCalled := false
	callback := func(err error) {
		callbackCalled = true
	}

	client.SendAsync(msg, callback)

	// Wait for async operation
	time.Sleep(100 * time.Millisecond)

	// Note: This test doesn't actually send email, it just verifies async behavior
	assert.True(t, callbackCalled)
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"invalid-email", false},
		{"", false},
		{"@example.com", false},
		{"test@", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestGenerateBoundary(t *testing.T) {
	boundary1 := generateBoundary()
	boundary2 := generateBoundary()

	assert.NotEmpty(t, boundary1)
	assert.NotEmpty(t, boundary2)
	assert.NotEqual(t, boundary1, boundary2)
}

func TestAttachment(t *testing.T) {
	attachment := &Attachment{
		Filename: "test.txt",
		Content:  []byte("test content"),
		MimeType: "text/plain",
	}

	assert.Equal(t, "test.txt", attachment.Filename)
	assert.Equal(t, []byte("test content"), attachment.Content)
	assert.Equal(t, "text/plain", attachment.MimeType)
}

func TestEmailMessage(t *testing.T) {
	msg := &EmailMessage{
		From:     "from@example.com",
		To:       []string{"to1@example.com", "to2@example.com"},
		CC:       []string{"cc@example.com"},
		BCC:      []string{"bcc@example.com"},
		Subject:  "Test Subject",
		Body:     "Test Body",
		Priority: "high",
		Headers:  map[string]string{"X-Custom": "value"},
	}

	assert.Equal(t, "from@example.com", msg.From)
	assert.Len(t, msg.To, 2)
	assert.Len(t, msg.CC, 1)
	assert.Len(t, msg.BCC, 1)
	assert.Equal(t, "Test Subject", msg.Subject)
	assert.Equal(t, "Test Body", msg.Body)
	assert.Equal(t, "high", msg.Priority)
	assert.Equal(t, "value", msg.Headers["X-Custom"])
}
