package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kettek/ebijam24/internal/render"
)

type UIOptions struct {
	Scale  float64
	Width  int
	Height int
}

type UI struct {
	dudePanel DudePanel
	roomPanel RoomPanel
	options   *UIOptions
}

func NewUI(dudeList []*Dude) *UI {
	ui := &UI{}

	{
		panelSprite := Must(render.NewSprite("ui/panels"))
		ui.dudePanel = DudePanel{
			top:          Must(render.NewSubSprite(panelSprite, 0, 0, 16, 16)),
			topright:     Must(render.NewSubSprite(panelSprite, 16, 0, 16, 16)),
			mid:          Must(render.NewSubSprite(panelSprite, 0, 16, 16, 16)),
			midright:     Must(render.NewSubSprite(panelSprite, 16, 16, 16, 16)),
			bot:          Must(render.NewSubSprite(panelSprite, 0, 32, 16, 16)),
			botright:     Must(render.NewSubSprite(panelSprite, 16, 32, 16, 16)),
			dudeProfiles: profilesFromDudes(dudeList),
		}
	}
	{
		panelSprite := Must(render.NewSprite("ui/botPanel"))
		ui.roomPanel = RoomPanel{
			topleft:  Must(render.NewSubSprite(panelSprite, 0, 0, 16, 32)),
			left:     Must(render.NewSubSprite(panelSprite, 0, 16, 16, 32)),
			topmid:   Must(render.NewSubSprite(panelSprite, 16, 0, 16, 32)),
			mid:      Must(render.NewSubSprite(panelSprite, 16, 16, 16, 32)),
			topright: Must(render.NewSubSprite(panelSprite, 32, 0, 16, 32)),
			right:    Must(render.NewSubSprite(panelSprite, 32, 16, 16, 32)),
		}
	}
	return ui
}

func (ui *UI) Layout(o *UIOptions) {
	ui.options = o
	ui.dudePanel.Layout(o)
	ui.roomPanel.Layout(o)
}

func (ui *UI) Update(o *UIOptions) {
	ui.dudePanel.Update(o)
	ui.roomPanel.Update(o)
}

func (ui *UI) Draw(o *render.Options) {
	o.DrawImageOptions.GeoM.Scale(ui.options.Scale, ui.options.Scale)
	ui.dudePanel.Draw(o)
	o.DrawImageOptions.GeoM.Reset()
	o.DrawImageOptions.GeoM.Scale(ui.options.Scale, ui.options.Scale)
	ui.roomPanel.Draw(o)
}

type DudePanel struct {
	render.Originable
	render.Positionable
	drawered     bool
	height       int
	top          *render.Sprite
	topright     *render.Sprite
	mid          *render.Sprite
	midright     *render.Sprite
	bot          *render.Sprite
	botright     *render.Sprite
	drawerInterp InterpNumber
	dudeProfiles []*DudeProfile
}

type DudeProfile struct {
	render.Positionable
	dude    *Dude
	hovered bool
}

func profilesFromDudes(dudes []*Dude) []*DudeProfile {
	profiles := []*DudeProfile{}
	for _, dude := range dudes {
		profiles = append(profiles, &DudeProfile{dude: dude})
	}
	return profiles
}

// ...
func (dp *DudeProfile) InBounds(x, y, dpy float64) bool {
	px, py := dp.Position()
	px += 11
	py += dpy + 40
	profileWidth := 32

	fmt.Println(x, y, dpy, px, py, px+float64(profileWidth), py+float64(profileWidth))
	if x > px && x < px+float64(profileWidth) && y > py && y < py+float64(profileWidth) {
		return true
	}
	return false
}

func (dp *DudePanel) Layout(o *UIOptions) {
	dp.height = o.Height - o.Height/3

	// Position at vertical center.
	dp.SetPosition(0, float64(o.Height/2)-float64(dp.height)/2-32)

	// Position dude faces
	yOffset := 24
	dpx, dpy := dp.Position()
	for i, p := range dp.dudeProfiles {
		p.SetPosition(dpx, float64(i*yOffset)-(float64(dpy/2)-24))
	}
}

