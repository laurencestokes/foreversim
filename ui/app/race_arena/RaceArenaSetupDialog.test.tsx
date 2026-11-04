import { render } from '@testing-library/react';
import { expect, it } from 'vitest';

import { RaceArenaSetupDialog } from './RaceArenaSetupDialog';

it('shows the recorded reaction-time range with the saved setup', () => {
	const { getByRole } = render(
		<RaceArenaSetupDialog
			open
			onOpenChange={() => {}}
			title="Recorded Hunter setup"
			gear="Preset"
			rotation="Default"
			commit={'a'.repeat(40)}
			setup={{ equipment: [], talentsString: '', talents: [], reactionTime: { minMs: 150, maxMs: 350 } }}
		/>,
	);
	expect(getByRole('dialog', { name: 'Recorded Hunter setup' }).textContent).toContain(
		'Each reaction time is drawn uniformly between 150 and 350 milliseconds.',
	);
});
