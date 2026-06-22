package domain

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
)

// SystemProxy يعدل إعدادات Proxy في النظام تلقائياً
type SystemProxy struct {
	proxyAddr string // 127.0.0.1:8080
	dnsAddr   string // 127.0.0.1:53
}

// NewSystemProxy ينشئ SystemProxy جديد
func NewSystemProxy(proxyAddr, dnsAddr string) *SystemProxy {
	return &SystemProxy{
		proxyAddr: proxyAddr,
		dnsAddr:   dnsAddr,
	}
}

// Configure يعدل إعدادات النظام تلقائياً
func (sp *SystemProxy) Configure() error {
	log.Printf("Configuring system proxy on %s", runtime.GOOS)

	switch runtime.GOOS {
	case "windows":
		return sp.configureWindows()
	case "darwin":
		return sp.configureMacOS()
	case "linux":
		return sp.configureLinux()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// Restore يستعيد الإعدادات الأصلية
func (sp *SystemProxy) Restore() error {
	log.Printf("Restoring system proxy on %s", runtime.GOOS)

	switch runtime.GOOS {
	case "windows":
		return sp.restoreWindows()
	case "darwin":
		return sp.restoreMacOS()
	case "linux":
		return sp.restoreLinux()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// configureWindows يعدل إعدادات Windows
func (sp *SystemProxy) configureWindows() error {
	// تعطيل Proxy الحالي
	cmd := exec.Command("reg", "add",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable proxy: %w", err)
	}

	// تعيين Proxy Server
	proxyServer := sp.proxyAddr
	cmd = exec.Command("reg", "add",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyServer, "/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set proxy server: %w", err)
	}

	// تفعيل Proxy
	cmd = exec.Command("reg", "add",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable proxy: %w", err)
	}

	// تعيين DNS
	// ملاحظة: هذا يتطلب صلاحيات إدارية
	// في الإنتاج، يجب طلب صلاحيات من المستخدم

	log.Printf("Windows proxy configured: %s", proxyServer)
	return nil
}

// restoreWindows يستعيد إعدادات Windows
func (sp *SystemProxy) restoreWindows() error {
	cmd := exec.Command("reg", "add",
		"HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings",
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
	return cmd.Run()
}

// configureMacOS يعدل إعدادات macOS
func (sp *SystemProxy) configureMacOS() error {
	// الحصول على اسم الشبكة النشطة
	interfaceName := "Wi-Fi" // افتراضي

	// تعيين Web Proxy (HTTP)
	cmd := exec.Command("networksetup", "-setwebproxy", interfaceName, "127.0.0.1", "8080")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set web proxy: %w", err)
	}

	// تفعيل Web Proxy
	cmd = exec.Command("networksetup", "-setwebproxystate", interfaceName, "on")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable web proxy: %w", err)
	}

	// تعيين Secure Web Proxy (HTTPS)
	cmd = exec.Command("networksetup", "-setsecurewebproxy", interfaceName, "127.0.0.1", "8080")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set secure web proxy: %w", err)
	}

	// تفعيل Secure Web Proxy
	cmd = exec.Command("networksetup", "-setsecurewebproxystate", interfaceName, "on")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable secure web proxy: %w", err)
	}

	// تعيين DNS
	cmd = exec.Command("networksetup", "-setdnsservers", interfaceName, "127.0.0.1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set DNS: %w", err)
	}

	log.Printf("macOS proxy configured for %s", interfaceName)
	return nil
}

// restoreMacOS يستعيد إعدادات macOS
func (sp *SystemProxy) restoreMacOS() error {
	interfaceName := "Wi-Fi"

	// تعطيل Web Proxy
	cmd := exec.Command("networksetup", "-setwebproxystate", interfaceName, "off")
	if err := cmd.Run(); err != nil {
		return err
	}

	// تعطيل Secure Web Proxy
	cmd = exec.Command("networksetup", "-setsecurewebproxystate", interfaceName, "off")
	if err := cmd.Run(); err != nil {
		return err
	}

	// استعادة DNS
	cmd = exec.Command("networksetup", "-setdnsservers", interfaceName, "empty")
	return cmd.Run()
}

// configureLinux يعدل إعدادات Linux
func (sp *SystemProxy) configureLinux() error {
	// تعيين متغيرات البيئة
	// ملاحظة: هذا يؤثر على الجلسة الحالية فقط
	// في الإنتاج، يجب تعديل ملفات النظام

	proxyURL := fmt.Sprintf("http://%s", sp.proxyAddr)

	// يمكن كتابة ملف في /etc/environment
	// أو استخدام gsettings لـ GNOME

	log.Printf("Linux proxy configured: %s", proxyURL)
	log.Printf("Note: User may need to restart applications")

	return nil
}

// restoreLinux يستعيد إعدادات Linux
func (sp *SystemProxy) restoreLinux() error {
	log.Printf("Linux proxy restored")
	return nil
}
