package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/kettek/ebijam24/assets"
	"github.com/kettek/ebijam24/internal/render"
)

type DudeActivity int

const (
	Idle          DudeActivity = iota
	FirstEntering              // First entering the tower.
	StairsToUp
	StairsFromDown
	GoingUp   // Entering the room from a staircase, this basically does the fancy slice offset/limiting.
	Centering // Move the dude to the center of the room.
	Moving    // Move the dude counter-clockwise.
	Leaving   // Move the dude to the stairs.
	GoingDown // Leaving the room to the stairs, opposite of GoingUp.
	EnterPortal
	Ded
	BornAgain
	Waiting // maybe some random shufflin about
	FightBoss
)

type Dude struct {
	name         string
	invincible   bool
	xp           int
	gold         int
	profession   ProfessionKind
	stats        Stats
	equipped     map[EquipmentType]*Equipment
	inventory    []*Equipment
	story        *Story // current story da dude be in
	room         *Room  // current room the dude is in
	stack        *render.Stack
	shadow       *render.Stack
	timer        int
	activity     DudeActivity
	activityDone bool
	variation    float64
	enemy        *Enemy  // currently fighting enemy
	trueRotation float64 // This is the absolute rotation of the dude, ignoring facing.
	// for updating dude infos
	dirtyEquipment bool
	dirtyStats     bool
}

func NewDude(pk ProfessionKind, level int) *Dude {
	dude := &Dude{}

	stack, err := render.NewStack("dudes/liltest", "", "")
	if err != nil {
		panic(err)
	}

	// Randomize which dude it be.
	stack.SetStack(stack.Stacks()[rand.Intn(len(stack.Stacks()))])
	stack.SetAnimation("base")

	// Get shadow.
	shadowStack, err := render.NewStack("dudes/shadow", "", "")
	if err != nil {
		panic(err)
	}
	dude.shadow = shadowStack

	// Assign a random dude skin
	stackNames := stack.Stacks()
	stack.SetStack(stackNames[rand.Intn(len(stackNames))])
	stack.SetOriginToCenter()

	dude.name = assets.GetRandomName()
	dude.xp = 0
	dude.gold = 0
	dude.profession = pk

	// Initialize stats and equipment
	profession := NewProfession(pk, level)
	dude.stats = profession.StartingStats()

	for i := 0; i < level-1; i++ {
		dude.stats.LevelUp(false)
	}

	dude.inventory = make([]*Equipment, 0)

	dude.equipped = make(map[EquipmentType]*Equipment)
	for _, eq := range profession.StartingEquipment() {
		dude.Equip(eq)
	}

	dude.variation = -6 + rand.Float64()*12

	dude.stack = stack
	dude.stack.VgroupOffset = 1

	return dude
}

func (d *Dude) SetActivity(a DudeActivity) {
	d.activity = a
	d.timer = 0
	if a == Ded {
		d.stack.SetAnimation("ded")
	} else if a == BornAgain {
		d.stack.SetAnimation("base")
	}
}

