package gbr

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mulyo-go/framework/config"
	helper "github.com/mulyo-go/framework/helper"
	logger "github.com/mulyo-go/framework/logger"
)

var (
	GlobalDataProvider func(c *gin.Context, data gin.H)
)

func SetGlobalDataProvider(fn func(c *gin.Context, data gin.H)) {
	GlobalDataProvider = fn
}

func AddFunc(name string, fn any) {
	FuncMap[name] = fn
}

var (
	templateCache = make(map[string]*template.Template)
	templateMu    sync.RWMutex
	templateFS    embed.FS
	moduleFS      embed.FS
	useEmbed      bool
)

// templateStack for @push/@stack support
type templateStack struct {
	items map[string]string
	data  interface{} // template data untuk render variables
}

// unescapeStackContent unescapes control characters (\t, \n, \r, \", \\) produced by strconv.Quote
func unescapeStackContent(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 't':
				b.WriteByte('\t')
				i++
			case 'n':
				b.WriteByte('\n')
				i++
			case 'r':
				b.WriteByte('\r')
				i++
			case '"':
				b.WriteByte('"')
				i++
			case '\\':
				b.WriteByte('\\')
				i++
			default:
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func (s *templateStack) Set(name, content string) string {
	s.items[name] = unescapeStackContent(content)
	return ""
}

func (s *templateStack) Prepend(name, content string) string {
	s.items[name] = unescapeStackContent(content) + s.items[name]
	return ""
}

func (s *templateStack) Render(name string) template.HTML {
	raw, ok := s.items[name]
	if !ok || raw == "" {
		return ""
	}
	// Parse content sebagai template dan execute dengan data
	t, err := template.New("__stack_" + name).Funcs(FuncMap).Parse(raw)
	if err != nil {
		return template.HTML(raw)
	}
	var buf strings.Builder
	if err := t.Execute(&buf, s.data); err != nil {
		return template.HTML(raw)
	}
	return template.HTML(buf.String())
}

// FuncMap berisi semua template functions yang tersedia di GBR
var FuncMap = template.FuncMap{
	// --- String & Text Functions ---
	"upper":      strings.ToUpper,
	"lower":      strings.ToLower,
	"title":      strings.Title,
	"ucfirst":    ucfirst,
	"kebab":      toKebab,
	"snake":      toSnake,
	"camel":      toCamel,
	"studly":     toStudly,
	"slug":       toSlug,
	"limit":      strLimit,
	"contains":   strings.Contains,
	"startsWith": strings.HasPrefix,
	"endsWith":   strings.HasSuffix,
	"replace":    strings.ReplaceAll,
	"trim":       strings.TrimSpace,
	"reverse":    strReverse,
	"wordCount":  wordCount,
	"substr":     strSubstr,
	"squish":     squish,
	"headline":   headline,
	"plural":     strPlural,
	"singular":   strSingular,
	"padLeft":    padLeft,
	"padRight":   padRight,
	"join":       strings.Join,
	"split":      strings.Split,
	"nl2br":      nl2br,
	"excerpt":    excerpt,
	"mask":       maskString,

	// --- Math & Number Functions ---
	"add":           numAdd,
	"sub":           numSub,
	"mul":           numMul,
	"div":           numDiv,
	"mod":           numMod,
	"min":           numMin,
	"max":           numMax,
	"abs":           math.Abs,
	"round":         math.Round,
	"floor":         math.Floor,
	"ceil":          math.Ceil,
	"number_format": formatNumber,
	"percentage":    formatPercent,
	"bytes":         formatBytes,

	// --- Currency & Formatting ---
	"formatIDR": formatIDR,
	"idr":       formatIDR,

	// --- Date & Time Functions ---
	"date":    dateFormat,
	"now":     time.Now,
	"tglIndo": tglIndo,
	"diffForHumans": timeAgo,
	"timeAgo":       timeAgo,

	// --- Logic, Comparison & Conditionals ---
	"ternary":  ternary,
	"default":    defaultVal,
	"defaultVal": defaultVal,
	"empty":      isEmpty,
	"notEmpty":   notEmpty,
	"eqAny":      eqAny,

	// --- Array & Slice Functions ---
	"inArray": inArray,
	"hasKey":  hasKey,
	"dict":    dictHelper,
	"list":    listHelper,

	// --- HTML & Escaping ---
	"safeHTML": func(s any) template.HTML {
		return template.HTML(fmt.Sprintf("%v", s))
	},
	"safeCSS": func(s any) template.CSS {
		return template.CSS(fmt.Sprintf("%v", s))
	},
	"safeJS": func(s any) template.JS {
		return template.JS(fmt.Sprintf("%v", s))
	},
	"safeURL": func(s any) template.URL {
		return template.URL(fmt.Sprintf("%v", s))
	},
	"json": func(v any) template.JS {
		b, err := json.Marshal(v)
		if err != nil {
			return template.JS("null")
		}
		return template.JS(b)
	},
	"js": func(v any) template.JS {
		b, err := json.Marshal(v)
		if err != nil {
			return template.JS("null")
		}
		return template.JS(b)
	},
	"dump": func(v any) string {
		return fmt.Sprintf("%+v", v)
	},
	"dd": func(v any) string {
		return fmt.Sprintf("%+v", v)
	},
	"deref": func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	},

	// --- Internal Framework Helpers ---
	"newStack": func() *templateStack {
		return &templateStack{items: map[string]string{}}
	},
	"urlFile": helper.URLFile,
	"filterMenus": func(menus any, allowedIDs []int64) any {
		return menus
	},
}

