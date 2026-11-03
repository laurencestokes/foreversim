package database

import "slices"

// The November 2 Shaman values retained by the launch projection. The keys are SpellEffect IDs.
var launchShamanBasePoints = map[int32]float64{
	// Chain Lightning
	679158:  216,
	679273:  283,
	681324:  428,
	688920:  512,
	1072633: 108,
	1072635: 141.5,
	1072636: 214,
	1072638: 256,
	// Earth Shock
	686790: 20.5,
	685548: 36.5,
	686227: 67,
	686672: 130,
	687897: 242,
	688149: 383,
	688559: 531,
	// Flame Shock
	686823: 25,
	685531: 51,
	685532: 12,
	686555: 95,
	686556: 24,
	688330: 164,
	688331: 42,
	688934: 245,
	688935: 64,
	704945: 292,
	704946: 80,
	// Frost Shock
	686558: 98,
	687459: 222.5,
	688813: 355.5,
	688022: 506,
	// Lightning Bolt
	678918:  12,
	679369:  24,
	679030:  41,
	679313:  73,
	679378:  109,
	684438:  149,
	688309:  195,
	688310:  241,
	691075:  295,
	691780:  353,
	1072563: 6,
	1072564: 12,
	1072565: 20.5,
	1072567: 36.5,
	1072569: 54.5,
	1072625: 74.5,
	1072626: 97.5,
	1072627: 120.5,
	1072628: 147.5,
	1072630: 176.5,
}

var launchShamanCoefficients = map[int32]float64{
	// Chain Lightning
	679158:  0.714,
	679273:  0.714,
	681324:  0.714,
	688920:  0.714,
	1072633: 0.357,
	1072635: 0.357,
	1072636: 0.357,
	1072638: 0.357,
}

var launchChainLightningMana = map[int32]int32{
	421:   248,
	930:   304,
	2860:  428,
	10605: 495,
}

// Only the captured build still missing these announced values receives the preview. A later
// Shaman capture uses its game rows and requires a fresh comparison against the launch notes.
func applyLaunchShaman(in *storeInputs, rows []storeSpell, curves map[int32][][]float64) {
	if in.Builds["shaman"] != "1.60.1.70755" || in.Hotfixes["shaman"] != 113583 {
		return
	}
	for i := range rows {
		spell := &rows[i]
		for j := range spell.Effects {
			effect := &spell.Effects[j]
			if points, ok := launchShamanBasePoints[effect.ID]; ok {
				effect.BasePoints = points
			}
			if coefficient, ok := launchShamanCoefficients[effect.ID]; ok {
				effect.SPCoef = coefficient
			}
		}
		if mana, ok := launchChainLightningMana[spell.ID]; ok {
			spell.CastTimeMs = 2500
			spell.Powers = slices.Clone(spell.Powers)
			spell.Powers[0].Cost = mana
		}
		if spell.ID == 17364 {
			spell.CooldownMs = 6000
		}
	}
	curves[16256] = [][]float64{{6, 12, 18, 24, 30}} // Flurry
	curves[408438] = [][]float64{{10, 20, 30}}       // Lightning Overload
	curves[415140] = [][]float64{{50, 100, 150}}     // Mental Dexterity
}
