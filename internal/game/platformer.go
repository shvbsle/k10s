package game

import (
	"fmt"

	tl "github.com/JoelOtter/termloop"
)

type Game struct {
	game         *tl.Game
	state        GameState
	currentLevel int
	level        *GameLevel
}

func NewGame() *Game {
	return &Game{
		game:         tl.NewGame(),
		state:        StateTitleScreen,
		currentLevel: 1,
	}
}

func (g *Game) Start() {
	g.game.Screen().SetFps(30)
	g.game.SetEndKey(tl.KeyCtrlC)
	g.showTitleScreen()
	g.game.Start()
}

func (g *Game) showTitleScreen() {
	titleLevel := tl.NewBaseLevel(tl.Cell{
		Bg: ColorBackground,
		Fg: ColorText,
		Ch: ' ',
	})

	titleScreen := &TitleScreenEntity{game: g}
	titleLevel.AddEntity(titleScreen)

	g.game.Screen().SetLevel(titleLevel)
}

func (g *Game) startLevel(levelNum int) {
	g.startLevelWithState(levelNum, -1, -1, 0)
}

func (g *Game) startLevelWithState(levelNum int, kittenX, kittenY, score int) {
	g.currentLevel = levelNum
	g.level = NewGameLevel(levelNum, g.game.Screen())

	g.level.ensureBuilt()

	if score > 0 {
		g.level.SetScore(score)
	}

	if kittenX >= 0 && kittenY >= 0 && g.level.kitten != nil {
		g.level.kitten.SetPosition(kittenX, kittenY)
	}

	screenWidth, screenHeight := g.game.Screen().Size()
	levelController := &LevelController{
		game:         g,
		level:        g.level,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
	}
	g.level.AddEntity(levelController)

	g.game.Screen().SetLevel(g.level)
	g.state = StatePlaying
}

func (g *Game) nextLevel() {
	if g.currentLevel < 3 {
		g.startLevel(g.currentLevel + 1)
	} else {
		g.showWinScreen()
	}
}

func (g *Game) showWinScreen() {
	g.state = StateWin
	winLevel := tl.NewBaseLevel(tl.Cell{
		Bg: ColorBackground,
		Fg: ColorText,
		Ch: ' ',
	})

	winScreen := &WinScreenEntity{game: g}
	winLevel.AddEntity(winScreen)

	g.game.Screen().SetLevel(winLevel)
}

type TitleScreenEntity struct {
	game *Game
}

func (t *TitleScreenEntity) Draw(screen *tl.Screen) {
	screenWidth, screenHeight := screen.Size()

	titleWidth := 77
	titleHeight := len(TitleScreen)
	startX := (screenWidth - titleWidth) / 2
	startY := (screenHeight - titleHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for i, line := range TitleScreen {
		col := 0
		for _, ch := range line {
			screen.RenderCell(startX+col, startY+i, &tl.Cell{
				Fg: ColorText,
				Bg: ColorBackground,
				Ch: ch,
			})
			col++
		}
	}
}

func (t *TitleScreenEntity) Tick(event tl.Event) {
	if event.Type == tl.EventKey {
		switch event.Key {
		case tl.KeySpace:
			t.game.startLevel(1)
		}
	}
}

type LevelController struct {
	game         *Game
	level        *GameLevel
	tickCount    int
	screenWidth  int
	screenHeight int
}

func (lc *LevelController) Draw(screen *tl.Screen) {
	scoreText := tl.NewText(2, 2,
		fmt.Sprintf("Score: %d/%d", lc.level.GetScore(), lc.level.GetTotalFish()),
		ColorText, ColorBackground)
	scoreText.Draw(screen)
}

func (lc *LevelController) Tick(event tl.Event) {
	lc.tickCount++

	if event.Type == tl.EventResize {
		screenWidth, screenHeight := lc.game.game.Screen().Size()
		if screenWidth != lc.screenWidth || screenHeight != lc.screenHeight {
			lc.level.Resize(lc.screenWidth, lc.screenHeight, screenWidth, screenHeight)
			lc.screenWidth = screenWidth
			lc.screenHeight = screenHeight
			lc.level.AddEntity(lc)
			return
		}
	}

	if lc.tickCount%30 == 0 {
		if lc.level.IsComplete() {
			lc.game.nextLevel()
		}
	}
}

type WinScreenEntity struct {
	game *Game
}

func (w *WinScreenEntity) Draw(screen *tl.Screen) {
	screenWidth, screenHeight := screen.Size()

	congratsMsg := "CONGRATULATIONS!"
	helpMsg := "You helped the kitten reach the Control Plane!"
	exitMsg := "Press Ctrl+C to return to k10s"

	congratsX := screenWidth/2 - len(congratsMsg)/2
	helpX := screenWidth/2 - len(helpMsg)/2
	exitX := screenWidth/2 - len(exitMsg)/2

	for i, ch := range congratsMsg {
		screen.RenderCell(congratsX+i, screenHeight/2-3, &tl.Cell{
			Fg: ColorControlPlane,
			Bg: ColorBackground,
			Ch: ch,
		})
	}

	for i, ch := range helpMsg {
		screen.RenderCell(helpX+i, screenHeight/2-1, &tl.Cell{
			Fg: ColorText,
			Bg: ColorBackground,
			Ch: ch,
		})
	}

	for i, ch := range exitMsg {
		screen.RenderCell(exitX+i, screenHeight/2+2, &tl.Cell{
			Fg: ColorText,
			Bg: ColorBackground,
			Ch: ch,
		})
	}
}

func (w *WinScreenEntity) Tick(event tl.Event) {
}

func LaunchGame() {
	game := NewGame()
	game.Start()
}
