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

	for i, line := range TitleScreen {
		text := tl.NewText(2, 2+i, line, ColorText, ColorBackground)
		titleLevel.AddEntity(text)
	}

	titleScreen := &TitleScreenEntity{game: g}
	titleLevel.AddEntity(titleScreen)

	g.game.Screen().SetLevel(titleLevel)
}

func (g *Game) startLevel(levelNum int) {
	g.currentLevel = levelNum
	g.level = NewGameLevel(levelNum)

	levelController := &LevelController{
		game:  g,
		level: g.level,
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

	congratsText := tl.NewText(GameWidth/2-15, GameHeight/2-3, "CONGRATULATIONS!", ColorControlPlane, ColorBackground)
	winLevel.AddEntity(congratsText)

	messageText := tl.NewText(GameWidth/2-20, GameHeight/2-1, "You helped the kitten reach the Control Plane!", ColorText, ColorBackground)
	winLevel.AddEntity(messageText)

	exitText := tl.NewText(GameWidth/2-16, GameHeight/2+2, "Press Ctrl+C to return to k10s", ColorText, ColorBackground)
	winLevel.AddEntity(exitText)

	winScreen := &WinScreenEntity{game: g}
	winLevel.AddEntity(winScreen)

	g.game.Screen().SetLevel(winLevel)
}

type TitleScreenEntity struct {
	game *Game
}

func (t *TitleScreenEntity) Draw(screen *tl.Screen) {}

func (t *TitleScreenEntity) Tick(event tl.Event) {
	if event.Type == tl.EventKey {
		switch event.Key {
		case tl.KeySpace:
			t.game.startLevel(1)
		}
	}
}

type LevelController struct {
	game      *Game
	level     *GameLevel
	tickCount int
}

func (lc *LevelController) Draw(screen *tl.Screen) {
	scoreText := tl.NewText(2, 2,
		fmt.Sprintf("Score: %d/%d", lc.level.GetScore(), lc.level.GetTotalFish()),
		ColorText, ColorBackground)
	scoreText.Draw(screen)
}

func (lc *LevelController) Tick(event tl.Event) {
	lc.tickCount++

	if lc.tickCount%30 == 0 {
		if lc.level.IsComplete() {
			lc.game.nextLevel()
		}
	}
}

type WinScreenEntity struct {
	game *Game
}

func (w *WinScreenEntity) Draw(screen *tl.Screen) {}

func (w *WinScreenEntity) Tick(event tl.Event) {
}

func LaunchGame() {
	game := NewGame()
	game.Start()
}
