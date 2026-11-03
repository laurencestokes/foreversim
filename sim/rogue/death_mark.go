package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The game files' rank-four tooltip (1325758, description variables 1021) states twelve percent
// stored per combo point and a cap of 156 + 245 per point + four percent of Attack Power per point.
// The critical-strike contribution to that cap is a server script with no formula in the available
// rows. This preview adds zero for that term. It is a conservative cap assumption, not a lower
// bound on the rotation's DPS. The November 2 notes add doubled Garrote and Rupture storage and
// give a twenty-yard range; the captured rank's range is twenty-five yards.
func (rogue *Rogue) registerDeathMark() {
	if !rogue.Talents.DeathMark {
		return
	}
	type mark struct{ fraction, cap, stored float64 }
	marks := make([]mark, len(rogue.Env.AllUnits))
	coup := spellData.ImprovedEviscerate.MultiplierAt(rogue.Talents.ImprovedEviscerate)
	rank := spelldata.MustFind(1325758)
	config := spelldata.SpellConfig(&rogue.Unit, rank, spelldata.Melee(core.ProcMaskEmpty))
	config.ClassSpellMask = RogueSpellDeathMark
	config.Flags |= core.SpellFlagIgnoreModifiers | core.SpellFlagIgnoreResists
	config.MaxRange = 20
	config.ExtraCastCondition = rogue.FinisherExtraCastCondition
	config.Dot = spelldata.DotConfig(rank, rank.PeriodicEffect(), spelldata.Label("Death Mark - "+rogue.Label))
	config.Dot.OnTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		m := &marks[target.UnitIndex]
		dot.Spell.CalcAndDealPeriodicDamage(sim, target, min(m.stored, m.cap)*coup, dot.OutcomeTick)
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		points := float64(rogue.ComboPoints())
		marks[target.UnitIndex] = mark{fraction: 0.12 * points * coup, cap: 156 + 245*points + 0.04*spell.MeleeAttackPower(target)*points}
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