func (d *Dude) Update(story *Story, req *ActivityRequests) {
	// NOTE: We should replace Centering/Moving direct position/rotation setting with a "pathing node" that the dude seeks to follow. This would allow more smoothly doing turns and such, as we could have a turn limit the dude would follow automatically...
	switch d.activity {
	case Idle:
		// Do nothing.
	case Ded:
		// Also do nothing!
	case BornAgain:
		d.SetActivity(Moving) // Is this safe to just set moving?
	case FirstEntering:
		cx, cy := d.Position()
		distance := story.DistanceFromCenter(cx, cy)
		if distance < 50+d.variation {
			d.SetActivity(Centering)
			d.stack.HeightOffset = 0
		} else {
			r := story.AngleFromCenter(cx, cy) + d.variation/5000
			nx, ny := story.PositionFromCenter(r, distance-d.Speed()*100)

			face := math.Atan2(ny-cy, nx-cx)
			d.trueRotation = face

			req.Add(MoveActivity{dude: d, face: face, x: nx, y: ny, cb: func(success bool) {
				d.stack.HeightOffset -= 0.15
				if d.stack.HeightOffset <= 0 {
					d.stack.HeightOffset = 0
				}
				d.SyncEquipment()
			}})
		}
	case StairsToUp:
		d.timer++

		if d.timer < 40 {
			cx, cy := d.Position()
			r := story.AngleFromCenter(cx, cy) + d.variation/5000
			nx, ny := story.PositionFromCenter(r-0.005, RoomPath+d.variation)

			d.stack.VgroupOffset = d.timer / 2

			face := math.Atan2(ny-cy, nx-cx)
			d.trueRotation = face

			req.Add(MoveActivity{dude: d, face: face, x: nx, y: ny, cb: func(success bool) {
				if success {
					d.SyncEquipment()
				}
			}})
		} else {
			req.Add(StoryEnterNextActivity{dude: d, cb: func(success bool) {
				if success {
					d.SyncEquipment()
				}
			}})
			d.stack.VgroupOffset = 0
			d.SetActivity(StairsFromDown)
		}
	case StairsFromDown:
		d.timer++
		if d.stack.SliceOffset == 0 {
			d.stack.SliceOffset = d.stack.SliceCount()
			d.stack.MaxSliceIndex = 1
		}
		cx, cy := d.Position()
		r := story.AngleFromCenter(cx, cy) + d.variation/5000
		nx, ny := story.PositionFromCenter(r-0.01, RoomPath+d.variation)

		face := math.Atan2(ny-cy, nx-cx)
		d.trueRotation = face

		req.Add(MoveActivity{dude: d, face: face, x: nx, y: ny, cb: func(success bool) {
			d.SyncEquipment()
		}})
		if d.timer >= 2 {
			d.stack.SliceOffset--
			d.stack.MaxSliceIndex++
			d.timer = 0
		}
		if d.stack.SliceOffset <= 0 {
			d.stack.SliceOffset = 0
			d.stack.MaxSliceIndex = 0
			d.SetActivity(Moving)
		}
	case GoingUp:
		d.timer++
		if d.stack.SliceOffset == 0 {
			d.stack.SliceOffset = d.stack.SliceCount()
			d.stack.MaxSliceIndex = 1
			cx, cy := d.Position()
			distance := story.DistanceFromCenter(cx, cy)
			r := story.AngleFromCenter(cx, cy)
			nx, ny := story.PositionFromCenter(r, distance+d.Speed()*100)

			face := math.Atan2(ny-cy, nx-cx)
			d.trueRotation = face

			req.Add(MoveActivity{dude: d, face: face, x: nx, y: ny, cb: func(success bool) {
				d.SyncEquipment()
			}})
		}
		if d.timer >= 15 {
			d.stack.SliceOffset--
			d.stack.MaxSliceIndex++
			d.timer = 0
		}
		if d.stack.SliceOffset <= 0 {
			d.stack.SliceOffset = 0
			d.stack.MaxSliceIndex = 0
			d.SetActivity(Centering)
		}
	case Centering:
		cx, cy := d.Position()
		distance := story.DistanceFromCenter(cx, cy)
		if distance >= RoomPath+d.variation {
			d.SetActivity(Moving)
		} else {
			r := story.AngleFromCenter(cx, cy) + d.variation/5000
			nx, ny := story.PositionFromCenter(r, distance+d.Speed()*100)

			face := math.Atan2(ny-cy, nx-cx)
			d.trueRotation = face

			req.Add(MoveActivity{dude: d, face: face, x: nx, y: ny, cb: func(success bool) {
				if success {
					d.SyncEquipment()
				}
			}})
		}
	case Moving:
		cx, cy := d.Position()
		r := story.AngleFromCenter(cx, cy) + d.variation/5000
		nx, ny := story.PositionFromCenter(r-d.Speed(), RoomPath+d.variation)

		face := math.Atan2(ny-cy, nx-cx)
		d.trueRotation = face

		// Face inwards if we have an enemy!
		if d.enemy != nil {
			fx, fy := story.PositionFromCenter(r-d.Speed(), d.variation)
			face = math.Atan2(fy-cy, fx-cx)
		}

		req.Add(MoveActivity{dude: d, face: face, x: nx, y: ny, cb: func(success bool) {
			if success {
				d.SyncEquipment()
			}
		}})
	case Leaving:
		// TODO
	case GoingDown:
		// TODO
	case EnterPortal:
		d.timer++
		// Wait a little bit before entering!
		if d.timer >= 30 {
			cx, cy := d.Position()
			distance := story.DistanceFromCenter(cx, cy)

			d.stack.Transparency = float32(d.timer-30) / 20
			d.shadow.Transparency = float32(d.timer-30) / 20

			if distance < PortalDistance-4+d.variation {
				d.stack.Transparency = 1
				d.shadow.Transparency = 1
				d.SetActivity(Idle)
				if !d.IsDead() {
					req.Add(TowerLeaveActivity{dude: d})
				}
			} else {
				r := story.AngleFromCenter(cx, cy)
				nx, ny := story.PositionFromCenter(r, distance-0.005*100)

				face := math.Atan2(ny-cy, nx-cx)
				d.trueRotation = face

				req.Add(MoveActivity{dude: d, face: face, x: nx, y: ny, cb: func(success bool) {
					d.SyncEquipment()
				}})
			}
		}

	}

	d.stack.Update()

	// Update equipment
	for _, eq := range d.equipped {
		if eq != nil {
			eq.Update()
		}
	}

	// Update enemy if there is one
	if d.enemy != nil {
		d.enemy.Update(d)
	}
}

