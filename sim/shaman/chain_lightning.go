package shaman

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var chainLightningRanks = castableRanks(spellData.ChainLightning)

func (shaman *Shaman) registerChainLightningSpell() {
	shaman.registerWithOverloads(chainLightningRanks, chainLightningOverloadRanks, shaman.newChainLightningSpellConfig)
}

// The cast, with the overload it may trigger; the overload itself, from its own row, where overload is
// nil. Both jump alike, so an overload reaches as many targets as the cast, starting on the target that
// triggered it.
func (shaman *Shaman) newChainLightningSpellConfig(rank *spelldata.Spell, overload *core.Spell) core.SpellConfig {
	config := shaman.newElectricSpellConfig(rank, overload == nil)
	config.ClassSpellMask = SpellMaskChainLightning
	if overload == nil {
		config.ClassSpellMask = SpellMaskChainLightningOverload
	}

	// The effect states the targets the bolt reaches and what each jump keeps.
	damage := rank.DamageEffect()
	maxHits := int32(damage.ChainTargets)
	bounce := float64(damage.ChainAmp)

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		numHits := min(maxHits, shaman.Env.ActiveTargetCount())
		results := make([]*core.SpellResult, numHits)

		// Every hit is rolled before any lands, so a proc off the first target cannot raise the
		// damage of the next.
		curTarget := target
		for hitIndex := range numHits {
			results[hitIndex] = spell.CalcDamage(sim, curTarget, damage.Average(), spell.OutcomeMagicHitAndCrit)
			curTarget = sim.Environment.NextActiveTargetUnit(curTarget)
			spell.DamageMultiplier *= bounce
		}

		// The talent gives both spells one chance to cast a copy on the same target. The launch
		// model rolls on the primary hit once per cast; extra rolls on bounces are unverified.
		canOverload := overload != nil && len(results) > 0 && results[0].Landed()
		for _, result := range results {
			spell.DealDamage(sim, result)
			spell.DamageMultiplier /= bounce
		}

		// The overloads are priced once every hit of the cast has landed, so a debuff charge one of
		// those hits spends, such as Stormstrike's, does not also raise an overload that jumps there.
		if canOverload && sim.Proc(shaman.GetOverloadChance(), "Lightning Overload") {
			overload.Cast(sim, target)
		}
	}

	return config
}
