package schema

import (
	"fmt"
	"strings"
)

// CompileCreate generates the CREATE TABLE SQL query based on dialect
func CompileCreate(driver string, b *Blueprint) []string {
	statements := make([]string, 0)
	var colDefs []string
	var primaryKeys []string

	driver = strings.ToLower(driver)

	for _, col := range b.Columns {
		def := compileColumn(driver, col)
		colDefs = append(colDefs, def)
		if col.IsPrimary && !col.IsAutoIncrement {
			primaryKeys = append(primaryKeys, quoteIdentifier(driver, col.Name))
		}
	}

	if len(primaryKeys) > 0 {
		colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	tableName := quoteIdentifier(driver, b.TableName)
	sql := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", tableName, strings.Join(colDefs, ",\n  "))
	if driver == "mysql" {
		sql += " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"
	}
	statements = append(statements, sql)

	// Add indexes
	for _, idx := range b.Indexes {
		idxName := fmt.Sprintf("%s_%s_idx", b.TableName, strings.Join(idx.Columns, "_"))
		var quotedCols []string
		for _, c := range idx.Columns {
			quotedCols = append(quotedCols, quoteIdentifier(driver, c))
		}
		if idx.IsUnique {
			statements = append(statements, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)",
				quoteIdentifier(driver, idxName), tableName, strings.Join(quotedCols, ", ")))
		} else {
			statements = append(statements, fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
				quoteIdentifier(driver, idxName), tableName, strings.Join(quotedCols, ", ")))
		}
	}

	return statements
}

// CompileAlter generates ALTER TABLE statements
func CompileAlter(driver string, b *Blueprint) []string {
	statements := make([]string, 0)
	driver = strings.ToLower(driver)
	tableName := quoteIdentifier(driver, b.TableName)

	for _, col := range b.Columns {
		def := compileColumn(driver, col)
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, def))
	}

	for _, colName := range b.DropCols {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, quoteIdentifier(driver, colName)))
	}

	return statements
}

func quoteIdentifier(driver, name string) string {
	switch driver {
	case "mysql":
		return fmt.Sprintf("`%s`", name)
	case "postgres":
		return fmt.Sprintf("\"%s\"", name)
	default:
		return fmt.Sprintf("\"%s\"", name)
	}
}

func compileColumn(driver string, col *Column) string {
	name := quoteIdentifier(driver, col.Name)
	var typeDef string

	switch col.Type {
	case "bigInteger":
		if col.IsAutoIncrement {
			switch driver {
			case "mysql":
				typeDef = "BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY"
			case "postgres":
				typeDef = "BIGSERIAL PRIMARY KEY"
			default:
				typeDef = "INTEGER PRIMARY KEY AUTOINCREMENT"
			}
			return fmt.Sprintf("%s %s", name, typeDef)
		}
		typeDef = "BIGINT"
	case "integer":
		if col.IsAutoIncrement {
			switch driver {
			case "mysql":
				typeDef = "INT UNSIGNED AUTO_INCREMENT PRIMARY KEY"
			case "postgres":
				typeDef = "SERIAL PRIMARY KEY"
			default:
				typeDef = "INTEGER PRIMARY KEY AUTOINCREMENT"
			}
			return fmt.Sprintf("%s %s", name, typeDef)
		}
		typeDef = "INT"
	case "tinyInteger":
		typeDef = "TINYINT"
		if driver == "postgres" {
			typeDef = "SMALLINT"
		}
	case "smallInteger":
		typeDef = "SMALLINT"
	case "string":
		length := col.Length
		if length <= 0 {
			length = 255
		}
		typeDef = fmt.Sprintf("VARCHAR(%d)", length)
	case "text":
		typeDef = "TEXT"
	case "longText":
		switch driver {
		case "mysql":
			typeDef = "LONGTEXT"
		default:
			typeDef = "TEXT"
		}
	case "boolean":
		switch driver {
		case "postgres":
			typeDef = "BOOLEAN"
		default:
			typeDef = "TINYINT(1)"
		}
	case "decimal":
		p := col.Precision
		s := col.Scale
		if p <= 0 {
			p = 10
		}
		typeDef = fmt.Sprintf("DECIMAL(%d, %d)", p, s)
	case "float":
		typeDef = "FLOAT"
	case "double":
		typeDef = "DOUBLE"
		if driver == "postgres" {
			typeDef = "DOUBLE PRECISION"
		}
	case "date":
		typeDef = "DATE"
	case "time":
		typeDef = "TIME"
	case "datetime":
		switch driver {
		case "postgres":
			typeDef = "TIMESTAMP"
		default:
			typeDef = "DATETIME"
		}
	case "timestamp":
		switch driver {
		case "postgres":
			typeDef = "TIMESTAMP"
		default:
			typeDef = "TIMESTAMP"
		}
	default:
		typeDef = col.Type
	}

	parts := []string{name, typeDef}

	if col.IsNullable {
		parts = append(parts, "NULL")
	} else {
		parts = append(parts, "NOT NULL")
	}

	if col.HasDefault {
		parts = append(parts, "DEFAULT "+formatDefaultValue(col.DefaultValue))
	}

	if col.IsUnique {
		parts = append(parts, "UNIQUE")
	}

	return strings.Join(parts, " ")
}
