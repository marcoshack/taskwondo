package dataport

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/storage"
)

// Importer imports data from a tar.gz archive into an empty database and storage.
type Importer struct {
	db    *sql.DB
	store storage.Storage
}

// NewImporter creates a new Importer.
func NewImporter(db *sql.DB, store storage.Storage) *Importer {
	return &Importer{db: db, store: store}
}

// Import reads a tar.gz archive and restores all data into the database and storage.
// The target database must be empty (migrations applied, but no user data).
func (imp *Importer) Import(ctx context.Context, r io.Reader) error {
	logger := log.Ctx(ctx)
	start := time.Now()

	// Extract archive to temp directory.
	tmpDir, err := os.MkdirTemp("", "taskwondo-import-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	logger.Info().Str("tmp_dir", tmpDir).Msg("extracting archive")
	if err := extractTarGz(r, tmpDir); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	// Read and validate manifest.
	manifest, err := readManifest(filepath.Join(tmpDir, "manifest.json"))
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	if manifest.FormatVersion != FormatVersion {
		return fmt.Errorf("unsupported format version %d (expected %d)", manifest.FormatVersion, FormatVersion)
	}

	currentVersion, err := getMigrationVersion(ctx, imp.db)
	if err != nil {
		return fmt.Errorf("getting current migration version: %w", err)
	}
	if manifest.MigrationVersion != currentVersion {
		return fmt.Errorf("migration version mismatch: archive=%d, database=%d", manifest.MigrationVersion, currentVersion)
	}

	logger.Info().
		Int("format_version", manifest.FormatVersion).
		Uint("migration_version", manifest.MigrationVersion).
		Int("tables", len(manifest.Tables)).
		Int("attachments", manifest.AttachmentCount).
		Msg("manifest validated")

	// Safety check: database must be empty.
	if err := checkDatabaseEmpty(ctx, imp.db); err != nil {
		return err
	}

	// Build row count lookup from manifest.
	expectedCounts := make(map[string]int, len(manifest.Tables))
	for _, ts := range manifest.Tables {
		expectedCounts[ts.Name] = ts.RowCount
	}

	// Import each table in dependency order.
	for _, table := range ExportTables {
		dataPath := filepath.Join(tmpDir, table.Filename)
		count, err := imp.importTable(ctx, table, dataPath)
		if err != nil {
			return fmt.Errorf("importing table %s: %w", table.Name, err)
		}

		expected, ok := expectedCounts[table.Name]
		if ok && count != expected {
			return fmt.Errorf("table %s: imported %d rows but manifest expected %d", table.Name, count, expected)
		}
		logger.Info().Str("table", table.Name).Int("rows", count).Msg("imported table")
	}

	// Upload attachment files to storage.
	attachDir := filepath.Join(tmpDir, "attachments")
	attachCount, err := imp.importAttachments(ctx, tmpDir, attachDir)
	if err != nil {
		return fmt.Errorf("importing attachments: %w", err)
	}

	logger.Info().
		Int("attachments", attachCount).
		Dur("elapsed", time.Since(start)).
		Msg("import complete")

	return nil
}

// importTable reads a JSON file and bulk-inserts all rows via pq.CopyIn.
func (imp *Importer) importTable(ctx context.Context, table TableDef, dataPath string) (int, error) {
	data, err := os.ReadFile(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // Table file missing means 0 rows.
		}
		return 0, fmt.Errorf("reading file: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return 0, fmt.Errorf("unmarshaling JSON: %w", err)
	}

	if len(rows) == 0 {
		return 0, nil
	}

	tx, err := imp.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Disable triggers (including FK checks) during COPY for tables with
	// self-referential FKs. This mirrors pg_restore behavior.
	if table.Name == "work_items" {
		if _, err := tx.ExecContext(ctx, "ALTER TABLE work_items DISABLE TRIGGER ALL"); err != nil {
			return 0, fmt.Errorf("disabling triggers: %w", err)
		}
	}

	stmt, err := tx.Prepare(pq.CopyIn(table.Name, table.ImportColumns...))
	if err != nil {
		return 0, fmt.Errorf("preparing COPY: %w", err)
	}

	for i, row := range rows {
		values := make([]interface{}, len(table.ImportColumns))
		for j, col := range table.ImportColumns {
			values[j], err = convertImportValue(row[col], col)
			if err != nil {
				return 0, fmt.Errorf("row %d, column %s: %w", i, col, err)
			}
		}
		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			return 0, fmt.Errorf("row %d: %w", i, err)
		}
	}

	// Flush COPY.
	if _, err := stmt.ExecContext(ctx); err != nil {
		return 0, fmt.Errorf("flushing COPY: %w", err)
	}
	if err := stmt.Close(); err != nil {
		return 0, fmt.Errorf("closing statement: %w", err)
	}

	// Re-enable triggers after COPY.
	if table.Name == "work_items" {
		if _, err := tx.ExecContext(ctx, "ALTER TABLE work_items ENABLE TRIGGER ALL"); err != nil {
			return 0, fmt.Errorf("re-enabling triggers: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing: %w", err)
	}

	return len(rows), nil
}

// importAttachments reads attachment metadata from the imported data and uploads
// files from the temp directory to object storage.
func (imp *Importer) importAttachments(ctx context.Context, tmpDir, attachDir string) (int, error) {
	logger := log.Ctx(ctx)

	// Read attachment metadata to get content types.
	contentTypes := make(map[string]string)
	for _, table := range ExportTables {
		if table.Name == "attachments" {
			data, err := os.ReadFile(filepath.Join(tmpDir, table.Filename))
			if err != nil {
				break
			}
			var attachments []map[string]interface{}
			if err := json.Unmarshal(data, &attachments); err != nil {
				break
			}
			for _, a := range attachments {
				if key, ok := a["storage_key"].(string); ok {
					ct, _ := a["content_type"].(string)
					if ct == "" {
						ct = "application/octet-stream"
					}
					contentTypes[key] = ct
				}
			}
			break
		}
	}

	if _, err := os.Stat(attachDir); os.IsNotExist(err) {
		return 0, nil
	}

	entries, err := os.ReadDir(attachDir)
	if err != nil {
		return 0, fmt.Errorf("reading attachments dir: %w", err)
	}

	var count int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		key := entry.Name()
		filePath := filepath.Join(attachDir, key)

		info, err := entry.Info()
		if err != nil {
			logger.Warn().Err(err).Str("key", key).Msg("skipping attachment: stat failed")
			continue
		}

		f, err := os.Open(filePath)
		if err != nil {
			logger.Warn().Err(err).Str("key", key).Msg("skipping attachment: open failed")
			continue
		}

		ct := contentTypes[key]
		if ct == "" {
			ct = "application/octet-stream"
		}

		if _, err := imp.store.Put(ctx, key, f, info.Size(), ct); err != nil {
			f.Close()
			logger.Warn().Err(err).Str("key", key).Msg("failed to upload attachment")
			continue
		}
		f.Close()
		count++
	}

	logger.Info().Int("count", count).Msg("uploaded attachments")
	return count, nil
}

