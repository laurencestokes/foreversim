package core

import (
	"math"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

// Forever's racials, read from the client (build 1.60.1.70009: SkillLineAbility for each race's
// racial skill line, then SpellEffect, SpellCooldowns, SpellMisc/SpellDuration, SpellAuraOptions
// and SpellEquippedItems for each spell). Every +10 resistance racial is gone, the weapon skill
// racials pay critical strike while that weapon type is held, and each race has a new passive or
// cooldown. Racials that only move, stealth, dispel, regenerate or break crowd control are not
// modelled: nothing the sim measures depends on them. Blood Elf and Draenei are not Forever races
// (CharBaseInfo has no row for either).
func applyRaceEffects(agent Agent) {
	character := agent.GetCharacter()

	switch character.Race {
	case proto.Race_RaceDwarf:
		// Mace Specialization 1259719: +1% crit with one- and two-handed maces.
		applyWeaponCritSpecialization(character, "Mace Specialization", 1259719, 1, proto.WeaponType_WeaponTypeMace)
		// Big Game Hunter 1259721: +5% damage against Beasts.
		applyMobTypeDamageBonus(character, proto.MobType_MobTypeBeast, 1.05)
		registerStoneform(character)
	case proto.Race_RaceGnome:
		applyExpansiveMind(character)
		registerEureka(character)
	case proto.Race_RaceHuman:
		// The Human Spirit 20598: +5% Spirit (Classic's value, not TBC's 10%).
		character.MultiplyStat(stats.Spirit, 1.05)
		// Sword Specialization 20597: +2% crit with one- and two-handed swords. Mace
		// Specialization moved to the Dwarves.
		applyWeaponCritSpecialization(character, "Sword Specialization", 20597, 2, proto.WeaponType_WeaponTypeSword)
	case proto.Race_RaceNightElf:
		// Quickness 20582: +1% dodge and 2% run speed.
		character.PseudoStats.BaseDodgeChance += 0.01
		character.PseudoStats.MovementSpeedMultiplier *= 1.02
		registerElunesLight(character)
	case proto.Race_RaceOrc:
		// Axe Specialization 20574: +1% crit with one- and two-handed axes. Command is gone.
		applyWeaponCritSpecialization(character, "Axe Specialization", 20574, 1, proto.WeaponType_WeaponTypeAxe)
		registerBloodFury(character)
		registerShatterCurse(character)
	case proto.Race_RaceTauren:
		// Endurance 20550: +5% health and +1% hit with attacks and spells.
		character.MultiplyStat(stats.Health, 1.05)
		character.AddStats(stats.Stats{
			stats.PhysicalHitPercent: 1,
			stats.SpellHitPercent:    1,
		})
	case proto.Race_RaceTroll:
		// Beast Slaying 20557: +5% damage against Beasts. Neither ranged weapon
		// specialization is a Forever racial.
		applyMobTypeDamageBonus(character, proto.MobType_MobTypeBeast, 1.05)
		registerBerserking(character)
	case proto.Race_RaceUndead:
		registerTouchOfTheGrave(character)
	case proto.Race_RaceHighOrderSkyborne:
		applySkyborneSharedRacials(character)
		registerReadLeyLine(character)
	case proto.Race_RaceWindshaperSkyborne:
		applySkyborneSharedRacials(character)
		registerSkysight(character)
	}
}

// The passives both Skyborne halves share (racial skill line 2980).
func applySkyborneSharedRacials(character *Character) {
	// Wind Blessed 1259710: +1% attack and casting speed.
	MakePermanent(character.RegisterAura(Aura{
		Label:    "Wind Blessed",
		ActionID: ActionID{SpellID: 1259710},
	}).AttachMultiplyAttackSpeed(1.01).AttachMultiplyCastSpeed(1.01))
	// Elemental Insight 1259707: +5% damage against Elementals.
	applyMobTypeDamageBonus(character, proto.MobType_MobTypeElemental, 1.05)
}

// Read Ley Line 1259705: +100% health and mana regeneration (Energized, 1270842) for 15 sec away
// from a ley line, 2 min cooldown. Left to the APL. The client lists it and Skysight on the shared
// Skyborne skill line; the High Order take this one and the Windshapers Skysight, as wowsims reads it.
func registerReadLeyLine(character *Character) {
	aura := character.RegisterAura(Aura{
		Label:    "Energized",
		ActionID: ActionID{SpellID: 1270842},
		Duration: time.Second * 15,
		OnGain: func(aura *Aura, sim *Simulation) {
			if aura.Unit.HasManaBar() {
				aura.Unit.MultiplyManaRegenSpeed(sim, 2)
			}
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			if aura.Unit.HasManaBar() {
				aura.Unit.MultiplyManaRegenSpeed(sim, 0.5)
			}
		},
	})

	character.RegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: 1259705},
		Flags:    SpellFlagAPL | SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			DefaultCast: Cast{
				GCD: GCDDefault,
			},
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Minute * 2,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
		RelatedSelfBuff: aura,
	})
}

