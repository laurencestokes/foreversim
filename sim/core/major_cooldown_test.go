package core

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

const (
	fakeDefensiveID int32 = 993001
	fakeOffensiveID int32 = 993002
)

func init() {
	RegisterAgentFactory(
		proto.Player_Mage{},
		proto.Spec_SpecMage,
		func(char *Character, _ *proto.Player, _ *proto.Raid) Agent {
			return &fakeCooldownMage{Character: *char, casts: map[int32]int{}}
		},
		func(player *proto.Player, spec interface{}) {
			player.Spec = spec.(*proto.Player_Mage)
		},
	)
}

// A stand-in with one survival and one DPS major cooldown, each off the GCD on a 10 s cooldown, which
// counts how often each is used.
type fakeCooldownMage struct {
	Character
	casts map[int32]int
}

func (m *fakeCooldownMage) GetCharacter() *Character       { return &m.Character }
func (m *fakeCooldownMage) ApplyTalents()                  {}
func (m *fakeCooldownMage) Reset(_ *Simulation)            {}
func (m *fakeCooldownMage) OnGCDReady(_ *Simulation)       {}
func (m *fakeCooldownMage) OnEncounterStart(_ *Simulation) {}

func (m *fakeCooldownMage) Initialize() {
	m.registerCooldown(fakeDefensiveID, CooldownTypeSurvival)
	m.registerCooldown(fakeOffensiveID, CooldownTypeDPS)
}

func (m *fakeCooldownMage) registerCooldown(id int32, cdType CooldownType) {
	spell := m.RegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: id},
		Flags:    SpellFlagNoOnCastComplete,
		Cast:     CastConfig{CD: Cooldown{Timer: m.NewTimer(), Duration: 10 * time.Second}},
		ApplyEffects: func(_ *Simulation, _ *Unit, _ *Spell) {
			m.casts[id]++
		},
	})
	m.AddMajorCooldown(MajorCooldown{Spell: spell, Type: cdType})
}

func autocastRotation() *proto.APLRotation {
	return &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{{
		Action: &proto.APLAction{Action: &proto.APLAction_AutocastOtherCooldowns{AutocastOtherCooldowns: &proto.APLActionAutocastOtherCooldowns{}}},
	}}}
}

func castRotation(id int32) *proto.APLRotation {
	return &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{{
		Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: ActionID{SpellID: id}.ToProto()}}},
	}}}
}

// Runs one 60 s iteration with a stand-in per rotation, the raid listing the given tanks, and returns
// the stand-ins.
func runCooldownMages(t *testing.T, tanks []*proto.UnitReference, rotations ...*proto.APLRotation) []*fakeCooldownMage {
	t.Helper()
	var players []*proto.Player
	for _, rotation := range rotations {
		players = append(players, &proto.Player{
			Name:      "Mage",
			Class:     proto.Class_ClassMage,
			Race:      proto.Race_RaceHuman,
			Buffs:     &proto.IndividualBuffs{},
			Spec:      &proto.Player_Mage{},
			Equipment: &proto.EquipmentSpec{},
			Rotation:  rotation,
		})
	}
	sim := NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{Players: players, Buffs: &proto.PartyBuffs{}}},
			Tanks:   tanks,
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "target", Level: 63, MobType: proto.MobType_MobTypeHumanoid}},
			Duration: 60,
		},
	}, simsignals.CreateSignals())
	sim.runOnce()

	var mages []*fakeCooldownMage
	for _, player := range sim.Raid.Parties[0].Players {
		mages = append(mages, player.(*fakeCooldownMage))
	}
	return mages
}

func playerRef(index int32) *proto.UnitReference {
	return &proto.UnitReference{Type: proto.UnitReference_Player, Index: index}
}

// The manager uses a survival cooldown on its own only for a player in a tank slot, any slot, whether
// or not a target is assigned to it, and a DPS cooldown for everyone.
func TestSurvivalCooldownsAutocastOnlyForTanks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		tanks []*proto.UnitReference
		want  []bool
	}{
		{"no tanks", nil, []bool{false, false}},
		{"the second player tanks", []*proto.UnitReference{playerRef(1)}, []bool{false, true}},
		{"the second player in the second slot", []*proto.UnitReference{playerRef(0), playerRef(1)}, []bool{true, true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mages := runCooldownMages(t, tc.tanks, autocastRotation(), autocastRotation())
			for i, mage := range mages {
				if used := mage.casts[fakeDefensiveID] > 0; used != tc.want[i] {
					t.Errorf("player %d used the survival cooldown %d times, want used %v", i, mage.casts[fakeDefensiveID], tc.want[i])
				}
				if mage.casts[fakeOffensiveID] == 0 {
					t.Errorf("player %d never used the DPS cooldown", i)
				}
			}
		})
	}
}

// An APL action that names a survival cooldown uses it for a player no tank slot lists.
func TestSurvivalCooldownCastByTheRotationForANonTank(t *testing.T) {
	mage := runCooldownMages(t, nil, castRotation(fakeDefensiveID))[0]
	if mage.casts[fakeDefensiveID] == 0 {
		t.Errorf("the rotation never cast the survival cooldown")
	}
}

// A stat buff's cooldown type: any defensive stat or resistance makes it survival, mana regen and
// spirit alone mana, and anything else DPS.
func TestInferCDTypeFromTheBuffedStats(t *testing.T) {
	for _, tc := range []struct {
		stats []stats.Stat
		want  CooldownType
	}{
		{[]stats.Stat{stats.Armor}, CooldownTypeSurvival},
		{[]stats.Stat{stats.BlockValue}, CooldownTypeSurvival},
		{[]stats.Stat{stats.BlockPercent}, CooldownTypeSurvival},
		{[]stats.Stat{stats.DodgePercent}, CooldownTypeSurvival},
		{[]stats.Stat{stats.ParryPercent}, CooldownTypeSurvival},
		{[]stats.Stat{stats.ArcaneResistance}, CooldownTypeSurvival},
		{[]stats.Stat{stats.FireResistance}, CooldownTypeSurvival},
		{[]stats.Stat{stats.FrostResistance}, CooldownTypeSurvival},
		{[]stats.Stat{stats.NatureResistance}, CooldownTypeSurvival},
		{[]stats.Stat{stats.ShadowResistance}, CooldownTypeSurvival},
		{[]stats.Stat{stats.Spirit, stats.Armor}, CooldownTypeSurvival},
		{[]stats.Stat{stats.MP5}, CooldownTypeMana},
		{[]stats.Stat{stats.Spirit}, CooldownTypeMana},
		{[]stats.Stat{stats.MP5, stats.Spirit}, CooldownTypeMana},
		{[]stats.Stat{stats.Spirit, stats.Strength}, CooldownTypeDPS},
		{[]stats.Stat{stats.AttackPower}, CooldownTypeDPS},
		{[]stats.Stat{stats.SpellCritPercent}, CooldownTypeDPS},
		{nil, CooldownTypeDPS},
	} {
		if got := (&StatBuffAura{BuffedStatTypes: tc.stats}).InferCDType(); got != tc.want {
			t.Errorf("%v: type %v, want %v", tc.stats, got, tc.want)
		}
	}
}
