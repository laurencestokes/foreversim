import { fireEvent, render, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { groupBuilds, type RaceArenaResults, type RaceList, type RaceResult, rankRaces } from './raceArenaData';
import { RaceArenaExplorer } from './RaceArenaExplorer';

const TIERS = [{ name: 'S', maxBehind: 0.5 }, { name: 'A', maxBehind: 1 }, { name: 'B', maxBehind: 1.5 }, { name: 'C', maxBehind: 2.5 }, { name: 'D' }];
const tierOf = (behind: number) => TIERS.find(t => t.maxBehind === undefined || behind <= t.maxBehind)!.name;
const row = (race: string, behind: number): RaceResult => ({ race, dps: 600 * (1 - behind / 100), behind, tier: tierOf(behind), racials: '' });

const list = (seconds: number, target: string, gaps: Record<string, number>): RaceList => ({
	spec: 'warrior',
	class: 'Warrior',
	build: 'Fury 17/34/0',
	talents: '',
	gear: 'p0.bis',
	rotation: 'dps',
	consumables: 'Arena-Melee',
	target,
	seconds,
	races: Object.entries(gaps)
		.map(([race, behind]) => row(race, behind))
		.sort((a, b) => a.behind - b.behind),
});

// Orc wins the short fight on its cooldown, Human the long ones; Dwarf only shines on Beasts.
const data: RaceArenaResults = {
	commit: 'abc1234',
	generated: '2026-10-21T00:00:00Z',
	iterations: 50000,
	fight: { seconds: 180, durations: [90, 180, 360], targetLevel: 63, targetArmor: 3731 },
	tiers: TIERS,
	noise: { maxError: 0.04, maxDifference: 0.11 },
	setups: {},
	lists: [
		list(90, 'Mechanical', { Orc: 0, Human: 0.6, Dwarf: 2 }),
		list(180, 'Mechanical', { Human: 0, Orc: 0.4, Dwarf: 2.1 }),
		list(360, 'Mechanical', { Human: 0, Orc: 0.9, Dwarf: 2.2 }),
		list(90, 'Beast', { Dwarf: 0, Orc: 3, Human: 3.5 }),
		list(180, 'Beast', { Dwarf: 0, Human: 3.2, Orc: 3.4 }),
		list(360, 'Beast', { Dwarf: 0, Human: 3.1, Orc: 3.8 }),
	],
};

describe('rankRaces', () => {
	const [group] = groupBuilds(data);

	it('ranks by the largest gap across the chosen fights, not the average', () => {
		const neutral = [90, 180, 360].map(seconds => ({ seconds, target: 'Mechanical' }));
		const ranking = rankRaces(group, neutral, 'S');
		expect(ranking.map(s => s.race)).toEqual(['Human', 'Orc', 'Dwarf']);
		expect(ranking[0]).toMatchObject({ race: 'Human', worst: 0.6, topTier: 2, conditions: 3 });
	});

	it('follows the fights chosen', () => {
		expect(rankRaces(group, [{ seconds: 90, target: 'Mechanical' }], 'S')[0].race).toBe('Orc');
		expect(rankRaces(group, [{ seconds: 180, target: 'Beast' }], 'S')[0].race).toBe('Dwarf');
	});
});

describe('RaceArenaExplorer', () => {
	const pick = (container: HTMLElement) => within(container).getByTestId('best-race').textContent;

	it('names the safest race for the chosen fights and updates with the filters', () => {
		const { getByTestId, getByRole } = render(<RaceArenaExplorer data={data} />);
		const rowEl = getByTestId('best-race-row');
		expect(pick(rowEl)).toBe('Human');
		expect(rowEl.textContent).toContain('S in 2 of 3 fights');

		// Only the 90 second fight: Orc's opener wins.
		fireEvent.click(getByRole('button', { name: '3 min' }));
		fireEvent.click(getByRole('button', { name: '6 min' }));
		expect(pick(rowEl)).toBe('Orc');

		// The last fight length cannot be switched off.
		fireEvent.click(getByRole('button', { name: '90 s' }));
		expect(getByRole('button', { name: '90 s' }).getAttribute('aria-pressed')).toBe('true');

		// Beasts only: the Dwarf.
		fireEvent.click(getByRole('button', { name: 'Beast' }));
		fireEvent.click(getByRole('button', { name: 'Neutral' }));
		expect(pick(rowEl)).toBe('Dwarf');
	});

	it('shows every race against every fight, and the fight length chart, on demand', () => {
		const { getByRole, getByTestId, queryByTestId } = render(<RaceArenaExplorer data={data} />);
		expect(queryByTestId('race-heatmap')).toBeNull();
		fireEvent.click(getByRole('button', { name: 'Show detail' }));

		const heatmap = getByTestId('race-heatmap');
		const rows = within(heatmap).getAllByRole('row').slice(2);
		expect(rows.map(r => within(r).getByRole('rowheader').textContent)).toEqual(['Human', 'Orc', 'Dwarf']);
		expect(within(heatmap).getByLabelText('Dwarf, 3 min, beast target: tier S, best')).toBeTruthy();
		expect(within(heatmap).getByLabelText('Orc, 6 min, neutral target: tier A, -0.90%')).toBeTruthy();

		const chart = getByTestId('fight-length-chart');
		expect(chart.getAttribute('aria-label')).toContain('neutral target');
		expect(within(chart).getByText('90 s')).toBeTruthy();
		expect(within(chart).getByText('6 min')).toBeTruthy();
	});
});
