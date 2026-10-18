package database

import (
	"testing"

	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
	"github.com/wowsims/forever/tools/database/dbc"
)

func TestAConsumableTakesItsHoliday(t *testing.T) {
	for _, tc := range []struct {
		name       string
		consumable dbc.Consumable
		holiday    string
	}{
		{"RequiredHoliday 335", dbc.Consumable{Id: 990001, RequiredHoliday: 335}, "Love is in the Air"},
		{"a RequiredHoliday with no known name", dbc.Consumable{Id: 990002, RequiredHoliday: 990}, "Holiday 990"},
		{"no RequiredHoliday is sold all year", dbc.Consumable{Id: 990003}, ""},
		{"Fire-toasted Bun, by HolidayConsumables", dbc.Consumable{Id: 23327}, "Midsummer Fire Festival"},
		{"Toasted Smorc, by HolidayConsumables over its RequiredHoliday", dbc.Consumable{Id: 23211, RequiredHoliday: 335}, "Midsummer Fire Festival"},
	} {
		if got := consumableHoliday(&tc.consumable); got != tc.holiday {
			t.Errorf("%s: holiday %q, want %q", tc.name, got, tc.holiday)
		}
	}

	withDBCInputs(t)
	instance := dbc.GetDBC()
	for _, id := range []int{23327, 23211} {
		if got, want := shippedConsumable(t, instance, id).Holiday, HolidayConsumables[int32(id)]; got != want {
			t.Errorf("%d: shipped with holiday %q, want %q", id, got, want)
		}
	}
}

// Nutritious Food hands the amount of its periodic trigger to Well Fed, whose own row states
// another.
func TestNutritiousFoodListsTheAmountItHandsWellFed(t *testing.T) {
	withDBCInputs(t)
	instance := dbc.GetDBC()

	for _, tc := range []struct {
		name string
		id   int
		stat stats.Stat
	}{
		{"Smoked Desert Dumplings, 1248401 to Well Fed 1248422", 20452, stats.Strength},
		{"Swiftstrike Steak, 1248399 to Well Fed 1248420", 250072, stats.Agility},
		{"Lobster Stew, 1249516 to Well Fed 1249520", 13933, stats.SpellDamage},
	} {
		consumable := shippedConsumable(t, instance, tc.id)
		food := spelldata.MustFind(consumable.ItemEffects[0].BuffId)
		trigger := food.FirstAura(dbcenums.A_PERIODIC_TRIGGER_SPELL_WITH_VALUE)
		want := trigger.BaseValue()
		if want == 0 {
			t.Fatalf("%s: %d states no amount it hands on", tc.name, food.ID)
		}
		wellFed := spelldata.MustFind(trigger.TriggerID)
		for i := range wellFed.Effects {
			if own := wellFed.Effects[i].BaseValue(); own == want {
				t.Fatalf("%s: Well Fed %d states the handed amount %v itself, so the test cannot tell the two apart", tc.name, wellFed.ID, own)
			}
		}
		if got := stats.FromProtoArray(consumable.Stats)[tc.stat]; got != want {
			t.Errorf("%s: %v %v, want %v", tc.name, got, tc.stat, want)
		}
	}
}
