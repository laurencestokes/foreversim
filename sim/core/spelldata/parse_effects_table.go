package spelldata

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/stats"
)

// The misc values the stat and school auras key on.
const (
	miscAllSchools  int32 = 127 // every school bit, which the sim states as one multiplier
	miscMagicSchool int32 = 126 // every school but physical, which is what spell damage covers
	miscArmor       int32 = 1   // A_MOD_RESISTANCE and A_MOD_BASE_RESISTANCE_PCT state armor as school 1
	miscAllStats    int32 = -1  // A_MOD_STAT and A_MOD_TOTAL_STAT_PERCENTAGE: all five at once
)

// The five stats the client counts in its own order, which is what A_MOD_STAT and
// A_MOD_TOTAL_STAT_PERCENTAGE index by.
var clientStats = [5]stats.Stat{stats.Strength, stats.Agility, stats.Stamina, stats.Intellect, stats.Spirit}

// The state one ParseEffects or ParseStatic call shares with the table.
type parser struct {
	unit  *core.Unit
	spell *Spell

	// The character the unit belongs to, for the rows whose helper is a character's rather than a
	// unit's. An aura on a unit with no character of its own - an enemy's debuff - leaves it nil,
	// and those rows are skipped there.
	character *core.Character

	// ParseStatic: there is no aura whose gain hands a Simulation to the attachment, so the rows
	// that need one to act are skipped instead of attached.
	static bool

	// The caller gates every mod through a closure, so a row that cannot be turned back off is
	// skipped instead of attached.
	conditional bool

	// The row states CumulativeAura, so a value follows the aura's stacks. The rows whose value
	// cannot be scaled are skipped while this is set; IgnoreStacks clears it.
	stacking bool
}

// One attachment made for one effect: the sim kind it became, the value in the sim's own units, and
// the single operation that turns it on, off or up. The level is 0 for off, 1 for the plain value and
// N for an aura sitting at N stacks.
type attachment struct {
	kind  string
	value float64
	set   func(sim *core.Simulation, level float64)

	// The operation reads the Simulation it is handed, so it cannot act on an aura that was already
	// up when the parse ran.
	needsSim bool
}

// What the parser does with one effect, by the aura it applies. A row answers nil for an effect it
// cannot take, which sends the effect to Skipped and to the report.
type row func(p *parser, e *Effect, v float64) *attachment

