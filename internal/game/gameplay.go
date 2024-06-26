package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kettek/ebijam24/assets"
	"github.com/kettek/ebijam24/internal/render"
)

type GameStatePlay struct {
	titleTimer     int
	wobbler        float64
	updateTicker   int
	returningDudes []*Dude
	boss           *Enemy
}

func (s *GameStatePlay) Begin(g *Game) {
	// Make sure our dudes are visible.
	for _, d := range g.dudes {
		d.stack.Transparency = 0
		d.shadow.Transparency = 0
	}

	// Add our dudes to the tower!
	g.tower.AddDudes(g.dudes...)

	g.camera.SetRotation(math.Pi / 8)
	g.camera.SetStory(0)
}
func (s *GameStatePlay) End(g *Game) {
	// Reset tower
	g.tower.Reset()

	// Replace our current dudes!
	g.dudes = nil
	g.dudes = append(g.dudes, s.returningDudes...)
	for i, s := range g.tower.Stories {
		if !s.open {
			s.Open()
			if i-1 >= 0 {
				// Remove door, ofc
				g.tower.Stories[i-1].RemoveDoor()
				// Remove old portal
				g.tower.Stories[i-1].RemovePortal()
				g.tower.portalOpen = false
			}
			break
		}
	}

	// Clear out any floating text.
	g.tower.ClearTexts()

	// Clear any lingering enemies
	for _, d := range g.dudes {
		d.enemy = nil
	}

	// Collect gold
	s.CollectGold(g)
	// Collect inventory
	s.CollectInventory(g)
}
func (s *GameStatePlay) Update(g *Game) GameState {
	if g.autoplay {
		if g.selectedDude == nil || g.selectedDude.IsDead() {
			for _, d := range g.dudes {
				if !d.IsDead() {
					g.selectedDude = d
					break
				}
			}
		}
	}
	s.titleTimer++

	if handled, kind := g.CheckUI(); !handled {
		if kind == UICheckClick {
			g.selectedDude = nil
		}
	}

	if g.selectedDude != nil {
		if g.selectedDude.story != nil {
			g.camera.SetStory(g.selectedDude.story.level)
			r := g.selectedDude.trueRotation
			g.camera.SetRotationAt(-r, 1)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.TogglePause()
	}
	s.wobbler += 0.05

	// Update the game!
	if !g.paused {
		s.updateTicker++
		if s.updateTicker > g.speed {
			var req ActivityRequests
			g.tower.Update(&req, g)
			for _, u := range req {
				switch u := u.(type) {
				case DudeDeadActivity:
					g.UpdateInfo()
					aliveDudes := g.GetAliveDudes()

					// If the game has no more alive dudes, we lose
					if len(aliveDudes) == 0 {
						return &GameStateLose{}
					} else if !g.tower.HasAliveDudes() {
						// However, if the game has alive and the tower does not, continue
						g.tower.ClearBodies()
						return &GameStateBuild{}
					}
				case TowerCompleteActivity:
					if !u.dude.IsDead() {
						s.returningDudes = append(s.returningDudes, u.dude)
						g.tower.RemoveDude(u.dude)
						if !g.tower.HasAliveDudes() {
							return &GameStateWin{}
						}
					}
				case TowerLeaveActivity:
					if !u.dude.IsDead() {
						s.returningDudes = append(s.returningDudes, u.dude)
						g.tower.RemoveDude(u.dude)
						if !g.tower.HasAliveDudes() {
							// No more alive dudes! Switch game state, yo.
							g.tower.ClearBodies()
							return &GameStateBuild{}
						}
					}
				case RoomStartBossActivity:
					g.ui.feedback.Msg(FeedbackBad, "a boss -- can ur dudes make it??")
					g.ui.bossPanel.hidden = false
					g.ui.bossPanel.current = float64(u.room.boss.stats.currentHp) / float64(u.room.boss.stats.totalHp)
					g.ui.bossPanel.text.SetText(u.room.boss.Name())
					s.boss = u.room.boss
				case RoomEndBossActivity:
					g.ui.bossPanel.hidden = true
					s.boss = nil
				}
			}

			// FIXME: We need a RoomHurtBossActivity or some such...
			if s.boss != nil {
				g.ui.bossPanel.current = float64(s.boss.stats.currentHp) / float64(s.boss.stats.totalHp)
			}

			s.updateTicker = 0
		}
	}

	return nil
}
func (s *GameStatePlay) Draw(g *Game, screen *ebiten.Image) {

	if s.titleTimer < 240 {
		opts := render.TextOptions{
			Screen: screen,
			Font:   assets.DisplayFont,
			Color:  assets.ColorTitle,
		}
		opts.GeoM.Scale(4, 4)

		w, h := text.Measure("ADVENTURE", opts.Font.Face, opts.Font.LineHeight)
		w *= 4
		h *= 4

		opts.GeoM.Translate(-w/2, -h/2)
		opts.GeoM.Rotate(math.Sin(s.wobbler) * 0.05)
		opts.GeoM.Translate(w/2, h/2)
		opts.GeoM.Translate(float64(screen.Bounds().Dx()/2)-w/2, float64(screen.Bounds().Dy()/4)-h/2)

		render.DrawText(&opts, "ADVENTURE")
	}

	// Draw pause
	if g.paused {
		geom := ebiten.GeoM{}
		geom.Scale(4, 4)
		opts := &render.TextOptions{
			Screen: screen,
			Font:   assets.DisplayFont,
			Color:  color.Black,
			GeoM:   geom,
		}

		w, h := text.Measure("PAUSED", opts.Font.Face, opts.Font.LineHeight)
		w *= 4
		h *= 4

		opts.GeoM.Translate(-w/2, -h/2)
		opts.GeoM.Rotate(math.Sin(s.wobbler) * 0.05)
		opts.GeoM.Translate(w/2, h/2)
		opts.GeoM.Translate(float64(screen.Bounds().Dx()/2)-w/2, float64(screen.Bounds().Dy()/4)-h/2)

		opts.GeoM.Translate(-10, -10)
		opts.Color = color.NRGBA{10, 0, 0, 200}
		render.DrawText(opts, "PAUSED")
		opts.GeoM.Translate(20, 20)
		render.DrawText(opts, "PAUSED")
		opts.Color = assets.ColorHeading
		opts.GeoM.Translate(-10, -10)
		render.DrawText(opts, "PAUSED")
	}
}

// For all dudes, remove their gold and add it to the player's gold.
func (s *GameStatePlay) CollectGold(g *Game) {
	gold := 0
	for _, dude := range g.dudes {
		gold += dude.gold
		dude.gold = 0
	}
	if gold == 0 {
		return
	}
	g.gold += gold

	g.ui.feedback.Msg(FeedbackGood, fmt.Sprintf("%d gold snarfed from yer dudes", int(gold)))
	AddMessage(MessageLoot, fmt.Sprintf("Collected %d gold from dudes.", int(gold)))
	g.UpdateInfo()
}

// For all dudes, remove their inventory and add it to player's inventory
func (s *GameStatePlay) CollectInventory(g *Game) {
	count := 0
	for _, dude := range g.dudes {
		count += len(dude.inventory)
		g.equipment = append(g.equipment, dude.inventory...)
		dude.inventory = make([]*Equipment, 0)
	}
	if count == 0 {
		return
	}
	AddMessage(MessageLoot, fmt.Sprintf("Collected %d items from dudes.", count))
	e := SortEquipment(g.ui.equipmentPanel.sortMethod, g.equipment)
	g.ui.equipmentPanel.SetEquipment(e)
}
