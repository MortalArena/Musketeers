package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// [WHY] ToolExecutor ينفذ الأدوات مع حدود أمان ونظام صلاحيات
// [HOW] يفرض حدود على استدعاءات الأدوات وحجم الملفات والمسارات ويدمج مع ToolRegistry
// [SAFETY] يمنع الحلقات اللانهائية والوصول غير المصرح به
type ToolExecutor struct {
	// حدود الأمان
	MaxToolCallsPerTask int    // [WHY] الحد الأقصى لاستدعاءات الأدوات (50)
	MaxFileSizeBytes    int64  // [WHY] الحد الأقصى لحجم الملف (10MB)
	AllowedBasePath     string // [WHY] المسار المسموح (مجلد الجلسة)

	// حالة التنفيذ
	taskCallCount map[string]int // [WHY] عداد استدعاءات الأدوات لكل مهمة
	taskCallMu    sync.RWMutex   // [SAFETY] لحماية العدادات

	// مدير أقفال الملفات
	fileLockManager *FileLockManager // [WHY] يدير أقفال الملفات لمنع التعارضات

	// [NEW] Registry + Role للتحكم بالصلاحيات
	registry   *ToolRegistry // [WHY] سجل الأدوات للتحقق من الصلاحيات
	agentRole  AgentRole     // [WHY] دور الوكيل الحالي

	// Logger
	logger *zap.Logger
}

// [WHY] NewToolExecutor ينشئ منفذ أدوات جديد بدون registry
func NewToolExecutor(allowedBasePath string, logger *zap.Logger) *ToolExecutor {
	if allowedBasePath == "" {
		allowedBasePath = "."
	}

	return &ToolExecutor{
		MaxToolCallsPerTask: 50,
		MaxFileSizeBytes:    10 * 1024 * 1024,
		AllowedBasePath:     allowedBasePath,
		taskCallCount:       make(map[string]int),
		fileLockManager:     NewFileLockManager("", logger),
		agentRole:           RoleRegular,
		logger:              logger,
	}
}

// [WHY] NewToolExecutorWithRegistry ينشئ منفذ أدوات مع registry ونظام صلاحيات
// [HOW] يهيئ الحدود والعدادات ويسجل registry ودور الوكيل
func NewToolExecutorWithRegistry(allowedBasePath string, registry *ToolRegistry, role AgentRole, logger *zap.Logger) *ToolExecutor {
	if allowedBasePath == "" {
		allowedBasePath = "."
	}
	if registry == nil {
		registry = NewToolRegistry()
	}
	if role == "" {
		role = RoleRegular
	}

	return &ToolExecutor{
		MaxToolCallsPerTask: 50,
		MaxFileSizeBytes:    10 * 1024 * 1024,
		AllowedBasePath:     allowedBasePath,
		taskCallCount:       make(map[string]int),
		fileLockManager:     NewFileLockManager("", logger),
		registry:            registry,
		agentRole:           role,
		logger:              logger,
	}
}

// SetRegistry يضبط سجل الأدوات
func (te *ToolExecutor) SetRegistry(registry *ToolRegistry) {
	te.registry = registry
}

// SetAgentRole يضبط دور الوكيل
func (te *ToolExecutor) SetAgentRole(role AgentRole) {
	te.agentRole = role
}

// GetAgentRole يعيد دور الوكيل الحالي
func (te *ToolExecutor) GetAgentRole() AgentRole {
	return te.agentRole
}

// GetRegistry يعيد سجل الأدوات
func (te *ToolExecutor) GetRegistry() *ToolRegistry {
	return te.registry
}