func (d *Dude) SyncEquipment() {
	// Piggy-back syncing shadow here
	d.shadow.SetOrigin(d.stack.Origin())
	d.shadow.SetPosition(d.stack.Position())
	d.shadow.SetRotation(d.stack.Rotation())
	for _, eq := range d.equipped {
		if eq != nil && eq.stack != nil {
			// Set equipment position to dude position
			eq.stack.SliceOffset = d.stack.SliceOffset
			eq.stack.MaxSliceIndex = d.stack.MaxSliceIndex
			eq.stack.HeightOffset = d.stack.HeightOffset
			eq.stack.VgroupOffset = d.stack.VgroupOffset
			eq.stack.Transparency = d.stack.Transparency
			eq.stack.SetOrigin(d.stack.Origin())
			eq.stack.SetPosition(d.stack.Position())
			eq.stack.SetRotation(d.stack.Rotation())
		}
	}
}

func (d *Dude) Draw(o *render.Options) {
	d.stack.Draw(o)
	d.shadow.Draw(o)

	if d.IsDead() {
		return
	}

	// Draw equipment
	for _, eq := range d.equipped {
		if eq != nil {
			eq.Draw(o)
		}
	}

	// Reset colors, as equipment may have munged it.
	o.DrawImageOptions.ColorScale.Reset()

	// Draw enemy if there is one
	if d.enemy != nil {
		d.enemy.Draw(*o)
	}
}

func (d *Dude) DrawProfile(o *render.Options) {
	stack := render.CopyStack(d.stack)
	stack.SetPosition(0, 0)
	stack.SetOrigin(0, 0)
	stack.SetRotation(-math.Pi / 2)
	stack.Draw(o)

	// Draw armor (like helmet or soemthing) ?
	armor := d.equipped[EquipmentTypeArmor]
	if armor != nil && armor.stack != nil {
		stack = render.CopyStack(armor.stack)
		stack.SetPosition(0, 0)
		stack.SetOrigin(0, 0)
		stack.SetRotation(-math.Pi / 2)
		stack.Draw(o)
	}
}