// The auras the parser knows, keyed the way the client files them. A_ADD_FLAT_MODIFIER and
// A_ADD_PCT_MODIFIER name a spell property in their misc value and read the two tables below it;
// the percentage table is handed its value as a fraction.
var auraTable = map[dbcenums.EffectAuraType]row{
	dbcenums.A_ADD_FLAT_MODIFIER: func(p *parser, e *Effect, v float64) *attachment {
		return lookup(flatModTable, dbcenums.SpellModOp(e.Misc))(p, e, v)
	},
	dbcenums.A_ADD_PCT_MODIFIER: func(p *parser, e *Effect, v float64) *attachment {
		return lookup(pctModTable, dbcenums.SpellModOp(e.Misc))(p, e, v/100)
	},

	// The damage the unit deals and takes, by school mask.
	dbcenums.A_MOD_DAMAGE_PERCENT_DONE: func(p *parser, e *Effect, v float64) *attachment {
		return p.schoolMultiplier("damage-dealt", e.Misc, percentMultiplier(v),
			&p.unit.PseudoStats.DamageDealtMultiplier, &p.unit.PseudoStats.SchoolDamageDealtMultiplier)
	},
	dbcenums.A_MOD_DAMAGE_PERCENT_TAKEN: func(p *parser, e *Effect, v float64) *attachment {
		return p.schoolMultiplier("damage-taken", e.Misc, percentMultiplier(v),
			&p.unit.PseudoStats.DamageTakenMultiplier, &p.unit.PseudoStats.SchoolDamageTakenMultiplier)
	},

	// Flat damage done, which the sim keeps as a stat per school.
	dbcenums.A_MOD_DAMAGE_DONE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statsBuff(damageDoneStats(e.Misc), v)
	},

	dbcenums.A_MOD_THREAT: func(p *parser, e *Effect, v float64) *attachment {
		return p.pseudoMultiplier("threat", []*float64{&p.unit.PseudoStats.ThreatMultiplier},
			percentMultiplier(v))
	},

	dbcenums.A_MOD_ATTACK_POWER: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.AttackPower, v)
	},
	dbcenums.A_MOD_RANGED_ATTACK_POWER: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.RangedAttackPower, v)
	},

	dbcenums.A_MOD_STAT: func(p *parser, e *Effect, v float64) *attachment {
		return p.statsBuff(clientStatList(e.Misc), v)
	},
	dbcenums.A_MOD_TOTAL_STAT_PERCENTAGE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statMultiplier(clientStatList(e.Misc), percentMultiplier(v))
	},

	// Hit and crit, which the sim states in percentage points the way the client does.
	dbcenums.A_MOD_HIT_CHANCE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.PhysicalHitPercent, v)
	},
	dbcenums.A_MOD_SPELL_HIT_CHANCE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.SpellHitPercent, v)
	},
	dbcenums.A_MOD_WEAPON_CRIT_PERCENT: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.PhysicalCritPercent, v)
	},
	dbcenums.A_MOD_SPELL_CRIT_CHANCE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.SpellCritPercent, v)
	},
	dbcenums.A_MOD_CRIT_PCT: func(p *parser, e *Effect, v float64) *attachment {
		return p.statsBuff([]stats.Stat{stats.PhysicalCritPercent, stats.SpellCritPercent}, v)
	},

	// The three haste auras. Each multiplies a speed the unit recomputes, so each needs the
	// Simulation an aura's gain hands over.
	dbcenums.A_MOD_CASTING_SPEED_NOT_STACK: func(p *parser, e *Effect, v float64) *attachment {
		return p.speed("cast-speed", (*core.Unit).MultiplyCastSpeed, percentMultiplier(v))
	},
	dbcenums.A_MOD_MELEE_HASTE_3: func(p *parser, e *Effect, v float64) *attachment {
		return p.speed("melee-speed", (*core.Unit).MultiplyMeleeSpeed, percentMultiplier(v))
	},
	dbcenums.A_MOD_ATTACKSPEED: func(p *parser, e *Effect, v float64) *attachment {
		return p.speed("attack-speed", (*core.Unit).MultiplyAttackSpeed, percentMultiplier(v))
	},

	// Healing. 135 is the flat bonus to healing done, 136 the multiplier on it, and 118 the
	// multiplier on healing taken; 115 is the flat bonus to healing taken.
	dbcenums.A_MOD_HEALING_DONE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.HealingPower, v)
	},
	dbcenums.A_MOD_HEALING_DONE_PERCENT: func(p *parser, e *Effect, v float64) *attachment {
		return p.pseudoMultiplier("healing-dealt", []*float64{&p.unit.PseudoStats.HealingDealtMultiplier},
			percentMultiplier(v))
	},
	dbcenums.A_MOD_HEALING_PCT: func(p *parser, e *Effect, v float64) *attachment {
		return p.pseudoMultiplier("healing-taken", []*float64{&p.unit.PseudoStats.HealingTakenMultiplier},
			percentMultiplier(v))
	},
	dbcenums.A_MOD_HEALING: func(p *parser, e *Effect, v float64) *attachment {
		return p.pseudoAdd("healing-taken-flat", &p.unit.PseudoStats.BonusHealingTaken, v)
	},

	// Mana regen, which the client states per five seconds on the mana bar.
	dbcenums.A_MOD_POWER_REGEN: func(p *parser, e *Effect, v float64) *attachment {
		if e.Misc != int32(dbcenums.POWER_MANA) {
			return nil
		}
		return p.statBuff(stats.MP5, v)
	},

	dbcenums.A_MOD_INCREASE_HEALTH: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.Health, v)
	},
	dbcenums.A_MOD_INCREASE_HEALTH_PERCENT: func(p *parser, e *Effect, v float64) *attachment {
		return p.statMultiplier([]stats.Stat{stats.Health}, percentMultiplier(v))
	},

	// Armor is school 1 of the resistance aura; the other bits are the five magic resistances.
	dbcenums.A_MOD_RESISTANCE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statsBuff(resistanceStats(e.Misc), v)
	},

	// The client states the armor ladder twice, once on base armor and once on bonus armor, and the
	// tooltip states armor from items, which is the equipment share the sim scales.
	dbcenums.A_MOD_BASE_RESISTANCE_PCT: func(p *parser, e *Effect, v float64) *attachment {
		if e.Misc&miscArmor == 0 {
			return nil
		}
		return p.equipScaling(stats.Armor, percentMultiplier(v))
	},

	// Avoidance. The sim keeps block as a fraction (sim/core/unit.go:888 adds the rating share as
	// rating/per-percent/100) and dodge and parry as ratings, so each takes its own conversion.
	dbcenums.A_MOD_BLOCK_PERCENT: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.BlockPercent, v/100)
	},
	dbcenums.A_MOD_DODGE_PERCENT: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.DodgeRating, v*core.DodgeRatingPerDodgePercent)
	},
	dbcenums.A_MOD_PARRY_PERCENT: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.ParryRating, v*core.ParryRatingPerParryPercent)
	},

	// The off-hand bonus is a damage modifier on the off-hand's own hits rather than on a family of
	// spells, so it carries a proc mask instead of the effect's class flags.
	dbcenums.A_MOD_OFFHAND_DAMAGE_PCT: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_DamageDone_Pct", core.SpellModConfig{
			Kind:     core.SpellMod_DamageDone_Pct,
			ProcMask: core.ProcMaskMeleeOH,
		}, v/100)
	},

	// One point of expertise is a quarter percent, which is what the rating constant counts in.
	dbcenums.A_MOD_EXPERTISE: func(p *parser, e *Effect, v float64) *attachment {
		return p.statBuff(stats.ExpertiseRating, v*core.ExpertisePerQuarterPercentReduction)
	},

	// How long a crowd control effect lasts on the unit, by the mechanic the misc value names.
	dbcenums.A_MECHANIC_DURATION_MOD: func(p *parser, e *Effect, v float64) *attachment {
		switch e.Misc {
		case int32(dbcenums.MECHANIC_FEAR):
			return p.pseudoMultiplier("fear-duration",
				[]*float64{&p.unit.PseudoStats.FearDurationMultiplier}, percentMultiplier(v))
		case int32(dbcenums.MECHANIC_STUN):
			return p.pseudoMultiplier("stun-duration",
				[]*float64{&p.unit.PseudoStats.StunDurationMultiplier}, percentMultiplier(v))
		}
		return nil
	},
}

