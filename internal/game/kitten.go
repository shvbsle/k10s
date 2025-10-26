package game

import tl "github.com/JoelOtter/termloop"

type Kitten struct {
	*tl.Entity
	velocityY float64
	onGround  bool
	level     *tl.BaseLevel
}

func NewKitten(x, y int, level *tl.BaseLevel) *Kitten {
	k := &Kitten{
		Entity:    tl.NewEntity(x, y, KittenWidth, KittenHeight),
		velocityY: 0,
		onGround:  false,
		level:     level,
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
	if event.Type == tl.EventKey {
		x, y := k.Position()

		switch event.Key {
		case tl.KeyArrowLeft:
			k.SetPosition(x-MoveSpeed, y)
		case tl.KeyArrowRight:
			k.SetPosition(x+MoveSpeed, y)
		case tl.KeySpace:
			if k.onGround {
				k.velocityY = JumpVelocity
				k.onGround = false
			}
		}
	}

	k.velocityY += Gravity
	x, y := k.Position()
	newY := float64(y) + k.velocityY
	k.SetPosition(x, int(newY))

	k.onGround = false
}

func (k *Kitten) Draw(screen *tl.Screen) {
	k.Entity.Draw(screen)
}

func (k *Kitten) Collide(collision tl.Physical) {
	if _, ok := collision.(*Platform); ok {
		kx, ky := k.Position()
		_, py := collision.Position()
		_, ph := collision.Size()

		if k.velocityY > 0 && ky+KittenHeight >= py && ky < py+ph {
			k.SetPosition(kx, py-KittenHeight)
			k.velocityY = 0
			k.onGround = true
		}
	}

	if fish, ok := collision.(*Fish); ok {
		if !fish.collected {
			fish.Collect()
		}
	}
}