func (d *Dude) Trigger(e Event) Activity {
	// Trigger equipped equipment
	// It may modify event amounts
	for _, eq := range d.equipped {
		if eq != nil {
			eq.Activate(e)
		}
	}

	switch e := e.(type) {
	case EventDudeHit:
		if d.IsDead() {
			return DudeDeadActivity{dude: d}
		}
	case EventCombatRoom:
		// Can't fight if u ded
		if d.IsDead() {
			return nil
		}
		// Attack enemy if there is one
		if d.enemy != nil {
			damage, isCrit := d.GetDamage()
			if damage == 0 {
				d.Trigger(EventDudeMiss{dude: d, enemy: d.enemy})
				AddMessage(
					MessageNeutral,
					fmt.Sprintf("%s missed their attack against %s!", d.name, d.enemy.name),
				)
			} else if isCrit {
				AddMessage(
					MessageGood,
					fmt.Sprintf("%s crit %s for %d damage!", d.name, d.enemy.name, damage),
				)
				d.Trigger(EventDudeCrit{dude: d, enemy: d.enemy, amount: damage})
			}
			enemyKilled := d.enemy.Damage(d.stats.strength)

			if enemyKilled {
				xp := d.enemy.XP()
				gold := d.enemy.Gold()
				d.Trigger(EventGoldGain{dude: d, amount: gold})
				d.AddXP(xp)
				AddMessage(
					MessageGood,
					fmt.Sprintf("%s defeated %s and gained %d xp and %d gp", d.name, d.enemy.name, xp, gold),
				)
				if d.room != nil {
					loot := d.room.RollLoot(d.GetCalculatedStats().luck)
					if loot != nil {
						d.AddToInventory(loot)
					}
				}
				d.enemy = nil
			} else {
				takenDamage, isDodge := d.ApplyDamage(d.enemy.Hit())
				if !isDodge {
					if act := d.Trigger(EventDudeHit{dude: d, amount: takenDamage}); act != nil {
						return act
					}
					AddMessage(
						MessageBad,
						fmt.Sprintf("%s took %d damage from %s", d.name, takenDamage, d.enemy.name),
					)
				} else {
					d.Trigger(EventDudeDodge{dude: d, enemy: d.enemy})
					AddMessage(
						MessageNeutral,
						fmt.Sprintf("%s dodged an attack from %s", d.name, d.enemy.name),
					)
				}
				if d.IsDead() {
					d.enemy = nil
					return DudeDeadActivity{dude: d}
				}
			}
		}
		// Else it may be a trap room
		if act := d.room.GetRoomEffect(e); act != nil {
			return act
		}
	case EventEnterRoom:
		if act := d.room.GetRoomEffect(e); act != nil {
			return act
		}
	case EventCenterRoom:
		if act := d.room.GetRoomEffect(e); act != nil {
			return act
		}
	case EventLeaveRoom:
		if act := d.room.GetRoomEffect(e); act != nil {
			return act
		}
	case EventEquip:
		//fmt.Println(d.name, "equipped", e.equipment.Name())
		if d.stack != nil {
			d.floatingText(fmt.Sprintf("equip %s", e.equipment.Name()), color.NRGBA{100, 200, 200, 255}, 120, 0.4)
			AddMessage(
				MessageNeutral,
				fmt.Sprintf("%s equipped %s", d.name, e.equipment.Name()),
			)
		}
	case EventUnequip:
		d.dirtyEquipment = true
		//fmt.Println(d.name, "unequipped", e.equipment.Name())
		d.floatingText(fmt.Sprintf("remove %s", e.equipment.Name()), color.NRGBA{200, 100, 100, 255}, 120, 0.4)
	case EventGoldGain:
		d.UpdateGold(e.amount)
		//fmt.Println(d.name, "gained", e.amount, "gold")
		d.floatingText(fmt.Sprintf("+%dgp", e.amount), color.NRGBA{255, 255, 0, 255}, 40, 0.6)
	case EventGoldLoss:
		d.UpdateGold(e.amount)
		//fmt.Println(d.name, "lost", e.amount, "gold")
		d.floatingText(fmt.Sprintf("-%dgp", e.amount), color.NRGBA{255, 255, 0, 255}, 40, 0.4)
	}
	return nil
}

func (d *Dude) floatingText(text string, color color.NRGBA, lifetime int, speed float64) {
	if d == nil || d.story == nil {
		return
	}
	t := MakeFloatingTextFromDude(d, text, color, lifetime, speed)
	d.story.AddText(t)
}

func (d *Dude) Position() (float64, float64) {
	return d.stack.Position()
}

func (d *Dude) SetPosition(x, y float64) {
	d.stack.SetPosition(x, y)
}

func (d *Dude) Rotation() float64 {
	return d.stack.Rotation()
}

func (d *Dude) SetRotation(r float64) {
	d.stack.SetRotation(r)
}

func (d *Dude) Room() *Room {
	return d.room
}

func (d *Dude) SetRoom(r *Room) {
	d.room = r
}

func (d *Dude) Name() string {
	return d.name
}

func (d *Dude) Level() int {
	return d.stats.level
}

// Scale speed with agility
// Thinkin we probably shouldn't calculate this like this...
func (d *Dude) Speed() float64 {
	// This values probably belong somewhere else
	speedScale := 0.1
	baseSpeed := 0.01
	// Slow dude down when in combat.
	if d.enemy != nil {
		return baseSpeed * (1 + speedScale)
	}
	stats := d.GetCalculatedStats()
	return baseSpeed * (1 + float64(stats.agility/10)*speedScale)
}

