import i18n from '@i18n/config';
import { buildArenaFixture } from '@testing/build_arena_fixture';
import { fireEvent, render, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

import english from '../../../assets/locales/en/build_arena.json';
import { BuildArenaPageView } from './BuildArenaPage';

beforeAll(() => i18n.addResourceBundle('en', 'build_arena', english, true, true));
afterAll(() => i18n.removeResourceBundle('en', 'build_arena'));
vi.mock('../landing/footer_content', () => ({ fetchFooterContent: async () => null }));

const pick = async (root: HTMLElement, label: string, option: string) => {
	const user = userEvent.setup();
	await user.click(within(root).getByRole('combobox', { name: label }));
	await user.click(within(root.ownerDocument.body).getByRole('option', { name: option }));
};

describe('Build Arena', () => {
	it('distinguishes shared preset labels by spec and exposes Hunter melee assumptions beside the result', () => {
		const data = {
			...buildArenaFixture,
			builds: buildArenaFixture.builds.map((build, index) =>
				index < 2
					? { ...build, class: 'Shaman', spec: index === 0 ? 'elemental_shaman' : 'enhancement_shaman', name: 'Testing' }
					: index === 2
						? { ...build, class: 'Hunter', spec: 'hunter', name: 'SV Weave', caveats: ['Weaving assumes uninterrupted access to melee range.'] }
						: index === 3
							? { ...build, class: 'Mage', spec: 'mage', name: 'Fire / Fire' }
							: build,
			),
		};
		const { getByRole, getByText, queryByText } = render(<BuildArenaPageView data={data} />);
		expect(getByRole('button', { name: 'Inspect Elemental Shaman: Testing' })).toBeTruthy();
		expect(getByRole('button', { name: 'Inspect Enhancement Shaman: Testing' })).toBeTruthy();
		expect(getByRole('button', { name: 'Inspect Mage: Fire' })).toBeTruthy();
		expect(queryByText('Fire / Fire')).toBeNull();
		expect(getByText('Weaving assumes uninterrupted access to melee range.')).toBeTruthy();
	});
	it('keeps the reference and tiers while search hides its row', () => {
		const { getByRole, getByText, queryByRole } = render(<BuildArenaPageView data={buildArenaFixture} />);
		fireEvent.input(getByRole('textbox', { name: 'Find a build' }), { target: { value: 'Fire' } });
		const rows = within(getByRole('table', { name: 'Build rankings' })).getAllByRole('row');
		expect(rows).toHaveLength(2);
		expect(rows[1].textContent).toContain('5.00%');
		expect(rows[1].getAttribute('data-tier')).toBe('S');
		expect(getByText(/Reference: Warrior: Fury/).textContent).toContain('Showing 1 of 6');
		fireEvent.input(getByRole('textbox', { name: 'Find a build' }), { target: { value: 'missing' } });
		expect(getByRole('status').textContent).toContain('No builds match');
		expect(queryByRole('table', { name: 'Build rankings' })).toBeNull();
	});
	it('updates the population and excludes illegal class/race combinations', async () => {
		const { container, getByRole, getByText } = render(<BuildArenaPageView data={buildArenaFixture} />);
		await pick(container, 'Compare builds', 'Mage');
		expect(getByText(/Reference: Mage: Fire/)).toBeTruthy();
		await pick(container, 'Race selection', 'Orc');
		expect(getByRole('status').textContent).toContain('No recorded builds');
		expect(getByText(/2 builds have no legal Orc result/)).toBeTruthy();
		await pick(container, 'Compare builds', 'All classes');
		expect(within(getByRole('table', { name: 'Build rankings' })).getAllByRole('row')).toHaveLength(3);
		expect(getByText(/4 builds have no legal Orc result/)).toBeTruthy();
	});
	it('changes the leader for the selected duration and target', async () => {
		const { container, getByText } = render(<BuildArenaPageView data={buildArenaFixture} />);
		await pick(container, 'Fight length', '90 seconds');
		expect(getByText(/Reference: Mage: Fire/)).toBeTruthy();
		await pick(container, 'Target type', 'Beast');
		expect(getByText(/Reference: Warrior: Arms/)).toBeTruthy();
	});
	it('opens recorded details, downloads the selected request and returns focus from gear inspection', async () => {
		const user = userEvent.setup();
		const { getByRole } = render(<BuildArenaPageView data={buildArenaFixture} />);
		await user.click(getByRole('button', { name: 'Inspect Warrior: Fury' }));
		const dialog = getByRole('dialog', { name: 'Warrior: Fury' });
		expect(within(dialog).getByRole('table', { name: 'All legal races in this encounter' })).toBeTruthy();
		expect(within(dialog).getByText('Flask of Supreme Power')).toBeTruthy();
		expect(within(dialog).getByText('Unknown item (ID 999999)')).toBeTruthy();
		await pick(dialog, 'Race for saved request', 'Orc');
		const download = within(dialog).getByRole('link', { name: 'Download Go RaidSimRequest' });
		const request = JSON.parse(decodeURIComponent(download.getAttribute('href')!.split(',').slice(1).join(',')));
		expect(request.raid.parties[0].players[0].race).toBe('RaceOrc');
		expect(request.encounter.duration).toBe(180);
		const gearButton = within(dialog).getByRole('button', { name: 'Gear, talents and swaps' });
		await user.click(gearButton);
		const gear = getByRole('dialog', { name: 'Warrior: Fury: recorded gear and talents' });
		expect(within(gear).getByRole('link', { name: 'Lionheart Helm' })).toBeTruthy();
		await user.keyboard('{Escape}');
		expect(getByRole('dialog', { name: 'Warrior: Fury' })).toBeTruthy();
		expect(document.activeElement).toBe(gearButton);
		await user.keyboard('{Escape}');
		expect(document.activeElement).toBe(getByRole('button', { name: 'Inspect Warrior: Fury' }));
	});
});