// Skysight 1259686: +10% movement speed (Elemental Blessing, 1259688) for 30 sec away from an
// elemental convergence, 2 min cooldown. Left to the APL.
func registerSkysight(character *Character) {
	aura := character.RegisterAura(Aura{
		Label:    "Elemental Blessing",
		ActionID: ActionID{SpellID: 1259688},
		Duration: time.Second * 30,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.MultiplyMovementSpeed(sim, 1.1)
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.MultiplyMovementSpeed(sim, 1/1.1)
		},
	})

	character.RegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: 1259686},
		Flags:    SpellFlagAPL | SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			DefaultCast: Cast{
				GCD: GCDDefault,
			},
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Minute * 2,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
		RelatedSelfBuff: aura,
	})
}

// The weapon skill racials: crit with both attacks and spells while a weapon of the type is in
// either hand (SpellEquippedItems lists the one- and two-handed subclasses for each).
func applyWeaponCritSpecialization(character *Character, label string, spellID int32, critPercent float64, weaponType proto.WeaponType) {
	hasWeaponEquipped := func() bool {
		for _, weapon := range []*Item{character.MainHand(), character.OffHand()} {
			if weapon != nil && weapon.WeaponType == weaponType {
				return true
			}
		}
		return false
	}

	aura := character.RegisterAura(Aura{
		Label:      label,
		ActionID:   ActionID{SpellID: spellID},
		Duration:   NeverExpires,
		BuildPhase: Ternary(hasWeaponEquipped(), CharacterBuildPhaseBase, CharacterBuildPhaseNone),
	}).AttachStatsBuff(stats.Stats{
		stats.PhysicalCritPercent: critPercent,
		stats.SpellCritPercent:    critPercent,
	})

	if hasWeaponEquipped() {
		MakePermanent(aura)
	}

	character.RegisterItemSwapCallback([]proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand, proto.ItemSlot_ItemSlotOffHand}, func(sim *Simulation, _ proto.ItemSlot) {
		if hasWeaponEquipped() {
			aura.Activate(sim)
		} else {
			aura.Deactivate(sim)
		}
	})
}

// "Increased damage against <creature type>" racials. In Vanilla and TBC these also raise the
// crit multiplier (see AttackTable.CritMultiplier).
func applyMobTypeDamageBonus(character *Character, mobType proto.MobType, multiplier float64) {
	character.Env.RegisterPostFinalizeEffect(func() {
		for _, at := range character.AttackTables {
			if at.Defender.MobType == mobType {
				at.DamageDealtMultiplier *= multiplier
				at.CritMultiplier *= multiplier
			}
		}
	})
}

// Expansive Mind: +5% to the maximum of the class's own resource - rage 1259802, energy
// 1259803, mana 20591 - instead of Classic's 5% Intellect.
func applyExpansiveMind(character *Character) {
	switch {
	case character.HasRageBar():
		character.maxRage *= 1.05
	case character.HasEnergyBar():
		character.maxEnergy *= 1.05
	case character.HasManaBar():
		character.MultiplyStat(stats.Mana, 1.05)
	}
}

// EurekaParts switches the two halves of Eureka! on and off, so a test can measure what each is
// worth (core.EurekaSplit). Both are on everywhere else.
var EurekaParts = struct{ Cost, Damage bool }{true, true}

