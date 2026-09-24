package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var rakeRank = spellData.Rake.Highest()
var rakeTick = rakeRank.PeriodicEffect()

// Forever's Rake no longer scales with attack power: the client states a flat hit and a flat tick,
// and carries no BonusCoefficientFromAP on either.
func (druid *Druid) registerRakeSpell() {
	druid.Rake = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rakeRank.ID},
		SpellSchool:    rakeRank.SpellSchool(),
		DefenseType:    rakeRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: DruidSpellRake,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:           rakeRank.RankNumber(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   int32(rakeRank.Cost()),
			Refund: rakeRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rakeRank.GCD(),
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		MaxRange:         core.MaxMeleeRange,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:    "Rake",
				Duration: rakeRank.Duration(),
			},
			NumberOfTicks: int32(rakeRank.Duration() / rakeTick.Period()),
			TickLength:    rakeTick.Period(),

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.SnapshotPhysical(target, rakeTick.Average(core.CharacterLevel))
				druid.UpdateBleedPower(druid.Rake, sim, target, true, true)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, periodicTickOutcome(rakeRank, dot))
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, rakeRank.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, spell.ComboPointMetrics())
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
		},

		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				dot := spell.Dot(target)
				return dot.CalcSnapshotDamage(sim, target, dot.OutcomeTick)
			}
			ticks := spell.CalcPeriodicDamage(sim, target, rakeTick.Average(core.CharacterLevel), spell.OutcomeExpectedMagicAlwaysHit)
			attackTable := spell.Unit.AttackTables[target.UnitIndex]
			critChance := spell.PhysicalCritChance(attackTable)
			ticks.Damage *= 1 + critChance*(spell.CritDamageMultiplier(attackTable)-1)
			return ticks
		},
	})

	druid.Rake.ShortName = "Rake"
}

func (druid *Druid) CurrentRakeCost() float64 {
	return druid.Rake.Cost.GetCurrentCost()
}
