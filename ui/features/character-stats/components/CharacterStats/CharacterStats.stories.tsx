import i18n from '@i18n/config';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { storyStatFilter } from '@testing/storybook/fixtures/stat-filters';
import { Table, TableCell, TableRow } from '@ui-kit/Table';

import { CharacterStatsHeaderView, CharacterStatsPanelView } from './CharacterStats';

const ROWS = [
	['Strength', '320'],
	['Agility', '180'],
	['Stamina', '410'],
	['Attack Power', '1,240'],
	['Melee Crit', '24.50%'],
];

const SampleTable = () => (
	<Table variant="plain" flush>
		<tbody>
			{ROWS.map(([label, value]) => (
				<TableRow key={label} className="ui-character-stats-row">
					<TableCell className="ui-character-stats-label">{label}</TableCell>
					<TableCell className="ui-character-stats-value">{value}</TableCell>
				</TableRow>
			))}
		</tbody>
	</Table>
);

const meta = {
	title: 'Features/CharacterStats/CharacterStats',
	component: CharacterStatsPanelView,
	subcomponents: { CharacterStatsHeaderView },
	tags: ['autodocs'],
	args: {
		title: 'Stats',
		filter: storyStatFilter('character-stats-filter', i18n.t('stat_filters.triggers.character_stats')),
		children: <SampleTable />,
	},
	argTypes: { filter: { control: false }, children: { control: false } },
	decorators: [
		Story => (
			<div className="flex w-62 flex-col">
				<Story />
			</div>
		),
	],
} satisfies Meta<typeof CharacterStatsPanelView>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Header: Story = { render: args => <CharacterStatsHeaderView {...args} /> };
