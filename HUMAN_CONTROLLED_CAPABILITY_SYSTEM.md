# نظام إدارة الصلاحيات المربوط بنظام التفويضات
## Human-Controlled Capability Management System

## الرؤية الأساسية

ربط نظام الصلاحيات الجديد (Capability Governance) بنظام التفويضات الموجود (pkg/delegation)، مع خيارات مرنة للعميل البشري للتحكم في الصلاحيات.

---

## أنظمة التحكم المختلفة

### 1. الوضع الأوتوماتيكي (Automatic Mode)
العميل البشري يترك للوكيل مدير الجلسة تحديد الصلاحيات للوكلاء والنماذج المشاركة.

### 2. الوضع اليدوي (Manual Mode)
العميل البشري يختار مدير الجلسة ويخصص الصلاحيات لكل نموذج/وكيل عند إنشاء الجلسة.

### 3. التعديل الديناميكي (Dynamic Modification)
العميل البشري يمكنه تعديل الصلاحيات يدوياً أثناء عمل الجلسة حسب تطور مشروعه أو رغبته.

---

## البنية المعمارية المتكاملة

```
Human Client
├── Session Creation
│   ├── Mode Selection (Automatic/Manual)
│   ├── Session Manager Selection
│   └── Initial Capability Assignment
├── Session Runtime
│   ├── Dynamic Capability Modification
│   ├── Real-time Permission Updates
│   └── Delegation Chain Management
└── Session Monitoring
    ├── Capability Usage Tracking
    ├── Permission Audit Log
    └── Security Alerts
```

---

## نظام إدارة الصلاحيات للعميل البشري

```go
type HumanCapabilityManager struct {
    delegationManager    *delegation.DelegationManager
    capabilityGovernance *CapabilityGovernanceManager
    sessionManager       *SessionManager
    eventBus             *eventbus.EventBus
    logger               *zap.Logger
    mu                   sync.RWMutex
}

type SessionCapabilityConfig struct {
    SessionID           string                 // معرف الجلسة
    Mode                ControlMode            وضع التحكم
    SessionManagerAgent string                 // معرف وكيل مدير الجلسة
    AgentCapabilities   map[string]*AgentCapabilityConfig // صلاحيات الوكلاء
    HumanOverride       bool                   // هل العميل البشري يمكنه التجاوز؟
    AllowDynamicMod    bool                   // هل يسمح بالتعديل الديناميكي؟
    ModificationHistory []*CapabilityModification // تاريخ التعديلات
    CreatedAt          time.Time              // وقت الإنشاء
    LastModified       time.Time              // آخر تعديل
}

type ControlMode string

const (
    ModeAutomatic ControlMode = "automatic" // أوتوماتيكي
    ModeManual    ControlMode = "manual"    // يدوي
)

type AgentCapabilityConfig struct {
    AgentID         string                 // معرف الوكيل
    Role            string                 // الدور (Architect, Implementer, etc.)
    Capabilities    []CapabilityGrant      // الصلاحيات الممنوحة
    DelegationChain []*DelegationRecord    // سلسلة التفويضات
    Restrictions    map[string]interface{} // القيود
    LastUpdated     time.Time              // آخر تحديث
    UpdatedBy       string                 // من حدّث
}

type CapabilityModification struct {
    ModificationID  string                 // معرف التعديل
    AgentID         string                 // معرف الوكيل
    OldCapabilities []CapabilityGrant      // الصلاحيات القديمة
    NewCapabilities []CapabilityGrant      // الصلاحيات الجديدة
    ModifiedBy      string                 // من عدّل
    Reason          string                 // سبب التعديل
    ModifiedAt      time.Time              // وقت التعديل
}
```

---

## الوضع الأوتوماتيكي (Automatic Mode)

### المفهوم
العميل البشري يترك للوكيل مدير الجلسة تحديد الصلاحيات بناءً على:
- تحليل المهام المطلوبة
- قدرات الوكلاء المتاحة
- توزيع الأدوار الأمثل

### التنفيذ

