package druid

var ravageRank = spellData.Ravage.Highest()

// TODO: uncalled -- Ravage is live Forever content (spellData.Ravage carries four ranks,
// 6785/6787/9866/9867) but RegisterFeralCatSpells does not wire this up, so druid.Ravage
// stays nil. Before wiring it, settle the multiplier below: the ladder states
// E_WEAPON_PERCENT_DAMAGE 350 for every rank, not the 385 taken from TBC's spell 27005.
// Live Shred has the same disagreement (shred.go hardcodes 2.25 against a stated 155).
func (druid *Druid) registerRavageSpell() {
	panic("To be implemented")

	// The TBC implementation, kept for the port:
	// // 385% weapon damage, which the client states as E_WEAPON_PERCENT_DAMAGE = 384 on spell 27005 -
	// // the same shape as Shred's 224 / 2.25. The flat addend is the rank's own value, scaled by that
	// // multiplier the way the tooltip shows it.
	// const weaponMultiplier = 3.85
	// const highHpCritPercentBonus = 50.0
	//
	// druid.Ravage = druid.RegisterSpell(core.SpellConfig{
	// 	ActionID:         core.ActionID{SpellID: ravageRank.ID},
	// 	CastRequirement:  ravageRank.CastRequirement(),
	// 	SpellSchool:      ravageRank.SpellSchool(),
	// 	DefenseType:      ravageRank.DefenseTypeCore(),
	// 	ProcMask:         core.ProcMaskMeleeMHSpecial,
	// 	ClassSpellMask:   DruidSpellRavage,
	// 	Flags:            core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
	// 	DamageMultiplier: weaponMultiplier,
	// 	ThreatMultiplier: 1,
	// 	BonusCoefficient: 1,
	// 	MaxRange:         core.MaxMeleeRange,
	//
	// 	EnergyCost: core.EnergyCostOptions{
	// 		Cost:   int32(ravageRank.Cost()),
	// 		Refund: 0.8,
	// 	},
	//
	// 	Cast: core.CastConfig{
	// 		DefaultCast: core.Cast{
	// 			GCD: ravageRank.GCD(),
	// 		},
	//
	// 		IgnoreHaste: true,
	// 	},
	//
	// 	ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
	// 		return druid.ProwlAura.IsActive() && !druid.PseudoStats.InFrontOfTarget && !druid.CannotShredTarget
	// 	},
	//
	// 	ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
	// 		if sim.IsExecutePhase90() {
	// 			spell.BonusCritPercent += highHpCritPercentBonus
	// 		}
	//
	// 		baseDamage := ravageRank.DamageEffect().Average(core.CharacterLevel) + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
	// 		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
	//
	// 		if result.Landed() {
	// 			druid.AddComboPoints(sim, 1, spell.ComboPointMetrics())
	// 		} else {
	// 			spell.IssueRefund(sim)
	// 		}
	//
	// 		if sim.IsExecutePhase90() {
	// 			spell.BonusCritPercent -= highHpCritPercentBonus
	// 		}
	// 	},
	//
	// 	ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
	// 		if sim.IsExecutePhase90() {
	// 			spell.BonusCritPercent += highHpCritPercentBonus
	// 		}
	//
	// 		baseDamage := ravageRank.DamageEffect().Average(core.CharacterLevel) + spell.Unit.AutoAttacks.MH().CalculateAverageWeaponDamage(spell.MeleeAttackPower(target))
	// 		result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeExpectedMeleeWeaponSpecialHitAndCrit)
	//
	// 		if sim.IsExecutePhase90() {
	// 			spell.BonusCritPercent -= highHpCritPercentBonus
	// 		}
	//
	// 		return result
	// 	},
	// })
}
