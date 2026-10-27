import { fireEvent, render, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { RaceArenaPage, type RaceArenaResults, type RaceList, tierRange } from './RaceArenaPage';

const TIERS = [{ name: 'S', maxBehind: 0.5 }, { name: 'A', maxBehind: 1 }, { name: 'B', maxBehind: 1.5 }, { name: 'C', maxBehind: 2.5 }, { name: 'D' }];

const list = (overrides: Partial<RaceList>): RaceList => ({
	spec: 'warrior',
	class: 'Warrior',
	build: 'Fury 17/34/0',
	talents: '30305213-550501015050010051',
	gear: 'p0.bis',
	rotation: 'dps_no_reck',
	consumables: 'Arena-Melee',
	target: 'Mechanical',
	races: [],
	...overrides,
});

const fixture: RaceArenaResults = {
	commit: 'b894a61f93d96908bf18e14ee0c2173b07512652',
	generated: '2026-09-24T12:00:00Z',
	iterations: 50000,
	fight: { seconds: 180, targetLevel: 63, targetArmor: 3731 },
	tiers: TIERS,
	noise: { maxError: 0.038, maxDifference: 0.107 },
	setups: {
		'warrior|Fury 17/34/0': {
			equipment: [
				{ slot: 'Main hand', item: { id: 1, name: 'Recorded mace of Testing', suffixId: 3, enchant: 'Recorded enchant' } },
				{ slot: 'Off hand' },
			],
			itemSwap: [{ slot: 'Main hand', item: { id: 2, name: 'Recorded sword' } }],
			talentsString: '30305213-550501015050010051',
			talents: [
				{ name: 'Arms', points: 2, talents: [{ id: 10, name: 'Recorded talent', rank: 2, maxRank: 3 }] },
				{ name: 'Protection', points: 0, talents: [] },
			],
		},
	},
	lists: [
		list({
			races: [
				{ race: 'Undead', dps: 600, behind: 0, tier: 'S', racials: 'Touch of the Grave' },
				{ race: 'Human', dps: 595, behind: 0.83, tier: 'A', relabelled: 'Sword', racials: 'Ruthless (+1% crit)' },
				{ race: 'Troll', dps: 580, behind: 3.33, tier: 'D', racials: 'Berserking' },
			],
		}),
		list({
			spec: 'mage',
			class: 'Mage',
			build: 'Frost 14/0/37',
			consumables: 'Arena-Caster',
			rotation: 'frost',
			races: [
				{ race: 'Human', dps: 560, behind: 0, tier: 'S', relabelled: 'Sword', racials: 'Ruthless (+1% crit) + The Human Spirit (+10% Spirit)' },
				{ race: 'Gnome', dps: 552, behind: 1.43, tier: 'B', racials: 'Expansive Mind' },
			],
		}),
		list({
			target: 'Elemental',
			races: [
				{ race: 'Skyborne', dps: 630, behind: 0, tier: 'S', racials: 'Wind Blessed (+2% haste) + Elemental Insight' },
				{ race: 'Undead', dps: 600, behind: 4.76, tier: 'D', racials: 'Touch of the Grave' },
			],
		}),
	],
};

const chipsIn = (row: Element) => [...row.querySelectorAll('[data-testid="race-chip"] strong')].map(chip => chip.textContent);
const rowOf = (card: HTMLElement, tier: string) => card.querySelector(`[data-testid="tier-row"][data-tier="${tier}"]`)!;

describe('RaceArenaPage', () => {
	it('opens on the main fight length and switches to the others', () => {
		const leader: Record<number, string> = { 90: 'Orc', 180: 'Human', 360: 'Undead' };
		const lengths: RaceArenaResults = {
			...fixture,
			fight: { ...fixture.fight, durations: [90, 180, 360] },
			lists: [90, 180, 360].map(seconds => list({ seconds, races: [{ race: leader[seconds], dps: 600, behind: 0, tier: 'S', racials: '' }] })),
		};
		const { getByRole, getAllByTestId } = render(<RaceArenaPage data={lengths} />);
		const shown = () =>
			getAllByTestId('race-class')
				.map(section => section.textContent)
				.join();
		expect(getByRole('button', { name: '3 min fight' }).getAttribute('aria-pressed')).toBe('true');
		expect(shown()).toContain('Human');
		fireEvent.click(getByRole('button', { name: '90 s fight' }));
		expect(shown()).toContain('Orc');
		expect(shown()).not.toContain('Human');
		fireEvent.click(getByRole('button', { name: '6 min fight' }));
		expect(shown()).toContain('Undead');
	});

	it('explains the upstream presets, weapon rules and measured uncertainty', () => {
		const { getByRole, getByTestId } = render(<RaceArenaPage data={fixture} />);
		const method = getByRole('heading', { name: 'Method' }).closest('section')!;
		expect(method.textContent).toContain('50,000 iterations per race');
		expect(method.textContent).toContain('0.038%');
		expect(method.textContent).toContain('wowsims/forever');
		expect(method.textContent).toContain('actual weapon types');
		expect(method.textContent).toContain('Both Skyborne races');
		expect(method.textContent).toContain('pre-release baseline');
		expect(method.textContent).toContain('development gear');
		expect(method.textContent).toContain('October 26 and 27 racial tuning');
		expect(method.textContent).toContain('Blood Fury has a 2-minute cooldown');
		expect(method.textContent).toContain('Touch of the Grave has a 15-second internal cooldown');
		expect(method.textContent).toContain('Attuned and Charged are removed on entering a raid');
		expect(method.textContent).toContain('Dwarf and Tauren have no racial crit-damage bonus');
		expect(getByTestId('tier-legend').querySelectorAll('li')).toHaveLength(5);
	});

	it('opens the recorded gear and talents, including swaps, and closes with Escape', async () => {
		const { getByRole, findByRole, queryByRole } = render(<RaceArenaPage data={fixture} />);
		fireEvent.click(getByRole('button', { name: 'View setup for Warrior: Fury 17/34/0' }));
		const dialog = await findByRole('dialog', { name: 'Warrior: Fury 17/34/0' });
		const gear = within(dialog).getByRole('tabpanel', { name: 'Gear & enchants' });
		expect(within(gear).getByRole('link', { name: 'Recorded mace of Testing' }).getAttribute('href')).toContain('item=1?level=60&bonus=3');
		expect(gear.textContent).toContain('Recorded enchant');
		expect(gear.textContent).toContain('Empty');
		expect(within(gear).getByRole('heading', { name: 'Weapon swaps' })).toBeTruthy();
		expect(within(gear).getByRole('link', { name: 'Recorded sword' })).toBeTruthy();
		fireEvent.click(within(dialog).getByRole('tab', { name: 'Talents (2/0)' }));
		const talents = within(dialog).getByRole('tabpanel', { name: 'Talents (2/0)' });
		expect(within(talents).getByRole('link', { name: 'Recorded talent' }).getAttribute('href')).toContain('spell=10?rank=2');
		expect(talents.textContent).toContain('2/3');
		expect(talents.textContent).toContain('No points spent.');
		expect(talents.textContent).toContain(fixture.setups['warrior|Fury 17/34/0'].talentsString);
		fireEvent.keyDown(dialog, { key: 'Escape' });
		await waitFor(() => expect(queryByRole('dialog')).toBeNull());
	});

	it('puts every race in its tier row, grouped by class, on the neutral target first', () => {
		const { getAllByTestId } = render(<RaceArenaPage data={fixture} />);
		expect(getAllByTestId('race-class').map(group => group.querySelector('h2')!.textContent)).toEqual(['Warrior', 'Mage']);
		const [warrior] = getAllByTestId('race-list');
		expect(within(warrior).getByRole('heading', { name: 'Fury 17/34/0' })).toBeTruthy();
		expect(chipsIn(rowOf(warrior, 'S'))).toEqual(['Undead']);
		expect(chipsIn(rowOf(warrior, 'A'))).toEqual(['Human']);
		expect(chipsIn(rowOf(warrior, 'D'))).toEqual(['Troll']);
		// An empty tier still has its row, so every list reads the same way down.
		expect(chipsIn(rowOf(warrior, 'B'))).toEqual([]);
		expect(rowOf(warrior, 'B').textContent).toContain('-');
	});

	it('marks a race whose weapons were relabelled, and only that race', () => {
		const { getAllByTestId } = render(<RaceArenaPage data={fixture} />);
		const [warrior] = getAllByTestId('race-list');
		const relabelled = within(warrior).getAllByTestId('relabelled');
		expect(relabelled).toHaveLength(1);
		expect(relabelled[0].textContent).toBe('as Sword');
		expect(relabelled[0].closest('[data-testid="race-chip"]')!.querySelector('strong')!.textContent).toBe('Human');
		expect(rowOf(warrior, 'A').textContent).toContain('-0.83%');
		expect(rowOf(warrior, 'S').textContent).toContain('best');
	});

	it('switches to a situational target', () => {
		const { getByRole, getAllByTestId } = render(<RaceArenaPage data={fixture} />);
		fireEvent.click(getByRole('button', { name: 'Elemental target' }));
		const lists = getAllByTestId('race-list');
		expect(lists).toHaveLength(1);
		expect(chipsIn(rowOf(lists[0], 'S'))).toEqual(['Skyborne']);
		expect(getByRole('button', { name: 'Elemental target' }).getAttribute('aria-pressed')).toBe('true');
	});

	it('stamps the commit the numbers came from', () => {
		const { getByTestId } = render(<RaceArenaPage data={fixture} />);
		const source = getByTestId('race-arena-source');
		expect(source.textContent).toContain('b894a61');
		expect(source.querySelector('a')!.getAttribute('href')).toContain('/commit/b894a61f93d96908bf18e14ee0c2173b07512652');
	});
});

describe('tierRange', () => {
	it('words each tier from the thresholds', () => {
		expect(TIERS.map((_, index) => tierRange(TIERS, index))).toEqual([
			'within 0.5% of the best',
			'0.5% to 1.0% behind',
			'1.0% to 1.5% behind',
			'1.5% to 2.5% behind',
			'more than 2.5% behind',
		]);
	});
});
