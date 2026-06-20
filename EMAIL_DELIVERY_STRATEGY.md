# Email Delivery Strategy - Musketeers Project

## Overview
This document outlines the email delivery strategy for the Musketeers project, including anti-spam measures, delivery optimization, and compliance with email standards.

## Email Delivery Architecture

### Components
1. **EmailClient** (`pkg/email/email.go`) - SMTP client for sending emails
2. **EmailManager** (`pkg/orchestrator/email_system.go`) - Email management system
3. **EmailIntegrator** (`pkg/email/integration.go`) - Integration between systems

### Delivery Flow
```
User/Agent → EmailManager → EmailIntegrator → EmailClient → SMTP Server → Recipient
```

## Anti-Spam Strategy

### 1. Sender Authentication

#### SPF (Sender Policy Framework)
**Implementation**: Add SPF records to DNS

```txt
v=spf1 include:_spf.google.com ~all
```

**Purpose**: Verify that the sender is authorized to send email from the domain

**Configuration**:
- Add SPF record to domain DNS
- Include all authorized mail servers
- Use ~all (soft fail) or -all (hard fail) based on policy

#### DKIM (DomainKeys Identified Mail)
**Implementation**: Sign all outgoing emails

```go
// Add DKIM signature to email headers
dkimSignature := fmt.Sprintf(
    "v=1; a=rsa-sha256; c=relaxed/relaxed; d=%s; s=%s; h=from:to:subject:date; b=%s",
    domain,
    selector,
    base64Signature,
)
```

**Purpose**: Verify that the email was not modified in transit

**Configuration**:
- Generate DKIM key pair
- Publish public key in DNS
- Sign all outgoing emails
- Verify incoming DKIM signatures

#### DMARC (Domain-based Message Authentication, Reporting, and Conformance)
**Implementation**: Add DMARC record to DNS

```txt
v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com; ruf=mailto:dmarc@example.com
```

**Purpose**: Specify how to handle emails that fail SPF/DKIM checks

**Configuration**:
- Start with p=none (monitoring mode)
- Move to p=quarantine after monitoring
- Eventually use p=reject for strict policy
- Set up rua and ruf for reports

### 2. Rate Limiting

#### Per-User Rate Limits
**Implementation**: Limit emails per user per time period

```go
type RateLimiter struct {
    mu       sync.Mutex
    limits   map[string]*UserLimit
}

type UserLimit struct {
    EmailsSent     int
    LastReset      time.Time
    DailyLimit     int
    HourlyLimit    int
}

func (rl *RateLimiter) CheckLimit(userID string) (bool, error) {
    // Check hourly and daily limits
    // Return false if limit exceeded
}
```

**Configuration**:
- Free tier: 100 emails/day, 10 emails/hour
- Basic tier: 1,000 emails/day, 100 emails/hour
- Pro tier: 10,000 emails/day, 1,000 emails/hour
- Enterprise: Unlimited with negotiation

#### Per-Recipient Rate Limits
**Implementation**: Limit emails to same recipient

```go
type RecipientRateLimiter struct {
    mu         sync.Mutex
    recipients map[string]*RecipientLimit
}

type RecipientLimit struct {
    EmailsSent int
    LastReset  time.Time
    DailyLimit int
}

func (rrl *RecipientRateLimiter) CheckLimit(recipient string) (bool, error) {
    // Check daily limit per recipient
    // Return false if limit exceeded
}
```

**Configuration**:
- Free tier: 10 emails/day to same recipient
- Basic tier: 50 emails/day to same recipient
- Pro tier: 200 emails/day to same recipient
- Enterprise: 1,000 emails/day to same recipient

### 3. Content Filtering

#### Spam Score Calculation
**Implementation**: Calculate spam score based on content

