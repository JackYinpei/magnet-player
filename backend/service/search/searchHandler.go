package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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

func SearchMovie(magnet_filename string) (MovieInfo, error) {
	if magnet_filename == "" {
		return MovieInfo{}, fmt.Errorf("missing magnet_filename parameter")
	}
	url := "https://deepsearch.jina.ai/v1/chat/completions"

	payload := []byte(fmt.Sprintf(`
{	
"model": "jina-deepsearch-v1",
"messages": [
{
"role": "user",
"content": "你现在要帮用户根据一个magnet 文件名获取这个magnet 中的电影名称，以及电影上映的年份，并且最后只返回json格式的数据，例如用户输入的是"s子w：m法s传q.2024.HD1080p.中文字幕.mp4"，那么你就要在网上搜索用户要处理的信息并加上"电影" 关键字，并根据互联网信息然后推断出这部电影的名字是"狮子王: 木法沙传奇",然后你回答的就只能是一个json格式的字符串数据"{\"filename\":\"狮子王: 木法沙传奇\",\"year\":2024}", 不要带任何"根据提供的文件..." 等等这种额外信息。"
},
{
"role": "user",
"content": "这里就是你要处理的信息%s"
}
],
"stream": false,
"reasoning_effort": "low",
"max_attempts": 2
}`, magnet_filename))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return MovieInfo{}, fmt.Errorf("error creating request")
	}

	req.Header.Set("Content-Type", "application/json")
	apiKey := backend.GetEnv("JINA_API_KEY")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return MovieInfo{}, fmt.Errorf("error making request to AI service")
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return MovieInfo{}, fmt.Errorf("error reading response from AI service")
	}

	// Parse the Jina API response
	var jinaResp JinaResponse
	if err := json.Unmarshal(bodyBytes, &jinaResp); err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return MovieInfo{}, fmt.Errorf("error parsing response from AI service")
	}

	// Check if we have a valid response
	if len(jinaResp.Choices) == 0 || jinaResp.Choices[0].Message.Content == "" {
		fmt.Println("Invalid response from AI service")
		return MovieInfo{}, fmt.Errorf("invalid response from AI service")
	}

	// Extract the movie info from the AI response
	content := jinaResp.Choices[0].Message.Content

	// Clean up the content by removing any non-JSON text
	// Look for JSON within the response using regex
	re := regexp.MustCompile(`\{.*\}`)
	jsonMatch := re.FindString(content)

	if jsonMatch == "" {
		fmt.Println("Could not find JSON in response:", content)
		return MovieInfo{}, fmt.Errorf("could not extract movie information from AI response")
	}

	// Parse the extracted JSON
	var movieInfo MovieInfo
	if err := json.Unmarshal([]byte(jsonMatch), &movieInfo); err != nil {
		fmt.Println("Error parsing movie info JSON:", err)
		fmt.Println("Raw JSON:", jsonMatch)
		return MovieInfo{}, fmt.Errorf("error parsing movie information")
	}

	// Ensure we have a non-empty filename
	if movieInfo.Filename == "" {
		fmt.Println("Empty movie name in response")
		return MovieInfo{}, fmt.Errorf("could not determine movie name")
	}

	// Try to get complete movie details from TMDB
	updatedMovieInfo, err := GetMovieDetails(movieInfo.Filename, movieInfo.Year)
	if err != nil {
		// Just log the error and continue with basic info
		fmt.Printf("Warning: couldn't get movie details: %v\n", err)
		return movieInfo, nil
	}

	// Copy over the original filename to preserve it
	updatedMovieInfo.Filename = movieInfo.Filename

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

	// First, search for the movie to get its TMDB ID
	searchURL := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?api_key=%s&query=%s",
		tmdbAPIKey, url.QueryEscape(movieName))

	// Add year to search query if available
	if year > 0 {
		searchURL += fmt.Sprintf("&year=%d", year)
	}

	// Make the search request
	resp, err := http.Get(searchURL)
	if err != nil {
		return MovieInfo{}, fmt.Errorf("error making request to TMDB search API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return MovieInfo{}, fmt.Errorf("TMDB search API returned non-OK status: %s", resp.Status)
	}

	// Read and parse the search response
	var searchResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return MovieInfo{}, fmt.Errorf("error parsing TMDB search response: %w", err)
	}

	// Check if we found any results
	if len(searchResp.Results) == 0 {
		return MovieInfo{}, fmt.Errorf("no movies found matching '%s'", movieName)
	}

	// Get the first result's ID
	movieID := searchResp.Results[0].ID

	// Now get the detailed movie information
	detailsURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d?api_key=%s",
		movieID, tmdbAPIKey)

	// Make the details request
	detailsResp, err := http.Get(detailsURL)
	if err != nil {
		return MovieInfo{}, fmt.Errorf("error making request to TMDB details API: %w", err)
	}
	defer detailsResp.Body.Close()

	if detailsResp.StatusCode != http.StatusOK {
		return MovieInfo{}, fmt.Errorf("TMDB details API returned non-OK status: %s", detailsResp.Status)
	}

	// Read and parse the details response
	var details TMDBMovieDetails
	if err := json.NewDecoder(detailsResp.Body).Decode(&details); err != nil {
		return MovieInfo{}, fmt.Errorf("error parsing TMDB details response: %w", err)
	}

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
