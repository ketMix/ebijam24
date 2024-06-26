package render

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kettek/ebijam24/assets"
)

type Stack struct {
	Positionable
	Rotateable
	Originable
	data             *assets.Staxie // Reference to the underlying stack data for subimages, etc.
	currentStack     *assets.StaxieStack
	currentAnimation *assets.StaxieAnimation
	currentFrame     *assets.StaxieFrame
	frameCounter     int
	MaxSliceIndex    int
	VgroupOffset     int
	SliceOffset      int
	SliceColorOffset int
	SliceColorMin    float64
	HeightOffset     float64
	NoLighting       bool
	Transparency     float32
	ColorScale       ebiten.ColorScale
}

func NewStack(name string, stackName string, animationName string) (*Stack, error) {
	staxie, err := assets.LoadStaxie(name)
	if err != nil {
		return nil, err
	}
	if stackName == "" {
		for k := range staxie.Stacks {
			stackName = k
			break
		}
	}

	stack, ok := staxie.Stacks[stackName]
	if !ok {
		return nil, fmt.Errorf("stack %s does not exist in %s", stackName, name)
	}

	if animationName == "" {
		for k := range stack.Animations {
			animationName = k
			break
		}
	}
	animation, ok := stack.Animations[animationName]
	if !ok {
		return nil, fmt.Errorf("animation %s does not exist in %s", animationName, stackName)
	}

	frame, ok := animation.GetFrame(0)
	if !ok {
		return nil, fmt.Errorf("frame 0 does not exist in %s", animationName)
	}

	return &Stack{data: staxie, currentStack: stack, currentAnimation: &animation, currentFrame: frame, SliceColorMin: 0.5}, nil
}

func CopyStack(stack *Stack) *Stack {
	return &Stack{
		Positionable:     stack.Positionable,
		Rotateable:       stack.Rotateable,
		Originable:       stack.Originable,
		data:             stack.data,
		currentStack:     stack.currentStack,
		currentAnimation: stack.currentAnimation,
		currentFrame:     stack.currentFrame,
		frameCounter:     stack.frameCounter,
		MaxSliceIndex:    stack.MaxSliceIndex,
		VgroupOffset:     stack.VgroupOffset,
		SliceOffset:      stack.SliceOffset,
		HeightOffset:     stack.HeightOffset,
		SliceColorMin:    0.5,
	}
}

func (s *Stack) Draw(o *Options) {
	if s.currentFrame == nil {
		return
	}

	opts := ebiten.DrawImageOptions{}

	// Rotate about origin.
	ox, oy := s.Origin()
	opts.GeoM.Translate(-ox, -oy)
	opts.GeoM.Rotate(s.Rotation())
	opts.GeoM.Translate(ox, oy)

	// Translate to position.
	opts.GeoM.Translate(s.Position())

	// Add additional transforms.
	opts.GeoM.Concat(o.DrawImageOptions.GeoM)

	opts.GeoM.Translate(0, s.HeightOffset)

	oldGeoM := ebiten.GeoM{}
	oldGeoM.Concat(opts.GeoM)
	for index := 0; index < len(s.currentFrame.Slices); index++ {
		if index+s.SliceOffset >= len(s.currentFrame.Slices) {
			break
		}
		if s.MaxSliceIndex != 0 && index >= s.MaxSliceIndex {
			break
		}
		slice := s.currentFrame.Slices[index+s.SliceOffset]
		i := index

		// TODO: Make this configurable
		c := float64(index) / float64(len(s.currentFrame.Slices)+s.SliceColorOffset)
		c = math.Min(1.0, math.Max(s.SliceColorMin, c))
		color := float32(c)
		// TODO: Add color offsets...
		if s.NoLighting {
			color = 1.0
		}

		opts.ColorScale.Reset()
		opts.ColorScale.Scale(color, color, color, 1.0)
		opts.ColorScale.ScaleWithColorScale(o.DrawImageOptions.ColorScale)

		opts.ColorScale.ScaleWithColorScale(s.ColorScale)

		if s.Transparency != 0 {
			opts.ColorScale.ScaleAlpha(1.0 - s.Transparency)
		}

		if o.VGroup != nil {
			i += s.VgroupOffset

			opts.GeoM.Reset()
			opts.GeoM.Concat(oldGeoM)
			opts.GeoM.Translate(0, float64(i*o.VGroup.Height))
			o.VGroup.Images[0].DrawImage(slice.Image, &opts)
		} else if o.Screen != nil {
			o.Screen.DrawImage(slice.Image, &opts)
			opts.GeoM.Translate(0, -o.Pitch)
		}
		//opts.GeoM.Skew(-0.002, 0.002) // Might be able to sine this with delta to create a wave effect...
	}
}

func (s *Stack) Update() {
	s.frameCounter++
	if s.frameCounter >= int(s.currentAnimation.Frametime) {
		s.frameCounter = 0
		nextFrame, ok := s.currentAnimation.GetFrame(s.currentFrame.Index + 1)
		if !ok {
			nextFrame, _ = s.currentAnimation.GetFrame(0)
		}
		s.currentFrame = nextFrame
	}
}

func (s *Stack) SliceCount() int {
	return len(s.currentFrame.Slices)
}

func (s *Stack) Width() int {
	return s.currentFrame.Slices[0].Image.Bounds().Dx()
}

func (s *Stack) Height() int {
	return s.currentFrame.Slices[0].Image.Bounds().Dy()
}

func (s *Stack) SetStaxie(name string) error {
	staxie, err := assets.LoadStaxie(name)
	if err != nil {
		return err
	}
	s.data = staxie
	return nil
}

func (s *Stack) SetStack(name string) error {
	stack, ok := s.data.Stacks[name]
	if !ok {
		return fmt.Errorf("stack %s", name)
	}
	s.currentStack = stack

	return s.SetAnimation(s.currentAnimation.Name)
}

func (s *Stack) Stacks() []string {
	return s.data.StackNames
}

func (s *Stack) SetAnimation(name string) error {
	animation, ok := s.currentStack.GetAnimation(name)
	if !ok {
		return fmt.Errorf("animation %s", name)
	}
	s.currentAnimation = &animation

	return s.SetFrame(0)
}

func (s *Stack) SetFrame(index int) error {
	frame, ok := s.currentAnimation.GetFrame(index)
	if !ok {
		return fmt.Errorf("frame %d", index)
	}
	s.currentFrame = frame
	return nil
}

func (s *Stack) SetOriginToCenter() {
	s.SetOrigin(float64(s.currentFrame.Slices[0].Image.Bounds().Dx())/2, float64(s.currentFrame.Slices[0].Image.Bounds().Dy())/2)
}
