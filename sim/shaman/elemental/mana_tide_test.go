package elemental

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func hasManaTide(t *testing.T, talents string) bool {
	player := core.WithSpec(&proto.Player{
		Race:          proto.Race_RaceTroll,
		Class:         proto.Class_ClassShaman,
		Equipment:     &proto.EquipmentSpec{},
		Consumables:   &proto.ConsumesSpec{},
		TalentsString: talents,
		Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{ClassOptions: &proto.ShamanOptions{}}}})
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 100},
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim.Raid.Parties[0].Players[0].GetCharacter().HasAura("Mana Tide Totem (External)")
}

// The talented totem is the shaman's own, not only a party buff set by hand.
func TestManaTideTotemTalentDropsTheTotem(t *testing.T) {
	if hasManaTide(t, DefaultTalents) {
		t.Errorf("Mana Tide Totem without the talent")
	}
	if !hasManaTide(t, DefaultTalents+"001") {
		t.Errorf("no Mana Tide Totem with the talent")
	}
}
