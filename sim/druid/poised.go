package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var poisedPassive = spelldata.MustFind(1328829)
var poisedBuff = spelldata.MustFind(1328827)

func (druid *Druid) registerPoised() {
	if !druid.HasEnergyBar() || druid.Level < int32(poisedPassive.SpellLevel) {
		return
	}
	effect := poisedBuff.EffectN(1)
	config := spelldata.AuraConfig(poisedBuff)
	// The preview spends the charge on a qualifying Clearcasting cast as well.
	config.OnCastComplete = func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
		if spell.Cost == nil || spell.Cost.BaseCost <= 0 || !spell.MatchesFlags(effect.ClassFlags) {
			return
		}
		if _, ok := spell.Cost.ResourceCostImpl.(*core.EnergyCost); ok {
			aura.Deactivate(sim)
		}
	}
	druid.PoisedAura = druid.RegisterAura(config).AttachSpellMod(core.SpellModConfig{
		Kind:         core.SpellMod_PowerCost_Flat,
		ResourceType: proto.ResourceType_ResourceTypeEnergy,
		ClassFlags:   effect.ClassFlags,
		IntValue:     int32(effect.Average()),
	})
}

func (druid *Druid) activatePoised(sim *core.Simulation) {
	if druid.PoisedAura != nil && druid.CurrentEnergy() >= druid.MaximumEnergy() {
		druid.PoisedAura.Activate(sim)
	}
}
