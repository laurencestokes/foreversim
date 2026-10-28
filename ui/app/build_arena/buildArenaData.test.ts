import { buildArenaFixture } from '@testing/build_arena_fixture';
import { describe, expect, it } from 'vitest';

import { type BuildArenaResults, matchesBuildSearch, rankBuilds, requestForRow, roundedGap } from './buildArenaData';

const selection = { seconds: 180, target: 'Mechanical', className: '', race: '' };
const data: BuildArenaResults = buildArenaFixture;

describe('build ranking', () => {
	it('rounds the gap before assigning each tier boundary', () => {
		for (const [dps, gap, tier] of [
			[950.04, 5, 'S'],
			[949.94, 5.01, 'A'],
			[900, 10, 'A'],
			[899.94, 10.01, 'B'],
			[800, 20, 'B'],
			[799.94, 20.01, 'C'],
			[700, 30, 'C'],
			[699.94, 30.01, 'D'],
		] as const) {
			expect(roundedGap(dps, 1000)).toBe(gap);
			const boundaryData = { ...data, rows: data.rows.map(row => (row.buildId === 'beta' ? { ...row, dps } : row)) };
			expect(rankBuilds(boundaryData, selection).standings.find(row => row.build.id === 'beta')?.tier).toBe(tier);
		}
	});
	it('uses one best legal race per build and preserves the leader during search', () => {
		const { standings } = rankBuilds(data, selection);
		expect(standings.map(row => row.tier)).toEqual(['S', 'S', 'A', 'B', 'C', 'D']);
		expect(standings[0].row.race).toBe('Human');
		expect(standings.filter(row => matchesBuildSearch(row, 'Fire'))[0].behind).toBe(5);
		expect(standings[0].build.id).toBe('alpha');
		expect(standings[0].closeRaces).toEqual(['Human', 'Orc']);
	});
	it('changes the reference with class, race and encounter choices and excludes illegal races', () => {
		expect(rankBuilds(data, { ...selection, className: 'Mage' }).standings[0].build.id).toBe('beta');
		const orcs = rankBuilds(data, { ...selection, race: 'Orc' });
		expect(orcs.standings.map(row => row.build.class)).toEqual(['Warrior', 'Warrior']);
		expect(orcs.omitted).toHaveLength(4);
		expect(rankBuilds(data, { ...selection, seconds: 90 }).standings[0].build.id).toBe('beta');
		expect(rankBuilds(data, { ...selection, target: 'Beast' }).standings[0].build.id).toBe('zeta');
	});
	it('sorts equal estimates by build id and race name', () => {
		const tied = { ...data, rows: data.rows.map(row => ({ ...row, dps: 1000 })) };
		expect(rankBuilds(tied, selection).standings.map(row => `${row.build.id}:${row.row.race}`)).toEqual([
			'alpha:Human',
			'beta:Gnome',
			'delta:Gnome',
			'epsilon:Gnome',
			'gamma:Gnome',
			'zeta:Human',
		]);
	});
	it('marks tier boundaries and differences at simulation precision using independent errors', () => {
		const standings = rankBuilds(data, selection).standings;
		expect(standings.find(row => row.build.id === 'beta')?.boundaryUncertain).toBe(true);
		expect(standings[0].boundaryUncertain).toBe(false);
		const close = { ...data, rows: data.rows.map(row => ({ ...row, dps: 1000 - (row.buildId === 'alpha' ? 0 : 0.1) })) };
		expect(rankBuilds(close, selection).standings[1].indistinguishable).toBe(true);
	});
	it('downloads the recorded request with only the selected race and encounter changed', () => {
		const build = buildArenaFixture.builds[0];
		const row = data.rows.find(row => row.buildId === build.id && row.race === 'Orc' && row.seconds === 90 && row.target === 'Beast')!;
		const request = requestForRow(build, row);
		expect(request).toEqual({
			...build.request,
			raid: {
				...build.request.raid,
				parties: [{ ...build.request.raid.parties[0], players: [{ ...build.request.raid.parties[0].players[0], race: 'RaceOrc' }] }],
			},
			encounter: { ...build.request.encounter, duration: 90, targets: [{ ...build.request.encounter.targets[0], mobType: 'MobTypeBeast' }] },
		});
		expect(build.request.encounter.duration).toBe(180);
	});
});