```go
type SpamFilter struct {
    rules []SpamRule
}

type SpamRule struct {
    Name     string
    Pattern  string
    Weight   int
    Category string
}

func (sf *SpamFilter) CalculateScore(email *Email) int {
    score := 0
    for _, rule := range sf.rules {
        if matchesPattern(email, rule.Pattern) {
            score += rule.Weight
        }
    }
    return score
}
```

**Spam Indicators**:
- Excessive use of capital letters
- Suspicious keywords (viagra, casino, etc.)
- Misleading subject lines
- Excessive punctuation (!!!, ???)
- Hidden text or white-on-white text
- Suspicious URLs
- Missing or invalid SPF/DKIM

**Configuration**:
- Score < 5: Allow delivery
- Score 5-10: Quarantine
- Score > 10: Reject

#### Content Validation
**Implementation**: Validate email content before sending

```go
func ValidateContent(email *Email) error {
    // Check for spam indicators
    // Validate URLs
    // Check for malicious content
    // Validate attachments
    return nil
}
```

**Validation Rules**:
- Subject line length: 1-78 characters
- Body text length: 1-100,000 characters
- Attachment size: < 25MB
- Attachment types: Whitelist only
- URL validation: Check against blacklist

### 4. Reputation Management

#### Sender Reputation Score
**Implementation**: Track sender reputation

```go
type ReputationManager struct {
    mu          sync.RWMutex
    reputations map[string]*Reputation
}

type Reputation struct {
    Score         float64
    EmailsSent    int
    EmailsDelivered int
    Complaints    int
    Bounces       int
    LastUpdated  time.Time
}

func (rm *ReputationManager) UpdateScore(userID string, event ReputationEvent) {
    // Update reputation based on event
    // Events: delivery, bounce, complaint, spam report
}
```

**Reputation Factors**:
- Delivery rate: 90%+ = good, < 90% = poor
- Bounce rate: < 2% = good, > 5% = poor
- Complaint rate: < 0.1% = good, > 0.5% = poor
- Spam report rate: < 0.01% = good, > 0.1% = poor

**Actions Based on Reputation**:
- Score > 80: Full delivery
- Score 60-80: Throttled delivery
- Score 40-60: Quarantine
- Score < 40: Reject

#### IP Reputation
**Implementation**: Monitor IP reputation

```go
type IPReputationChecker struct {
    blacklists []string
}

func (irc *IPReputationChecker) CheckIP(ip string) (bool, error) {
    // Check against DNS blacklists
    // Return true if IP is clean
}
```

**Blacklists to Check**:
- Spamhaus ZEN
- SpamCop
- Barracuda
- Spamhaus XBL
- Spamhaus PBL

### 5. Bounce Management

#### Bounce Classification
**Implementation**: Classify bounce types

```go
type BounceClassifier struct {
    patterns map[string]BounceType
}

type BounceType string

const (
    BounceHard    BounceType = "hard"
    BounceSoft    BounceType = "soft"
    BounceSpam    BounceType = "spam"
    BounceUnknown BounceType = "unknown"
)

func (bc *BounceClassifier) Classify(bounceMessage string) BounceType {
    // Classify based on bounce message
}
```

**Bounce Types**:
- Hard bounce: Invalid email address, domain doesn't exist
- Soft bounce: Mailbox full, temporary server issue
- Spam bounce: Marked as spam by recipient
- Unknown bounce: Unable to classify

**Actions Based on Bounce Type**:
- Hard bounce: Remove from list immediately
- Soft bounce: Retry 3 times, then remove
- Spam bounce: Remove immediately, flag sender
- Unknown bounce: Retry 3 times, then remove

#### Bounce Handling
**Implementation**: Process bounces automatically

```go
type BounceHandler struct {
    classifier *BounceClassifier
    reputation *ReputationManager
}

func (bh *BounceHandler) HandleBounce(emailID string, bounceMessage string) error {
    bounceType := bh.classifier.Classify(bounceMessage)
    
    switch bounceType {
    case BounceHard:
        bh.removeRecipient(emailID)
    case BounceSoft:
        bh.retryEmail(emailID)
    case BounceSpam:
        bh.removeRecipient(emailID)
        bh.flagSender(emailID)
    }
    
    return nil
}
```

