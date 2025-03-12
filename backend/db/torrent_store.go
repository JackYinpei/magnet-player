package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// TorrentRecord represents a stored torrent in the database
type TorrentRecord struct {
	InfoHash     string    `json:"infoHash"`
	Name         string    `json:"name"`
	MagnetURI    string    `json:"magnetUri"`
	AddedAt      time.Time `json:"addedAt"`
	DataPath     string    `json:"dataPath,omitempty"`
	MovieDetails *MovieDetails `json:"movieDetails,omitempty"`
}

// MovieDetails represents the movie information
type MovieDetails struct {
	Filename    string   `json:"filename"`
	Year        int      `json:"year,omitempty"`
	PosterUrl   string   `json:"posterUrl,omitempty"`
	BackdropUrl string   `json:"backdropUrl,omitempty"`
	Overview    string   `json:"overview,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	VoteCount   int      `json:"voteCount,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Runtime     int      `json:"runtime,omitempty"`
	TmdbId      int      `json:"tmdbId,omitempty"`
	ReleaseDate string   `json:"releaseDate,omitempty"`
	OriginalTitle string `json:"originalTitle,omitempty"`
	Popularity  float64  `json:"popularity,omitempty"`
	Status      string   `json:"status,omitempty"`
	Files       []FileInfo `json:"files,omitempty"`
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

// SaveTorrent saves a torrent to the database
func (s *TorrentStore) SaveTorrent(record TorrentRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var movieDetailsJSON sql.NullString
	if record.MovieDetails != nil {
		movieDetailsBytes, err := json.Marshal(record.MovieDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal movie details: %w", err)
		}
		movieDetailsJSON = sql.NullString{
			String: string(movieDetailsBytes),
			Valid:  true,
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO torrents (info_hash, name, magnet_uri, added_at, data_path, movie_details)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(info_hash) DO UPDATE SET
			name = ?,
			magnet_uri = ?,
			added_at = ?,
			data_path = ?,
			movie_details = ?
	`, 
	record.InfoHash, 
	record.Name, 
	record.MagnetURI, 
	record.AddedAt, 
	record.DataPath, 
	movieDetailsJSON,
	record.Name, 
	record.MagnetURI, 
	record.AddedAt,
	record.DataPath, 
	movieDetailsJSON)
	
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
			data_path,
			movie_details
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
		var dataPath sql.NullString
		var movieDetailsJSON sql.NullString

		err := rows.Scan(
			&record.InfoHash, 
			&record.Name, 
			&record.MagnetURI, 
			&addedAt,
			&dataPath,
			&movieDetailsJSON,
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

		if dataPath.Valid {
			record.DataPath = dataPath.String
		}

		if movieDetailsJSON.Valid {
			var movieDetails MovieDetails
			if err := json.Unmarshal([]byte(movieDetailsJSON.String), &movieDetails); err != nil {
				log.Printf("Error unmarshaling movie details for %s: %v", record.InfoHash, err)
			} else {
				record.MovieDetails = &movieDetails
			}
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

	_, err := s.db.Exec("DELETE FROM torrents WHERE info_hash = ?", infoHash)
	return err
}

// UpdateMovieInfo updates the movie information for a torrent
func (s *TorrentStore) UpdateMovieInfo(infoHash string, movieDetails *MovieDetails) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var movieDetailsJSON sql.NullString
	if movieDetails != nil {
		movieDetailsBytes, err := json.Marshal(movieDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal movie details: %w", err)
		}
		movieDetailsJSON = sql.NullString{
			String: string(movieDetailsBytes),
			Valid:  true,
		}
	}

	_, err := s.db.Exec("UPDATE torrents SET movie_details = ? WHERE info_hash = ?", 
		movieDetailsJSON, infoHash)
	return err
}

// UpdateTorrentFiles updates the file information for a torrent
func (s *TorrentStore) UpdateTorrentFiles(infoHash string, files []FileInfo) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// First get the current movie details
	var movieDetailsJSON sql.NullString
	err := s.db.QueryRow("SELECT movie_details FROM torrents WHERE info_hash = ?", infoHash).Scan(&movieDetailsJSON)
	if err != nil {
		return err
	}

	var movieDetails MovieDetails
	if movieDetailsJSON.Valid {
		if err := json.Unmarshal([]byte(movieDetailsJSON.String), &movieDetails); err != nil {
			return fmt.Errorf("failed to unmarshal movie details: %w", err)
		}
	}

	// Update the files field
	movieDetails.Files = files

	// Marshal back to JSON
	updatedMovieDetailsBytes, err := json.Marshal(movieDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal updated movie details: %w", err)
	}

	// Update the database
	_, err = s.db.Exec("UPDATE torrents SET movie_details = ? WHERE info_hash = ?", 
		string(updatedMovieDetailsBytes), infoHash)
	return err
}

// GetTorrent retrieves a single torrent by its info hash
func (s *TorrentStore) GetTorrent(infoHash string) (TorrentRecord, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var record TorrentRecord
	var addedAt sql.NullTime
	var dataPath sql.NullString
	var movieDetailsJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT 
			info_hash, 
			name, 
			magnet_uri, 
			added_at,
			data_path,
			movie_details
		FROM torrents
		WHERE info_hash = ?
	`, infoHash).Scan(
		&record.InfoHash,
		&record.Name,
		&record.MagnetURI,
		&addedAt,
		&dataPath,
		&movieDetailsJSON,
	)

	if err != nil {
		return TorrentRecord{}, err
	}

	// Handle null values
	if addedAt.Valid {
		record.AddedAt = addedAt.Time
	} else {
		record.AddedAt = time.Now() // Default to current time if null
	}

	if dataPath.Valid {
		record.DataPath = dataPath.String
	}

	if movieDetailsJSON.Valid {
		var movieDetails MovieDetails
		if err := json.Unmarshal([]byte(movieDetailsJSON.String), &movieDetails); err != nil {
			log.Printf("Error unmarshaling movie details for %s: %v", infoHash, err)
		} else {
			record.MovieDetails = &movieDetails
		}
	}

	return record, nil
}

// UpdateDataPath updates the data path for a torrent
func (s *TorrentStore) UpdateDataPath(infoHash string, dataPath string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, err := s.db.Exec("UPDATE torrents SET data_path = ? WHERE info_hash = ?", dataPath, infoHash)
	return err
}