```go
func (hcm *HumanCapabilityManager) CreateAutomaticSession(
    ctx context.Context,
    humanClientID string,
    agents []string,
    taskDescription string,
) (*SessionCapabilityConfig, error) {
    
    // 1. إنشاء الجلسة
    sessionID := generateSessionID()
    
    // 2. اختيار مدير الجلسة تلقائياً
    sessionManagerAgent := hcm.selectSessionManagerAgent(agents)
    
    // 3. تحليل المهام المطلوبة
    requiredCapabilities := hcm.analyzeTaskRequirements(taskDescription)
    
    // 4. توزيع الصلاحيات تلقائياً
    agentCapabilities := make(map[string]*AgentCapabilityConfig)
    for _, agentID := range agents {
        capabilities := hcm.assignCapabilitiesForAgent(
            agentID,
            requiredCapabilities,
            sessionManagerAgent,
        )
        
        agentCapabilities[agentID] = &AgentCapabilityConfig{
            AgentID:      agentID,
            Role:         hcm.determineRole(agentID, capabilities),
            Capabilities: capabilities,
            LastUpdated:  time.Now(),
            UpdatedBy:    sessionManagerAgent,
        }
    }
    
    // 5. إنشاء التكوين
    config := &SessionCapabilityConfig{
        SessionID:           sessionID,
        Mode:                ModeAutomatic,
        SessionManagerAgent: sessionManagerAgent,
        AgentCapabilities:   agentCapabilities,
        HumanOverride:       true,
        AllowDynamicMod:     true,
        CreatedAt:           time.Now(),
        LastModified:        time.Now(),
    }
    
    // 6. تطبيق الصلاحيات
    if err := hcm.applyCapabilities(ctx, config); err != nil {
        return nil, fmt.Errorf("failed to apply capabilities: %w", err)
    }
    
    // 7. نشر حدث
    hcm.publishEvent("automatic_session_created", map[string]interface{}{
        "session_id": sessionID,
        "session_manager": sessionManagerAgent,
        "agents": agents,
    })
    
    return config, nil
}

func (hcm *HumanCapabilityManager) selectSessionManagerAgent(agents []string) string {
    // اختيار الوكيل الأنسب كمدير جلسة
    // يمكن أن يكون Claude أو GPT أو أي وكيل لديه قدرات إدارية
    // حالياً: اختيار أول وكيل في القائمة
    return agents[0]
}

func (hcm *HumanCapabilityManager) analyzeTaskRequirements(description string) []string {
    // تحليل وصف المهمة لتحديد الصلاحيات المطلوبة
    // يمكن استخدام LLM للتحليل
    // حالياً: إرجاع صلاحيات افتراضية
    return []string{
        "filesystem.read",
        "filesystem.write",
        "memory.read",
        "memory.write",
        "terminal.execute",
    }
}

func (hcm *HumanCapabilityManager) assignCapabilitiesForAgent(
    agentID string,
    requiredCapabilities []string,
    sessionManagerAgent string,
) []CapabilityGrant {
    
    // توزيع الصلاحيات بناءً على نوع الوكيل
    // حالياً: منح جميع الصلاحيات
    grants := make([]CapabilityGrant, 0, len(requiredCapabilities))
    for _, cap := range requiredCapabilities {
        parts := strings.Split(cap, ".")
        if len(parts) >= 2 {
            grants = append(grants, CapabilityGrant{
                CapabilityName: parts[0],
                Actions:       []string{parts[1]},
                Resources:     []string{"*"},
            })
        }
    }
    
    return grants
}
```

---

## الوضع اليدوي (Manual Mode)

### المفهوم
العميل البشري يختار مدير الجلسة ويخصص الصلاحيات لكل نموذج/وكيل عند إنشاء الجلسة.

### التنفيذ

```go
func (hcm *HumanCapabilityManager) CreateManualSession(
    ctx context.Context,
    humanClientID string,
    sessionManagerAgent string,
    agentCapabilities map[string][]CapabilityGrant,
) (*SessionCapabilityConfig, error) {
    
    // 1. إنشاء الجلسة
    sessionID := generateSessionID()
    
    // 2. بناء تكوين الوكلاء
    agentConfigs := make(map[string]*AgentCapabilityConfig)
    for agentID, capabilities := range agentCapabilities {
        agentConfigs[agentID] = &AgentCapabilityConfig{
            AgentID:      agentID,
            Role:         hcm.determineRole(agentID, capabilities),
            Capabilities: capabilities,
            LastUpdated:  time.Now(),
            UpdatedBy:    humanClientID,
        }
    }
    
    // 3. إنشاء التكوين
    config := &SessionCapabilityConfig{
        SessionID:           sessionID,
        Mode:                ModeManual,
        SessionManagerAgent: sessionManagerAgent,
        AgentCapabilities:   agentConfigs,
        HumanOverride:       true,
        AllowDynamicMod:     true,
        CreatedAt:           time.Now(),
        LastModified:        time.Now(),
    }
    
    // 4. تطبيق الصلاحيات
    if err := hcm.applyCapabilities(ctx, config); err != nil {
        return nil, fmt.Errorf("failed to apply capabilities: %w", err)
    }
    
    // 5. نشر حدث
    hcm.publishEvent("manual_session_created", map[string]interface{}{
        "session_id": sessionID,
        "session_manager": sessionManagerAgent,
        "agents": getKeys(agentCapabilities),
    })
    
    return config, nil
}
```