// TODO: Refine this
func (d *Dude) GetDamage() (int, bool) {
	wasCrit := false
	stats := d.GetCalculatedStats()

	luckScaling := 0.1
	logisticScaling := func(x float64, max float64) float64 {
		return max / (1 + math.Exp(-luckScaling*(x-50)))
	}

	// Calculate crit chance
	baseCritChance := 0.05
	maxCritChance := 0.25
	critChance := baseCritChance + logisticScaling(float64(stats.luck), maxCritChance-baseCritChance)

	// Calculate miss chance (inverted from luck)
	baseMissChance := 0.1
	minMissChance := 0.01
	missChanceReduction := logisticScaling(float64(stats.luck), baseMissChance-minMissChance)
	missChance := math.Max(baseMissChance-missChanceReduction, minMissChance)

	randRoll := rand.Float64()
	multiplier := 1.0
	if randRoll < critChance {
		d.AddXP(1)
		d.floatingText("*CRIT*", color.NRGBA{255, 128, 255, 128}, 60, 1.0)
		multiplier = 2.0
		wasCrit = true
	} else if rand.Float64() < missChance {
		d.floatingText("*miss*", color.NRGBA{128, 128, 128, 128}, 30, 0.5)
		multiplier = 0.0
	}

	amount := int(float64(stats.strength) * multiplier)
	return amount, wasCrit
}

func (d *Dude) ApplyDamage(amount int) (int, bool) {
	if d.IsDead() {
		return 0, false
	}

	// Luck and agility can cause dodge
	stats := d.GetCalculatedStats()
	baseChance := 0.05                                   // 5% base dodge chance
	luckContribution := float64(stats.luck) * 0.005      // 0.5% per luck point
	agilityContribution := float64(stats.agility) * 0.01 // 1% per agility point

	dodgeChance := baseChance + luckContribution + agilityContribution

	// Cap the maximum dodge chance at 50%
	chance := math.Min(dodgeChance, 0.5)
	if rand.Float64() < chance {
		d.AddXP(1)
		d.floatingText("*dodge*", color.NRGBA{255, 255, 0, 128}, 30, 0.5)
		AddMessage(
			MessageNeutral,
			fmt.Sprintf("%s dodged an attack", d.name),
		)
		return 0, true
	}

	// Apply defense stat
	amount = stats.ApplyDefense(amount)
	d.stats.currentHp -= amount

	if d.stats.currentHp <= 0 {
		if d.invincible {
			d.stats.currentHp = d.stats.totalHp
		} else {
			d.stats.currentHp = 0
		}
	}

	if d.stats.currentHp == 0 {
		d.SetActivity(Ded)

		d.floatingText("RIP", color.NRGBA{64, 64, 64, 255}, 80, 1)
		AddMessage(
			MessageBad,
			fmt.Sprintf("%s took %d damage and was defeated", d.name, amount),
		)
	} else {
		d.floatingText(fmt.Sprintf("%d", -amount), color.NRGBA{255, 0, 0, 255}, 40, 0.5)
	}
	d.dirtyStats = true
	return amount, false
}

func (d *Dude) Stats() *Stats {
	return &d.stats
}

func (d *Dude) Profession() ProfessionKind {
	return d.profession
}

func (d *Dude) Gold() int {
	return d.gold
}

func (d *Dude) UpdateGold(amount int) {
	d.gold += amount
	if d.gold < 0 {
		d.gold = 0
	}
}

// Equips item to dude
func (d *Dude) Equip(eq *Equipment) *Equipment {
	e := d.Unequip(eq.Type())
	if e != nil {
		d.Trigger(EventUnequip{dude: d, equipment: e}) // Event isolated to dude?
	}
	d.equipped[eq.Type()] = eq
	d.Trigger(EventEquip{dude: d, equipment: eq}) // Event isolated to dude?

	// If equipment is in inventory, remove it
	for i, e := range d.inventory {
		if e == eq {
			d.inventory = append(d.inventory[:i], d.inventory[i+1:]...)
			break
		}
	}
	d.dirtyEquipment = true
	return e
}

func (d *Dude) Unequip(t EquipmentType) *Equipment {
	if e, ok := d.equipped[t]; ok {
		d.Trigger(EventUnequip{dude: d, equipment: e}) // Event isolated to dude?
		// Delete the equipment from the equipped map
		delete(d.equipped, t)
		d.dirtyEquipment = true
		return e // Return the unequipped item
	}
	return nil
}

func (d *Dude) CanEquip(eq *Equipment) bool {
	equippable := len(eq.professions) == 0
	if !equippable {
		for _, p := range eq.professions {
			if p == d.profession {
				return true
			}
		}
	}
	return false
}

