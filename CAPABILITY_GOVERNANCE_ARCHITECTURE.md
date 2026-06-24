# معمارية نظام Capability Governance
## Capability-Based Access Control for Multi-Agent Sessions

## الرؤية الأساسية

التحول من **Tool Ownership** إلى **Capability Governance**

### المشكلة الحالية
```
السؤال الخاطئ: من يملك الأداة؟
```

### الحل الصحيح
```
السؤال الصحيح: من يملك حق استخدام الأداة الآن؟
```

---

## البنية المعمارية المقترحة

### الفصل بين المفاهيم

```
Tool (الأداة)
├── terminal
├── browser
├── github
├── docker
└── filesystem

Capability (القدرة)
├── terminal.read
├── terminal.execute
├── filesystem.read
├── filesystem.write
├── github.read
└── github.push

Delegation (التفويض)
├── Human
├── Session Manager
├── Agent
└── Workflow
```

---

## نظام Capability Token

### المفهوم
كل وكيل يحمل **Capability Token** بدلاً من الأدوات

```go
type CapabilityToken struct {
    TokenID         string                 // معرف فريد للرمز
    AgentID         string                 // معرف الوكيل
    SessionID       string                 // معرف الجلسة
    GrantedBy       string                 // من منح الصلاحية
    GrantedAt       time.Time              // وقت المنح
    ExpiresAt       time.Time              // وقت الانتهاء
    Capabilities    map[string]*CapabilityGrant // الصلاحيات الممنوحة
    Conditions      map[string]interface{} // شروط الاستخدام
    Metadata        map[string]interface{} // بيانات إضافية
    Status          TokenStatus            // حالة الرمز
    RevokedAt       *time.Time             // وقت الإلغاء (إن وجد)
    RevokedBy       string                 // من ألغى الرمز
}

type CapabilityGrant struct {
    CapabilityName string                 // اسم القدرة
    Actions        []string               // الإجراءات المسموحة
    Resources      []string               // الموارد المسموحة
    Constraints    map[string]interface{} // القيود
    GrantedAt      time.Time              // وقت المنح
    ExpiresAt      *time.Time             // وقت الانتهاء (اختياري)
}

type TokenStatus string

const (
    TokenActive    TokenStatus = "active"
    TokenExpired   TokenStatus = "expired"
    TokenRevoked   TokenStatus = "revoked"
    TokenSuspended TokenStatus = "suspended"
)
```

---

## نظام Delegation الديناميكي

### مصادر التفويض

```go
type DelegationSource string

const (
    DelegationHuman          DelegationSource = "human"
    DelegationSessionManager DelegationSource = "session_manager"
    DelegationAgent          DelegationSource = "agent"
    DelegationWorkflow       DelegationSource = "workflow"
    DelegationPolicy         DelegationSource = "policy"
)

type DelegationRequest struct {
    RequestID       string                 // معرف الطلب
    Delegator       string                 // المفوّض
    Delegatee       string                 // المفوَّض إليه
    SessionID       string                 // معرف الجلسة
    Capabilities    []CapabilityGrant      // الصلاحيات المطلوبة
    Duration        time.Duration          // مدة التفويض
    Conditions      map[string]interface{} // شروط التفويض
    Reason          string                 // سبب التفويض
    RequestedAt     time.Time              // وقت الطلب
}

type DelegationResponse struct {
    RequestID    string        // معرف الطلب
    TokenID      string        // معرف الرمز الممنوح
    Granted      bool          // هل تم المنح؟
    Reason       string        // سبب الرفض (إن وجد)
    GrantedAt    time.Time     // وقت المنح
    ExpiresAt    time.Time     // وقت الانتهاء
}
```

---

## Capability Governance Manager

