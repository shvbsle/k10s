package game

import (
	"fmt"

	tl "github.com/JoelOtter/termloop"
)

const (
	DistancePointsPerUnit = 0.1
	FishBasePoints        = 100
	ComboMultiplierStep   = 0.3
	MaxComboMultiplier    = 5.0
)

type ScoreManager struct {
	*tl.Entity
	distance          int
	fishCollected     int
	combo             int
	totalScore        int
	multiplier        float64
	lastDistance      int
	level             *GameLevel
	distanceScoreFrac float64
}

func NewScoreManager(level *GameLevel) *ScoreManager {
	sm := &ScoreManager{
		Entity:            tl.NewEntity(0, 0, 1, 1),
		distance:          0,
		fishCollected:     0,
		combo:             0,
		totalScore:        0,
		multiplier:        1.0,
		lastDistance:      0,
		level:             level,
		distanceScoreFrac: 0,
	}

	return sm
}

func (sm *ScoreManager) Tick(event tl.Event) {
	if sm.level.kitten == nil {
		return
	}

	currentDistance := sm.level.kitten.GetDistance()
	distanceDelta := currentDistance - sm.lastDistance
	sm.lastDistance = currentDistance

	sm.distance = currentDistance

	sm.distanceScoreFrac += float64(distanceDelta) * DistancePointsPerUnit
	pointsToAdd := int(sm.distanceScoreFrac)
	if pointsToAdd > 0 {
		sm.totalScore += pointsToAdd
		sm.distanceScoreFrac -= float64(pointsToAdd)
	}
}

func (sm *ScoreManager) OnFishCollected() {
	sm.fishCollected++
	sm.combo++

	sm.multiplier = 1.0 + float64(sm.combo)*ComboMultiplierStep
	if sm.multiplier > MaxComboMultiplier {
		sm.multiplier = MaxComboMultiplier
	}

	points := int(float64(FishBasePoints) * sm.multiplier)
	sm.totalScore += points
}

func (sm *ScoreManager) ResetCombo() {
	sm.combo = 0
	sm.multiplier = 1.0
}

func (sm *ScoreManager) GetTotalScore() int {
	return sm.totalScore
}

func (sm *ScoreManager) GetDistance() int {
	return sm.distance
}

func (sm *ScoreManager) GetFishCollected() int {
	return sm.fishCollected
}

func (sm *ScoreManager) GetCombo() int {
	return sm.combo
}

func (sm *ScoreManager) GetMultiplier() float64 {
	return sm.multiplier
}

func (sm *ScoreManager) GetScoreBreakdown() string {
	return fmt.Sprintf("Score: %d | Fish: %d | Distance: %d | Combo: x%.1f",
		sm.totalScore, sm.fishCollected, sm.distance, sm.multiplier)
}

func (sm *ScoreManager) Reset() {
	sm.distance = 0
	sm.fishCollected = 0
	sm.combo = 0
	sm.totalScore = 0
	sm.multiplier = 1.0
	sm.lastDistance = 0
	sm.distanceScoreFrac = 0
}

func (sm *ScoreManager) Draw(screen *tl.Screen) {
}
