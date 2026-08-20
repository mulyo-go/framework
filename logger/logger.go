package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	LevelDebug   = "debug"
	LevelInfo    = "info"
	LevelWarning = "warning"
	LevelError   = "error"
	LevelFatal   = "fatal"
)

var levelPriority = map[string]int{
	LevelDebug:   0,
	LevelInfo:    1,
	LevelWarning: 2,
	LevelError:   3,
	LevelFatal:   4,
}

var (
	logDir     string
	minLevel   string
	mu         sync.Mutex
	consoleLog bool
)

func init() {
	logDir = findProjectRoot()
	minLevel = LevelDebug
	consoleLog = true
	os.MkdirAll(logDir, 0755)
}

// findProjectRoot mencari root project dari cwd dengan cara navigasi ke atas sampai menemukan go.mod
func findProjectRoot() string {
	dir, _ := os.Getwd()

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "storage", "logs")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback: relatif
	return filepath.Join("storage", "logs")
}

func SetLevel(level string) {
	minLevel = level
}

func SetDir(dir string) {
	// Jika path relatif, jadikan absolut dari project root
	if !filepath.IsAbs(dir) {
		root, _ := findProjectRootAbs()
		dir = filepath.Join(root, dir)
	}
	logDir = dir
	os.MkdirAll(dir, 0755)
}

func findProjectRootAbs() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return ".", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir, fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func SetConsole(on bool) {
	consoleLog = on
}

func getLogFile(level string) string {
	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.log", strings.ToLower(level), date)
	return filepath.Join(logDir, filename)
}

func callerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "???:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

func writeLog(level, message string, context ...map[string]interface{}) {
	if levelPriority[level] < levelPriority[minLevel] {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	caller := callerInfo(3)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s.%s: %s", timestamp, strings.ToUpper(level), caller, message))

	if len(context) > 0 && len(context[0]) > 0 {
		sb.WriteString(" |")
		for k, v := range context[0] {
			sb.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}

	line := sb.String()

	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(getLogFile(level), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		f.WriteString(line + "\n")
		f.Close()
	}

	if consoleLog {
		log.Println(line)
	}

	if level == LevelFatal {
		os.Exit(1)
	}
}

func Debug(message string, context ...map[string]interface{}) {
	writeLog(LevelDebug, message, context...)
}

func Info(message string, context ...map[string]interface{}) {
	writeLog(LevelInfo, message, context...)
}

func Warning(message string, context ...map[string]interface{}) {
	writeLog(LevelWarning, message, context...)
}

func Error(message string, context ...map[string]interface{}) {
	writeLog(LevelError, message, context...)
}

func Fatal(message string, context ...map[string]interface{}) {
	writeLog(LevelFatal, message, context...)
}

func Debugf(format string, args ...interface{}) {
	writeLog(LevelDebug, fmt.Sprintf(format, args...))
}

func Infof(format string, args ...interface{}) {
	writeLog(LevelInfo, fmt.Sprintf(format, args...))
}

func Warningf(format string, args ...interface{}) {
	writeLog(LevelWarning, fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...interface{}) {
	writeLog(LevelError, fmt.Sprintf(format, args...))
}

func Fatalf(format string, args ...interface{}) {
	writeLog(LevelFatal, fmt.Sprintf(format, args...))
}