// SetFS menyimpan embed FS untuk template dan module views
func SetFS(tmpl, mod embed.FS) {
	templateFS = tmpl
	moduleFS = mod
	useEmbed = true
}

func LoadTemplates() {
	templateMu.Lock()
	defer templateMu.Unlock()

	// Pre-parse error pages
	for _, code := range []string{"404", "500", "403"} {
		for _, ext := range []string{".gbr.html", ".blade.html"} {
			name := "Error/" + code + ext
			var t *template.Template
			var err error
			if useEmbed {
				t, err = template.ParseFS(templateFS, "Template/"+name)
			} else {
				t, err = template.ParseFiles(filepath.Join("Template", filepath.FromSlash(name)))
			}
			if err == nil {
				templateCache["Error/"+code] = t
				templateCache[name] = t
				break
			}
		}
	}

	// Pre-parse layout
	for _, ext := range []string{".gbr.html", ".blade.html"} {
		layoutPath := "Template/Default/layout" + ext
		var t *template.Template
		var err error
		if useEmbed {
			t, err = template.ParseFS(templateFS, layoutPath)
		} else {
			t, err = template.ParseFiles(filepath.Join("Template", "Default", "layout"+ext))
		}
		if err == nil {
			templateCache["layout"] = t
			templateCache["Default/layout"] = t
			templateCache["Template/Default/layout"+ext] = t
			break
		}
	}
}

func getTemplate(name string) *template.Template {
	templateMu.RLock()
	defer templateMu.RUnlock()
	return templateCache[name]
}

func enrichTemplateData(c *gin.Context, data gin.H) gin.H {
	if data == nil {
		data = gin.H{}
	}

	data["LoggedIn"] = false
	data["UserName"] = ""
	data["UserMenuIDs"] = []int64{}

	if userName, exists := c.Get("user_name"); exists && userName != nil {
		data["LoggedIn"] = true
		data["UserName"] = userName
	}
	if menuIDs, ok := c.Get("user_menus"); ok {
		data["UserMenuIDs"] = menuIDs
	}

	if config.Session != nil {
		sess := config.StartSession(c)
		if flashes := sess.GetFlashes(); len(flashes) > 0 {
			for k, v := range flashes {
				data["Flash"+strings.Title(k)] = v
			}
			data["Flashes"] = flashes
		}
	}

	// CSRF token untuk @csrf directive
	if csrfToken, exists := c.Get("csrf_token"); exists {
		data["CsrfToken"] = csrfToken
	}

	// Custom application global data hook
	if GlobalDataProvider != nil {
		GlobalDataProvider(c, data)
	}

	// Stack untuk @push/@stack (CSS, JS per halaman)
	data["__stacks"] = &templateStack{items: map[string]string{}, data: data}

	return data
}

var (
	projectRootOnce   sync.Once
	cachedProjectRoot string
)

func findProjectRoot() string {
	projectRootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			return
		}
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				cachedProjectRoot = dir
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	})
	return cachedProjectRoot
}

func readFileWithFallback(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		return content, nil
	}
	root := findProjectRoot()
	if root != "" {
		absPath := filepath.Join(root, path)
		if rootContent, rootErr := os.ReadFile(absPath); rootErr == nil {
			return rootContent, nil
		}
	}
	return nil, err
}

func viewFileExists(path string) bool {
	path = filepath.ToSlash(path)
	if useEmbed {
		if strings.HasPrefix(path, "Template/") {
			if _, err := templateFS.ReadFile(path); err == nil {
				return true
			}
		} else {
			if _, err := moduleFS.ReadFile(path); err == nil {
				return true
			}
		}
	}

	if _, err := os.Stat(filepath.FromSlash(path)); err == nil {
		return true
	}
	root := findProjectRoot()
	if root != "" {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err == nil {
			return true
		}
	}
	return false
}

// resolveViewPath resolves a view name (with or without .gbr.html/.blade.html) to its file path
func resolveViewPath(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), "/")
	if name == "" {
		return ""
	}

	name = filepath.ToSlash(name)

	var nameCandidates []string
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".gbr.html") || strings.HasSuffix(lower, ".blade.html") || strings.HasSuffix(lower, ".html") {
		nameCandidates = append(nameCandidates, name)
	} else {
		nameCandidates = append(nameCandidates, name+".gbr.html", name+".blade.html", name)
	}

	for _, cand := range nameCandidates {
		// If candidate explicitly starts with Template/ or Module/
		if strings.HasPrefix(cand, "Template/") || strings.HasPrefix(cand, "Module/") {
			if viewFileExists(cand) {
				return cand
			}
		}

		if strings.HasPrefix(cand, "Metronic/") || strings.HasPrefix(cand, "Default/") || strings.HasPrefix(cand, "Error/") {
			if viewFileExists("Template/" + cand) {
				return "Template/" + cand
			}
		}

		// Check Module/ first (default for application views)
		if viewFileExists("Module/" + cand) {
			return "Module/" + cand
		}

		// Check Template/
		if viewFileExists("Template/" + cand) {
			return "Template/" + cand
		}

		// Check direct path
		if viewFileExists(cand) {
			return cand
		}
	}

	// Default fallback if not matched yet
	if strings.HasPrefix(name, "Template/") || strings.HasPrefix(name, "Metronic/") || strings.HasPrefix(name, "Default/") || strings.HasPrefix(name, "Error/") {
		prefix := ""
		if !strings.HasPrefix(name, "Template/") {
			prefix = "Template/"
		}
		if strings.HasSuffix(lower, ".gbr.html") || strings.HasSuffix(lower, ".blade.html") || strings.HasSuffix(lower, ".html") {
			return prefix + name
		}
		return prefix + name + ".gbr.html"
	}

	if strings.HasSuffix(lower, ".gbr.html") || strings.HasSuffix(lower, ".blade.html") || strings.HasSuffix(lower, ".html") {
		return "Module/" + name
	}
	return "Module/" + name + ".gbr.html"
}

