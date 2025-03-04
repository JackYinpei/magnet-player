package search

import (
	"log"
	"testing"

	"github.com/torrentplayer/backend/backend"
)

func TestSearch(t *testing.T) {
	backend.LoadEnv()
	res, err := SearchMovie("蜡笔小新：我们的恐龙日记[国日多音轨+中文字幕].2024.1080p.HamiVideo.WEB-DL.AAC2.0.H.264-DreamHD")
	if err != nil {
		t.Error(err)
	}
	t.Log(res)
}

func TestGetMovieDetail(t *testing.T) {
	backend.LoadEnvFrom("/root/magnet-player/backend/.env")
	movieDetail, _ := GetMovieDetails("蜡笔小新：我们的恐龙日记", 2024)
	log.Println(movieDetail)
}