### 6. Feedback Loop

#### Spam Complaints
**Implementation**: Handle spam complaints

```go
type ComplaintHandler struct {
    reputation *ReputationManager
}

func (ch *ComplaintHandler) HandleComplaint(emailID string, recipient string) error {
    // Remove recipient from list
    // Update sender reputation
    // Log complaint for analysis
    return nil
}
```

**Complaint Sources**:
- Feedback Loop (FBL) from ISPs
- Spam reports from recipients
- Abuse reports from recipients

**Actions**:
- Remove complaining recipient
- Update sender reputation
- Investigate pattern of complaints
- Take action on repeat offenders

#### Unsubscribe Handling
**Implementation**: Process unsubscribe requests

```go
type UnsubscribeHandler struct {
    listManager *ListManager
}

func (uh *UnsubscribeHandler) HandleUnsubscribe(listID string, recipient string) error {
    // Remove recipient from list
    // Confirm unsubscribe
    // Log unsubscribe for analysis
    return nil
}
```

**Requirements**:
- One-click unsubscribe link in all emails
- Unsubscribe must be processed within 10 business days
- Confirmation email sent after unsubscribe
- Unsubscribe preference honored for 10 days

## Delivery Optimization

### 1. Queue Management

#### Priority Queue
**Implementation**: Prioritize important emails

```go
type EmailQueue struct {
    highPriority   []*Email
    normalPriority []*Email
    lowPriority    []*Email
    mu             sync.Mutex
}

func (eq *EmailQueue) Enqueue(email *Email, priority string) {
    eq.mu.Lock()
    defer eq.mu.Unlock()
    
    switch priority {
    case "high":
        eq.highPriority = append(eq.highPriority, email)
    case "normal":
        eq.normalPriority = append(eq.normalPriority, email)
    case "low":
        eq.lowPriority = append(eq.lowPriority, email)
    }
}

func (eq *EmailQueue) Dequeue() *Email {
    eq.mu.Lock()
    defer eq.mu.Unlock()
    
    if len(eq.highPriority) > 0 {
        email := eq.highPriority[0]
        eq.highPriority = eq.highPriority[1:]
        return email
    }
    
    if len(eq.normalPriority) > 0 {
        email := eq.normalPriority[0]
        eq.normalPriority = eq.normalPriority[1:]
        return email
    }
    
    if len(eq.lowPriority) > 0 {
        email := eq.lowPriority[0]
        eq.lowPriority = eq.lowPriority[1:]
        return email
    }
    
    return nil
}
```

**Priority Levels**:
- High: System notifications, password resets
- Normal: Regular emails, newsletters
- Low: Marketing emails, promotional content

#### Retry Logic
**Implementation**: Retry failed deliveries

```go
type RetryManager struct {
    attempts map[string]int
    maxAttempts int
    backoff    time.Duration
}

func (rm *RetryManager) ShouldRetry(emailID string) bool {
    rm.mu.Lock()
    defer rm.mu.Unlock()
    
    attempts := rm.attempts[emailID]
    if attempts >= rm.maxAttempts {
        return false
    }
    
    rm.attempts[emailID]++
    return true
}

func (rm *RetryManager) GetBackoff(attempts int) time.Duration {
    return rm.backoff * time.Duration(attempts)
}
```

**Retry Strategy**:
- Attempt 1: Immediate
- Attempt 2: 5 minutes
- Attempt 3: 30 minutes
- Attempt 4: 2 hours
- Attempt 5: 6 hours
- Max attempts: 5

### 2. Throttling

#### Adaptive Throttling
**Implementation**: Adjust sending rate based on response

