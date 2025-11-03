package game

import (
	"math/rand"

	tl "github.com/JoelOtter/termloop"
)

type FishPattern int

const (
	PatternLow FishPattern = iota
	PatternHigh
	PatternDiagonal
	PatternCluster
)

const (
	FishPoolSize        = 30
	FishSpawnInterval   = 60
	PatternChangeChance = 0.3
	FishSpawnChance     = 0.4
)

type FishSpawner struct {
	*tl.Entity
	fishPool            []*Fish
	activeFish          []*Fish
	lastSpawnX          int
	currentPattern      FishPattern
	level               *GameLevel
	scoreManager        *ScoreManager
	screen              *tl.Screen
	platformManager     *PlatformManager
	ticksSinceLastSpawn int
}

func NewFishSpawner(level *GameLevel, scoreManager *ScoreManager, screen *tl.Screen, platformManager *PlatformManager) *FishSpawner {
	fs := &FishSpawner{
		Entity:              tl.NewEntity(0, 0, 1, 1),
		fishPool:            make([]*Fish, 0, FishPoolSize),
		activeFish:          make([]*Fish, 0),
		lastSpawnX:          35,
		currentPattern:      PatternLow,
		level:               level,
		scoreManager:        scoreManager,
		screen:              screen,
		platformManager:     platformManager,
		ticksSinceLastSpawn: 0,
	}

	for i := 0; i < FishPoolSize; i++ {
		fish := NewFish(0, 0, func() {
			fs.scoreManager.OnFishCollected()
		})
		fs.fishPool = append(fs.fishPool, fish)
	}

	return fs
}

func (fs *FishSpawner) Tick(event tl.Event) {
	levelOffsetX, _ := fs.level.Offset()
	fs.cleanupFish(-levelOffsetX - 20)
}

func (fs *FishSpawner) OnPlatformCreated(platform *Platform) {
	if rand.Float64() > FishSpawnChance {
		return
	}

	if rand.Float64() < PatternChangeChance {
		fs.currentPattern = FishPattern(rand.Intn(4))
	}

	px, py := platform.Position()
	pw, _ := platform.Size()

	switch fs.currentPattern {
	case PatternLow:
		fs.spawnLowPattern(px, py, pw)
	case PatternHigh:
		fs.spawnHighPattern(px, py, pw)
	case PatternDiagonal:
		fishX := px + pw/2 - FishWidth/2
		fishY := py - 5
		if fishY < 3 {
			fishY = 3
		}
		fs.spawnFish(fishX, fishY)
	case PatternCluster:
		fs.spawnClusterPattern(px, py, pw)
	}
}

func (fs *FishSpawner) spawnLowPattern(px, py, pw int) {
	fishX := px + pw/2 - FishWidth/2
	fishY := py - FishHeight - 1

	fs.spawnFish(fishX, fishY)
	fs.lastSpawnX = px + pw
}

func (fs *FishSpawner) spawnHighPattern(px, py, pw int) {
	maxJumpHeight := int(JumpVelocity * JumpVelocity / (2 * Gravity))
	fishX := px + pw/2 - FishWidth/2
	fishY := py - maxJumpHeight

	if fishY < 3 {
		fishY = 3
	}

	fs.spawnFish(fishX, fishY)
	fs.lastSpawnX = px + pw
}

func (fs *FishSpawner) spawnClusterPattern(px, py, pw int) {
	clusterSize := 3 + rand.Intn(2)

	for i := 0; i < clusterSize; i++ {
		offsetX := (i - clusterSize/2) * (FishWidth + 2)
		offsetY := -FishHeight - 1 - (i % 2)

		fishX := px + pw/2 + offsetX
		fishY := py + offsetY

		if fishY < 3 {
			fishY = 3
		}

		fs.spawnFish(fishX, fishY)
	}

	fs.lastSpawnX = px + pw
}

func (fs *FishSpawner) spawnFish(x, y int) {
	fish := fs.getInactiveFish()
	if fish == nil {
		return
	}

	fish.collected = false
	fish.SetPosition(x, y)
	fs.activeFish = append(fs.activeFish, fish)
	fs.level.AddEntity(fish)
	fs.level.IncrementTotalFish()
}

func (fs *FishSpawner) getInactiveFish() *Fish {
	for _, fish := range fs.fishPool {
		if !fs.isFishActive(fish) {
			return fish
		}
	}
	return nil
}

func (fs *FishSpawner) isFishActive(targetFish *Fish) bool {
	for _, fish := range fs.activeFish {
		if fish == targetFish {
			return true
		}
	}
	return false
}

func (fs *FishSpawner) cleanupFish(leftEdge int) {
	newActiveFish := make([]*Fish, 0, len(fs.activeFish))

	for _, fish := range fs.activeFish {
		fx, _ := fish.Position()
		if fx+FishWidth < leftEdge {
			fs.level.RemoveEntity(fish)
		} else {
			newActiveFish = append(newActiveFish, fish)
		}
	}

	fs.activeFish = newActiveFish
}

func (fs *FishSpawner) Draw(screen *tl.Screen) {
}

func (fs *FishSpawner) Reset() {
	for _, fish := range fs.activeFish {
		fs.level.RemoveEntity(fish)
	}
	fs.activeFish = fs.activeFish[:0]
	fs.lastSpawnX = 0
	fs.currentPattern = PatternLow
	fs.ticksSinceLastSpawn = 0
}
