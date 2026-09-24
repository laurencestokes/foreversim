package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var demoralizingRoarRank = spellData.DemoralizingRoar.Highest()

func (druid *Druid) registerDemoralizingRoarSpell() {
	druid.registerDemoralizingRoarAura()

	druid.DemoralizingRoar = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: demoralizingRoarRank.ID},
		SpellSchool:    demoralizingRoarRank.SpellSchool(),
		DefenseType:    demoralizingRoarRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskEmpty,
		ClassSpellMask: DruidSpellDemoralizingRoar,
		Flags:          core.SpellFlagAPL,

		RageCost: core.RageCostOptions{
			Cost: int32(demoralizingRoarRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: demoralizingRoarRank.GCD(),
			},
			IgnoreHaste: true,
		},

		ThreatMultiplier: 1,
		// Two threat a level, the sim's long-standing value; the client states none.
		FlatThreatBonus: 2 * float64(core.CharacterLevel),

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range druid.Env.Encounter.AllTargetUnits {
				result := spell.CalcAndDealOutcome(sim, aoeTarget, spell.OutcomeMagicHit)
				if result.Landed() {
					druid.DemoralizingRoarAuras.Get(aoeTarget).Activate(sim)
				}
			}
		},

		RelatedAuraArrays: druid.DemoralizingRoarAuras.ToMap(),
	})
}

func (druid *Druid) registerDemoralizingRoarAura() {
	druid.DemoralizingRoarAuras = druid.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		// TODO: Forever drops Feral Aggression and folds it into the base ability - the client's
		// rank 5 states -205 attack power where core's shared aura is TBC's -411. Untalented (0
		// points) until the core aura carries the Forever number.
		return core.DemoralizingRoarAura(target, 0)
	})
}
