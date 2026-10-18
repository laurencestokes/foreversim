import assert from 'node:assert/strict';
import { test } from 'node:test';

import { attachSetups, snapshotBuild } from './snapshot.mjs';

const trees = [
	{ name: 'First tree', talents: [{ fancyName: 'First talent', spellId: 10, maxPoints: 5 }] },
	{ name: 'Empty tree', talents: [{ fancyName: 'Unused talent', spellId: 20, maxPoints: 3 }] },
	{ name: 'Last tree', talents: [{ fancyName: 'Last talent', spellId: 30, maxPoints: 1 }] },
];
const player = () => {
	const items = Array.from({ length: 17 }, () => ({}));
	items[14] = { id: 1, randomSuffix: 3, enchant: 4 };
	const swapped = Array.from({ length: 17 }, () => ({}));
	swapped[14] = { id: 2 };
	return {
		equipment: { items },
		itemSwap: { items: swapped },
		enableItemSwap: true,
		talentsString: '2--1',
		database: {
			items: [
				{ id: 1, name: 'Test mace' },
				{ id: 2, name: 'Test sword' },
			],
			randomSuffixes: [{ id: 3, name: 'of Testing' }],
			enchants: [{ effectId: 4, name: 'Test enchant' }],
		},
	};
};

test('records equipped suffixes, enchants, empty slots and active weapon swaps', () => {
	const setup = snapshotBuild(player(), trees);
	assert.deepEqual(setup.equipment[14], { slot: 'Main hand', item: { id: 1, name: 'Test mace of Testing', suffixId: 3, enchant: 'Test enchant' } });
	assert.deepEqual(setup.equipment[15], { slot: 'Off hand' });
	assert.deepEqual(setup.itemSwap, [{ slot: 'Main hand', item: { id: 2, name: 'Test sword' } }]);
	assert.equal(snapshotBuild({ ...player(), enableItemSwap: false }, trees).itemSwap, undefined);
});

test('keeps an empty middle talent tree and records selected ranks', () => {
	const setup = snapshotBuild(player(), trees);
	assert.deepEqual(
		setup.talents.map(tree => tree.points),
		[2, 0, 1],
	);
	assert.deepEqual(setup.talents[1].talents, []);
	assert.deepEqual(setup.talents[2].talents, [{ id: 30, name: 'Last talent', rank: 1, maxRank: 1 }]);
	assert.equal(setup.talentsString, '2--1');
});

test('rejects missing equipment data and ranks that cannot match the recorded trees', () => {
	assert.throws(() => snapshotBuild({ ...player(), database: { items: [] } }, trees), /Missing equipped item/);
	assert.throws(() => snapshotBuild({ ...player(), talentsString: '9' }, trees), /Invalid rank/);
});

test('attaches one setup across targets without changing results or provenance', () => {
	const input = { spec: 'test', build: 'Build', gear: 'Gear', rotation: 'Rotation', player: player(), setup: snapshotBuild(player(), trees) };
	const list = {
		spec: input.spec,
		build: input.build,
		gear: input.gear,
		rotation: input.rotation,
		talents: input.player.talentsString,
		races: [{ race: 'Dwarf', dps: 123 }],
	};
	const original = {
		commit: 'recorded-engine',
		generated: 'recorded-date',
		lists: [
			{ ...list, target: 'Mechanical' },
			{ ...list, target: 'Beast' },
		],
	};
	const result = attachSetups(original, [input]);
	assert.deepEqual(result, { ...original, setups: { 'test|Build': input.setup } });
	assert.throws(() => attachSetups(original, [{ ...input, player: { talentsString: '1' } }]), /Talents differ/);
	assert.throws(() => attachSetups(original, []), /Missing setup/);
});
