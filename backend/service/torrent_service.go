package service

import (
	"fmt"
	"log"
	"time"

	"github.com/torrentplayer/backend/config"
	"github.com/torrentplayer/backend/db"
	"github.com/torrentplayer/backend/torrent"
)

// TorrentService 种子服务层
type TorrentService struct {
	torrentClient *torrent.Client
	torrentStore  *db.TorrentStore
	config        *config.Config
}

// NewTorrentService 创建种子服务实例
func NewTorrentService(client *torrent.Client, store *db.TorrentStore, cfg *config.Config) *TorrentService {
	return &TorrentService{
		torrentClient: client,
		torrentStore:  store,
		config:        cfg,
	}
}

// AddMagnet 添加磁力链接
func (s *TorrentService) AddMagnet(magnetURI string) (*torrent.TorrentInfo, error) {
	// 验证磁力链接
	if magnetURI == "" {
		return nil, fmt.Errorf("磁力链接不能为空")
	}

	// 调用torrent客户端添加磁力链接
	torrentInfo, err := s.torrentClient.AddMagnet(magnetURI)
	if err != nil {
		return nil, fmt.Errorf("添加磁力链接失败: %w", err)
	}

	// 保存到数据库
	record := &db.TorrentRecord{
		InfoHash:  torrentInfo.InfoHash,
		Name:      torrentInfo.Name,
		MagnetURI: magnetURI,
		AddedAt:   torrentInfo.AddedAt,
		Length:    torrentInfo.Length,
		Progress:  torrentInfo.Progress,
		State:     torrentInfo.State,
	}

	if err := s.torrentStore.AddTorrent(record); err != nil {
		log.Printf("警告: 保存种子到数据库失败: %v", err)
		// 不阻断流程，继续返回种子信息
	}

	return torrentInfo, nil
}

// ListTorrents 获取所有种子列表
func (s *TorrentService) ListTorrents() ([]torrent.TorrentInfo, error) {
	return s.torrentClient.ListTorrents(), nil
}

// GetTorrent 获取指定种子信息
func (s *TorrentService) GetTorrent(infoHash string) (*torrent.TorrentInfo, error) {
	if infoHash == "" {
		return nil, fmt.Errorf("InfoHash不能为空")
	}

	torrentClient, exists := s.torrentClient.GetTorrent(infoHash)
	if !exists {
		return nil, fmt.Errorf("种子不存在")
	}

	// 获取详细信息
	torrents := s.torrentClient.ListTorrents()
	for _, t := range torrents {
		if t.InfoHash == infoHash {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("种子信息获取失败")
}

// ListFiles 获取种子文件列表
func (s *TorrentService) ListFiles(infoHash string) ([]torrent.FileInfo, error) {
	if infoHash == "" {
		return nil, fmt.Errorf("InfoHash不能为空")
	}

	return s.torrentClient.ListFiles(infoHash)
}

// UpdateMovieDetails 更新电影详情
func (s *TorrentService) UpdateMovieDetails(infoHash string, movieDetails *db.MovieDetails) error {
	if infoHash == "" {
		return fmt.Errorf("InfoHash不能为空")
	}

	if movieDetails == nil {
		return fmt.Errorf("电影详情不能为空")
	}

	// 获取现有记录
	record, err := s.torrentStore.GetTorrent(infoHash)
	if err != nil {
		return fmt.Errorf("获取种子记录失败: %w", err)
	}

	if record == nil {
		return fmt.Errorf("种子记录不存在")
	}

	// 更新电影详情
	record.MovieDetails = movieDetails

	// 保存到数据库
	if err := s.torrentStore.UpdateTorrentMovieDetail(record); err != nil {
		return fmt.Errorf("更新电影详情失败: %w", err)
	}

	return nil
}

// GetMovieDetails 获取所有电影详情
func (s *TorrentService) GetMovieDetails() ([]*db.TorrentRecord, error) {
	return s.torrentStore.GetAllTorrents()
}

// SaveTorrentData 保存种子数据
func (s *TorrentService) SaveTorrentData(infoHash string, torrentData *TorrentUpdateData) error {
	if infoHash == "" {
		return fmt.Errorf("InfoHash不能为空")
	}

	if torrentData == nil {
		return fmt.Errorf("种子数据不能为空")
	}

	// 构建更新记录
	record := &db.TorrentRecord{
		InfoHash:   infoHash,
		Name:       torrentData.Name,
		Length:     torrentData.Length,
		Files:      torrentData.Files,
		Downloaded: torrentData.Downloaded,
		Progress:   torrentData.Progress,
		State:      torrentData.State,
		MagnetURI:  torrentData.InfoHash, // 这里可能需要修正
		AddedAt:    torrentData.AddedAt,
	}

	// 更新到数据库
	if err := s.torrentStore.UpdateTorrent(record); err != nil {
		return fmt.Errorf("保存种子数据失败: %w", err)
	}

	return nil
}

// DeleteTorrent 删除种子
func (s *TorrentService) DeleteTorrent(infoHash string) error {
	if infoHash == "" {
		return fmt.Errorf("InfoHash不能为空")
	}

	// 从数据库删除
	if err := s.torrentStore.DeleteTorrent(infoHash); err != nil {
		return fmt.Errorf("删除种子记录失败: %w", err)
	}

	// TODO: 从torrent客户端删除
	// s.torrentClient.RemoveTorrent(infoHash)

	return nil
}

// RestoreTorrentsFromDB 从数据库恢复种子到torrent客户端
func (s *TorrentService) RestoreTorrentsFromDB() error {
	log.Println("正在从数据库恢复种子...")
	
	torrents, err := s.torrentStore.GetAllTorrents()
	if err != nil {
		return fmt.Errorf("从数据库获取种子失败: %w", err)
	}

	restoredCount := 0
	for _, t := range torrents {
		if t.MagnetURI != "" {
			log.Printf("正在恢复种子: %s, %s", t.Name, t.InfoHash)
			
			// 构建完整的磁力链接
			magnetURI := t.MagnetURI
			if !containsString(magnetURI, "magnet:?") {
				magnetURI = "magnet:?xt=urn:btih:" + t.InfoHash
			}
			
			_, err := s.torrentClient.AddMagnet(magnetURI)
			if err != nil {
				log.Printf("恢复种子失败 %s: %v", t.InfoHash, err)
				continue
			}
			restoredCount++
		}
	}
	
	log.Printf("已从数据库恢复 %d/%d 个种子", restoredCount, len(torrents))
	return nil
}

// TorrentUpdateData 种子更新数据结构
type TorrentUpdateData struct {
	InfoHash   string            `json:"infoHash"`
	Name       string            `json:"name"`
	Length     int64             `json:"length"`
	Files      []db.FileInfo     `json:"files"`
	Downloaded int64             `json:"downloaded"`
	Progress   float32           `json:"progress"`
	State      string            `json:"state"`
	AddedAt    time.Time         `json:"addedAt"`
}

// 辅助函数
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}