---

## التعديل الديناميكي (Dynamic Modification)

### المفهوم
العميل البشري يمكنه تعديل الصلاحيات يدوياً أثناء عمل الجلسة حسب تطور مشروعه أو رغبته.

### التنفيذ

```go
func (hcm *HumanCapabilityManager) ModifyAgentCapabilities(
    ctx context.Context,
    sessionID string,
    humanClientID string,
    agentID string,
    newCapabilities []CapabilityGrant,
    reason string,
) error {
    
    // 1. التحقق من صلاحية التعديل
    config, err := hcm.getSessionConfig(sessionID)
    if err != nil {
        return fmt.Errorf("session not found: %w", err)
    }
    
    if !config.AllowDynamicMod {
        return fmt.Errorf("dynamic modification not allowed for this session")
    }
    
    // 2. حفظ الصلاحيات القديمة
    oldCapabilities := config.AgentCapabilities[agentID].Capabilities
    
    // 3. تحديث الصلاحيات
    config.AgentCapabilities[agentID].Capabilities = newCapabilities
    config.AgentCapabilities[agentID].LastUpdated = time.Now()
    config.AgentCapabilities[agentID].UpdatedBy = humanClientID
    
    // 4. تسجيل التعديل
    modification := &CapabilityModification{
        ModificationID:  generateModificationID(),
        AgentID:         agentID,
        OldCapabilities: oldCapabilities,
        NewCapabilities: newCapabilities,
        ModifiedBy:      humanClientID,
        Reason:          reason,
        ModifiedAt:      time.Now(),
    }
    config.ModificationHistory = append(config.ModificationHistory, modification)
    config.LastModified = time.Now()
    
    // 5. تطبيق الصلاحيات الجديدة
    if err := hcm.applyCapabilities(ctx, config); err != nil {
        // التراجع في حالة الفشل
        config.AgentCapabilities[agentID].Capabilities = oldCapabilities
        return fmt.Errorf("failed to apply capabilities: %w", err)
    }
    
    // 6. نشر حدث
    hcm.publishEvent("capabilities_modified", map[string]interface{}{
        "session_id": sessionID,
        "agent_id": agentID,
        "modified_by": humanClientID,
        "reason": reason,
    })
    
    return nil
}

func (hcm *HumanCapabilityManager) RevokeAgentCapabilities(
    ctx context.Context,
    sessionID string,
    humanClientID string,
    agentID string,
    capabilitiesToRevoke []string,
    reason string,
) error {
    
    // 1. الحصول على التكوين
    config, err := hcm.getSessionConfig(sessionID)
    if err != nil {
        return fmt.Errorf("session not found: %w", err)
    }
    
    // 2. إزالة الصلاحيات المحددة
    agentConfig := config.AgentCapabilities[agentID]
    oldCapabilities := agentConfig.Capabilities
    
    newCapabilities := make([]CapabilityGrant, 0)
    for _, grant := range agentConfig.Capabilities {
        shouldRevoke := false
        for _, capToRevoke := range capabilitiesToRevoke {
            if grant.CapabilityName == capToRevoke {
                shouldRevoke = true
                break
            }
        }
        if !shouldRevoke {
            newCapabilities = append(newCapabilities, grant)
        }
    }
    
    // 3. تحديث الصلاحيات
    return hcm.ModifyAgentCapabilities(
        ctx,
        sessionID,
        humanClientID,
        agentID,
        newCapabilities,
        reason,
    )
}

func (hcm *HumanCapabilityManager) GrantAgentCapabilities(
    ctx context.Context,
    sessionID string,
    humanClientID string,
    agentID string,
    capabilitiesToGrant []CapabilityGrant,
    reason string,
) error {
    
    // 1. الحصول على التكوين
    config, err := hcm.getSessionConfig(sessionID)
    if err != nil {
        return fmt.Errorf("session not found: %w", err)
    }
    
    // 2. إضافة الصلاحيات الجديدة
    agentConfig := config.AgentCapabilities[agentID]
    oldCapabilities := agentConfig.Capabilities
    
    newCapabilities := append([]CapabilityGrant{}, oldCapabilities...)
    newCapabilities = append(newCapabilities, capabilitiesToGrant...)
    
    // 3. تحديث الصلاحيات
    return hcm.ModifyAgentCapabilities(
        ctx,
        sessionID,
        humanClientID,
        agentID,
        newCapabilities,
        reason,
    )
}
```

---

