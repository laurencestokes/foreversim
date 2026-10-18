import type { Meta, StoryObj } from '@storybook/react-vite';
import { fn } from 'storybook/test';

import { EnchantPresetsView } from './EnchantPresets';

const meta = {
	title: 'Features/EnchantPresets/EnchantPresets/EnchantPresets',
	component: EnchantPresetsView,
	tags: ['autodocs'],
	args: {
		presets: [
			{ name: 'Undead', checked: true },
			{ name: 'Beasts', checked: false },
			{ name: 'Outdoor', checked: true },
		],
		onCheckedChange: fn(),
		onNew: fn(),
		onEdit: fn(),
		onDelete: fn(),
		dialog: null,
	},
	argTypes: { dialog: { control: false } },
	decorators: [
		Story => (
			<div className="max-w-80">
				<Story />
			</div>
		),
	],
} satisfies Meta<typeof EnchantPresetsView>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Empty: Story = { args: { presets: [] } };
