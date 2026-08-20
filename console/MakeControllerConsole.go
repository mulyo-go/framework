package console

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type MakeControllerConsole struct{}

func init() {
	Register(&MakeControllerConsole{})
}

func (c *MakeControllerConsole) Name() string {
	return "make:controller"
}

func (c *MakeControllerConsole) Aliases() []string {
	return []string{"make-controller"}
}

func (c *MakeControllerConsole) Usage() string {
	return "<ControllerName> [-m <Module/Plugin>] [--resource]"
}

func (c *MakeControllerConsole) Description() string {
	return "create a new controller file (JSON API by default, or full CRUD + GBR Views with --resource)"
}

func toKebab(s string) string {
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := matchFirstCap.ReplaceAllString(s, "${1}-${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}-${2}")
	return strings.ToLower(snake)
}

func (c *MakeControllerConsole) Run(args []string) error {
	var rawName string
	var moduleName string
	var isResource bool

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "-m" || arg == "-p" || arg == "--module" || arg == "--plugin" {
			if i+1 < len(args) {
				moduleName = strings.TrimSpace(args[i+1])
				i++
			}
		} else if strings.HasPrefix(arg, "-m=") || strings.HasPrefix(arg, "--module=") {
			moduleName = strings.TrimSpace(strings.SplitN(arg, "=", 2)[1])
		} else if strings.HasPrefix(arg, "-p=") || strings.HasPrefix(arg, "--plugin=") {
			moduleName = strings.TrimSpace(strings.SplitN(arg, "=", 2)[1])
		} else if arg == "--resource" || arg == "-r" || arg == "--crud" {
			isResource = true
		} else if !strings.HasPrefix(arg, "-") && rawName == "" {
			rawName = arg
		}
	}

	if rawName == "" {
		PrintCommandHelp(c.Name())
		return fmt.Errorf("controller name is required (example: mulyo-go make:controller ProductController -m Kasir --resource)")
	}

	if strings.Contains(rawName, "/") || strings.Contains(rawName, "\\") {
		parts := strings.FieldsFunc(rawName, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		if len(parts) >= 2 {
			moduleName = parts[0]
			rawName = parts[len(parts)-1]
		}
	}

	ctrlName := rawName
	if !strings.HasSuffix(ctrlName, "Controller") {
		ctrlName += "Controller"
	}
	baseName := strings.TrimSuffix(ctrlName, "Controller")

	var targetDir string
	var pkgName string
	var viewBasePath string
	var routePrefix string

	if moduleName != "" {
		targetDir = filepath.Join("Module", moduleName, "Controller")
		pkgName = "controller"
		viewBasePath = filepath.Join("Module", moduleName, "View", baseName)
		routePrefix = "/" + toKebab(moduleName) + "/" + toKebab(baseName)
	} else {
		targetDir = filepath.Join("app", "Http", "Controllers")
		pkgName = "controllers"
		viewBasePath = filepath.Join("Template", baseName)
		routePrefix = "/" + toKebab(baseName)
	}

	_ = os.MkdirAll(targetDir, 0755)
	filePath := filepath.Join(targetDir, ctrlName+".go")

	var content string
	if isResource {
		viewRelPrefix := moduleName + "/View/" + baseName
		if moduleName == "" {
			viewRelPrefix = baseName
		}

		tagGet := string(rune(96)) + "route:\"get/:id\"" + string(rune(96))
		tagEdit := string(rune(96)) + "route:\"edit/:id\"" + string(rune(96))
		content = "package " + pkgName + "\n\n" +
			"import (\n" +
			"\t\"net/http\"\n" +
			"\t\"strconv\"\n\n" +
			"\tcontrollers \"mulyo-go/app/Http/Controllers\"\n" +
			"\tdispatcher \"github.com/mulyo-go/framework/http/dispatcher\"\n" +
			"\t\"github.com/gin-gonic/gin\"\n" +
			")\n\n" +
			"type " + ctrlName + " struct {\n" +
			"\tcontrollers.BaseController\n\n" +
			"\t_Get  struct{} " + tagGet + "\n" +
			"\t_Edit struct{} " + tagEdit + "\n" +
			"}\n\n" +
			"func init() {\n" +
			"\tdispatcher.RegisterController(&" + ctrlName + "{})\n" +
			"}\n\n" +
			"// Index displays the main list view\n" +
			"func (c *" + ctrlName + ") Index(ctx *gin.Context) {\n" +
			"\tc.Render(ctx, \"" + viewRelPrefix + "/Index\", gin.H{\n" +
			"\t\t\"Title\": \"" + baseName + " Management\",\n" +
			"\t})\n" +
			"}\n\n" +
			"// DataTable returns server-side JSON for DataTables\n" +
			"func (c *" + ctrlName + ") DataTable(ctx *gin.Context) {\n" +
			"\t// query := c.DB().Table(\"" + strings.ToLower(baseName) + "\").Where(\"deleted_at IS NULL\")\n" +
			"\t// dt := helper.NewDataTable(query, ctx).WithSearch([]string{\"name\"})\n" +
			"\t// resp, _ := dt.Build(&rows)\n" +
			"\tctx.JSON(http.StatusOK, gin.H{\"data\": []any{}})\n" +
			"}\n\n" +
			"// Create displays the create form\n" +
			"func (c *" + ctrlName + ") Create(ctx *gin.Context) {\n" +
			"\tc.Render(ctx, \"" + viewRelPrefix + "/Create\", gin.H{\n" +
			"\t\t\"Title\": \"Create " + baseName + "\",\n" +
			"\t})\n" +
			"}\n\n" +
			"// Store handles saving new data\n" +
			"func (c *" + ctrlName + ") Store(ctx *gin.Context) {\n" +
			"\tname := ctx.PostForm(\"name\")\n" +
			"\tif name == \"\" {\n" +
			"\t\tctx.JSON(http.StatusBadRequest, gin.H{\"success\": false, \"message\": \"Name is required\"})\n" +
			"\t\treturn\n" +
			"\t}\n" +
			"\tctx.JSON(http.StatusOK, gin.H{\"success\": true, \"message\": \"Data created successfully\"})\n" +
			"}\n\n" +
			"// Edit displays the edit form\n" +
			"func (c *" + ctrlName + ") Edit(ctx *gin.Context) {\n" +
			"\tidStr := dispatcher.PathParam(ctx, \"id\")\n" +
			"\tif idStr == \"\" {\n" +
			"\t\tidStr = ctx.Query(\"id\")\n" +
			"\t}\n" +
			"\tid, _ := strconv.Atoi(idStr)\n" +
			"\tc.Render(ctx, \"" + viewRelPrefix + "/Edit\", gin.H{\n" +
			"\t\t\"Title\": \"Edit " + baseName + "\",\n" +
			"\t\t\"ID\":    id,\n" +
			"\t})\n" +
			"}\n\n" +
			"// Update handles updating data\n" +
			"func (c *" + ctrlName + ") Update(ctx *gin.Context) {\n" +
			"\tid := ctx.PostForm(\"id\")\n" +
			"\tctx.JSON(http.StatusOK, gin.H{\"success\": true, \"message\": \"Data \" + id + \" updated successfully\"})\n" +
			"}\n\n" +
			"// Delete handles soft-deleting data\n" +
			"func (c *" + ctrlName + ") Delete(ctx *gin.Context) {\n" +
			"\tid := ctx.PostForm(\"id\")\n" +
			"\tctx.JSON(http.StatusOK, gin.H{\"success\": true, \"message\": \"Data \" + id + \" deleted successfully\"})\n" +
			"}\n"

		_ = os.MkdirAll(viewBasePath, 0755)
		createResourceViews(viewBasePath, baseName, routePrefix)
	} else {
		content = "package " + pkgName + "\n\n" +
			"import (\n" +
			"\t\"net/http\"\n\n" +
			"\tcontrollers \"mulyo-go/app/Http/Controllers\"\n" +
			"\tdispatcher \"github.com/mulyo-go/framework/http/dispatcher\"\n" +
			"\t\"github.com/gin-gonic/gin\"\n" +
			")\n\n" +
			"type " + ctrlName + " struct {\n" +
			"\tcontrollers.BaseController\n" +
			"}\n\n" +
			"func init() {\n" +
			"\tdispatcher.RegisterController(&" + ctrlName + "{})\n" +
			"}\n\n" +
			"// Index returns JSON response\n" +
			"func (c *" + ctrlName + ") Index(ctx *gin.Context) {\n" +
			"\tctx.JSON(http.StatusOK, gin.H{\n" +
			"\t\t\"status\":  \"success\",\n" +
			"\t\t\"message\": \"" + ctrlName + " API Index\",\n" +
			"\t\t\"data\":    nil,\n" +
			"\t})\n" +
			"}\n"
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write controller file: %w", err)
	}

	fmt.Printf("✅ Successfully created controller: %s\n", filePath)
	if isResource {
		fmt.Printf("   📁 Views created in: %s\n", viewBasePath)
	}

	autoRegisterController(moduleName, ctrlName)

	return nil
}

