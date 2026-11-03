package rogue

import (
	"slices"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

func (rogue *Rogue) registerSubtletyTalents() {
	rogue.registerOpportunity()
	rogue.registerDirtyDeeds()
	rogue.registerImprovedAmbush()
	rogue.registerInitiative()
	rogue.registerPremeditation()
	rogue.registerSerratedBlades()
	rogue.registerElusiveness()
	rogue.registerGhostlyStrike()
	rogue.registerPreparation()
	rogue.registerHemorrhage()
	rogue.registerDeathMark()
	// Boot Knife runs on Shank. Setup, stealth, movement and detection talents are outside the
	// damage rotation against a boss that does not attack the Rogue.
}

func (rogue *Rogue) registerOpportunity() {
	if rogue.Talents.Opportunity == 0 {
		return
	}

	// The November 2 development notes give the five-rank talent four percent per point.
	rogue.AddStaticMod(core.SpellModConfig{
		Kind: core.SpellMod_DamageDone_Flat, ClassMask: RogueSpellBackstab | RogueSpellGarrote | RogueSpellAmbush | RogueSpellMutilateHit,
		FloatValue: []float64{0, 0.04, 0.08, 0.12, 0.16, 0.2}[rogue.Talents.Opportunity],
	})
}

func (rogue *Rogue) registerInitiative() {
	if rogue.Talents.Initiative == 0 {
		return
	}

	initiativeRank := spellData.Initiative.Rank(rogue.Talents.Initiative)
	initMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: initiativeRank.ID})

	trigger := spelldata.ProcTrigger(&rogue.Character, initiativeRank,
		func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			rogue.AddComboPoints(sim, 1, initMetrics)
		})
	trigger.ClassSpellMask = RogueSpellGarrote | RogueSpellAmbush
	trigger.TriggerImmediately = true
	trigger.Outcome = core.OutcomeLanded
	trigger.RequireDamageDealt = false
	rogue.MakeProcTriggerAura(trigger)
}

var ghostlyStrikeRank = spellData.GhostlyStrike.Highest()

func (rogue *Rogue) registerGhostlyStrike() {
	if !rogue.Talents.GhostlyStrike {
		return
	}

	config := spelldata.SpellConfig(&rogue.Unit, ghostlyStrikeRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
	config.Flags |= SpellFlagBuilder
	config.ClassSpellMask = RogueSpellGhostlyStrike
	config.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), ghostlyStrikeRank.EffectN(4).Percent(), ghostlyStrikeRank.EffectN(1).Percent())

	// The next Ambush or Garrote in the seven-second window can be cast outside Stealth.
	rogue.GhostlyStrikeAura = rogue.RegisterAura(spelldata.AuraConfig(ghostlyStrikeRank)).
		ApplyOnGain(func(_ *core.Aura, _ *core.Simulation) {
			rogue.Ambush.CastRequirement.CasterForm = true
			rogue.Garrote.CastRequirement.CasterForm = true
		}).
		ApplyOnExpire(func(_ *core.Aura, _ *core.Simulation) {
			rogue.Ambush.CastRequirement.CasterForm = false
			rogue.Garrote.CastRequirement.CasterForm = false
		})
	rogue.GhostlyStrikeAura.OnCastComplete = func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
		if spell.Matches(RogueSpellAmbush | RogueSpellGarrote) {
			aura.Deactivate(sim)
		}
	}

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		baseDamage := spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target)) // Data suggests un-normalized

		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

		if result.Landed() {
			rogue.AddComboPoints(sim, int32(ghostlyStrikeRank.ProcEnergizeEffect().Roll(sim)), spell.ComboPointMetrics())
			rogue.GhostlyStrikeAura.Activate(sim)
		} else {
			spell.IssueRefund(sim)
		}
	}

	rogue.GhostlyStrike = rogue.RegisterSpell(config)

	rogue.RegisterItemSwapCallback([]proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand}, func(s *core.Simulation, slot proto.ItemSlot) {
		rogue.GhostlyStrike.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), ghostlyStrikeRank.EffectN(4).Percent(), ghostlyStrikeRank.EffectN(1).Percent())
	})
}

func (rogue *Rogue) registerImprovedAmbush() {
	if rogue.Talents.ImprovedAmbush == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.ImprovedAmbush.Rank(rogue.Talents.ImprovedAmbush))
}

