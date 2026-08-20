package helper

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidationError representasi error validasi
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// ValidateStruct validasi struct dengan tag validate
func ValidateStruct(s interface{}) []ValidationError {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var errors []ValidationError
	for _, err := range err.(validator.ValidationErrors) {
		errors = append(errors, ValidationError{
			Field:   err.Field(),
			Tag:     err.Tag(),
			Message: formatMessage(err),
		})
	}
	return errors
}

// formatMessage format pesan error jadi user-friendly
func formatMessage(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s wajib diisi", field)
	case "email":
		return fmt.Sprintf("%s harus berformat email", field)
	case "min":
		return fmt.Sprintf("%s minimal %s karakter", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s maksimal %s karakter", field, fe.Param())
	case "len":
		return fmt.Sprintf("%s harus %s karakter", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s harus lebih besar atau sama dengan %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s harus lebih kecil atau sama dengan %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("%s harus lebih besar dari %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s harus lebih kecil dari %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s harus salah satu dari: %s", field, fe.Param())
	case "url":
		return fmt.Sprintf("%s harus berformat URL", field)
	case "uuid":
		return fmt.Sprintf("%s harus berformat UUID", field)
	case "alpha":
		return fmt.Sprintf("%s hanya boleh huruf", field)
	case "alphanum":
		return fmt.Sprintf("%s hanya boleh huruf dan angka", field)
	case "numeric":
		return fmt.Sprintf("%s harus berupa angka", field)
	default:
		return fmt.Sprintf("%s tidak valid (%s)", field, fe.Tag())
	}
}

// SanitizeString menghapus karakter berbahaya dari string
func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}

// SanitizeHTML menghapus semua tag HTML
func SanitizeHTML(s string) string {
	s = SanitizeString(s)
	// Simple tag removal
	for {
		start := strings.Index(s, "<")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], ">")
		if end == -1 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	return s
}

// IsEmpty mengecek apakah string kosong setelah trim
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotEmpty mengecek apakah string tidak kosong
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}
