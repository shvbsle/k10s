package game

import tl "github.com/JoelOtter/termloop"

type Kitten struct {
	*tl.Entity
	velocityY float64
	onGround  bool
	level     *tl.BaseLevel
	screen    *tl.Screen
}

func NewKitten(x, y int, level *tl.BaseLevel, screen *tl.Screen) *Kitten {
	k := &Kitten{
		Entity:    tl.NewEntity(x, y, KittenWidth, KittenHeight),
		velocityY: 0,
		onGround:  false,
		level:     level,
		screen:    screen,
	}

	k.SetCell(0, 0, &tl.Cell{Fg: ColorKitten, Ch: '/'})
	k.SetCell(1, 0, &tl.Cell{Fg: ColorKitten, Ch: '\\'})
	k.SetCell(2, 0, &tl.Cell{Fg: ColorKitten, Ch: '_'})
	k.SetCell(3, 0, &tl.Cell{Fg: ColorKitten, Ch: '/'})
	k.SetCell(4, 0, &tl.Cell{Fg: ColorKitten, Ch: '\\'})

	k.SetCell(0, 1, &tl.Cell{Fg: ColorKitten, Ch: '('})
	k.SetCell(1, 1, &tl.Cell{Fg: ColorKitten, Ch: 'o'})
	k.SetCell(2, 1, &tl.Cell{Fg: ColorKitten, Ch: '.'})
	k.SetCell(3, 1, &tl.Cell{Fg: ColorKitten, Ch: 'o'})
	k.SetCell(4, 1, &tl.Cell{Fg: ColorKitten, Ch: ')'})

	k.SetCell(0, 2, &tl.Cell{Fg: ColorKitten, Ch: ' '})
	k.SetCell(1, 2, &tl.Cell{Fg: ColorKitten, Ch: '>'})
	k.SetCell(2, 2, &tl.Cell{Fg: ColorKitten, Ch: '^'})
	k.SetCell(3, 2, &tl.Cell{Fg: ColorKitten, Ch: '<'})
	k.SetCell(4, 2, &tl.Cell{Fg: ColorKitten, Ch: ' '})

	return k
}

func (k *Kitten) Tick(event tl.Event) {
	wasOnGround := k.onGround
	k.onGround = false

	screenWidth, screenHeight := k.screen.Size()

	if event.Type == tl.EventKey {
		x, y := k.Position()

		switch event.Key {
		case tl.KeyArrowLeft:
			newX := x - MoveSpeed
			if newX < 0 {
				newX = 0
			}
			k.SetPosition(newX, y)
		case tl.KeyArrowRight:
			newX := x + MoveSpeed
			if newX > screenWidth-KittenWidth {
				newX = screenWidth - KittenWidth
			}
			k.SetPosition(newX, y)
		case tl.KeySpace:
			if wasOnGround {
				k.velocityY = JumpVelocity
			}
		}
	}

	k.velocityY += Gravity
	x, y := k.Position()
	newY := float64(y) + k.velocityY

	finalY := int(newY)
	if finalY > screenHeight {
		finalY = screenHeight
		k.velocityY = 0
	}

	k.SetPosition(x, finalY)
}

func (k *Kitten) Draw(screen *tl.Screen) {
	k.Entity.Draw(screen)
}

func (k *Kitten) Collide(collision tl.Physical) {
	if _, ok := collision.(*Platform); ok {
		kx, ky := k.Position()
		_, py := collision.Position()
		_, ph := collision.Size()

		if k.velocityY >= 0 && ky+KittenHeight >= py && ky < py+ph {
			k.SetPosition(kx, py-KittenHeight)
			k.velocityY = 0
			k.onGround = true
		} else if k.velocityY < 0 && ky <= py+ph && ky+KittenHeight > py {
			k.SetPosition(kx, py+ph)
			k.velocityY = 0
		}
	}

	if fish, ok := collision.(*Fish); ok {
		if !fish.collected {
			fish.Collect()
		}
	}
}

func (k *Kitten) GetState() (x, y int, velocityY float64, onGround bool) {
	x, y = k.Position()
	return x, y, k.velocityY, k.onGround
}

func (k *Kitten) SetState(x, y int, velocityY float64, onGround bool) {
	k.SetPosition(x, y)
	k.velocityY = velocityY
	k.onGround = onGround
}