## الربط مع نظام التفويضات الموجود

```go
func (hcm *HumanCapabilityManager) applyCapabilities(
    ctx context.Context,
    config *SessionCapabilityConfig,
) error {
    
    for agentID, agentConfig := range config.AgentCapabilities {
        for _, grant := range agentConfig.Capabilities {
            // 1. إنشاء سجل تفويض
            delegationRecord, err := hcm.createDelegationRecord(
                config.SessionManagerAgent,
                agentID,
                grant,
                24*time.Hour, // مدة افتراضية
            )
            if err != nil {
                return fmt.Errorf("failed to create delegation: %w", err)
            }
            
            // 2. إضافة السجل إلى سلسلة التفويضات
            agentConfig.DelegationChain = append(
                agentConfig.DelegationChain,
                delegationRecord,
            )
            
            // 3. منح الصلاحية عبر نظام Capability Governance
            _, err = hcm.capabilityGovernance.GrantCapability(
                ctx,
                config.SessionManagerAgent,
                agentID,
                grant,
                24*time.Hour,
            )
            if err != nil {
                return fmt.Errorf("failed to grant capability: %w", err)
            }
        }
    }
    
    return nil
}

func (hcm *HumanCapabilityManager) createDelegationRecord(
    delegator string,
    delegatee string,
    grant CapabilityGrant,
    duration time.Duration,
) (*delegation.DelegationRecord, error) {
    
    // تحويل CapabilityGrant إلى DelegationScope
    scope := delegation.DelegationScope{
        AllowedActions: grant.Actions,
    }
    
    // إنشاء سجل تفويض
    // ملاحظة: هذا يتطلب مفتاح خاص للمفوض
    // حالياً: نستخدم مفتاح افتراضي للتجربة
    // في الإنتاج: يجب استخدام مفاتيح حقيقية
    
    record := &delegation.DelegationRecord{
        ID:           generateDelegationID(),
        DelegatorDID: delegator,
        DelegateDID:  delegatee,
        Scope:        scope,
        ExpiresAt:    time.Now().Add(duration),
        // Signature: سيتم إضافته عند استخدام مفتاح حقيقي
    }
    
    return record, nil
}
```

---

## واجهة المستخدم للعميل البشري

### 1. إنشاء جلسة (أوتوماتيكي)

```json
{
  "mode": "automatic",
  "agents": ["claude", "gpt", "codex"],
  "task_description": "بناء تطبيق ويب متكامل"
}
```

### 2. إنشاء جلسة (يدوي)

```json
{
  "mode": "manual",
  "session_manager": "claude",
  "agent_capabilities": {
    "claude": [
      {
        "capability_name": "filesystem",
        "actions": ["read"],
        "resources": ["*"]
      }
    ],
    "codex": [
      {
        "capability_name": "filesystem",
        "actions": ["write"],
        "resources": ["*"]
      },
      {
        "capability_name": "terminal",
        "actions": ["execute"],
        "resources": ["*"]
      }
    ]
  }
}
```

### 3. تعديل الصلاحيات (ديناميكي)

```json
{
  "session_id": "session-123",
  "agent_id": "claude",
  "action": "grant",
  "capabilities": [
    {
      "capability_name": "terminal",
      "actions": ["execute"],
      "resources": ["*"]
    }
  ],
  "reason": "Claude يحتاج Terminal Access مؤقتاً"
}
```

### 4. إلغاء الصلاحيات

```json
{
  "session_id": "session-123",
  "agent_id": "codex",
  "action": "revoke",
  "capabilities": ["terminal.execute"],
  "reason": "Codex لم يعد يحتاج Terminal Access"
}
```

---

## التكامل مع SessionManager

```go
type SessionManager struct {
    // ... الحقول الموجودة
    humanCapabilityManager *HumanCapabilityManager
    sessionCapabilityConfigs map[string]*SessionCapabilityConfig
}

func (sm *SessionManager) InitializeWithHumanControl(
    ctx context.Context,
    agentExecutor AgentExecutor,
    humanClientID string,
    mode ControlMode,
    agents []string,
    taskDescription string,
) error {
    
    // 1. تهيئة مدير الجلسة
    if err := sm.Initialize(ctx, agentExecutor); err != nil {
        return err
    }
    
    // 2. إنشاء تكوين الصلاحيات
    var config *SessionCapabilityConfig
    var err error
    
    switch mode {
    case ModeAutomatic:
        config, err = sm.humanCapabilityManager.CreateAutomaticSession(
            ctx,
            humanClientID,
            agents,
            taskDescription,
        )
    case ModeManual:
        // في الوضع اليدوي، يجب توفير الصلاحيات بشكل منفصل
        return fmt.Errorf("manual mode requires explicit capability assignment")
    }
    
    if err != nil {
        return fmt.Errorf("failed to create capability config: %w", err)
    }
    
    // 3. حفظ التكوين
    sm.sessionCapabilityConfigs[config.SessionID] = config
    
    // 4. بدء الجلسة
    sm.logger.Info("Session initialized with human control",
        zap.String("session_id", config.SessionID),
        zap.String("mode", string(mode)),
        zap.String("session_manager", config.SessionManagerAgent),
    )
    
    return nil
}
```