// [WHY] ExecuteTool ينفذ أداة مع نظام صلاحيات كامل
// [HOW] 1. فحص العداد 2. فحص الصلاحية 3. فحص المسارات 4. أقفال الملفات 5. التنفيذ
// [SAFETY] ثلاث طبقات أمان: عداد، صلاحية، مسار
func (te *ToolExecutor) ExecuteTool(ctx context.Context, taskID, toolName string, params map[string]interface{}) (interface{}, error) {
	// [SAFETY] الطبقة 1: فحص العدادات
	if !te.checkToolCallLimit(taskID) {
		return nil, fmt.Errorf("تجاوز الحد الأقصى لاستدعاءات الأدوات: %d", te.MaxToolCallsPerTask)
	}
	te.incrementToolCallCount(taskID)

	// [SAFETY] الطبقة 2: فحص الصلاحية عبر registry
	if te.registry != nil {
		def, err := te.registry.Get(toolName)
		if err != nil {
			return nil, fmt.Errorf("أداة غير موجودة: %s", toolName)
		}
		if !def.HasPermission(te.agentRole) {
			return nil, fmt.Errorf("صلاحية مرفوضة: الدور %s لا يمكنه استخدام أداة %s", te.agentRole, toolName)
		}
	}

	// [SAFETY] الطبقة 3: فحص المسارات للملفات
	if toolName == "read_file" || toolName == "write_file" || toolName == "file_list" || toolName == "file_delete" {
		filePath, ok := params["path"].(string)
		if !ok {
			return nil, fmt.Errorf("المعامل path مطلوب")
		}
		if !te.isPathAllowed(filePath) {
			return nil, fmt.Errorf("المسار غير مسموح: %s", filePath)
		}
		if toolName == "read_file" || toolName == "file_list" {
			if err := te.checkFileSize(filePath); err != nil {
				if toolName == "file_list" {
					// file_list يتجاهل خطأ الحجم
				} else {
					return nil, err
				}
			}
		}

		// [SAFETY] أقفال الملفات للكتابة والحذف
		if toolName == "write_file" || toolName == "file_delete" {
			absPath := filepath.Join(te.AllowedBasePath, filePath)
			if err := te.fileLockManager.Lock(ctx, absPath, taskID); err != nil {
				return nil, fmt.Errorf("فشل الحصول على قفل الملف: %w", err)
			}
			defer te.fileLockManager.Unlock(absPath)
		}
	}

	// [HOW] تنفيذ الأداة
	result, err := te.executeToolInternal(ctx, toolName, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// [WHY] executeToolInternal ينفذ الأداة فعلياً
// [HOW] يحاول أولاً من الأدوات المدمجة، ثم من registry
// [SAFETY] يستخدم context للإلغاء
func (te *ToolExecutor) executeToolInternal(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// [HOW] الأدوات المدمجة (عمليات ملفات + HTTP)
	switch toolName {
	case "read_file":
		return te.readFile(ctx, params)
	case "write_file":
		return te.writeFile(ctx, params)
	case "file_list":
		return te.listFiles(ctx, params)
	case "file_delete":
		return te.deleteFile(ctx, params)
	case "http_request":
		return te.httpRequest(ctx, params)
	}

	// [HOW] إذا كانت الأداة مسجلة في registry، ننفذها عبر handler
	if te.registry != nil {
		result, err := te.registry.Execute(ctx, te.agentRole, toolName, params)
		if err == nil {
			return result, nil
		}
		// إذا كان الخطأ "tool not found"، نكمل للرسالة الافتراضية
		if !strings.Contains(err.Error(), "tool not found") {
			return nil, err
		}
	}

	return nil, fmt.Errorf("أداة غير مدعومة: %s", toolName)
}

// ============================================================
// أدوات الملفات
// ============================================================

func (te *ToolExecutor) readFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}
	absPath := filepath.Join(te.AllowedBasePath, path)
	data, err := te.readFileWithContext(ctx, absPath)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": string(data),
		"path":    path,
	}, nil
}

func (te *ToolExecutor) readFileWithContext(ctx context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, 4096)
	var result []byte
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if n == 0 {
				return result, nil
			}
			result = append(result, buf[:n]...)
		}
	}
}

func (te *ToolExecutor) writeFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل content مطلوب")
	}
	absPath := filepath.Join(te.AllowedBasePath, path)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("فشل إنشاء المجلد: %w", err)
	}

	err := te.writeFileWithContext(ctx, absPath, []byte(content))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

