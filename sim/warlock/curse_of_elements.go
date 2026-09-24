package warlock

import (
	"github.com/wowsims/forever/sim/core"
)

// Forever folded Curse of Shadow into Curse of the Elements: its top rank (1311680) drops every
// magic resistance and raises all magic damage taken by 10%, which is what core's aura already
// does. Malediction is no longer a bonus on the curse - the beta client makes it a flat damage
// modifier on the warlock's own spells - so no ranks are passed in.
func (warlock *Warlock) registerCurseOfElements() {
	rank := spellData.CurseOfTheElements.Highest()

	// Untagged (caster 0), not tagged with our raid index: the APLs ask for aura 27228 on the target,
	// which only an untagged aura answers. Tagged, every warlock but the raid's first never saw its
	// own curse and recast it every GCD (the rankings raid had all four builds at ~10 DPS). One Curse
	// of the Elements per target is also the game's rule, so the warlocks of a raid share it.
	warlock.CurseOfElementsAuras = warlock.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.CurseOfElementsAura(target, 0, 0)
	})

	warlock.CurseOfElements = warlock.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: WarlockSpellCurseOfElements,

		ManaCost: core.ManaCostOptions{FlatCost: int32(rank.Cost())},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: rank.GCD(),
			},
		},

		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				aura := warlock.CurseOfElementsAuras.Get(target)
				warlock.takeCurseSlot(sim, target, aura)
				aura.Activate(sim)
			}

			spell.DealOutcome(sim, result)
		},

		RelatedAuraArrays: warlock.CurseOfElementsAuras.ToMap(),
	})
}