---

## المراقبة والتدقيق

```go
type CapabilityAuditLog struct {
    AuditID        string                 // معرف التدقيق
    SessionID      string                 // معرف الجلسة
    AgentID        string                 // معرف الوكيل
    Action         string                 // الإجراء (grant, revoke, modify)
    Capabilities   []string               // الصلاحيات المتأثرة
    PerformedBy    string                 // من نفذ الإجراء
    Reason         string                 // سبب الإجراء
    Timestamp      time.Time              // وقت الإجراء
    Result         string                 // النتيجة (success, failure)
    Details        map[string]interface{} // تفاصيل إضافية
}

func (hcm *HumanCapabilityManager) LogCapabilityAction(
    sessionID string,
    agentID string,
    action string,
    capabilities []string,
    performedBy string,
    reason string,
    result string,
) {
    
    log := &CapabilityAuditLog{
        AuditID:      generateAuditID(),
        SessionID:    sessionID,
        AgentID:      agentID,
        Action:       action,
        Capabilities: capabilities,
        PerformedBy:  performedBy,
        Reason:       reason,
        Timestamp:    time.Now(),
        Result:       result,
    }
    
    // حفظ السجل
    hcm.saveAuditLog(log)
    
    // نشر حدث
    hcm.publishEvent("capability_audit", map[string]interface{}{
        "audit_id": log.AuditID,
        "session_id": log.SessionID,
        "agent_id": log.AgentID,
        "action": log.Action,
        "performed_by": log.PerformedBy,
        "result": log.Result,
    })
}
```

---

## المزايا

### 1. المرونة الكاملة
- أوتوماتيكي للإعدادات السريعة
- يدوي للتحكم الدقيق
- تعديل ديناميكي للتغييرات أثناء العمل

### 2. الأمان الشامل
- ربط مع نظام التفويضات الموجود
- تتبع كامل للتعديلات
- سجل تدقيق شامل

### 3. سهولة الاستخدام
- واجهة بسيطة للعميل البشري
- أوضاع واضحة ومفهومة
- تعديلات سريعة وسهلة

### 4. التكامل الكامل
- ربط مع pkg/delegation الموجود
- ربط مع pkg/capability الجديد
- ربط مع SessionManager

---

## خطة التنفيذ

### المرحلة 1: HumanCapabilityManager (1 أسبوع)
1. تصميم HumanCapabilityManager struct
2. تنفيذ CreateAutomaticSession
3. تنفيذ CreateManualSession
4. تنفيذ التعديلات الديناميكية

### المرحلة 2: الربط مع Delegation (1 أسبوع)
1. تنفيذ createDelegationRecord
2. تنفيذ applyCapabilities
3. التكامل مع pkg/delegation
4. التحقق من التوقيعات

### المرحلة 3: واجهة المستخدم (1 أسبوع)
1. تصميم واجهة API
2. تنفيذ نقاط النهاية
3. تصميم واجهة المستخدم
4. اختبار التجربة

### المرحلة 4: التكامل (1 أسبوع)
1. التكامل مع SessionManager
2. التكامل مع EventBus
3. التكامل مع CollectiveMemory
4. اختبارات التكامل

### المرحلة 5: المراقبة والتدقيق (1 أسبوع)
1. تنفيذ CapabilityAuditLog
2. تنفيذ سجل التدقيق
3. تنفيذ التنبيهات
4. اختبارات الأمان

---

## الاستنتاج

هذا النظام يحقق جميع متطلباتك:

1. **الربط مع نظام التفويضات**: متكامل بالكامل مع pkg/delegation
2. **الوضع الأوتوماتيكي**: Session Manager يدير الصلاحيات تلقائياً
3. **الوضع اليدوي**: العميل البشري يحدد الصلاحيات عند الإنشاء
4. **التعديل الديناميكي**: العميل البشري يمكنه التعديل أثناء العمل
5. **المرونة الكاملة**: يناسب جميع احتياجات العميل البشري

النظام يوازن بين الأتمتة والتحكم اليدوي، مع الحفاظ على الأمان والتتبع الكامل.
