package dispatcher

import (
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

var controllerRegistry = make(map[string]reflect.Type)

// paramRouteRegistry menyimpan definisi route per method.
// key: "Module.Controller.Method" → "/:category/:id"
var paramRouteRegistry = make(map[string]string)

// paramNameRegistry menyimpan mapping nama→index per method.
// key: "Module.Controller.Method" → {"category": 0, "id": 1}
var paramNameRegistry = make(map[string]map[string]int)

type requestContextSetter interface {
	SetRequestContext(c *gin.Context, module, controller, action string)
}

// toPascal converts kebab-case or snake_case to PascalCase.
func toPascal(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "_", "-")
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "-") {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if len(p) == 1 {
			parts[i] = strings.ToUpper(p)
		} else {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, "")
}

// RouteRegistrar diimplementasi oleh controller yang ingin register custom routes
// daripada mengandalkan auto-dispatch.
type RouteRegistrar interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

// GetRouteRegistrars mengembalikan semua controller yang mengimplementasi RouteRegistrar
func GetRouteRegistrars() map[string]RouteRegistrar {
	result := make(map[string]RouteRegistrar)
	for key, t := range controllerRegistry {
		v := reflect.New(t)
		if reg, ok := v.Interface().(RouteRegistrar); ok {
			result[key] = reg
		}
	}
	return result
}

// RegisterParamRoute mendaftarkan definisi route untuk suatu method.
// Format: "/:category/:id" → paramCount=2, names={"category":0, "id":1}
func RegisterParamRoute(controllerKey, method, routePattern string) {
	key := controllerKey + "." + method
	paramRouteRegistry[key] = routePattern

	names := make(map[string]int)
	idx := 0
	for _, segment := range strings.Split(routePattern, "/") {
		if strings.HasPrefix(segment, ":") {
			names[segment[1:]] = idx
			idx++
		}
	}
	paramNameRegistry[key] = names
}

// CheckPathParamCount memvalidasi jumlah path params sesuai registrasi.
func CheckPathParamCount(controllerKey, method string, actualCount int) bool {
	key := controllerKey + "." + method
	route, registered := paramRouteRegistry[key]
	if !registered {
		return true
	}
	expected := 0
	for _, s := range strings.Split(route, "/") {
		if strings.HasPrefix(s, ":") {
			expected++
		}
	}
	return actualCount == expected
}

// GetParamNames mengembalikan mapping nama→index untuk suatu method.
func GetParamNames(controllerKey, method string) map[string]int {
	key := controllerKey + "." + method
	return paramNameRegistry[key]
}

// PathParam mengambil path parameter berdasarkan nama atau index.
// Pakai nama:   PathParam(c, "category") ← lebih readable
// Pakai index:  PathParam(c, 0)          ← tetap didukung
func PathParam(c *gin.Context, keyOrIndex interface{}) string {
	// Ambil values dari context
	val, ok := c.Get("_auto_path_params")
	if !ok {
		return ""
	}
	params, ok := val.([]string)
	if !ok {
		return ""
	}

	switch k := keyOrIndex.(type) {
	case string:
		// Lookup by name dari mapping yang disimpan di context
		namesVal, namesOk := c.Get("_auto_param_names")
		if namesOk {
			if names, ok := namesVal.(map[string]int); ok {
				if idx, exists := names[k]; exists && idx < len(params) {
					return params[idx]
				}
			}
		}
		return ""
	case int:
		if k >= 0 && k < len(params) {
			return params[k]
		}
		return ""
	default:
		return ""
	}
}

func RegisterController(controller interface{}) {
	t := reflect.TypeOf(controller)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	pkg := t.PkgPath()
	parts := strings.Split(pkg, "/")
	moduleIndex := -1
	for i := 0; i < len(parts); i++ {
		if parts[i] == "Module" && i+1 < len(parts) {
			moduleIndex = i + 1
			break
		}
	}
	if moduleIndex == -1 {
		return
	}
	moduleName := parts[moduleIndex]
	name := t.Name()
	key := moduleName + "." + name
	controllerRegistry[key] = t

	// Auto-detect route params dari struct tag `route`
	// Format: `route:"show/:id"` atau `route:"detail/:category/:id"`
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("route")
		if tag == "" {
			continue
		}
		tagParts := strings.Split(tag, "/")
		methodName := tagParts[0]
		routePattern := "/" + strings.Join(tagParts[1:], "/")
		pascalMethod := toPascal(methodName)
		paramKey := key + "." + pascalMethod
		names := make(map[string]int)
		idx := 0
		for _, segment := range strings.Split(routePattern, "/") {
			if strings.HasPrefix(segment, ":") {
				names[segment[1:]] = idx
				idx++
			}
		}
		paramRouteRegistry[paramKey] = routePattern
		paramNameRegistry[paramKey] = names
	}
}

// ListControllers mengembalikan daftar semua controller yang ter-register beserta method-nya.
func ListControllers() []ControllerInfo {
	var list []ControllerInfo
	for key, t := range controllerRegistry {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			continue
		}
		info := ControllerInfo{
			Module:     parts[0],
			Controller: parts[1],
		}
		v := reflect.New(t)
		for i := 0; i < v.NumMethod(); i++ {
			method := v.Method(i)
			mt := method.Type()
			// Hanya ambil method yang punya 1 arg *gin.Context
			if mt.NumIn() == 1 && mt.In(0) == reflect.TypeOf(&gin.Context{}) {
				info.Methods = append(info.Methods, v.Type().Method(i).Name)
			}
		}
		list = append(list, info)
	}
	return list
}

type ControllerInfo struct {
	Module     string
	Controller string
	Methods    []string
}

func Dispatch(c *gin.Context, moduleName, controllerName, methodName string) bool {
	key := moduleName + "." + controllerName
	t, ok := controllerRegistry[key]
	if !ok {
		log.Println("dispatch not found:", key)
		return false
	}
	v := reflect.New(t)
	if setter, ok := v.Interface().(requestContextSetter); ok {
		setter.SetRequestContext(c, moduleName, controllerName, methodName)
	}
	m := v.MethodByName(methodName)
	if !m.IsValid() {
		log.Println("dispatch method not found:", key, methodName)
		return false
	}
	if m.Type().NumIn() != 1 {
		log.Println("dispatch invalid args:", key, methodName)
		return false
	}
	argType := m.Type().In(0)
	if argType != reflect.TypeOf(&gin.Context{}) {
		log.Println("dispatch invalid arg type:", key, methodName)
		return false
	}
	if fn := runtime.FuncForPC(m.Pointer()); fn != nil {
		file, line := fn.FileLine(m.Pointer())
		c.Set("trace.at", fmt.Sprintf("%s:%d", filepath.Base(file), line))
	}
	m.Call([]reflect.Value{reflect.ValueOf(c)})
	return true
}
