import type { RaceArenaSetup } from './RaceArenaSetupDialog';

export type RaceResult = { race: string; dps: number; behind: number; tier: string; relabelled?: string; racials: string };
export type RaceList = {
	spec: string;
	class: string;
	build: string;
	talents: string;
	gear: string;
	rotation: string;
	consumables: string;
	target: string;
	// The fight length the list was run at; absent on results from before fight lengths were split.
	seconds?: number;
	keptWeaponType?: boolean;
	weaponType?: string;
	races: Array<RaceResult>;
};
export type Tier = { name: string; maxBehind?: number };
export type RaceArenaResults = {
	commit: string;
	generated: string;
	iterations: number;
	fight: { seconds: number; durations?: Array<number>; targetLevel: number; targetArmor: number };
	tiers: Array<Tier>;
	noise: { maxError: number; maxDifference: number };
	lists: Array<RaceList>;
	setups: Record<string, RaceArenaSetup>;
};

// The runner groups DPS builds by these spec keys.
const SPEC_NAMES: Record<string, string> = {
	balance_druid: 'Balance Druid',
	feral_druid: 'Feral Druid',
	hunter: 'Hunter',
	mage: 'Mage',
	retribution_paladin: 'Retribution Paladin',
	shadow_priest: 'Shadow Priest',
	smite_priest: 'Smite Priest',
	rogue: 'Rogue',
	elemental_shaman: 'Elemental Shaman',
	enhancement_shaman: 'Enhancement Shaman',
	warlock: 'Warlock',
	warrior: 'Warrior',
};

export const specName = (spec: string) => SPEC_NAMES[spec] ?? spec;

// The creature types the lists were run against. Mechanical is the race arena's neutral target;
// the other two are the ones a racial pays extra against.
export const TARGETS: Array<{ key: string; label: string; short: string; blurb: string }> = [
	{ key: 'Mechanical', label: 'Neutral target', short: 'Neutral', blurb: 'A target no racial pays extra against. This is the main view.' },
	{
		key: 'Elemental',
		label: 'Elemental target',
		short: 'Elemental',
		blurb: "Adds the Skyborne's Elemental Insight: +5% damage against Elementals.",
	},
	{
		key: 'Beast',
		label: 'Beast target',
		short: 'Beast',
		blurb: "Adds the Dwarf's Big Game Hunter and the Troll's Beast Slaying: +5% damage against Beasts.",
	},
];

export const secondsOf = (data: RaceArenaResults, list: RaceList) => list.seconds ?? data.fight.seconds;

export const fightLabel = (seconds: number) => (seconds % 60 === 0 ? `${seconds / 60} min` : `${seconds} s`);

export const durationsOf = (data: RaceArenaResults) => data.fight.durations ?? [data.fight.seconds];

// One fight a build was run in: a length and a target type.
export type Condition = { seconds: number; target: string };
export const conditionKey = (condition: Condition) => `${condition.seconds}|${condition.target}`;

// One DPS build, with its list for every condition it was run in.
export type BuildGroup = {
	key: string;
	spec: string;
	class: string;
	build: string;
	races: Array<string>;
	lists: Map<string, RaceList>;
};

export const groupBuilds = (data: RaceArenaResults): Array<BuildGroup> => {
	const groups = new Map<string, BuildGroup>();
	for (const list of data.lists) {
		const key = `${list.spec}|${list.build}`;
		let group = groups.get(key);
		if (!group) {
			group = { key, spec: list.spec, class: list.class, build: list.build, races: [], lists: new Map() };
			groups.set(key, group);
		}
		group.lists.set(conditionKey({ seconds: secondsOf(data, list), target: list.target }), list);
		for (const race of list.races) {
			if (!group.races.includes(race.race)) group.races.push(race.race);
		}
	}
	return [...groups.values()];
};

// How one race fares across a set of conditions. Worst is the largest gap behind the best race in
// any of them: a race with a small worst gap is never far off, whichever of those fights it meets.
export type RaceStanding = {
	race: string;
	worst: number;
	mean: number;
	topTier: number;
	conditions: number;
};

export const rankRaces = (group: BuildGroup, conditions: Array<Condition>, topTierName: string): Array<RaceStanding> => {
	const lists = conditions.map(condition => group.lists.get(conditionKey(condition))).filter((list): list is RaceList => !!list);
	if (!lists.length) return [];
	const standings = group.races.map(race => {
		const gaps = lists.map(list => list.races.find(row => row.race === race)).filter((row): row is RaceResult => !!row);
		return {
			race,
			worst: Math.max(...gaps.map(row => row.behind)),
			mean: gaps.reduce((sum, row) => sum + row.behind, 0) / gaps.length,
			topTier: gaps.filter(row => row.tier === topTierName).length,
			conditions: gaps.length,
		};
	});
	return standings.filter(s => s.conditions === lists.length).sort((a, b) => a.worst - b.worst || a.mean - b.mean || a.race.localeCompare(b.race));
};
