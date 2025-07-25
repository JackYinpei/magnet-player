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
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
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
	mutex sync.RWMutex
}

// NewTorrentStore creates a new TorrentStore with improved connection management
func NewTorrentStore(dbManager *DatabaseManager) (*TorrentStore, error) {
	return &TorrentStore{
		db: dbManager.GetDB(),
	}, nil
}

// NewTorrentStoreWithPath creates a TorrentStore with direct path (deprecated)
// Use NewTorrentStore with DatabaseManager instead
func NewTorrentStoreWithPath(dbPath string) (*TorrentStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 基本优化设置
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// 创建表
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &TorrentStore{
		db: db,
	}, nil
}

// createTables creates the necessary tables (used by deprecated constructor)
func createTables(db *sql.DB) error {
	_, err := db.Exec(`
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
	return err
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
		return fmt.Errorf("序列化文件列表失败: %w", err)
	}

	// Convert MovieDetails to JSON if it exists
	var movieDetailsJSON []byte
	if record.MovieDetails != nil {
		movieDetailsJSON, err = json.Marshal(record.MovieDetails)
		if err != nil {
			return fmt.Errorf("序列化电影详情失败: %w", err)
		}
	}

	// Set timestamps
	now := time.Now()
	if record.AddedAt.IsZero() {
		record.AddedAt = now
	}

	// Insert the torrent record with optimized query
	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO torrents (
			info_hash, name, magnet_uri, added_at, data_path, 
			length, files, downloaded, progress, state, movie_details,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.InfoHash, record.Name, record.MagnetURI, record.AddedAt, record.DataPath,
		record.Length, string(filesJSON), record.Downloaded, record.Progress, record.State,
		string(movieDetailsJSON), now, now,
	)
	
	if err != nil {
		return fmt.Errorf("插入种子记录失败: %w", err)
	}

	return nil
}

// GetTorrent retrieves a torrent record by its info hash (with read lock)
func (s *TorrentStore) GetTorrent(infoHash string) (*TorrentRecord, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var record TorrentRecord
	var filesJSON, movieDetailsJSON sql.NullString
	var addedAt, createdAt, updatedAt sql.NullString

	err := s.db.QueryRow(`
		SELECT info_hash, name, magnet_uri, added_at, data_path, 
		       length, files, downloaded, progress, state, movie_details,
		       created_at, updated_at
		FROM torrents WHERE info_hash = ?
	`, infoHash).Scan(
		&record.InfoHash, &record.Name, &record.MagnetURI, &addedAt, &record.DataPath,
		&record.Length, &filesJSON, &record.Downloaded, &record.Progress, &record.State,
		&movieDetailsJSON, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil when no record is found
		}
		return nil, fmt.Errorf("查询种子记录失败: %w", err)
	}

	// Parse timestamps
	if addedAt.Valid {
		record.AddedAt, err = time.Parse(time.RFC3339, addedAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析添加时间失败: %w", err)
		}
	}

	if createdAt.Valid {
		record.CreatedAt, err = time.Parse(time.RFC3339, createdAt.String)
		if err != nil {
			// 向后兼容，如果解析失败就使用AddedAt
			record.CreatedAt = record.AddedAt
		}
	}

	if updatedAt.Valid {
		record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt.String)
		if err != nil {
			// 向后兼容，如果解析失败就使用AddedAt
			record.UpdatedAt = record.AddedAt
		}
	}

	// Unmarshal files JSON
	if filesJSON.Valid && filesJSON.String != "" {
		err = json.Unmarshal([]byte(filesJSON.String), &record.Files)
		if err != nil {
			return nil, fmt.Errorf("反序列化文件列表失败: %w", err)
		}
	}

	// Unmarshal movie details JSON if it exists
	if movieDetailsJSON.Valid && movieDetailsJSON.String != "" {
		record.MovieDetails = &MovieDetails{}
		err = json.Unmarshal([]byte(movieDetailsJSON.String), record.MovieDetails)
		if err != nil {
			return nil, fmt.Errorf("反序列化电影详情失败: %w", err)
		}
	}

	return &record, nil
}

// GetAllTorrents retrieves all torrent records from the database (optimized with read lock and pagination support)
func (s *TorrentStore) GetAllTorrents() ([]*TorrentRecord, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	rows, err := s.db.Query(`
		SELECT info_hash, name, magnet_uri, added_at, data_path, 
		       length, files, downloaded, progress, state, movie_details,
		       created_at, updated_at
		FROM torrents 
		ORDER BY added_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询种子列表失败: %w", err)
	}
	defer rows.Close()

	var torrents []*TorrentRecord

	for rows.Next() {
		var record TorrentRecord
		var filesJSON, movieDetailsJSON sql.NullString
		var addedAt, createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&record.InfoHash, &record.Name, &record.MagnetURI, &addedAt, &record.DataPath,
			&record.Length, &filesJSON, &record.Downloaded, &record.Progress, &record.State,
			&movieDetailsJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描种子记录失败: %w", err)
		}

		// Parse timestamps
		if addedAt.Valid {
			record.AddedAt, err = time.Parse(time.RFC3339, addedAt.String)
			if err != nil {
				return nil, fmt.Errorf("解析添加时间失败: %w", err)
			}
		}

		if createdAt.Valid {
			record.CreatedAt, err = time.Parse(time.RFC3339, createdAt.String)
			if err != nil {
				record.CreatedAt = record.AddedAt // 向后兼容
			}
		}

		if updatedAt.Valid {
			record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt.String)
			if err != nil {
				record.UpdatedAt = record.AddedAt // 向后兼容
			}
		}

		// Unmarshal files JSON
		if filesJSON.Valid && filesJSON.String != "" {
			err = json.Unmarshal([]byte(filesJSON.String), &record.Files)
			if err != nil {
				return nil, fmt.Errorf("反序列化文件列表失败: %w", err)
			}
		}

		// Unmarshal movie details JSON if it exists
		if movieDetailsJSON.Valid && movieDetailsJSON.String != "" {
			record.MovieDetails = &MovieDetails{}
			err = json.Unmarshal([]byte(movieDetailsJSON.String), record.MovieDetails)
			if err != nil {
				return nil, fmt.Errorf("反序列化电影详情失败: %w", err)
			}
		}

		torrents = append(torrents, &record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历种子记录失败: %w", err)
	}

	return torrents, nil
}

