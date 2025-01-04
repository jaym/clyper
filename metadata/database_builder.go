package metadata

import (
	"database/sql"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"

	"github.com/jaym/clyper/objstore"
)

type preparedStatementKey string

const (
	insertEpisodeStmt   preparedStatementKey = "insertEpisodeStmt"
	insertThumbnailStmt preparedStatementKey = "insertThumbnailStmt"
	insertVideoStmt     preparedStatementKey = "insertVideoStmt"
	insertSubtitlStmt   preparedStatementKey = "insertSubtitleStmt"
)

type DatabaseBuilder struct {
	objReader          objstore.ObjectReader
	db                 *sql.DB
	preparedStatements map[preparedStatementKey]*sql.Stmt
	outputDatabasePath string
	tmpDatabasePath    string
}

func NewDatabaseBuilder(dbPath string, objReader objstore.ObjectReader) (*DatabaseBuilder, error) {
	dbPathDirName := path.Dir(dbPath)
	tmpDbPath := path.Join(dbPathDirName, "tmp.db")
	_, err := os.Stat(tmpDbPath)
	if err == nil {
		// Remove the temporary database
		err = os.Remove(tmpDbPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to remove temporary database")
			return nil, err
		}
	}
	db, err := sql.Open("sqlite3", tmpDbPath)
	if err != nil {
		return nil, err
	}

	// Execute the schema
	schemaBytes, err := SchemaFS.ReadFile("schema.sql")
	if err != nil {
		db.Close() // nolint: errcheck
		log.Error().Err(err).Msg("Failed to read schema.sql")
		return nil, err
	}

	// Execute the schema
	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		db.Close() // nolint: errcheck
		log.Error().Err(err).Msg("Failed to execute schema.sql")
		return nil, err
	}

	preparedStatements := make(map[preparedStatementKey]*sql.Stmt)
	for key, stmt := range map[preparedStatementKey]string{
		insertEpisodeStmt:   `INSERT INTO episodes (season, episode) VALUES (?, ?)`,
		insertThumbnailStmt: `INSERT INTO thumbnails (episode_id, storage_key, start_ts, end_ts) VALUES (?, ?, ?, ?)`,
		insertVideoStmt:     `INSERT INTO videos (episode_id, storage_key) VALUES (?, ?)`,
		insertSubtitlStmt:   `INSERT INTO subtitles (episode_id, text, start_ts, end_ts) VALUES (?, ?,?, ?)`,
	} {
		preparedStmt, err := db.Prepare(stmt)
		if err != nil {
			db.Close() // nolint: errcheck
			log.Error().Err(err).Msg("Failed to prepare statement")
			return nil, err
		}

		preparedStatements[key] = preparedStmt
	}

	return &DatabaseBuilder{
		objReader:          objReader,
		db:                 db,
		preparedStatements: preparedStatements,
		outputDatabasePath: dbPath,
		tmpDatabasePath:    tmpDbPath,
	}, nil
}

func (b *DatabaseBuilder) Build() error {
	// Compact the database
	_, err := b.db.Exec("VACUUM")
	if err != nil {
		log.Error().Err(err).Msg("Failed to compact the database")
		return err
	}

	// Close the database
	err = b.db.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close the database")
		return err
	}

	// Move the temporary database to the output path
	err = os.Rename(b.tmpDatabasePath, b.outputDatabasePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to move the temporary database")
		return err
	}

	return nil
}

func (b *DatabaseBuilder) AddEpisodeMetadata(metadata EpisodeMetadata) error {
	// Insert the episode
	res, err := b.preparedStatements[insertEpisodeStmt].Exec(metadata.Season, metadata.Episode)
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert episode")
		return err
	}

	episodeID, err := res.LastInsertId()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get episode ID")
		return err
	}

	// Insert the thumbnails
	for _, thumb := range metadata.Thumbs {
		_, err = b.preparedStatements[insertThumbnailStmt].Exec(episodeID, thumb.Key, thumb.Start, thumb.End)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert thumbnail")
			return err
		}
	}

	// Insert the video
	_, err = b.preparedStatements[insertVideoStmt].Exec(episodeID, metadata.VideoFileKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert video")
		return err
	}

	for _, s := range metadata.Subtitles {
		_, err = b.preparedStatements[insertSubtitlStmt].Exec(episodeID, s.Text, s.Start, s.End)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert subtitle")
			return err
		}
	}

	return nil
}
