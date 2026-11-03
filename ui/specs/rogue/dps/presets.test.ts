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
	Presets.TALENTS_MUTILATE_ASN_UPSTREAM,
	Presets.TALENTS_COMBAT_SS_UPSTREAM,
	Presets.TALENTS_COMBAT_BS_UPSTREAM,
	Presets.TALENTS_SUB_HEMO_UPSTREAM,
	Presets.TALENTS_SUB_BS_UPSTREAM,
])('$name talents', preset => {
	it('spends the available points without breaking a tier or prerequisite', () => {
		const points = parseTalentsString(rogueTalentsConfig, preset.data.talentsString);
		expect(totalPointsSpent(points)).toBe(MAX_POINTS_PLAYER);
		expect(brokenTalentNames(rogueTalentsConfig, points)).toEqual([]);
		expect(serializeTalentsString(points)).toBe(preset.data.talentsString);
		expect(preset.data.talentLayout).toBe(TalentLayout.TalentLayoutHotfix);
	});
});

describe.each([
	[Presets.PRESET_BUILD_MUTILATE_ASN_UPSTREAM, Presets.PRESET_BUILD_MUTILATE_ASN, [32, 19, 0]],
	[Presets.PRESET_BUILD_COMBAT_SS_UPSTREAM, Presets.PRESET_BUILD_COMBAT_SS, [20, 31, 0]],
	[Presets.PRESET_BUILD_COMBAT_BS_UPSTREAM, Presets.PRESET_BUILD_COMBAT_BS, [18, 31, 2]],
	[Presets.PRESET_BUILD_SUB_HEMO_UPSTREAM, Presets.PRESET_BUILD_SUB_HEMO, [20, 0, 31]],
	[Presets.PRESET_BUILD_SUB_BS_UPSTREAM, Presets.PRESET_BUILD_SUB_BS, [15, 8, 28]],
] as const)('%s upstream allocation', (upstream, original, totals) => {
	it('keeps the baseline gear and rotation while offering a different allocation', () => {
		const points = parseTalentsString(rogueTalentsConfig, upstream.talents!.data.talentsString);
		expect(points.map(tree => tree.reduce((sum, rank) => sum + rank, 0))).toEqual(totals);
		expect(upstream.talents!.data.talentsString).not.toBe(original.talents!.data.talentsString);
		expect(upstream.gear).toBe(original.gear);
		expect(upstream.rotation).toBe(original.rotation);
		expect(upstream.settings).toBe(original.settings);
	});
});

it('maps the upstream Subtlety allocations by talent name', () => {
	for (const preset of [Presets.TALENTS_SUB_HEMO_UPSTREAM, Presets.TALENTS_SUB_BS_UPSTREAM]) {
		const points = parseTalentsString(rogueTalentsConfig, preset.data.talentsString);
		const ranks = Object.fromEntries(rogueTalentsConfig.flatMap((tree, ti) => tree.talents.map((talent, i) => [talent.fieldName, points[ti][i]])));
		expect(ranks.improvedAmbush).toBe(3);
		expect(ranks.dirtyTricks).toBe(0);
		expect(ranks.bootKnife).toBe(0);
		expect(ranks.deathMark).toBe(0);
	}
});
