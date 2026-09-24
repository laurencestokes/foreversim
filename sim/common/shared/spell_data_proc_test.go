package shared

import (
	"strings"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Eternal Power, the Dormant Heart of the Mountain proc: 1249118 carries the proc flags and 1249119
// is the buff it applies. The item IDs below are the test's own, so that registering them cannot
// collide with a real effect.
const (
	healProcTrigger int32 = 1249118
	healProcBuff    int32 = 1249119
)

// core skips an effect whose item this client does not ship, which every id below is: the test's own
// ids are chosen to be absent from the shipped database so that registering them cannot collide with
// a real effect. Putting them in the database first is what lets the registration run under the
// with_db build tag as it does without it.
func withTestItems(ids ...int32) {
	items := make([]*proto.SimItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, &proto.SimItem{Id: id, Name: "Test Trinket"})
	}
	core.AddToDatabase(&proto.SimDatabase{Items: items})
}

func withTestEnchant(effectID int32) {
	core.AddToDatabase(&proto.SimDatabase{
		Enchants: []*proto.SimEnchant{{EffectId: effectID, Name: "Test Enchant"}},
	})
}

func TestSpellDataProcRegistersEveryVariant(t *testing.T) {
	withTestItems(990001, 990002)
	NewSpellDataProc(SpellDataProc{TriggerSpellID: healProcTrigger, BuffSpellID: healProcBuff},
		[]ItemVariant{
			{ItemID: 990001, ItemName: "Reissued Test Trinket"},
			{ItemID: 990002, ItemName: "Test Trinket"},
		})

	if !core.HasItemEffect(990001) || !core.HasItemEffect(990002) {
		t.Error("a variant was left unregistered")
	}

	// Only the highest ID goes into the test suite, so a dozen re-issues of one trinket do not each
	// get their own fixture entry.
	if core.HasItemEffectForTest(990001) {
		t.Error("the lower-ID variant was added to the test suite as well")
	}
	if !core.HasItemEffectForTest(990002) {
		t.Error("the highest-ID variant was kept out of the test suite")
	}
	if !core.AddEffectsToTest {
		t.Error("the variant loop left effects out of the test suite for whatever registers next")
	}
}

// A row that names no hits the sim hears would register a listener that can never fire. 1249119 is
// the buff half of the pair above: it carries the stats and no proc flags at all.
func TestSpellDataProcSkipsARowWithNoListener(t *testing.T) {
	withTestItems(990003)
	NewSpellDataProc(SpellDataProc{TriggerSpellID: healProcBuff},
		[]ItemVariant{{ItemID: 990003, ItemName: "Listenerless Test Trinket"}})

	if core.HasItemEffect(990003) {
		t.Error("an effect whose row states no listener was registered anyway")
	}
}

// An enchant states its own name and registers through the enchant registry instead.
func TestSpellDataProcRegistersAnEnchant(t *testing.T) {
	withTestEnchant(990004)
	NewSpellDataProc(SpellDataProc{
		Name:           "Test Enchant",
		EnchantID:      990004,
		TriggerSpellID: healProcTrigger,
		BuffSpellID:    healProcBuff,
	}, nil)

	if !core.HasEnchantEffect(990004) {
		t.Error("an enchant proc with no variants was left unregistered")
	}
	if core.HasItemEffect(990004) {
		t.Error("an enchant proc was registered as an item effect")
	}
}

// A row pair in the shape the charge rules read: a trigger that grants the buff and a buff that
// states its charges and what spends them.
func chargeRows() (*spelldata.Spell, *spelldata.Spell) {
	trigger := &spelldata.Spell{
		ID: 990100, Name: "Charge Trigger", DurationMs: 12000, ProcChance: 100,
		ProcChanceSource: spelldata.ProcChanceAlways,
		ProcFlags:        [2]uint32{0: dbcenums.PROC_FLAG_DEAL_MELEE_ABILITY},
	}
	buff := &spelldata.Spell{
		ID: 990101, Name: "Charge Buff", DurationMs: 12000, ProcChance: 100, ProcCharges: 2,
		ProcChanceSource: spelldata.ProcChanceAlways,
		ProcFlags:        [2]uint32{0: dbcenums.PROC_FLAG_DEAL_MELEE_SWING},
	}

	return trigger, buff
}

