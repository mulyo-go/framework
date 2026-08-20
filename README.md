# Mulyo Framework

Mulyo is an ultra-fast, lightweight, pure Go Web Framework engine inspired by Laravel.

## 🚀 Features

- 🎨 **GBR Template Engine**: Syntax directives, components, layout inheritance, and runtime compilation.
- ⚡ **Auto & Manual Routing**: Automatic controller dispatching with path parameter matching and customizable access guards.
- 📊 **Server-Side DataTables**: Fluent DataTable query helper for database pagination, search, and sorting.
- 🛠️ **Console CLI Tooling**: Artisan-like CLI command registry & runner.
- 🗄️ **Multi-Database & Redis**: Database connection management for MySQL, PostgreSQL, SQLite, and Redis caching.
- 📝 **Structured Logging**: Minute-based rolling logs, request tracing, and panic recovery.

## 📦 Installation

```bash
go get github.com/mulyo-go/framework
```

## 📖 Subpackages

- `github.com/mulyo-go/framework/gbr`: GBR template engine
- `github.com/mulyo-go/framework/helper`: Datatables, string, crypto/password, response utilities
- `github.com/mulyo-go/framework/http/dispatcher`: Controller dynamic dispatcher
- `github.com/mulyo-go/framework/logger`: Logging subsystem
- `github.com/mulyo-go/framework/console`: CLI console command runner
- `github.com/mulyo-go/framework/config`: App, database, redis, and session manager

## 📄 License

MIT License
