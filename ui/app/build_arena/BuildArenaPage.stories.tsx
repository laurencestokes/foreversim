import i18n from '@i18n/config';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { buildArenaFixture } from '@testing/build_arena_fixture';
import { expect, userEvent, within } from 'storybook/test';

import english from '../../../assets/locales/en/build_arena.json';
import { BuildArenaPageView } from './BuildArenaPage';

const meta = {
	title: 'App/Build arena',
	beforeEach: () => {
		i18n.addResourceBundle('en', 'build_arena', english, true, true);
	},
	component: BuildArenaPageView,
	args: { data: buildArenaFixture },
	parameters: { layout: 'fullscreen', spec: 'none' },
} satisfies Meta<typeof BuildArenaPageView>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Rankings: Story = {};
export const EmptyComparison: Story = { args: { data: { ...buildArenaFixture, builds: [], rows: [] } } };
export const InspectRecordedSetup: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await userEvent.click(canvas.getByRole('button', { name: 'Inspect Warrior: Fury' }));
		const dialog = await within(canvasElement.ownerDocument.body).findByRole('dialog', { name: 'Warrior: Fury' });
		await expect(within(dialog).getByRole('link', { name: 'Download Go RaidSimRequest' })).toBeVisible();
		await userEvent.click(within(dialog).getByRole('button', { name: 'Gear, talents and swaps' }));
		const setup = await within(canvasElement.ownerDocument.body).findByRole('dialog', { name: 'Warrior: Fury: recorded gear and talents' });
		await expect(within(setup).getByRole('link', { name: 'Lionheart Helm' })).toBeVisible();
	},
};
