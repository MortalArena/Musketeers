package sandbox

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// SandboxConfig إعدادات الصندوق الرملي
type SandboxConfig struct {
	MemoryLimitPages uint32 // 1 page = 64KB. 800 pages ≈ 50MB حد أقصى للذاكرة
	WasmBinary       []byte
}

// Executor ينفذ أكواد WASM في بيئة معزولة تماماً
type Executor struct {
	runtime wazero.Runtime
}

// NewExecutor ينشئ بيئة تشغيل WASM جديدة وآمنة
func NewExecutor(ctx context.Context) (*Executor, error) {
	// ✅ تطبيق حد الذاكرة على مستوى Runtime
	rConfig := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(800) // 800 pages ≈ 50MB

	r := wazero.NewRuntimeWithConfig(ctx, rConfig)

	// إضافة دعم WASI الأساسي (معزول)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	return &Executor{runtime: r}, nil
}

// Execute ينفذ وحدة WASM مع فرض حدود الموارد الصارمة
func (e *Executor) Execute(ctx context.Context, config SandboxConfig, funcName string, args ...uint64) (uint64, error) {
	compiled, err := e.runtime.CompileModule(ctx, config.WasmBinary)
	if err != nil {
		return 0, fmt.Errorf("failed to compile wasm module: %w", err)
	}

	// منع الوصول للملفات والشبكة
	modConfig := wazero.NewModuleConfig().
		WithName("isolated-plugin").
		WithFSConfig(wazero.NewFSConfig())

	mod, err := e.runtime.InstantiateModule(ctx, compiled, modConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to instantiate wasm module: %w", err)
	}
	defer mod.Close(ctx)

	// استدعاء الدالة المطلوبة
	results, err := mod.ExportedFunction(funcName).Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to call wasm function '%s': %w", funcName, err)
	}

	if len(results) == 0 {
		return 0, nil
	}
	return results[0], nil
}

// Close يغلق بيئة التشغيل ويحرر الذاكرة بالكامل
func (e *Executor) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}
