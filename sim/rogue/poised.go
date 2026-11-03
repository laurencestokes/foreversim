package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Composure adds five or ten to the buff's discount in the November 2 development notes.
// The resource filter also reaches the sim's Shank action, which the game files do not name.
func (rogue *Rogue) registerPoised() {
	if rogue.Level < 18 {
		return
	}
	rank := spelldata.MustFind(1328826)
	config := spelldata.AuraConfig(rank)
	config.OnCastComplete = func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
		if spell.Cost != nil && spell.Cost.BaseCost > 0 {
			if _, ok := spell.Cost.ResourceCostImpl.(*core.EnergyCost); ok {
				aura.Deactivate(sim)
			}
		}
	}
	rogue.PoisedAura = rogue.RegisterAura(config).AttachSpellMod(core.SpellModConfig{
		Kind: core.SpellMod_PowerCost_Flat, ResourceType: proto.ResourceType_ResourceTypeEnergy,
		IntValue: int32(rank.EffectN(1).BasePoints) - []int32{0, 5, 10}[rogue.Talents.Composure],
	})
}

func (rogue *Rogue) activatePoised(sim *core.Simulation) {
	if rogue.PoisedAura != nil && rogue.CurrentEnergy() >= rogue.MaximumEnergy() {
		rogue.PoisedAura.Activate(sim)
	}
}
