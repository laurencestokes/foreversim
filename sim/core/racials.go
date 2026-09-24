package core

import (
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

// Forever's racials, read from the client (build 1.60.1.69977: SkillLineAbility for each race's
// racial skill line, then SpellEffect, SpellCooldowns, SpellMisc/SpellDuration, SpellAuraOptions
// and SpellEquippedItems for each spell). Every +10 resistance racial is gone, the weapon skill
// racials pay critical strike while that weapon type is held, and each race has a new passive or
// cooldown. Racials that only move, stealth, dispel, regenerate or break crowd control are not
// modelled: nothing the sim measures depends on them.
//
// Blood Elf and Draenei are not Forever races (CharBaseInfo has no row for either); their TBC
// racials stay below only so an old saved setting still builds.
func applyRaceEffects(agent Agent) {
	character := agent.GetCharacter()

	switch character.Race {
	case proto.Race_RaceBloodElf:
		character.stats[stats.ArcaneResistance] += 5
		character.stats[stats.FireResistance] += 5
		character.stats[stats.FrostResistance] += 5
		character.stats[stats.NatureResistance] += 5
		character.stats[stats.ShadowResistance] += 5

		var actionID ActionID

		var resourceMetrics *ResourceMetrics = nil
		if resourceMetrics == nil {
			if character.HasEnergyBar() {
				actionID = ActionID{SpellID: 25046}
				resourceMetrics = character.NewEnergyMetrics(actionID)
			} else if character.HasManaBar() {
				actionID = ActionID{SpellID: 28730}
				resourceMetrics = character.NewManaMetrics(actionID)
			}
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
			ApplyEffects: func(sim *Simulation, _ *Unit, spell *Spell) {
				if spell.Unit.HasEnergyBar() {
					spell.Unit.AddEnergy(sim, 10, resourceMetrics)
				} else if spell.Unit.HasManaBar() {
					spell.Unit.AddMana(sim, 10, resourceMetrics)
				}
			},
		})

		character.AddMajorCooldown(MajorCooldown{
			Spell:    spell,
			Type:     CooldownTypeDPS,
			Priority: CooldownPriorityLow,
			ShouldActivate: func(sim *Simulation, character *Character) bool {
				if spell.Unit.HasEnergyBar() {
					return character.CurrentEnergy() <= character.maxEnergy-10
				}
				return true
			},
		})
	case proto.Race_RaceDraenei:
		character.stats[stats.ShadowResistance] += 10

		switch character.Class {
		case proto.Class_ClassHunter, proto.Class_ClassPaladin, proto.Class_ClassWarrior:
			MakePermanent(DraneiRacialAura(character, false))
		case proto.Class_ClassMage, proto.Class_ClassPriest, proto.Class_ClassShaman:
			MakePermanent(DraneiRacialAura(character, true))
		}

		character.RegisterSpell(SpellConfig{
			ActionID:    ActionID{SpellID: 28880},
			Flags:       SpellFlagAPL | SpellFlagHelpful | SpellFlagIgnoreModifiers,
			ProcMask:    ProcMaskSpellHealing,
			SpellSchool: SpellSchoolHoly,
			DefenseType: DefenseTypeMagic,

			MaxRange: 40,

			Cast: CastConfig{
				DefaultCast: Cast{
					CastTime: time.Millisecond * 1500,
				},
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Second * 15,
				},
			},

			DamageMultiplier: 1.0,
			ThreatMultiplier: 1.0,

			Hot: DotConfig{
				Aura: Aura{
					Label: "Gift of the Naaru" + character.Label,
				},
				NumberOfTicks:       5,
				TickLength:          time.Second * 3,
				AffectedByCastSpeed: false,
				OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
					healValue := float64((35.0 + 15*CharacterLevel) / dot.ExpectedTickCount())
					dot.Spell.CalcAndDealPeriodicHealing(sim, target, healValue, dot.OutcomeTick)
				},
			},

			ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
				spell.Hot(target).Activate(sim)
			},
		})
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
		// Quickness 20582: +1% dodge (and 2% run speed).
		character.PseudoStats.BaseDodgeChance += 0.01
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
	case proto.Race_RaceSkyborneHighOrder, proto.Race_RaceSkyborneWindshaper:
		// Both halves share one racial skill line (2980); the faction choice changes nothing
		// the sim measures.
		// Wind Blessed 1259710: +1% attack and casting speed.
		MakePermanent(character.RegisterAura(Aura{
			Label:    "Wind Blessed",
			ActionID: ActionID{SpellID: 1259710},
		}).AttachMultiplyAttackSpeed(1.01).AttachMultiplyCastSpeed(1.01))
		// Elemental Insight 1259707: +5% damage against Elementals.
		applyMobTypeDamageBonus(character, proto.MobType_MobTypeElemental, 1.05)
	}
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

