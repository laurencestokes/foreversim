package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Forever's Starfire tops out at rank 7, so the max-rank entry moves down from TBC's 8 rather than
// naming a rank the table does not hold. Rank 6 stays registered for downranking.
var StarfireRankMap = spelldata.Ranked(spellData.Starfire.Rank(6).ID, spellData.Starfire.Rank(7).ID)

func (druid *Druid) registerStarfireSpell(rankConfig *spelldata.Spell) {
	spell := druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rankConfig.ID},
		SpellSchool:    rankConfig.SpellSchool(),
		DefenseType:    rankConfig.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		ClassSpellMask: DruidSpellStarfire,
		Flags:          core.SpellFlagAPL,
		Rank:           rankConfig.RankNumber(),
		MaxRange:       float64(rankConfig.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rankConfig.Cost()),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      rankConfig.GCD(),
				CastTime: rankConfig.CastTime(),
			},
		},

		BonusCoefficient: rankConfig.DamageEffect().Coeff(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, rankConfig.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
		},
	})

	druid.Starfire = append(druid.Starfire, spell)
}