```go
type CapabilityGovernanceManager struct {
    tokens          map[string]*CapabilityToken // الرموز النشطة
    delegationLog   []*DelegationRecord          // سجل التفويضات
    policyEngine    *policy.Engine               // محرك السياسات
    sessionManager  *SessionManager              // مدير الجلسة
    eventBus        *eventbus.EventBus           // ناقل الأحداث
    logger          *zap.Logger
    mu              sync.RWMutex
}

type DelegationRecord struct {
    RecordID      string                 // معرف السجل
    Delegator     string                 // المفوّض
    Delegatee     string                 // المفوَّض إليه
    TokenID       string                 // معرف الرمز
    Capabilities  []string               // الصلاحيات الممنوحة
    GrantedAt     time.Time              // وقت المنح
    ExpiresAt     time.Time              // وقت الانتهاء
    RevokedAt     *time.Time             // وقت الإلغاء
    Reason        string                 // سبب التفويض
    Metadata      map[string]interface{} // بيانات إضافية
}

// GrantCapability يمنح صلاحية لوكيل
func (cgm *CapabilityGovernanceManager) GrantCapability(
    ctx context.Context,
    delegator string,
    delegatee string,
    capability CapabilityGrant,
    duration time.Duration,
) (*CapabilityToken, error) {
    
    // التحقق من الصلاحيات
    if !cgm.canDelegate(ctx, delegator, capability) {
        return nil, fmt.Errorf("delegator does not have permission to grant this capability")
    }
    
    // إنشاء الرمز
    token := &CapabilityToken{
        TokenID:      generateTokenID(),
        AgentID:      delegatee,
        SessionID:    cgm.getSessionID(ctx),
        GrantedBy:    delegator,
        GrantedAt:    time.Now(),
        ExpiresAt:    time.Now().Add(duration),
        Capabilities: map[string]*CapabilityGrant{
            capability.CapabilityName: &capability,
        },
        Status: TokenActive,
    }
    
    // حفظ الرمز
    cgm.mu.Lock()
    cgm.tokens[token.TokenID] = token
    cgm.mu.Unlock()
    
    // تسجيل التفويض
    cgm.logDelegation(delegator, delegatee, token.TokenID, []string{capability.CapabilityName})
    
    // نشر حدث
    cgm.publishEvent("capability_granted", map[string]interface{}{
        "token_id": token.TokenID,
        "delegator": delegator,
        "delegatee": delegatee,
        "capability": capability.CapabilityName,
    })
    
    return token, nil
}

// RevokeCapability يلغي صلاحية
func (cgm *CapabilityGovernanceManager) RevokeCapability(
    ctx context.Context,
    revoker string,
    tokenID string,
    reason string,
) error {
    
    cgm.mu.Lock()
    defer cgm.mu.Unlock()
    
    token, exists := cgm.tokens[tokenID]
    if !exists {
        return fmt.Errorf("token not found: %s", tokenID)
    }
    
    // التحقق من الصلاحية
    if !cgm.canRevoke(ctx, revoker, token) {
        return fmt.Errorf("revoker does not have permission to revoke this token")
    }
    
    // إلغاء الرمز
    token.Status = TokenRevoked
    token.RevokedAt = &[]time.Time{time.Now()}[0]
    token.RevokedBy = revoker
    
    // نشر حدث
    cgm.publishEvent("capability_revoked", map[string]interface{}{
        "token_id": tokenID,
        "revoker": revoker,
        "reason": reason,
    })
    
    return nil
}

// CheckCapability يتحقق من صلاحية
func (cgm *CapabilityGovernanceManager) CheckCapability(
    ctx context.Context,
    agentID string,
    capabilityName string,
    action string,
) bool {
    
    cgm.mu.RLock()
    defer cgm.mu.RUnlock()
    
    // البحث عن رمز نشط للوكيل
    for _, token := range cgm.tokens {
        if token.AgentID == agentID && token.Status == TokenActive {
            // التحقق من انتهاء الصلاحية
            if time.Now().After(token.ExpiresAt) {
                token.Status = TokenExpired
                continue
            }
            
            // التحقق من الصلاحية
            if grant, exists := token.Capabilities[capabilityName]; exists {
                // التحقق من الإجراء
                for _, allowedAction := range grant.Actions {
                    if allowedAction == action {
                        return true
                    }
                }
            }
        }
    }
    
    return false
}
```

---

## الحل الهجين: Shared + Isolated

### الأدوات المنطقية (Shared)

```go
type LogicalTool struct {
    Name        string
    Type        ToolType
    Shared      bool
    Capabilities []string
}

type ToolType string

const (
    ToolTypeMemory    ToolType = "memory"
    ToolTypeSkills    ToolType = "skills"
    ToolTypeChannels  ToolType = "channels"
    ToolTypeRegistry  ToolType = "registry"
    ToolTypeKnowledge ToolType = "knowledge"
)

// الأدوات المشتركة
var SharedTools = []*LogicalTool{
    {
        Name:   "memory",
        Type:   ToolTypeMemory,
        Shared: true,
        Capabilities: []string{
            "memory.read",
            "memory.write",
            "memory.search",
        },
    },
    {
        Name:   "skills",
        Type:   ToolTypeSkills,
        Shared: true,
        Capabilities: []string{
            "skills.read",
            "skills.execute",
            "skills.learn",
        },
    },
    {
        Name:   "channels",
        Type:   ToolTypeChannels,
        Shared: true,
        Capabilities: []string{
            "channels.read",
            "channels.write",
            "channels.join",
        },
    },
}
```

