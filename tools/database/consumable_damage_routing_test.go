package database

import (
	"strings"
	"testing"

	"github.com/wowsims/forever/tools/database/dbc"
)

func TestADamageOverTimeOnAnAreaIsNotRoutedAsDamage(t *testing.T) {
	withDBCInputs(t)
	instance := dbc.GetDBC()

	radiationBomb := instance.Consumables[215168]
	routing := RouteConsumable(&radiationBomb, instance)
	if routing.Supported() || !strings.Contains(routing.Reason(), "lands on an area") {
		t.Errorf("Ez-Thro Radiation Bomb 215168 routes as %q (%s), want refused for 435968's damage over time on an area",
			routing.Kind, routing.Reason())
	}

	highYield := instance.Consumables[215127]
	if routing := RouteConsumable(&highYield, instance); !routing.Supported() || routing.Kind != ConsumableDamage {
		t.Errorf("High-Yield Radiation Bomb 215127 routes as %q (%s), want damage", routing.Kind, routing.Reason())
	}
}
