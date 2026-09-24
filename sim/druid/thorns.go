package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var thornsRank = spellData.Thorns.Highest()

// Self-cast Thorns. Reuses the core raid-buff aura: if the Thorns raid buff is selected it is
// already registered (buffs apply before Initialize) and wins, otherwise this registers it.
func (druid *Druid) registerThornsSpell() {
	thornsAura := druid.GetAura("Thorns")
	if thornsAura == nil {
		// TODO: Forever drops Brambles; the core aura still takes a rank for it, so it is
		// pinned to 0 until we know whether the effect moved onto another talent.
		thornsAura = core.ThornsAura(druid.GetCharacter(), 0)
	}

	druid.RegisterSpell(Humanoid, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: thornsRank.ID},
		SpellSchool:    thornsRank.SpellSchool(),
		DefenseType:    thornsRank.DefenseTypeCore(),
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ClassSpellMask: DruidSpellThorns,
		ProcMask:       core.ProcMaskEmpty,
		MaxRange:       float64(thornsRank.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(thornsRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: thornsRank.GCD(),
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			thornsAura.Activate(sim)
		},

		RelatedSelfBuff: thornsAura,
	})
}
