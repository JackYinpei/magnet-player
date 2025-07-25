package service

import (
	"fmt"

	"github.com/torrentplayer/backend/config"
	"github.com/torrentplayer/backend/service/search"
)

// SearchService 搜索服务层
type SearchService struct {
	config *config.Config
}

// NewSearchService 创建搜索服务实例
func NewSearchService(cfg *config.Config) *SearchService {
	return &SearchService{
		config: cfg,
	}
}

// SearchMovie 搜索电影信息
func (s *SearchService) SearchMovie(filename string) (*search.MovieInfo, error) {
	if filename == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}

	// 调用搜索服务
	movieInfo, err := search.SearchMovie(filename)
	if err != nil {
		return nil, fmt.Errorf("搜索电影失败: %w", err)
	}

	return &movieInfo, nil
}

// GetMovieDetails 获取电影详细信息
func (s *SearchService) GetMovieDetails(movieName string, year int) (*search.MovieInfo, error) {
	if movieName == "" {
		return nil, fmt.Errorf("电影名称不能为空")
	}

	movieInfo, err := search.GetMovieDetails(movieName, year)
	if err != nil {
		return nil, fmt.Errorf("获取电影详情失败: %w", err)
	}

	return &movieInfo, nil
}

// GetMoviePoster 获取电影海报URL
func (s *SearchService) GetMoviePoster(movieName string, year int) (string, error) {
	if movieName == "" {
		return "", fmt.Errorf("电影名称不能为空")
	}

	posterURL, err := search.GetMoviePoster(movieName, year)
	if err != nil {
		return "", fmt.Errorf("获取电影海报失败: %w", err)
	}

	return posterURL, nil
}