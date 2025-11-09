package game

import tl "github.com/JoelOtter/termloop"

type PlatformType int

const (
	PlatformPod PlatformType = iota
	PlatformNode
	PlatformService
	PlatformControlPlane
)

type Platform struct {
	*tl.Entity
	platformType PlatformType
}

func NewPlatform(x, y, width int, pType PlatformType) *Platform {
	p := &Platform{
		Entity:       tl.NewEntity(x, y, width, PlatformHeight),
		platformType: pType,
	}

	var color tl.Attr

	switch pType {
	case PlatformPod:
		color = ColorPod
	case PlatformNode:
		color = ColorNode
	case PlatformService:
		color = ColorService
	case PlatformControlPlane:
		color = ColorControlPlane
	}

	for i := 0; i < width; i++ {
		p.SetCell(i, 0, &tl.Cell{Bg: color})
	}

	return p
}

func (p *Platform) Draw(screen *tl.Screen) {
	p.Entity.Draw(screen)
}
