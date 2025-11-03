package game

import tl "github.com/JoelOtter/termloop"

type Kitten struct {
	*tl.Entity
	velocityY        float64
	velocityX        float64
	positionX        float64
	onGround         bool
	jumpInProgress   bool
	distanceTraveled int
	level            *tl.BaseLevel
	screen           *tl.Screen
	isDead           bool
}

func NewKitten(x, y int, level *tl.BaseLevel, screen *tl.Screen) *Kitten {
	k := &Kitten{
		Entity:           tl.NewEntity(x, y, KittenWidth, KittenHeight),
		velocityY:        0,
		velocityX:        AutoScrollSpeed,
		positionX:        float64(x),
		onGround:         true,
		jumpInProgress:   false,
		distanceTraveled: 0,
		level:            level,
		screen:           screen,
		isDead:           false,
	}

	sprite := [][]int{
		{1, 0, 1, 0, 1},
		{1, 2, 1, 2, 1},
		{0, 1, 1, 1, 0},
	}

	colorMap := map[int]tl.Attr{
		1: tl.ColorYellow,
		2: tl.ColorWhite,
	}

	for j := 0; j < KittenHeight; j++ {
		for i := 0; i < KittenWidth; i++ {
			if sprite[j][i] != 0 {
				k.SetCell(i, j, &tl.Cell{Bg: colorMap[sprite[j][i]]})
			}
		}
	}

	return k
}

func (k *Kitten) Tick(event tl.Event) {
	_, screenHeight := k.screen.Size()

	if event.Type == tl.EventKey {
		switch event.Key {
		case tl.KeySpace:
			if k.onGround && !k.jumpInProgress {
				k.velocityY = JumpVelocity
				k.jumpInProgress = true
			}
		}
	}

	k.velocityY += Gravity
	x, y := k.Position()

	k.positionX += k.velocityX
	newX := int(k.positionX)
	newY := float64(y) + k.velocityY

	finalY := int(newY)

	if finalY > screenHeight+10 {
		k.isDead = true
		return
	}

	actualMovement := newX - x
	k.SetPosition(newX, finalY)
	k.distanceTraveled += actualMovement
}

func (k *Kitten) Draw(screen *tl.Screen) {
	screenWidth, _ := screen.Size()
	kittenX, _ := k.Position()

	targetX := screenWidth / 3
	offsetX := targetX - kittenX

	k.level.SetOffset(offsetX, 0)

	k.Entity.Draw(screen)
}

func (k *Kitten) Collide(collision tl.Physical) {
	if platform, ok := collision.(*Platform); ok {
		kx, ky := k.Position()
		px, py := platform.Position()
		pw, ph := platform.Size()

		kittenBottom := ky + KittenHeight
		kittenRight := kx + KittenWidth
		platformRight := px + pw

		horizontalOverlap := kittenRight > px && kx < platformRight

		if k.velocityY >= 0 && kittenBottom >= py && ky < py+ph && horizontalOverlap {
			k.SetPosition(kx, py-KittenHeight)
			k.velocityY = 0
			k.onGround = true
			k.jumpInProgress = false
		} else if k.velocityY < 0 && ky <= py+ph && kittenBottom > py && horizontalOverlap {
			k.SetPosition(kx, py+ph)
			k.velocityY = 0
		}
	}

	if fish, ok := collision.(*Fish); ok {
		if !fish.collected {
			fish.Collect()
		}
	}

	if obstacle, ok := collision.(*Obstacle); ok {
		if obstacle.IsActive() {
			k.isDead = true
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
	k.jumpInProgress = false
}

func (k *Kitten) IsDead() bool {
	return k.isDead
}

func (k *Kitten) GetDistance() int {
	return k.distanceTraveled
}

func (k *Kitten) Reset(x, y int) {
	k.SetPosition(x, y)
	k.positionX = float64(x)
	k.velocityY = 0
	k.velocityX = AutoScrollSpeed
	k.onGround = true
	k.jumpInProgress = false
	k.distanceTraveled = 0
	k.isDead = false
}
