import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, within } from 'storybook/test';

import { RaceArenaPage } from './RaceArenaPage';

const meta = { title: 'App/Race arena', component: RaceArenaPage, parameters: { layout: 'fullscreen' } } satisfies Meta<typeof RaceArenaPage>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Rankings: Story = {};

export const InspectSetup: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await userEvent.click(canvas.getByRole('button', { name: 'View setup for Feral Druid: Hybrid' }));
		const dialog = await within(canvasElement.ownerDocument.body).findByRole('dialog', { name: 'Feral Druid: Hybrid' });
		await expect(within(dialog).getByRole('link', { name: 'Wolfshead Helm' })).toBeVisible();
	},
};