// If the provided equipment should replace currently equipped item:
//   - equipped does not have perk
//   - new item is better
//   - new item belongs to profession
func (d *Dude) ShouldEquip(eq *Equipment) bool {
	equippable := len(eq.professions) == 0
	if !equippable {
		for _, p := range eq.professions {
			if p == d.profession {
				equippable = true
				break
			}
		}
	}

	equipped := d.equipped[eq.Type()]
	if equipped == nil {
		return equippable
	}

	newItemBetter := eq.LevelWithQuality() > equipped.LevelWithQuality()

	ep := equipped.perk
	np := eq.perk
	if ep == nil {
		return equippable && newItemBetter
	} else if np == nil {
		return false
	}

	newItemPerkBetter := np.Name() == ep.Name() && np.Quality() >= ep.Quality()
	return equippable && newItemPerkBetter && newItemBetter
}

func (d *Dude) AddToInventory(eq *Equipment) {
	d.inventory = append(d.inventory, eq)
	shouldEquip := d.ShouldEquip(eq)

	professions := eq.professions
	hasProfession := false
	if len(professions) == 0 {
		hasProfession = true
	} else {
		for _, p := range professions {
			if p == d.profession {
				hasProfession = true
				break
			}
		}
	}
	shouldEquip = shouldEquip && hasProfession
	if shouldEquip {
		d.Equip(eq)
	} else {
		//fmt.Println(d.name, "added", eq.Name(), "to inventory")
		d.floatingText(fmt.Sprintf("+%s", eq.Name()), color.NRGBA{200, 200, 50, 128}, 100, 1.0)
	}
}

func (d *Dude) Inventory() []*Equipment {
	return d.inventory
}

func (d *Dude) Equipped() map[EquipmentType]*Equipment {
	return d.equipped
}

// Returns the stats of the dude with the equipment stats added
func (d *Dude) GetCalculatedStats() *Stats {
	stats := NewStats(nil, false)
	stats = stats.Add(&d.stats)
	for _, eq := range d.equipped {
		stats = stats.Add(eq.Stats())
	}
	stats.currentHp = d.stats.currentHp
	return stats
}

func (d *Dude) AddXP(xp int) {
	d.xp += xp
	// If level reached
	nextLevelXP := d.NextLevelXP()
	if d.xp >= nextLevelXP {
		d.xp -= nextLevelXP
		d.stats.LevelUp(false)
		d.floatingText("LEVEL UP", color.NRGBA{100, 255, 255, 255}, 80, 1)
		AddMessage(
			MessageGood,
			fmt.Sprintf("%s leveled up to level %d", d.name, d.Level()),
		)
	} else {
		d.floatingText(fmt.Sprintf("+%dxp", xp), color.NRGBA{100, 200, 200, 200}, 50, 1)
	}
	d.dirtyStats = true
}

func (d *Dude) XP() int {
	return d.xp
}
func (d *Dude) NextLevelXP() int {
	return 50 * d.Level()
}

func (d *Dude) Heal(amount int) int {
	// no healing dead dudes
	if d.IsDead() {
		return 0
	}

	initialHP := d.stats.currentHp
	stats := d.GetCalculatedStats()
	amount *= (stats.wisdom / 10) + 1 // Wisdom scaling
	d.stats.currentHp += amount
	if d.stats.currentHp > stats.totalHp {
		d.stats.currentHp = stats.totalHp
	}
	amount = d.stats.currentHp - initialHP

	if amount > 0 && d.story != nil {
		d.floatingText(fmt.Sprintf("+%d", amount), color.NRGBA{0, 255, 0, 255}, 40, 0.5)
		AddMessage(
			MessageNeutral,
			fmt.Sprintf("%s healed for %d", d.name, amount),
		)
		d.dirtyStats = true
	}
	return amount
}

func (d *Dude) FullHeal() {
	stats := d.GetCalculatedStats()

	// No rez
	if !d.IsDead() {
		d.Heal(stats.totalHp)
		d.dirtyStats = true
	}
}

func (d *Dude) RestoreUses() {
	restored := false
	for _, eq := range d.equipped {
		if eq != nil {
			restored = eq.RestoreUses() || restored
		}
	}
	if restored && d.story != nil {
		d.floatingText("+eq restore", color.NRGBA{0, 128, 255, 200}, 40, 0.5)
		AddMessage(
			MessageNeutral,
			fmt.Sprintf("%s restored equipment uses", d.name),
		)
		d.dirtyEquipment = true
	}
}

