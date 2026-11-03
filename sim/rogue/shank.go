package rogue

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The two unnamed-rank rows learn at 24 and 44; the second supersedes the first in SkillLineAbility.
// The saved OtherAction identity remains stable while the config reads the learned game spell.
// Energy and Venom apply on cast, and Boot Knife on a landed attack, pending in-game verification.
func (rogue *Rogue) registerShank() {
	data := spelldata.MustFind(1329283)
	if rogue.Level < int32(data.SpellLevel) {
		return
	}
	rank := int32(1)
	if higher := spelldata.MustFind(1329549); rogue.Level >= int32(higher.SpellLevel) {
		data = higher
		rank = 2
	}
	energy := data.EffectN(2).Average() + spellData.FlashOfSteel.EffectAt(1).ValueAt(rogue.Talents.FlashOfSteel)
	config := spelldata.SpellConfig(&rogue.Unit, data, spelldata.Melee(core.ProcMaskMeleeOHSpecial))
	config.ActionID = core.ActionID{OtherID: proto.OtherAction_OtherActionShank}
	config.Rank = rank
	config.ClassSpellMask = RogueSpellShank
	config.DamageMultiplier = data.EffectN(1).Percent()
	config.Cast.CD.Duration += time.Duration(spellData.FlashOfSteel.EffectAt(2).ValueAt(rogue.Talents.FlashOfSteel)) * time.Millisecond
	// The tooltip forbids misses; dodge, parry and block still use the ordinary special hit table.
	config.BonusHitPercent = 100
	if rogue.Talents.BootKnife > 0 {
		config.Flags |= SpellFlagBuilder
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		if rogue.VenomAura != nil {
			rogue.VenomAura.Activate(sim)
		}
		result := spell.CalcAndDealDamage(sim, target, rogue.OHWeaponDamage(sim, spell.MeleeAttackPower(target)), spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		rogue.AddEnergy(sim, energy, spell.EnergyMetrics())
		if !result.Landed() {
			spell.IssueRefund(sim)
		} else if rogue.Talents.BootKnife > 0 && sim.Proc(spellData.BootKnife.EffectAt(1).FractionAt(rogue.Talents.BootKnife), "Boot Knife") {
			rogue.AddComboPoints(sim, int32(spellData.BootKnifeTriggered.Highest().EnergizeEffect().Average()), spell.ComboPointMetrics())
		}
	}
	rogue.Shank = rogue.RegisterSpell(config)
}
