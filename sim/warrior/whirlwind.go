package warrior

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var whirlwindRank = spellData.Whirlwind.Highest()

func (warrior *Warrior) registerWhirlwind() {
	// The talent buffs the closest enemy. The sim represents it
	// with the selected target because it has no positions for individual enemies.
	primaryMultiplier := 1.0
	if warrior.Talents.ImprovedWhirlwind > 0 {
		primaryMultiplier += spellData.ImprovedWhirlwind.Rank(warrior.Talents.ImprovedWhirlwind).EffectN(2).Percent()
	}
	calcDamage := func(sim *core.Simulation, target *core.Unit, spell *core.Spell, baseDamage float64) core.SpellResultSlice {
		first := true
		return spell.CalcCleaveDamageWithVariance(sim, target, int32(whirlwindRank.MaxTargets), spell.OutcomeMeleeWeaponSpecialHitAndCrit,
			func(_ *core.Simulation, _ *core.Spell) float64 {
				if first {
					first = false
					// CalcDamage adds flat Physical damage after this callback; its bonus scales too.
					bonus := spell.BonusDamage(warrior.AttackTables[target.UnitIndex])
					return baseDamage + (baseDamage+bonus)*(primaryMultiplier-1)
				}
				return baseDamage
			})
	}
	// The tooltip of 1680 strikes with both melee weapons, and the row states only the one weapon
	// damage effect, so the off-hand strike is a sub-spell on the same row.
	ohConfig := spelldata.SpellConfig(&warrior.Unit, whirlwindRank,
		spelldata.Melee(core.ProcMaskMeleeOHSpecial), spelldata.Proc())
	ohConfig.ActionID = ohConfig.ActionID.WithHand(proto.ItemSlot_ItemSlotOffHand)

	ohConfig.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := warrior.OHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
		calcDamage(sim, target, spell, baseDamage)
		spell.DealBatchedAoeDamage(sim)
	}

	whirlwindOH := warrior.RegisterSpell(ohConfig)

	// TODO: In-game testing required for threat multiplier / threat bonus.
	// The cast is the main-hand strike, so a rotation casts the main-hand copy.
	config := spelldata.SpellConfig(&warrior.Unit, whirlwindRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
	config.ActionID = config.ActionID.WithHand(proto.ItemSlot_ItemSlotMainHand)

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := warrior.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
		results := calcDamage(sim, target, spell, baseDamage)
		warrior.CastNormalizedSweepingStrikesAttack(results, sim)
		spell.DealBatchedAoeDamage(sim)

		if warrior.HasOHWeapon() {
			whirlwindOH.Cast(sim, target)
		}
	}

	warrior.RegisterSpell(config)
}
