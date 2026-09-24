package spelldata

import (
	"slices"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

// One refusal per shape, against the rows the resolver's own tests are built from.
func TestItemProcUnsupported(t *testing.T) {
	withRows(t, procRows())

	for _, tc := range []struct {
		spellID      int32
		isWeaponProc bool
		want         []string
		why          string
	}{
		{2200, false, nil, "a chance on an effect of its own and a mask of melee hits"},
		{2000, false, []string{"the trigger carries a class mask"},
			"a column chance and a mask of melee hits, behind a filter the item cannot reproduce"},
		{2300, false, []string{"states no rate"}, "the 101 sentinel with no override to answer it"},
		{2400, false, nil, "the same sentinel, answered by an override into RPPM"},
		{2500, false, []string{"no callback in the proc mask", "no proc mask to measure its rate on"},
			"procs per minute with no hits to measure them on"},
		{2500, true, nil, "the same row on a weapon, where the weapon counts the hits"},
		{2100, true, []string{"states no rate"},
			"the 100 sentinel is no rate on a weapon proc: the game casts it off the hit, with no condition to meet"},
		{2100, false, nil, "the same sentinel on an aura, where the condition is the mask's"},
		{2600, false, []string{"named ability", "ProcTypeMask KILL"},
			"a trigger restricted to one ability, and a bit the decoder models nothing for"},
		{2700, false, []string{"the trigger carries a class mask"},
			"an item worn by every class cannot carry one class's filter"},
		{2800, false, []string{"the trigger carries a class mask"},
			"a family no class owns still names a set of spells the listener cannot be narrowed to"},
	} {
		got := ItemProcUnsupported(Find(tc.spellID), tc.isWeaponProc)
		if !slices.Equal(got, tc.want) {
			t.Errorf("spell %d (weapon proc %v): unsupported = %v, want %v - %s",
				tc.spellID, tc.isWeaponProc, got, tc.want, tc.why)
		}
	}

	if got := ItemProcUnsupported(Find(999999), false); len(got) != 1 || got[0] != "no row in the store" {
		t.Errorf("unsupported = %v, want the missing row named", got)
	}
}

// The sim refuses to register a proc whose row states no callback, and the audit has to see the same
// rows that way or the two would drift apart silently.
func TestItemProcUnsupportedCoversTheRegistrationsOwnRefusal(t *testing.T) {
	withRows(t, procRows())

	row := *Find(2000)
	row.ProcFlags = [2]uint32{0: dbcenums.PROC_FLAG_KILL}
	if core.DecodeProcTypeMask(row.ProcFlags, row.ProcHint).Callback != core.CallbackEmpty {
		t.Fatal("the row the test is built on states a callback after all")
	}

	if !slices.Contains(ItemProcUnsupported(&row, false), "no callback in the proc mask") {
		t.Error("a row the sim would not register was called supported")
	}
}