func (rogue *Rogue) registerElusiveness() {
	if rogue.Talents.Elusiveness == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.Elusiveness.Rank(rogue.Talents.Elusiveness))
}

func (rogue *Rogue) registerSerratedBlades() {
	if rogue.Talents.SerratedBlades == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.SerratedBlades.Rank(rogue.Talents.SerratedBlades))
}

func (rogue *Rogue) registerPreparation() {
	if !rogue.Talents.Preparation {
		return
	}

	// Prep purely runs a server-side script, so no client data to reference.

	config := spelldata.SpellConfig(&rogue.Unit, spellData.Preparation.Highest(), spelldata.Flags(core.SpellFlagAPL|core.SpellFlagMCD))
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		if rogue.ColdBlood != nil {
			rogue.ColdBlood.CD.Set(0)
		}
		if rogue.Premeditation != nil {
			rogue.Premeditation.CD.Set(0)
		}
		if rogue.Vanish != nil {
			rogue.Vanish.CD.Set(0)
		}
		if rogue.GhostlyStrike != nil {
			rogue.GhostlyStrike.CD.Set(0)
		}
		if rogue.BladeFlurry != nil {
			rogue.BladeFlurry.CD.Set(0)
		}
		if rogue.AdrenalineRush != nil {
			rogue.AdrenalineRush.CD.Set(0)
		}
	}

	rogue.Preparation = rogue.RegisterSpell(config)

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: rogue.Preparation,
		Type:  core.CooldownTypeDPS,
	})
}

func (rogue *Rogue) registerDirtyDeeds() {
	if rogue.Talents.DirtyDeeds == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.DirtyDeeds.Rank(rogue.Talents.DirtyDeeds))
}

var hemorrhageRank = spellData.Hemorrhage.Highest()

func (rogue *Rogue) registerHemorrhage() {
	if !rogue.Talents.Hemorrhage {
		return
	}

	config := spelldata.SpellConfig(&rogue.Unit, hemorrhageRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
	config.Flags |= SpellFlagBuilder
	config.ClassSpellMask = RogueSpellHemorrhage
	config.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), hemorrhageRank.EffectN(5).Percent(), hemorrhageRank.EffectN(4).Percent())

	hemoAuras := rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		hemoAura := target.GetOrRegisterAura(spelldata.AuraConfig(hemorrhageRank, spelldata.Label(hemorrhageRank.Name+" - "+rogue.Label)))
		// The November 2 development notes extend the caster's bleed bonus to Garrote.
		rank := *hemorrhageRank
		rank.Effects = slices.Clone(hemorrhageRank.Effects)
		for i, flags := range garroteRank.ClassFlags.Mask {
			rank.Effects[2].ClassFlags.Mask[i] |= flags
		}
		spelldata.ParseEffects(nil, hemoAura, &rank,
			spelldata.FromCaster(&rogue.Unit, DDBC_Hemorrhage, DDBC_Total))
		return hemoAura
	})

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		baseDamage := rogue.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		if result.Landed() {
			hemoAuras.Get(target).Activate(sim)
			rogue.AddComboPoints(sim, int32(hemorrhageRank.ProcEnergizeEffect().Roll(sim)), spell.ComboPointMetrics())
		} else {
			spell.IssueRefund(sim)
		}
	}

	rogue.Hemorrhage = rogue.GetOrRegisterSpell(config)

	rogue.RegisterItemSwapCallback([]proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand}, func(s *core.Simulation, slot proto.ItemSlot) {
		rogue.Hemorrhage.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), hemorrhageRank.EffectN(5).Percent(), hemorrhageRank.EffectN(4).Percent())
	})
}

var premeditationRank = spellData.Premeditation.Highest()

func (rogue *Rogue) registerPremeditation() {
	if !rogue.Talents.Premeditation {
		return
	}

	config := spelldata.SpellConfig(&rogue.Unit, premeditationRank, spelldata.Flags(core.SpellFlagAPL|core.SpellFlagMCD))
	premedMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: premeditationRank.ID})

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.AddComboPoints(sim, int32(premeditationRank.ProcEnergizeEffect().Roll(sim)), premedMetrics)
	}

	rogue.Premeditation = rogue.GetOrRegisterSpell(config)

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: rogue.Premeditation,
		Type:  core.CooldownTypeDPS,
	})
}
