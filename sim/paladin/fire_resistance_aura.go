package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/buffs"
)

var FireResistanceAuraRankMap = spellData.FireResistanceAura

// Fire Resistance Aura
// https://www.wowhead.com/forever/spell=19900
//
// Gives 60 additional Fire resistance to all party and raid members within 30 yards. Players may
// only have one Aura on them per Paladin at any one time.
func (paladin *Paladin) registerFireResistanceAura() {
	FireResistanceAuraRankMap.RegisterAll(func(row shared.SpellData) {
		aura := buffs.FireResistanceAura(&paladin.Character, true, auraRank(row))
		paladin.registerAuraSpell(row, aura, SpellMaskFireResistanceAura)
	})
}
