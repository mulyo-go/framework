package schema

type Blueprint struct {
	TableName   string
	Columns     []*Column
	DropCols    []string
	Indexes     []*Index
	ForeignKeys []*ForeignKey
	IsAlter     bool
}

type Index struct {
	Name     string
	Columns  []string
	IsUnique bool
}

type ForeignKey struct {
	Column     string
	RefColumn  string
	RefTable   string
	OnDelete   string
	OnUpdate   string
}

func NewBlueprint(table string, isAlter bool) *Blueprint {
	return &Blueprint{
		TableName: table,
		IsAlter:   isAlter,
		Columns:   make([]*Column, 0),
		DropCols:  make([]string, 0),
		Indexes:   make([]*Index, 0),
	}
}

func (b *Blueprint) addColumn(name, colType string) *Column {
	col := &Column{
		Name:       name,
		Type:       colType,
		IsNullable: false,
	}
	b.Columns = append(b.Columns, col)
	return col
}

// ID creates an auto-incrementing BigInteger (64-bit) primary key named "id" (or custom name)
func (b *Blueprint) ID(name ...string) *Column {
	colName := "id"
	if len(name) > 0 && name[0] != "" {
		colName = name[0]
	}
	col := b.addColumn(colName, "bigInteger")
	col.IsPrimary = true
	col.IsAutoIncrement = true
	return col
}

func (b *Blueprint) Increments(name string) *Column {
	col := b.addColumn(name, "integer")
	col.IsPrimary = true
	col.IsAutoIncrement = true
	return col
}

func (b *Blueprint) BigIncrements(name string) *Column {
	col := b.addColumn(name, "bigInteger")
	col.IsPrimary = true
	col.IsAutoIncrement = true
	return col
}

func (b *Blueprint) String(name string, length ...int) *Column {
	col := b.addColumn(name, "string")
	if len(length) > 0 && length[0] > 0 {
		col.Length = length[0]
	} else {
		col.Length = 255
	}
	return col
}

func (b *Blueprint) Text(name string) *Column {
	return b.addColumn(name, "text")
}

func (b *Blueprint) LongText(name string) *Column {
	return b.addColumn(name, "longText")
}

func (b *Blueprint) Integer(name string) *Column {
	return b.addColumn(name, "integer")
}

func (b *Blueprint) TinyInteger(name string) *Column {
	return b.addColumn(name, "tinyInteger")
}

func (b *Blueprint) SmallInteger(name string) *Column {
	return b.addColumn(name, "smallInteger")
}

func (b *Blueprint) BigInteger(name string) *Column {
	return b.addColumn(name, "bigInteger")
}

func (b *Blueprint) Boolean(name string) *Column {
	return b.addColumn(name, "boolean")
}

func (b *Blueprint) Decimal(name string, precision, scale int) *Column {
	col := b.addColumn(name, "decimal")
	col.Precision = precision
	col.Scale = scale
	return col
}

func (b *Blueprint) Float(name string) *Column {
	return b.addColumn(name, "float")
}

func (b *Blueprint) Double(name string) *Column {
	return b.addColumn(name, "double")
}

func (b *Blueprint) Date(name string) *Column {
	return b.addColumn(name, "date")
}

func (b *Blueprint) Time(name string) *Column {
	return b.addColumn(name, "time")
}

func (b *Blueprint) DateTime(name string) *Column {
	return b.addColumn(name, "datetime")
}

func (b *Blueprint) Timestamp(name string) *Column {
	return b.addColumn(name, "timestamp")
}

// Timestamps adds created_at and updated_at nullable timestamps
func (b *Blueprint) Timestamps() {
	b.addColumn("created_at", "timestamp").Nullable()
	b.addColumn("updated_at", "timestamp").Nullable()
}

// SoftDeletes adds deleted_at nullable timestamp
func (b *Blueprint) SoftDeletes() {
	b.addColumn("deleted_at", "timestamp").Nullable()
}

func (b *Blueprint) DropColumn(names ...string) {
	b.DropCols = append(b.DropCols, names...)
}

func (b *Blueprint) DropTimestamps() {
	b.DropColumn("created_at", "updated_at")
}

func (b *Blueprint) DropSoftDeletes() {
	b.DropColumn("deleted_at")
}

func (b *Blueprint) Index(columns ...string) {
	b.Indexes = append(b.Indexes, &Index{
		Columns:  columns,
		IsUnique: false,
	})
}

func (b *Blueprint) Unique(columns ...string) {
	b.Indexes = append(b.Indexes, &Index{
		Columns:  columns,
		IsUnique: true,
	})
}