// resolveLayoutPath resolves layout name relative to Template/
func resolveLayoutPath(layoutName string) string {
	layoutName = strings.TrimPrefix(strings.TrimSpace(layoutName), "/")
	if layoutName == "" {
		return ""
	}
	layoutName = filepath.ToSlash(layoutName)

	var candidates []string
	lower := strings.ToLower(layoutName)
	if strings.HasSuffix(lower, ".gbr.html") || strings.HasSuffix(lower, ".blade.html") || strings.HasSuffix(lower, ".html") {
		candidates = append(candidates, layoutName)
	} else {
		candidates = append(candidates, layoutName+".gbr.html", layoutName+".blade.html", layoutName)
	}

	for _, cand := range candidates {
		if strings.HasPrefix(cand, "Template/") {
			if viewFileExists(cand) {
				return cand
			}
		}
		if viewFileExists("Template/" + cand) {
			return "Template/" + cand
		}
		if viewFileExists(cand) {
			return cand
		}
	}

	if strings.HasPrefix(layoutName, "Template/") {
		return layoutName
	}
	return "Template/" + layoutName
}

func readViewContent(viewPath string) (string, error) {
	viewPath = filepath.ToSlash(viewPath)
	var content []byte
	var err error

	if useEmbed {
		if strings.HasPrefix(viewPath, "Template/") {
			content, err = templateFS.ReadFile(viewPath)
		} else {
			content, err = moduleFS.ReadFile(viewPath)
		}
		if err != nil {
			content, err = readFileWithFallback(viewPath)
		}
	} else {
		content, err = readFileWithFallback(viewPath)
	}
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// detectLayoutFromContent extracts layout path from template invocation in view content.
func detectLayoutFromContent(content string) string {
	matches := layoutRefRegex.FindAllStringSubmatch(content, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		if len(matches[i]) < 2 {
			continue
		}
		layoutName := matches[i][1]
		if layoutName == "" || layoutName == "content" {
			continue
		}
		lower := strings.ToLower(layoutName)
		if strings.HasSuffix(lower, "layout.gbr.html") || strings.HasSuffix(lower, "layout.blade.html") || strings.HasSuffix(lower, "layout") || strings.Contains(lower, "layout") {
			return layoutName
		}
	}
	return ""
}

// layoutRefRegex matches {{ template "LayoutPath" . }} in view files
var layoutRefRegex = regexp.MustCompile(`\{\{-?\s*template\s+"([^"]+)"\s+\.\s*-?\}\}`)

func stripLayoutReference(content string, layoutName string) string {
	if layoutName == "" {
		return strings.TrimSpace(content)
	}
	pattern := regexp.MustCompile(`\{\{-?\s*template\s+"` + regexp.QuoteMeta(layoutName) + `"\s+\.\s*-?\}\}`)
	return strings.TrimSpace(pattern.ReplaceAllString(content, ""))
}

func ensureContentTemplate(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return trimmed
	}
	if strings.Contains(trimmed, `{{ define "content" }}`) {
		return trimmed
	}
	return `{{ define "content" }}` + "\n" + trimmed + "\n{{ end }}"
}

func prepareViewTemplate(viewPath string) (string, string, error) {
	rawContent, err := readViewContent(viewPath)
	if err != nil {
		return "", "", err
	}

	preprocessedContent := preprocessGbr(rawContent)
	layoutName := detectLayoutFromContent(rawContent)
	if layoutName == "" {
		layoutName = detectLayoutFromContent(preprocessedContent)
	}
	if layoutName != "" {
		preprocessedContent = ensureContentTemplate(stripLayoutReference(preprocessedContent, layoutName))
	}

	return preprocessedContent, layoutName, nil
}

func parseTemplateFile(fs embed.FS, path string, embedded bool) (*template.Template, error) {
	baseName := filepath.Base(path)
	root := template.New(baseName).Funcs(FuncMap)
	var content string
	if embedded {
		rawContent, err := fs.ReadFile(path)
		if err != nil {
			rawContent, err = readFileWithFallback(path)
		}
		if err != nil {
			return nil, err
		}
		content = string(rawContent)
		if _, err = root.Parse(content); err != nil {
			return nil, err
		}
		if err = parseReferencedTemplates(root, content, map[string]bool{}); err != nil {
			return nil, err
		}
		return root, nil
	}
	rawContent, err := readFileWithFallback(path)
	if err != nil {
		return nil, err
	}
	content = string(rawContent)
	if _, err = root.Parse(content); err != nil {
		return nil, err
	}
	if err = parseReferencedTemplates(root, content, map[string]bool{}); err != nil {
		return nil, err
	}
	return root, nil
}

