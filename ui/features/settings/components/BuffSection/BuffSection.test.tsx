import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { BuffSection, type BuffSectionProps } from './BuffSection';

const renderSection = (props: Omit<BuffSectionProps, 'children'>) =>
	render(
		<BuffSection {...props}>
			<span data-testid="buff-pickers" />
		</BuffSection>,
	);

const block = (name: string) => document.querySelector(`[data-block="${name}"]`) as HTMLElement;

describe('BuffSection', () => {
	it.each<[BuffSectionProps['section'], string, string | null]>([
		['raid_buffs', 'buffs-settings', 'settings_tab.raid_buffs.description'],
		['party_buffs', 'party-buffs-settings', 'settings_tab.party_buffs.description'],
		['debuffs', 'debuffs-settings', null],
	])('titles the %s block, with its own description where it has one', (section, name, description) => {
		renderSection({ section, empty: false });

		expect(screen.getByTestId('content-block-title').textContent).toBe(`settings_tab.${section}.title`);
		expect(block(name).querySelector('p')?.textContent ?? null).toBe(description);
		expect(block(name).contains(screen.getByTestId('buff-pickers'))).toBe(true);
	});

	it.each<BuffSectionProps['section']>(['raid_buffs', 'debuffs'])('puts the filter in the %s header', section => {
		render(
			<BuffSection section={section} empty={true} filter={<button data-testid="buff-filter" />}>
				<span />
			</BuffSection>,
		);

		expect(screen.getByTestId('content-block-header').contains(screen.getByTestId('buff-filter'))).toBe(true);
	});

	it('drops the body of an empty section and keeps its header', () => {
		renderSection({ section: 'raid_buffs', empty: true });

		expect(screen.getByTestId('content-block-title').textContent).toBe('settings_tab.raid_buffs.title');
		expect(screen.queryByTestId('content-block-body')).toBeNull();
		expect(screen.queryByTestId('buff-pickers')).toBeNull();
	});
});
