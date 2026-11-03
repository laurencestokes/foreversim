import { OtherAction } from '@generated/proto/common';
import type { TalentsConfig } from '@sim/talents/config';
import { render, screen } from '@testing-library/react';
import { expect, it } from 'vitest';

import { BulkTalentsGrid } from './BulkTalentsGrid';

it('shows the synthetic talent icon and name without a game-spell link', () => {
	const config: TalentsConfig<Record<string, never>> = [
		{
			name: 'Preview',
			backgroundUrl: '',
			talents: [{ fancyName: 'Shank', spellId: 0, otherAction: OtherAction.OtherActionShank, location: { rowIdx: 0, colIdx: 0 }, maxPoints: 1 }],
		},
	];
	render(<BulkTalentsGrid config={config} talentsString="1" />);
	expect(screen.getByText('Shank')).toBeTruthy();
	const talent = screen.getByTestId('bulk-talents-grid-talent');
	expect(talent.querySelector<HTMLElement>('.ui-talent-picker-root')!.style.backgroundImage).toContain('.jpg');
	expect(talent.querySelector('a')).toBeNull();
});
