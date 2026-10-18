package dbc

import (
	"slices"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

func TestGetRacesFromRaceMask(t *testing.T) {
	for _, tc := range []struct {
		mask int64
		want []proto.Race
	}{
		{1<<1 | 1<<9, []proto.Race{proto.Race_RaceBloodElf, proto.Race_RaceOrc}},
		{1<<32 | 1<<0, []proto.Race{proto.Race_RaceHuman, proto.Race_RaceHighOrderSkyborne}},
		{1<<33 | 1<<10, []proto.Race{proto.Race_RaceWindshaperSkyborne}},
		{-1, nil},
		{0b11<<32 | 0b10_1111_1111, nil},
	} {
		if got := GetRacesFromRaceMask(tc.mask); !slices.Equal(got, tc.want) {
			t.Errorf("mask %#x: %v, want %v", tc.mask, got, tc.want)
		}
	}
}

// Bit n-1 is class n; bits of classes no player can be (6, 10, 12-15) drop out.
func TestGetClassesFromClassMask(t *testing.T) {
	for _, tc := range []struct {
		mask int
		want []proto.Class
	}{
		{1 << 1, []proto.Class{proto.Class_ClassPaladin}},
		{1<<3 | 1<<0, []proto.Class{proto.Class_ClassWarrior, proto.Class_ClassRogue}},
		{1<<10 | 1<<6, []proto.Class{proto.Class_ClassShaman, proto.Class_ClassDruid}},
		{1<<7 | 1<<9 | 1<<11 | 1<<12 | 1<<13 | 1<<14, []proto.Class{proto.Class_ClassMage}},
		{1<<5 | 1<<9, nil},
		{0, nil},
		{-1, nil},
	} {
		if got := GetClassesFromClassMask(tc.mask); !slices.Equal(got, tc.want) {
			t.Errorf("mask %#x: %v, want %v", tc.mask, got, tc.want)
		}
	}
}
