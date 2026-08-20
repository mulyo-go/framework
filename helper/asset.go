package helper

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

var (
	assetURLs   map[string]string
	assetOnce   sync.Once
	assetFS     fs.FS
	assetPrefix string // "Web/Public" or "Web/Public/assets"
)

// SetAssetFS menyetel FS dan prefix untuk precompute asset URLs
func SetAssetFS(f fs.FS, prefix string) {
	assetFS = f
	assetPrefix = prefix
}

// BuildAssetURLs membangun map path → URL dengan cache busting MD5
// Dipanggil sekali di startup. Hasilnya bisa diakses dari template.
func BuildAssetURLs() map[string]string {
	assetOnce.Do(func() {
		assetURLs = make(map[string]string)
		if assetFS == nil {
			return
		}

		_ = fs.WalkDir(assetFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			// Hanya process file asset yang umum
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".css", ".js", ".svg", ".png", ".jpg", ".jpeg", ".gif", ".ico",
				".woff", ".woff2", ".ttf", ".eot", ".map", ".json":
				// proses
			default:
				return nil
			}

			body, err := fs.ReadFile(assetFS, path)
			if err != nil {
				return nil
			}

			// Build key: gunakan slash sebagai separator
			key := filepath.ToSlash(path)

			// MD5 untuk cache busting
			hash := md5.Sum(body)
			v := fmt.Sprintf("%x", hash[:4])

			// URL: /public/assets/...?v=hash
			// path sudah relatif dari Web/Public, jadi tinggal tambah /public/
			url := "/public/" + key + "?v=" + v
			assetURLs[key] = url

			return nil
		})
	})
	return assetURLs
}

// GetAssetURL mengembalikan URL asset dengan cache busting
// Contoh: GetAssetURL("assets/media/logos/favicon.ico") → "/public/assets/media/logos/favicon.ico?v=abc123"
func GetAssetURL(path string) string {
	if assetURLs == nil {
		BuildAssetURLs()
	}

	path = strings.TrimPrefix(path, "/")

	// Coba exact match
	if url, ok := assetURLs[path]; ok {
		return url
	}

	// Fallback: tanpa prefix
	if url, ok := assetURLs["assets/"+path]; ok {
		return url
	}

	// Tidak ditemukan, return path asli
	return "/public/" + path
}

// AssetURLs mengembalikan map asset URLs untuk digunakan di templates
func AssetURLs() map[string]string {
	if assetURLs == nil {
		BuildAssetURLs()
	}
	return assetURLs
}
