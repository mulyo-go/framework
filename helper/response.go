package helper

import (
	"net/http"

	"github.com/mulyo-go/framework/config"

	"github.com/gin-gonic/gin"
)

// --- JSON Response ---

// Success response (200)
func ResponseSuccess(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

// Created response (201)
func ResponseCreated(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

// Error response (generic)
func ResponseError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"status":  "error",
		"code":    code,
		"message": message,
	})
}

// BadRequest response (400)
func ResponseBadRequest(c *gin.Context, message string) {
	ResponseError(c, http.StatusBadRequest, message)
}

// Unauthorized response (401)
func ResponseUnauthorized(c *gin.Context, message string) {
	ResponseError(c, http.StatusUnauthorized, message)
}

// Forbidden response (403)
func ResponseForbidden(c *gin.Context, message string) {
	ResponseError(c, http.StatusForbidden, message)
}

// NotFound response (404)
func ResponseNotFound(c *gin.Context, message string) {
	ResponseError(c, http.StatusNotFound, message)
}

// ServerError response (500)
func ResponseServerError(c *gin.Context, message string) {
	ResponseError(c, http.StatusInternalServerError, message)
}

// ValidationError response (422) dengan detail errors
func ResponseValidation(c *gin.Context, errors map[string]string) {
	c.JSON(http.StatusUnprocessableEntity, gin.H{
		"status": "error",
		"code":   422,
		"message": "Validasi gagal",
		"errors": errors,
	})
}

// Paginated response
func ResponsePaginated(c *gin.Context, message string, data interface{}, page, perPage, total int) {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": message,
		"data":    data,
		"pagination": gin.H{
			"current_page": page,
			"per_page":     perPage,
			"total_items":  total,
			"total_pages":  totalPages,
		},
	})
}

// --- Redirect with Flash ---

// Redirect dengan flash message
func ResponseRedirect(c *gin.Context, url string, flashType string, message string) {
	if flashType != "" && message != "" {
		sess := config.StartSession(c)
		switch flashType {
		case "success":
			sess.FlashSuccess(message)
		case "error":
			sess.FlashError(message)
		case "warning":
			sess.FlashWarning(message)
		case "info":
			sess.FlashInfo(message)
		}
	}
	c.Redirect(http.StatusFound, url)
	c.Abort()
}

// RedirectBack - redirect ke halaman sebelumnya
func ResponseRedirectBack(c *gin.Context, defaultUrl string, flashType string, message string) {
	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = defaultUrl
	}
	ResponseRedirect(c, referer, flashType, message)
}


// Aliases for convenience
var (
	JsonSuccess    = ResponseSuccess
	JsonCreated    = ResponseCreated
	JsonError      = ResponseError
	JsonValidation = ResponseValidation
	Success        = ResponseSuccess
	Created        = ResponseCreated
	Error          = ResponseError
)