func chargeAura() *core.StatBuffAura {
	return &core.StatBuffAura{Aura: &core.Aura{Label: "Charge Buff Proc"}}
}

// A buff that is its own trigger hears the hits that granted it, so a spender read off that row
// would take a charge back on the hit that handed it over.
func TestSpellDataProcChargeSpenderNeedsABuffOfItsOwn(t *testing.T) {
	trigger, buff := chargeRows()

	sameRow := chargeAura()
	attachChargeSpender(&core.Character{}, SpellDataProc{Name: "Test"}, trigger, trigger, sameRow)
	if sameRow.OnSpellHitDealt != nil {
		t.Error("a buff that is its own trigger got a charge spender, which would spend the charge it was just given")
	}

	ownRow := chargeAura()
	attachChargeSpender(&core.Character{}, SpellDataProc{Name: "Test"}, trigger, buff, ownRow)
	if ownRow.OnSpellHitDealt == nil {
		t.Error("a buff stating its own charges and flags got no spender")
	}
}

// The rows a trigger cannot be built from are the rows spelldata.ProcTrigger panics on, and asking
// first is the only thing between a charge spender and that panic.
func TestSpellDataProcStatesATrigger(t *testing.T) {
	_, buff := chargeRows()
	if !statesATrigger(buff) {
		t.Error("a row stating both a listener and a chance was refused")
	}

	// Damage of any kind names a listener without naming a hit kind, so this row hears something and
	// still has no mask a procs-per-minute rate could be measured against.
	maskless := *buff
	maskless.ProcFlags = [2]uint32{0: dbcenums.PROC_FLAG_TAKE_ANY_DAMAGE}
	maskless.ProcChanceSource = spelldata.ProcChancePPM
	maskless.RPPM = 2
	if statesATrigger(&maskless) {
		t.Error("a procs-per-minute row with no mask to measure it on was accepted")
	}

	pastTheEffects := *buff
	pastTheEffects.ProcChanceSource = spelldata.ProcChanceEffectN
	pastTheEffects.ProcChanceEffect = 3
	if statesATrigger(&pastTheEffects) {
		t.Error("a row naming an effect it does not carry as its chance was accepted")
	}
}

// The client leaves the duration off the buff on a fair few procs and states it on the trigger, and
// an aura of no duration is one core refuses to activate.
func TestProcBuffDurationFallsBackToTheTrigger(t *testing.T) {
	trigger, buff := chargeRows()
	cfg := SpellDataProc{Name: "Test", ItemID: 990102}

	buff.DurationMs = 0
	if got := procBuffDuration(cfg, trigger, buff); got != 12*time.Second {
		t.Errorf("duration = %v, want the trigger's 12s", got)
	}

	trigger.DurationMs = 0
	requirePanicContaining(t, "states a duration", func() {
		procBuffDuration(cfg, trigger, buff)
	})
}

func requirePanicContaining(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("no panic, want one mentioning %q", want)
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, want) {
			t.Fatalf("panic %v, want one mentioning %q", r, want)
		}
	}()
	fn()
}

// A character bare enough to register an effect on: no environment, no raid, no equipment, which is
// all the registration reads off it.
type testAgent struct{ character *core.Character }

func (a *testAgent) GetCharacter() *core.Character                                    { return a.character }
func (a *testAgent) Initialize()                                                      {}
func (a *testAgent) AddRaidBuffs(_ *proto.RaidBuffs)                                  {}
func (a *testAgent) AddPartyBuffs(_ *proto.PartyBuffs)                                {}
func (a *testAgent) ApplyTalents()                                                    {}
func (a *testAgent) Reset(_ *core.Simulation)                                         {}
func (a *testAgent) OnManaTick(_ *core.Simulation)                                    {}
func (a *testAgent) OnEncounterStart(_ *core.Simulation)                              {}
func (a *testAgent) ExecuteCustomRotation(_ *core.Simulation)                         {}
func (a *testAgent) NewAPLValue(_ *core.APLRotation, _ *proto.APLValue) core.APLValue { return nil }
func (a *testAgent) NewAPLAction(_ *core.APLRotation, _ *proto.APLAction) core.APLActionImpl {
	return nil
}

