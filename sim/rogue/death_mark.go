package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

// The rank-four tooltip (1325758, description variables 1021) gives twelve percent storage per
// combo point and a cap of (156 + 245 per point + four percent of AP per point) times (1 + melee crit).
// The tooltip's $mcrit reads the player's sheet PhysicalCritPercent, before target suppression.
// The cap snapshots that stat at application and remains tooltip-derived pending in-game verification.
func (rogue *Rogue) registerDeathMark() {
	if !rogue.Talents.DeathMark {
		return
	}
	type mark struct{ fraction, cap, stored float64 }
	marks := make([]mark, len(rogue.Env.AllUnits))
	coup := spellData.CoupDeGrace.EffectAt(2).MultiplierAt(rogue.Talents.ImprovedEviscerate)
	rank := spelldata.MustFind(1325758)
	config := spelldata.SpellConfig(&rogue.Unit, rank, spelldata.Melee(core.ProcMaskEmpty))
	config.ClassSpellMask = RogueSpellDeathMark
	config.Flags |= core.SpellFlagIgnoreModifiers | core.SpellFlagIgnoreResists
	config.ExtraCastCondition = rogue.FinisherExtraCastCondition
	config.Dot = spelldata.DotConfig(rank, rank.PeriodicEffect(), spelldata.Label("Death Mark - "+rogue.Label))
	config.Dot.OnTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		m := &marks[target.UnitIndex]
		dot.Spell.CalcAndDealPeriodicDamage(sim, target, min(m.stored, m.cap)*coup, dot.OutcomeTick)
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		points := float64(rogue.ComboPoints())
		marks[target.UnitIndex] = mark{fraction: 0.12 * points * coup, cap: (156 + 245*points + 0.04*spell.MeleeAttackPower(target)*points) * (1 + max(0, rogue.GetStat(stats.PhysicalCritPercent))/100)}
		spell.Dot(target).Apply(sim)
		rogue.ApplyFinisher(sim, spell)
	}
	rogue.DeathMark = rogue.RegisterSpell(config)
	store := func(_ *core.Aura, _ *core.Simulation, spell *core.Spell, result *core.SpellResult) {
		if spell == rogue.DeathMark || result.Damage <= 0 || !rogue.DeathMark.Dot(result.Target).IsActive() {
			return
		}
		m := &marks[result.Target.UnitIndex]
		fraction := m.fraction
		if spell.Matches(RogueSpellGarrote | RogueSpellRupture) {
			fraction *= 2
		}
		m.stored = min(m.cap, m.stored+result.Damage*fraction)
	}
	core.MakePermanent(rogue.RegisterAura(core.Aura{
		Label: "Death Mark - Storage", OnSpellHitDealt: store, OnPeriodicDamageDealt: store,
	}))
}
