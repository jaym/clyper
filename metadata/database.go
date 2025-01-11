package metadata

import (
	"context"
	"database/sql"
)

type Database struct {
	db                 *sql.DB
	preparedStatements map[preparedStatementKey]*sql.Stmt
}

const (
	searchStmt             preparedStatementKey = "searchStmt"
	listThumbsForwardStmt  preparedStatementKey = "listThumbsForwardsStmt"
	listThumbsBackwardStmt preparedStatementKey = "listThumbsBackwardsStmt"
	videoFileStmt          preparedStatementKey = "videoFileStmt"
)

func OpenDatabase(dbPath string) (*Database, error) {
	// Open the database as read-only
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	preparedStatements := make(map[preparedStatementKey]*sql.Stmt)
	for key, query := range map[preparedStatementKey]string{
		searchStmt:             `SELECT episodes.season, episodes.episode, subtitles.start_ts, subtitles.end_ts, subtitles.text FROM subtitles INNER JOIN subtitles_fts ON subtitles.rowid = subtitles_fts.rowid INNER JOIN episodes ON subtitles.episode_id=episodes.id WHERE subtitles_fts MATCH ? limit 100`,
		listThumbsForwardStmt:  `SELECT storage_key, start_ts, end_ts FROM thumbnails WHERE episode_id = (SELECT id from episodes where season = ? AND episode = ?) AND start_ts >= ? ORDER BY start_ts ASC LIMIT ?`,
		listThumbsBackwardStmt: `SELECT storage_key, start_ts, end_ts FROM thumbnails WHERE episode_id = (SELECT id from episodes where season = ? AND episode = ?) AND start_ts <= ? ORDER BY start_ts DESC LIMIT ?`,
		videoFileStmt:          `SELECT storage_key FROM videos WHERE episode_id = (SELECT id from episodes where season = ? AND episode = ?)`,
	} {
		stmt, err := db.Prepare(query)
		if err != nil {
			db.Close() // nolint: errcheck
			return nil, err
		}

		preparedStatements[key] = stmt
	}

	return &Database{
		db:                 db,
		preparedStatements: preparedStatements,
	}, nil
}

type SearchResult struct {
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Text    string `json:"text"`
}

func (d *Database) Search(ctx context.Context, queryString string) ([]SearchResult, error) {
	rows, err := d.preparedStatements[searchStmt].QueryContext(ctx, queryString)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		err := rows.Scan(&result.Season, &result.Episode, &result.Start, &result.End, &result.Text)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func (d *Database) ListThumbnails(ctx context.Context, season int, episode int, timestamp int, count int, reverse bool) ([]ThumbMetadata, error) {
	var stmtKey preparedStatementKey
	if reverse {
		stmtKey = listThumbsBackwardStmt
	} else {
		stmtKey = listThumbsForwardStmt
	}

	rows, err := d.preparedStatements[stmtKey].QueryContext(ctx, season, episode, timestamp, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ThumbMetadata
	for rows.Next() {
		result := ThumbMetadata{}
		err := rows.Scan(&result.Key, &result.Start, &result.End)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (d *Database) GetVideoFileKey(ctx context.Context, season int, episode int) (string, error) {
	var key string
	err := d.preparedStatements[videoFileStmt].QueryRowContext(ctx, season, episode).Scan(&key)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
