package game

import (
	"fmt"
	"log"
	"time"

	tl "github.com/JoelOtter/termloop"
)

type Game struct {
	game            *tl.Game
	state           GameState
	level           *GameLevel
	kitten          *Kitten
	platformManager *PlatformManager
	fishSpawner     *FishSpawner
	scoreManager    *ScoreManager
	stars           *Stars
	hud             *HUD
	finalScore      int
	finalDistance   int
	finalFish       int
	rank            int
	isNewHighScore  bool
	highScores      *HighScores
}

func NewGame() *Game {
	return &Game{
		game:  tl.NewGame(),
		state: StateTitleScreen,
	}
}

func (g *Game) Start() error {
	g.game.Screen().SetFps(GameFPS)
	g.game.SetEndKey(tl.KeyCtrlC)
	g.showTitleScreen()
	g.game.Start()
	return nil
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

func (g *Game) startGame() {
	screen := g.game.Screen()
	_, screenHeight := screen.Size()

	g.level = NewGameLevel(screen, screenHeight)

	g.platformManager = NewPlatformManager(g.level.BaseLevel, screen)
	g.level.AddEntity(g.platformManager)

	startingLedgeY := g.platformManager.GetStartingLedgeY()
	kittenStartX := 15
	kittenStartY := startingLedgeY - KittenHeight - 2
	g.kitten = NewKitten(kittenStartX, kittenStartY, g.level.BaseLevel, screen)
	g.level.AddEntity(g.kitten)
	g.level.SetKitten(g.kitten)

	g.scoreManager = NewScoreManager(g.level)
	g.level.AddEntity(g.scoreManager)

	g.fishSpawner = NewFishSpawner(g.level, g.scoreManager, screen, g.platformManager)
	g.level.AddEntity(g.fishSpawner)

	g.platformManager.SetFishSpawner(g.fishSpawner)

	gameController := &GameController{
		game: g,
	}
	g.level.AddEntity(gameController)

	highScores, err := LoadHighScores()
	if err != nil {
		log.Printf("Warning: could not load high scores: %v", err)
		highScores = &HighScores{Entries: []HighScore{}}
	}
	g.highScores = highScores

	if g.stars != nil {
		g.game.Screen().RemoveEntity(g.stars)
	}
	g.stars = &Stars{}
	g.game.Screen().AddEntity(g.stars)

	if g.hud != nil {
		g.game.Screen().RemoveEntity(g.hud)
	}
	g.hud = &HUD{
		scoreManager: g.scoreManager,
		highScores:   g.highScores,
	}
	g.game.Screen().AddEntity(g.hud)

	g.game.Screen().SetLevel(g.level)
	g.state = StatePlaying
}

func (g *Game) showGameOver() {
	g.state = StateLose

	g.finalScore = g.scoreManager.GetTotalScore()
	g.finalDistance = g.scoreManager.GetDistance()
	g.finalFish = g.scoreManager.GetFishCollected()

	g.rank = g.highScores.GetRank(g.finalScore)
	g.isNewHighScore = g.highScores.IsHighScore(g.finalScore)

	if g.isNewHighScore {
		newScore := HighScore{
			Score:    g.finalScore,
			Distance: g.finalDistance,
			Fish:     g.finalFish,
			Date:     time.Now(),
			PlayerID: getPlayerID(),
		}
		g.highScores.Add(newScore)
		if err := g.highScores.Save(); err != nil {
			log.Printf("Warning: could not save high scores: %v", err)
		}
	}

	g.scoreManager.Reset()

	gameOverLevel := tl.NewBaseLevel(tl.Cell{
		Bg: ColorBackground,
		Fg: ColorText,
		Ch: ' ',
	})

	gameOverScreen := &GameOverScreenEntity{game: g}
	gameOverLevel.AddEntity(gameOverScreen)

	g.game.Screen().SetLevel(gameOverLevel)
}

func (g *Game) restart() {
	g.startGame()
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
				Fg: ColorTitle,
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
			t.game.startGame()
		}
	}
}

type GameController struct {
	game *Game
}

func (gc *GameController) Draw(screen *tl.Screen) {
}

func (gc *GameController) Tick(event tl.Event) {
	if gc.game.kitten != nil && gc.game.kitten.IsDead() {
		gc.game.showGameOver()
	}
}

type HUD struct {
	scoreManager *ScoreManager
	highScores   *HighScores
}

