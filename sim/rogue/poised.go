package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The Energy filter also reaches the saved Shank action.
func (rogue *Rogue) registerPoised() {
	passive := spelldata.MustFind(1328825)
	if rogue.Level < int32(passive.SpellLevel) {
		return
	}
	rank := passive.Refs()[0]
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
		IntValue: int32(rank.EffectN(1).BasePoints + spellData.Composure.ValueAt(rogue.Talents.Composure)),
	})
}

func (rogue *Rogue) activatePoised(sim *core.Simulation) {
	if rogue.PoisedAura != nil && rogue.CurrentEnergy() >= rogue.MaximumEnergy() {
		rogue.PoisedAura.Activate(sim)
	}
}