func readTemplateContent(templatePath string) (string, error) {
	templatePath = filepath.ToSlash(templatePath)
	if useEmbed {
		content, err := templateFS.ReadFile(templatePath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	content, err := readFileWithFallback(filepath.FromSlash(templatePath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func readReferencedTemplateContent(templateName string) (string, string, error) {
	templateName = strings.TrimPrefix(strings.TrimSpace(templateName), "/")
	if templateName == "" {
		return "", "", os.ErrNotExist
	}
	templateName = filepath.ToSlash(templateName)

	var names []string
	lower := strings.ToLower(templateName)
	if strings.HasSuffix(lower, ".gbr.html") || strings.HasSuffix(lower, ".blade.html") || strings.HasSuffix(lower, ".html") {
		names = append(names, templateName)
	} else {
		names = append(names, templateName+".gbr.html", templateName+".blade.html", templateName)
	}

	for _, name := range names {
		candidates := []struct {
			templatePath string
			defineName   string
		}{
			{templatePath: "Template/" + name, defineName: templateName},
			{templatePath: "Module/" + name, defineName: templateName},
			{templatePath: name, defineName: templateName},
		}

		for _, candidate := range candidates {
			var content string
			var err error
			if strings.HasPrefix(candidate.templatePath, "Template/") {
				content, err = readTemplateContent(candidate.templatePath)
			} else {
				content, err = readViewContent(candidate.templatePath)
			}
			if err == nil {
				return candidate.defineName, content, nil
			}
		}
	}

	return "", "", os.ErrNotExist
}

func parseReferencedTemplates(t *template.Template, content string, loaded map[string]bool) error {
	matches := layoutRefRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		templateName := match[1]
		lower := strings.ToLower(templateName)
		if templateName == "" || templateName == "content" || strings.HasSuffix(lower, "layout.gbr.html") || strings.HasSuffix(lower, "layout.blade.html") {
			continue
		}
		if loaded[templateName] {
			continue
		}
		loaded[templateName] = true

		defineName, partialContent, err := readReferencedTemplateContent(templateName)
		if err != nil {
			continue
		}

		partialContent = preprocessGbr(partialContent)
		if !strings.Contains(partialContent, `{{ define "`) {
			partialContent = `{{ define "` + defineName + `" }}` + "\n" + partialContent + "\n{{ end }}"
		}

		partialTree := template.New("partial").Funcs(FuncMap)
		partialTree, errParse := partialTree.Parse(partialContent)
		if errParse != nil {
			return errParse
		}
		for _, pt := range partialTree.Templates() {
			if pt.Name() == "partial" {
				t.AddParseTree(templateName, pt.Tree)
			} else {
				t.AddParseTree(pt.Name(), pt.Tree)
			}
		}
		if err = parseReferencedTemplates(t, partialContent, loaded); err != nil {
			return err
		}
	}

	return nil
}

func executeStandaloneView(c *gin.Context, name string, content string, data gin.H) error {
	t, err := template.New(filepath.Base(name)).Funcs(FuncMap).Parse(content)
	if err != nil {
		return err
	}
	if err = parseReferencedTemplates(t, content, map[string]bool{}); err != nil {
		return err
	}

	execName := filepath.Base(name)
	for _, tmpl := range t.Templates() {
		if tmpl.Name() == "content" {
			execName = "content"
			break
		}
	}

	return t.ExecuteTemplate(c.Writer, execName, data)
}

func executeViewWithLayout(c *gin.Context, name string, layoutName string, content string, data gin.H) error {
	layoutPath := resolveLayoutPath(layoutName)

	var (
		t   *template.Template
		err error
	)
	if useEmbed {
		t, err = parseTemplateFile(templateFS, layoutPath, true)
	} else {
		t, err = parseTemplateFile(templateFS, layoutPath, false)
	}
	if err != nil {
		return err
	}

	if err = parseReferencedTemplates(t, content, map[string]bool{}); err != nil {
		return err
	}

	viewTree := template.New("view").Funcs(FuncMap)
	if _, err = viewTree.Parse(content); err != nil {
		return err
	}
	for _, tmpl := range viewTree.Templates() {
		if tmpl == nil || tmpl.Tree == nil || tmpl.Name() == "view" {
			continue
		}
		t.AddParseTree(tmpl.Name(), tmpl.Tree)
	}

	preRenderStacks(content, data)

	return t.ExecuteTemplate(c.Writer, filepath.Base(layoutPath), data)
}

func Render(c *gin.Context, name string, data gin.H) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	data = enrichTemplateData(c, data)

	var t *template.Template
	var err error

	if strings.HasPrefix(name, "Error/") || strings.HasPrefix(name, "Default/") {
		t = getTemplate(name)
		if t == nil {
			viewPath := resolveViewPath(name)
			if useEmbed {
				t, err = template.ParseFS(templateFS, viewPath)
			} else {
				t, err = template.ParseFiles(filepath.FromSlash(viewPath))
			}
			if err == nil {
				clone := template.New(filepath.Base(viewPath)).Funcs(FuncMap)
				for _, tmpl := range t.Templates() {
					clone.AddParseTree(tmpl.Name(), tmpl.Tree)
				}
				err = clone.ExecuteTemplate(c.Writer, filepath.Base(viewPath), data)
				return
			}
		} else {
			clone := template.New(filepath.Base(name)).Funcs(FuncMap)
			for _, tmpl := range t.Templates() {
				clone.AddParseTree(tmpl.Name(), tmpl.Tree)
			}
			err = clone.ExecuteTemplate(c.Writer, filepath.Base(name), data)
		}
	} else {
		viewPath := resolveViewPath(name)
		preprocessedContent, layoutName, prepErr := prepareViewTemplate(viewPath)
		if prepErr != nil {
			logger.Errorf("gbr view not found: %s (view: %s)", viewPath, name)
			c.Status(http.StatusNotFound)
			return
		}

		if layoutName == "" {
			err = executeStandaloneView(c, name, preprocessedContent, data)
		} else {
			err = executeViewWithLayout(c, name, layoutName, preprocessedContent, data)
		}
	}

	if err != nil {
		logger.Errorf("gbr render error [%s]: %v", name, err)
		if config.Env("DEBUG", "false") == "true" {
			renderDebugError(c, name, err, "")
		} else {
			c.Status(http.StatusInternalServerError)
			_, _ = c.Writer.Write([]byte("template error"))
		}
	}
}

// RenderWithLayout - render view dengan layout custom
// layout: path relative ke Template/ (contoh: "Metronic/layout.gbr.html" atau "Metronic/layout")
func RenderWithLayout(c *gin.Context, name string, layout string, data gin.H) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	data = enrichTemplateData(c, data)

	viewPath := resolveViewPath(name)
	content, prepErr := readViewContent(viewPath)
	if prepErr != nil {
		logger.Errorf("gbr view not found: %s (view: %s)", viewPath, name)
		c.Status(http.StatusNotFound)
		return
	}

	preprocessedContent := ensureContentTemplate(stripLayoutReference(preprocessGbr(content), layout))
	err := executeViewWithLayout(c, name, layout, preprocessedContent, data)
	if err != nil {
		logger.Errorf("gbr render error [%s] layout [%s]: %v", name, layout, err)
		if config.Env("DEBUG", "false") == "true" {
			renderDebugError(c, name, err, "")
		} else {
			c.Status(http.StatusInternalServerError)
			_, _ = c.Writer.Write([]byte("template error"))
		}
	}
}

// isAPIRequest cek apakah request dari AJAX atau ke /api/*
func isAPIRequest(c *gin.Context) bool {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		return true
	}
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		return true
	}
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		return true
	}
	return false
}