```go
type Throttler struct {
    currentRate   int
    maxRate       int
    minRate       int
    successRate   float64
    mu            sync.Mutex
}

func (t *Throttler) AdjustRate(success bool) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if success {
        t.successRate += 0.1
        if t.successRate > 0.95 && t.currentRate < t.maxRate {
            t.currentRate += 10
        }
    } else {
        t.successRate -= 0.1
        if t.successRate < 0.90 && t.currentRate > t.minRate {
            t.currentRate -= 10
        }
    }
}
```

**Throttling Rules**:
- Success rate > 95%: Increase rate by 10%
- Success rate 90-95%: Maintain rate
- Success rate < 90%: Decrease rate by 10%
- Minimum rate: 10 emails/minute
- Maximum rate: 1,000 emails/minute

#### Domain Throttling
**Implementation**: Limit emails per recipient domain

```go
type DomainThrottler struct {
    mu          sync.Mutex
    domains     map[string]*DomainLimit
}

type DomainLimit struct {
    EmailsSent  int
    LastReset   time.Time
    DailyLimit  int
}

func (dt *DomainThrottler) CheckLimit(domain string) (bool, error) {
    dt.mu.Lock()
    defer dt.mu.Unlock()
    
    limit, exists := dt.domains[domain]
    if !exists {
        limit = &DomainLimit{
            DailyLimit: 1000,
            LastReset:  time.Now(),
        }
        dt.domains[domain] = limit
    }
    
    // Reset if new day
    if time.Since(limit.LastReset) > 24*time.Hour {
        limit.EmailsSent = 0
        limit.LastReset = time.Now()
    }
    
    return limit.EmailsSent < limit.DailyLimit, nil
}
```

**Domain Limits**:
- Gmail: 500 emails/day
- Yahoo: 500 emails/day
- Outlook: 500 emails/day
- Corporate domains: 1,000 emails/day
- Custom domains: Negotiated

### 3. Monitoring

#### Delivery Metrics
**Implementation**: Track delivery metrics

```go
type DeliveryMetrics struct {
    Sent         int64
    Delivered    int64
    Opened       int64
    Clicked      int64
    Bounced      int64
    Complained   int64
    Unsubscribed int64
}

func (dm *DeliveryMetrics) CalculateDeliveryRate() float64 {
    if dm.Sent == 0 {
        return 0
    }
    return float64(dm.Delivered) / float64(dm.Sent) * 100
}

func (dm *DeliveryMetrics) CalculateOpenRate() float64 {
    if dm.Delivered == 0 {
        return 0
    }
    return float64(dm.Opened) / float64(dm.Delivered) * 100
}

func (dm *DeliveryMetrics) CalculateClickRate() float64 {
    if dm.Opened == 0 {
        return 0
    }
    return float64(dm.Clicked) / float64(dm.Opened) * 100
}
```

**Key Metrics**:
- Delivery rate: Target > 95%
- Open rate: Target > 20%
- Click rate: Target > 2%
- Bounce rate: Target < 2%
- Complaint rate: Target < 0.1%
- Unsubscribe rate: Target < 0.5%

#### Real-time Monitoring
**Implementation**: Monitor delivery in real-time

```go
type DeliveryMonitor struct {
    metrics    *DeliveryMetrics
    alerts     chan *Alert
    thresholds map[string]float64
}

type Alert struct {
    Type      string
    Metric    string
    Value     float64
    Threshold float64
    Timestamp time.Time
}

func (dm *DeliveryMonitor) CheckThresholds() {
    deliveryRate := dm.metrics.CalculateDeliveryRate()
    if deliveryRate < dm.thresholds["delivery_rate"] {
        dm.alerts <- &Alert{
            Type:      "threshold_breach",
            Metric:    "delivery_rate",
            Value:     deliveryRate,
            Threshold: dm.thresholds["delivery_rate"],
            Timestamp: time.Now(),
        }
    }
}
```

