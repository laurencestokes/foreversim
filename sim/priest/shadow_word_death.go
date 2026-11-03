package priest

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var ShadowWordDeathRankMap = spellData.ShadowWordDeath

func (priest *Priest) registerShadowWordDeathSpell(_ int32, rank *spelldata.Spell) {
	backlash := rank.EffectN(3).Percent()
	baseDamage := rank.DamageEffect().Average()

	config := priest.directDamageSpellConfig(rank, PriestSpellShadowWordDeath)
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		landed := result.Landed()
		killed := target.HasHealthBar() && result.Damage >= target.CurrentHealth()
		// A single-target health encounter tracks the enemy's remaining health in its damage total.
		// Timed encounters and individual enemies in multi-target health encounters model no kills.
		if target.Type == core.EnemyUnit && !target.HasHealthBar() && len(sim.Encounter.AllTargetUnits) == 1 && sim.Encounter.EndFightAtHealth > 0 {
			killed = sim.Encounter.DamageTaken+result.Damage >= sim.Encounter.EndFightAtHealth
		}
		spell.DealDamage(sim, result)
		if landed && !killed {
			priest.RemoveHealth(sim, priest.MaxHealth()*backlash)
		}
	}

	priest.RegisterSpell(config)
}
