package mage

import (
	"time"

	"github.com/wowsims/forever/sim/core"
)

func (mage *Mage) registerArcaneMissilesSpell() {
	arcaneMissilesRank := spellData.ArcaneMissiles.Highest()
	missileRank := spellData.ArcaneMissilesTriggered.Rank(arcaneMissilesRank.RankNumber())

	// One missile a second for the channel; the row states the channel's length, not its period.
	tickLength := time.Second
	numTicks := int32(arcaneMissilesRank.Duration() / tickLength)

	arcaneMissilesTickSpell := mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: missileRank.ID},
		SpellSchool:    missileRank.SpellSchool(),
		DefenseType:    missileRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagNoOnCastComplete,
		ClassSpellMask: MageSpellArcaneMissilesTick,
		MissileSpeed:   float64(missileRank.Speed),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: missileRank.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, missileRank.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: arcaneMissilesRank.ID},
		SpellSchool:    arcaneMissilesRank.SpellSchool(),
		DefenseType:    arcaneMissilesRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagChanneled | core.SpellFlagAPL,
		ClassSpellMask: MageSpellArcaneMissilesCast,

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(arcaneMissilesRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: arcaneMissilesRank.GCD(),
			},
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "ArcaneMissiles",
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					// The channel holds the Arcane Blast stacks until the last missile is out.
					if mage.ArcaneBlastAura != nil {
						mage.ArcaneBlastAura.Deactivate(sim)
					}
				},
			},
			NumberOfTicks: numTicks,
			TickLength:    tickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				arcaneMissilesTickSpell.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Dot(target).Apply(sim)
		},
	})
}
