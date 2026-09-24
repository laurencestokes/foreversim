package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

// Forever's Blizzard is an area trigger that casts a damage spell (1279949 at rank 6) every second.
func (mage *Mage) registerBlizzardSpell() {
	blizzardRank := spellData.Blizzard.Highest()
	// The damage is the spell BlizzardTriggered casts each tick, at the same rank; the tick length is
	// Blizzard's own periodic dummy.
	blizzardTickSpell := spellData.BlizzardTriggered.Rank(blizzardRank.RankNumber())
	blizzardTick := blizzardTickSpell.DamageEffect()
	tickLength := blizzardRank.Effect(dbcenums.A_PERIODIC_DUMMY, 0).Period()
	blizzardActionId := core.ActionID{SpellID: blizzardRank.ID}

	// Improved Blizzard's chill, a separate spell so Fingers of Frost can roll on it.
	var improvedBlizzard *core.Spell
	if mage.Talents.ImprovedBlizzard > 0 {
		improvedBlizzardRank := spellData.ImprovedBlizzardTriggered.Highest()
		improvedBlizzard = mage.RegisterSpell(core.SpellConfig{
			ActionID:       core.ActionID{SpellID: improvedBlizzardRank.ID},
			SpellSchool:    core.SpellSchoolFrost,
			DefenseType:    core.DefenseTypeMagic,
			ProcMask:       core.ProcMaskSpellDamageProc,
			Flags:          core.SpellFlagNoLogs | core.SpellFlagNoMetrics | core.SpellFlagNoOnCastComplete,
			ClassSpellMask: MageSpellImprovedBlizzard,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHitNoHitCounter)
			},
		})
	}

	blizzardTickCast := mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: blizzardTickSpell.ID},
		SpellSchool:    blizzardRank.SpellSchool(),
		DefenseType:    blizzardRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagNoOnCastComplete,
		ClassSpellMask: MageSpellBlizzard,

		DamageMultiplier: 1,
		BonusCoefficient: blizzardTick.Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			results := spell.CalcAndDealAoeDamage(sim, blizzardTick.Average(core.CharacterLevel), spell.OutcomeMagicHit)
			if improvedBlizzard == nil {
				return
			}
			for _, result := range results {
				if result.Landed() {
					improvedBlizzard.Cast(sim, result.Target)
				}
			}
		},
	})

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       blizzardActionId,
		SpellSchool:    blizzardRank.SpellSchool(),
		DefenseType:    blizzardRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagChanneled | core.SpellFlagAPL,
		ClassSpellMask: MageSpellBlizzard,
		ManaCost: core.ManaCostOptions{
			FlatCost: int32(blizzardRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: blizzardRank.GCD(),
			},
		},
		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label:    "Blizzard",
				ActionID: blizzardActionId,
			},
			NumberOfTicks: int32(blizzardRank.Duration() / tickLength),
			TickLength:    tickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				blizzardTickCast.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.AOEDot().Apply(sim)
		},
	})
}
