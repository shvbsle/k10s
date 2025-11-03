package kitten

import "github.com/shvbsle/k10s/internal/plugins/kitten/game"

type KittenClimberPlugin struct{}

func (k *KittenClimberPlugin) Name() string {
	return "kitten"
}

func (k *KittenClimberPlugin) Description() string {
	return "Kitten Climber - An infinite runner platformer game"
}

func (k *KittenClimberPlugin) Commands() []string {
	return []string{"play", "game", "kitten"}
}

func (k *KittenClimberPlugin) Launch() error {
	return game.LaunchGame()
}

func New() *KittenClimberPlugin {
	return &KittenClimberPlugin{}
}
