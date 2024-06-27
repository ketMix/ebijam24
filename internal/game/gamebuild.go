package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kettek/ebijam24/assets"
	"github.com/kettek/ebijam24/internal/render"
)

type GameStateBuild struct {
	availableRooms   []*RoomDef
	wobbler          float64
	titleTimer       int
	nextStory        *Story
	focusedRoom      *Room
	highlightedRooms []*Room
	placingRoom      *RoomDef
	placingIndex     int
}

func (s *GameStateBuild) Begin(g *Game) {
	g.camera.SetMode(render.CameraModeTower)

	// On build phase, full heal all dudes and restore uses
	for _, d := range g.dudes {
		d.FullHeal()
		d.RestoreUses()
	}

	for _, st := range g.tower.Stories {
		if st.doorStack != nil {
			s.nextStory = st
			break
		}
	}
	// This shouldn't happen.
	if s.nextStory == nil {
		panic("No next story found!")
	}
	// Generate our new rooms.
	required := GetRequiredRooms(s.nextStory.level, 2)
	optional := GetOptionalRooms(s.nextStory.level, 9) // 6 is minimum, but let's given 3 more for fun.
	s.availableRooms = append(s.availableRooms, required...)
	s.availableRooms = append(s.availableRooms, optional...)
	// Update room panel.
	g.ui.roomPanel.SetRoomDefs(s.availableRooms)
	// Add onClick handler.
	g.ui.roomPanel.onItemClick = func(which int) {
		g.ui.roomInfoPanel.hidden = false
		selected := s.availableRooms[which]
		s.placingRoom = selected
		s.placingIndex = which
		g.ui.roomInfoPanel.title.SetText(fmt.Sprintf("%s %s room", selected.size.String(), selected.kind.String()))
		g.ui.roomInfoPanel.description.SetText(selected.GetDescription())
		g.ui.roomInfoPanel.cost.SetText(fmt.Sprintf("Cost: %d", GetRoomCost(selected.kind, selected.size, s.nextStory.level)))
	}

	// Update info.
	g.UpdateInfo()
}
func (s *GameStateBuild) End(g *Game) {
	g.ui.roomPanel.onItemClick = nil
	g.ui.roomPanel.SetRoomDefs(nil)
}
func (s *GameStateBuild) Update(g *Game) GameState {
	s.wobbler += 0.05
	s.titleTimer++
	/*if s.titleTimer > 120 {
		return &GameStatePlay{}
	}*/

	if handled, kind := g.CheckUI(); !handled {
		if kind == UICheckHover {
			mx, my := IntToFloat2(ebiten.CursorPosition())

			cx := float64(g.lastWidth) / 2
			cy := float64(g.lastHeight) / 2

			r := math.Atan2(my-cy, mx-cx) - g.camera.Rotation()
			roomIndex := s.nextStory.RoomIndexFromAngle(r)

			// Highlight all rooms equal to size of placing.
			for _, room := range s.highlightedRooms {
				room.highlight = false
			}
			if s.placingRoom != nil {
				s.highlightedRooms = nil
				for i := roomIndex; i < roomIndex+int(s.placingRoom.size) && i < 7; i++ {
					s.nextStory.rooms[i].highlight = true
					s.highlightedRooms = append(s.highlightedRooms, s.nextStory.rooms[i])
				}
			}

			// Ensure focusing our actual target root room.
			if s.focusedRoom != nil {
				s.focusedRoom.highlight = false
			}
			s.focusedRoom = s.nextStory.rooms[roomIndex]
			s.focusedRoom.highlight = true

		} else if kind == UICheckClick {
			s.TryPlaceRoom(g)
		}
	} else {
		if s.focusedRoom != nil {
			s.focusedRoom.highlight = false
			s.focusedRoom = nil
		}
		for _, room := range s.highlightedRooms {
			room.highlight = false
		}
		s.highlightedRooms = nil
	}

	return nil
}