func (dp *DudePanel) Update(o *UIOptions) {
	dp.drawerInterp.Update()

	dpx, dpy := dp.Position()
	mx, my := IntToFloat2(ebiten.CursorPosition())

	maxX := (dpx + 32) * o.Scale
	maxY := (dpy + float64(dp.height)) * o.Scale

	if mx > dpx && mx < maxX && my > dpy && my < maxY {
		if dp.drawered {
			dp.drawered = false
			dp.drawerInterp.Set(0, 3)
		}
	} else {
		if !dp.drawered {
			dp.drawered = true
			dp.drawerInterp.Set(-48, 3)
		}
	}

	if !dp.drawered {
		// TODO: Convert mouse pos to dude clicking?
		// Testing with single dude
		p := dp.dudeProfiles[0]
		if p.InBounds(mx, my, dpy) {
			fmt.Println("hovered over my guy: ", p.dude.name)
			p.hovered = true
		} else {
			p.hovered = false
		}
	}
}

func (dp *DudePanel) Draw(o *render.Options) {
	o.DrawImageOptions.GeoM.Translate(dp.drawerInterp.Current, 0)
	y := 0
	o.DrawImageOptions.GeoM.Translate(dp.Position())
	// top
	dp.top.Draw(o)
	o.DrawImageOptions.GeoM.Translate(16, 0)
	dp.topright.Draw(o)
	o.DrawImageOptions.GeoM.Translate(0, 16)
	y += 16
	o.DrawImageOptions.GeoM.Translate(-16, 0)

	// Save these top options for drawing dude profiles
	topOptions := render.Options{
		Screen:           o.Screen,
		DrawImageOptions: o.DrawImageOptions,
	}

	// mid
	for ; y < dp.height-16; y += 16 {
		dp.mid.Draw(o)
		o.DrawImageOptions.GeoM.Translate(16, 0)
		dp.midright.Draw(o)
		o.DrawImageOptions.GeoM.Translate(-16, 0)
		o.DrawImageOptions.GeoM.Translate(0, 16)
	}
	// bottom
	dp.bot.Draw(o)
	o.DrawImageOptions.GeoM.Translate(16, 0)
	dp.botright.Draw(o)

	// Draw dudes below top
	for _, p := range dp.dudeProfiles {
		// Save these top options for drawing dude profiles
		profileOptions := render.Options{
			Screen:           o.Screen,
			DrawImageOptions: topOptions.DrawImageOptions,
		}
		profileOptions.DrawImageOptions.GeoM.Translate(p.Position())
		profileOptions.DrawImageOptions.GeoM.Scale(2, 2)
		p.dude.DrawProfile(&profileOptions)
	}
}

type RoomPanel struct {
	render.Originable
	render.Positionable
	drawered     bool
	drawerInterp InterpNumber
	width        int
	left         *render.Sprite
	topleft      *render.Sprite
	mid          *render.Sprite
	topmid       *render.Sprite
	right        *render.Sprite
	topright     *render.Sprite
}

func (rp *RoomPanel) Layout(o *UIOptions) {
	rp.width = o.Width - o.Width/3
	rp.SetPosition(float64(o.Width/2)-float64(rp.width)/2, float64(o.Height)-96)
}

func (rp *RoomPanel) Update(o *UIOptions) {
	rp.drawerInterp.Update()

	rpx, rpy := rp.Position()
	mx, my := IntToFloat2(ebiten.CursorPosition())

	maxX := (rpx + float64(rp.width)) * o.Scale
	maxY := (rpy) * o.Scale

	if mx > rpx && mx < maxX && my > rpy && my < maxY {
		if rp.drawered {
			rp.drawered = false
			rp.drawerInterp.Set(0, 3.5)
		}
	} else {
		if !rp.drawered {
			rp.drawered = true
			rp.drawerInterp.Set(64, 3.5)
		}
	}
}

func (rp *RoomPanel) Draw(o *render.Options) {
	o.DrawImageOptions.GeoM.Translate(0, rp.drawerInterp.Current)
	x := 0
	o.DrawImageOptions.GeoM.Translate(rp.Position())
	// topleft
	rp.topleft.Draw(o)
	o.DrawImageOptions.GeoM.Translate(0, 32)
	// left
	rp.left.Draw(o)
	o.DrawImageOptions.GeoM.Translate(32, -32)
	// mid
	x += 32
	for ; x < rp.width-32; x += 32 {
		rp.topmid.Draw(o)
		o.DrawImageOptions.GeoM.Translate(0, 32)
		rp.mid.Draw(o)
		o.DrawImageOptions.GeoM.Translate(0, -32)
		o.DrawImageOptions.GeoM.Translate(32, 0)
	}
	// topright
	rp.topright.Draw(o)
	o.DrawImageOptions.GeoM.Translate(0, 32)
	// right
	rp.right.Draw(o)
}
