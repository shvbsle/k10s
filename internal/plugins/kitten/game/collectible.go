package game

import tl "github.com/JoelOtter/termloop"

type Fish struct {
	*tl.Entity
	collected bool
	onCollect func()
}

func NewFish(x, y int, onCollect func()) *Fish {
	f := &Fish{
		Entity:    tl.NewEntity(x, y, FishWidth, FishHeight),
		collected: false,
		onCollect: onCollect,
	}

	f.SetCell(0, 0, &tl.Cell{Bg: tl.ColorCyan})
	f.SetCell(1, 0, &tl.Cell{Bg: ColorFish})
	f.SetCell(2, 0, &tl.Cell{Bg: tl.ColorCyan})

	return f
}

func (f *Fish) Draw(screen *tl.Screen) {
	if !f.collected {
		f.Entity.Draw(screen)
	}
}

func (f *Fish) Collect() {
	if !f.collected {
		f.collected = true
		if f.onCollect != nil {
			f.onCollect()
		}
	}
}

func (f *Fish) IsCollected() bool {
	return f.collected
}
