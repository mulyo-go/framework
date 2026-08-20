package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type RequestInfo struct {
	Method     string
	URL        string
	Module     string
	Controller string
	Action     string
}

type requestInfoKey struct{}

func WithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, info)
}

func GetRequestInfo(ctx context.Context) (RequestInfo, bool) {
	val := ctx.Value(requestInfoKey{})
	if val == nil {
		return RequestInfo{}, false
	}
	info, ok := val.(RequestInfo)
	return info, ok
}

// Konfigurasi Log Aplikasi (Info/Access)
var AppLogConfig = struct {
	Enabled    bool
	Filename   string
	MaxSize    int // megabytes
	MaxBackups int
	MaxAge     int // days
	Compress   bool
}{
	Enabled:    true,
	Filename:   "storage/logs/app.log",
	MaxSize:    10, // 10MB
	MaxBackups: 5,
	MaxAge:     30,
	Compress:   true,
}

// Konfigurasi Error Log (System Error, Panic, DB Error)
var ErrorLogConfig = struct {
	Enabled    bool
	Filename   string
	MaxSize    int // megabytes
	MaxBackups int
	MaxAge     int // days
	Compress   bool
}{
	Enabled:    true,
	Filename:   "storage/logs/error.log",
	MaxSize:    10, // 10MB
	MaxBackups: 5,
	MaxAge:     30,
	Compress:   true,
}

// Konfigurasi Log Query Database
var QueryLogConfig = struct {
	Enabled       bool
	Filename      string
	MaxSize       int // megabytes
	MaxBackups    int
	MaxAge        int // days
	Compress      bool
	SlowThreshold time.Duration
	LogLevel      gormlogger.LogLevel
}{
	Enabled:       true,
	Filename:      "storage/logs/query/query.log",
	MaxSize:       10, // 10MB
	MaxBackups:    5,
	MaxAge:        30,
	Compress:      true,
	SlowThreshold: 200 * time.Millisecond,
	LogLevel:      gormlogger.Info, // Info = Log semua query
}

// GetAppLoggerWriter mengembalikan io.Writer yang support rotation untuk App Log
func GetAppLoggerWriter() io.Writer {
	if !AppLogConfig.Enabled {
		return io.Discard
	}

	// Pastikan folder ada
	dir := filepath.Dir(AppLogConfig.Filename)
	_ = os.MkdirAll(dir, 0755)

	return &lumberjack.Logger{
		Filename:   AppLogConfig.Filename,
		MaxSize:    AppLogConfig.MaxSize,
		MaxBackups: AppLogConfig.MaxBackups,
		MaxAge:     AppLogConfig.MaxAge,
		Compress:   AppLogConfig.Compress,
	}
}

// GetErrorLoggerWriter mengembalikan io.Writer yang support rotation untuk Error Log
func GetErrorLoggerWriter() io.Writer {
	if !ErrorLogConfig.Enabled {
		return io.Discard
	}

	// Pastikan folder ada
	dir := filepath.Dir(ErrorLogConfig.Filename)
	_ = os.MkdirAll(dir, 0755)

	return &lumberjack.Logger{
		Filename:   ErrorLogConfig.Filename,
		MaxSize:    ErrorLogConfig.MaxSize,
		MaxBackups: ErrorLogConfig.MaxBackups,
		MaxAge:     ErrorLogConfig.MaxAge,
		Compress:   ErrorLogConfig.Compress,
	}
}

type minuteFileWriter struct {
	mu         sync.Mutex
	dir        string
	prefix     string
	currentKey string
	file       *os.File
}

func newMinuteFileWriter(dir, prefix string) *minuteFileWriter {
	_ = os.MkdirAll(dir, 0755)
	return &minuteFileWriter{dir: dir, prefix: prefix}
}

func NewMinuteFileWriter(dir, prefix string) io.Writer {
	return newMinuteFileWriter(dir, prefix)
}

func (w *minuteFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	key := time.Now().Format("2006-01-02-15-04")
	if w.file == nil || w.currentKey != key {
		if w.file != nil {
			_ = w.file.Close()
		}
		w.currentKey = key
		filename := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, key))
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return 0, err
		}
		w.file = f
	}
	return w.file.Write(p)
}

// utils.FileWithLineNum replacement untuk mencari caller file
func getCaller(skip int) string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!strings.Contains(file, "gorm.io") && !strings.Contains(file, "config/log.go")) {
			return fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}
	return ""
}

type GormErrorLogger struct {
	ConnectionName string
	QueryLogger    *log.Logger
	ErrorLogger    *log.Logger
	SlowThreshold  time.Duration
	LogLevel       gormlogger.LogLevel
}

func (l *GormErrorLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &GormErrorLogger{
		ConnectionName: l.ConnectionName,
		QueryLogger:    l.QueryLogger,
		ErrorLogger:    l.ErrorLogger,
		SlowThreshold:  l.SlowThreshold,
		LogLevel:       level,
	}
}

func (l *GormErrorLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel < gormlogger.Info {
		return
	}
	l.QueryLogger.Printf("[%s] "+msg, append([]interface{}{l.ConnectionName}, data...)...)
}

func (l *GormErrorLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel < gormlogger.Warn {
		return
	}
	l.QueryLogger.Printf("[%s] "+msg, append([]interface{}{l.ConnectionName}, data...)...)
}

func (l *GormErrorLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel < gormlogger.Error {
		return
	}
	l.ErrorLogger.Printf("[%s] "+msg, append([]interface{}{l.ConnectionName}, data...)...)
}

func (l *GormErrorLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel == gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	caller := getCaller(2)

	method := "-"
	url := "-"
	controller := "-"
	if info, ok := GetRequestInfo(ctx); ok {
		if info.Method != "" {
			method = info.Method
		}
		if info.URL != "" {
			url = info.URL
		}
		if info.Module != "" || info.Controller != "" || info.Action != "" {
			controller = strings.Trim(strings.Join([]string{info.Module, info.Controller, info.Action}, "."), ".")
		}
	}

	status := "OK"
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		status = "ERROR"
	} else if l.SlowThreshold > 0 && elapsed > l.SlowThreshold {
		status = "SLOW"
	}

	trace := method + " " + url + " -> " + controller
	l.QueryLogger.Printf("[%s] trace=%s | at=%s | %s | %s | rows=%d | %s", l.ConnectionName, trace, caller, status, elapsed, rows, sql)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.ErrorLogger.Printf("[%s] trace=%s | at=%s | %s | %s | %v | %s", l.ConnectionName, trace, caller, status, elapsed, err, sql)
	}
}

// GetGormLogger mengembalikan custom logger untuk GORM
func GetGormLogger(connectionName string) gormlogger.Interface {
	if !QueryLogConfig.Enabled {
		return gormlogger.Default.LogMode(gormlogger.Silent)
	}

	// Pastikan folder query log ada
	queryDir := filepath.Dir(QueryLogConfig.Filename)
	_ = os.MkdirAll(queryDir, 0755)

	return &GormErrorLogger{
		ConnectionName: connectionName,
		QueryLogger:    log.New(newMinuteFileWriter(queryDir, "query"), "", log.LstdFlags),
		ErrorLogger:    log.New(GetErrorLoggerWriter(), "", log.LstdFlags),
		SlowThreshold:  QueryLogConfig.SlowThreshold,
		LogLevel:       QueryLogConfig.LogLevel,
	}
}