// convertImportValue converts a JSON-decoded value into a type suitable for pq.CopyIn.
func convertImportValue(v interface{}, colName string) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	switch val := v.(type) {
	case string:
		// Detect timestamp strings and parse them.
		if isTimestampColumn(colName) {
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				// Try date-only for DATE columns.
				t, err = time.Parse("2006-01-02", val)
				if err != nil {
					return val, nil // Pass as string, let PG handle it.
				}
			}
			return t, nil
		}
		return val, nil
	case float64:
		// JSON numbers are float64; convert to int64 for integer columns.
		if isIntColumn(colName) {
			return int64(val), nil
		}
		return val, nil
	case bool:
		return val, nil
	case []interface{}:
		// Convert JSON array to pq.StringArray.
		sa := make(pq.StringArray, len(val))
		for i, elem := range val {
			s, ok := elem.(string)
			if !ok {
				return nil, fmt.Errorf("expected string array element, got %T", elem)
			}
			sa[i] = s
		}
		return sa, nil
	case map[string]interface{}:
		// JSONB columns: re-encode to string.
		b, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("re-encoding JSONB: %w", err)
		}
		return string(b), nil
	default:
		return v, nil
	}
}

// isTimestampColumn returns true if the column name suggests a timestamp or date type.
func isTimestampColumn(col string) bool {
	return strings.HasSuffix(col, "_at") || col == "due_date"
}

// isIntColumn returns true if the column name suggests an integer type.
func isIntColumn(col string) bool {
	switch col {
	case "item_number", "item_counter", "position", "edit_count", "size_bytes":
		return true
	}
	return false
}

// checkDatabaseEmpty verifies that the target database has no user data.
func checkDatabaseEmpty(ctx context.Context, db *sql.DB) error {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return fmt.Errorf("checking users table: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("database is not empty (%d users found); import only works on a fresh installation", count)
	}
	return nil
}

// extractTarGz extracts a tar.gz archive to the given directory.
func extractTarGz(r io.Reader, destDir string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		// Sanitize path to prevent directory traversal.
		cleanName := filepath.Clean(hdr.Name)
		if strings.Contains(cleanName, "..") {
			return fmt.Errorf("invalid tar entry path: %s", hdr.Name)
		}

		target := filepath.Join(destDir, cleanName)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", target, err)
			}

			f, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			// Limit copy size to prevent decompression bombs (10 GB per file).
			if _, err := io.Copy(f, io.LimitReader(tr, 10<<30)); err != nil {
				f.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			f.Close()
		}
	}
	return nil
}

// readManifest reads and parses the manifest.json file.
func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest file: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}