package dataport

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/storage"
)

// Exporter exports all data from the database and storage into a tar.gz archive.
type Exporter struct {
	db    *sql.DB
	store storage.Storage
}

// NewExporter creates a new Exporter.
func NewExporter(db *sql.DB, store storage.Storage) *Exporter {
	return &Exporter{db: db, store: store}
}

// Export writes a complete data export to w as a tar.gz archive.
func (e *Exporter) Export(ctx context.Context, w io.Writer) error {
	logger := log.Ctx(ctx)
	start := time.Now()

	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	var manifest Manifest
	manifest.FormatVersion = FormatVersion
	manifest.CreatedAt = time.Now().UTC()

	// Get current migration version.
	version, err := getMigrationVersion(ctx, e.db)
	if err != nil {
		return fmt.Errorf("getting migration version: %w", err)
	}
	manifest.MigrationVersion = version

	// Export each table.
	for _, table := range ExportTables {
		count, err := e.exportTable(ctx, tw, table)
		if err != nil {
			return fmt.Errorf("exporting table %s: %w", table.Name, err)
		}
		manifest.Tables = append(manifest.Tables, TableSummary{
			Name:     table.Name,
			RowCount: count,
		})
		logger.Info().Str("table", table.Name).Int("rows", count).Msg("exported table")
	}

	// Export attachment files from storage.
	attachmentCount, skipped, err := e.exportAttachments(ctx, tw)
	if err != nil {
		return fmt.Errorf("exporting attachments: %w", err)
	}
	manifest.AttachmentCount = attachmentCount
	manifest.SkippedAttachments = skipped

	// Write manifest as the last entry.
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := writeTarEntry(tw, "manifest.json", manifestData); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	logger.Info().
		Int("tables", len(manifest.Tables)).
		Int("attachments", attachmentCount).
		Int("skipped_attachments", len(skipped)).
		Dur("elapsed", time.Since(start)).
		Msg("export complete")

	return nil
}

// exportTable runs the export query for a table and writes the result as a JSON array
// into the tar archive.
func (e *Exporter) exportTable(ctx context.Context, tw *tar.Writer, table TableDef) (int, error) {
	rows, err := e.db.QueryContext(ctx, table.ExportQuery)
	if err != nil {
		return 0, fmt.Errorf("querying: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("getting columns: %w", err)
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return 0, fmt.Errorf("getting column types: %w", err)
	}

	var result []map[string]interface{}

	for rows.Next() {
		scanDest := makeScanDest(colTypes)
		if err := rows.Scan(scanDest...); err != nil {
			return 0, fmt.Errorf("scanning row: %w", err)
		}

		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			row[col] = convertValue(scanDest[i])
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterating rows: %w", err)
	}

	// Ensure empty tables produce [] not null.
	if result == nil {
		result = []map[string]interface{}{}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return 0, fmt.Errorf("marshaling: %w", err)
	}

	if err := writeTarEntry(tw, table.Filename, data); err != nil {
		return 0, fmt.Errorf("writing tar entry: %w", err)
	}

	return len(result), nil
}

// exportAttachments downloads files from object storage and writes them into the archive.
func (e *Exporter) exportAttachments(ctx context.Context, tw *tar.Writer) (int, []string, error) {
	logger := log.Ctx(ctx)

	rows, err := e.db.QueryContext(ctx, `SELECT DISTINCT storage_key FROM attachments`)
	if err != nil {
		return 0, nil, fmt.Errorf("querying attachment keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return 0, nil, fmt.Errorf("scanning key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("iterating keys: %w", err)
	}

	var count int
	var skipped []string

	for _, key := range keys {
		rc, info, err := e.store.Get(ctx, key)
		if err != nil {
			logger.Warn().Err(err).Str("key", key).Msg("skipping attachment: download failed")
			skipped = append(skipped, key)
			continue
		}

		entryPath := "attachments/" + key
		hdr := &tar.Header{
			Name:    entryPath,
			Size:    info.Size,
			Mode:    0644,
			ModTime: time.Now(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			rc.Close()
			return count, skipped, fmt.Errorf("writing tar header for %s: %w", key, err)
		}
		if _, err := io.Copy(tw, rc); err != nil {
			rc.Close()
			return count, skipped, fmt.Errorf("writing attachment %s: %w", key, err)
		}
		rc.Close()
		count++
	}

	logger.Info().Int("count", count).Int("skipped", len(skipped)).Msg("exported attachments")
	return count, skipped, nil
}

// makeScanDest creates scan destination values based on column types.
func makeScanDest(colTypes []*sql.ColumnType) []interface{} {
	dest := make([]interface{}, len(colTypes))
	for i, ct := range colTypes {
		dbType := ct.DatabaseTypeName()
		switch dbType {
		case "BOOL":
			dest[i] = new(sql.NullBool)
		case "INT4", "INT8", "INT2":
			dest[i] = new(sql.NullInt64)
		case "FLOAT4", "FLOAT8", "NUMERIC":
			dest[i] = new(sql.NullFloat64)
		case "TIMESTAMPTZ", "TIMESTAMP":
			dest[i] = new(sql.NullTime)
		case "DATE":
			dest[i] = new(sql.NullTime)
		case "UUID":
			dest[i] = new(sql.NullString)
		case "JSONB", "JSON":
			dest[i] = new(sql.NullString)
		case "_TEXT", "_VARCHAR":
			dest[i] = &pq.StringArray{}
		default:
			dest[i] = new(sql.NullString)
		}
	}
	return dest
}

// convertValue converts a scanned SQL value into a JSON-friendly representation.
func convertValue(v interface{}) interface{} {
	switch val := v.(type) {
	case *sql.NullString:
		if !val.Valid {
			return nil
		}
		return val.String
	case *sql.NullBool:
		if !val.Valid {
			return nil
		}
		return val.Bool
	case *sql.NullInt64:
		if !val.Valid {
			return nil
		}
		return val.Int64
	case *sql.NullFloat64:
		if !val.Valid {
			return nil
		}
		return val.Float64
	case *sql.NullTime:
		if !val.Valid {
			return nil
		}
		return val.Time.UTC().Format(time.RFC3339Nano)
	case *pq.StringArray:
		if val == nil || len(*val) == 0 {
			return []string{}
		}
		return []string(*val)
	default:
		return nil
	}
}

// writeTarEntry adds a file entry to the tar archive.
func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

// getMigrationVersion queries the current migration version from schema_migrations.
func getMigrationVersion(ctx context.Context, db *sql.DB) (uint, error) {
	var version uint
	err := db.QueryRowContext(ctx, `SELECT version FROM schema_migrations LIMIT 1`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("querying schema_migrations: %w", err)
	}
	return version, nil
}