func newTestAgent() *testAgent {
	character := core.NewCharacter(&core.Party{}, 0, &proto.Player{
		Name:      "Proc Tester",
		Class:     proto.Class_ClassWarrior,
		Race:      proto.Race_RaceHuman,
		Spec:      &proto.Player_ProtectionWarrior{ProtectionWarrior: &proto.ProtectionWarrior{}},
		Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{}},
	})

	return &testAgent{character: &character}
}

func auraByLabel(character *core.Character, label string) *core.Aura {
	for _, aura := range character.GetAuras() {
		if aura.Label == label {
			return aura
		}
	}

	return nil
}

// The whole registration on a character: the buff as the rows describe it, the listener that applies
// it, and the internal cooldown handed to the buff for the APL values that ask after it.
func TestSpellDataProcAppliesToACharacter(t *testing.T) {
	const itemID int32 = 990201

	core.AddToDatabase(&proto.SimDatabase{Items: []*proto.SimItem{{
		Id:   itemID,
		Name: "Test Trinket",
		ItemEffects: []*proto.ItemEffect{{
			BuffId:           990111,
			EffectDurationMs: 15000,
			ScalingOptions: map[int32]*proto.ScalingItemEffectProperties{
				0: {Stats: map[int32]float64{int32(proto.Stat_StatSpellDamage): 40}},
			},
			Effect: &proto.ItemEffect_Proc{Proc: &proto.ProcEffect{}},
		}},
	}}})

	trigger := &spelldata.Spell{
		ID: 990110, Name: "Test Trinket Trigger", ProcChance: 15, ICDMs: 45000,
		ProcChanceSource: spelldata.ProcChanceColumn,
		ProcFlags:        [2]uint32{0: dbcenums.PROC_FLAG_DEAL_MELEE_ABILITY},
	}
	buff := &spelldata.Spell{ID: 990111, Name: "Test Trinket Buff", DurationMs: 15000}

	cfg := SpellDataProc{Name: "Test Trinket", ItemID: itemID,
		TriggerSpellID: trigger.ID, BuffSpellID: buff.ID}

	agent := newTestAgent()
	applySpellDataProc(agent, cfg, cfg.effectSource(), trigger, buff)

	procAura := auraByLabel(agent.character, "Test Trinket Proc")
	if procAura == nil {
		t.Fatal("the buff the proc applies was not registered")
	}
	if procAura.Duration != 15*time.Second {
		t.Errorf("buff duration = %v, want the buff row's 15s", procAura.Duration)
	}

	triggerAura := auraByLabel(agent.character, cfg.Name)
	if triggerAura == nil {
		t.Fatal("the listener was not registered")
	}
	// MakeProcTriggerAura keeps the action a proc is keyed by apart from the one its metrics use.
	if triggerAura.ActionIDForProc != (core.ActionID{ItemID: itemID}) {
		t.Errorf("listener action = %v, want the item's", triggerAura.ActionIDForProc)
	}
	if triggerAura.Icd == nil || triggerAura.Icd.Duration != 45*time.Second {
		t.Fatalf("listener ICD = %v, want the trigger row's 45s", triggerAura.Icd)
	}
	if procAura.Icd != triggerAura.Icd {
		t.Error("the buff was not handed the listener's cooldown, which is what the ICD-aware APL values read")
	}
}

