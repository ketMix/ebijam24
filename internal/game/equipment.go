package game

import (
	"fmt"
	"math"

	"github.com/kettek/ebijam24/assets"
	"github.com/kettek/ebijam24/internal/render"
)

type EquipmentQuality float32

// Number of uses for equipment
const (
	EquipmentQualityCommon    EquipmentQuality = 0
	EquipmentQualityUncommon  EquipmentQuality = 1
	EquipmentQualityRare      EquipmentQuality = 2
	EquipmentQualityEpic      EquipmentQuality = 3
	EquipmentQualityLegendary EquipmentQuality = 4
)

func (eq EquipmentQuality) String() string {
	switch eq {
	case EquipmentQualityCommon:
		return "Common"
	case EquipmentQualityUncommon:
		return "Uncommon"
	case EquipmentQualityRare:
		return "Rare"
	case EquipmentQualityEpic:
		return "Epic"
	case EquipmentQualityLegendary:
		return "Legendary"
	default:
		return "Unknown"
	}
}

func (eq EquipmentQuality) Color() string {
	switch eq {
	case EquipmentQualityUncommon:
		return "green"
	case EquipmentQualityRare:
		return "blue"
	case EquipmentQualityEpic:
		return "purple"
	case EquipmentQualityLegendary:
		return "orange"
	default:
		return "white"
	}
}

type EquipmentType string

// Type of equipment
const (
	EquipmentTypeWeapon    EquipmentType = "weapon"
	EquipmentTypeArmor     EquipmentType = "armor"
	EquipmentTypeAccessory EquipmentType = "accessory"
)

func (et EquipmentType) String() string {
	switch et {
	case EquipmentTypeWeapon:
		return "Weapon"
	case EquipmentTypeArmor:
		return "Armor"
	case EquipmentTypeAccessory:
		return "Accessory"
	default:
		return "Unknown"
	}
}

// An equipment is the stuff you get
type Equipment struct {
	//material   string // Material of the equipment, maybe

	name          string           // Standard name of the equipment ("Bow", "Sword", "Book", "Boots")
	level         int              // Level of the equipment affects stats, maybe this could be material too?
	quality       EquipmentQuality // Quality of the equipment, dictates total uses
	uses          int              // Current uses of the equipment (number of times the perk can be triggered)
	totalUses     int              // Total uses of the equipment
	description   string           // Description of the equipment
	equipmentType EquipmentType    // Type of equipment (weapon, armor, etc.)

	perk        Perk              // Perk of the equipment (if any)
	stats       *Stats            // Stats of the equipment (if any)
	stack       *render.Stack     // How to draw the equipment
	professions []*ProfessionKind // If restricted to a profession
	Draw        func(*render.Options)
}

// Fetches the equipment by name
// Used for creating equipment in the game.
// Should find the equipment by name from loaded equipment
func NewEquipment(name string, level int, quality EquipmentQuality, perk Perk) *Equipment {
	baseEquipment, err := assets.LoadEquipment(name)
	if err != nil {
		fmt.Println("Error loading equipment: ", err)
		return nil
	}

	// Parse equipment asset to equipment
	stack, err := render.NewStack(baseEquipment.StackPath, "", "")
	if err != nil {
		fmt.Println("Error loading equipment stack: ", err)
		stack = nil
	}
	professions := make([]*ProfessionKind, len(baseEquipment.Professions))

	// Convert the professions to ProfessionKind
	for i, p := range baseEquipment.Professions {
		professions[i] = new(ProfessionKind)
		*professions[i] = ProfessionKind(p)
	}

	// If base equipment has perk, load it
	// hmmm...
	if perk == nil {
		switch baseEquipment.Perk {
		case "Heal On Room Enter":
			perk = PerkHealOnRoomEnter{PerkQualityCommon}
		}
	}

	equipment := &Equipment{
		name:          baseEquipment.Name,
		level:         level,
		quality:       quality,
		uses:          int(quality) + 1,
		totalUses:     int(quality) + 1,
		description:   baseEquipment.Description,
		equipmentType: EquipmentType(baseEquipment.Type),
		professions:   professions,
		perk:          perk,
		stack:         stack,
		stats: &Stats{
			totalHp:   baseEquipment.Stats["totalHp"],
			strength:  baseEquipment.Stats["strength"],
			wisdom:    baseEquipment.Stats["wisdom"],
			defense:   baseEquipment.Stats["defense"],
			agility:   baseEquipment.Stats["agility"],
			cowardice: baseEquipment.Stats["cowardice"],
			luck:      baseEquipment.Stats["luck"],
		},
	}

	equipment.Draw = func(o *render.Options) {
		if equipment.stack != nil {
			equipment.stack.Draw(o)
		}
	}
	return equipment
}