func (h *HUD) Draw(screen *tl.Screen) {
	scoreText := fmt.Sprintf("Score: %d", h.scoreManager.GetTotalScore())
	fishText := fmt.Sprintf("Fish: %d", h.scoreManager.GetFishCollected())
	distanceText := fmt.Sprintf("Distance: %d", h.scoreManager.GetDistance())
	comboText := fmt.Sprintf("Combo: x%.1f", h.scoreManager.GetMultiplier())

	h.renderText(screen, 2, 2, scoreText, ColorText)
	h.renderText(screen, 2, 3, fishText, ColorText)
	h.renderText(screen, 2, 4, distanceText, ColorText)

	highScore := 0
	if len(h.highScores.Entries) > 0 {
		topScores := h.highScores.GetTop(1)
		highScore = topScores[0].Score
	}
	highScoreText := fmt.Sprintf("High Score: %d", highScore)
	h.renderText(screen, 2, 5, highScoreText, ColorText)

	if h.scoreManager.GetCombo() > 0 {
		h.renderText(screen, 2, 6, comboText, ColorKitten)
	}

	h.renderText(screen, 2, 8, "Ctrl+C to exit", ColorText)
}

func (h *HUD) renderText(screen *tl.Screen, x, y int, text string, color tl.Attr) {
	for i, ch := range text {
		screen.RenderCell(x+i, y, &tl.Cell{
			Fg: color,
			Bg: ColorBackground,
			Ch: ch,
		})
	}
}

func (h *HUD) Tick(event tl.Event) {
}

type GameOverScreenEntity struct {
	game *Game
}

func (g *GameOverScreenEntity) Draw(screen *tl.Screen) {
	_, screenHeight := screen.Size()

	gameOverMsg := "GAME OVER!"
	scoreMsg := fmt.Sprintf("Final Score: %d", g.game.finalScore)

	rankMsg := ""
	if g.game.isNewHighScore && g.game.rank > 0 {
		rankMsg = fmt.Sprintf("NEW HIGH SCORE! Rank #%d", g.game.rank)
	} else if g.game.rank > 0 {
		rankMsg = fmt.Sprintf("Rank: #%d", g.game.rank)
	}

	distanceMsg := fmt.Sprintf("Distance: %d", g.game.finalDistance)
	fishMsg := fmt.Sprintf("Fish Collected: %d", g.game.finalFish)

	centerY := screenHeight / 2
	currentY := centerY - 8

	g.renderCentered(screen, currentY, gameOverMsg, ColorControlPlane)
	currentY += 2

	g.renderCentered(screen, currentY, scoreMsg, ColorKitten)
	currentY++

	if rankMsg != "" {
		rankColor := ColorText
		if g.game.isNewHighScore {
			rankColor = ColorKitten
		}
		g.renderCentered(screen, currentY, rankMsg, rankColor)
		currentY++
	}

	g.renderCentered(screen, currentY, distanceMsg, ColorText)
	currentY++
	g.renderCentered(screen, currentY, fishMsg, ColorText)
	currentY += 2

	if g.game.highScores != nil && len(g.game.highScores.Entries) > 0 {
		g.renderCentered(screen, currentY, "-- HIGH SCORES --", ColorText)
		currentY++

		topScores := g.game.highScores.GetTop(3)
		for i, entry := range topScores {
			scoreText := fmt.Sprintf("#%d: %d pts (%d fish)", i+1, entry.Score, entry.Fish)
			g.renderCentered(screen, currentY, scoreText, ColorText)
			currentY++
		}
		currentY++
	}

	g.renderCentered(screen, currentY, "Press SPACE to restart", ColorText)
	currentY++
	g.renderCentered(screen, currentY, "Press Ctrl+C to return to k10s", ColorText)
}

func (g *GameOverScreenEntity) renderCentered(screen *tl.Screen, y int, text string, color tl.Attr) {
	screenWidth, _ := screen.Size()
	x := screenWidth/2 - len(text)/2

	for i, ch := range text {
		screen.RenderCell(x+i, y, &tl.Cell{
			Fg: color,
			Bg: ColorBackground,
			Ch: ch,
		})
	}
}

func (g *GameOverScreenEntity) Tick(event tl.Event) {
	if event.Type == tl.EventKey {
		switch event.Key {
		case tl.KeySpace:
			g.game.restart()
		}
	}
}

func LaunchGame() error {
	game := NewGame()
	return game.Start()
}
