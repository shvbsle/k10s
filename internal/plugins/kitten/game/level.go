package game

import (
	tl "github.com/JoelOtter/termloop"
)

type GameLevel struct {
	*tl.BaseLevel
	totalFish       int
	kitten          *Kitten
	screen          *tl.Screen
	fish            []*Fish
	effectiveHeight int
}

func NewGameLevel(screen *tl.Screen, effectiveHeight int) *GameLevel {
	if effectiveHeight == 0 {
		_, effectiveHeight = screen.Size()
	}

	level := &GameLevel{
		BaseLevel: tl.NewBaseLevel(tl.Cell{
			Bg: ColorBackground,
			Fg: ColorText,
			Ch: ' ',
		}),
		totalFish:       0,
		screen:          screen,
		effectiveHeight: effectiveHeight,
	}

	return level
}

func (l *GameLevel) GetTotalFish() int {
	return l.totalFish
}

func (l *GameLevel) IncrementTotalFish() {
	l.totalFish++
}

func (l *GameLevel) SetKitten(kitten *Kitten) {
	l.kitten = kitten
}

func (l *GameLevel) GetKitten() *Kitten {
	return l.kitten
}

func (l *GameLevel) AddFish(fish *Fish) {
	l.fish = append(l.fish, fish)
}

func (l *GameLevel) Reset() {
	l.totalFish = 0
	l.fish = nil
}
