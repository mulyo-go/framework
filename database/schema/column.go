package schema

import "fmt"

type Column struct {
	Name            string
	Type            string
	Length          int
	Precision       int
	Scale           int
	IsNullable      bool
	IsPrimary       bool
	IsAutoIncrement bool
	IsUnique        bool
	HasDefault      bool
	DefaultValue    any
	Comment         string
}

func (c *Column) Nullable() *Column {
	c.IsNullable = true
	return c
}

func (c *Column) NotNull() *Column {
	c.IsNullable = false
	return c
}

func (c *Column) Default(val any) *Column {
	c.HasDefault = true
	c.DefaultValue = val
	return c
}

func (c *Column) Unique() *Column {
	c.IsUnique = true
	return c
}

func (c *Column) Primary() *Column {
	c.IsPrimary = true
	return c
}

func (c *Column) AutoIncrement() *Column {
	c.IsAutoIncrement = true
	return c
}

func (c *Column) SetComment(comment string) *Column {
	c.Comment = comment
	return c
}

func formatDefaultValue(val any) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}