### الأدوات التنفيذية (Isolated)

```go
type ExecutionTool struct {
    Name        string
    Type        ToolType
    Shared      bool
    RequiresSandbox bool
    Capabilities []string
}

const (
    ToolTypeTerminal ToolType = "terminal"
    ToolTypeBrowser  ToolType = "browser"
    ToolTypeSandbox  ToolType = "sandbox"
    ToolTypeRuntime  ToolType = "runtime"
)

// الأدوات المعزولة
var IsolatedTools = []*ExecutionTool{
    {
        Name:           "terminal",
        Type:           ToolTypeTerminal,
        Shared:         false,
        RequiresSandbox: true,
        Capabilities: []string{
            "terminal.read",
            "terminal.execute",
        },
    },
    {
        Name:           "browser",
        Type:           ToolTypeBrowser,
        Shared:         false,
        RequiresSandbox: true,
        Capabilities: []string{
            "browser.navigate",
            "browser.interact",
        },
    },
}
```

---

## طبقة التنفيذ المحسّنة

```go
type ExecutionLayer struct {
    capabilityManager *CapabilityGovernanceManager
    sharedTools       map[string]*LogicalTool
    isolatedTools     map[string]*ExecutionTool
    sandboxManager    *SandboxManager
    logger            *zap.Logger
}

func (el *ExecutionLayer) Execute(
    ctx context.Context,
    agentID string,
    toolName string,
    action string,
    args map[string]interface{},
) (*Result, error) {
    
    // التحقق من الصلاحية
    capabilityName := fmt.Sprintf("%s.%s", toolName, action)
    if !el.capabilityManager.CheckCapability(ctx, agentID, capabilityName, action) {
        return nil, fmt.Errorf("agent %s does not have capability %s", agentID, capabilityName)
    }
    
    // تحديد نوع الأداة
    tool, isShared := el.sharedTools[toolName]
    if !isShared {
        isolatedTool, exists := el.isolatedTools[toolName]
        if !exists {
            return nil, fmt.Errorf("tool not found: %s", toolName)
        }
        
        // تنفيذ في صندوق رمل معزول
        return el.executeInSandbox(ctx, agentID, isolatedTool, action, args)
    }
    
    // تنفيذ مشترك
    return el.executeShared(ctx, agentID, tool, action, args)
}

func (el *ExecutionLayer) executeInSandbox(
    ctx context.Context,
    agentID string,
    tool *ExecutionTool,
    action string,
    args map[string]interface{},
) (*Result, error) {
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := el.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // تنفيذ في الصندوق الرملي
    return sandbox.Execute(ctx, tool.Name, action, args)
}
```

---

## سيناريو عملي

### السيناريو 1: توزيع الأدوار

```
العميل البشري يطلب:
"أريد من Claude أن يكون Architect و Codex أن يكون Implementer"
```

```go
// Session Manager يمنح الصلاحيات
sessionManager.GrantCapability(
    ctx,
    "human",           // المفوّض
    "claude",          // المفوَّض إليه
    CapabilityGrant{
        CapabilityName: "filesystem",
        Actions:       []string{"read"},
        Resources:     []string{"*"},
    },
    2*time.Hour,      // مدة التفويض
)

sessionManager.GrantCapability(
    ctx,
    "human",           // المفوّض
    "codex",           // المفوَّض إليه
    CapabilityGrant{
        CapabilityName: "filesystem",
        Actions:       []string{"write"},
        Resources:     []string{"*"},
    },
    2*time.Hour,      // مدة التفويض
)
```

### السيناريو 2: تفويض مؤقت

```
بعد ساعتين، Claude يحتاج Terminal Access مؤقتاً
```

