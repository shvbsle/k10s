package game

import tl "github.com/JoelOtter/termloop"

type GameLevel struct {
	*tl.BaseLevel
	levelNum  int
	score     int
	totalFish int
	kitten    *Kitten
	screen    *tl.Screen
	built     bool
	fish      []*Fish
}

func NewGameLevel(levelNum int, screen *tl.Screen) *GameLevel {
	level := &GameLevel{
		BaseLevel: tl.NewBaseLevel(tl.Cell{
			Bg: ColorBackground,
			Fg: ColorText,
			Ch: ' ',
		}),
		levelNum:  levelNum,
		score:     0,
		totalFish: 0,
		screen:    screen,
		built:     false,
	}

	return level
}

func (l *GameLevel) ensureBuilt() {
	l.ensureBuiltWithFish(nil)
}

func (l *GameLevel) ensureBuiltWithFish(collectedFish []bool) {
	if !l.built {
		l.built = true
		switch l.levelNum {
		case 1:
			l.buildLevel1(collectedFish)
		case 2:
			l.buildLevel2(collectedFish)
		case 3:
			l.buildLevel3(collectedFish)
		default:
			l.buildLevel1(collectedFish)
		}
	}
}

func (l *GameLevel) Draw(screen *tl.Screen) {
	l.ensureBuilt()
	l.BaseLevel.Draw(screen)
}

func (l *GameLevel) buildLevel1(collectedFish []bool) {
	screenWidth, screenHeight := l.screen.Size()

	groundY := screenHeight - 2
	ground := NewPlatform(0, groundY, screenWidth, PlatformNode)
	l.AddEntity(ground)

	platform1 := NewPlatform(screenWidth*15/80, screenHeight-6, 12, PlatformPod)
	l.AddEntity(platform1)

	platform2 := NewPlatform(screenWidth*35/80, screenHeight-10, 12, PlatformPod)
	l.AddEntity(platform2)

	platform3 := NewPlatform(screenWidth*55/80, screenHeight-14, 12, PlatformService)
	l.AddEntity(platform3)

	l.fish = make([]*Fish, 0)

	fish1 := NewFish(screenWidth*20/80, screenHeight-8, l.incrementScore)
	l.AddEntity(fish1)
	l.fish = append(l.fish, fish1)
	l.totalFish++

	fish2 := NewFish(screenWidth*40/80, screenHeight-12, l.incrementScore)
	l.AddEntity(fish2)
	l.fish = append(l.fish, fish2)
	l.totalFish++

	fish3 := NewFish(screenWidth*60/80, screenHeight-16, l.incrementScore)
	l.AddEntity(fish3)
	l.fish = append(l.fish, fish3)
	l.totalFish++

	for i, collected := range collectedFish {
		if i < len(l.fish) && collected {
			l.fish[i].collected = true
		}
	}

	l.kitten = NewKitten(5, groundY-3, l.BaseLevel, l.screen)
	l.AddEntity(l.kitten)

	levelText := tl.NewText(2, 1, "Level 1: Tutorial - Collect all fish!", ColorText, ColorBackground)
	l.AddEntity(levelText)

	scoreText := tl.NewText(2, 2, "Score: 0", ColorText, ColorBackground)
	l.AddEntity(scoreText)
}

func (l *GameLevel) buildLevel2(collectedFish []bool) {
	screenWidth, screenHeight := l.screen.Size()

	groundY := screenHeight - 2
	ground := NewPlatform(0, groundY, screenWidth, PlatformNode)
	l.AddEntity(ground)

	platform1 := NewPlatform(screenWidth*10/80, screenHeight-5, 10, PlatformPod)
	l.AddEntity(platform1)

	platform2 := NewPlatform(screenWidth*25/80, screenHeight-8, 10, PlatformPod)
	l.AddEntity(platform2)

	platform3 := NewPlatform(screenWidth*40/80, screenHeight-11, 10, PlatformPod)
	l.AddEntity(platform3)

	platform4 := NewPlatform(screenWidth*55/80, screenHeight-14, 10, PlatformService)
	l.AddEntity(platform4)

	platform5 := NewPlatform(screenWidth*30/80, screenHeight-17, 15, PlatformService)
	l.AddEntity(platform5)

	l.fish = make([]*Fish, 0)

	fish1 := NewFish(screenWidth*13/80, screenHeight-7, l.incrementScore)
	l.AddEntity(fish1)
	l.fish = append(l.fish, fish1)
	l.totalFish++

	fish2 := NewFish(screenWidth*28/80, screenHeight-10, l.incrementScore)
	l.AddEntity(fish2)
	l.fish = append(l.fish, fish2)
	l.totalFish++

	fish3 := NewFish(screenWidth*43/80, screenHeight-13, l.incrementScore)
	l.AddEntity(fish3)
	l.fish = append(l.fish, fish3)
	l.totalFish++

	fish4 := NewFish(screenWidth*58/80, screenHeight-16, l.incrementScore)
	l.AddEntity(fish4)
	l.fish = append(l.fish, fish4)
	l.totalFish++

	fish5 := NewFish(screenWidth*37/80, screenHeight-19, l.incrementScore)
	l.AddEntity(fish5)
	l.fish = append(l.fish, fish5)
	l.totalFish++

	for i, collected := range collectedFish {
		if i < len(l.fish) && collected {
			l.fish[i].collected = true
		}
	}

	l.kitten = NewKitten(3, groundY-3, l.BaseLevel, l.screen)
	l.AddEntity(l.kitten)

	levelText := tl.NewText(2, 1, "Level 2: The Cluster - Navigate the nodes!", ColorText, ColorBackground)
	l.AddEntity(levelText)

	scoreText := tl.NewText(2, 2, "Score: 0", ColorText, ColorBackground)
	l.AddEntity(scoreText)
}