func getRootModuleName() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "mulyo-go"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return "mulyo-go"
}

func autoRegisterController(moduleName, ctrlName string) {
	registryPath := filepath.Join("config", "registry", "controllers.go")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return
	}

	content := string(data)
	rootModule := getRootModuleName()

	var importPath string
	var aliasName string
	if moduleName != "" {
		importPath = rootModule + "/Module/" + moduleName + "/Controller"
		aliasName = strings.ToLower(moduleName[:1]) + moduleName[1:] + "Controller"
	} else {
		importPath = rootModule + "/app/Http/Controllers"
		aliasName = "appControllers"
	}

	regStatement := fmt.Sprintf("\tdispatcher.RegisterController(&%s.%s{})", aliasName, ctrlName)
	if strings.Contains(content, "&"+aliasName+"."+ctrlName+"{}") {
		return
	}

	// 1. Ensure import is present
	if !strings.Contains(content, `"`+importPath+`"`) {
		importEntry := fmt.Sprintf("\t%s \"%s\"", aliasName, importPath)
		if idx := strings.Index(content, "import ("); idx != -1 {
			endImport := strings.Index(content[idx:], ")")
			if endImport != -1 {
				actualEnd := idx + endImport
				content = content[:actualEnd] + importEntry + "\n" + content[actualEnd:]
			}
		}
	}

	// 2. Insert into RegisterControllers()
	if idx := strings.Index(content, "func RegisterControllers()"); idx != -1 {
		lastBrace := strings.LastIndex(content[idx:], "}")
		if lastBrace != -1 {
			actualBrace := idx + lastBrace
			content = content[:actualBrace] + regStatement + "\n" + content[actualBrace:]
		}
	}

	if err := os.WriteFile(registryPath, []byte(content), 0644); err == nil {
		fmt.Printf("   🔗 Auto-registered in: %s\n", registryPath)
	}
}