// A_ADD_FLAT_MODIFIER: the client's own amount, in the property's units.
var flatModTable = map[dbcenums.SpellModOp]row{
	dbcenums.SPELLMOD_COST: func(p *parser, e *Effect, v float64) *attachment {
		// The client states rage on a 0-1000 bar, so a -30 here is 3 rage. The caster's bar stands in
		// for the power type of the spells the modifier reaches, which the effect does not state.
		if p.unit.HasRageBar() {
			v /= 10
		}
		return p.modInt("SpellMod_PowerCost_Flat", p.modConfig(e, core.SpellMod_PowerCost_Flat), v)
	},
	dbcenums.SPELLMOD_CASTING_TIME: func(p *parser, e *Effect, v float64) *attachment {
		return p.modTime("SpellMod_CastTime_Flat", p.modConfig(e, core.SpellMod_CastTime_Flat), v)
	},
	dbcenums.SPELLMOD_COOLDOWN: func(p *parser, e *Effect, v float64) *attachment {
		return p.modTime("SpellMod_Cooldown_Flat", p.modConfig(e, core.SpellMod_Cooldown_Flat), v)
	},
	dbcenums.SPELLMOD_GLOBAL_COOLDOWN: func(p *parser, e *Effect, v float64) *attachment {
		return p.modTime("SpellMod_GlobalCooldown_Flat", p.modConfig(e, core.SpellMod_GlobalCooldown_Flat), v)
	},
	dbcenums.SPELLMOD_CRITICAL_CHANCE: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_BonusCrit_Percent", p.modConfig(e, core.SpellMod_BonusCrit_Percent), v)
	},
	dbcenums.SPELLMOD_RESIST_MISS_CHANCE: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_BonusHit_Percent", p.modConfig(e, core.SpellMod_BonusHit_Percent), v)
	},
	dbcenums.SPELLMOD_DURATION: func(p *parser, e *Effect, v float64) *attachment {
		return p.modTime("SpellMod_Duration_Flat", p.modConfig(e, core.SpellMod_Duration_Flat), v)
	},
	dbcenums.SPELLMOD_CHARGES: func(p *parser, e *Effect, v float64) *attachment {
		return p.modInt("SpellMod_BuffMaxStacks_Flat", p.modConfig(e, core.SpellMod_BuffMaxStacks_Flat), v)
	},
	dbcenums.SPELLMOD_RANGE: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_Range_Flat", p.modConfig(e, core.SpellMod_Range_Flat), v)
	},
	dbcenums.SPELLMOD_EFFECT1: flatEffectAmount,
	dbcenums.SPELLMOD_EFFECT2: flatEffectAmount,
	dbcenums.SPELLMOD_EFFECT3: flatEffectAmount,
}

