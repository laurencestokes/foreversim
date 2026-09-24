package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var mangleRank = spellData.Mangle.Highest()

// Forever ships ONE Mangle - 407995 and 1238069/1238070/1238073 on the Feral Combat line, all with
// ShapeshiftMask [144,0], which is Bear and Dire Bear only. The TBC Cat/Bear split is gone with it,
// so this is the only Mangle registrar and the name still says "Bear" because that is the form it
// is restricted to. Its effects are weapon damage and a flat bonus only (1238073): none of TBC's
// bleed debuff.
func (druid *Druid) registerMangleBearSpell() {
	if !druid.Talents.Mangle {
		return
	}

	druid.MangleAuras = druid.NewEnemyAuraArray(core.MangleAura)

	druid.MangleBear = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: mangleRank.ID},
		SpellSchool:    mangleRank.SpellSchool(),
		DefenseType:    mangleRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		ClassSpellMask: DruidSpellMangleBear,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:           mangleRank.RankNumber(),

		RageCost: core.RageCostOptions{
			Cost:   int32(mangleRank.Cost()),
			Refund: mangleRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: mangleRank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: max(mangleRank.Cooldown(), mangleRank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1.5,
		MaxRange:         core.MaxMeleeRange,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Berserk lets Mangle strike up to 3 targets (417141 tooltip, "$s3 targets").
			numTargets := int32(1)
			if druid.BerserkAura.IsActive() {
				numTargets = min(3, sim.Environment.ActiveTargetCount())
			}

			for i := range numTargets {
				baseDamage := mangleRank.DamageEffect().Average(core.CharacterLevel) + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
				result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

				if i == 0 && !result.Landed() {
					spell.IssueRefund(sim)
				}
				target = sim.Environment.NextActiveTargetUnit(target)
			}

			// Berserk removes Mangle's cooldown (client 417141).
			if druid.BerserkAura.IsActive() {
				spell.CD.Reset()
			}
		},
	})
}