// jsonError helper untuk response JSON error
func jsonError(c *gin.Context, status int, message string, detail gin.H) {
	resp := gin.H{
		"http_status": status,
		"error":       http.StatusText(status),
		"message":     message,
	}
	for k, v := range detail {
		resp[k] = v
	}
	c.JSON(status, resp)
}

func RenderError(c *gin.Context, status int) {
	if isAPIRequest(c) {
		jsonError(c, status, http.StatusText(status), nil)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)

	codeStr := strconv.Itoa(status)
	t := getTemplate("Error/" + codeStr)
	if t == nil {
		t = getTemplate("Error/" + codeStr + ".gbr.html")
	}
	if t == nil {
		t = getTemplate("Error/" + codeStr + ".blade.html")
	}

	if t != nil {
		var err error
		if tmpl := t.Lookup(t.Name()); tmpl != nil {
			err = tmpl.Execute(c.Writer, enrichTemplateData(c, gin.H{}))
		} else {
			err = t.Execute(c.Writer, enrichTemplateData(c, gin.H{}))
		}
		if err == nil {
			return
		}
	}

	viewPath := resolveViewPath("Error/" + codeStr)
	content, err := readViewContent(viewPath)
	if err != nil {
		_, _ = c.Writer.Write([]byte(http.StatusText(status)))
		return
	}
	preprocessed := preprocessGbr(content)
	_ = executeStandaloneView(c, "Error/"+codeStr, preprocessed, enrichTemplateData(c, gin.H{}))
}

// ---------- Helper function implementations ----------

func ucfirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func toKebab(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "-")
}

func toSnake(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "_")
}

func toCamel(s string) string {
	parts := strings.Fields(strings.ToLower(s))
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func toStudly(s string) string {
	parts := strings.Fields(strings.ToLower(s))
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func toSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var result []rune
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result = append(result, r)
			prevDash = false
		} else if r == ' ' || r == '-' || r == '_' {
			if !prevDash && len(result) > 0 {
				result = append(result, '-')
				prevDash = true
			}
		}
	}
	return strings.TrimRight(string(result), "-")
}

func strLimit(s string, length int, suffix string) string {
	if len(s) <= length {
		return s
	}
	if suffix == "" {
		suffix = "..."
	}
	return s[:length] + suffix
}

func strReverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func wordCount(s string) int {
	return len(strings.Fields(s))
}

