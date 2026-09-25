package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/buffs"
)

var FrostResistanceAuraRankMap = spellData.FrostResistanceAura

// Frost Resistance Aura
// https://www.wowhead.com/forever/spell=19898
//
// Gives 60 additional Frost resistance to all party and raid members within 30 yards. Players may
// only have one Aura on them per Paladin at any one time.
func (paladin *Paladin) registerFrostResistanceAura() {
	FrostResistanceAuraRankMap.RegisterAll(func(row shared.SpellData) {
		aura := buffs.FrostResistanceAura(&paladin.Character, true, auraRank(row))
		paladin.registerAuraSpell(row, aura, SpellMaskFrostResistanceAura)
	})
}