func createResourceViews(viewDir, baseName, routePrefix string) {
	indexView := fmt.Sprintf("{{ define \"toolbar\" }}\n" +
		"<div class=\"app-toolbar pt-6 pb-2\" id=\"kt_app_toolbar\">\n" +
		"    <div class=\"app-container container-fluid d-flex align-items-stretch\" id=\"kt_app_toolbar_container\">\n" +
		"        <div class=\"app-toolbar-wrapper d-flex flex-stack flex-wrap gap-4 w-100\">\n" +
		"            <div class=\"page-title d-flex flex-column justify-content-center gap-1 me-3\">\n" +
		"                <h1 class=\"page-heading d-flex flex-column justify-content-center text-gray-900 fw-bold fs-3 m-0\">{{ .Title }}</h1>\n" +
		"                <ul class=\"breadcrumb breadcrumb-separatorless fw-semibold fs-7 my-0\">\n" +
		"                    <li class=\"breadcrumb-item text-muted\"><a class=\"text-muted text-hover-primary\" href=\"/\">Home</a></li>\n" +
		"                    <li class=\"breadcrumb-item\"><span class=\"bullet bg-gray-500 w-5px h-2px\"></span></li>\n" +
		"                    <li class=\"breadcrumb-item text-gray-800\">%s</li>\n" +
		"                </ul>\n" +
		"            </div>\n" +
		"            <div class=\"d-flex align-items-center gap-2 gap-lg-3\">\n" +
		"                <a href=\"%s/create\" class=\"btn btn-sm btn-primary\">\n" +
		"                    <i class=\"ki-duotone ki-plus fs-2\"></i> Add New %s\n" +
		"                </a>\n" +
		"            </div>\n" +
		"        </div>\n" +
		"    </div>\n" +
		"</div>\n" +
		"{{ end }}\n\n" +
		"{{ define \"content\" }}\n" +
		"<div class=\"card card-table-custom\">\n" +
		"    <div class=\"card-header border-0 pt-6\">\n" +
		"        <div class=\"card-title\">\n" +
		"            <div class=\"d-flex align-items-center position-relative my-1\">\n" +
		"                <i class=\"ki-duotone ki-magnifier fs-3 position-absolute ms-5\"><span class=\"path1\"></span><span class=\"path2\"></span></i>\n" +
		"                <input type=\"text\" data-kt-filter=\"search\" class=\"form-control form-control-solid w-250px ps-13\" placeholder=\"Search %s...\" />\n" +
		"            </div>\n" +
		"        </div>\n" +
		"    </div>\n" +
		"    <div class=\"card-body pt-0\">\n" +
		"        <div class=\"table-responsive\">\n" +
		"            <table class=\"table align-middle table-row-dashed fs-6 gy-5\" id=\"kt_datatable\">\n" +
		"                <thead>\n" +
		"                    <tr class=\"text-start text-gray-500 fw-bold fs-7 text-uppercase gs-0\">\n" +
		"                        <th class=\"min-w-70px\">ID</th>\n" +
		"                        <th class=\"min-w-150px\">Name</th>\n" +
		"                        <th class=\"text-end min-w-100px\">Actions</th>\n" +
		"                    </tr>\n" +
		"                </thead>\n" +
		"                <tbody class=\"text-gray-600 fw-semibold\">\n" +
		"                </tbody>\n" +
		"            </table>\n" +
		"        </div>\n" +
		"    </div>\n" +
		"</div>\n" +
		"{{ end }}\n\n" +
		"@push('js')\n" +
		"<script src=\"https://cdn.jsdelivr.net/npm/sweetalert2@11\"></script>\n" +
		"<script>\n" +
		"$(document).ready(function() {\n" +
		"    // DataTable initialization\n" +
		"});\n" +
		"</script>\n" +
		"@endpush\n\n" +
		"{{ template \"Metronic/layout.gbr.html\" . }}\n", baseName, routePrefix, baseName, baseName)

	createView := fmt.Sprintf("{{ define \"toolbar\" }}\n" +
		"<div class=\"app-toolbar pt-6 pb-2\" id=\"kt_app_toolbar\">\n" +
		"    <div class=\"app-container container-fluid d-flex align-items-stretch\" id=\"kt_app_toolbar_container\">\n" +
		"        <div class=\"app-toolbar-wrapper d-flex flex-stack flex-wrap gap-4 w-100\">\n" +
		"            <div class=\"page-title d-flex flex-column justify-content-center gap-1 me-3\">\n" +
		"                <h1 class=\"page-heading d-flex flex-column justify-content-center text-gray-900 fw-bold fs-3 m-0\">{{ .Title }}</h1>\n" +
		"            </div>\n" +
		"            <div class=\"d-flex align-items-center gap-2 gap-lg-3\">\n" +
		"                <a href=\"%s\" class=\"btn btn-sm btn-light\">\n" +
		"                    <i class=\"ki-duotone ki-arrow-left fs-2\"></i> Back to List\n" +
		"                </a>\n" +
		"            </div>\n" +
		"        </div>\n" +
		"    </div>\n" +
		"</div>\n" +
		"{{ end }}\n\n" +
		"{{ define \"content\" }}\n" +
		"<div class=\"card card-table-custom\">\n" +
		"    <div class=\"card-header border-0 pt-6\">\n" +
		"        <h3 class=\"fw-bold text-gray-800 fs-4 m-0\">Add New %s</h3>\n" +
		"    </div>\n" +
		"    <div class=\"card-body pt-0\">\n" +
		"        <form class=\"form\" id=\"createForm\" method=\"POST\" action=\"%s/store\">\n" +
		"            @csrf\n" +
		"            <div class=\"mb-5\">\n" +
		"                <label class=\"fs-6 fw-semibold mb-2 required\">Name</label>\n" +
		"                <input type=\"text\" class=\"form-control form-control-solid\" placeholder=\"Enter name\" name=\"name\" id=\"name\" required />\n" +
		"            </div>\n" +
		"            <div class=\"d-flex justify-content-end pt-5\">\n" +
		"                <a href=\"%s\" class=\"btn btn-light me-3\">Cancel</a>\n" +
		"                <button type=\"submit\" class=\"btn btn-primary\" id=\"btnSubmit\">\n" +
		"                    <span class=\"indicator-label\"><i class=\"ki-duotone ki-check fs-2 me-1\"></i> Save</span>\n" +
		"                    <span class=\"indicator-progress d-none\">Saving...<span class=\"spinner-border spinner-border-sm align-middle ms-2\"></span></span>\n" +
		"                </button>\n" +
		"            </div>\n" +
		"        </form>\n" +
		"    </div>\n" +
		"</div>\n" +
		"{{ end }}\n\n" +
		"@push('js')\n" +
		"<script src=\"https://cdn.jsdelivr.net/npm/sweetalert2@11\"></script>\n" +
		"<script>\n" +
		"$(document).ready(function() {\n" +
		"    $('#createForm').on('submit', function(e) {\n" +
		"        e.preventDefault();\n" +
		"        var formData = $(this).serializeArray();\n" +
		"        if (typeof getCsrfToken === 'function') {\n" +
		"            formData.push({ name: '_token', value: getCsrfToken() });\n" +
		"        }\n" +
		"        $('#btnSubmit').attr('disabled', true).find('.indicator-label').addClass('d-none');\n" +
		"        $('#btnSubmit').find('.indicator-progress').removeClass('d-none');\n" +
		"        $.ajax({\n" +
		"            url: '%s/store',\n" +
		"            type: 'POST',\n" +
		"            data: formData,\n" +
		"            dataType: 'json',\n" +
		"            success: function(resp) {\n" +
		"                Swal.fire({\n" +
		"                    text: resp.message || 'Data created successfully',\n" +
		"                    icon: 'success',\n" +
		"                    buttonsStyling: false,\n" +
		"                    confirmButtonText: 'OK',\n" +
		"                    customClass: { confirmButton: 'btn btn-primary' }\n" +
		"                }).then(function() {\n" +
		"                    window.location.href = '%s';\n" +
		"                });\n" +
		"            },\n" +
		"            error: function(xhr) {\n" +
		"                $('#btnSubmit').attr('disabled', false).find('.indicator-label').removeClass('d-none');\n" +
		"                $('#btnSubmit').find('.indicator-progress').addClass('d-none');\n" +
		"                var msg = (xhr.responseJSON && xhr.responseJSON.message) ? xhr.responseJSON.message : 'Failed to submit form.';\n" +
		"                Swal.fire({ text: msg, icon: 'error', buttonsStyling: false, confirmButtonText: 'OK', customClass: { confirmButton: 'btn btn-primary' } });\n" +
		"            }\n" +
		"        });\n" +
		"    });\n" +
		"});\n" +
		"</script>\n" +
		"@endpush\n\n" +
		"{{ template \"Metronic/layout.gbr.html\" . }}\n", routePrefix, baseName, routePrefix, routePrefix, routePrefix, routePrefix)

	editView := fmt.Sprintf("{{ define \"toolbar\" }}\n" +
		"<div class=\"app-toolbar pt-6 pb-2\" id=\"kt_app_toolbar\">\n" +
		"    <div class=\"app-container container-fluid d-flex align-items-stretch\" id=\"kt_app_toolbar_container\">\n" +
		"        <div class=\"app-toolbar-wrapper d-flex flex-stack flex-wrap gap-4 w-100\">\n" +
		"            <div class=\"page-title d-flex flex-column justify-content-center gap-1 me-3\">\n" +
		"                <h1 class=\"page-heading d-flex flex-column justify-content-center text-gray-900 fw-bold fs-3 m-0\">{{ .Title }}</h1>\n" +
		"            </div>\n" +
		"            <div class=\"d-flex align-items-center gap-2 gap-lg-3\">\n" +
		"                <a href=\"%s\" class=\"btn btn-sm btn-light\">\n" +
		"                    <i class=\"ki-duotone ki-arrow-left fs-2\"></i> Back to List\n" +
		"                </a>\n" +
		"            </div>\n" +
		"        </div>\n" +
		"    </div>\n" +
		"</div>\n" +
		"{{ end }}\n\n" +
		"{{ define \"content\" }}\n" +
		"<div class=\"card card-table-custom\">\n" +
		"    <div class=\"card-header border-0 pt-6\">\n" +
		"        <h3 class=\"fw-bold text-gray-800 fs-4 m-0\">Edit %s</h3>\n" +
		"    </div>\n" +
		"    <div class=\"card-body pt-0\">\n" +
		"        <form class=\"form\" id=\"editForm\" method=\"POST\" action=\"%s/update\">\n" +
		"            @csrf\n" +
		"            <input type=\"hidden\" name=\"id\" value=\"{{ .ID }}\" />\n" +
		"            <div class=\"mb-5\">\n" +
		"                <label class=\"fs-6 fw-semibold mb-2 required\">Name</label>\n" +
		"                <input type=\"text\" class=\"form-control form-control-solid\" placeholder=\"Enter name\" name=\"name\" id=\"name\" required />\n" +
		"            </div>\n" +
		"            <div class=\"d-flex justify-content-end pt-5\">\n" +
		"                <a href=\"%s\" class=\"btn btn-light me-3\">Cancel</a>\n" +
		"                <button type=\"submit\" class=\"btn btn-primary\" id=\"btnSubmit\">\n" +
		"                    <span class=\"indicator-label\"><i class=\"ki-duotone ki-check fs-2 me-1\"></i> Update</span>\n" +
		"                    <span class=\"indicator-progress d-none\">Updating...<span class=\"spinner-border spinner-border-sm align-middle ms-2\"></span></span>\n" +
		"                </button>\n" +
		"            </div>\n" +
		"        </form>\n" +
		"    </div>\n" +
		"</div>\n" +
		"{{ end }}\n\n" +
		"@push('js')\n" +
		"<script src=\"https://cdn.jsdelivr.net/npm/sweetalert2@11\"></script>\n" +
		"<script>\n" +
		"$(document).ready(function() {\n" +
		"    $('#editForm').on('submit', function(e) {\n" +
		"        e.preventDefault();\n" +
		"        var formData = $(this).serializeArray();\n" +
		"        if (typeof getCsrfToken === 'function') {\n" +
		"            formData.push({ name: '_token', value: getCsrfToken() });\n" +
		"        }\n" +
		"        $('#btnSubmit').attr('disabled', true).find('.indicator-label').addClass('d-none');\n" +
		"        $('#btnSubmit').find('.indicator-progress').removeClass('d-none');\n" +
		"        $.ajax({\n" +
		"            url: '%s/update',\n" +
		"            type: 'POST',\n" +
		"            data: formData,\n" +
		"            dataType: 'json',\n" +
		"            success: function(resp) {\n" +
		"                Swal.fire({\n" +
		"                    text: resp.message || 'Data updated successfully',\n" +
		"                    icon: 'success',\n" +
		"                    buttonsStyling: false,\n" +
		"                    confirmButtonText: 'OK',\n" +
		"                    customClass: { confirmButton: 'btn btn-primary' }\n" +
		"                }).then(function() {\n" +
		"                    window.location.href = '%s';\n" +
		"                });\n" +
		"            },\n" +
		"            error: function(xhr) {\n" +
		"                $('#btnSubmit').attr('disabled', false).find('.indicator-label').removeClass('d-none');\n" +
		"                $('#btnSubmit').find('.indicator-progress').addClass('d-none');\n" +
		"                var msg = (xhr.responseJSON && xhr.responseJSON.message) ? xhr.responseJSON.message : 'Failed to submit form.';\n" +
		"                Swal.fire({ text: msg, icon: 'error', buttonsStyling: false, confirmButtonText: 'OK', customClass: { confirmButton: 'btn btn-primary' } });\n" +
		"            }\n" +
		"        });\n" +
		"    });\n" +
		"});\n" +
		"</script>\n" +
		"@endpush\n\n" +
		"{{ template \"Metronic/layout.gbr.html\" . }}\n", routePrefix, baseName, routePrefix, routePrefix, routePrefix, routePrefix)

	_ = os.WriteFile(filepath.Join(viewDir, "Index.gbr.html"), []byte(indexView), 0644)
	_ = os.WriteFile(filepath.Join(viewDir, "Create.gbr.html"), []byte(createView), 0644)
	_ = os.WriteFile(filepath.Join(viewDir, "Edit.gbr.html"), []byte(editView), 0644)
}
