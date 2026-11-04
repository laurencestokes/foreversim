import { TalentLayout } from '@generated/proto/common';
import { DpsWarrior } from '@sim/player/specs/warrior';
import { brokenTalentNames } from '@sim/talents/broken_talents';
import { playerTalentStringToProto } from '@sim/talents/factory';
import { parseTalentsString, totalPointsSpent } from '@sim/talents/talents_string';
import { warriorTalentsConfig } from '@sim/talents/warrior';
import { describe, expect, it } from 'vitest';

import { ProtectionTalents } from '../protection/presets';
import { ArmsTalents, FuryTalents } from './presets';

describe('Warrior hotfix presets', () => {
	for (const preset of [ArmsTalents, FuryTalents]) {
		it(`${preset.name} spends 51 legal points in the hotfix tree`, () => {
			const points = parseTalentsString(warriorTalentsConfig, preset.data.talentsString);
			expect(preset.data.talentLayout).toBe(TalentLayout.TalentLayoutHotfix);
			expect(totalPointsSpent(points)).toBe(51);
			expect(brokenTalentNames(warriorTalentsConfig, points)).toEqual([]);
			const talents = playerTalentStringToProto(DpsWarrior, preset.data.talentsString);
			expect(talents.recklessSwings).toBe(preset === FuryTalents);
			expect(talents.improvedTacticalMastery).toBe(3);
		});
	}

	it('Fury takes the new Whirlwind talent and the three-point Flurry', () => {
		const talents = playerTalentStringToProto(DpsWarrior, FuryTalents.data.talentsString);
		expect(talents.improvedWhirlwind).toBe(2);
		expect(talents.flurry).toBe(3);
		expect(talents.bloodthirst && talents.deathWish).toBe(true);
	});

	it('Arms retains the two-handed damage talents through their stable fields', () => {
		const talents = playerTalentStringToProto(DpsWarrior, ArmsTalents.data.talentsString);
		expect(talents.twoHandedWeaponSpecialization).toBe(3);
		expect(talents.bloodthrill).toBe(2);
		expect(talents.improvedSlam).toBe(2);
	});

	it('Protection retains 51 legal points in the hotfix tree', () => {
		const points = parseTalentsString(warriorTalentsConfig, ProtectionTalents.data.talentsString);
		expect(ProtectionTalents.data.talentLayout).toBe(TalentLayout.TalentLayoutHotfix);
		expect(totalPointsSpent(points)).toBe(51);
		expect(brokenTalentNames(warriorTalentsConfig, points)).toEqual([]);
	});
});