// Eureka!: the next three of the class's listed abilities within 15 sec cost 10% less and deal 10%
// more damage (periodic damage included); 2 min cooldown. One spell per class, each with its own
// ability list: warrior 1259813, rogue 1259812, mage 1259817, warlock 1259821, priest 1259823; 3
// charges from SpellAuraOptions. Build 70009 set every class's cost cut to 10% (69977 had 40%, 20%,
// 50%, 50% and 15%). The class supplies its list through Character.EurekaSpellMask; one that has
// not gets every class ability.
func registerEureka(character *Character) {
	var spellID int32
	switch character.Class {
	case proto.Class_ClassWarrior:
		spellID = 1259813
	case proto.Class_ClassRogue:
		spellID = 1259812
	case proto.Class_ClassMage:
		spellID = 1259817
	case proto.Class_ClassWarlock:
		spellID = 1259821
	case proto.Class_ClassPriest:
		spellID = 1259823
	default:
		return
	}
	const costReduction = 0.10
	actionID := ActionID{SpellID: spellID}
	mask := character.EurekaSpellMask
	if mask == 0 {
		// A class that has not listed its abilities takes all of them.
		mask = math.MaxInt64
	}
	chargeMask := character.EurekaChargeMask
	if chargeMask == 0 {
		chargeMask = mask
	}

	// A charge goes with each use of a listed ability. Abilities flagged to skip the cast
	// callbacks (a warrior's queued Heroic Strike and Cleave) spend theirs on the hit instead,
	// once per use however many targets it strikes.
	spends := func(spell *Spell) bool {
		return spell.Matches(chargeMask) && !spell.Flags.Matches(SpellFlagPassiveSpell)
	}
	var lastSpell *Spell
	lastUse := time.Duration(-1)

	aura := character.RegisterAura(Aura{
		Label:     "Eureka!",
		ActionID:  actionID,
		Duration:  time.Second * 15,
		MaxStacks: 3,
		OnReset: func(_ *Aura, _ *Simulation) {
			lastSpell, lastUse = nil, -1
		},
		OnCastComplete: func(aura *Aura, sim *Simulation, spell *Spell) {
			if spends(spell) {
				aura.RemoveStack(sim)
			}
		},
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, _ *SpellResult) {
			if !spell.Flags.Matches(SpellFlagNoOnCastComplete) || !spends(spell) {
				return
			}
			if spell == lastSpell && sim.CurrentTime == lastUse {
				return
			}
			lastSpell, lastUse = spell, sim.CurrentTime
			aura.RemoveStack(sim)
		},
		OnStacksChange: func(aura *Aura, sim *Simulation, _ int32, newStacks int32) {
			if newStacks == 0 {
				aura.Deactivate(sim)
			}
		},
	})
	if EurekaParts.Cost {
		aura.AttachSpellMod(SpellModConfig{
			Kind:       SpellMod_PowerCost_Pct,
			ClassMask:  mask,
			FloatValue: -costReduction,
		})
	}
	if EurekaParts.Damage {
		// Periodic damage included: with DynamicDoTs every tick reads the spell's current multiplier,
		// so a covered damage over time effect takes the bonus while Eureka! is up.
		aura.AttachSpellMod(SpellModConfig{
			Kind:       SpellMod_DamageDone_Pct,
			ClassMask:  mask,
			FloatValue: 0.10,
		})
	}

	spell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Minute * 2,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
			aura.SetStacks(sim, aura.MaxStacks)
		},
		RelatedSelfBuff: aura,
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell: spell,
		Type:  CooldownTypeDPS,
	})
}

// Elune's Light 1259799: +10% crit chance for 15 sec, 3 min cooldown.
func registerElunesLight(character *Character) {
	RegisterTemporaryStatsOnUseCD(character, "Elune's Light", stats.Stats{
		stats.PhysicalCritPercent: 10,
		stats.SpellCritPercent:    10,
	}, time.Second*15, SpellConfig{
		ActionID: ActionID{SpellID: 1259799},
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Minute * 3,
			},
		},
	})
}

// Blood Fury 20572: +10% attack power, ranged attack power and spell power (damage and healing)
// for 15 sec, 2 min cooldown. Percentages of everything the orc has, where Classic's paid a share
// of base and Strength-derived attack power only.
func registerBloodFury(character *Character) {
	actionID := ActionID{SpellID: 20572}

	aura := character.NewTemporaryStatMultiplierAura(Aura{
		Label:    "Blood Fury",
		ActionID: actionID,
		Duration: time.Second * 15,
	}, []StatMultiplier{
		{Stat: stats.AttackPower, Multiplier: 1.1},
		{Stat: stats.RangedAttackPower, Multiplier: 1.1},
		{Stat: stats.SpellDamage, Multiplier: 1.1},
		{Stat: stats.HealingPower, Multiplier: 1.1},
	})

	cooldown := Cooldown{
		Timer:    character.NewTimer(),
		Duration: time.Minute * 2,
	}
	aura.Icd = &cooldown

	spell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			CD: cooldown,
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
		RelatedSelfBuff: aura.Aura,
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell:    spell,
		Type:     CooldownTypeDPS,
		BuffAura: aura,
	})
}

