package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	store := &Store{DB: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS seen_topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site TEXT NOT NULL,
			external_id TEXT NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			published_at TEXT,
			created_at TEXT NOT NULL,
			UNIQUE(site, external_id)
		);`,
		`CREATE TABLE IF NOT EXISTS user_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			site TEXT NOT NULL,
			category_id INTEGER NOT NULL DEFAULT 0,
			label TEXT NOT NULL,
			keyword_expr TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS push_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			topic_url TEXT NOT NULL,
			pushed_at TEXT NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

type UserSubscription struct {
	ID          int64
	UserID      int64
	Site        string
	CategoryID  int
	Label       string
	KeywordExpr string
	Enabled     bool
	CreatedAt   time.Time
}

func (s *Store) MarkSeen(site, externalID, title, url string, publishedAt time.Time) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO seen_topics(site, external_id, title, url, published_at, created_at) VALUES(?,?,?,?,?,?)`,
		site, externalID, title, url, publishedAt.Format(time.RFC3339), time.Now().Format(time.RFC3339))
	return err
}

func (s *Store) IsSeen(site, externalID string) (bool, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(1) FROM seen_topics WHERE site=? AND external_id=?`, site, externalID).Scan(&n)
	return n > 0, err
}

func (s *Store) AddSubscription(sub UserSubscription) error {
	_, err := s.DB.Exec(`INSERT INTO user_subscriptions(user_id, site, category_id, label, keyword_expr, enabled, created_at) VALUES(?,?,?,?,?,?,?)`,
		sub.UserID, sub.Site, sub.CategoryID, sub.Label, sub.KeywordExpr, 1, time.Now().Format(time.RFC3339))
	return err
}

func (s *Store) ListSubscriptions(userID int64) ([]UserSubscription, error) {
	query := `SELECT id, user_id, site, category_id, label, keyword_expr, enabled, created_at FROM user_subscriptions`
	args := []any{}
	if userID != 0 {
		query += ` WHERE user_id=?`
		args = append(args, userID)
	}
	query += ` ORDER BY id DESC`
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserSubscription
	for rows.Next() {
		var sub UserSubscription
		var enabled int
		var created string
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Site, &sub.CategoryID, &sub.Label, &sub.KeywordExpr, &enabled, &created); err != nil {
			return nil, err
		}
		sub.Enabled = enabled == 1
		sub.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, sub)
	}
	return out, nil
}

func (s *Store) DeleteSubscription(userID int64, subID int64) error {
	res, err := s.DB.Exec(`DELETE FROM user_subscriptions WHERE id=? AND user_id=?`, subID, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

func (s *Store) CountUserPushesToday(userID int64) (int, error) {
	dayPrefix := time.Now().Format("2006-01-02")
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(1) FROM push_logs WHERE user_id=? AND pushed_at LIKE ?`, userID, dayPrefix+"%").Scan(&n)
	return n, err
}

func (s *Store) AddPushLog(userID int64, topicURL string) error {
	_, err := s.DB.Exec(`INSERT INTO push_logs(user_id, topic_url, pushed_at) VALUES(?,?,?)`, userID, topicURL, time.Now().Format(time.RFC3339))
	return err
}

func NormalizeCSVInt64(v string) []int64 {
	parts := strings.Split(v, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		_ = p
	}
	return out
}
