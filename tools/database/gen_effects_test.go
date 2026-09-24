package database

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// A trigger restricted to one named ability is a shape no ProcTypeMask states, so the wording is the
// only evidence there is. The rows below are the client's own, raw and rendered: the store reads the
// raw description and the item generator the rendered tooltip, and both go through this matcher.
func TestNamedAbilityWording(t *testing.T) {
	for _, tc := range []struct {
		named bool
		text  string
		why   string
	}{
		{true, "Your casts of $?s2060[Greater Heal]?s5185[Healing Touch][Greater Heal, Healing Touch] in combat grant up to $1249119s1 increased healing.",
			"Eternal Power 1249118, as the client writes it"},
		{true, "Your casts of Greater Heal in combat grant up to 40 increased healing and up to 13 increased damage for 15s.",
			"the same tooltip, rendered"},
		{true, "Your Backstab has a $m1% chance to cause your next Ambush to not require Stealth.",
			`"has" after a named ability`},
		{true, "Heals from your Earth Shield have a $s1% chance to make your next cast time heal instant cast.",
			`"have" after a named ability`},
		{true, "When you cast Chain Heal or Riptide you gain $s1 charges of Tidal Waves.",
			"a cast of one named ability"},
		{true, "Your Shock spells have a chance to deal extra damage.", "the wording the matcher already read"},
		{false, "When you cast a healing spell, gain Mana equal to $m1% of the base cost of the spell.",
			"no ability is named"},
		{false, "20% chance to regain 100 mana when you cast a Judgement.",
			"a lowercase article keeps the class of spell general"},
		{false, "Multiple casts of Divine Light do not accumulate this shield.",
			"not the caster's own casts, and not a trigger clause"},
		{false, "Your melee attacks have a chance to deal extra damage.",
			"no ability is named"},
		{false, "2% chance on successful spellcast to increase your Spirit by 150 for 15s.",
			"Darkmoon Card: Blue Dragon, which fires off any cast"},
	} {
		if got := procTooltipHints(tc.text).Matches(core.ProcHintNamedAbility); got != tc.named {
			t.Errorf("named ability = %v, want %v for %s:\n  %s", got, tc.named, tc.why, tc.text)
		}
	}
}