func strSubstr(s string, start, length int) string {
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

func squish(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func headline(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}

func strPlural(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

func strSingular(s string) string {
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

func padLeft(s string, length int, pad string) string {
	if len(s) >= length {
		return s
	}
	if pad == "" {
		pad = " "
	}
	return strings.Repeat(pad, length-len(s)) + s
}

func padRight(s string, length int, pad string) string {
	if len(s) >= length {
		return s
	}
	if pad == "" {
		pad = " "
	}
	return s + strings.Repeat(pad, length-len(s))
}

func formatIDR(n interface{}) string {
	if n == nil {
		return "Rp 0"
	}
	var val float64
	switch v := n.(type) {
	case int:
		val = float64(v)
	case int8:
		val = float64(v)
	case int16:
		val = float64(v)
	case int32:
		val = float64(v)
	case int64:
		val = float64(v)
	case uint:
		val = float64(v)
	case uint8:
		val = float64(v)
	case uint16:
		val = float64(v)
	case uint32:
		val = float64(v)
	case uint64:
		val = float64(v)
	case float32:
		val = float64(v)
	case float64:
		val = v
	case []byte:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		if err != nil {
			return string(v)
		}
		val = parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return v
		}
		val = parsed
	default:
		return fmt.Sprintf("%v", n)
	}

	intVal := int64(math.Round(val))
	s := fmt.Sprintf("%d", intVal)
	isNegative := false
	if intVal < 0 {
		isNegative = true
		s = s[1:]
	}

	result := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(ch)
	}
	if isNegative {
		return "-Rp " + result
	}
	return "Rp " + result
}

func ternary(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

func toFloat(n any) float64 {
	if n == nil {
		return 0
	}
	switch v := n.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	case []byte:
		f, _ := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		return f
	default:
		return 0
	}
}

func numAdd(args ...any) float64 {
	var total float64
	for _, a := range args {
		total += toFloat(a)
	}
	return total
}

func numSub(a, b any) float64 {
	return toFloat(a) - toFloat(b)
}

func numMul(a, b any) float64 {
	return toFloat(a) * toFloat(b)
}

func numDiv(a, b any) float64 {
	fb := toFloat(b)
	if fb == 0 {
		return 0
	}
	return toFloat(a) / fb
}

func numMod(a, b any) int {
	ia := int(toFloat(a))
	ib := int(toFloat(b))
	if ib == 0 {
		return 0
	}
	return ia % ib
}

func numMin(a, b any) float64 {
	fa, fb := toFloat(a), toFloat(b)
	if fa < fb {
		return fa
	}
	return fb
}

func numMax(a, b any) float64 {
	fa, fb := toFloat(a), toFloat(b)
	if fa > fb {
		return fa
	}
	return fb
}

func formatNumber(n any, decimals int, decPoint, thousandsSep string) string {
	val := toFloat(n)
	isNeg := val < 0
	if isNeg {
		val = -val
	}

	intPart := int64(val)
	var fracStr string
	if decimals > 0 {
		pow := math.Pow10(decimals)
		fracPart := int64(math.Round((val - float64(intPart)) * pow))
		format := fmt.Sprintf("%%0%dd", decimals)
		fracStr = decPoint + fmt.Sprintf(format, fracPart)
	}

	s := strconv.FormatInt(intPart, 10)
	var formattedInt strings.Builder
	l := len(s)
	for i, ch := range s {
		if i > 0 && (l-i)%3 == 0 {
			formattedInt.WriteString(thousandsSep)
		}
		formattedInt.WriteRune(ch)
	}

	res := formattedInt.String() + fracStr
	if isNeg {
		res = "-" + res
	}
	return res
}

func formatPercent(n any, decimals int) string {
	val := toFloat(n)
	if val < 1 && val > -1 && val != 0 {
		val *= 100
	}
	return fmt.Sprintf("%.*f%%", decimals, val)
}

func formatBytes(b any) string {
	bytes := toFloat(b)
	if bytes <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for bytes >= 1024 && i < len(units)-1 {
		bytes /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", bytes, units[i])
	}
	return fmt.Sprintf("%.2f %s", bytes, units[i])
}

func parseTime(t any) time.Time {
	if t == nil {
		return time.Time{}
	}
	switch v := t.(type) {
	case time.Time:
		return v
	case int64:
		if v > 1000000000000 {
			return time.UnixMilli(v)
		}
		return time.Unix(v, 0)
	case int:
		return time.Unix(int64(v), 0)
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return time.Time{}
		}
		layouts := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
			"02-01-2006",
			"02/01/2006",
		}
		for _, layout := range layouts {
			if tm, err := time.Parse(layout, v); err == nil {
				return tm
			}
		}
	}
	return time.Time{}
}

func dateFormat(t any, layout string) string {
	tm := parseTime(t)
	if tm.IsZero() {
		return "-"
	}

	replacer := strings.NewReplacer(
		"Y", "2006",
		"y", "06",
		"m", "01",
		"d", "02",
		"H", "15",
		"h", "03",
		"i", "04",
		"s", "05",
		"A", "PM",
		"a", "pm",
		"M", "Jan",
		"F", "January",
		"D", "Mon",
		"l", "Monday",
	)
	goLayout := replacer.Replace(layout)
	return tm.Format(goLayout)
}

func tglIndo(t any) string {
	tm := parseTime(t)
	if tm.IsZero() {
		return "-"
	}
	bulan := []string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	return fmt.Sprintf("%02d %s %d", tm.Day(), bulan[tm.Month()], tm.Year())
}

func timeAgo(t any) string {
	tm := parseTime(t)
	if tm.IsZero() {
		return "-"
	}
	diff := time.Since(tm)
	if diff < 0 {
		diff = -diff
	}
	seconds := int(diff.Seconds())
	if seconds < 5 {
		return "baru saja"
	}
	if seconds < 60 {
		return fmt.Sprintf("%d detik yang lalu", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%d menit yang lalu", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%d jam yang lalu", hours)
	}
	days := hours / 24
	if days < 30 {
		return fmt.Sprintf("%d hari yang lalu", days)
	}
	months := days / 30
	if months < 12 {
		return fmt.Sprintf("%d bulan yang lalu", months)
	}
	years := days / 365
	return fmt.Sprintf("%d tahun yang lalu", years)
}

func nl2br(s string) template.HTML {
	escaped := template.HTMLEscapeString(s)
	return template.HTML(strings.ReplaceAll(escaped, "\n", "<br/>"))
}