// GetTorrentsPaginated 分页获取种子列表
func (s *TorrentStore) GetTorrentsPaginated(limit, offset int) ([]*TorrentRecord, int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 获取总数
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM torrents").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("获取种子总数失败: %w", err)
	}

	// 获取分页数据
	rows, err := s.db.Query(`
		SELECT info_hash, name, magnet_uri, added_at, data_path, 
		       length, files, downloaded, progress, state, movie_details,
		       created_at, updated_at
		FROM torrents 
		ORDER BY added_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询分页种子列表失败: %w", err)
	}
	defer rows.Close()

	var torrents []*TorrentRecord
	for rows.Next() {
		var record TorrentRecord
		var filesJSON, movieDetailsJSON sql.NullString
		var addedAt, createdAt, updatedAt sql.NullString

		err := rows.Scan(
			&record.InfoHash, &record.Name, &record.MagnetURI, &addedAt, &record.DataPath,
			&record.Length, &filesJSON, &record.Downloaded, &record.Progress, &record.State,
			&movieDetailsJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描分页种子记录失败: %w", err)
		}

		// 解析时间戳（简化版，复用上面的逻辑）
		if addedAt.Valid {
			record.AddedAt, _ = time.Parse(time.RFC3339, addedAt.String)
		}
		if createdAt.Valid {
			record.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		} else {
			record.CreatedAt = record.AddedAt
		}
		if updatedAt.Valid {
			record.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
		} else {
			record.UpdatedAt = record.AddedAt
		}

		// 反序列化JSON数据
		if filesJSON.Valid && filesJSON.String != "" {
			json.Unmarshal([]byte(filesJSON.String), &record.Files)
		}
		if movieDetailsJSON.Valid && movieDetailsJSON.String != "" {
			record.MovieDetails = &MovieDetails{}
			json.Unmarshal([]byte(movieDetailsJSON.String), record.MovieDetails)
		}

		torrents = append(torrents, &record)
	}

	return torrents, total, rows.Err()
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
