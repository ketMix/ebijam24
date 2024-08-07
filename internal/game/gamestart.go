package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kettek/ebijam24/internal/render"
)

type GameStateStart struct {
	newDudes []*Dude
	length   int
}

func (s *GameStateStart) Begin(g *Game) {
	g.titleFadeOutTick = TITLE_FADE_TICK
	g.audioController.PlayRoomTracks()

	if s.length == 0 {
		s.length = 3
	}

	// Give the player a reasonable amount of GOLD
	g.gold = 750

	professions := []ProfessionKind{Knight, Vagabond, Ranger, Cleric}
	dudeLimit := len(professions) * 2
	for i := 0; i < dudeLimit; i++ {
		pk := professions[i%len(professions)]
		dude := NewDude(pk, 1)
		if g.simMode {
			dude.invincible = true
		}
		s.newDudes = append(s.newDudes, dude)
	}

	// Create a new tower, yo.
	tower := NewTower()
	story := NewStory()
	story.Open()
	tower.AddStory(story)
	if s.length == -1 {
		tower.targetStories = -1
	} else {
		tower.targetStories = 3 + s.length*3
	}

	g.tower = tower
	g.camera.SetMode(render.CameraModeTower)
	g.ui.hint.Show()
}
func (s *GameStateStart) End(g *Game) {
	g.dudes = append(g.dudes, s.newDudes...)
	g.camera.SetMode(render.CameraModeStack)
}
func (s *GameStateStart) Update(g *Game) GameState {
	//return &GameStateWin{}
	return &GameStateBuild{}
}
func (s *GameStateStart) Draw(g *Game, screen *ebiten.Image) {
}