// Berserking 20554: +10% attack and casting speed for 10 sec, 3 min cooldown, for every class
// and at any health (Classic scaled 10-30% with health missing). The client has no power cost.
func registerBerserking(character *Character) {
	actionID := ActionID{SpellID: 20554}
	aura := character.RegisterAura(Aura{
		Label:    "Berserking",
		ActionID: actionID,
		Duration: time.Second * 10,
	}).AttachMultiplyAttackSpeed(1.1).AttachMultiplyCastSpeed(1.1)

	spell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Minute * 3,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
		RelatedSelfBuff: aura,
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell: spell,
		Type:  CooldownTypeDPS,
	})
}

// Touch of the Grave: attacks and spells that land have a chance to drain health from the
// target. 1260189 (warrior, paladin, rogue) procs 5% of the time, 1260201 (priest, mage,
// warlock) 10%, both with a 1 sec proc cooldown (SkillLineAbility class masks 11 and 400,
// SpellAuraOptions); damage over time ticks do not roll. The drain, 1260198, is a health leech of
// 5% of the caster's maximum health, Shadow, and flagged unable to crit. It has no flag to ignore
// the caster's damage bonuses, so it takes them, and it can be resisted.
func registerTouchOfTheGrave(character *Character) {
	auraID, procChance := int32(1260189), 0.05
	switch character.Class {
	case proto.Class_ClassPriest, proto.Class_ClassMage, proto.Class_ClassWarlock:
		auraID, procChance = 1260201, 0.10
	}

	drainID := ActionID{SpellID: 1260198}
	healthMetrics := character.NewHealthMetrics(drainID)

	drain := character.RegisterSpell(SpellConfig{
		ActionID:    drainID,
		SpellSchool: SpellSchoolShadow,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskEmpty,
		// A proc off another hit: it must not feed the procs that spawned it.
		Flags: SpellFlagNoOnCastComplete | SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			result := spell.CalcAndDealDamage(sim, target, character.MaxHealth()*0.05, spell.OutcomeMagicHit)
			if result.Landed() && character.HasHealthBar() {
				character.GainHealth(sim, result.Damage, healthMetrics)
			}
		},
	})

	character.MakeProcTriggerAura(ProcTrigger{
		Name:       "Touch of the Grave",
		ActionID:   ActionID{SpellID: auraID},
		Callback:   CallbackOnSpellHitDealt,
		ProcMask:   ProcMaskMelee | ProcMaskRanged | ProcMaskSpellDamage,
		Outcome:    OutcomeLanded,
		ProcChance: procChance,
		ICD:        time.Second,
		Handler: func(sim *Simulation, _ *Spell, result *SpellResult) {
			drain.Cast(sim, result.Target)
		},
	})
}

// Stoneform 20594: -10% Physical damage taken for 8 sec (and poison, disease and bleed
// removal), 3 min cooldown. Classic's +10% armor is gone. Survival cooldowns only fire under a
// health threshold, so it waits to be used by hand or by the APL.
func registerStoneform(character *Character) {
	registerDamageTakenCooldown(character, "Stoneform", ActionID{SpellID: 20594}, 0.9, stats.SchoolIndexPhysical)
}

// Shatter Curse 1299026: -15% magic damage taken for 8 sec and removes a curse, 3 min
// cooldown. Replaces Command.
func registerShatterCurse(character *Character) {
	registerDamageTakenCooldown(character, "Shatter Curse", ActionID{SpellID: 1299026}, 0.85,
		stats.SchoolIndexArcane, stats.SchoolIndexFire, stats.SchoolIndexFrost, stats.SchoolIndexHoly, stats.SchoolIndexNature, stats.SchoolIndexShadow)
}

func registerDamageTakenCooldown(character *Character, label string, actionID ActionID, multiplier float64, schools ...stats.SchoolIndex) {
	aura := character.RegisterAura(Aura{
		Label:    label,
		ActionID: actionID,
		Duration: time.Second * 8,
	})
	for _, school := range schools {
		aura.AttachMultiplicativePseudoStatBuff(&character.PseudoStats.SchoolDamageTakenMultiplier[school], multiplier)
	}

	spell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Minute * 3,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
		RelatedSelfBuff: aura,
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell: spell,
		Type:  CooldownTypeSurvival,
	})
}
