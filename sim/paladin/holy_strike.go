package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
)

var HolyStrikeRankMap = spellData.HolyStrike

// Holy Strike
// https://www.wowhead.com/forever/spell=10333
//
// An instant strike that causes 50% weapon damage plus an additional 93 as Holy damage. 10 sec
// cooldown.
//
// The row states the flat part as its normalized-weapon-damage effect (121) and the percentage as a
// weapon-percent-damage effect (31). The client adds the flat amount to the normalized swing and
// then takes the percentage of the sum, the way it does for Backstab. Build 70009's ranks are
// 25/29/32/36/39/43/46/50%, so rank 8 is 50% of (weapon + 81 to 105).
func (paladin *Paladin) registerHolyStrike(row shared.SpellData) {
	weaponPercent := effectAt(row, 1).Value / 100
	flat := holyStrikeDamage[row.Rank]

	paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: row.SpellID},
		SpellSchool:    row.SpellSchool,
		DefenseType:    row.DefenseType,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		ClassSpellMask: SpellMaskHolyStrike,
		Rank:           row.Rank,
		MaxRange:       core.MaxMeleeRange,

		ManaCost: manaCost(row),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: row.GCD,
			},
			// Client cooldown category 2404 holds Holy Strike and Hammer of the Righteous.
			CD: core.Cooldown{
				Timer:    paladin.sharedTimer(&paladin.holyStrikeTimer),
				Duration: row.Cooldown,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: row.Direct.BonusCoefficient(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := weaponPercent * (spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target)) + sim.Roll(flat[0], flat[1]))
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

// The flat roll per rank at its max level. The table holds the centre of it, truncated
// (spell_data_auto_gen.go reads rank 8 as 93 for the client's 81-105).
var holyStrikeDamage = map[int32][2]float64{
	1: {11, 14}, 2: {15, 20}, 3: {17, 23}, 4: {22, 29}, 5: {32, 40}, 6: {53, 68}, 7: {73, 91}, 8: {81, 105},
}