// A damage proc has no buff at all: the row it names is a spell that deals damage, and what the
// client states about that spell is its school, its hit table and the amount it rolls. What it does
// not state is that the game casts it off a hit, which is the GCD, the cost and the cast time this
// has to strip.
func TestSpellDataDamageProcAppliesToACharacter(t *testing.T) {
	const itemID int32 = 990301

	core.AddToDatabase(&proto.SimDatabase{Items: []*proto.SimItem{{Id: itemID, Name: "Test Spike"}}})

	trigger := &spelldata.Spell{
		ID: 990310, Name: "Test Spike Trigger", ProcChance: 5,
		ProcChanceSource: spelldata.ProcChanceColumn,
		ProcFlags:        [2]uint32{0: dbcenums.PROC_FLAG_TAKE_MELEE_SWING},
	}
	damage := &spelldata.Spell{
		ID: 990311, Name: "Test Spike Bolt", School: 8, DefenseType: 1,
		GCDMs: 1500, StartRecoveryCategory: 133,
		Powers:  []spelldata.Power{{Type: 0, Cost: 100}},
		Effects: []spelldata.Effect{{SpellID: 990311, Type: dbcenums.E_SCHOOL_DAMAGE, BasePoints: 50}},
	}

	cfg := SpellDataProc{Name: "Test Spike", ItemID: itemID,
		TriggerSpellID: trigger.ID, BuffSpellID: damage.ID}

	agent := newTestAgent()
	applySpellDataDamageProc(agent, cfg, cfg.effectSource(), trigger, damage)

	spell := agent.character.GetSpell(core.ActionID{SpellID: damage.ID})
	if spell == nil {
		t.Fatal("the spell the proc casts was not registered")
	}
	if spell.SpellSchool != core.SpellSchoolNature {
		t.Errorf("school = %v, want the row's Nature", spell.SpellSchool)
	}
	if spell.DefenseType != core.DefenseTypeMagic {
		t.Errorf("defense type = %v, want the row's magic", spell.DefenseType)
	}
	for _, flag := range []core.SpellFlag{core.SpellFlagPassiveSpell, core.SpellFlagNoOnCastComplete,
		core.SpellFlagNoOnDamageDealt, core.SpellFlagProc} {
		if !spell.Flags.Matches(flag) {
			t.Errorf("flags = %b, want %b set", spell.Flags, flag)
		}
	}
	if spell.ProcMask != core.ProcMaskEmpty {
		t.Errorf("proc mask = %v, want none: a proc's damage is heard through its flags", spell.ProcMask)
	}
	if spell.DefaultCast.GCD != 0 || spell.DefaultCast.CastTime != 0 || spell.DefaultCast.Cost != 0 {
		t.Errorf("cast = %+v, want no global cooldown, no cast time and no cost", spell.DefaultCast)
	}

	triggerAura := auraByLabel(agent.character, cfg.Name)
	if triggerAura == nil {
		t.Fatal("the listener was not registered")
	}
	if triggerAura.ActionIDForProc != (core.ActionID{ItemID: itemID}) {
		t.Errorf("listener action = %v, want the item's", triggerAura.ActionIDForProc)
	}
}

// Which unit a proc's damage lands on, per callback. The self-hit case is the one the sapper charges
// created: a character deals that hit to itself, and a proc reading the result answered it by
// hitting its own wearer.
func TestProcDamageTarget(t *testing.T) {
	character := &core.Character{Unit: core.Unit{Label: "Wearer"}}
	enemy := &core.Unit{Label: "Enemy"}
	attacker := &core.Unit{Label: "Attacker"}
	character.CurrentTarget = enemy

	for _, tc := range []struct {
		callback core.AuraCallback
		spell    *core.Spell
		result   *core.SpellResult
		want     *core.Unit
		why      string
	}{
		{core.CallbackOnSpellHitTaken, &core.Spell{Unit: attacker}, &core.SpellResult{Target: &character.Unit},
			attacker, "a hit taken answers the attacker, not the wearer the result names"},
		{core.CallbackOnSpellHitDealt, &core.Spell{Unit: &character.Unit}, &core.SpellResult{Target: attacker},
			attacker, "a hit dealt answers whatever was hit"},
		{core.CallbackOnSpellHitDealt, &core.Spell{Unit: &character.Unit}, &core.SpellResult{Target: &character.Unit},
			enemy, "a hit the character dealt to itself answers the current target"},
		{core.CallbackOnPeriodicDamageDealt, &core.Spell{Unit: &character.Unit}, &core.SpellResult{Target: &character.Unit},
			enemy, "the same for a tick"},
		{core.CallbackOnHealDealt, &core.Spell{Unit: &character.Unit}, &core.SpellResult{Target: attacker},
			enemy, "a heal names an ally, so nothing there says what to damage"},
		{core.CallbackOnCastComplete, &core.Spell{Unit: &character.Unit}, nil,
			enemy, "a cast carries no result at all"},
	} {
		if got := procDamageTarget(character, tc.callback, tc.spell, tc.result); got != tc.want {
			t.Errorf("target = %v, want %v: %s", got.Label, tc.want.Label, tc.why)
		}
	}
}
