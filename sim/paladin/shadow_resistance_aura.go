package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/buffs"
)

var ShadowResistanceAuraRankMap = spellData.ShadowResistanceAura

// Shadow Resistance Aura
// https://www.wowhead.com/forever/spell=19896
//
// Gives 60 additional Shadow resistance to all party and raid members within 30 yards. Players
// may only have one Aura on them per Paladin at any one time.
func (paladin *Paladin) registerShadowResistanceAura() {
	ShadowResistanceAuraRankMap.RegisterAll(func(row shared.SpellData) {
		aura := buffs.ShadowResistanceAura(&paladin.Character, true, auraRank(row))
		paladin.registerAuraSpell(row, aura, SpellMaskShadowResistanceAura)
	})
}
