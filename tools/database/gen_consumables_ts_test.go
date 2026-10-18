package database

import "testing"

func TestConsumableTSName(t *testing.T) {
	for name, want := range map[string]string{
		"Mighty Rage Potion":                   "MightyRagePotion",
		"R.O.I.D.S.":                           "ROIDS",
		"Kreeg's Stout Beatdown":               "KreegsStoutBeatdown",
		"Scroll of Arcane Protection - Fire I": "ScrollOfArcaneProtectionFireI",
		"Filet o' Flank":                       "FiletOFlank",
		"EZ-Thro Copper Bomb XL":               "EZThroCopperBombXL",
		"M73 Frag Grenade":                     "M73FragGrenade",
		"7 Days":                               "Item7Days",
	} {
		if got := consumableTSName(name); got != want {
			t.Errorf("consumableTSName(%q) = %q, want %q", name, got, want)
		}
	}
}
