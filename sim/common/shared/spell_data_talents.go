package shared

import "fmt"

// Reads the ladder by points spent. Rank 0 is untaken and answers 0, where ByRank would panic.
//
//	spellData.NaturesReach.ValueAt(2)   // 20, the client's number as it stands
func (t SpellDataTableOf[T]) ValueAt(rank int32) float64 {
	return ladderValue(t, rank, nil)
}

// The client states a percentage as an integer: 16, not 0.16.
//
//	spellData.Bloodthrill.FractionAt(5)   // 0.10, from 10
func (t SpellDataTableOf[T]) FractionAt(rank int32) float64 {
	return t.ValueAt(rank) / 100
}

// The sign comes from the data: Improved Righteous Fury states -2/-4/-6, so rank 3 gives 0.94.
//
//	spellData.ImprovedBloodrage.MultiplierAt(2)   // 1.5, from 50
func (t SpellDataTableOf[T]) MultiplierAt(rank int32) float64 {
	return 1 + t.FractionAt(rank)
}

// The client states rage and energy on a 0-1000 bar: a -30 cost modifier is 3 rage, Charge's
// energize of 150 is 15.
//
//	spellData.ImprovedCharge.TenthsAt(2)   // 6 rage, from 60
//	spellData.BoundlessRage.TenthsAt(3)    // 30 rage, from 300
func (t SpellDataTableOf[T]) TenthsAt(rank int32) float64 {
	return t.ValueAt(rank) / 10
}

// The client's proc chance as a fraction, which is the form a ProcTrigger takes. Rank 0 is untaken
// and answers 0.
func (t SpellDataTableOf[T]) ProcChanceAt(rank int32) float64 {
	return ladderValue(t, rank, func(row SpellData) float64 { return float64(row.ProcChance) }) / 100
}

// The procs per minute WithSpellDataPPM gave the rank, for NewLegacyPPMManager and its static form.
// Rank 0 is untaken and answers 0.
func (t SpellDataTableOf[T]) PPMAt(rank int32) float64 {
	return ladderValue(t, rank, func(row SpellData) float64 { return row.PPM })
}

// For the case Effect cannot serve: two effects sharing an aura and misc value, as Tactical Mastery's
// two threat modifiers do. The index is the client's EffectIndex, not the slice position.
func (t SpellDataTableOf[T]) EffectAt(index int32) SpellDataEffectLadder[T] {
	pick := func(row SpellData) float64 {
		for _, e := range row.Effects {
			if e.Index == index {
				return e.Value
			}
		}
		panic(fmt.Sprintf("spell %d rank %d has no effect at index %d", row.SpellID, row.Rank, index))
	}
	return SpellDataEffectLadder[T]{table: t, pick: pick}
}

func (t SpellDataTableOf[T]) Effect(aura SpellDataAura, misc int32) SpellDataEffectLadder[T] {
	pick := func(row SpellData) float64 { return row.Effect(aura, misc).Value }
	return SpellDataEffectLadder[T]{table: t, pick: pick}
}

type SpellDataEffectLadder[T SpellDataRanked] struct {
	table SpellDataTableOf[T]
	pick  func(SpellData) float64
}

// The picked effect's value by points spent; rank 0 answers 0.
//
//	spellData.ShieldSpecialization.EffectAt(1).ValueAt(3)   // the chance effect's 60
func (l SpellDataEffectLadder[T]) ValueAt(rank int32) float64 {
	return ladderValue(l.table, rank, l.pick)
}

// The picked effect's percentage as a fraction.
//
//	spellData.BloodCraze.EffectAt(1).FractionAt(3)   // 0.2, from 20
func (l SpellDataEffectLadder[T]) FractionAt(rank int32) float64 {
	return l.ValueAt(rank) / 100
}

// 1 plus the picked effect's fraction, with the sign the data gives it.
//
//	spellData.DualWieldSpecialization.EffectAt(1).MultiplierAt(3)   // 1.6, from 60
func (l SpellDataEffectLadder[T]) MultiplierAt(rank int32) float64 {
	return 1 + l.FractionAt(rank)
}

// The picked effect in rage or energy, from the client's 0-1000 bar.
//
//	spellData.RagingBlows.EffectAt(1).TenthsAt(1)   // -2 rage on Cleave, from -20
func (l SpellDataEffectLadder[T]) TenthsAt(rank int32) float64 {
	return l.ValueAt(rank) / 10
}

func ladderValue[T SpellDataRanked](table SpellDataTableOf[T], rank int32, pick func(SpellData) float64) float64 {
	if rank <= 0 {
		return 0
	}

	row, ok := any(table.ByRank(rank)).(SpellData)
	if !ok {
		panic(fmt.Sprintf("rank %d is not a SpellData, so it carries no effects or proc chance", rank))
	}
	if pick != nil {
		return pick(row)
	}

	// Nothing named means nothing to choose between - reading the first of several silently is the
	// bug this shape exists to prevent.
	if len(row.Effects) != 1 {
		panic(fmt.Sprintf("spell %d rank %d has %d effects - name the one you mean with Effect(aura, misc)",
			row.SpellID, rank, len(row.Effects)))
	}
	return row.Effects[0].Value
}
