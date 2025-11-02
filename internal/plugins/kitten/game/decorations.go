package game

import tl "github.com/JoelOtter/termloop"

type Stars struct{}

func (s *Stars) Draw(screen *tl.Screen) {
	screenWidth, screenHeight := screen.Size()

	stars := []struct {
		xFraction float64
		yFraction float64
		ch        rune
		color     tl.Attr
	}{
		{0.06, 0.08, '*', tl.ColorWhite},
		{0.15, 0.15, '.', tl.ColorWhite},
		{0.22, 0.04, '✦', tl.ColorYellow},
		{0.31, 0.18, '.', tl.ColorWhite},
		{0.40, 0.11, '*', tl.ColorWhite},
		{0.47, 0.22, '·', tl.ColorWhite},
		{0.56, 0.07, '.', tl.ColorWhite},
		{0.65, 0.14, '*', tl.ColorYellow},
		{0.72, 0.04, '·', tl.ColorWhite},
		{0.81, 0.18, '.', tl.ColorWhite},
		{0.90, 0.11, '*', tl.ColorWhite},
		{0.10, 0.30, '.', tl.ColorWhite},
		{0.19, 0.37, '*', tl.ColorWhite},
		{0.27, 0.26, '·', tl.ColorYellow},
		{0.37, 0.33, '.', tl.ColorWhite},
		{0.50, 0.41, '*', tl.ColorWhite},
		{0.60, 0.30, '·', tl.ColorWhite},
		{0.69, 0.37, '.', tl.ColorWhite},
		{0.77, 0.26, '*', tl.ColorYellow},
		{0.87, 0.33, '.', tl.ColorWhite},
		{0.12, 0.48, '*', tl.ColorWhite},
		{0.25, 0.56, '.', tl.ColorWhite},
		{0.44, 0.52, '·', tl.ColorWhite},
		{0.62, 0.59, '*', tl.ColorWhite},
		{0.75, 0.48, '.', tl.ColorYellow},
		{0.09, 0.67, '.', tl.ColorWhite},
		{0.21, 0.74, '*', tl.ColorWhite},
		{0.35, 0.63, '·', tl.ColorYellow},
		{0.52, 0.70, '.', tl.ColorWhite},
		{0.66, 0.78, '*', tl.ColorWhite},
		{0.80, 0.67, '·', tl.ColorWhite},
		{0.92, 0.74, '.', tl.ColorYellow},
		{0.15, 0.85, '*', tl.ColorWhite},
		{0.31, 0.93, '.', tl.ColorWhite},
		{0.47, 0.81, '·', tl.ColorWhite},
		{0.60, 0.89, '*', tl.ColorYellow},
		{0.72, 0.85, '.', tl.ColorWhite},
		{0.85, 0.93, '·', tl.ColorWhite},
	}

	for _, star := range stars {
		x := int(float64(screenWidth) * star.xFraction)
		y := int(float64(screenHeight) * star.yFraction)

		if x >= 0 && x < screenWidth && y >= 0 && y < screenHeight {
			screen.RenderCell(x, y, &tl.Cell{
				Fg: star.color,
				Bg: ColorBackground,
				Ch: star.ch,
			})
		}
	}
}

func (s *Stars) Tick(event tl.Event) {}
