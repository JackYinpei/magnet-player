package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// TorrentRecord represents a stored torrent in the database
type TorrentRecord struct {
	InfoHash     string        `json:"infoHash"`
	Name         string        `json:"name"`
	Length       int64         `json:"length"`
	Files        []FileInfo    `json:"files,omitempty"`
	Downloaded   int64         `json:"downloaded"`
	Progress     float32       `json:"progress"`
	State        string        `json:"state"`
	MagnetURI    string        `json:"magnetUri"`
	AddedAt      time.Time     `json:"addedAt"`
	DataPath     string        `json:"dataPath,omitempty"`
	MovieDetails *MovieDetails `json:"movieDetails,omitempty"`
}

// MovieDetails represents the movie information
type MovieDetails struct {
	Filename      string   `json:"filename,omitempty"`
	Year          int      `json:"year,omitempty"`
	PosterUrl     string   `json:"posterUrl,omitempty"`
	BackdropUrl   string   `json:"backdropUrl,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	Rating        float64  `json:"rating,omitempty"`
	VoteCount     int      `json:"voteCount,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Runtime       int      `json:"runtime,omitempty"`
	TmdbId        int      `json:"tmdbId,omitempty"`
	ReleaseDate   string   `json:"releaseDate,omitempty"`
	OriginalTitle string   `json:"originalTitle,omitempty"`
	Popularity    float64  `json:"popularity,omitempty"`
	Status        string   `json:"status,omitempty"`
	Tagline       string   `json:"tagline,omitempty"`
	// Adult            bool    `json:"adult"`
	// BackdropPath     string  `json:"backdrop_path,omitempty"`
	// GenreIds         []int   `json:"genre_ids,omitempty"`
	// Id               int     `json:"id,omitempty"`
	// OriginalLanguage string  `json:"original_language,omitempty"`
	// OriginalTitle    string  `json:"original_title,omitempty"`
	// Overview         string  `json:"overview,omitempty"`
	// Popularity       float64 `json:"popularity,omitempty"`
	// PosterPath       string  `json:"poster_path,omitempty"`
	// ReleaseDate      string  `json:"releaseDate,omitempty"`
	// Title            string  `json:"title,omitempty"`
	// Video            bool    `json:"video,omitempty"`
	// VoteAverage      float64 `json:"vote_average,omitempty"`
	// VoteCount        int     `json:"vote_count,omitempty"`
}

// FileInfo represents information about a file in a torrent
type FileInfo struct {
	Path       string  `json:"path"`
	Length     int64   `json:"length"`
	Progress   float32 `json:"progress"`
	FileIndex  int     `json:"fileIndex"`
	TorrentID  string  `json:"torrentId"`
	IsVideo    bool    `json:"isVideo"`
	IsPlayable bool    `json:"isPlayable"`
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
			data_path TEXT,
			length INTEGER,
			files TEXT,
			downloaded INTEGER,
			progress REAL,
			state TEXT,
			movie_details TEXT
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &TorrentStore{
		db: db,
	}, nil
}

// Close closes the database connection
func (s *TorrentStore) Close() error {
	return s.db.Close()
}

