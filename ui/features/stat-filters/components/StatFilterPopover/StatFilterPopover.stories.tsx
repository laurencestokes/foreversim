import { Class } from '@generated/proto/common';
import type { UnitStat } from '@sim/proto/stats';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { STAT_FILTER_UNIVERSE as UNIVERSE, WARRIOR_STAT_DEFAULTS as WARRIOR_DEFAULTS } from '@testing/storybook/fixtures/stat-filters';
import { useState } from 'react';
import { fn, userEvent, within } from 'storybook/test';

import { STAT_GROUPS } from '../../model/stat_groups';
import { StatFilterPopover, type StatFilterPopoverProps } from './StatFilterPopover';

const TWO_GROUPS = STAT_GROUPS.filter(group => group.key === 'attributes' || group.key === 'defense').flatMap(group => group.stats);

/** Holds the list the way the hook does: a change sets the slot, and a reset clears it back to `shown`. */
const StatFilterPopoverStory = ({ shown: initialShown, customised: initialCustomised, onChange, onReset, ...props }: StatFilterPopoverProps) => {
	const [selection, setSelection] = useState<UnitStat[] | null>(initialCustomised ? [...initialShown] : null);
	return (
		<StatFilterPopover
			{...props}
			shown={selection ?? initialShown}
			customised={selection !== null}
			onChange={stats => {
				setSelection(stats);
				onChange(stats);
			}}
			onReset={() => {
				setSelection(null);
				onReset();
			}}
		/>
	);
};

const meta = {
	title: 'Features/StatFilters/StatFilterPopover',
	component: StatFilterPopover,
	args: {
		universe: UNIVERSE,
		shown: WARRIOR_DEFAULTS,
		customised: false,
		playerClass: Class.ClassWarrior,
		onChange: fn(),
		onReset: fn(),
		groupsStorageKey: 'storybook__statFilterGroups',
		ariaLabel: 'Filter character stats',
		testId: 'stat-filter',
	},
	argTypes: {
		universe: { control: false },
		shown: { control: false },
	},
	render: args => <StatFilterPopoverStory {...args} />,
	play: async ({ canvasElement }) => {
		await userEvent.click(within(canvasElement).getByTestId('stat-filter-button'));
	},
} satisfies Meta<typeof StatFilterPopover>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
export const Customised: Story = { args: { shown: WARRIOR_DEFAULTS.slice(0, 3), customised: true } };
export const OnlyTwoGroups: Story = { args: { universe: TWO_GROUPS, shown: TWO_GROUPS.slice(0, 2) } };
