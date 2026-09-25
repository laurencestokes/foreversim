package druid

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The spells client 16870's cost modifier names, as the sim knows them. Wrath, Faerie Fire and the
// forms are not among them.
const clearcastingSpells = DruidSpellEntanglingRoots | DruidSpellDemoralizingRoar | DruidSpellHurricane |
	DruidSpellFerociousBite | DruidSpellInsectSwarm | DruidSpellLacerate | DruidSpellPrimalBite | DruidSpellMaul |
	DruidSpellMoonfire | DruidSpellRake | DruidSpellRavage | DruidSpellRip | DruidSpellShred | DruidSpellStarfire |
	DruidSpellSwipe | DruidSpellThorns | DruidSpellHealingTouch | DruidSpellRegrowth | DruidSpellLifebloom |
	DruidSpellRejuvenation | DruidSpellTranquility | DruidSpellSwiftmend

// The client states no rate for 16864 (its chance column of 100 is the "no roll here" convention),
// so this is the 2 procs a minute upstream's port used.
const omenOfClarityPPM = 2.0

// Omen of Clarity (16864), a baseline Balance passive in Forever: melee hits and spells can grant
// Clearcasting (16870), making the next ability that costs something and is in its mask free.
// Moonkin Form (24858) doubles the chance and halves the 10 s cooldown.
//
// ponytail: PPM measured on the paw swing for white hits, the equipped weapon for specials (upstream's
// rule) and the cast time, at least a GCD, for spells; swap in a stated rate if the client ever gets one.
func (druid *Druid) applyOmenOfClarity() {
	clearcasting := spellData.OmenOfClarityTriggered.Highest()

	druid.ClearcastingAura = druid.RegisterAura(core.Aura{
		Label:    "Clearcasting",
		ActionID: core.ActionID{SpellID: clearcasting.ID},
		Duration: clearcasting.Duration(),
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// "Not consumed by Wrath or by spells or abilities that cost no resources."
			if spell.Matches(clearcastingSpells) && spell.Cost != nil && spell.Cost.BaseCost > 0 {
				aura.Deactivate(sim)
			}
		},
	}).AttachSpellMod(core.SpellModConfig{
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		ClassMask:  clearcastingSpells,
		FloatValue: clearcasting.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).Percent(),
	})

	omen := spellData.OmenOfClarity.Highest()
	moonkin := spellData.MoonkinForm.Highest()
	moonkinChance := 1 + moonkin.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_CHANCE_OF_SUCCESS)).Percent()
	moonkinCooldown := 1 + moonkin.Effect(dbcenums.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_PROC_COOLDOWN)).Percent()

	var omenAura *core.Aura
	trigger := spelldata.ProcTrigger(&druid.Character, omen, func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
		druid.ClearcastingAura.Activate(sim)
	}, spelldata.Chance(1))

	trigger.ExtraCondition = func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) bool {
		var seconds float64
		switch {
		case spell.ProcMask.Matches(core.ProcMaskSpellDamage | core.ProcMaskSpellHealing):
			seconds = max(spell.DefaultCast.CastTime, core.GCDDefault).Seconds()
		case spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) || !druid.HasMHWeapon():
			seconds = druid.AutoAttacks.MH().SwingSpeed
		default:
			seconds = druid.GetMHWeapon().SwingSpeed
		}

		chance := omenOfClarityPPM * seconds / 60
		omenAura.Icd.Duration = omen.ICD()
		if druid.MoonkinFormAura.IsActive() {
			chance *= moonkinChance
			omenAura.Icd.Duration = time.Duration(float64(omen.ICD()) * moonkinCooldown)
		}

		return sim.Proc(chance, "Omen of Clarity")
	}

	omenAura = druid.MakeProcTriggerAura(trigger)
}