// A_ADD_PCT_MODIFIER: a percentage the client states as an integer, handed over as a fraction.
var pctModTable = map[dbcenums.SpellModOp]row{
	dbcenums.SPELLMOD_DAMAGE:      pctDamageDone,
	dbcenums.SPELLMOD_ALL_EFFECTS: pctDamageDone,
	dbcenums.SPELLMOD_DOT: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_DotDamageDone_Pct", p.modConfig(e, core.SpellMod_DotDamageDone_Pct), v)
	},
	dbcenums.SPELLMOD_COST: func(p *parser, e *Effect, v float64) *attachment {
		// The additive bucket, which is where sim/core/spell_mod.go files a 108 with misc 14.
		return p.modFloat("SpellMod_PowerCost_Pct_Add", p.modConfig(e, core.SpellMod_PowerCost_Pct_Add), v)
	},
	dbcenums.SPELLMOD_CASTING_TIME: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_CastTime_Pct", p.modConfig(e, core.SpellMod_CastTime_Pct), v)
	},
	dbcenums.SPELLMOD_COOLDOWN: func(p *parser, e *Effect, v float64) *attachment {
		// The field is the multiplier itself, not a bonus on top of one.
		return p.modMultiplier("SpellMod_Cooldown_Multiplier", p.modConfig(e, core.SpellMod_Cooldown_Multiplier), 1+v)
	},
	dbcenums.SPELLMOD_CRIT_DAMAGE_BONUS: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_CritMultiplier_Flat", p.modConfig(e, core.SpellMod_CritMultiplier_Flat), v)
	},
	dbcenums.SPELLMOD_THREAT: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_ThreatMultiplier_Pct", p.modConfig(e, core.SpellMod_ThreatMultiplier_Pct), v)
	},
	dbcenums.SPELLMOD_DURATION: func(p *parser, e *Effect, v float64) *attachment {
		return p.modFloat("SpellMod_DotBaseDuration_Pct", p.modConfig(e, core.SpellMod_DotBaseDuration_Pct), v)
	},
	dbcenums.SPELLMOD_EFFECT1: pctEffectAmount,
	dbcenums.SPELLMOD_EFFECT2: pctEffectAmount,
	dbcenums.SPELLMOD_EFFECT3: pctEffectAmount,
}

// SPELLMOD_DAMAGE and SPELLMOD_ALL_EFFECTS both raise every hit the spell deals, which the sim
// sums into spell.DamageMultiplierAdditive.
func pctDamageDone(p *parser, e *Effect, v float64) *attachment {
	return p.modFloat("SpellMod_DamageDone_Flat", p.modConfig(e, core.SpellMod_DamageDone_Flat), v)
}

// SPELLMOD_EFFECT1/2/3 name one effect of the target spell, and which effect carries the damage is a
// property of that spell rather than of this one. Reading it would mean resolving every spell the
// class mask names, so the parser assumes the named effect is the damage and says so in the kind: a
// caller whose spell states something else there has to wire that effect by hand.
func flatEffectAmount(p *parser, e *Effect, v float64) *attachment {
	return p.modFloat(effectAssumedKind(e.Misc, "SpellMod_BaseDamage_Flat"),
		p.modConfig(e, core.SpellMod_BaseDamage_Flat), v)
}