// Eureka!: the next three of the class's listed abilities within 15 sec cost less and deal 10%
// more damage (periodic damage included); 2 min cooldown. One spell per class, each with its own
// ability list and cost cut: warrior 1259813 (40%), rogue 1259812 (20%), mage 1259817 (50%),
// warlock 1259821 (50%), priest 1259823 (15%); 3 charges from SpellAuraOptions. The class supplies
// its list through Character.EurekaSpellMask; a class that has not is left without it.
func registerEureka(character *Character) {
	if character.EurekaSpellMask == 0 {
		return
	}

	var spellID int32
	var costReduction float64
	switch character.Class {
	case proto.Class_ClassWarrior:
		spellID, costReduction = 1259813, 0.40
	case proto.Class_ClassRogue:
		spellID, costReduction = 1259812, 0.20
	case proto.Class_ClassMage:
		spellID, costReduction = 1259817, 0.50
	case proto.Class_ClassWarlock:
		spellID, costReduction = 1259821, 0.50
	case proto.Class_ClassPriest:
		spellID, costReduction = 1259823, 0.15
	default:
		return
	}
	actionID := ActionID{SpellID: spellID}
	mask := character.EurekaSpellMask
	chargeMask := character.EurekaChargeMask
	if chargeMask == 0 {
		chargeMask = mask
	}

	// A charge goes with each use of a listed ability. Abilities flagged to skip the cast
	// callbacks (a warrior's queued Heroic Strike and Cleave) spend theirs on the hit instead,
	// once per use however many targets it strikes.
	var lastSpell *Spell
	lastUse := time.Duration(-1)
	spendCharge := func(aura *Aura, sim *Simulation, spell *Spell) {
		if !spell.Matches(chargeMask) || spell.Flags.Matches(SpellFlagPassiveSpell) {
			return
		}
		if spell == lastSpell && sim.CurrentTime == lastUse {
			return
		}
		lastSpell, lastUse = spell, sim.CurrentTime
		aura.RemoveStack(sim)
	}

	aura := character.RegisterAura(Aura{
		Label:     "Eureka!",
		ActionID:  actionID,
		Duration:  time.Second * 15,
		MaxStacks: 3,
		OnReset: func(_ *Aura, _ *Simulation) {
			lastSpell, lastUse = nil, -1
		},
		OnCastComplete: spendCharge,
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, _ *SpellResult) {
			if spell.Flags.Matches(SpellFlagNoOnCastComplete) {
				spendCharge(aura, sim, spell)
			}
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
		// Periodic damage included: a damage over time effect snapshots the multiplier when applied.
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

// Blood Fury 20572: +10% attack power, ranged attack power and spell power for 15 sec, 2 min
// cooldown. Percentages of everything the orc has, where Classic's paid a share of base and
// Strength-derived attack power only.
func registerBloodFury(character *Character) {
	actionID := ActionID{SpellID: 20572}
	aura := character.RegisterAura(Aura{
		Label:    "Blood Fury",
		ActionID: actionID,
		Duration: time.Second * 15,
	}).AttachStatDependency(character.NewDynamicMultiplyStat(stats.AttackPower, 1.1)).
		AttachStatDependency(character.NewDynamicMultiplyStat(stats.RangedAttackPower, 1.1)).
		AttachStatDependency(character.NewDynamicMultiplyStat(stats.SpellDamage, 1.1))

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
		},
		RelatedSelfBuff: aura,
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell: spell,
		Type:  CooldownTypeDPS,
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
// warlock) 10%, both with a 1 sec proc cooldown; the drain, 1260198, is a health leech of 5% of
// the caster's maximum health. Taken to be Shadow damage that can be resisted; whether it can
// crit is unknown, so it does not.
func registerTouchOfTheGrave(character *Character) {
	procChance := 0.05
	switch character.Class {
	case proto.Class_ClassPriest, proto.Class_ClassMage, proto.Class_ClassWarlock:
		procChance = 0.10
	}

	actionID := ActionID{SpellID: 1260198}
	healthMetrics := character.NewHealthMetrics(actionID)

	drain := character.RegisterSpell(SpellConfig{
		ActionID:    actionID,
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

	icd := Cooldown{
		Timer:    character.NewTimer(),
		Duration: time.Second,
	}

	MakePermanent(character.RegisterAura(Aura{
		Label:    "Touch of the Grave",
		ActionID: actionID,
		OnSpellHitDealt: func(_ *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if spell == drain || !result.Landed() || !icd.IsReady(sim) {
				return
			}
			if sim.Proc(procChance, "Touch of the Grave") {
				icd.Use(sim)
				drain.Cast(sim, result.Target)
			}
		},
	}))
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
