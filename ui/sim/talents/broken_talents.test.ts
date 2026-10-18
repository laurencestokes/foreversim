import { describe, expect, it } from 'vitest';

import { brokenTalentNames, findBrokenTalents } from './broken_talents';
import type { TalentsConfig } from './config';
import { newTalentsConfig } from './config';
import { parseTalentsString } from './talents_string';

type Fake = Record<string, never>;

const COLS = 4;

const mkTalent = (index: number, overrides: Record<string, unknown> = {}) => ({
	fieldName: `talent${index}`,
	fancyName: `Talent ${index}`,
	location: { rowIdx: Math.floor(index / COLS), colIdx: index % COLS },
	spellId: 1000 + index,
	maxPoints: 5,
	...overrides,
});

// Talent 5 (row 1) needs talent 1 full; talent 9 (row 2) needs talent 5 full; talent 7 (row 1) needs talent 6 on its
// left, and talent 4 (row 1) needs talent 5 on its right.
const config: TalentsConfig<Fake> = newTalentsConfig<Fake>([
	{
		name: 'A',
		backgroundUrl: 'a.jpg',
		talents: Array.from({ length: 4 * COLS }, (_, index) =>
			mkTalent(
				index,
				index === 4
					? { prereqLocation: { rowIdx: 1, colIdx: 1 } }
					: index === 5
						? { prereqLocation: { rowIdx: 0, colIdx: 1 } }
						: index === 9
							? { prereqLocation: { rowIdx: 1, colIdx: 1 } }
							: index === 7
								? { prereqLocation: { rowIdx: 1, colIdx: 2 } }
								: {},
			),
		),
	},
	{ name: 'B', backgroundUrl: 'b.jpg', talents: Array.from({ length: 4 * COLS }, (_, index) => mkTalent(index)) },
]);

const brokenIdxs = (talentsString: string, treeIdx = 0) =>
	findBrokenTalents(config, parseTalentsString(config, talentsString))[treeIdx].flatMap((broken, idx) => (broken ? [idx] : []));

describe('findBrokenTalents', () => {
	it('finds nothing in a legal tree', () => {
		expect(brokenIdxs('5500' + '05')).toEqual([]);
	});

	it('breaks a talent whose rows above hold too few points', () => {
		expect(brokenIdxs('4000' + '1')).toEqual([4]);
	});

	it('breaks a child whose prerequisite is not full, and leaves the prerequisite alone', () => {
		expect(brokenIdxs('1400' + '01')).toEqual([5]);
	});

	it('breaks a prerequisite in the same row before its child', () => {
		expect(brokenIdxs('5000' + '0045')).toEqual([7]);
		expect(brokenIdxs('5000' + '0055')).toEqual([]);
	});

	it('cascades: a broken talent unlocks no tier and fills no prerequisite', () => {
		expect(brokenIdxs('5400' + '0550' + '05')).toEqual([5, 9]);
		expect(brokenIdxs('4000' + '5500' + '1')).toEqual([4, 5, 8]);
	});

	it('decides a prerequisite to the right of its child in the same row', () => {
		expect(brokenIdxs('5500' + '15')).toEqual([]);
		expect(brokenIdxs('5500' + '14')).toEqual([4]);
		expect(brokenIdxs('5400' + '15')).toEqual([4, 5]);
	});

	it('checks every tree on its own', () => {
		expect(brokenIdxs('5-4000' + '1', 0)).toEqual([]);
		expect(brokenIdxs('5-4000' + '1', 1)).toEqual([4]);
	});

	it('names the broken talents in tree order', () => {
		expect(brokenTalentNames(config, parseTalentsString(config, '4000' + '1-4000' + '2'))).toEqual(['Talent 4', 'Talent 4']);
	});
});
