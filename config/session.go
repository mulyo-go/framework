package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SessionManager struct {
	cookieName string
	ttl        time.Duration
}

var Session *SessionManager

func InitSession() {
	name := Env("SESSION_COOKIE_NAME", "sico_session")
	ttlStr := Env("SESSION_TTL_SECONDS", "3600")
	ttlSeconds := parseInt(ttlStr, 3600)
	Session = &SessionManager{
		cookieName: name,
		ttl:        time.Duration(ttlSeconds) * time.Second,
	}
}

func (s *SessionManager) CookieName() string {
	return s.cookieName
}

func (s *SessionManager) getID(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(s.cookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	id := newSessionID()
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return id
}

func (s *SessionManager) Get(r *http.Request, w http.ResponseWriter) map[string]interface{} {
	id := s.getID(w, r)
	if SessionCache == nil {
		return map[string]interface{}{}
	}
	key := "session:" + id
	v, ok := SessionCache.Get(key)
	if !ok {
		return map[string]interface{}{}
	}
	data, ok := v.(string)
	if !ok {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

func (s *SessionManager) Save(r *http.Request, w http.ResponseWriter, data map[string]interface{}) {
	id := s.getID(w, r)
	if SessionCache == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	key := "session:" + id
	_ = SessionCache.Set(key, string(b), s.ttl)
}

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func parseInt(s string, def int) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n == 0 {
		return def
	}
	return n
}

// GinSession membungkus SessionManager agar bisa dipakai dengan *gin.Context
type GinSession struct {
	mgr  *SessionManager
	data map[string]interface{}
	w    http.ResponseWriter
	r    *http.Request
}

// StartSession mengembalikan session wrapper untuk Gin
func StartSession(ctx *gin.Context) *GinSession {
	if Session == nil {
		return &GinSession{data: map[string]interface{}{}}
	}
	data := Session.Get(ctx.Request, ctx.Writer)
	return &GinSession{
		mgr:  Session,
		data: data,
		w:    ctx.Writer,
		r:    ctx.Request,
	}
}

func (s *GinSession) Get(key string) interface{} {
	if s.data == nil {
		return nil
	}
	return s.data[key]
}

func (s *GinSession) Set(key string, value interface{}) {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}
	s.data[key] = value
	if s.mgr != nil {
		s.mgr.Save(s.r, s.w, s.data)
	}
}

func (s *GinSession) Destroy() {
	if s.data == nil {
		return
	}
	s.data = make(map[string]interface{})
	if s.mgr != nil {
		s.mgr.Save(s.r, s.w, s.data)
	}
}

func (s *GinSession) GetAll() map[string]interface{} {
	if s.data == nil {
		return map[string]interface{}{}
	}
	return s.data
}

// --- Flash Messages ---

// SetFlash menyimpan flash message (one-time, auto-hapus setelah dibaca)
func (s *GinSession) SetFlash(key string, value interface{}) {
	s.Set("flash:"+key, value)
}

// GetFlash mengambil dan menghapus flash message
func (s *GinSession) GetFlash(key string) interface{} {
	if s.data == nil {
		return nil
	}
	val := s.data["flash:"+key]
	if val != nil {
		delete(s.data, "flash:"+key)
		s.mgr.Save(s.r, s.w, s.data)
	}
	return val
}

// GetFlashes mengambil semua flash messages sekaligus (untuk template)
func (s *GinSession) GetFlashes() map[string]interface{} {
	result := make(map[string]interface{})
	if s.data == nil {
		return result
	}
	keysToDelete := []string{}
	for k, v := range s.data {
		if len(k) > 6 && k[:6] == "flash:" {
			result[k[6:]] = v
			keysToDelete = append(keysToDelete, k)
		}
	}
	for _, k := range keysToDelete {
		delete(s.data, k)
	}
	if len(keysToDelete) > 0 {
		s.mgr.Save(s.r, s.w, s.data)
	}
	return result
}

// FlashSuccess convenience method
func (s *GinSession) FlashSuccess(msg string) {
	s.SetFlash("success", msg)
}

// FlashError convenience method
func (s *GinSession) FlashError(msg string) {
	s.SetFlash("error", msg)
}

// FlashWarning convenience method
func (s *GinSession) FlashWarning(msg string) {
	s.SetFlash("warning", msg)
}

// FlashInfo convenience method
func (s *GinSession) FlashInfo(msg string) {
	s.SetFlash("info", msg)
}
