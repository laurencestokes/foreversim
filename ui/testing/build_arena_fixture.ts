const builds = [
	{ id: 'alpha', class: 'Warrior', name: 'Fury', dps: 1000, gear: 'P1', devGear: false },
	{ id: 'beta', class: 'Mage', name: 'Fire', dps: 950.04, gear: 'Pre-raid', devGear: false },
	{ id: 'gamma', class: 'Rogue', name: 'Combat', dps: 900, gear: 'P1', devGear: false },
	{ id: 'delta', class: 'Mage', name: 'Frost', dps: 800, gear: 'Development', devGear: true },
	{ id: 'epsilon', class: 'Rogue', name: 'Assassination', dps: 700, gear: 'P1', devGear: false },
	{ id: 'zeta', class: 'Warrior', name: 'Arms', dps: 600, gear: 'Pre-raid', devGear: false },
];
const setup = {
	equipment: [{ slot: 'Head', item: { id: 12640, name: 'Lionheart Helm', enchant: '+8 Strength' } }],
	itemSwap: [{ slot: 'Main hand', item: { id: 19352, name: 'Chromatically Tempered Sword' } }],
	talentsString: '050050-550',
	talents: [{ name: 'Arms', points: 5, talents: [{ id: 12282, name: 'Improved Heroic Strike', rank: 3, maxRank: 3 }] }],
};
export const buildArenaFixture = {
	schemaVersion: 1 as const,
	publishedAt: '2026-10-28T12:00:00Z',
	upstreamCommit: 'abcdef123456',
	verifiedCommit: '123456abcdef',
	iterations: 50000,
	fight: { seconds: 180, durations: [90, 180, 360], targetLevel: 63, targetArmor: 3731, targets: ['Mechanical', 'Elemental', 'Beast'] },
	tiers: [{ name: 'S', maxBehind: 5 }, { name: 'A', maxBehind: 10 }, { name: 'B', maxBehind: 20 }, { name: 'C', maxBehind: 30 }, { name: 'D' }],
	sources: [
		{
			id: 'fixture',
			commit: '123456abcdef',
			computedAt: '2026-10-27T12:00:00Z',
			publishedAt: '2026-10-28T12:00:00Z',
			reused: true,
			note: 'Recorded preset results.',
		},
	],
	builds: builds.map(build => ({
		id: build.id,
		inputFingerprint: `input-${build.id}`,
		class: build.class,
		spec: build.class.toLowerCase(),
		name: build.name,
		gear: build.gear,
		rotation: 'Default',
		status: 'Alpha',
		devGear: build.devGear,
		caveats: build.devGear ? ['Development preset.'] : [],
		setup,
		consumableNames: [
			{ id: 13512, name: 'Flask of Supreme Power' },
			{ id: 13444, name: 'Major Mana Potion' },
		],
		request: {
			raid: {
				buffs: { arcaneBrilliance: true },
				debuffs: { sunderArmor: true },
				parties: [
					{
						buffs: { battleShout: true },
						players: [
							{
								name: build.name,
								race: 'RaceHuman',
								talentsString: setup.talentsString,
								equipment: { items: [{ id: 12640 }] },
								buffs: { blessingOfKings: true },
								consumables: { flaskId: 13512, foodId: 999999, resourcePotId: 13444, potions: [13444] },
								database: {
									items: [{ id: 12640, name: 'Lionheart Helm' }],
								},
								warrior: { options: { stance: 'BerserkerStance' } },
								rotation: { type: 'TypeAPL', priorityList: [{ action: { castSpell: { spellId: { spellId: 11567 } } } }] },
							},
						],
					},
				],
			},
			encounter: { duration: 180, targets: [{ level: 63, mobType: 'MobTypeMechanical', stats: [3731] }] },
			simOptions: { iterations: 50000, randomSeed: '101' },
		},
	})),
	rows: builds.flatMap(build =>
		[90, 180, 360].flatMap(seconds =>
			['Mechanical', 'Elemental', 'Beast'].flatMap(target =>
				(build.class === 'Warrior' ? ['Human', 'Orc'] : ['Human', 'Gnome']).map((race, index) => ({
					buildId: build.id,
					inputFingerprint: `input-${build.id}`,
					seconds,
					target,
					race,
					raceValue: `Race${race}`,
					dps: (build.dps - index * 0.1) * (seconds === 90 && build.id === 'beta' ? 1.2 : target === 'Beast' && build.id === 'zeta' ? 2 : 1),
					error: 0.1,
					iterations: 50000,
					sourceId: 'fixture',
				})),
			),
		),
	),
	excluded: [{ spec: 'protection_warrior', reason: 'Tank preset.' }],
};