func (d *Dude) RandomEquippedItem() *Equipment {
	equippedTypes := []EquipmentType{}
	for t, eq := range d.equipped {
		if eq != nil {
			equippedTypes = append(equippedTypes, t)
		}
	}

	if len(equippedTypes) == 0 {
		return nil
	}

	et := equippedTypes[rand.Intn(len(equippedTypes))]
	return d.equipped[et]
}

func (d *Dude) LevelUpEquipment(amount int, maxQuality EquipmentQuality) {
	// Random equipped item
	eq := d.RandomEquippedItem()
	if eq == nil {
		return
	}

	leveled := false
	for i := 0; i < amount; i++ {
		leveled = eq.LevelUp(maxQuality) || leveled
	}

	if !leveled {
		return
	}

	//fmt.Println(d.name, "leveled up equipment by", amount)
	d.floatingText(fmt.Sprintf("+eq up %d", amount), color.NRGBA{128, 128, 255, 255}, 50, 0.5)
	AddMessage(
		MessageLoot,
		fmt.Sprintf("%s leveled up %s by %d", d.name, eq.Name(), amount),
	)
	d.dirtyEquipment = true
}

func (d *Dude) Perkify(maxQuality PerkQuality) {
	eq := d.RandomEquippedItem()
	if eq == nil {
		return
	}

	// Assign random perk
	if eq.perk == nil {
		prevName := eq.Name()
		eq.perk = GetRandomPerk(PerkQualityTrash)
		//fmt.Println(d.name, "upgraded his equipment", prevName, "with", eq.perk.Name())
		d.floatingText(fmt.Sprintf("+%s perk %s", prevName, eq.perk.Name()), color.NRGBA{128, 255, 128, 255}, 100, 0.5)
		AddMessage(
			MessageLoot,
			fmt.Sprintf("%s upgraded %s with %s", d.name, prevName, eq.perk.Name()),
		)
	} else {
		// Level up perk
		previousQuality := eq.perk.Quality()
		previousName := eq.Name()
		eq.perk.LevelUp(maxQuality)
		if eq.perk.Quality() != previousQuality {
			//fmt.Println(eq.perk.Quality(), previousQuality)
			//fmt.Println(d.name, "upgraded his equipment", previousName, "to", eq.Name())
			d.floatingText(fmt.Sprintf("+eq %s upgrade to %s", previousName, eq.Name()), color.NRGBA{128, 255, 128, 255}, 100, 0.5)
			AddMessage(
				MessageLoot,
				fmt.Sprintf("%s upgraded %s to %s", d.name, previousName, eq.Name()),
			)
		}
	}
	d.dirtyEquipment = true
}