func (s *GameStateBuild) Draw(g *Game, screen *ebiten.Image) {
	if s.titleTimer < 240 {
		opts := render.TextOptions{
			Screen: screen,
			Font:   assets.DisplayFont,
			Color:  assets.ColorState,
		}
		opts.GeoM.Scale(4, 4)

		w, h := text.Measure("BUILD", opts.Font.Face, opts.Font.LineHeight)
		w *= 4
		h *= 4

		opts.GeoM.Translate(-w/2, -h/2)
		opts.GeoM.Rotate(math.Sin(s.wobbler) * 0.05)
		opts.GeoM.Translate(w/2, h/2)
		opts.GeoM.Translate(float64(screen.Bounds().Dx()/2)-w/2, float64(screen.Bounds().Dy()/4)-h/2)

		render.DrawText(&opts, "BUILD")
	}
}

func (s *GameStateBuild) TryPlaceRoom(g *Game) {
	if s.focusedRoom != nil {
		if s.placingRoom == nil {
			g.ui.feedback.Msg(FeedbackGeneric, "select a room to place :)")
		} else {
			// If it's not stairs or empty, allow the player to sell it back at full value.
			if s.focusedRoom.kind != Stairs && s.focusedRoom.kind != Empty {
				g.gold += GetRoomCost(s.focusedRoom.kind, s.focusedRoom.size, s.nextStory.level)
				s.availableRooms = append(s.availableRooms, GetRoomDef(s.focusedRoom.kind, s.focusedRoom.size))
				g.ui.roomPanel.SetRoomDefs(s.availableRooms)
				g.UpdateInfo()
				g.ui.feedback.Msg(FeedbackGood, fmt.Sprintf("%s %s sold!", s.focusedRoom.size.String(), s.focusedRoom.kind.String()))
				s.nextStory.RemoveRoom(s.focusedRoom.index)
			} else {
				if g.gold-GetRoomCost(s.placingRoom.kind, s.placingRoom.size, s.nextStory.level) < 0 {
					g.ui.feedback.Msg(FeedbackBad, "ur broke lol")
				} else {
					room := NewRoom(s.placingRoom.size, s.placingRoom.kind)
					if err := s.nextStory.PlaceRoom(room, s.focusedRoom.index); err != nil {
						g.ui.feedback.Msg(FeedbackBad, err.Error())
					} else {
						// it worked!11!
						g.gold -= GetRoomCost(s.placingRoom.kind, s.placingRoom.size, s.nextStory.level)
						g.UpdateInfo()
						g.ui.feedback.Msg(FeedbackGood, fmt.Sprintf("%s %s placed!", s.placingRoom.size.String(), s.placingRoom.kind.String()))
						// Remove from rooms.
						s.availableRooms = append(s.availableRooms[:s.placingIndex], s.availableRooms[s.placingIndex+1:]...)
						// Resync UI, I guess.
						g.ui.roomPanel.SetRoomDefs(s.availableRooms)
						g.ui.roomInfoPanel.hidden = true
						// I'm lazy.
						if s.placingIndex >= len(s.availableRooms) {
							s.placingIndex--
							if s.placingIndex < 0 {
								s.placingIndex = 0
							}
						}
						if s.placingIndex < len(s.availableRooms) {
							g.ui.roomPanel.onItemClick(s.placingIndex)
						}
					}
				}
			}
		}
	}

}

func (s *GameStateBuild) BuyDude(g *Game) {
	// COST?
	cost := 100
	if g.gold < cost {
		AddMessage(MessageError, "Not enough gold to buy a dude.")
		return
	}
	g.gold -= cost
	level := len(g.tower.Stories) / 2
	if level < 1 {
		level = 1
	}

	// Random profession ??
	profession := RandomProfessionKind()
	dude := NewDude(profession, level)
	g.dudes = append(g.dudes, dude)
	g.UpdateInfo()
}

func (s *GameStateBuild) BuyEquipment(g *Game) {
	// COST?
	cost := 50
	if g.gold < cost {
		AddMessage(MessageError, "Not enough gold to buy equipment.")
		return
	}
	g.gold -= cost

	level := len(g.tower.Stories) / 2
	e := GetRandomEquipment(level)
	g.equipment = append(g.equipment, e)
}

func (s *GameStateBuild) SellEquipment(g *Game, e *Equipment) {
	if e == nil {
		return
	}
	g.gold += int(e.GoldValue())
	for i, eq := range g.equipment {
		if eq == e {
			g.equipment = append(g.equipment[:i], g.equipment[i+1:]...)
			break
		}
	}

	// Trigger on sell event
	e.Activate(EventSell{equipment: e})

}
