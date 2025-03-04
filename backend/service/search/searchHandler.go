package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	jsonschema "github.com/sashabaranov/go-openai/jsonschema"
	"github.com/torrentplayer/backend/backend"
)

// MovieInfo represents the structured response we want to return to the frontend
type MovieInfo struct {
	Filename      string   `json:"filename"`
	Year          int      `json:"year"`
	PosterURL     string   `json:"posterUrl,omitempty"`
	BackdropURL   string   `json:"backdropUrl,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	Rating        float64  `json:"rating,omitempty"`
	VoteCount     int      `json:"voteCount,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Runtime       int      `json:"runtime,omitempty"`
	TMDBID        int      `json:"tmdbId,omitempty"`
	ReleaseDate   string   `json:"releaseDate,omitempty"`
	OriginalTitle string   `json:"originalTitle,omitempty"`
	Adult         bool     `json:"adult,omitempty"`
	Popularity    float64  `json:"popularity,omitempty"`
	Status        string   `json:"status,omitempty"`
	Tagline       string   `json:"tagline,omitempty"`
}

// JinaResponse represents the response structure from the Jina API
type JinaResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// TMDBSearchResponse represents the response structure from the TMDB search API
type TMDBSearchResponse struct {
	Page         int `json:"page"`
	TotalResults int `json:"total_results"`
	TotalPages   int `json:"total_pages"`
	Results      []struct {
		ID               int     `json:"id"`
		Title            string  `json:"title"`
		OriginalTitle    string  `json:"original_title"`
		Overview         string  `json:"overview"`
		PosterPath       string  `json:"poster_path"`
		BackdropPath     string  `json:"backdrop_path"`
		ReleaseDate      string  `json:"release_date"`
		VoteAverage      float64 `json:"vote_average"`
		VoteCount        int     `json:"vote_count"`
		Popularity       float64 `json:"popularity"`
		Adult            bool    `json:"adult"`
		GenreIDs         []int   `json:"genre_ids"`
		OriginalLanguage string  `json:"original_language"`
	} `json:"results"`
}

