package helper

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DataTableResponse adalah format JSON standar DataTables Server-Side Processing.
type DataTableResponse struct {
	Draw            int         `json:"draw"`
	RecordsTotal    int64       `json:"recordsTotal"`
	RecordsFiltered int64       `json:"recordsFiltered"`
	Data            interface{} `json:"data"`
}

// DataTable membantu membangun query server-side DataTables dengan GORM.
type DataTable struct {
	db                *gorm.DB
	ctx               *gin.Context
	search            []string          // kolom untuk pencarian global
	sort              map[string]string // default sort
	allowedRawColumns map[string]bool   // whitelist kolom untuk raw where
}

// NewDataTable membuat instance DataTable baru.
func NewDataTable(db *gorm.DB, ctx *gin.Context) *DataTable {
	return &DataTable{
		db:  db,
		ctx: ctx,
	}
}

// WithSearch menentukan kolom yang akan dicari secara global.
func (dt *DataTable) WithSearch(columns []string) *DataTable {
	dt.search = columns
	return dt
}

// WithSort menentukan default sort jika tidak ada order dari DataTables.
func (dt *DataTable) WithSort(sort map[string]string) *DataTable {
	dt.sort = sort
	return dt
}

// WithRawColumns menentukan whitelist kolom yang diizinkan untuk filter raw where.
// Hanya kolom dalam daftar ini yang boleh digunakan di parameter _raw.
func (dt *DataTable) WithRawColumns(columns []string) *DataTable {
	dt.allowedRawColumns = make(map[string]bool, len(columns))
	for _, col := range columns {
		dt.allowedRawColumns[col] = true
	}
	return dt
}

// Build menjalankan query, menerapkan filter/paginasi, dan mengembalikan response.
// dest harus berupa pointer ke slice, misal: &[]map[string]interface{} atau &[]MyStruct.
func (dt *DataTable) Build(dest interface{}) (*DataTableResponse, error) {
	draw := formInt(dt.ctx, "draw")
	start := formInt(dt.ctx, "start")
	length := formInt(dt.ctx, "length")

	// Hitung total records dari base query (tanpa filter pencarian)
	totalDB := dt.db.Session(&gorm.Session{})
	var recordsTotal int64
	if err := totalDB.Count(&recordsTotal).Error; err != nil {
		return nil, err
	}

	// Query yang akan difilter
	filteredDB := dt.db.Session(&gorm.Session{})

	// Terapkan semua filter (GORM chain immutable, harus ditangkap return value-nya)
	filteredDB = dt.applyTempSearch(filteredDB)
	filteredDB = dt.applyRawWhere(filteredDB)
	filteredDB = dt.applyGlobalSearch(filteredDB)
	filteredDB = dt.applyInSearch(filteredDB)

	// Hitung records setelah filter
	var recordsFiltered int64
	countDB := filteredDB.Session(&gorm.Session{})
	if err := countDB.Count(&recordsFiltered).Error; err != nil {
		return nil, err
	}

	// Terapkan sorting
	filteredDB = dt.applySort(filteredDB)

	// Terapkan paginasi (start di DataTables adalah offset 0-based)
	if length > 0 {
		offset := start
		if offset < 0 {
			offset = 0
		}
		filteredDB = filteredDB.Offset(offset).Limit(length)
	}

	// Ambil data
	if err := filteredDB.Find(dest).Error; err != nil {
		return nil, err
	}

	return &DataTableResponse{
		Draw:            draw,
		RecordsTotal:    recordsTotal,
		RecordsFiltered: recordsFiltered,
		Data:            dest,
	}, nil
}

// ---------- filter helpers ----------

