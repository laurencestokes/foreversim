package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/buffs"
)

var RetributionAuraRankMap = spellData.RetributionAura

// Retribution Aura
// https://www.wowhead.com/forever/spell=10301
//
// Causes 30 Holy damage to any creature that strikes a party member within 30 yards. Players may
// only have one Aura on them per Paladin at any one time.
func (paladin *Paladin) registerRetributionAura() {
	RetributionAuraRankMap.RegisterAll(func(row shared.SpellData) {
		rank := auraRank(row)
		rank.Value = shared.SpellDataMin(row.Direct)
		aura := buffs.RetributionAuraBuff(&paladin.Character, true, rank, 0)
		paladin.registerAuraSpell(row, aura, SpellMaskRetributionAura)
	})
}