func pctEffectAmount(p *parser, e *Effect, v float64) *attachment {
	return p.modFloat(effectAssumedKind(e.Misc, "SpellMod_DamageDone_Flat"),
		p.modConfig(e, core.SpellMod_DamageDone_Flat), v)
}

func effectAssumedKind(misc int32, kind string) string {
	n := 1
	switch dbcenums.SpellModOp(misc) {
	case dbcenums.SPELLMOD_EFFECT2:
		n = 2
	case dbcenums.SPELLMOD_EFFECT3:
		n = 3
	}
	return "effect" + string(rune('0'+n)) + "-assumed-damage " + kind
}

// The row for a misc value, or one that skips: a modifier op the table does not know is reported the
// same way an unknown aura is.
func lookup(table map[dbcenums.SpellModOp]row, op dbcenums.SpellModOp) row {
	if r, ok := table[op]; ok {
		return r
	}
	return skipRow
}

func skipRow(_ *parser, _ *Effect, _ float64) *attachment {
	return nil
}

// The spells a modifier effect names. The client leaves the mask empty on an effect that means the
// caster's whole family, and a spell with no family of its own leaves the mod unrestricted.
func (p *parser) modConfig(e *Effect, kind core.SpellModType) core.SpellModConfig {
	cfg := core.SpellModConfig{Kind: kind, ClassFlags: e.ClassFlags}
	if e.ClassFlags.IsZero() && p.spell.ClassFlags.Family != 0 {
		cfg.ClassFlags = core.ClassFlags{
			Family: p.spell.ClassFlags.Family,
			Mask:   [4]uint32{^uint32(0), ^uint32(0), ^uint32(0), ^uint32(0)},
		}
	}
	return cfg
}

// A spell mod the parse turns on and off. AddDynamicMod rather than AddStaticMod even on a passive:
// a mod that can be switched off is what lets an expiring aura and a conditional share one path.
func (p *parser) mod(kind string, cfg core.SpellModConfig, value float64, scale func(*core.SpellMod, float64)) *attachment {
	mod := p.unit.AddDynamicMod(cfg)
	return &attachment{kind: kind, value: value, set: func(_ *core.Simulation, level float64) {
		if level <= 0 {
			mod.Deactivate()
			return
		}
		scale(mod, level)
		mod.Activate()
	}}
}

func (p *parser) modFloat(kind string, cfg core.SpellModConfig, v float64) *attachment {
	cfg.FloatValue = v
	return p.mod(kind, cfg, v, func(mod *core.SpellMod, level float64) {
		mod.UpdateFloatValue(v * level)
	})
}

func (p *parser) modInt(kind string, cfg core.SpellModConfig, v float64) *attachment {
	cfg.IntValue = int32(v)
	return p.mod(kind, cfg, v, func(mod *core.SpellMod, level float64) {
		mod.UpdateIntValue(int32(v * level))
	})
}

// The client states a time modifier in milliseconds, which is also what Value reports.
func (p *parser) modTime(kind string, cfg core.SpellModConfig, ms float64) *attachment {
	cfg.TimeValue = core.DurationFromMillis(ms)
	return p.mod(kind, cfg, ms, func(mod *core.SpellMod, level float64) {
		mod.UpdateTimeValue(core.DurationFromMillis(ms * level))
	})
}

// A mod whose field is the multiplier itself rather than a bonus on top of one. Whether a second
// stack multiplies again or adds again is not stated anywhere, so a stacking row is skipped the way
// the other multiplier rows are.
func (p *parser) modMultiplier(kind string, cfg core.SpellModConfig, mult float64) *attachment {
	if p.stacking {
		return nil
	}

	cfg.FloatValue = mult
	return p.mod(kind, cfg, mult, func(mod *core.SpellMod, _ float64) {
		mod.UpdateFloatValue(mult)
	})
}

func (p *parser) statBuff(stat stats.Stat, v float64) *attachment {
	return p.statsBuff([]stats.Stat{stat}, v)
}

