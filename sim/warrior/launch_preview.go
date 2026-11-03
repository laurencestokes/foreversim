package warrior

import "github.com/wowsims/forever/sim/core/spelldata"

// The November 2 launch notes supply these values while the game files retain the earlier rows.
// TODO: Remove each guarded copy when the game files carry its launch value.
func launchTalent(rank *spelldata.Spell, points int32) *spelldata.Spell {
	index, old, next := 1, 0.0, 0.0
	switch rank.ID {
	case 12295:
		old, next = 3*float64(points), 5*float64(points)
	case 12163:
		old, next = float64(points), 2*float64(points)
	case 1289682:
		old, next = 4*float64(points), 10*float64(points)
	case 12319:
		old, next = 5*float64(points), 10*float64(points)
	case 12862:
		index, old, next = 3, -3000, -2500*float64(points)
	case 12792:
		old, next = 5*float64(points), []float64{0, 7, 13, 20}[points]
	default:
		return rank
	}
	if rank.EffectN(index).BaseValue() != old {
		return rank
	}
	copy := *rank
	copy.Effects = append([]spelldata.Effect(nil), rank.Effects...)
	copy.EffectN(index).BasePoints = next
	return &copy
}
