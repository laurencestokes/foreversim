import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import { pathToFileURL } from 'node:url';

const slots = [
	'Head',
	'Neck',
	'Shoulders',
	'Back',
	'Chest',
	'Wrist',
	'Hands',
	'Waist',
	'Legs',
	'Feet',
	'Ring 1',
	'Ring 2',
	'Trinket 1',
	'Trinket 2',
	'Main hand',
	'Off hand',
	'Ranged / relic',
];

export function snapshotBuild(player, trees) {
	const database = player.database;
	const equipment = (items, swaps = false) =>
		slots.flatMap((slot, index) => {
			const spec = items?.[index];
			if (!spec?.id) return swaps ? [] : [{ slot }];
			const item = database.items.find(item => item.id === spec.id);
			assert.ok(item?.name, `Missing equipped item ${spec.id}`);
			const suffix = spec.randomSuffix ? database.randomSuffixes?.find(suffix => suffix.id === spec.randomSuffix) : undefined;
			assert.ok(!spec.randomSuffix || suffix?.name, `Missing suffix ${spec.randomSuffix}`);
			const enchant = spec.enchant ? database.enchants?.find(enchant => enchant.effectId === spec.enchant) : undefined;
			assert.ok(!spec.enchant || enchant?.name, `Missing enchant ${spec.enchant}`);
			return [
				{
					slot,
					item: {
						id: item.id,
						name: [item.name, suffix?.name].filter(Boolean).join(' '),
						...(spec.randomSuffix ? { suffixId: spec.randomSuffix } : {}),
						...(enchant ? { enchant: enchant.name } : {}),
					},
				},
			];
		});
	const strings = player.talentsString.split('-');
	assert.ok(strings.length <= trees.length, 'Unknown talent tree');
	const talents = trees.map((tree, treeIndex) => {
		const digits = strings[treeIndex] ?? '';
		assert.match(digits, /^\d*$/, 'Invalid talent string');
		assert.ok(digits.length <= tree.talents.length, 'Unknown talent');
		const selected = tree.talents.flatMap((talent, index) => {
			const rank = Number(digits[index] ?? 0);
			assert.ok(rank <= talent.maxPoints, `Invalid rank for ${talent.fancyName}`);
			return rank
				? [
						{
							id: talent.spellId,
							...(talent.definitionId ? { definitionId: talent.definitionId } : {}),
							name: talent.fancyName,
							rank,
							maxRank: talent.maxPoints,
						},
					]
				: [];
		});
		return { name: tree.name, points: selected.reduce((sum, talent) => sum + talent.rank, 0), talents: selected };
	});
	return {
		equipment: equipment(player.equipment.items),
		...(player.enableItemSwap ? { itemSwap: equipment(player.itemSwap?.items, true) } : {}),
		talentsString: player.talentsString,
		talents,
	};
}

export function attachSetups(results, inputs) {
	const setups = {};
	for (const list of results.lists) {
		const build = inputs.find(input => input.spec === list.spec && input.build === list.build);
		assert.ok(build?.setup, `Missing setup for ${list.spec} / ${list.build}`);
		assert.equal(build.player.talentsString, list.talents, `Talents differ for ${list.build}`);
		assert.equal(build.gear, list.gear, `Gear preset differs for ${list.build}`);
		assert.equal(build.rotation, list.rotation, `Rotation differs for ${list.build}`);
		setups[`${list.spec}|${list.build}`] = build.setup;
	}
	return { ...results, setups };
}

// Without a results path, enrich the runner's input. With one, attach setups to existing results
// using the talent definitions at their recorded engine revision; their numbers and date stay intact.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
	const [inputPath, resultsPath] = process.argv.slice(2);
	assert.ok(inputPath, 'Usage: node tools/race_arena/snapshot.mjs <input.json> [results.json]');
	const inputs = JSON.parse(fs.readFileSync(inputPath, 'utf8'));
	const results = resultsPath ? JSON.parse(fs.readFileSync(resultsPath, 'utf8')) : undefined;
	const trees = new Map();
	for (const build of inputs) {
		if (!trees.has(build.class)) {
			const file = `ui/sim/talents/trees/${build.class.toLowerCase()}.json`;
			const data = results ? execFileSync('git', ['show', `${results.commit}:${file}`], { encoding: 'utf8' }) : fs.readFileSync(file, 'utf8');
			trees.set(build.class, JSON.parse(data));
		}
		build.setup = snapshotBuild(build.player, trees.get(build.class));
	}
	const output = results ? attachSetups(results, inputs) : inputs;
	fs.writeFileSync(resultsPath ?? inputPath, JSON.stringify(output, null, '\t') + '\n');
	console.log(`Saved setups for ${inputs.length} builds to ${resultsPath ?? inputPath}`);
}