// Flat stats, which follow the stacks: the attachment keeps what it has handed out and adds the
// difference, so a level of 2 is worth twice the client's amount.
func (p *parser) statsBuff(sts []stats.Stat, v float64) *attachment {
	if len(sts) == 0 {
		return nil
	}

	return additive("stat "+statNames(sts), v, func(sim *core.Simulation, delta float64) {
		bonus := stats.Stats{}
		for _, stat := range sts {
			bonus[stat] = delta
		}
		if sim == nil {
			p.unit.AddStats(bonus)
		} else {
			p.unit.AddStatsDynamic(sim, bonus)
		}
	})
}

// A multiplier on a stat. The sim states it as a dependency, which cannot carry a per-stack value, so
// a stacking row is skipped rather than attached at one stack. The static path has no aura to follow
// and no Simulation to answer a later Refresh with, so a conditional row is skipped there too.
func (p *parser) statMultiplier(sts []stats.Stat, mult float64) *attachment {
	if len(sts) == 0 || p.stacking {
		return nil
	}

	kind := "multiply-stat " + statNames(sts)

	if p.static {
		if p.conditional {
			return nil
		}
		applied := false
		return &attachment{kind: kind, value: mult, set: func(_ *core.Simulation, level float64) {
			if level <= 0 || applied {
				return
			}
			applied = true
			for _, stat := range sts {
				p.unit.MultiplyStat(stat, mult)
			}
		}}
	}

	deps := make([]*stats.StatDependency, len(sts))
	for i, stat := range sts {
		deps[i] = p.unit.NewDynamicMultiplyStat(stat, mult)
	}

	active := false
	return &attachment{kind: kind, value: mult, set: func(sim *core.Simulation, level float64) {
		if (level > 0) == active {
			return
		}
		active = level > 0
		for _, dep := range deps {
			if active {
				p.unit.EnableBuildPhaseStatDep(sim, dep)
			} else {
				p.unit.DisableBuildPhaseStatDep(sim, dep)
			}
		}
	}}
}

// A multiplier on the equipment share of a stat, which is what the client's base-resistance modifier
// states. The character keeps one multiplier per stat, so the value cannot follow stacks, and the
// static path has no Simulation for a later Refresh to hand over.
//
// Expiry undoes the multiplier by dividing by it, so a row that states -100% or worse has no way
// back and is reported rather than applied.
func (p *parser) equipScaling(stat stats.Stat, mult float64) *attachment {
	if p.character == nil || p.stacking || (p.static && p.conditional) || mult <= 0 {
		return nil
	}

	return multiplier("equip-scaling "+stat.StatName(), mult, func(sim *core.Simulation, factor float64) {
		if sim == nil {
			p.character.ApplyEquipScaling(stat, factor)
		} else {
			p.character.ApplyDynamicEquipScaling(sim, stat, factor)
		}
	})
}

// A pseudo-stat the sim multiplies rather than adds. A stack multiplies again, which the parser
// refuses to assume: a stacking row is skipped unless the caller says the value does not follow the
// stacks.
//
// Expiry divides the multiplier back out, so a row that states -100% or worse leaves the field at
// zero or at an infinity and is reported rather than applied.
func (p *parser) pseudoMultiplier(kind string, fields []*float64, mult float64) *attachment {
	if p.stacking || mult <= 0 {
		return nil
	}

	return multiplier(kind, mult, func(_ *core.Simulation, factor float64) {
		for _, field := range fields {
			*field *= factor
		}
	})
}

// A pseudo-stat the sim adds to, which follows the stacks the way a flat stat does.
func (p *parser) pseudoAdd(kind string, field *float64, v float64) *attachment {
	return additive(kind, v, func(_ *core.Simulation, delta float64) {
		*field += delta
	})
}

// One of the speeds the unit recomputes on every change. They take the Simulation the aura's gain
// hands over, so the static path skips them; a stack multiplies the speed again, which the parser
// does not assume.
func (p *parser) speed(kind string, apply func(*core.Unit, *core.Simulation, float64), mult float64) *attachment {
	if p.static || p.stacking {
		return nil
	}

	a := multiplier(kind, mult, func(sim *core.Simulation, factor float64) {
		apply(p.unit, sim, factor)
	})
	a.needsSim = true
	return a
}

