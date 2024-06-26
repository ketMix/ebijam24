package game

import (
	"math/rand"
)

// ProfessionKind is an enumeration of the different kinds of Professions a dude can have
type ProfessionKind string

func (p *ProfessionKind) String() string {
	switch *p {
	case Vagabond:
		return "Vagabond"
	case Knight:
		return "Knight"
	case Cleric:
		return "Cleric"
	case Ranger:
		return "Ranger"
	default:
		return "Unknown"
	}
}

const (
	// Medium defense, medium attack, medium hp
	Vagabond ProfessionKind = "vagabond"

	// High defense, low attack, high hp
	Knight ProfessionKind = "knight"

	// Low defense, low attack, *can heal*
	Cleric ProfessionKind = "cleric"

	// Medium defense, high attack, low hp *ranged*
	Ranger ProfessionKind = "ranger"
)

// A profession defines a dude's abilities.
// It also defines the dude's appearance.
type Profession struct {
	kind              ProfessionKind
	description       string
	startingStats     Stats
	startingEquipment []*Equipment
}

func RandomProfessionKind() ProfessionKind {
	professions := []ProfessionKind{Vagabond, Knight, Cleric, Ranger}
	return professions[rand.Intn(len(professions))]
}

// WeightedRandomProfessionKind returns a profession kind based on the dudes' professions
// The lower the frequency of a profession, the higher the weight
func WeightedRandomProfessionKind(dudes []*Dude) ProfessionKind {
	professionCount := make(map[ProfessionKind]int)

	// Count the number of each profession
	for _, d := range dudes {
		professionCount[d.profession]++
	}

	// Calculate weights (lower frequency = higher weight)
	professions := []ProfessionKind{Vagabond, Knight, Cleric, Ranger}
	// Calculate weights (lower frequency = much higher weight)
	weights := make([]float64, len(professions))
	for i, profession := range professions {
		count := professionCount[profession]
		if count == 0 {
			weights[i] = 1.0
		} else {
			weights[i] = 1.0 / float64(count*count)
		}
	}

	// Create a cumulative weight array
	cumulativeWeights := make([]float64, len(professions))
	cumulativeWeights[0] = weights[0]
	for i := 1; i < len(professions); i++ {
		cumulativeWeights[i] = cumulativeWeights[i-1] + weights[i]
	}

	// Select a profession based on the weights
	randomValue := rand.Float64() * cumulativeWeights[len(cumulativeWeights)-1]
	for i, cumulativeWeight := range cumulativeWeights {
		if randomValue < cumulativeWeight {
			return professions[i]
		}
	}
	return professions[0] // Fallback, should not reach here
}

func NewProfession(kind ProfessionKind, level int) *Profession {
	switch kind {
	case Knight:
		return &Profession{
			kind:          Knight,
			description:   "A knight in shining armor",
			startingStats: *getStartingStats(Knight, 1),
			startingEquipment: []*Equipment{
				NewEquipment("Plate", 1, EquipmentQualityCommon, nil),
				NewEquipment("Sword", 1, EquipmentQualityCommon, nil),
				NewEquipment("Shield", 1, EquipmentQualityCommon, nil),
			},
		}
	case Cleric:
		return &Profession{
			kind:          Cleric,
			description:   "A cleric who can heal",
			startingStats: *getStartingStats(Cleric, 1),
			startingEquipment: []*Equipment{
				NewEquipment("Staff", 1, EquipmentQualityCommon, nil),
				NewEquipment("Robe", 1, EquipmentQualityCommon, nil),
			},
		}
	case Vagabond:
		return &Profession{
			kind:          Vagabond,
			description:   "A vagabond with no home",
			startingStats: *getStartingStats(Vagabond, 1),
			startingEquipment: []*Equipment{
				NewEquipment("Dagger", 1, EquipmentQualityCommon, nil),
				NewEquipment("Leather", 1, EquipmentQualityCommon, nil),
			},
		}
	case Ranger:
		return &Profession{
			kind:          Ranger,
			description:   "A ranger who can shoot from afar",
			startingStats: *getStartingStats(Ranger, 1),
			startingEquipment: []*Equipment{
				NewEquipment("Bow", 1, EquipmentQualityCommon, nil),
				NewEquipment("Leather", 1, EquipmentQualityCommon, nil),
			},
		}
	}
	return nil
}

func (p *Profession) String() string {
	return p.kind.String()
}
func (p *Profession) Description() string {
	return p.description
}
func (p *Profession) StartingStats() Stats {
	return p.startingStats
}
func (p *Profession) StartingEquipment() []*Equipment {
	return p.startingEquipment
}

// Professions are created using their level change modifiers to stats and a given level
// Then they level up and apply the changes
func getStartingStats(kind ProfessionKind, level int) *Stats {
	switch kind {
	case Knight:
		return NewStats(&Stats{
			level:      level,
			totalHp:    7,
			strength:   2,
			wisdom:     1,
			defense:    3,
			agility:    1,
			confidence: 5, // balls get bigger
			luck:       0,
		}, false)
	case Cleric:
		return NewStats(&Stats{
			level:      level,
			totalHp:    5,
			strength:   1,
			wisdom:     3,
			defense:    1,
			agility:    2,
			confidence: 1, // balls get smaller
			luck:       0,
		}, false)
	case Vagabond:
		return NewStats(&Stats{
			level:      level,
			totalHp:    7,
			strength:   3,
			wisdom:     1,
			defense:    2,
			agility:    1,
			confidence: 3,
			luck:       0,
		}, false)
	case Ranger:
		return NewStats(&Stats{
			level:      1,
			totalHp:    5,
			strength:   2,
			wisdom:     1,
			defense:    1,
			agility:    3,
			confidence: 0,
			luck:       0,
		}, false)
	default:
		// you useless jobless bum
		return NewStats(nil, false)

	}
}