func (te *ToolExecutor) writeFileWithContext(ctx context.Context, path string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	chunkSize := 32768
	for i := 0; i < len(data); i += chunkSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			end := i + chunkSize
			if end > len(data) {
				end = len(data)
			}
			if _, err := file.Write(data[i:end]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (te *ToolExecutor) listFiles(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, _ := params["path"].(string)
	if path == "" {
		path = "."
	}
	absPath := filepath.Join(te.AllowedBasePath, path)

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("فشل قراءة المجلد: %w", err)
	}

	files := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"name":  entry.Name(),
			"dir":   entry.IsDir(),
			"size":  info.Size(),
			"mtime": info.ModTime(),
		})
	}

	return map[string]interface{}{
		"path":  path,
		"files": files,
		"count": len(files),
	}, nil
}

func (te *ToolExecutor) deleteFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}
	absPath := filepath.Join(te.AllowedBasePath, path)

	if err := os.Remove(absPath); err != nil {
		return nil, fmt.Errorf("فشل حذف الملف: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// ============================================================
// أمان HTTP - SSRF Protection
// ============================================================

func isPrivateURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	if parsed.Scheme != "https" {
		return true
	}
	host := parsed.Hostname()

	blocked := []string{
		"localhost", "127.", "10.", "192.168.", "172.16.",
		"169.254.", "::1", "[::1]", "0.0.0.0",
	}
	for _, b := range blocked {
		if strings.HasPrefix(host, b) {
			return true
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}

	metadataEndpoints := []string{
		"metadata.google.internal",
		"169.254.169.254",
		"metadata.azure.net",
	}
	for _, endpoint := range metadataEndpoints {
		if host == endpoint {
			return true
		}
	}

	return false
}

func (te *ToolExecutor) httpRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل url مطلوب")
	}
	if isPrivateURL(url) {
		return nil, fmt.Errorf("SSRF: private/internal URLs not allowed: %s", url)
	}

	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if isPrivateURL(req.URL.String()) {
				return fmt.Errorf("redirect to private URL not allowed: %s", req.URL.String())
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(body),
	}, nil
}

// ============================================================
// أدوات المساعدة
// ============================================================

func (te *ToolExecutor) checkToolCallLimit(taskID string) bool {
	te.taskCallMu.RLock()
	defer te.taskCallMu.RUnlock()
	count, exists := te.taskCallCount[taskID]
	if !exists {
		return true
	}
	return count < te.MaxToolCallsPerTask
}

func (te *ToolExecutor) incrementToolCallCount(taskID string) {
	te.taskCallMu.Lock()
	defer te.taskCallMu.Unlock()
	te.taskCallCount[taskID]++
}

func (te *ToolExecutor) isPathAllowed(path string) bool {
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) {
		return false
	}
	if strings.Contains(cleanPath, "..") {
		return false
	}
	absPath, err := filepath.Abs(filepath.Join(te.AllowedBasePath, cleanPath))
	if err != nil {
		return false
	}
	allowedAbsPath, err := filepath.Abs(te.AllowedBasePath)
	if err != nil {
		return false
	}
	// المسموح: المسار يساوي تماماً المسار الأساسي أو يبدأ به + separator
	if absPath == allowedAbsPath {
		return true
	}
	return strings.HasPrefix(absPath, allowedAbsPath+string(filepath.Separator))
}

func (te *ToolExecutor) checkFileSize(path string) error {
	absPath := filepath.Join(te.AllowedBasePath, path)
	info, err := os.Stat(absPath)
	if err != nil {
		return nil // الملف غير موجود
	}
	if info.Size() > te.MaxFileSizeBytes {
		return fmt.Errorf("حجم الملف يتجاوز الحد الأقصى: %d bytes", info.Size())
	}
	return nil
}

// ResetTaskCallCount يصفر عداد مهمة
func (te *ToolExecutor) ResetTaskCallCount(taskID string) {
	te.taskCallMu.Lock()
	defer te.taskCallMu.Unlock()
	delete(te.taskCallCount, taskID)
}

// GetTaskCallCount يحصل على عداد مهمة
func (te *ToolExecutor) GetTaskCallCount(taskID string) int {
	te.taskCallMu.RLock()
	defer te.taskCallMu.RUnlock()
	return te.taskCallCount[taskID]
}

// GetAvailableTools يعيد قائمة الأدوات المسموحة لدور الوكيل الحالي
func (te *ToolExecutor) GetAvailableTools() []ToolInfo {
	if te.registry == nil {
		return nil
	}
	return te.registry.GetToolsByRole(te.agentRole)
}
