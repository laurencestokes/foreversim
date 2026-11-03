package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The top rank, 7, and the cheaper rank 6, which an APL can cast in its place.
var StarfireRankMap = spelldata.Ranked(spellData.Starfire.Rank(6).ID, spellData.Starfire.Rank(7).ID)

func (druid *Druid) registerStarfireSpell(rankConfig *spelldata.Spell) {
	damage := rankConfig.DamageEffect()
	base := damage.Average()
	// The November 2 notes raise base points by about 40%. These are the two pre-tuning rows in
	// builds 70707 and 70753. A refreshed row supplies its own amount without the preview increase.
	if (rankConfig.ID == 9876 && damage.BasePoints == 337) || (rankConfig.ID == 25298 && damage.BasePoints == 381) {
		base += 0.4 * damage.BasePoints
	}

	config := spelldata.SpellConfig(&druid.Unit, rankConfig, spelldata.Magic(core.ProcMaskSpellDamage))
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, base, spell.OutcomeMagicHitAndCrit)
	}

	druid.Starfire = append(druid.Starfire, druid.RegisterSpell(config))
}