// AddTorrent adds a new torrent record to the database
func (s *TorrentStore) AddTorrent(record *TorrentRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Convert Files slice to JSON
	filesJSON, err := json.Marshal(record.Files)
	if err != nil {
		return err
	}

	// Convert MovieDetails to JSON if it exists
	var movieDetailsJSON []byte
	if record.MovieDetails != nil {
		movieDetailsJSON, err = json.Marshal(record.MovieDetails)
		if err != nil {
			return err
		}
	}

	// Insert the torrent record
	_, err = s.db.Exec(`
		INSERT INTO torrents (
			info_hash, name, magnet_uri, added_at, data_path, 
			length, files, downloaded, progress, state, movie_details
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.InfoHash, record.Name, record.MagnetURI, record.AddedAt, record.DataPath,
		record.Length, string(filesJSON), record.Downloaded, record.Progress, record.State,
		string(movieDetailsJSON),
	)
	return err
}

// GetTorrent retrieves a torrent record by its info hash
func (s *TorrentStore) GetTorrent(infoHash string) (*TorrentRecord, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var record TorrentRecord
	var filesJSON, movieDetailsJSON string
	var addedAt string

	err := s.db.QueryRow(`
		SELECT info_hash, name, magnet_uri, added_at, data_path, 
		       length, files, downloaded, progress, state, movie_details
		FROM torrents WHERE info_hash = ?
	`, infoHash).Scan(
		&record.InfoHash, &record.Name, &record.MagnetURI, &addedAt, &record.DataPath,
		&record.Length, &filesJSON, &record.Downloaded, &record.Progress, &record.State,
		&movieDetailsJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil when no record is found
		}
		return nil, err
	}

	// Parse added_at time
	record.AddedAt, err = time.Parse(time.RFC3339, addedAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal files JSON
	if filesJSON != "" {
		err = json.Unmarshal([]byte(filesJSON), &record.Files)
		if err != nil {
			return nil, err
		}
	}

	// Unmarshal movie details JSON if it exists
	if movieDetailsJSON != "" {
		record.MovieDetails = &MovieDetails{}
		err = json.Unmarshal([]byte(movieDetailsJSON), record.MovieDetails)
		if err != nil {
			return nil, err
		}
	}

	return &record, nil
}

// GetAllTorrents retrieves all torrent records from the database
func (s *TorrentStore) GetAllTorrents() ([]*TorrentRecord, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	rows, err := s.db.Query(`
		SELECT info_hash, name, magnet_uri, added_at, data_path, 
		       length, files, downloaded, progress, state, movie_details
		FROM torrents
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []*TorrentRecord

	for rows.Next() {
		var record TorrentRecord
		var filesJSON, movieDetailsJSON string
		var addedAt string

		err := rows.Scan(
			&record.InfoHash, &record.Name, &record.MagnetURI, &addedAt, &record.DataPath,
			&record.Length, &filesJSON, &record.Downloaded, &record.Progress, &record.State,
			&movieDetailsJSON,
		)
		if err != nil {
			return nil, err
		}

		// Parse added_at time
		record.AddedAt, err = time.Parse(time.RFC3339, addedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal files JSON
		if filesJSON != "" {
			err = json.Unmarshal([]byte(filesJSON), &record.Files)
			if err != nil {
				return nil, err
			}
		}

		// Unmarshal movie details JSON if it exists
		if movieDetailsJSON != "" {
			record.MovieDetails = &MovieDetails{}
			err = json.Unmarshal([]byte(movieDetailsJSON), record.MovieDetails)
			if err != nil {
				return nil, err
			}
		}

		torrents = append(torrents, &record)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return torrents, nil
}

// UpdateTorrent updates an existing torrent record in the database
func (s *TorrentStore) UpdateTorrent(record *TorrentRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if the torrent exists
	var exists bool
	err := s.db.QueryRow("SELECT 1 FROM torrents WHERE info_hash = ?", record.InfoHash).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("torrent with info_hash %s does not exist", record.InfoHash)
		}
		return err
	}

	// Convert Files slice to JSON
	filesJSON, err := json.Marshal(record.Files)
	if err != nil {
		return err
	}

	// Update the torrent record
	_, err = s.db.Exec(`
		UPDATE torrents 
		SET name = ?, magnet_uri = ?, added_at = ?, data_path = ?, length = ?, files = ?, downloaded = ?, progress = ?, state = ?
		WHERE info_hash = ?
	`,
		record.Name, record.MagnetURI, record.AddedAt, record.DataPath,
		record.Length, string(filesJSON), record.Downloaded, record.Progress, record.State, record.InfoHash,
	)
	return err
}

// UpdateTorrentMovieDetail updates an existing torrent record in the database
func (s *TorrentStore) UpdateTorrentMovieDetail(record *TorrentRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if the torrent exists
	var exists bool
	err := s.db.QueryRow("SELECT 1 FROM torrents WHERE info_hash = ?", record.InfoHash).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("torrent with info_hash %s does not exist", record.InfoHash)
		}
		return err
	}

	// Convert Files slice to JSON
	MovieDetails, err := json.Marshal(record.MovieDetails)
	if err != nil {
		return err
	}

	// Update the torrent record
	_, err = s.db.Exec(`
		UPDATE torrents 
		SET name = ?, magnet_uri = ?, added_at = ?, data_path = ?, length = ?, movie_details = ?, downloaded = ?, progress = ?, state = ?
		WHERE info_hash = ?
	`,
		record.Name, record.MagnetURI, record.AddedAt, record.DataPath,
		record.Length, string(MovieDetails), record.Downloaded, record.Progress, record.State, record.InfoHash,
	)
	return err
}

// DeleteTorrent removes a torrent record from the database
func (s *TorrentStore) DeleteTorrent(infoHash string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Delete the torrent record
	result, err := s.db.Exec("DELETE FROM torrents WHERE info_hash = ?", infoHash)
	if err != nil {
		return err
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("torrent with info_hash %s does not exist", infoHash)
	}

	return nil
}