func excerpt(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	trimmed := s[:maxLen]
	lastSpace := strings.LastIndex(trimmed, " ")
	if lastSpace > 0 {
		trimmed = trimmed[:lastSpace]
	}
	return strings.TrimSpace(trimmed) + "..."
}

func maskString(s string, start, length int, maskChar string) string {
	if maskChar == "" {
		maskChar = "*"
	}
	rs := []rune(s)
	if start >= len(rs) {
		return s
	}
	end := start + length
	if end > len(rs) {
		end = len(rs)
	}
	for i := start; i < end; i++ {
		rs[i] = []rune(maskChar)[0]
	}
	return string(rs)
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return strings.TrimSpace(rv.String()) == ""
	case reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

func notEmpty(v any) bool {
	return !isEmpty(v)
}

func eqAny(target any, candidates ...any) bool {
	for _, c := range candidates {
		if fmt.Sprintf("%v", target) == fmt.Sprintf("%v", c) {
			return true
		}
	}
	return false
}

func inArray(needle any, haystack any) bool {
	if haystack == nil {
		return false
	}
	rv := reflect.ValueOf(haystack)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return false
	}
	needleStr := fmt.Sprintf("%v", needle)
	for i := 0; i < rv.Len(); i++ {
		if fmt.Sprintf("%v", rv.Index(i).Interface()) == needleStr {
			return true
		}
	}
	return false
}

func hasKey(m any, key any) bool {
	if m == nil {
		return false
	}
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		return false
	}
	for _, k := range rv.MapKeys() {
		if fmt.Sprintf("%v", k.Interface()) == fmt.Sprintf("%v", key) {
			return true
		}
	}
	return false
}

func dictHelper(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires an even number of arguments")
	}
	dict := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func listHelper(items ...any) []any {
	return items
}

