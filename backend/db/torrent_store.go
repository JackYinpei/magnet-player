package db

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// TorrentRecord represents a stored torrent in the database
type TorrentRecord struct {
	InfoHash   string    `json:"infoHash"`
	Name       string    `json:"name"`
	MagnetURI  string    `json:"magnetUri"`
	AddedAt    time.Time `json:"addedAt"`
	MovieName  string    `json:"movieName,omitempty"`  // Official movie name from search
	Year       int       `json:"year,omitempty"`       // Release year
	PosterURL  string    `json:"posterUrl,omitempty"`  // URL to movie poster image
}

// TorrentStore handles the storage and retrieval of torrent information
type TorrentStore struct {
	db    *sql.DB
	mutex sync.Mutex
}

// NewTorrentStore creates a new TorrentStore
func NewTorrentStore(dbPath string) (*TorrentStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create the torrents table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS torrents (
			info_hash TEXT PRIMARY KEY,
			name TEXT,
			magnet_uri TEXT NOT NULL,
			added_at TIMESTAMP,
			movie_name TEXT,
			year INTEGER,
			poster_url TEXT
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Check if new columns exist and add them if they don't
	// This handles the migration for existing databases
	var hasMovieName, hasYear, hasPosterURL bool
	
	// Check if movie_name column exists
	rows, err := db.Query("PRAGMA table_info(torrents)")
	if err != nil {
		db.Close()
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var cid int
		var name, type_name string
		var notnull, pk int
		var dflt_value interface{}
		if err := rows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err != nil {
			db.Close()
			return nil, err
		}
		
		if name == "movie_name" {
			hasMovieName = true
		} else if name == "year" {
			hasYear = true
		} else if name == "poster_url" {
			hasPosterURL = true
		}
	}
	
	// Add missing columns if needed
	if !hasMovieName {
		_, err = db.Exec("ALTER TABLE torrents ADD COLUMN movie_name TEXT")
		if err != nil {
			log.Printf("Error adding movie_name column: %v", err)
		}
	}
	
	if !hasYear {
		_, err = db.Exec("ALTER TABLE torrents ADD COLUMN year INTEGER")
		if err != nil {
			log.Printf("Error adding year column: %v", err)
		}
	}
	
	if !hasPosterURL {
		_, err = db.Exec("ALTER TABLE torrents ADD COLUMN poster_url TEXT")
		if err != nil {
			log.Printf("Error adding poster_url column: %v", err)
		}
	}

	return &TorrentStore{
		db: db,
	}, nil
}

// Close closes the database connection
func (s *TorrentStore) Close() error {
	return s.db.Close()
}

// SaveTorrent saves a torrent to the database
func (s *TorrentStore) SaveTorrent(record TorrentRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO torrents (info_hash, name, magnet_uri, added_at, movie_name, year, poster_url)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(info_hash) DO UPDATE SET
			name = ?,
			magnet_uri = ?,
			added_at = ?,
			movie_name = ?,
			year = ?,
			poster_url = ?
	`, record.InfoHash, record.Name, record.MagnetURI, record.AddedAt, 
	   record.MovieName, record.Year, record.PosterURL,
	   record.Name, record.MagnetURI, record.AddedAt,
	   record.MovieName, record.Year, record.PosterURL)
	
	return err
}

// GetAllTorrents retrieves all torrents from the database
func (s *TorrentStore) GetAllTorrents() ([]TorrentRecord, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	rows, err := s.db.Query(`
		SELECT 
			info_hash, 
			name, 
			magnet_uri, 
			added_at,
			movie_name, 
			year, 
			poster_url
		FROM torrents
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []TorrentRecord
	for rows.Next() {
		var record TorrentRecord
		var addedAt sql.NullTime
		var movieName sql.NullString
		var year sql.NullInt64
		var posterURL sql.NullString

		err := rows.Scan(
			&record.InfoHash, 
			&record.Name, 
			&record.MagnetURI, 
			&addedAt,
			&movieName,
			&year,
			&posterURL,
		)
		if err != nil {
			return nil, err
		}

		// Handle null values
		if addedAt.Valid {
			record.AddedAt = addedAt.Time
		} else {
			record.AddedAt = time.Now() // Default to current time if null
		}

		if movieName.Valid {
			record.MovieName = movieName.String
		}

		if year.Valid {
			record.Year = int(year.Int64)
		}

		if posterURL.Valid {
			record.PosterURL = posterURL.String
		}

		torrents = append(torrents, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return torrents, nil
}

// DeleteTorrent removes a torrent from the database
func (s *TorrentStore) DeleteTorrent(infoHash string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, err := s.db.Exec(`DELETE FROM torrents WHERE info_hash = ?`, infoHash)
	return err
}

// UpdateMovieInfo updates the movie information for a torrent
func (s *TorrentStore) UpdateMovieInfo(infoHash string, movieName string, year int, posterURL string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, err := s.db.Exec(
		`UPDATE torrents SET movie_name = ?, year = ?, poster_url = ? WHERE info_hash = ?`,
		movieName, year, posterURL, infoHash,
	)
	return err
}

// GetTorrent retrieves a single torrent by its info hash
func (s *TorrentStore) GetTorrent(infoHash string) (TorrentRecord, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var record TorrentRecord
	var addedAt sql.NullTime
	var movieName sql.NullString
	var year sql.NullInt64
	var posterURL sql.NullString

	err := s.db.QueryRow(`
		SELECT 
			info_hash, 
			name, 
			magnet_uri, 
			added_at,
			movie_name, 
			year, 
			poster_url
		FROM torrents
		WHERE info_hash = ?
	`, infoHash).Scan(
		&record.InfoHash,
		&record.Name,
		&record.MagnetURI,
		&addedAt,
		&movieName,
		&year,
		&posterURL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return TorrentRecord{}, fmt.Errorf("torrent not found: %s", infoHash)
		}
		return TorrentRecord{}, err
	}

	// Handle null values
	if addedAt.Valid {
		record.AddedAt = addedAt.Time
	} else {
		record.AddedAt = time.Now() // Default to current time if null
	}

	if movieName.Valid {
		record.MovieName = movieName.String
	}

	if year.Valid {
		record.Year = int(year.Int64)
	}

	if posterURL.Valid {
		record.PosterURL = posterURL.String
	}

	return record, nil
}
