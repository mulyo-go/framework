package helper

import (
	"crypto/rand"
	"strings"
)

// StrUpper mengubah string ke uppercase.
func StrUpper(s string) string {
	return strings.ToUpper(s)
}

// StrLower mengubah string ke lowercase.
func StrLower(s string) string {
	return strings.ToLower(s)
}

// StrTitle mengubah string ke title case.
func StrTitle(s string) string {
	return strings.Title(s) // deprecated but simple
}

// StrUcfirst huruf pertama uppercase.
func StrUcfirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// StrKebab mengubah string ke kebab-case.
func StrKebab(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "-")
}

// StrSnake mengubah string ke snake_case.
func StrSnake(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "_")
}

// StrCamel mengubah string ke camelCase.
func StrCamel(s string) string {
	parts := strings.Fields(strings.ToLower(s))
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

// StrStudly mengubah string ke StudlyCase (PascalCase).
func StrStudly(s string) string {
	parts := strings.Fields(strings.ToLower(s))
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

// StrSlug membuat URL slug dari string.
func StrSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var result []rune
	prevDash := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			result = append(result, r)
			prevDash = false
		} else if r == ' ' || r == '-' || r == '_' {
			if !prevDash && len(result) > 0 {
				result = append(result, '-')
				prevDash = true
			}
		}
	}
	s2 := string(result)
	s2 = strings.TrimRight(s2, "-")
	return s2
}

// StrLimit memotong string ke panjang tertentu dengan suffix.
func StrLimit(s string, length int, suffix string) string {
	if len(s) <= length {
		return s
	}
	if suffix == "" {
		suffix = "..."
	}
	return s[:length] + suffix
}

// StrContains mengecek apakah string mengandung substring.
func StrContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// StrStartsWith mengecek apakah string diawali prefix.
func StrStartsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// StrEndsWith mengecek apakah string diakhiri suffix.
func StrEndsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// StrReplace mengganti semua occurrence.
func StrReplace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// StrTrim menghapus whitespace di kedua ujung.
func StrTrim(s string) string {
	return strings.TrimSpace(s)
}

// StrReverse membalik string.
func StrReverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// StrWordCount menghitung jumlah kata.
func StrWordCount(s string) int {
	return len(strings.Fields(s))
}

// StrSubstr mengambil substring.
func StrSubstr(s string, start, length int) string {
	runes := []rune(s)
	if start < 0 {
		start = len(runes) + start
	}
	if start < 0 {
		start = 0
	}
	if start >= len(runes) {
		return ""
	}
	end := start + length
	if end > len(runes) || length < 0 {
		end = len(runes)
	}
	return string(runes[start:end])
}

// StrSquish menghapus whitespace berlebih.
func StrSquish(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// StrHeadline mengubah string ke headline (Title Case dengan spasi).
func StrHeadline(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}

// StrIs mengecek apakah string match dengan pattern (support * wildcard).
func StrIs(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}
	return s == pattern
}

// StrPlural sederhana (tambah 's').
func StrPlural(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

// StrSingular sederhana (hapus 's').
func StrSingular(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") || strings.HasSuffix(s, "zes") || strings.HasSuffix(s, "ches") || strings.HasSuffix(s, "shes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

// StrPadLeft menambah padding di kiri.
func StrPadLeft(s string, length int, pad string) string {
	if len(s) >= length {
		return s
	}
	if pad == "" {
		pad = " "
	}
	return strings.Repeat(pad, length-len(s)) + s
}

// StrPadRight menambah padding di kanan.
func StrPadRight(s string, length int, pad string) string {
	if len(s) >= length {
		return s
	}
	if pad == "" {
		pad = " "
	}
	return s + strings.Repeat(pad, length-len(s))
}

// StrRandom menghasilkan string random sepanjang length menggunakan crypto/rand.
func StrRandom(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