// TMDBMovieDetails represents the response structure from the TMDB movie details API
type TMDBMovieDetails struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	ReleaseDate   string  `json:"release_date"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Runtime       int     `json:"runtime"`
	Popularity    float64 `json:"popularity"`
	Adult         bool    `json:"adult"`
	Status        string  `json:"status"`
	Tagline       string  `json:"tagline"`
	Genres        []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
}

type SearchFileResponse struct {
	FileName string `json:"filename"`
	Year     int    `json:"year"`
}

func StructSearchFile(magnet_filename string) (*SearchFileResponse, error) {

	config := openai.DefaultConfig(backend.GetEnv("JINA_API_KEY"))
	config.BaseURL = "https://deepsearch.jina.ai/v1"
	client := openai.NewClientWithConfig(config)

	schema, _ := jsonschema.GenerateSchemaForType(SearchFileResponse{})
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "jina-deepsearch-v1",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "你现在要帮用户根据一个magnet 文件名获取这个magnet 中的电影名称，以及电影上映的年份，并且最后只返回json格式的数据，例如用户输入的是\"s子w：m法s传q.2024.HD1080p.中文字幕.mp4\"，那么你就要在网上搜索用户要处理的信息并加上\"电影\" 关键字，并根据互联网信息然后推断出这部电影的名字是\"狮子王: 木法沙传奇\",然后你回答的就只能是一个json格式的字符串数据\"{\"filename\":\"狮子王: 木法沙传奇\",\"year\":2024}\"\", 不要带任何\"根据提供的文件...\" 等等这种额外信息。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: magnet_filename,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
					Name:        "SearchFileResponse",
					Description: "你现在要帮用户根据一个magnet 文件名获取这个magnet 中的电影名称，以及电影上映的年份，并且最后只返回json格式的数据，例如用户输入的是\"s子w：m法s传q.2024.HD1080p.中文字幕.mp4\", 然后你回答的就只能是一个json格式的字符串数据\"{\"filename\":\"狮子王: 木法沙传奇\",\"year\":2024}\"",
					Strict:      true,
					Schema:      schema,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return nil, errors.New("error making API request: " + err.Error())
	}

	fmt.Println(resp.Choices[0].Message.Content)
	// Extract the content which should be a JSON string
	content := resp.Choices[0].Message.Content

	// Find the closing curly brace and trim everything after it
	if idx := strings.LastIndex(content, "}"); idx >= 0 {
		content = content[:idx+1]
	}
	var result SearchFileResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("error parsing content as SearchFileResponse: %w", err)
	}

	return &result, nil
}

func SearchMovie(magnet_filename string) (MovieInfo, error) {
	if magnet_filename == "" {
		return MovieInfo{}, fmt.Errorf("missing magnet_filename parameter")
	}

	movieInfo, err := StructSearchFile(magnet_filename)
	if err != nil {
		return MovieInfo{}, fmt.Errorf("error struct searching file: %w", err)
	}

	// Try to get complete movie details from TMDB
	updatedMovieInfo, err := GetMovieDetails(movieInfo.FileName, movieInfo.Year)
	if err != nil {
		// Just log the error and continue with basic info
		fmt.Printf("Warning: couldn't get movie details: %v\n", err)
		return MovieInfo{}, nil
	}

	// Copy over the original filename to preserve it
	updatedMovieInfo.Filename = movieInfo.FileName

	// Return the complete movie info
	return updatedMovieInfo, nil
}

// GetMoviePoster is a legacy function that calls GetMovieDetails and only returns the poster URL
func GetMoviePoster(movieName string, year int) (string, error) {
	movieInfo, err := GetMovieDetails(movieName, year)
	if err != nil {
		return "", err
	}
	return movieInfo.PosterURL, nil
}

// GetMovieDetails fetches complete movie information from TMDB API
func GetMovieDetails(movieName string, year int) (MovieInfo, error) {
	// Get the TMDB API key from environment variables
	tmdbAPIKey := backend.GetEnv("TMDB_API_KEY")
	if tmdbAPIKey == "" {
		return MovieInfo{}, fmt.Errorf("TMDB_API_KEY environment variable not set")
	}

	url := "https://api.themoviedb.org/3/search/movie?query=%s&include_adult=true&page=1&year=%d"

	req, _ := http.NewRequest("GET", fmt.Sprintf(url, movieName, year), nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+tmdbAPIKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return MovieInfo{}, err
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	// Read and parse the search response
	var searchResp TMDBSearchResponse
	json.Unmarshal(body, &searchResp)

	// Check if we found any results
	if len(searchResp.Results) == 0 {
		return MovieInfo{}, fmt.Errorf("no movies found matching '%s'", movieName)
	}

	// Get the first result's ID
	movieID := searchResp.Results[0].ID

	detailUrl := "https://api.themoviedb.org/3/movie/%d?language=zh-CN"

	detailReq, _ := http.NewRequest("GET", fmt.Sprintf(detailUrl, movieID), nil)

	detailReq.Header.Add("accept", "application/json")
	detailReq.Header.Add("Authorization", "Bearer "+tmdbAPIKey)

	detailRes, err := http.DefaultClient.Do(detailReq)
	if err != nil {
		return MovieInfo{}, err
	}

	defer detailRes.Body.Close()
	detailBody, _ := io.ReadAll(detailRes.Body)

	// Read and parse the details response
	var details TMDBMovieDetails
	json.Unmarshal(detailBody, &details)

	// Extract the release year from release date
	releaseYear := 0
	if details.ReleaseDate != "" {
		parts := strings.Split(details.ReleaseDate, "-")
		if len(parts) > 0 {
			releaseYear, _ = strconv.Atoi(parts[0])
		}
	}

	// Base URL for poster and backdrop images
	imageBaseURL := "https://image.tmdb.org/t/p/original"

	// Prepare genres array
	genres := make([]string, 0, len(details.Genres))
	for _, genre := range details.Genres {
		genres = append(genres, genre.Name)
	}

	// Create and populate the MovieInfo struct
	movieInfo := MovieInfo{
		Filename:      movieName,
		Year:          releaseYear,
		PosterURL:     imageBaseURL + details.PosterPath,
		BackdropURL:   imageBaseURL + details.BackdropPath,
		Overview:      details.Overview,
		Rating:        details.VoteAverage,
		VoteCount:     details.VoteCount,
		Genres:        genres,
		Runtime:       details.Runtime,
		TMDBID:        details.ID,
		ReleaseDate:   details.ReleaseDate,
		OriginalTitle: details.OriginalTitle,
		Adult:         details.Adult,
		Popularity:    details.Popularity,
		Status:        details.Status,
		Tagline:       details.Tagline,
	}

	return movieInfo, nil
}
