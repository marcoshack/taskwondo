package repository

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// cjkRe matches Han, Hiragana, Katakana and Hangul characters. PostgreSQL's
// built-in text-search configs tokenize on whitespace, so a CJK run ends up as
// one lexeme and phrase queries never match unless typed exactly.
var cjkRe = regexp.MustCompile(`[\p{Han}\p{Hiragana}\p{Katakana}\p{Hangul}]`)

// containsCJK reports whether s contains at least one CJK character, in which
// case search queries add an ILIKE substring fallback on top of FTS.
func containsCJK(s string) bool {
	return cjkRe.MatchString(s)
}

// likePattern wraps s with `%` wildcards and escapes the LIKE special
// characters (`\`, `%`, `_`) so user input is matched literally.
func likePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}

// searchFilterCondition builds the WHERE fragment matching `query` against the
// tsvector column at vectorCol. When the query contains CJK characters it also
// ORs an ILIKE substring match over rawCols, because Postgres' whitespace
// tokenizer cannot split CJK runs into lexemes and a pure tsquery would match
// nothing. Placeholders are `?` for use with queryBuilder.add.
func searchFilterCondition(vectorCol string, rawCols []string, query string) (string, []interface{}) {
	cond := fmt.Sprintf("(%s @@ plainto_tsquery('english', ?) OR %s @@ plainto_tsquery('simple', ?)", vectorCol, vectorCol)
	args := []interface{}{query, query}
	if containsCJK(query) {
		pattern := likePattern(query)
		for _, col := range rawCols {
			cond += fmt.Sprintf(" OR %s ILIKE ?", col)
			args = append(args, pattern)
		}
	}
	return cond + ")", args
}

// queryBuilder accumulates WHERE-clause conditions and their bound parameters,
// translating each `?` placeholder in a condition into the next `$N` for
// lib/pq. Use add() for conditions with parameters and addRaw() for literal
// fragments.
type queryBuilder struct {
	conditions []string
	args       []interface{}
	argIndex   int
}

// add appends a condition containing one or more `?` placeholders. Each `?` is
// replaced with the next `$N` and the matching arg is bound.
func (qb *queryBuilder) add(condition string, args ...interface{}) {
	for _, arg := range args {
		qb.argIndex++
		condition = strings.Replace(condition, "?", fmt.Sprintf("$%d", qb.argIndex), 1)
		qb.args = append(qb.args, arg)
	}
	qb.conditions = append(qb.conditions, condition)
}

// addRaw appends a condition with no bound parameters.
func (qb *queryBuilder) addRaw(condition string) {
	qb.conditions = append(qb.conditions, condition)
}

// whereClause returns a WHERE clause assembled from the accumulated conditions
// joined by AND, or an empty string when no conditions have been added.
func (qb *queryBuilder) whereClause() string {
	if len(qb.conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(qb.conditions, " AND ")
}

// listAllIDs returns all non-deleted entity IDs from the given table with pagination.
// Used by backfill operations to iterate through all entities.
func listAllIDs(ctx context.Context, db *sql.DB, table string, limit, offset int) ([]uuid.UUID, error) {
	query := fmt.Sprintf(
		`SELECT id FROM %s WHERE deleted_at IS NULL ORDER BY id LIMIT $1 OFFSET $2`, table)
	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing %s IDs: %w", table, err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning %s ID: %w", table, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// listAllIDsNoSoftDelete returns all entity IDs from the given table with pagination.
// Used for tables that don't have a deleted_at column.
func listAllIDsNoSoftDelete(ctx context.Context, db *sql.DB, table string, limit, offset int) ([]uuid.UUID, error) {
	query := fmt.Sprintf(
		`SELECT id FROM %s ORDER BY id LIMIT $1 OFFSET $2`, table)
	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing %s IDs: %w", table, err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning %s ID: %w", table, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