// Cursify the dude
// Rolls twice, once with wisdom, the other with luck, and takes the highest
// Has a chance to
// - Delevel equipment (high chance)
// - Delevel perk (medium chance)
// - Delevel dude (low chance)
func (d *Dude) Cursify(roomLevel int) {
	stats := d.GetCalculatedStats()

	// Ensure stats are not negative
	wis := max(stats.wisdom, 1)
	luck := max(stats.luck, 1)

	wisdomRoll := rand.Intn(wis) + 1 // ensure non-zero roll
	luckRoll := rand.Intn(luck) + 1
	highestRoll := max(wisdomRoll, luckRoll)

	// Higher the roll, lower the chance of being cursed
	threshold := 1.0 - math.Log10(float64(highestRoll+1))

	curseRoll := rand.Float64()
	if curseRoll > threshold {
		// Spared
		return
	}

	// Check for gold loss
	if curseRoll <= threshold*0.75 { // high chance for gold loss
		goldLoss := roomLevel * 5
		d.Trigger(EventGoldLoss{dude: d, amount: goldLoss})
		//fmt.Println(d.name, "lost", goldLoss, "gold")
		d.floatingText(fmt.Sprintf("-%dgp", goldLoss), color.NRGBA{255, 255, 0, 200}, 40, 0.5)

		// If gold is negative, set to 0
		if d.gold < 0 {
			d.gold = 0
		}

		AddMessage(
			MessageBad,
			fmt.Sprintf("%s lost %dgp", d.name, goldLoss),
		)
	}

	// Check for equipment delevel
	if curseRoll <= threshold*0.5 { // reduced chance for equipment delevel
		equipmentType := RandomEquipmentType()
		if eq := d.equipped[equipmentType]; eq != nil {
			eq.LevelDown()
			//fmt.Println(d.name, "lost a level on", eq.Name())
			d.floatingText(fmt.Sprintf("-eq level %s", eq.Name()), color.NRGBA{200, 200, 32, 200}, 50, 0.5)

			AddMessage(
				MessageBad,
				fmt.Sprintf("%s lost a level on %s", d.name, eq.Name()),
			)
		}
		d.dirtyEquipment = true
		d.dirtyStats = true
	}

	// Check for perk delevel
	if curseRoll <= threshold*0.25 { // even lower chance for perk delevel
		equipmentWithPerks := []EquipmentType{}
		for t, eq := range d.equipped {
			if eq != nil && eq.perk != nil {
				equipmentWithPerks = append(equipmentWithPerks, t)
			}
		}
		if len(equipmentWithPerks) > 0 {
			randomEquipType := equipmentWithPerks[rand.Intn(len(equipmentWithPerks))]
			if eq := d.equipped[randomEquipType]; eq != nil {
				eq.perk.LevelDown()
				//fmt.Println(d.name, "lost a perk level on", eq.Name())
				d.floatingText(fmt.Sprintf("-eq perk %s", eq.Name()), color.NRGBA{200, 200, 32, 200}, 50, 0.5)

				AddMessage(
					MessageBad,
					fmt.Sprintf("%s lost a perk level on %s", d.name, eq.Name()),
				)
			}
		}
		d.dirtyEquipment = true
		d.dirtyStats = true
	}

	// Check for dude delevel
	if curseRoll <= threshold*0.1 { // lowest chance for dude delevel
		d.stats.LevelDown()
		//fmt.Println(d.name, "lost a level")
		d.floatingText("-level", color.NRGBA{100, 0, 0, 200}, 50, 0.5)

		AddMessage(
			MessageBad,
			fmt.Sprintf("%s lost a level and is now level %d", d.name, d.stats.level),
		)
		d.dirtyStats = true
	}
}

func (d *Dude) TrapDamage(roomLevel int) Activity {
	// he's dead jim
	if d.IsDead() {
		d.SetActivity(Ded)
		return DudeDeadActivity{dude: d}
	}
	// Chance based on agility
	agilityRoll := rand.Intn(d.stats.agility + 1)

	// Higher agility, lower chance of being hit
	threshold := 1.0 - math.Log10(float64(agilityRoll+1))
	trapRoll := rand.Float64()

	if trapRoll > threshold {
		AddMessage(
			MessageNeutral,
			fmt.Sprintf("%s dodged damage from a trap", d.name),
		)
		return nil
	}

	// Damage based on room level
	damage := (roomLevel + 1) * 3

	amount, miss := d.ApplyDamage(damage)
	if !miss {
		AddMessage(
			MessageNeutral,
			fmt.Sprintf("%s took %d damage from a trap", d.name, amount),
		)
		d.dirtyStats = true
	}
	if d.IsDead() {
		d.SetActivity(Ded)
		return DudeDeadActivity{dude: d}
	}
	return nil
}

func (d *Dude) IsDead() bool {
	if d == nil {
		return true
	}

	// Nope, not dead
	if d.invincible && d.stats.currentHp <= 0 {
		d.stats.currentHp = d.stats.totalHp
	}
	return d.stats.currentHp <= 0 || d.activity == Ded
}

func SortDudes(sp SortProperty, dudes []*Dude) []*Dude {
	if len(dudes) <= 1 {
		return dudes
	}

	nameSort := func(i, j int) bool {
		return dudes[i].name < dudes[j].name
	}

	professionSort := func(i, j int) bool {
		a := dudes[i].profession
		b := dudes[j].profession
		if a == b {
			return nameSort(i, j)
		}
		return a < b
	}

	levelSort := func(i, j int) bool {
		a := dudes[i].GetCalculatedStats().level
		b := dudes[j].GetCalculatedStats().level
		if a == b {
			return nameSort(i, j)
		}
		return a < b
	}

	var sortMethod func(i, j int) bool = nameSort
	switch sp {
	case SortPropertyProfession:
		sortMethod = professionSort
	case SortPropertyLevel:
		sortMethod = levelSort
	case SortPropertyName:
	default:
	}

	d := make([]*Dude, len(dudes))
	copy(d, dudes)
	sort.SliceStable(d, sortMethod)
	return d
}
