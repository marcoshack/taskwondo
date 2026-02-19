package dataport

import "time"

// FormatVersion is incremented when the archive layout changes.
const FormatVersion = 1

// Manifest is written as manifest.json in the export archive.
type Manifest struct {
	FormatVersion      int            `json:"format_version"`
	CreatedAt          time.Time      `json:"created_at"`
	MigrationVersion   uint           `json:"migration_version"`
	Tables             []TableSummary `json:"tables"`
	AttachmentCount    int            `json:"attachment_count"`
	SkippedAttachments []string       `json:"skipped_attachments,omitempty"`
}

// TableSummary records the row count exported for one table.
type TableSummary struct {
	Name     string `json:"name"`
	RowCount int    `json:"row_count"`
}

// TableDef describes a single table for export and import.
type TableDef struct {
	Name          string   // database table name
	Filename      string   // path inside the archive (e.g. "data/01_users.json")
	ExportQuery   string   // SELECT for export (all columns except GENERATED)
	ImportColumns []string // column names for pq.CopyIn
}