func (l *GameLevel) buildLevel3(collectedFish []bool) {
	screenWidth, screenHeight := l.screen.Size()

	groundY := screenHeight - 2
	ground := NewPlatform(0, groundY, screenWidth, PlatformNode)
	l.AddEntity(ground)

	platform1 := NewPlatform(screenWidth*8/80, screenHeight-5, 10, PlatformPod)
	l.AddEntity(platform1)

	platform2 := NewPlatform(screenWidth*22/80, screenHeight-8, 10, PlatformPod)
	l.AddEntity(platform2)

	platform3 := NewPlatform(screenWidth*36/80, screenHeight-11, 10, PlatformService)
	l.AddEntity(platform3)

	platform4 := NewPlatform(screenWidth*50/80, screenHeight-14, 10, PlatformService)
	l.AddEntity(platform4)

	platform5 := NewPlatform(screenWidth*64/80, screenHeight-17, 10, PlatformService)
	l.AddEntity(platform5)

	controlPlane := NewPlatform(screenWidth*30/80, screenHeight-20, 20, PlatformControlPlane)
	l.AddEntity(controlPlane)

	l.fish = make([]*Fish, 0)

	fish1 := NewFish(screenWidth*11/80, screenHeight-7, l.incrementScore)
	l.AddEntity(fish1)
	l.fish = append(l.fish, fish1)
	l.totalFish++

	fish2 := NewFish(screenWidth*25/80, screenHeight-10, l.incrementScore)
	l.AddEntity(fish2)
	l.fish = append(l.fish, fish2)
	l.totalFish++

	fish3 := NewFish(screenWidth*39/80, screenHeight-13, l.incrementScore)
	l.AddEntity(fish3)
	l.fish = append(l.fish, fish3)
	l.totalFish++

	fish4 := NewFish(screenWidth*53/80, screenHeight-16, l.incrementScore)
	l.AddEntity(fish4)
	l.fish = append(l.fish, fish4)
	l.totalFish++

	fish5 := NewFish(screenWidth*67/80, screenHeight-19, l.incrementScore)
	l.AddEntity(fish5)
	l.fish = append(l.fish, fish5)
	l.totalFish++

	fish6 := NewFish(screenWidth*38/80, screenHeight-22, l.incrementScore)
	l.AddEntity(fish6)
	l.fish = append(l.fish, fish6)
	l.totalFish++

	for i, collected := range collectedFish {
		if i < len(l.fish) && collected {
			l.fish[i].collected = true
		}
	}

	l.kitten = NewKitten(3, groundY-3, l.BaseLevel, l.screen)
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

func (l *GameLevel) SetScore(score int) {
	l.score = score
}

func (l *GameLevel) GetTotalFish() int {
	return l.totalFish
}

func (l *GameLevel) IsComplete() bool {
	return l.score >= l.totalFish
}

func (l *GameLevel) Resize(oldWidth, oldHeight, newWidth, newHeight int) {
	if !l.built || l.kitten == nil {
		return
	}

	kittenX, kittenY, velocityY, onGround := l.kitten.GetState()
	currentScore := l.score

	collectedFish := make([]bool, len(l.fish))
	for i, fish := range l.fish {
		collectedFish[i] = fish.IsCollected()
	}

	scaleX := float64(newWidth) / float64(oldWidth)
	scaleY := float64(newHeight) / float64(oldHeight)

	newKittenX := int(float64(kittenX) * scaleX)
	newKittenY := int(float64(kittenY) * scaleY)

	l.BaseLevel = tl.NewBaseLevel(tl.Cell{
		Bg: ColorBackground,
		Fg: ColorText,
		Ch: ' ',
	})

	l.built = false
	l.totalFish = 0
	l.score = currentScore
	l.kitten = nil
	l.fish = nil

	l.ensureBuiltWithFish(collectedFish)

	if l.kitten != nil {
		l.kitten.SetState(newKittenX, newKittenY, velocityY, onGround)
	}
}
