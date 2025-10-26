package game

import tl "github.com/JoelOtter/termloop"

type GameLevel struct {
	*tl.BaseLevel
	levelNum  int
	score     int
	totalFish int
	kitten    *Kitten
}

func NewGameLevel(levelNum int) *GameLevel {
	level := &GameLevel{
		BaseLevel: tl.NewBaseLevel(tl.Cell{
			Bg: ColorBackground,
			Fg: ColorText,
			Ch: ' ',
		}),
		levelNum:  levelNum,
		score:     0,
		totalFish: 0,
	}

	level.buildLevel()
	return level
}

func (l *GameLevel) buildLevel() {
	switch l.levelNum {
	case 1:
		l.buildLevel1()
	case 2:
		l.buildLevel2()
	case 3:
		l.buildLevel3()
	default:
		l.buildLevel1()
	}
}

func (l *GameLevel) buildLevel1() {
	ground := NewPlatform(0, 22, GameWidth, PlatformNode)
	l.AddEntity(ground)

	platform1 := NewPlatform(15, 18, 12, PlatformPod)
	l.AddEntity(platform1)

	platform2 := NewPlatform(35, 14, 12, PlatformPod)
	l.AddEntity(platform2)

	platform3 := NewPlatform(55, 10, 12, PlatformService)
	l.AddEntity(platform3)

	fish1 := NewFish(20, 16, l.incrementScore)
	l.AddEntity(fish1)
	l.totalFish++

	fish2 := NewFish(40, 12, l.incrementScore)
	l.AddEntity(fish2)
	l.totalFish++

	fish3 := NewFish(60, 8, l.incrementScore)
	l.AddEntity(fish3)
	l.totalFish++

	l.kitten = NewKitten(5, 19, l.BaseLevel)
	l.AddEntity(l.kitten)

	levelText := tl.NewText(2, 1, "Level 1: Tutorial - Collect all fish!", ColorText, ColorBackground)
	l.AddEntity(levelText)

	scoreText := tl.NewText(2, 2, "Score: 0", ColorText, ColorBackground)
	l.AddEntity(scoreText)
}

func (l *GameLevel) buildLevel2() {
	ground := NewPlatform(0, 22, GameWidth, PlatformNode)
	l.AddEntity(ground)

	platform1 := NewPlatform(10, 19, 10, PlatformPod)
	l.AddEntity(platform1)

	platform2 := NewPlatform(25, 16, 10, PlatformPod)
	l.AddEntity(platform2)

	platform3 := NewPlatform(40, 13, 10, PlatformPod)
	l.AddEntity(platform3)

	platform4 := NewPlatform(55, 10, 10, PlatformService)
	l.AddEntity(platform4)

	platform5 := NewPlatform(30, 7, 15, PlatformService)
	l.AddEntity(platform5)

	fish1 := NewFish(13, 17, l.incrementScore)
	l.AddEntity(fish1)
	l.totalFish++

	fish2 := NewFish(28, 14, l.incrementScore)
	l.AddEntity(fish2)
	l.totalFish++

	fish3 := NewFish(43, 11, l.incrementScore)
	l.AddEntity(fish3)
	l.totalFish++

	fish4 := NewFish(58, 8, l.incrementScore)
	l.AddEntity(fish4)
	l.totalFish++

	fish5 := NewFish(37, 5, l.incrementScore)
	l.AddEntity(fish5)
	l.totalFish++

	l.kitten = NewKitten(3, 19, l.BaseLevel)
	l.AddEntity(l.kitten)

	levelText := tl.NewText(2, 1, "Level 2: The Cluster - Navigate the nodes!", ColorText, ColorBackground)
	l.AddEntity(levelText)

	scoreText := tl.NewText(2, 2, "Score: 0", ColorText, ColorBackground)
	l.AddEntity(scoreText)
}

func (l *GameLevel) buildLevel3() {
	ground := NewPlatform(0, 22, GameWidth, PlatformNode)
	l.AddEntity(ground)

	platform1 := NewPlatform(8, 19, 10, PlatformPod)
	l.AddEntity(platform1)

	platform2 := NewPlatform(22, 16, 10, PlatformPod)
	l.AddEntity(platform2)

	platform3 := NewPlatform(36, 13, 10, PlatformService)
	l.AddEntity(platform3)

	platform4 := NewPlatform(50, 10, 10, PlatformService)
	l.AddEntity(platform4)

	platform5 := NewPlatform(64, 7, 10, PlatformService)
	l.AddEntity(platform5)

	controlPlane := NewPlatform(30, 4, 20, PlatformControlPlane)
	l.AddEntity(controlPlane)

	fish1 := NewFish(11, 17, l.incrementScore)
	l.AddEntity(fish1)
	l.totalFish++

	fish2 := NewFish(25, 14, l.incrementScore)
	l.AddEntity(fish2)
	l.totalFish++

	fish3 := NewFish(39, 11, l.incrementScore)
	l.AddEntity(fish3)
	l.totalFish++

	fish4 := NewFish(53, 8, l.incrementScore)
	l.AddEntity(fish4)
	l.totalFish++

	fish5 := NewFish(67, 5, l.incrementScore)
	l.AddEntity(fish5)
	l.totalFish++

	fish6 := NewFish(38, 2, l.incrementScore)
	l.AddEntity(fish6)
	l.totalFish++

	l.kitten = NewKitten(3, 19, l.BaseLevel)
	l.AddEntity(l.kitten)

	levelText := tl.NewText(2, 1, "Level 3: Reach the Control Plane!", ColorText, ColorBackground)
	l.AddEntity(levelText)

	scoreText := tl.NewText(2, 2, "Score: 0", ColorText, ColorBackground)
	l.AddEntity(scoreText)
}

func (l *GameLevel) incrementScore() {
	l.score++
}

func (l *GameLevel) GetScore() int {
	return l.score
}

func (l *GameLevel) GetTotalFish() int {
	return l.totalFish
}

func (l *GameLevel) IsComplete() bool {
	return l.score >= l.totalFish
}
