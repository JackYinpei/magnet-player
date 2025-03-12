package torrent

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/torrentplayer/backend/db"
)

// Client wraps the anacrolix/torrent client with our own functions
type Client struct {
	client       *torrent.Client
	torrents     map[string]*torrent.Torrent
	torrentsLock sync.Mutex
}

// TorrentInfo represents information about a torrent
type TorrentInfo struct {
	InfoHash     string     `json:"infoHash"`
	Name         string     `json:"name"`
	Length       int64      `json:"length"`
	Files        []FileInfo `json:"files"`
	Downloaded   int64      `json:"downloaded"`
	Progress     float32    `json:"progress"`
	State        string     `json:"state"`
	AddedAt      time.Time  `json:"addedAt"`
	MovieDetails *db.MovieDetails `json:"movieDetails,omitempty"`
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

// NewClient creates a new torrent client
func NewClient(dataDir string) (*Client, error) {
	cfg := torrent.NewDefaultClientConfig()

	// 基本设置
	cfg.DataDir = dataDir
	cfg.NoUpload = false
	cfg.DisableWebseeds = false
	cfg.DisableTCP = false
	cfg.DisableUTP = false

	// 性能优化配置
	cfg.Seed = true                     // 启用做种
	cfg.ListenPort = 0                  // 随机端口以避免冲突
	cfg.NoDHT = false                   // 启用 DHT
	cfg.NoDefaultPortForwarding = false // 尝试使用 upnp 端口转发
	cfg.DisablePEX = false              // 启用 PEX (Peer Exchange)
	cfg.DropDuplicatePeerIds = true     // 优化连接管理

	// 连接配置
	cfg.EstablishedConnsPerTorrent = 50 // 增加每个种子的连接数
	cfg.TotalHalfOpenConns = 100        // 增加半开连接数
	cfg.TorrentPeersHighWater = 500     // 增加每个种子的最大 peer 数

	// 创建客户端实例
	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	
	// 在创建客户端后，我们将手动为每个新添加的种子配置公共 trackers
	
	return &Client{
		client:   client,
		torrents: make(map[string]*torrent.Torrent),
	}, nil
}

// Close shuts down the torrent client
func (c *Client) Close() {
	c.client.Close()
}

// AddMagnet adds a magnet link to the client
func (c *Client) AddMagnet(magnetURI string) (*TorrentInfo, error) {
	// 验证磁力链接格式
	if !strings.HasPrefix(magnetURI, "magnet:?") {
		return nil, fmt.Errorf("invalid magnet URI format")
	}

	// 添加磁力链接
	t, err := c.client.AddMagnet(magnetURI)
	if err != nil {
		return nil, err
	}

	// 为种子添加更多的 trackers 以提高发现速度
	publicTrackers := []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://tracker.openbittorrent.com:6969/announce",
		"udp://open.stealth.si:80/announce",
		"udp://exodus.desync.com:6969/announce",
		"udp://explodie.org:6969/announce",
		"http://tracker.opentrackr.org:1337/announce",
		"http://tracker.openbittorrent.com:80/announce",
		"udp://tracker.torrent.eu.org:451/announce",
		"udp://tracker.moeking.me:6969/announce",
		"udp://bt.oiyo.tk:6969/announce",
		"https://tracker.nanoha.org:443/announce",
		"https://tracker.lilithraws.org:443/announce",
	}

	for _, tracker := range publicTrackers {
		t.AddTrackers([][]string{{tracker}})
	}

	// 等待元数据，设置超时 (降低超时时间以提高体验)
	metadataTimeout := time.NewTimer(30 * time.Second)
	defer metadataTimeout.Stop()

	select {
	case <-t.GotInfo():
		// 继续处理
	case <-metadataTimeout.C:
		return nil, fmt.Errorf("timeout waiting for torrent metadata")
	}

	// 安全检查 - 确保 Info() 不为 nil
	if t.Info() == nil {
		return nil, fmt.Errorf("failed to get torrent info")
	}

	// 开始下载前进行额外的安全检查
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in DownloadAll: %v\n", r)
		}
	}()

	// 尝试启动下载
	safeDownloadAll(t)

	// 设置高优先级
	t.SetMaxEstablishedConns(100) // 允许更多的连接

	c.torrentsLock.Lock()
	defer c.torrentsLock.Unlock()

	// 保存种子信息
	infoHash := t.InfoHash().String()
	c.torrents[infoHash] = t

	// 返回种子信息
	return c.getTorrentInfo(t), nil
}

// safeDownloadAll 是 DownloadAll 的安全包装版本
func safeDownloadAll(t *torrent.Torrent) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in DownloadAll: %v\n", r)
		}
	}()

	if t != nil && t.Info() != nil {
		t.DownloadAll()
	}
}

