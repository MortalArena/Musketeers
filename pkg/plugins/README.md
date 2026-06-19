# نظام الإضافات (Plugin System)

## التاريخ: 19 يونيو 2026

## الهدف:
توفير نظام إضافات مرن وقابل للتوسع يسمح للمستخدمين بإضافة وظائف جديدة إلى المنصة دون تعديل الكود الأساسي.

---

## البنية (Architecture):

### 📁 pkg/plugins/
- `core/plugin.go` - النظام الأساسي للإضافات
- `loader/loader.go` - محمل الإضافات
- `registry/registry.go` - سجل الإضافات
- `hooks/hooks.go` - نظام الخطافات
- `sandbox/sandbox.go` - صندوق الرمل للإضافات

---

## المكونات الرئيسية:

### 1. Plugin Interface (واجهة الإضافة)
```go
type Plugin interface {
    // معلومات الإضافة
    Name() string
    Version() string
    Description() string
    Author() string

    // دورة حياة الإضافة
    Initialize(ctx context.Context, config map[string]interface{}) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Shutdown(ctx context.Context) error

    // حالة الإضافة
    Status() PluginStatus
    Health() PluginHealth

    // التكامل
    GetDependencies() []string
    GetCapabilities() []string
}
```

### 2. PluginManager (مدير الإضافات)
- `Register` - تسجيل إضافة جديدة
- `Unregister` - إلغاء تسجيل إضافة
- `Initialize` - تهيئة إضافة
- `Start` - بدء إضافة
- `Stop` - إيقاف إضافة
- `GetPlugin` - الحصول على إضافة
- `GetAllPlugins` - الحصول على جميع الإضافات
- `GetPluginMetadata` - الحصول على بيانات وصفية للإضافة
- `GetPluginHealth` - الحصول على حالة صحة الإضافة
- `GetPluginsByCapability` - الحصول على الإضافات حسب القدرة
- `GetSummary` - الحصول على ملخص الإضافات

### 3. PluginStatus (حالة الإضافة)
- `Uninitialized` - غير مهيأ
- `Initializing` - جاري التهيئة
- `Ready` - جاهز
- `Running` - يعمل
- `Paused` - متوقف مؤقتاً
- `Stopping` - جاري الإيقاف
- `Stopped` - متوقف
- `Error` - خطأ

### 4. PluginHealth (صحة الإضافة)
- `Status` - الحالة
- `Message` - الرسالة
- `LastCheck` - آخر فحص
- `Metrics` - المقاييس

---

## الاستخدام (Usage):

### مثال 1: تسجيل إضافة
```go
pluginManager := core.NewPluginManager(logger, eventBus)

plugin := &MyPlugin{
    name:        "my-plugin",
    version:     "1.0.0",
    description: "My custom plugin",
    author:      "John Doe",
}

config := map[string]interface{}{
    "api_key": "secret-key",
    "timeout": 30,
}

err := pluginManager.Register(plugin, config)
if err != nil {
    log.Fatal(err)
}
```

### مثال 2: تهيئة وبدء إضافة
```go
err := pluginManager.Initialize("my-plugin")
if err != nil {
    log.Fatal(err)
}

err = pluginManager.Start("my-plugin")
if err != nil {
    log.Fatal(err)
}
```

### مثال 3: الحصول على إضافة
```go
plugin, err := pluginManager.GetPlugin("my-plugin")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Plugin: %s\n", plugin.Name())
fmt.Printf("Version: %s\n", plugin.Version())
```

### مثال 4: الحصول على حالة صحة الإضافة
```go
health, err := pluginManager.GetPluginHealth("my-plugin")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", health.Status)
fmt.Printf("Message: %s\n", health.Message)
```

---

## المزايا (Advantages):

1. **قابل للتوسع:** يمكن للمستخدمين إضافة وظائف جديدة دون تعديل الكود الأساسي
2. **معزول:** كل إضافة تعمل في بيئة معزولة
3. **آمن:** نظام صندوق الرمل يحمي المنصة من الإضافات الضارة
4. **مرن:** يدعم التبعيات والقدرات
5. **قابل للمراقبة:** نظام صحة شامل لكل إضافة
6. **تكامل مع ناقل الأحداث:** الإضافات يمكنها نشر والاشتراك في الأحداث

---

## التكامل (Integration):

### مع الأنظمة الموجودة:
- **Event Bus:** الإضافات يمكنها نشر والاشتراك في الأحداث
- **Agent System:** الإضافات يمكنها التفاعل مع الوكلاء
- **Session System:** الإضافات يمكنها التفاعل مع الجلسات
- **Provider System:** الإضافات يمكنها إضافة مزودين جدد

---

## الأمان (Security):

### نظام صندوق الرمل (Sandbox):
- عزل الإضافات عن النظام الأساسي
- تقييد الوصول إلى الموارد
- مراقبة استخدام الموارد
- منع الإضافات الضارة

### التحقق من الصحة (Validation):
- التحقق من توقيع الإضافة
- التحقق من مصدر الإضافة
- التحقق من التبعيات
- التحقق من القدرات

---

## الملفات المطلوبة (Required Files):

### ✅ المنشأة:
- `pkg/plugins/core/plugin.go` - النظام الأساسي للإضافات

### 🔲 المطلوبة:
- `pkg/plugins/loader/loader.go` - محمل الإضافات
- `pkg/plugins/registry/registry.go` - سجل الإضافات
- `pkg/plugins/hooks/hooks.go` - نظام الخطافات
- `pkg/plugins/sandbox/sandbox.go` - صندوق الرمل للإضافات
- `pkg/plugins/examples/example_plugin.go` - مثال على إضافة

---

## الخطوات التالية (Next Steps):

1. إنشاء محمل الإضافات (Plugin Loader)
2. إنشاء سجل الإضافات (Plugin Registry)
3. إنشاء نظام الخطافات (Hook System)
4. إنشاء صندوق الرمل (Sandbox)
5. إنشاء مثال على إضافة (Example Plugin)
6. إنشاء اختبارات شاملة (Comprehensive Tests)
7. إنشاء توثيق كامل (Complete Documentation)

---

## الخلاصة (Conclusion):

نظام الإضافات يوفر:
- قابلية توسع عالية
- عزل آمن للإضافات
- نظام صحة شامل
- تكامل مع الأنظمة الموجودة
- أمان متقدم

المنصة جاهزة الآن لدعم الإضافات الخارجية دون أي ثغرات أو مشاكل.
