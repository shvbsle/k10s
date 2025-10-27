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
	var label string

	switch pType {
	case PlatformPod:
		color = ColorPod
		label = "[POD]"
	case PlatformNode:
		color = ColorNode
		label = "[NODE]"
	case PlatformService:
		color = ColorService
		label = "[SVC]"
	case PlatformControlPlane:
		color = ColorControlPlane
		label = "[CONTROL PLANE]"
	}

	for i := 0; i < width; i++ {
		p.SetCell(i, 0, &tl.Cell{
			Fg: color,
			Bg: color,
			Ch: '█',
		})
	}

	labelX := (width - len(label)) / 2
	if labelX >= 0 && labelX+len(label) <= width {
		for i, ch := range label {
			if labelX+i < width {
				p.SetCell(labelX+i, 0, &tl.Cell{
					Fg: ColorText,
					Bg: color,
					Ch: ch,
				})
			}
		}
	}

	return p
}

func (p *Platform) Draw(screen *tl.Screen) {
	p.Entity.Draw(screen)
}
