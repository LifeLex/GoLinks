package persistence

import "time"

// shortcutRow is the GORM-tagged DB model for the linktable table. We keep the
// model unexported and convert to the public links.Shortcut entity through the
// mapper so GORM tags never leak into the domain.
type shortcutRow struct {
	ID        uint      `gorm:"primaryKey"`
	Word      string    `gorm:"index:idx_linktable_word;not null"`
	Link      string    `gorm:"not null"`
	User      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName overrides GORM's default pluralisation. The table is "linktable"
// for historical reasons; renaming it is a separate migration concern.
func (shortcutRow) TableName() string { return "linktable" }

// queryRow is the GORM-tagged DB model for the queries table. The Shortcut
// pointer field declares the foreign key relationship so AutoMigrate creates
// the FK constraint; it is never populated by the application directly.
type queryRow struct {
	QueryID   uint         `gorm:"column:query_id;primaryKey"`
	WordID    uint         `gorm:"column:word_id;not null;index:idx_queries_word_id"`
	CreatedAt time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_queries_created_at"`
	Shortcut  *shortcutRow `gorm:"foreignKey:WordID;references:ID;constraint:OnDelete:RESTRICT,OnUpdate:RESTRICT"`
}

// TableName overrides GORM's default pluralisation.
func (queryRow) TableName() string { return "queries" }

// tagRow is the GORM-tagged DB model for the tags table. The table exists in
// the schema but is currently unused by the application.
type tagRow struct {
	ID       uint         `gorm:"primaryKey"`
	WordID   uint         `gorm:"column:word_id;not null"`
	Tag      string       `gorm:"not null"`
	Shortcut *shortcutRow `gorm:"foreignKey:WordID;references:ID;constraint:OnDelete:RESTRICT,OnUpdate:RESTRICT"`
}

// TableName overrides GORM's default pluralisation.
func (tagRow) TableName() string { return "tags" }

// Models returns every GORM model the application uses, in dependency order
// (parent tables first). Migrations consume this slice when running
// AutoMigrate so adding a model means touching this file plus a migration —
// nothing else.
func Models() []interface{} {
	return []interface{}{
		&shortcutRow{},
		&queryRow{},
		&tagRow{},
	}
}

// ModelsReverse returns Models() in reverse order — useful for DropTable
// operations, which need child tables dropped before their parents to avoid
// foreign-key violations.
func ModelsReverse() []interface{} {
	models := Models()
	reversed := make([]interface{}, len(models))
	for i, m := range models {
		reversed[len(models)-1-i] = m
	}
	return reversed
}
