package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/buffs"
)

var DevotionAuraRankMap = spellData.DevotionAura

// Devotion Aura
// https://www.wowhead.com/forever/spell=10293
//
// Gives 735 additional armor to party members within 30 yards. Players may only have one Aura on
// them per Paladin at any one time.
func (paladin *Paladin) registerDevotionAura() {
	DevotionAuraRankMap.RegisterAll(func(row shared.SpellData) {
		aura := buffs.DevotionAuraBuff(&paladin.Character, true, auraRank(row))
		paladin.registerAuraSpell(row, aura, SpellMaskDevotionAura)
	})
}
