import type { RaceArenaSetup } from '../race_arena/RaceArenaSetupDialog';

export type BuildArenaRow = {
	buildId: string;
	inputFingerprint: string;
	seconds: number;
	target: string;
	race: string;
	raceValue: string;
	dps: number;
	error: number;
	iterations: number;
	sourceId: string;
};
export type BuildArenaBuild = {
	id: string;
	inputFingerprint: string;
	spec: string;
	class: string;
	name: string;
	gear: string;
	rotation: string;
	status: string;
	devGear: boolean;
	caveats: string[];
	setup: RaceArenaSetup;
	consumableNames?: Array<{ id: number; name: string }>;
	request: Record<string, unknown>;
};
export type BuildArenaResults = {
	schemaVersion: 1;
	compatibility?: { method: string; evidence: string };
	publishedAt: string;
	upstreamCommit: string;
	verifiedCommit: string;
	iterations: number;
	fight: { seconds: number; durations: number[]; targetLevel: number; targetArmor: number; targets: string[] };
	tiers: Array<{ name: string; maxBehind?: number }>;
	sources: Array<{
		id: string;
		commit: string;
		publishedAt?: string;
		computedAt?: string;
		computedWindow?: { start: string; end: string };
		reused: boolean;
		note: string;
	}>;
	builds: BuildArenaBuild[];
	rows: BuildArenaRow[];
	excluded: Array<{ spec: string; reason: string }>;
};
export type BuildArenaSelection = { seconds: number; target: string; className: string; race: string };
export type BuildStanding = {
	build: BuildArenaBuild;
	row: BuildArenaRow;
	races: BuildArenaRow[];
	behind: number;
	tier: string;
	gapInterval: [number, number];
	boundaryUncertain: boolean;
	indistinguishable: boolean;
	raceSpread: number;
	closeRaces: string[];
};
const lexical = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0);
const compareRows = (a: BuildArenaRow, b: BuildArenaRow) => b.dps - a.dps || lexical(a.buildId, b.buildId) || lexical(a.race, b.race);
export const roundedGap = (dps: number, leaderDps: number) => Math.round(Math.max(0, 100 * (1 - dps / leaderDps)) * 100) / 100;
const tierForGap = (behind: number, tiers: BuildArenaResults['tiers']) => tiers.find(tier => tier.maxBehind === undefined || behind <= tier.maxBehind)!.name;
export const differenceIsNoise = (a: BuildArenaRow, b: BuildArenaRow) => Math.abs(a.dps - b.dps) <= 1.96 * Math.hypot(a.error, b.error);

export const rankBuilds = (data: BuildArenaResults, selection: BuildArenaSelection): { standings: BuildStanding[]; omitted: BuildArenaBuild[] } => {
	const omitted: BuildArenaBuild[] = [];
	const candidates = data.builds
		.filter(build => !selection.className || build.class === selection.className)
		.flatMap(build => {
			const races = data.rows
				.filter(
					row =>
						row.buildId === build.id &&
						row.inputFingerprint === build.inputFingerprint &&
						row.seconds === selection.seconds &&
						row.target === selection.target,
				)
				.sort(compareRows);
			const row = selection.race ? races.find(race => race.race === selection.race) : races[0];
			if (!row) {
				omitted.push(build);
				return [];
			}
			return [{ build, row, races }];
		})
		.sort((a, b) => compareRows(a.row, b.row));
	const leader = candidates[0]?.row;
	if (!leader) return { standings: [], omitted };
	return {
		omitted,
		standings: candidates.map(({ build, row, races }) => {
			const behind = roundedGap(row.dps, leader.dps);
			const halfWidth = row === leader ? 0 : 196 * Math.hypot(row.error / leader.dps, (row.dps * leader.error) / (leader.dps * leader.dps));
			const gapInterval: [number, number] = [100 * (1 - row.dps / leader.dps) - halfWidth, 100 * (1 - row.dps / leader.dps) + halfWidth];
			return {
				build,
				row,
				races,
				behind,
				tier: tierForGap(behind, data.tiers),
				gapInterval,
				boundaryUncertain:
					row !== leader &&
					data.tiers.some(tier => tier.maxBehind !== undefined && gapInterval[0] <= tier.maxBehind && gapInterval[1] >= tier.maxBehind),
				indistinguishable: row !== leader && differenceIsNoise(row, leader),
				raceSpread: roundedGap(races[races.length - 1].dps, races[0].dps),
				closeRaces: races.filter(race => differenceIsNoise(race, races[0])).map(race => race.race),
			};
		}),
	};
};

export const matchesBuildSearch = (standing: BuildStanding, search: string) =>
	[standing.build.name, standing.build.class, standing.build.spec, standing.build.gear, standing.build.rotation]
		.join(' ')
		.toLowerCase()
		.includes(search.trim().toLowerCase());

export const requestForRow = (build: BuildArenaBuild, row: BuildArenaRow): Record<string, unknown> => {
	const request = JSON.parse(JSON.stringify(build.request)) as Record<string, unknown>;
	const raid = request.raid as { parties: Array<{ players: Array<{ race: string }> }> };
	const encounter = request.encounter as { duration: number; targets: Array<{ mobType: string }> };
	raid.parties[0].players[0].race = row.raceValue;
	encounter.duration = row.seconds;
	encounter.targets[0].mobType = `MobType${row.target}`;
	return request;
};