func (e *Equipment) Update() {
	if e.stack == nil {
		return
	}
	e.stack.Update()
}

// Name returns the name of the equipment.
func (e *Equipment) Name() string {
	return fmt.Sprintf("%s (%s)", e.name, e.quality)
}

// Level returns the level of the equipment.
func (e *Equipment) Level() int {
	return e.level
}

func (e *Equipment) ChangeQuality(delta int) {
	if delta == 0 {
		return
	}

	// If we're trying to decrease the quality of a common item, don't
	if e.quality == EquipmentQualityCommon && delta < 0 {
		// We can subtract a use though
		if e.uses > 0 {
			e.uses--
		}
		return
	}

	// If we're trying to increase the quality of a legendary item, don't
	if e.quality == EquipmentQualityLegendary && delta > 0 {
		// We can add a use though
		if e.uses < e.totalUses {
			e.uses++
		}
		return
	}

	// Do this so we can ceil and floor changes and hit those thresholds
	if delta > 1 {
		for i := 0; i < delta; i++ {
			e.ChangeQuality(1)
		}
	} else if delta < -1 {
		for i := 0; i > delta; i-- {
			e.ChangeQuality(-1)
		}
	} else {
		// Here we are always 1 or -1 delta
		e.quality += EquipmentQuality(1 * delta)
		e.totalUses += delta
		e.uses += delta
		if e.uses > e.totalUses {
			e.uses = e.totalUses
		}
		if e.uses < 0 {
			e.uses = 0
		}
	}
}

// Levels up the weapon.
// If it hits 5 we can upgrade the quality
func (e *Equipment) LevelUp() {
	// If we have stats we can level this item up
	if e.stats == nil {
		return
	}

	e.level++

	// If we hit level 5 we can upgrade the quality
	if e.level == 5 && e.quality < EquipmentQualityLegendary {
		e.ChangeQuality(1)
		e.level = 0
	}
}

// Levels down weapon
func (e *Equipment) LevelDown() {
	if e.stats == nil {
		return
	}

	e.level--

	// If we hit level 0 downgrade the quality if possible
	if e.level < 0 && e.quality > EquipmentQualityCommon {
		e.ChangeQuality(-1)
		e.level = 4
	}
}

// Quality returns the quality of the equipment.
func (e *Equipment) Quality() EquipmentQuality {
	return e.quality
}

// Uses returns the uses of the equipment.
func (e *Equipment) Uses() int {
	return e.uses
}

// Description returns the description of the equipment.
func (e *Equipment) Description() string {
	return e.description
}

// Probably not necessary?
// // Perk returns the perk of the equipment.
// func (e *Equipment) Perk() Perk {
// 	return e.perk
// }

// Activate the equipment's perk and decrement the uses.
func (e *Equipment) Activate(event Event) {
	if e.perk == nil || e.uses == 0 {
		return
	}

	activated := e.perk.Check(event)
	if !activated {
		return
	}

	// Successfully activated the perk, decrement the uses
	e.uses--
	if e.uses < 0 {
		// Get ye gone!
		e.uses = 0
	}
}

// An equipment's stats will be combined with the dude's stats
// after being scaled by the equipment's quality and level
func (e *Equipment) Stats() *Stats {
	if e.stats == nil {
		return &Stats{}
	}

	// Scale the stats by the equipment's quality and level
	// Quality increases the stats by 15% per quality level
	// While level increases the base stats by 2% per level
	// This way when you upgrade quality, the stats continually increase
	qualityMultiplier := (float64(e.quality) * 0.15) + 1
	levelMultiplier := (float64(e.level) * 0.02) + 1
	m := qualityMultiplier * levelMultiplier

	applyMultiplier := func(s int) int {
		return int(math.Floor(float64(s) * m))
	}
	scaledStats := &Stats{
		totalHp:   applyMultiplier(e.stats.totalHp),
		strength:  applyMultiplier(e.stats.strength),
		wisdom:    applyMultiplier(e.stats.wisdom),
		defense:   applyMultiplier(e.stats.defense),
		agility:   applyMultiplier(e.stats.agility),
		cowardice: applyMultiplier(e.stats.cowardice),
	}
	return scaledStats
}

func (e *Equipment) CanEquip(p ProfessionKind) bool {
	fmt.Println("Checking if we can equip", e.name, "for profession", p, "from professions", e.professions)
	if e.professions == nil {
		return true
	}

	// If the profession is in the list of professions that can equip this item
	// then we can equip it
	for _, prof := range e.professions {
		if *prof == p {
			return true
		}
	}

	return false
}

func (e *Equipment) Type() EquipmentType {
	return e.equipmentType
}

func (e *Equipment) GoldValue() float32 {
	return float32(e.level * (1 + int(e.quality)))
}