```go
// Session Manager يمنح صلاحية مؤقتة
sessionManager.GrantCapability(
    ctx,
    "session_manager", // المفوّض
    "claude",          // المفوَّض إليه
    CapabilityGrant{
        CapabilityName: "terminal",
        Actions:       []string{"execute"},
        Resources:     []string{"*"},
    },
    10*time.Minute,    // مدة التفويض
)
```

### السيناريو 3: التنفيذ

```
Claude يريد تنفيذ أمر
```

```go
// Claude يطلب التنفيذ
result, err := executionLayer.Execute(
    ctx,
    "claude",
    "terminal",
    "execute",
    map[string]interface{}{
        "command": "go build",
    },
)

// طبقة التنفيذ تتحقق من الصلاحية
// هل يملك Claude terminal.execute؟
// نعم -> تنفذ في صندوق رمله الخاص
// لا -> ترفض
```

---

## التكامل مع pkg/capability الحالي

### توسيع الواجهات

```go
// توسيع Capability interface
type DynamicCapability interface {
    capability.Capability
    RequiresSandbox() bool
    IsShared() bool
    GetRequiredCapabilities() []string
}

// توسيع Manager
type GovernanceManager struct {
    *capability.Manager
    governance *CapabilityGovernanceManager
    execution  *ExecutionLayer
}

func NewGovernanceManager(
    policyEngine *policy.Engine,
    sessionManager *SessionManager,
    eventBus *eventbus.EventBus,
    logger *zap.Logger,
) *GovernanceManager {
    
    baseManager := capability.NewManager(policyEngine)
    
    governance := NewCapabilityGovernanceManager(
        policyEngine,
        sessionManager,
        eventBus,
        logger,
    )
    
    execution := NewExecutionLayer(
        governance,
        logger,
    )
    
    return &GovernanceManager{
        Manager:    baseManager,
        governance: governance,
        execution:  execution,
    }
}
```

---

## المزايا

### 1. المرونة
- صلاحيات ديناميكية قابلة للتغيير في أي وقت
- تفويض مؤقت للصلاحيات
- شروط وقيود قابلة للتخصيص

### 2. الأمان
- تحكم دقيق في الصلاحيات
- تتبع كامل للتفويضات
- إلغاء فوري للصلاحيات

### 3. الكفاءة
- أدوات منطقية مشتركة (لا استنساخ)
- أدوات تنفيذية معزولة (أمان)
- استخدام ذكي للموارد

### 4. البساطة
- فصل واضح بين Tool و Capability و Delegation
- بنية بسيطة وسهلة الفهم
- سهولة الصيانة والتوسع

---

## خطة التنفيذ

### المرحلة 1: Capability Token System (1 أسبوع)
1. تصميم CapabilityToken struct
2. تصميم CapabilityGrant struct
3. تنفيذ CapabilityGovernanceManager
4. تنفيذ GrantCapability و RevokeCapability
5. تنفيذ CheckCapability

### المرحلة 2: Delegation System (1 أسبوع)
1. تصميم DelegationRequest و DelegationResponse
2. تنفيذ مصادر التفويض المتعددة
3. تنفيذ سجل التفويضات
4. تنفيذ التحقق من الصلاحيات

### المرحلة 3: Hybrid Tool System (1 أسبوع)
1. تصميم LogicalTool و ExecutionTool
2. تنفيذ ExecutionLayer
3. تنفيذ SandboxManager
4. تنفيذ executeShared و executeInSandbox

### المرحلة 4: Integration (1 أسبوع)
1. التكامل مع pkg/capability الحالي
2. التكامل مع SessionManager
3. التكامل مع EventBus
4. التكامل مع Policy Engine

### المرحلة 5: Testing (1 أسبوع)
1. اختبارات الوحدة
2. اختبارات التكامل
3. اختبارات السيناريوهات
4. اختبارات الأمان

---

## الاستنتاج

هذا النظام يحقق رؤيتك بالكامل:

1. **التعاون الكامل**: الوكلاء يشاركون كل شيء عبر EventBus و CollectiveMemory
2. **السرعة**: لا انتظار، صلاحيات ديناميكية
3. **البساطة**: فصل واضح بين المفاهيم
4. **الأمان**: تحكم دقيق في الصلاحيات
5. **الاستقرار**: بنية مستقرة وقابلة للتوسع

النظام أقرب إلى **نظام تشغيل للوكلاء والبشر**، حيث تُمنح القدرات المؤقتة للوصول إلى الموارد وفق صلاحيات محددة.