// GetTorrent returns a torrent by info hash
func (c *Client) GetTorrent(infoHash string) (*torrent.Torrent, bool) {
	c.torrentsLock.Lock()
	defer c.torrentsLock.Unlock()

	t, ok := c.torrents[infoHash]
	return t, ok
}

// ListTorrents returns a list of all torrents
func (c *Client) ListTorrents() []TorrentInfo {
	c.torrentsLock.Lock()
	defer c.torrentsLock.Unlock()

	var infos []TorrentInfo
	for _, t := range c.torrents {
		infos = append(infos, *c.getTorrentInfo(t))
	}
	return infos
}

// ListFiles returns a list of all files in a torrent
func (c *Client) ListFiles(infoHash string) ([]FileInfo, error) {
	c.torrentsLock.Lock()
	t, ok := c.torrents[infoHash]
	c.torrentsLock.Unlock()

	if !ok {
		return nil, fmt.Errorf("torrent not found")
	}

	// 确保我们有种子信息
	if !t.Info().IsDir() && len(t.Files()) == 0 {
		// 单文件种子但文件列表为空，可能是元数据尚未完全下载
		return nil, fmt.Errorf("torrent metadata not yet complete")
	}

	files := make([]FileInfo, 0, len(t.Files()))

	// 文件索引和完成度
	for i, f := range t.Files() {
		// 检查文件是否为视频
		ext := strings.ToLower(filepath.Ext(f.DisplayPath()))
		isVideo := isVideoFile(ext)

		// 计算文件的下载进度
		bytesCompleted := f.BytesCompleted()
		fileLength := f.Length()
		progress := float32(0)

		if fileLength > 0 {
			progress = float32(bytesCompleted) / float32(fileLength)
		}

		// 判断是否可播放 (对视频文件，如果有至少5%的内容已下载，就认为可以开始播放)
		isPlayable := false
		if isVideo {
			// 对于视频文件：
			// 1. 如果已完成超过5%，认为有足够的缓冲可播放
			// 2. 或者已下载至少5MB数据（足够播放开头）
			isPlayable = progress >= 0.05 || bytesCompleted >= 5*1024*1024

			// 如果文件很小（小于10MB），只需要下载更少的部分
			if fileLength < 10*1024*1024 {
				isPlayable = progress >= 0.02
			}
		}

		files = append(files, FileInfo{
			Path:       f.DisplayPath(),
			Length:     fileLength,
			Progress:   progress,
			FileIndex:  i,
			TorrentID:  infoHash,
			IsVideo:    isVideo,
			IsPlayable: isPlayable,
		})
	}

	return files, nil
}

// getTorrentInfo creates a TorrentInfo struct from a torrent
func (c *Client) getTorrentInfo(t *torrent.Torrent) *TorrentInfo {
	info := t.Info()

	// Calculate total downloaded
	var downloaded int64
	for _, file := range t.Files() {
		downloaded += file.BytesCompleted()
	}

	// Calculate progress
	progress := float32(0)
	if info != nil && info.TotalLength() > 0 {
		progress = float32(downloaded) / float32(info.TotalLength())
	}

	// Get files info
	files := make([]FileInfo, 0, len(t.Files()))
	for i, file := range t.Files() {
		fileProgress := float32(0)
		if file.Length() > 0 {
			fileProgress = float32(file.BytesCompleted()) / float32(file.Length())
		}

		// Check if file is video
		ext := filepath.Ext(file.DisplayPath())
		isVideo := isVideoFile(ext)

		// A file is considered playable if it's a video and has at least some data
		isPlayable := isVideo && file.BytesCompleted() > 0

		files = append(files, FileInfo{
			Path:       file.DisplayPath(),
			Length:     file.Length(),
			Progress:   fileProgress,
			FileIndex:  i,
			TorrentID:  t.InfoHash().String(),
			IsVideo:    isVideo,
			IsPlayable: isPlayable,
		})
	}

	// Determine state
	state := "downloading"
	if t.Complete().Bool() {
		state = "completed"
	} else if t.Stats().ActivePeers == 0 {
		state = "stalled"
	}

	return &TorrentInfo{
		InfoHash:   t.InfoHash().String(),
		Name:       t.Name(),
		Length:     info.TotalLength(),
		Downloaded: downloaded,
		Progress:   progress,
		State:      state,
		Files:      files,
		AddedAt:    time.Now(),
		MovieDetails: nil,
	}
}

// isVideoFile checks if a file extension corresponds to a video file
func isVideoFile(ext string) bool {
	videoExts := map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".avi":  true,
		".mov":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
		".m4v":  true,
		".mpg":  true,
		".mpeg": true,
		".3gp":  true,
		".rmvb": true,
		".ts":   true,
		".m2ts": true,
	}

	return videoExts[ext]
}