**Alert Thresholds**:
- Delivery rate < 90%: Alert
- Bounce rate > 5%: Alert
- Complaint rate > 0.5%: Alert
- Unsubscribe rate > 1%: Alert

## Compliance

### 1. CAN-SPAM Act
**Requirements**:
- Clear opt-out mechanism
- Physical mailing address
- Accurate header information
- No misleading subject lines
- Honor opt-out requests within 10 business days

**Implementation**:
```go
type CANSPAMCompliance struct {
    unsubscribeURL string
    physicalAddr  string
}

func (csc *CANSPAMCompliance) Validate(email *Email) error {
    // Check for unsubscribe link
    // Check for physical address
    // Check for accurate headers
    // Check for non-misleading subject
    return nil
}
```

### 2. GDPR
**Requirements**:
- Explicit consent for marketing emails
- Right to opt-out
- Data protection
- Right to be forgotten

**Implementation**:
```go
type GDPRCompliance struct {
    consentManager *ConsentManager
}

func (gc *GDPRCompliance) CheckConsent(recipient string) (bool, error) {
    // Check if recipient has given consent
    return true, nil
}

func (gc *GDPRCompliance) HandleDataDeletionRequest(recipient string) error {
    // Delete all data for recipient
    return nil
}
```

### 3. CASL (Canada Anti-Spam Legislation)
**Requirements**:
- Express consent
- Identify sender
- Provide opt-out mechanism
- Include contact information

**Implementation**:
```go
type CASLCompliance struct {
    consentManager *ConsentManager
}

func (cc *CASLCompliance) Validate(email *Email) error {
    // Check for express consent
    // Check for sender identification
    // Check for opt-out mechanism
    // Check for contact information
    return nil
}
```

## Best Practices

### 1. Email Content
- Use clear, non-misleading subject lines
- Include physical mailing address
- Provide one-click unsubscribe
- Use plain text version alongside HTML
- Avoid spam trigger words
- Personalize content when possible

### 2. Technical Setup
- Implement SPF, DKIM, and DMARC
- Use dedicated IP addresses
- Monitor IP reputation
- Set up feedback loops
- Implement proper authentication

### 3. List Management
- Use double opt-in for new subscribers
- Regularly clean email lists
- Remove inactive subscribers
- Honor unsubscribe requests promptly
- Segment lists for targeted sending

### 4. Sending Practices
- Send during optimal times
- Avoid sending too frequently
- Monitor engagement metrics
- A/B test subject lines
- Personalize content

## Implementation Timeline

### Phase 1: Foundation (Week 1-2)
1. Implement SPF, DKIM, DMARC
2. Set up rate limiting
3. Implement basic content filtering

### Phase 2: Reputation (Week 3-4)
4. Implement reputation management
5. Set up bounce handling
6. Implement feedback loop

### Phase 3: Optimization (Week 5-6)
7. Implement queue management
8. Set up throttling
9. Implement monitoring

### Phase 4: Compliance (Week 7-8)
10. Implement CAN-SPAM compliance
11. Implement GDPR compliance
12. Implement CASL compliance

## Conclusion

The Musketeers project email delivery strategy focuses on deliverability, anti-spam measures, and compliance. By implementing these strategies, the system will maintain high deliverability rates while protecting against spam and ensuring compliance with email regulations.

**Key Success Metrics**:
- Delivery rate: > 95%
- Open rate: > 20%
- Click rate: > 2%
- Bounce rate: < 2%
- Complaint rate: < 0.1%

**Next Steps**: Implement Phase 1 foundation measures, then proceed with reputation management and optimization.

## Sign-Off

**Strategy Created By**: Cascade AI Assistant
**Creation Date**: June 20, 2026
**Files Referenced**: pkg/email/email.go, pkg/orchestrator/email_system.go, pkg/email/integration.go

**Strategy Status**: **COMPREHENSIVE** - Covers all aspects of email delivery and anti-spam.

**Next Steps**: Begin implementation of Phase 1 foundation measures.
