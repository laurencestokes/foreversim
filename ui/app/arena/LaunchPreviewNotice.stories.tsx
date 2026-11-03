import i18n from '@i18n/config';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { launchPreviewFixture } from '@testing/launch_preview_fixture';

import english from '../../../assets/locales/en/build_arena.json';
import { LaunchPreviewNotice } from './LaunchPreviewNotice';

const meta = {
	title: 'App/Launch preview notice',
	component: LaunchPreviewNotice,
	beforeEach: () => {
		i18n.addResourceBundle('en', 'build_arena', english, true, true);
	},
	args: { preview: launchPreviewFixture },
	parameters: { spec: 'none' },
} satisfies Meta<typeof LaunchPreviewNotice>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Projection: Story = {};
