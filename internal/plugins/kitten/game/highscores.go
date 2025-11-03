package game

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/shvbsle/k10s/internal/config"
)

const (
	MaxHighScores      = 10
	HighScoresFileName = "highscores.json"
)

type HighScore struct {
	Score    int       `json:"score"`
	Distance int       `json:"distance"`
	Fish     int       `json:"fish"`
	Date     time.Time `json:"date"`
	PlayerID string    `json:"player_id"`
	ID       int64     `json:"id"`
}

type HighScores struct {
	Entries []HighScore `json:"entries"`
}

func getHighScoresPath() (string, error) {
	pluginDir, err := config.GetPluginDataDir("kitten")
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginDir, HighScoresFileName), nil
}

func LoadHighScores() (*HighScores, error) {
	path, err := getHighScoresPath()
	if err != nil {
		return &HighScores{Entries: []HighScore{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &HighScores{Entries: []HighScore{}}, nil
		}
		return nil, err
	}

	var hs HighScores
	if err := json.Unmarshal(data, &hs); err != nil {
		log.Printf("Warning: corrupted high scores file, resetting: %v", err)
		return &HighScores{Entries: []HighScore{}}, nil
	}

	return &hs, nil
}

func (hs *HighScores) Save() error {
	path, err := getHighScoresPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (hs *HighScores) Add(score HighScore) bool {
	if score.ID == 0 {
		score.ID = time.Now().UnixNano()
	}

	hs.Entries = append(hs.Entries, score)

	sort.Slice(hs.Entries, func(i, j int) bool {
		return hs.Entries[i].Score > hs.Entries[j].Score
	})

	isNewHighScore := len(hs.Entries) <= MaxHighScores

	if len(hs.Entries) > MaxHighScores {
		for i, entry := range hs.Entries {
			if entry.ID == score.ID {
				if i < MaxHighScores {
					isNewHighScore = true
				}
				break
			}
		}
		hs.Entries = hs.Entries[:MaxHighScores]
	}

	return isNewHighScore
}

func (hs *HighScores) IsHighScore(score int) bool {
	if len(hs.Entries) < MaxHighScores {
		return true
	}

	return score > hs.Entries[MaxHighScores-1].Score
}

func (hs *HighScores) GetRank(score int) int {
	for i, entry := range hs.Entries {
		if score >= entry.Score {
			return i + 1
		}
	}

	if len(hs.Entries) < MaxHighScores {
		return len(hs.Entries) + 1
	}

	return 0
}

func (hs *HighScores) GetTop(n int) []HighScore {
	if n > len(hs.Entries) {
		n = len(hs.Entries)
	}
	return hs.Entries[:n]
}

func getPlayerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "player"
	}
	return hostname
}
