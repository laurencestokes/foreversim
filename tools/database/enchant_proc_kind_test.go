package database

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

func TestEnchantProcKind(t *testing.T) {
	withDBCInputs(t)

	for _, tc := range []struct {
		effectID int
		name     string
		want     proto.EnchantProcKind
	}{
		{1900, "Enchant Weapon - Crusader", proto.EnchantProcKind_EnchantProcKindCombat},
		{1898, "Enchant Weapon - Lifestealing", proto.EnchantProcKind_EnchantProcKindCombat},
		{803, "Enchant Weapon - Fiery Weapon", proto.EnchantProcKind_EnchantProcKindCombat},
		{8216, "Enchant Weapon - Insight", proto.EnchantProcKind_EnchantProcKindCombat},
		{1704, "Thorium Shield Spike", proto.EnchantProcKind_EnchantProcKindCombat},
		{8220, "Enchant Chest - Absorption", proto.EnchantProcKind_EnchantProcKindSurvival},
		{8721, "Enchant Weapon - Recovery", proto.EnchantProcKind_EnchantProcKindSurvival},
		{2613, "Enchant Gloves - Threat", proto.EnchantProcKind_EnchantProcKindNone},
		{1887, "Enchant Bracer - Greater Agility", proto.EnchantProcKind_EnchantProcKindNone},
		{2586, "Falcon's Call", proto.EnchantProcKind_EnchantProcKindNone},
		{911, "Enchant Boots - Minor Speed", proto.EnchantProcKind_EnchantProcKindNone},
		{8695, "Enchant Gloves - Lotus Claw", proto.EnchantProcKind_EnchantProcKindNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := enchantByName(t, tc.effectID, tc.name).ProcKind; got != tc.want {
				t.Errorf("proc kind %v, want %v", got, tc.want)
			}
		})
	}
}
