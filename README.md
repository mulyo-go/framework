# Mulyo Go Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/mulyo-go/framework.svg)](https://pkg.go.dev/github.com/mulyo-go/framework)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Mulyo Go** is an expressive, high-performance modular HMVC web framework for Go, combining the raw speed, concurrency, and type-safety of Go with the unmatched Developer Experience (DX) of Laravel.

---

## 📑 Table of Contents

- [Features](#-features)
- [Installation](#-installation)
- [CLI Tool Commands & Artisan](#-cli-tool-commands--artisan)
- [Module Cache & Maintenance](#-module-cache--maintenance)
- [Quick Start](#-quick-start)
- [Architecture & HMVC Directory Structure](#-architecture--hmvc-directory-structure)
- [Server-Side DataTables (`helper.NewDataTable`)](#-server-side-datatables-helpernewdatatable)
- [GBR Template Engine (Go Blade Renderer)](#-gbr-template-engine-go-blade-renderer)
- [Routing & Auto-Dispatcher](#-routing--auto-dispatcher)
- [Controllers & Scaffolding](#-controllers--scaffolding)
- [Session Management](#-session-management)
- [Database & Multi-DB (GORM)](#-database--multi-db-gorm)
- [Artisan Console Commands (`app/Console`)](#-artisan-console-commands-appconsole)
- [Global Helpers & Utilities](#-global-helpers--utilities)
- [Subpackages Overview](#-subpackages-overview)
- [License](#-license)

---

## 🚀 Features

- 🎨 **GBR Template Engine**: True Laravel Blade syntax preprocessor with `@if`, `@foreach`, `@extends`, `@section`, `@yield`, `@include`, `@csrf`, `@auth`, `@push`, `@stack`, and automatic `.gbr.html` extension resolution.
- ⚡ **Auto-Dispatcher & Conventional Routing**: No need to write hundreds of manual route definitions. Conventions map `/module/controller/action` automatically with struct tag parameter matching (`_Get`, `_Edit`).
- 📊 **Server-Side DataTables**: Production-ready `helper.NewDataTable` with global search, column sorting, date-range filtering, and whitelist security.
- 🛠️ **Artisan CLI Tooling (`mulyo-go`)**: Rich CLI with code generators (`make:controller`, `make:model`, `make:migration`, `make:command`), live hot-reload (`mulyo-go dev`, `mulyo-go serve -w -d`), and database migrator.
- 🗄️ **Multi-Database Connection Pools**: First-class support for MySQL, PostgreSQL, SQLite, and Redis caching.
- 🔒 **Security & CSRF Protection**: Built-in CSRF token generation/validation, password hashing (Bcrypt), XSS sanitization, and session guards.
- 📝 **Structured Logging**: Minute-based rolling logs, request tracing, and panic recovery.

---

## 📦 Installation

Install the framework module in your Go project:

```bash
go get github.com/mulyo-go/framework@latest
```

Install the global `mulyo-go` CLI tool:

```bash
go install github.com/mulyo-go/framework/cmd/mulyo-go@latest
```

---

## 🛠️ CLI Tool Commands & Artisan

The `mulyo-go` command-line tool provides a full suite of developer tools:

### Server & Development
```bash
# Start standard HTTP server
mulyo-go serve

# Start HTTP server with live hot-reloading (watch & debug flags)
mulyo-go serve -w -d
mulyo-go serve --watch --debug -p 8080

# Start directly in dev mode (hot reload via Air)
mulyo-go dev
mulyo-go dev -p 3000
```

### Self-Update & Versioning
```bash
# Update the mulyo-go CLI binary to the latest release
mulyo-go self-update
# or shorthand:
mulyo-go update

# Check current CLI version
mulyo-go version
mulyo-go -v
```

### Code Generation & Scaffolding
```bash
# Create a new HMVC module (Controller, Model, View folders)
mulyo-go make:module Kasir

# Create a new controller (with auto-registration in config/registry/controllers.go)
mulyo-go make:controller ProductController -m Kasir --resource

# Create GORM model & migration
mulyo-go make:model Product -m Kasir -t products -mig

# Create database migration & seeder
mulyo-go make:migration create_products_table
mulyo-go make:seeder ProductSeeder

# Create HTTP middleware & console command
mulyo-go make:middleware EnsureAuth
mulyo-go make:command SyncProducts

# Database migration & seed runners
mulyo-go migrate
mulyo-go migrate --rollback
mulyo-go db:seed
mulyo-go db:seed -class ProductSeeder

# Inspect and synchronize routes
mulyo-go route:list
mulyo-go route:generate
```

---

## 🧹 Module Cache & Maintenance

When upgrading framework tags or cleaning up cached proxy modules, run:

```bash
# Clear Go module download cache (fixes proxy checksum cache issues)
go clean -modcache

# Tidy dependencies and update go.sum
go mod tidy

# Download latest framework version
go get -u github.com/mulyo-go/framework@latest
```

---

## 📊 Server-Side DataTables (`helper.NewDataTable`)

The framework provides a high-performance helper for server-side processing with jQuery DataTables:

### 1. Controller Implementation

```go
package controller

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/mulyo-go/framework/helper"
    controllers "mulyo-go/app/Http/Controllers"
    "mulyo-go/Module/Kasir/Model"
)

type ProductController struct {
    controllers.BaseController
}

func (c *ProductController) DataTable(ctx *gin.Context) {
    // 1. Prepare Base GORM Query
    query := c.DB().Table("products").Where("deleted_at IS NULL")

    // 2. Build DataTable instance
    dt := helper.NewDataTable(query, ctx).
        WithSearch([]string{"name", "sku", "category"}). // Global search columns
        WithSort(map[string]string{"created_at": "DESC"}). // Default fallback sort
        WithRawColumns([]string{"status", "category_id"}) // Whitelist for _raw where filter

    // 3. Execute and fetch data into slice
    var rows []model.Product
    resp, err := dt.Build(&rows)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 4. Return standard DataTables JSON response
    ctx.JSON(http.StatusOK, resp)
}
```

### 2. Available DataTable Methods

| Method | Description |
| :--- | :--- |
| `helper.NewDataTable(db, ctx)` | Initialize DataTable helper with active GORM DB query and Gin Context. |
| `dt.WithSearch([]string{"col1", "col2"})` | Set columns searchable via global `search[value]`. |
| `dt.WithSort(map[string]string{"col": "DESC"})` | Set default fallback sorting order when no column order is requested by DataTables. |
| `dt.WithRawColumns([]string{"col1", "col2"})` | Whitelist allowed columns for base64 encoded `_raw` parameter queries (SQL injection prevention). |
| `dt.Build(&slice)` | Executes count total, applies filters/searches/orders, sets pagination offset/limit, and populates data into slice pointer. |

### 3. Supported Request Parameters

- `draw`, `start`, `length`: Standard DataTables pagination parameters.
- `search[value]`: Global search query string applied to `WithSearch` columns using `LIKE %keyword%`.
- `order[i][column]` & `order[i][dir]`: Column-specific sorting.
- `tempSearch[column]=val` & `tempOperator[column]=operator`: Advanced column filtering. Supports operators: `=`, `LIKE`, `BETWEEN` (e.g. `2026-01-01 to 2026-01-31`), `IN`, `IS NULL`, `IS NOT NULL`.
- `in_field=status` & `in_search=active,pending`: Multi-value IN filtering.
- `_raw=base64("status = 1")`: Safe base64 encoded raw where condition against whitelist columns.

### 4. Frontend jQuery DataTables Setup

```javascript
$('#productTable').DataTable({
    processing: true,
    serverSide: true,
    ajax: {
        url: '/kasir/product/datatable',
        type: 'POST',
        data: function (d) {
            d._token = $('meta[name="csrf-token"]').attr('content');
        }
    },
    columns: [
        { data: 'id' },
        { data: 'name' },
        { data: 'sku' },
        { data: 'price', render: $.fn.dataTable.render.number('.', ',', 0, 'Rp ') },
        { data: 'stock' }
    ]
});
```

---

## 🎨 GBR Template Engine (Go Blade Renderer)

GBR is a compile-time Blade preprocessor written in pure Go. It supports `.gbr.html` files with layout inheritance, directives, and embedded filesystem support.

### Directives Reference

```html
{{-- 1. Conditionals --}}
@if($isAdmin)
    <span class="badge badge-success">Admin</span>
@elseif($isEditor)
    <span class="badge badge-warning">Editor</span>
@else
    <span class="badge badge-secondary">Member</span>
@endif

{{-- 2. Loops --}}
@foreach($products as $item)
    <tr>
        <td>{{ $item.Name }}</td>
        <td>{{ formatIDR $item.Price }}</td>
    </tr>
@endforeach

{{-- 3. Forms & Security --}}
<form method="POST" action="/kasir/product/update">
    @csrf
    @method('PUT')
    <input type="text" name="name" value="{{ .Product.Name }}">
    <input type="checkbox" name="active" @checked($isActive)>
    <button type="submit">Save</button>
</form>

{{-- 4. Authentication State --}}
@auth
    <span>Hello, {{ .UserName }}</span>
    <a href="/auth/logout">Logout</a>
@endauth

@guest
    <a href="/auth/login">Login</a>
@endguest

{{-- 5. Flash Messages --}}
@session('success')
    <div class="alert alert-success">{{ .FlashSuccess }}</div>
@endsession

{{-- 6. Layout Stacks --}}
@push('scripts')
<script src="/assets/js/custom.js"></script>
@endpush
```

### Auto-Extension Resolution

When calling `c.Render`, the framework automatically resolves the view file path with `.gbr.html`:

```go
c.Render(ctx, "Kasir/View/Product/Index", gin.H{
    "Title": "Product List",
})
// Resolves automatically to: Module/Kasir/View/Product/Index.gbr.html
```

---

## ⚡ Routing & Auto-Dispatcher

Controllers are registered in `config/registry/controllers.go`:

```go
package registry

import (
    dispatcher "github.com/mulyo-go/framework/http/dispatcher"
    kasirController "mulyo-go/Module/Kasir/Controller"
)

func RegisterControllers() {
    dispatcher.RegisterController(&kasirController.ProductController{})
}
```

### URL Mapping Convention

| HTTP Method | URL Path | Method Executed |
| :--- | :--- | :--- |
| `GET` | `/kasir/product` | `Kasir.ProductController.Index(ctx)` |
| `GET` | `/kasir/product/create` | `Kasir.ProductController.Create(ctx)` |
| `POST` | `/kasir/product/store` | `Kasir.ProductController.Store(ctx)` |
| `GET` | `/kasir/product/edit/:id` | `Kasir.ProductController.Edit(ctx)` (via `_Edit` tag) |
| `POST` | `/kasir/product/update` | `Kasir.ProductController.Update(ctx)` |
| `POST` | `/kasir/product/delete` | `Kasir.ProductController.Delete(ctx)` |

---

## 🗄️ Database & Multi-DB (GORM)

Access database connection pools in your controllers:

```go
// Default DB connection
c.DB().Where("status = ?", "active").Find(&products)

// Secondary DB connection
analyticsDB := config.DB("analytics")
analyticsDB.Table("logs").Create(&logEntry)

// ACID Transaction
err := c.DB().Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&order).Error; err != nil {
        return err
    }
    return tx.Model(&Product{}).Where("id = ?", order.ProductID).
        Update("stock", gorm.Expr("stock - ?", order.Qty)).Error
})
```

---

## 🛠️ Artisan Console Commands (`app/Console`)

Create CLI commands in `app/Console/` to run background batch jobs, data synchronization, or scheduled tasks:

```go
package console

import (
    "flag"
    "fmt"
    "github.com/mulyo-go/framework/console"
    "github.com/mulyo-go/framework/config"
)

type SyncUserDataConsole struct{}

func init() {
    console.Register(&SyncUserDataConsole{})
}

func (c *SyncUserDataConsole) Name() string        { return "user:sync" }
func (c *SyncUserDataConsole) Aliases() []string    { return []string{"sync-user"} }
func (c *SyncUserDataConsole) Usage() string        { return "[-id <num>] <target>" }
func (c *SyncUserDataConsole) Description() string  { return "Sync user data from external system" }

func (c *SyncUserDataConsole) Run(args []string) error {
    fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
    id := fs.Int("id", 0, "Target ID")
    if err := fs.Parse(args); err != nil {
        return err
    }

    config.Load()
    db := config.DefaultDB()
    fmt.Printf("🚀 Syncing data for ID: %d\n", *id)
    // Business logic...
    return nil
}
```

Run via terminal:
```bash
go run . user:sync -id 42 master_users
# or
mulyo-go user:sync -id 42 master_users
```

---

## 🧩 Global Helpers & Utilities

Import `"github.com/mulyo-go/framework/helper"`:

- **String Helpers**: `helper.StrSlug`, `helper.StrKebab`, `helper.StrSnake`, `helper.StrCamel`, `helper.StrStudly`, `helper.StrLimit`, `helper.StrRandom`, `helper.StrSquish`.
- **URL & Paths**: `helper.Asset("css/app.css")`, `helper.URLFile("avatar.jpg")`, `helper.StoragePath()`, `helper.PublicPath()`.
- **Formatting**: `helper.FormatIDR(3500000)` &rarr; `"Rp 3.500.000"`, `helper.Ternary`, `helper.Default`.
- **Validation**: `helper.ValidateStruct(req)` with struct validation tags.
- **Password**: `helper.HashPassword(plain)` & `helper.CheckPassword(plain, hash)`.

---

## 📦 Subpackages Overview

| Subpackage | Purpose |
| :--- | :--- |
| `github.com/mulyo-go/framework/helper` | DataTables engine, string manipulation, formatting, Bcrypt password hashing, validation, response utilities |
| `github.com/mulyo-go/framework/gbr` | GBR template compiler, directives preprocessor, layouts, stacks, built-in FuncMap |
| `github.com/mulyo-go/framework/http/dispatcher` | Convention-over-configuration HMVC controller dispatcher & route inspector |
| `github.com/mulyo-go/framework/config` | Environment variables, GORM connection pools, Redis, session manager |
| `github.com/mulyo-go/framework/console` | Artisan command registry, generator commands, migrations runner |
| `github.com/mulyo-go/framework/logger` | Structured rolling file logger & request tracer |
| `github.com/mulyo-go/framework/database/migration` | Declarative Blueprint schema builder & migration tracker |

---

## 📄 License

The Mulyo Go Framework is open-sourced software licensed under the [MIT license](LICENSE).