func (dt *DataTable) applyTempSearch(db *gorm.DB) *gorm.DB {
	tempSearch := extractMap(dt.ctx, "tempSearch")
	if len(tempSearch) == 0 {
		return db
	}
	tempOperator := extractMap(dt.ctx, "tempOperator")

	for key, value := range tempSearch {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		operator := strings.ToUpper(strings.TrimSpace(tempOperator[key]))
		if operator == "" {
			operator = "="
		}

		col := escapeColumn(key)

		switch operator {
		case "BETWEEN", "NOT BETWEEN":
			parts := strings.Split(value, " to ")
			if len(parts) < 2 {
				parts = strings.Split(value, " ~ ")
			}
			if len(parts) < 2 {
				parts = strings.Split(value, ",")
			}
			if len(parts) == 2 {
				startStr := strings.TrimSpace(parts[0])
				endStr := strings.TrimSpace(parts[1])

				startDate, errStart := time.ParseInLocation("2006-01-02", startStr, time.Local)
				endDate, errEnd := time.ParseInLocation("2006-01-02", endStr, time.Local)

				if errStart == nil && errEnd == nil {
					endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999000000, time.Local)

					startMilli := startDate.UnixMilli()
					endMilli := endDate.UnixMilli()

					startSec := startDate.Unix()
					endSec := endDate.Unix()

					startFormatted := startDate.Format("2006-01-02 15:04:05")
					endFormatted := endDate.Format("2006-01-02 15:04:05")

					if operator == "BETWEEN" {
						db = db.Where("("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?)",
							startMilli, endMilli,
							startSec, endSec,
							startFormatted, endFormatted)
					} else {
						db = db.Where("NOT (("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?))",
							startMilli, endMilli,
							startSec, endSec,
							startFormatted, endFormatted)
					}
				} else {
					if operator == "BETWEEN" {
						db = db.Where(col+" BETWEEN ? AND ?", startStr, endStr)
					} else {
						db = db.Where(col+" NOT BETWEEN ? AND ?", startStr, endStr)
					}
				}
			} else if len(parts) == 1 {
				startStr := strings.TrimSpace(parts[0])
				startDate, errStart := time.ParseInLocation("2006-01-02", startStr, time.Local)
				if errStart == nil {
					endDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 23, 59, 59, 999000000, time.Local)
					startMilli := startDate.UnixMilli()
					endMilli := endDate.UnixMilli()

					startSec := startDate.Unix()
					endSec := endDate.Unix()

					startFormatted := startDate.Format("2006-01-02 15:04:05")
					endFormatted := endDate.Format("2006-01-02 15:04:05")

					if operator == "BETWEEN" {
						db = db.Where("("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?)",
							startMilli, endMilli,
							startSec, endSec,
							startFormatted, endFormatted)
					} else {
						db = db.Where("NOT (("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?))",
							startMilli, endMilli,
							startSec, endSec,
							startFormatted, endFormatted)
					}
				} else {
					db = db.Where(col+" = ?", startStr)
				}
			}
		case "LIKE", "NOT LIKE":
			db = db.Where(col+" "+operator+" ?", "%"+value+"%")
		case "LIKE -%", "NOT LIKE -%":
			op := strings.Replace(operator, " -%", "", -1)
			db = db.Where(col+" "+op+" ?", value+"%")
		case "%- LIKE", "%- NOT LIKE":
			op := strings.Replace(operator, "%- ", "", -1)
			db = db.Where(col+" "+op+" ?", "%"+value)
		case "IN", "NOT IN":
			parts := strings.Split(value, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			db = db.Where(col+" "+operator+" ?", parts)
		case "IS", "IS NOT":
			if strings.ToUpper(value) == "NULL" {
				db = db.Where(col + " " + operator + " NULL")
			} else {
				db = db.Where(col+" "+operator+" ?", value)
			}
		default:
			if startDate, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
				endDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 23, 59, 59, 999000000, time.Local)
				startMilli := startDate.UnixMilli()
				endMilli := endDate.UnixMilli()
				startSec := startDate.Unix()
				endSec := endDate.Unix()
				startFormatted := startDate.Format("2006-01-02 15:04:05")
				endFormatted := endDate.Format("2006-01-02 15:04:05")

				db = db.Where("("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?) OR ("+col+" BETWEEN ? AND ?) OR ("+col+" = ?)",
					startMilli, endMilli,
					startSec, endSec,
					startFormatted, endFormatted,
					value)
			} else {
				db = db.Where(col+" "+operator+" ?", value)
			}
		}
	}
	return db
}

