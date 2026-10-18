import { BULK_TAB_ID, TabActivationContext } from '@features/hooks/useActivateTab';
import type { ReforgeOptimizerModel } from '@features/reforge/model/reforge_optimizer';
import { ItemSlot } from '@generated/proto/common';
import { ReforgeSettings } from '@sim/settings/reforge_settings';
import { renderWithHost } from '@testing/react';
import { makeBulkPlayer } from '@testing/sim/bulk_player';
import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ReforgeSkippedSlots } from './ReforgeSkippedSlots';

const renderSkippedSlots = ({ simDisabled = false, onClose }: { simDisabled?: boolean; onClose?: () => void }) => {
	const activate = vi.fn();
	const player = makeBulkPlayer({ getGear: () => null });
	const model = {
		settings: new ReforgeSettings(player, {}),
		getSkippedEnchantSlots: () => [{ slot: ItemSlot.ItemSlotMainHand, reason: 'proc' }],
	} as unknown as ReforgeOptimizerModel;
	renderWithHost(
		<TabActivationContext value={activate}>
			<ReforgeSkippedSlots model={model} player={player} onClose={onClose} />
		</TabActivationContext>,
		{ player, sim: player.sim, simDisabled },
	);
	return activate;
};

describe('ReforgeSkippedSlots', () => {
	it('opens the Batch Sim tab from the popover and closes the popover', () => {
		const onClose = vi.fn();
		const activate = renderSkippedSlots({ onClose });

		fireEvent.click(screen.getByTestId('reforge-open-batch-sim'));

		expect(activate).toHaveBeenCalledExactlyOnceWith(BULK_TAB_ID);
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('offers no Batch Sim link on a page without that tab', () => {
		renderSkippedSlots({ simDisabled: true, onClose: vi.fn() });

		expect(screen.getByTestId('reforge-skipped-slots-proc')).not.toBeNull();
		expect(screen.queryByTestId('reforge-open-batch-sim')).toBeNull();
	});
});