func preRenderStacks(content string, data gin.H) {
	stacks, ok := data["__stacks"].(*templateStack)
	if !ok || stacks == nil {
		return
	}
	searchStr := `.__stacks.Set `
	pos := 0
	for {
		idx := strings.Index(content[pos:], searchStr)
		if idx < 0 {
			break
		}
		pos += idx + len(searchStr)
		name, endPos, ok1 := parseQuotedArg(content, pos)
		if !ok1 {
			continue
		}
		pos = endPos
		for pos < len(content) && content[pos] == ' ' {
			pos++
		}
		val, endPos2, ok2 := parseQuotedArg(content, pos)
		if !ok2 {
			continue
		}
		val = strings.ReplaceAll(val, `\n`, "\n")
		val = strings.ReplaceAll(val, `\"`, `"`)
		val = strings.ReplaceAll(val, `\\`, `\`)
		stacks.Set(name, val)
		pos = endPos2
	}
}

func parseQuotedArg(content string, start int) (string, int, bool) {
	if start >= len(content) || content[start] != '"' {
		return "", start, false
	}
	i := start + 1
	var buf []rune
	for i < len(content) {
		ch := rune(content[i])
		if ch == '\\' && i+1 < len(content) {
			buf = append(buf, '\\', rune(content[i+1]))
			i += 2
			continue
		}
		if ch == '"' {
			return string(buf), i + 1, true
		}
		buf = append(buf, ch)
		i++
	}
	return "", start, false
}

func defaultVal(v interface{}, defaultV interface{}) interface{} {
	if v == nil || v == "" {
		return defaultV
	}
	return v
}

// RenderDebugPanic menampilkan halaman debug untuk panic recovery
func RenderDebugPanic(c *gin.Context, rec interface{}) {
	isDebug := config.Env("DEBUG", "false") == "true"
	if !isDebug {
		RenderError(c, http.StatusInternalServerError)
		return
	}

	stack := string(debug.Stack())
	file := ""
	line := 0

	stackLines := strings.Split(stack, "\n")
	for _, l := range stackLines {
		l = strings.TrimSpace(l)
		if (strings.Contains(l, "mulyo-go/") || strings.Contains(l, "github.com/mulyo-go/framework")) && !strings.Contains(l, "bootstrap/app.go") {
			parts := strings.SplitN(l, ":", 2)
			if len(parts) == 2 {
				file = parts[0]
				lineStr := strings.SplitN(parts[1], " ", 2)[0]
				line, _ = strconv.Atoi(lineStr)
			}
			break
		}
	}

	renderDebugErrorWithLocation(c, file, line, fmt.Sprintf("Panic: %v", rec), stack)
}

// renderDebugError menampilkan halaman error debug untuk template error
func renderDebugError(c *gin.Context, viewName string, err error, stack string) {
	errMsg := err.Error()

	fileName := viewName
	errorLine := 0
	errorMsg := errMsg

	re := regexp.MustCompile(`template:\s*([^:]+):(\d+):\s*(.*)`)
	if matches := re.FindStringSubmatch(errMsg); len(matches) == 4 {
		fileName = strings.TrimSpace(matches[1])
		errorLine, _ = strconv.Atoi(matches[2])
		errorMsg = strings.TrimSpace(matches[3])
	}

	renderDebugPage(c, viewName, fileName, errorLine, errorMsg, stack)
}

func renderDebugErrorWithLocation(c *gin.Context, file string, line int, errorMsg string, stack string) {
	renderDebugPage(c, file, file, line, errorMsg, stack)
}

func renderDebugPage(c *gin.Context, viewName string, fileName string, errorLine int, errorMsg string, stack string) {
	if isAPIRequest(c) {
		detail := gin.H{
			"file": fileName,
			"line": errorLine,
			"view": viewName,
		}
		if stack != "" {
			detail["stack"] = stack
		}
		jsonError(c, http.StatusInternalServerError, errorMsg, detail)
		return
	}

	var sourceLines []string
	sourceFile := ""
	if fileName != "" {
		possiblePaths := []string{
			fileName,
			"Module/" + fileName,
			"Template/" + fileName,
		}
		for _, p := range possiblePaths {
			if content, readErr := readFileWithFallback(p); readErr == nil {
				sourceFile = p
				sourceLines = strings.Split(string(content), "\n")
				break
			}
		}
	}

	type LineInfo struct {
		Num     int
		Content string
		IsError bool
	}
	var contextLines []LineInfo
	if len(sourceLines) > 0 && errorLine > 0 {
		start := errorLine - 5
		if start < 1 {
			start = 1
		}
		end := errorLine + 5
		if end > len(sourceLines) {
			end = len(sourceLines)
		}
		for i := start; i <= end; i++ {
			contextLines = append(contextLines, LineInfo{
				Num:     i,
				Content: sourceLines[i-1],
				IsError: i == errorLine,
			})
		}
	}

	route := fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)

	debugPage := `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Debug Error</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,monospace;background:#1a1a2e;color:#e0e0e0;min-height:100vh;padding:0}
.header{background:#16213e;padding:20px 30px;border-bottom:3px solid #e94560}
.header h1{color:#e94560;font-size:20px;margin-bottom:4px}
.header .route{color:#8892b0;font-size:13px}
.container{max-width:1000px;margin:0 auto;padding:24px 30px}
.error-box{background:#0f3460;border-left:4px solid #e94560;padding:16px 20px;margin-bottom:20px;border-radius:0 6px 6px 0}
.error-box .label{color:#e94560;font-size:11px;text-transform:uppercase;letter-spacing:1px;margin-bottom:6px}
.error-box .msg{color:#fff;font-size:15px;line-height:1.5}
.file-info{display:flex;gap:16px;margin-bottom:20px}
.file-info .badge{background:#16213e;border:1px solid #0f3460;padding:8px 14px;border-radius:6px;font-size:13px}
.file-info .badge strong{color:#53c7f0}
.code-block{background:#0d1117;border:1px solid #30363d;border-radius:8px;overflow:hidden;margin-bottom:20px}
.code-header{background:#161b22;padding:10px 16px;border-bottom:1px solid #30363d;font-size:13px;color:#8b949e}
.code-body{overflow-x:auto}
.code-body table{border-collapse:collapse;width:100%}
.code-body td{padding:1px 16px;font-size:13px;line-height:1.6;white-space:pre;vertical-align:top}
.code-body .line-num{color:#484f58;text-align:right;user-select:none;width:50px;min-width:50px;padding:1px 12px 1px 16px}
.code-body .line-code{padding:1px 16px 1px 8px}
.code-body tr.error-line{background:rgba(233,69,96,0.15)}
.code-body tr.error-line .line-num{color:#e94560;font-weight:bold}
.code-body tr.error-line .line-code{color:#fff}
.stack-section{margin-top:20px}
.stack-section summary{cursor:pointer;color:#53c7f0;font-size:14px;padding:10px 0;outline:none}
.stack-section pre{background:#0d1117;border:1px solid #30363d;border-radius:8px;padding:16px;font-size:12px;line-height:1.6;overflow-x:auto;color:#8b949e;margin-top:8px;white-space:pre-wrap}
.back-link{display:inline-block;margin-top:20px;color:#53c7f0;text-decoration:none;font-size:13px}
.back-link:hover{text-decoration:underline}
</style>
</head>
<body>
<div class="header">
<h1>&#9888; GBR Template Error</h1>
<div class="route">` + route + `</div>
</div>
<div class="container">
<div class="error-box">
<div class="label">Error Message</div>
<div class="msg">` + template.HTMLEscapeString(errorMsg) + `</div>
</div>
<div class="file-info">`

	if sourceFile != "" {
		debugPage += `<div class="badge">File: <strong>` + template.HTMLEscapeString(sourceFile) + `</strong></div>`
	}
	if errorLine > 0 {
		debugPage += `<div class="badge">Line: <strong>` + strconv.Itoa(errorLine) + `</strong></div>`
	}
	debugPage += `<div class="badge">View: <strong>` + template.HTMLEscapeString(viewName) + `</strong></div>
</div>`

	if len(contextLines) > 0 {
		debugPage += `
<div class="code-block">
<div class="code-header">` + template.HTMLEscapeString(sourceFile) + `</div>
<div class="code-body"><table>`
		for _, line := range contextLines {
			class := ""
			if line.IsError {
				class = ` class="error-line"`
			}
			debugPage += `<tr` + class + `><td class="line-num">` + strconv.Itoa(line.Num) + `</td><td class="line-code">` + template.HTMLEscapeString(line.Content) + `</td></tr>`
		}
		debugPage += `</table></div></div>`
	}

	if stack != "" {
		debugPage += `
<div class="stack-section">
<details>
<summary>Show Stack Trace</summary>
<pre>` + template.HTMLEscapeString(stack) + `</pre>
</details>
</div>`
	}

	debugPage += `
<a class="back-link" href="javascript:history.back()">&larr; Back</a>
</div>
</body>
</html>`

	c.Status(http.StatusInternalServerError)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = c.Writer.WriteString(debugPage)
}