// A value added once per level: the attachment keeps what it has handed out and passes on the
// difference.
func additive(kind string, v float64, apply func(sim *core.Simulation, delta float64)) *attachment {
	current := 0.0
	return &attachment{kind: kind, value: v, set: func(sim *core.Simulation, level float64) {
		delta := v*level - current
		if delta == 0 {
			return
		}
		current = v * level
		apply(sim, delta)
	}}
}

// A multiplier applied while the level is up and divided back out when it drops. Only a row that does
// not stack reaches one, so the level is 0 or 1.
func multiplier(kind string, mult float64, apply func(sim *core.Simulation, factor float64)) *attachment {
	active := false
	return &attachment{kind: kind, value: mult, set: func(sim *core.Simulation, level float64) {
		if (level > 0) == active {
			return
		}
		active = level > 0

		factor := mult
		if !active {
			factor = 1 / mult
		}
		apply(sim, factor)
	}}
}

// The damage multipliers the sim keeps once for everything and once per school: the client's mask of
// every school is the first, and a narrower mask is the second, once per school bit it names.
func (p *parser) schoolMultiplier(kind string, mask int32, mult float64, all *float64, perSchool *[stats.SchoolLen]float64) *attachment {
	if mask == miscAllSchools {
		return p.pseudoMultiplier(kind, []*float64{all}, mult)
	}

	var fields []*float64
	for _, index := range schoolIndexes(mask) {
		fields = append(fields, &perSchool[index])
	}
	if len(fields) == 0 {
		return nil
	}
	return p.pseudoMultiplier(kind+"-by-school", fields, mult)
}

// The client states a percentage as an integer, and its sign comes from the data: a -6 is 0.94.
func percentMultiplier(v float64) float64 {
	return 1 + v/100
}

// The stats a client stat index names: -1 is all five, and 0 through 4 are one of them.
func clientStatList(misc int32) []stats.Stat {
	if misc == miscAllStats {
		return clientStats[:]
	}
	if misc < 0 || misc >= int32(len(clientStats)) {
		return nil
	}
	return []stats.Stat{clientStats[misc]}
}

// The stats a flat damage mask names: the sim keeps one stat for physical damage and one for magic,
// plus a stat per magic school, so a mask of every school is the first two together.
func damageDoneStats(mask int32) []stats.Stat {
	switch {
	case mask == miscAllSchools:
		return []stats.Stat{stats.PhysicalDamage, stats.SpellDamage}
	case mask == miscMagicSchool:
		return []stats.Stat{stats.SpellDamage}
	case mask == int32(core.SpellSchoolPhysical):
		return []stats.Stat{stats.PhysicalDamage}
	case isOneSchool(mask):
		return []stats.Stat{core.SpellSchool(mask).SchoolDamage()}
	}
	return nil
}

// The stats a resistance mask names: school 1 is armor, and holy has no resistance stat to take.
func resistanceStats(mask int32) []stats.Stat {
	var out []stats.Stat
	if mask&miscArmor != 0 {
		out = append(out, stats.Armor)
	}

	for bit := int32(core.SpellSchoolHoly); bit <= int32(core.SpellSchoolArcane); bit <<= 1 {
		if mask&bit == 0 {
			continue
		}
		if stat := core.SpellSchool(bit).ResistanceStat(); stat != 0 {
			out = append(out, stat)
		}
	}
	return out
}

// The school bits a mask names, as the positions the sim's per-school arrays index by.
func schoolIndexes(mask int32) []stats.SchoolIndex {
	var out []stats.SchoolIndex
	for bit := int32(1); bit <= int32(core.SpellSchoolArcane); bit <<= 1 {
		if mask&bit != 0 {
			out = append(out, core.SpellSchool(bit).SchoolIndex())
		}
	}
	return out
}

func isOneSchool(mask int32) bool {
	return mask > 0 && mask&(mask-1) == 0 && mask <= int32(core.SpellSchoolArcane)
}

func statNames(sts []stats.Stat) string {
	out := ""
	for i, stat := range sts {
		if i > 0 {
			out += "+"
		}
		out += stat.StatName()
	}
	return out
}
