package game

import (
	"math/rand"

	tl "github.com/JoelOtter/termloop"
)

const (
	AutoScrollSpeed     = 0.5
	PlatformPoolSize    = 25
	MinPlatformGap      = 8
	MaxPlatformGap      = 20
	MinPlatformWidth    = 8
	MaxPlatformWidth    = 18
	SpawnThreshold      = 80
	BaseY               = 15
	YVariation          = 6
	DifficultyIncrement = 0.0001
	MaxDifficulty       = 1.0
	StartingLedgeWidth  = 50
)

type PlatformManager struct {
	*tl.Entity
	platforms      []*Platform
	lastPlatformX  int
	difficulty     float64
	level          *tl.BaseLevel
	screen         *tl.Screen
	nextPlatformID int
	fishSpawner    *FishSpawner
}

func NewPlatformManager(level *tl.BaseLevel, screen *tl.Screen) *PlatformManager {
	pm := &PlatformManager{
		Entity:         tl.NewEntity(0, 0, 1, 1),
		platforms:      make([]*Platform, 0, PlatformPoolSize),
		lastPlatformX:  -20,
		difficulty:     0.0,
		level:          level,
		screen:         screen,
		nextPlatformID: 0,
	}

	pm.initializePlatforms()

	return pm
}

func (pm *PlatformManager) initializePlatforms() {
	_, screenHeight := pm.screen.Size()
	startingLedgeY := screenHeight / 2

	startingLedge := NewPlatform(0, startingLedgeY, StartingLedgeWidth, PlatformControlPlane)
	pm.platforms = append(pm.platforms, startingLedge)
	pm.level.AddEntity(startingLedge)
	pm.lastPlatformX = StartingLedgeWidth + 5

	for i := 0; i < 8; i++ {
		pm.spawnNextPlatform()
	}
}

func (pm *PlatformManager) GetStartingLedgeY() int {
	_, screenHeight := pm.screen.Size()
	return screenHeight / 2
}

func (pm *PlatformManager) SetFishSpawner(fs *FishSpawner) {
	pm.fishSpawner = fs
}

func (pm *PlatformManager) Tick(event tl.Event) {
	pm.difficulty += DifficultyIncrement
	if pm.difficulty > MaxDifficulty {
		pm.difficulty = MaxDifficulty
	}

	screenWidth, _ := pm.screen.Size()
	levelOffsetX, _ := pm.level.Offset()

	rightEdge := -levelOffsetX + screenWidth

	for rightEdge+SpawnThreshold > pm.lastPlatformX {
		pm.spawnNextPlatform()
	}

	leftEdge := -levelOffsetX - 20

	toRemove := []*Platform{}
	for _, platform := range pm.platforms {
		px, _ := platform.Position()
		pw, _ := platform.Size()
		if px+pw < leftEdge {
			toRemove = append(toRemove, platform)
		}
	}

	for _, platform := range toRemove {
		pm.removePlatform(platform)
	}
}

func (pm *PlatformManager) spawnNextPlatform() {
	gap := pm.calculateGap()
	x := pm.lastPlatformX + gap
	y := pm.calculateY()
	width := pm.calculateWidth()
	pType := pm.selectPlatformType()

	platform := NewPlatform(x, y, width, pType)
	pm.platforms = append(pm.platforms, platform)
	pm.level.AddEntity(platform)

	if pm.fishSpawner != nil {
		pm.fishSpawner.OnPlatformCreated(platform)
	}

	pm.lastPlatformX = x + width
	pm.nextPlatformID++
}

func (pm *PlatformManager) calculateGap() int {
	baseGap := MinPlatformGap
	maxGap := MaxPlatformGap

	difficultyFactor := pm.difficulty * 0.5

	gapRange := maxGap - baseGap
	adjustedMax := baseGap + int(float64(gapRange)*difficultyFactor)

	if adjustedMax > maxGap {
		adjustedMax = maxGap
	}

	if adjustedMax <= baseGap {
		return baseGap
	}

	return baseGap + rand.Intn(adjustedMax-baseGap)
}

func (pm *PlatformManager) calculateY() int {
	_, screenHeight := pm.screen.Size()
	baseY := screenHeight - BaseY

	maxJumpHeight := int(JumpVelocity * JumpVelocity / (2 * Gravity))
	yRange := YVariation
	if yRange > maxJumpHeight/2 {
		yRange = maxJumpHeight / 2
	}

	yOffset := rand.Intn(yRange*2) - yRange

	finalY := baseY + yOffset

	minY := 5
	maxY := screenHeight - 3
	if finalY < minY {
		finalY = minY
	}
	if finalY > maxY {
		finalY = maxY
	}

	return finalY
}

func (pm *PlatformManager) calculateWidth() int {
	baseWidth := MaxPlatformWidth
	minWidth := MinPlatformWidth

	shrinkFactor := pm.difficulty * 0.3

	width := baseWidth - int(float64(baseWidth-minWidth)*shrinkFactor)

	if width < minWidth {
		width = minWidth
	}

	return width
}

func (pm *PlatformManager) selectPlatformType() PlatformType {
	roll := rand.Float64()

	if roll < 0.5 {
		return PlatformPod
	} else if roll < 0.75 {
		return PlatformNode
	} else if roll < 0.95 {
		return PlatformService
	}
	return PlatformControlPlane
}

func (pm *PlatformManager) removePlatform(platform *Platform) {
	pm.level.RemoveEntity(platform)

	for i, p := range pm.platforms {
		if p == platform {
			pm.platforms = append(pm.platforms[:i], pm.platforms[i+1:]...)
			break
		}
	}
}

func (pm *PlatformManager) Draw(screen *tl.Screen) {
}

func (pm *PlatformManager) Reset() {
	for _, platform := range pm.platforms {
		pm.level.RemoveEntity(platform)
	}
	pm.platforms = pm.platforms[:0]
	pm.lastPlatformX = -20
	pm.difficulty = 0.0
	pm.nextPlatformID = 0

	pm.initializePlatforms()
}
