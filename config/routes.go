package config

import (
	"net/http"
	"strings"

	dispatcher "github.com/mulyo-go/framework/http/dispatcher"

	"github.com/gin-gonic/gin"
)

var (
	ErrorHandler func(c *gin.Context, status int)
	WebRoutesRegistrar func(r *gin.Engine)
	ApiRoutesRegistrar func(r *gin.Engine)
	RouteAccessChecker func(c *gin.Context, module, controller, method string) bool
)

func SetWebRoutes(fn func(r *gin.Engine)) {
	WebRoutesRegistrar = fn
}

func SetApiRoutes(fn func(r *gin.Engine)) {
	ApiRoutesRegistrar = fn
}

func SetRouteAccessChecker(fn func(c *gin.Context, module, controller, method string) bool) {
	RouteAccessChecker = fn
}

func renderError(c *gin.Context, status int) {
	if ErrorHandler != nil {
		ErrorHandler(c, status)
		return
	}
	c.Status(status)
}

func renderNotFound(c *gin.Context) {
	renderError(c, http.StatusNotFound)
}

var publicModules = map[string]bool{
	"auth":           true,
	"api":            true,
	"sample":         true,
	"sampletemplate": true,
	"lov":            true,
}

var publicPaths = []string{
	"/sample/show-url/",
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/dashboard/sample/chart")
	})

	if RoutingMode() == "manual" {
		if WebRoutesRegistrar != nil {
			WebRoutesRegistrar(r)
		}
	} else {
		registerAutoRoutes(r)
	}

	if ApiRoutesRegistrar != nil {
		ApiRoutesRegistrar(r)
	}
}

func isPublicRoute(module, path string) bool {
	if RoutingMode() == "manual" {
		return true
	}
	if IsWhiteListRoute() {
		return true
	}

	if publicModules[normalizeModuleKey(module)] {
		return true
	}
	lowerPath := strings.ToLower(path)
	for _, pp := range publicPaths {
		if strings.HasPrefix(lowerPath, pp) {
			return true
		}
	}
	return false
}

func normalizeModuleKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func registerAutoRoutes(r *gin.Engine) {
	r.Any("/:module/:controller", func(c *gin.Context) {
		module := c.Param("module")
		controller := c.Param("controller")
		moduleName := toPascal(module)
		controllerName := toPascal(controller) + "Controller"
		methodName := "Index"

		if !isPublicRoute(module, c.Request.URL.Path) {
			if RouteAccessChecker != nil {
				if !RouteAccessChecker(c, moduleName, controllerName, methodName) {
					return
				}
			}
		}

		c.Set("trace.controller", moduleName+"."+controllerName+"."+methodName)

		if !dispatcher.Dispatch(c, moduleName, controllerName, methodName) {
			renderNotFound(c)
		}
	})

	dispatchWithParams := func(c *gin.Context, params []string) {
		module := c.Param("module")
		controller := c.Param("controller")
		action := c.Param("action")

		c.Set("_auto_path_params", params)

		moduleName := toPascal(module)
		controllerName := toPascal(controller) + "Controller"
		methodName := "Index"
		if action != "" {
			methodName = toPascal(action)
		}
		controllerKey := moduleName + "." + controllerName

		if !isPublicRoute(module, c.Request.URL.Path) {
			if RouteAccessChecker != nil {
				if !RouteAccessChecker(c, moduleName, controllerName, methodName) {
					return
				}
			}
		}

		if !dispatcher.CheckPathParamCount(controllerKey, methodName, len(params)) {
			renderNotFound(c)
			return
		}

		if names := dispatcher.GetParamNames(controllerKey, methodName); names != nil {
			c.Set("_auto_param_names", names)
		}

		c.Set("trace.controller", controllerKey+"."+methodName)
		if !dispatcher.Dispatch(c, moduleName, controllerName, methodName) {
			renderNotFound(c)
		}
	}

	r.Any("/:module/:controller/:action", func(c *gin.Context) {
		dispatchWithParams(c, []string{})
	})

	r.Any("/:module/:controller/:action/*rest", func(c *gin.Context) {
		rest := strings.TrimPrefix(c.Param("rest"), "/")
		var params []string
		if rest != "" {
			params = strings.Split(rest, "/")
		}
		dispatchWithParams(c, params)
	})
}

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


func IsWhiteListRoute() bool {
	return Env("WHITE_LIST_ROUTE", "false") == "true"
}
