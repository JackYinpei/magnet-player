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
	Adult            bool    `json:"adult"`
	BackdropPath     string  `json:"backdrop_path,omitempty"`
	GenreIds         []int   `json:"genre_ids,omitempty"`
	Id               int     `json:"id,omitempty"`
	OriginalLanguage string  `json:"original_language,omitempty"`
	OriginalTitle    string  `json:"original_title,omitempty"`
	Overview         string  `json:"overview,omitempty"`
	Popularity       float64 `json:"popularity,omitempty"`
	PosterPath       string  `json:"poster_path,omitempty"`
	ReleaseDate      string  `json:"releaseDate,omitempty"`
	Title            string  `json:"title,omitempty"`
	Video            bool    `json:"video,omitempty"`
	VoteAverage      float64 `json:"vote_average,omitempty"`
	VoteCount        int     `json:"vote_count,omitempty"`
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

// serializeToJSON serializes a struct to JSON or returns an empty string if nil
func serializeToJSON(v interface{}) (sql.NullString, error) {
	if v == nil {
		return sql.NullString{Valid: false}, nil
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{Valid: false}, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return sql.NullString{
		String: string(bytes),
		Valid:  true,
	}, nil
}

// SaveTorrent saves a torrent to the database
func (s *TorrentStore) SaveTorrent(record *TorrentRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 序列化 MovieDetails 为 JSON
	movieDetailsJSON, err := serializeToJSON(record.MovieDetails)
	if err != nil {
		return fmt.Errorf("failed to serialize movie details: %w", err)
	}

	// 序列化 Files 为 JSON
	filesJSON, err := serializeToJSON(record.Files)
	if err != nil {
		return fmt.Errorf("failed to serialize files: %w", err)
	}

	// 使用事务保证操作的原子性
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 插入或更新记录
	_, err = tx.Exec(`
		INSERT INTO torrents (
			info_hash, name, magnet_uri, added_at, data_path, 
			length, files, downloaded, progress, state, movie_details
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(info_hash) DO UPDATE SET
			name = ?,
			magnet_uri = ?,
			added_at = ?,
			data_path = ?,
			length = ?,
			files = ?,
			downloaded = ?,
			progress = ?,
			state = ?,
			movie_details = ?
	`,
		// INSERT 值
		record.InfoHash, record.Name, record.MagnetURI, record.AddedAt, record.DataPath,
		record.Length, filesJSON, record.Downloaded, record.Progress, record.State, movieDetailsJSON,
		// UPDATE 值
		record.Name, record.MagnetURI, record.AddedAt, record.DataPath,
		record.Length, filesJSON, record.Downloaded, record.Progress, record.State, movieDetailsJSON)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetAllTorrents retrieves all torrents from the database
func (s *TorrentStore) GetAllTorrents() ([]*TorrentRecord, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	rows, err := s.db.Query(`
		SELECT 
			info_hash, name, magnet_uri, added_at, data_path, 
			length, files, downloaded, progress, state, movie_details
		FROM torrents
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []*TorrentRecord
	for rows.Next() {
		var (
			record           = &TorrentRecord{}
			addedAt          sql.NullTime
			dataPath         sql.NullString
			length           sql.NullInt64
			filesJSON        sql.NullString
			downloaded       sql.NullInt64
			progress         sql.NullFloat64
			state            sql.NullString
			movieDetailsJSON sql.NullString
		)

		err := rows.Scan(
			&record.InfoHash, &record.Name, &record.MagnetURI, &addedAt, &dataPath,
			&length, &filesJSON, &downloaded, &progress, &state, &movieDetailsJSON,
		)
		if err != nil {
			return nil, err
		}

		// 处理可能为 NULL 的字段
		if addedAt.Valid {
			record.AddedAt = addedAt.Time
		} else {
			record.AddedAt = time.Now()
		}

		if dataPath.Valid {
			record.DataPath = dataPath.String
		}

		if length.Valid {
			record.Length = length.Int64
		}

		if downloaded.Valid {
			record.Downloaded = downloaded.Int64
		}

		if progress.Valid {
			record.Progress = float32(progress.Float64)
		}

		if state.Valid {
			record.State = state.String
		}

		// 解析文件信息的 JSON
		if filesJSON.Valid && filesJSON.String != "" {
			var files []FileInfo
			if err := json.Unmarshal([]byte(filesJSON.String), &files); err != nil {
				log.Printf("警告: 解析种子 %s 的文件信息失败: %v", record.InfoHash, err)
			} else {
				record.Files = files
			}
		}

		// 解析电影详情的 JSON
		if movieDetailsJSON.Valid && movieDetailsJSON.String != "" {
			var movieDetails MovieDetails
			if err := json.Unmarshal([]byte(movieDetailsJSON.String), &movieDetails); err != nil {
				log.Printf("警告: 解析种子 %s 的电影详情失败: %v", record.InfoHash, err)
			} else {
				record.MovieDetails = &movieDetails
			}
		}

		torrents = append(torrents, record)
	}

	if err = rows.Err(); err != nil {
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

	if movieDetails == nil {
		return fmt.Errorf("movie details cannot be nil")
	}

	movieDetailsJSON, err := serializeToJSON(movieDetails)
	if err != nil {
		return fmt.Errorf("failed to serialize movie details: %w", err)
	}

	_, err = s.db.Exec("UPDATE torrents SET movie_details = ? WHERE info_hash = ?",
		movieDetailsJSON, infoHash)
	return err
}

// GetTorrent retrieves a single torrent by its info hash
func (s *TorrentStore) GetTorrent(infoHash string) (*TorrentRecord, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var (
		record           = &TorrentRecord{}
		addedAt          sql.NullTime
		dataPath         sql.NullString
		length           sql.NullInt64
		filesJSON        sql.NullString
		downloaded       sql.NullInt64
		progress         sql.NullFloat64
		state            sql.NullString
		movieDetailsJSON sql.NullString
	)

	err := s.db.QueryRow(`
		SELECT 
			info_hash, name, magnet_uri, added_at, data_path, 
			length, files, downloaded, progress, state, movie_details
		FROM torrents
		WHERE info_hash = ?
	`, infoHash).Scan(
		&record.InfoHash, &record.Name, &record.MagnetURI, &addedAt, &dataPath,
		&length, &filesJSON, &downloaded, &progress, &state, &movieDetailsJSON,
	)

	if err != nil {
		return nil, err
	}

	// 处理可能为 NULL 的字段
	if addedAt.Valid {
		record.AddedAt = addedAt.Time
	} else {
		record.AddedAt = time.Now()
	}

	if dataPath.Valid {
		record.DataPath = dataPath.String
	}

	if length.Valid {
		record.Length = length.Int64
	}

	if downloaded.Valid {
		record.Downloaded = downloaded.Int64
	}

	if progress.Valid {
		record.Progress = float32(progress.Float64)
	}

	if state.Valid {
		record.State = state.String
	}

	// 解析文件信息的 JSON
	if filesJSON.Valid && filesJSON.String != "" {
		var files []FileInfo
		if err := json.Unmarshal([]byte(filesJSON.String), &files); err != nil {
			log.Printf("警告: 解析种子 %s 的文件信息失败: %v", infoHash, err)
		} else {
			record.Files = files
		}
	}

	// 解析电影详情的 JSON
	if movieDetailsJSON.Valid && movieDetailsJSON.String != "" {
		var movieDetails MovieDetails
		if err := json.Unmarshal([]byte(movieDetailsJSON.String), &movieDetails); err != nil {
			log.Printf("警告: 解析种子 %s 的电影详情失败: %v", infoHash, err)
		} else {
			record.MovieDetails = &movieDetails
		}
	}

	return record, nil
}
