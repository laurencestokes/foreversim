import { Class, TalentLayout } from '@generated/proto/common';
import { describe, expect, it } from 'vitest';

import { classTalentsConfig, currentTalentLayout, loadSavedTalentString } from './factory';
import { parseTalentsString } from './talents_string';
import previous from './trees/history/prelaunch.json';

const oldString = (trees: string[][], ranks: Record<string, number>) =>
	trees
		.map(tree =>
			tree
				.map(name => ranks[name] ?? 0)
				.join('')
				.replace(/0+$/, ''),
		)
		.join('-')
		.replace(/-+$/, '');

describe('saved talent layouts', () => {
	it('keeps ranks on the same Warrior talent after a move and caps Flurry at its new maximum', () => {
		const config = classTalentsConfig[Class.ClassWarrior];
		const before = oldString(previous.warrior, { flurry: 5, deathWish: 1 });
		const migrated = loadSavedTalentString(Class.ClassWarrior, before, TalentLayout.TalentLayoutLegacy);
		const ranks = parseTalentsString(config, migrated)[1];
		const point = (field: string) => ranks[config[1].talents.findIndex(t => t.fieldName === field)];
		expect(point('flurry')).toBe(3);
		expect(point('deathWish')).toBe(1);
		expect(point('recklessSwings')).toBe(0);
		expect(loadSavedTalentString(Class.ClassWarrior, migrated, currentTalentLayout(Class.ClassWarrior))).toBe(migrated);
	});

	it('drops retired Rogue investments without giving their points to a new talent', () => {
		const before = oldString(previous.rogue, { murder: 2, quietus: 5, thousandCuts: 1 });
		expect(loadSavedTalentString(Class.ClassRogue, before, TalentLayout.TalentLayoutLegacy)).toBe('');
	});

	it('leaves a class with its original layout untouched', () => {
		expect(loadSavedTalentString(Class.ClassMage, '005--00', TalentLayout.TalentLayoutLegacy)).toBe('005--00');
	});
});
