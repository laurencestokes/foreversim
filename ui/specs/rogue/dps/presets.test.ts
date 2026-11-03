import { MAX_POINTS_PLAYER } from '@features/talents/model/can_set_points';
import { TalentLayout } from '@generated/proto/common';
import { brokenTalentNames } from '@sim/talents/broken_talents';
import { rogueTalentsConfig } from '@sim/talents/rogue';
import { parseTalentsString, serializeTalentsString, totalPointsSpent } from '@sim/talents/talents_string';
import { describe, expect, it } from 'vitest';

import * as Presets from './presets';

describe.each([
	Presets.TALENTS_MUTILATE_ASN,
	Presets.TALENTS_MUTILATE_COM,
	Presets.TALENTS_COMBAT_SS,
	Presets.TALENTS_COMBAT_BS,
	Presets.TALENTS_SUB_HEMO,
	Presets.TALENTS_SUB_BS,
])('$name launch talents', preset => {
	it('spends the available points without breaking a tier or prerequisite', () => {
		const points = parseTalentsString(rogueTalentsConfig, preset.data.talentsString);
		expect(totalPointsSpent(points)).toBe(MAX_POINTS_PLAYER);
		expect(brokenTalentNames(rogueTalentsConfig, points)).toEqual([]);
		expect(serializeTalentsString(points)).toBe(preset.data.talentsString);
		expect(preset.data.talentLayout).toBe(TalentLayout.TalentLayoutLaunch);
	});
});
