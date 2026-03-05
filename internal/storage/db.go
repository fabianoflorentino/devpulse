// Package storage persists health snapshots in a local SQLite database.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fabianoflorentino/devpulse/internal/metrics"
	_ "modernc.org/sqlite" // pure-Go SQLite driver; no cgo required
)

// DB wraps a SQLite database.
type DB struct {
	db *sql.DB
}

// Open opens (or creates) the DevPulse SQLite database.
// If path is empty, the file is placed in $HOME/.devpulse/devpulse.db.
func Open(path string) (*DB, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(home, ".devpulse")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
		path = filepath.Join(dir, "devpulse.db")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error { return d.db.Close() }

const schema = `
CREATE TABLE IF NOT EXISTS health_snapshots (
    id               INTEGER  PRIMARY KEY AUTOINCREMENT,
    repo             TEXT     NOT NULL,
    scanned_at       DATETIME NOT NULL,
    open_prs         INTEGER  NOT NULL DEFAULT 0,
    prs_no_reviewer  INTEGER  NOT NULL DEFAULT 0,
    avg_review_time  TEXT     NOT NULL DEFAULT '',
    stale_issues     INTEGER  NOT NULL DEFAULT 0,
    security_alerts  INTEGER  NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_health_repo_time ON health_snapshots (repo, scanned_at DESC);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

// SaveHealth persists a Health snapshot.
func (d *DB) SaveHealth(ctx context.Context, h *metrics.Health) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO health_snapshots
		    (repo, scanned_at, open_prs, prs_no_reviewer, avg_review_time, stale_issues, security_alerts)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		h.Repo, h.ScannedAt.Format(time.RFC3339),
		h.OpenPRs, h.PRsWithoutReviewer, h.AvgReviewTime,
		h.StaleIssues, h.SecurityAlerts,
	)
	return err
}

// LatestHealth returns the most recent Health snapshot for the given repo.
func (d *DB) LatestHealth(ctx context.Context, repo string) (*metrics.Health, error) {
	row := d.db.QueryRowContext(ctx, `
		SELECT repo, scanned_at, open_prs, prs_no_reviewer, avg_review_time, stale_issues, security_alerts
		FROM health_snapshots
		WHERE repo = ?
		ORDER BY scanned_at DESC
		LIMIT 1`, repo)

	h := &metrics.Health{}
	var scannedAt string
	if err := row.Scan(
		&h.Repo, &scannedAt,
		&h.OpenPRs, &h.PRsWithoutReviewer, &h.AvgReviewTime,
		&h.StaleIssues, &h.SecurityAlerts,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no snapshots found for repo %q", repo)
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339, scannedAt)
	if err == nil {
		h.ScannedAt = t
	}
	return h, nil
}

// ListRepos returns all distinct repository names that have been scanned.
func (d *DB) ListRepos(ctx context.Context) ([]string, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT DISTINCT repo FROM health_snapshots ORDER BY repo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}