func (dt *DataTable) applyRawWhere(db *gorm.DB) *gorm.DB {
	raw := strings.TrimSpace(formValue(dt.ctx, "_raw"))
	if raw == "" {
		return db
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return db
	}
	decodedStr := strings.TrimSpace(string(decoded))
	if decodedStr == "" {
		return db
	}

	// Parse format: "column operator value"
	// Operator yang diizinkan: =, !=, <>, LIKE, NOT LIKE
	rawWhereRe := regexp.MustCompile(`^([a-zA-Z0-9_.]+)\s+(=|!=|<>|[Ll][Ii][Kk][Ee]|[Nn][Oo][Tt]\s+[Ll][Ii][Kk][Ee])\s+(.+)$`)
	matches := rawWhereRe.FindStringSubmatch(decodedStr)
	if matches == nil {
		// Format tidak valid, skip filter
		return db
	}

	column := matches[1]
	operator := strings.ToUpper(strings.TrimSpace(matches[2]))
	value := strings.TrimSpace(matches[3])

	// Validasi kolom terhadap whitelist
	if len(dt.allowedRawColumns) > 0 && !dt.allowedRawColumns[column] {
		// Kolom tidak ada di whitelist, skip filter
		return db
	}

	// Gunakan escapeColumn untuk membungkus nama kolom dengan backtick
	escapedCol := escapeColumn(column)

	// Bangun query parameterized
	switch operator {
	case "LIKE", "NOT LIKE":
		db = db.Where(escapedCol+" "+operator+" ?", "%"+value+"%")
	default:
		db = db.Where(escapedCol+" "+operator+" ?", value)
	}
	return db
}

func (dt *DataTable) applyGlobalSearch(db *gorm.DB) *gorm.DB {
	searchValue := strings.TrimSpace(formValue(dt.ctx, "search[value]"))
	if searchValue == "" || len(dt.search) == 0 {
		return db
	}

	var conditions []string
	var args []interface{}
	for _, col := range dt.search {
		conditions = append(conditions, escapeColumn(col)+" LIKE ?")
		args = append(args, "%"+searchValue+"%")
	}

	if len(conditions) > 0 {
		return db.Where(strings.Join(conditions, " OR "), args...)
	}
	return db
}

func (dt *DataTable) applyInSearch(db *gorm.DB) *gorm.DB {
	inField := strings.TrimSpace(formValue(dt.ctx, "in_field"))
	inSearch := strings.TrimSpace(formValue(dt.ctx, "in_search"))
	if inField == "" || inSearch == "" {
		return db
	}
	parts := strings.Split(inSearch, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return db.Where(escapeColumn(inField)+" IN ?", parts)
}

func (dt *DataTable) applySort(db *gorm.DB) *gorm.DB {
	columns := parseColumns(dt.ctx)
	orders := parseOrders(dt.ctx)

	// DataTables order (order[column][dir]) punya prioritas tertinggi
	if len(orders) > 0 && len(columns) > 0 {
		for _, o := range orders {
			if o.Column < 0 || o.Column >= len(columns) {
				continue
			}
			col := columns[o.Column]
			if col == "" {
				continue
			}
			db = db.Order(escapeColumn(col) + " " + normalizeDir(o.Dir))
		}
		return db
	}

	// tempSort dari request
	tempSort := extractMap(dt.ctx, "tempSort")
	if len(tempSort) > 0 {
		for col, dir := range tempSort {
			db = db.Order(escapeColumn(col) + " " + normalizeDir(dir))
		}
		return db
	}

	// default sort
	if len(dt.sort) > 0 {
		for col, dir := range dt.sort {
			db = db.Order(escapeColumn(col) + " " + normalizeDir(dir))
		}
	}
	return db
}

// ---------- parser helpers ----------

func formValue(ctx *gin.Context, key string) string {
	return ctx.Request.FormValue(key)
}

func formInt(ctx *gin.Context, key string) int {
	v := formValue(ctx, key)
	n, _ := strconv.Atoi(v)
	return n
}

// extractMap mengambil map dari form data dengan format prefix[key]=value.
func extractMap(ctx *gin.Context, prefix string) map[string]string {
	_ = ctx.Request.ParseForm()
	result := make(map[string]string)
	prefix = prefix + "["
	for key, values := range ctx.Request.Form {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, "]") {
			k := key[len(prefix) : len(key)-1]
			if len(values) > 0 {
				result[k] = values[0]
			}
		}
	}
	return result
}

