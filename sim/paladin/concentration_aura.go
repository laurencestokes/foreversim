package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/buffs"
)

var ConcentrationAuraRankMap = spellData.ConcentrationAura

// Concentration Aura
// https://www.wowhead.com/forever/spell=19746
//
// Gives a 35% chance of ignoring spell interruption when damaged to all party members within 30
// yards. Players may only have one Aura on them per Paladin at any one time.
func (paladin *Paladin) registerConcentrationAura() {
	ConcentrationAuraRankMap.RegisterAll(func(row shared.SpellData) {
		aura := buffs.ConcentrationAura(&paladin.Character, true, auraRank(row))
		paladin.registerAuraSpell(row, aura, SpellMaskConcentrationAura)
	})
}
