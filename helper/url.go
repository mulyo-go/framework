package helper

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var basePath string

func init() {
	basePath, _ = os.Getwd()
}

// BasePath mengembalikan path ke root project
func BasePath(paths ...string) string {
	if len(paths) == 0 {
		return basePath
	}
	args := append([]string{basePath}, paths...)
	return filepath.Join(args...)
}

// StoragePath mengembalikan path ke folder storage
func StoragePath(paths ...string) string {
	args := append([]string{"storage"}, paths...)
	return BasePath(args...)
}

// PublicPath mengembalikan path ke folder public
func PublicPath(paths ...string) string {
	args := append([]string{"Web", "Public"}, paths...)
	return BasePath(args...)
}

// ConfigPath mengembalikan path ke folder config
func ConfigPath(paths ...string) string {
	args := append([]string{"config"}, paths...)
	return BasePath(args...)
}

// TemplatePath mengembalikan path ke folder template
func TemplatePath(paths ...string) string {
	args := append([]string{"Template"}, paths...)
	return BasePath(args...)
}

// Asset menghasilkan URL ke asset di public folder
func Asset(path string) string {
	path = strings.TrimPrefix(path, "/")
	return "/public/" + path
}

// URL menghasilkan URL absolut
func URL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// URLFile menghasilkan URL file dengan cache busting MD5
// Contoh: URLFile("/assets/css/style.bundle.css") → "/public/assets/css/style.bundle.css?v=a1b2c3d4"
func URLFile(path string) string {
	path = strings.TrimPrefix(path, "/")
	publicPath := filepath.Join(basePath, "Web", "Public", path)

	hash := fileMD5(publicPath)
	if hash == "" {
		return "/public/" + path
	}
	return fmt.Sprintf("/public/%s?v=%s", path, hash)
}

// fileMD5 menghitung MD5 hash dari file, return 8 char pertama atau "" jika gagal
func fileMD5(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash[:4])
}

// Route menghasilkan URL berdasarkan module/controller/action
func Route(module, controller, action string) string {
	return "/" + StrKebab(module) + "/" + StrKebab(controller) + "/" + StrKebab(action)
}