// parseColumns mengambil array columns[i][data] dari request DataTables.
func parseColumns(ctx *gin.Context) []string {
	_ = ctx.Request.ParseForm()
	cols := make(map[int]string)
	for key, values := range ctx.Request.Form {
		if !strings.HasPrefix(key, "columns[") {
			continue
		}
		rest := key[len("columns["):]
		idxEnd := strings.Index(rest, "]")
		if idxEnd < 0 {
			continue
		}
		idx, err := strconv.Atoi(rest[:idxEnd])
		if err != nil {
			continue
		}
		field := rest[idxEnd:]
		if field == "][data]" && len(values) > 0 {
			cols[idx] = values[0]
		}
	}
	if len(cols) == 0 {
		return nil
	}
	maxIdx := 0
	for k := range cols {
		if k > maxIdx {
			maxIdx = k
		}
	}
	result := make([]string, maxIdx+1)
	for k, v := range cols {
		result[k] = v
	}
	return result
}

// parseOrders mengambil array order[i][column] dan order[i][dir] dari request DataTables.
func parseOrders(ctx *gin.Context) []struct {
	Column int
	Dir    string
} {
	_ = ctx.Request.ParseForm()
	type item struct {
		Column int
		Dir    string
	}
	items := make(map[int]item)
	for key, values := range ctx.Request.Form {
		if !strings.HasPrefix(key, "order[") {
			continue
		}
		rest := key[len("order["):]
		idxEnd := strings.Index(rest, "]")
		if idxEnd < 0 {
			continue
		}
		idx, err := strconv.Atoi(rest[:idxEnd])
		if err != nil {
			continue
		}
		field := rest[idxEnd:]
		it := items[idx]
		if field == "][column]" && len(values) > 0 {
			if c, err := strconv.Atoi(values[0]); err == nil {
				it.Column = c
			}
		} else if field == "][dir]" && len(values) > 0 {
			it.Dir = values[0]
		}
		items[idx] = it
	}
	if len(items) == 0 {
		return nil
	}
	maxIdx := 0
	for k := range items {
		if k > maxIdx {
			maxIdx = k
		}
	}
	result := make([]struct {
		Column int
		Dir    string
	}, maxIdx+1)
	for k, v := range items {
		result[k] = struct {
			Column int
			Dir    string
		}{Column: v.Column, Dir: v.Dir}
	}
	return result
}

// ---------- utility ----------

func normalizeDir(dir string) string {
	if strings.ToUpper(strings.TrimSpace(dir)) == "DESC" {
		return "DESC"
	}
	return "ASC"
}

// escapeColumn membungkus setiap bagian nama kolom dengan backtick,
// mirip behavior PHP: `table`.`column`.
func escapeColumn(key string) string {
	parts := strings.Split(key, ".")
	for i, p := range parts {
		parts[i] = "`" + p + "`"
	}
	return strings.Join(parts, ".")
}
