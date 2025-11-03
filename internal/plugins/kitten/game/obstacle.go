package game

import tl "github.com/JoelOtter/termloop"

type ObstacleType int

const (
	ObstacleSpike ObstacleType = iota
	ObstacleMoving
)

type Obstacle struct {
	*tl.Entity
	obstacleType ObstacleType
	velocityY    float64
	minY         int
	maxY         int
	active       bool
}

func NewSpike(x, y int) *Obstacle {
	o := &Obstacle{
		Entity:       tl.NewEntity(x, y, 1, 2),
		obstacleType: ObstacleSpike,
		active:       true,
	}

	o.SetCell(0, 0, &tl.Cell{Fg: tl.ColorRed, Ch: '^'})
	o.SetCell(0, 1, &tl.Cell{Fg: tl.ColorRed, Ch: '|'})

	return o
}

func NewMovingHazard(x, y, minY, maxY int) *Obstacle {
	o := &Obstacle{
		Entity:       tl.NewEntity(x, y, 3, 1),
		obstacleType: ObstacleMoving,
		velocityY:    0.05,
		minY:         minY,
		maxY:         maxY,
		active:       true,
	}

	o.SetCell(0, 0, &tl.Cell{Bg: tl.ColorRed, Ch: ' '})
	o.SetCell(1, 0, &tl.Cell{Bg: tl.ColorRed, Ch: ' '})
	o.SetCell(2, 0, &tl.Cell{Bg: tl.ColorRed, Ch: ' '})

	return o
}

func (o *Obstacle) Tick(event tl.Event) {
	if !o.active {
		return
	}

	if o.obstacleType == ObstacleMoving {
		x, y := o.Position()
		newY := float64(y) + o.velocityY

		if newY >= float64(o.maxY) {
			newY = float64(o.maxY)
			o.velocityY = -o.velocityY
		} else if newY <= float64(o.minY) {
			newY = float64(o.minY)
			o.velocityY = -o.velocityY
		}

		o.SetPosition(x, int(newY))
	}
}

func (o *Obstacle) Draw(screen *tl.Screen) {
	if o.active {
		o.Entity.Draw(screen)
	}
}

func (o *Obstacle) IsActive() bool {
	return o.active
}

func (o *Obstacle) Deactivate() {
	o.active = false
}

func (o *Obstacle) Activate(x, y int) {
	o.active = true
	o.SetPosition(x, y)
	if o.obstacleType == ObstacleMoving {
		o.velocityY = 0.05
	}
}
