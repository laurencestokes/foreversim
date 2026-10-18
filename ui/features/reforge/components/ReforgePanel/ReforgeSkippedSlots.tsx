import { BULK_TAB_ID, useActivateTab } from '@features/hooks/useActivateTab';
import type { ReforgeOptimizerModel } from '@features/reforge/model/reforge_optimizer';
import i18n from '@i18n/config';
import { translateSlotName } from '@i18n/localization';
import { useSimHost } from '@sim/context/SimHostContext';
import { useStoreSubscribe } from '@sim/hooks/useStoreSubscribe';
import type { Player } from '@sim/player/player';
import { subscribeAll, subscribeBulkField, subscribePlayerField, subscribeReforgeField } from '@sim/state/subscriptions';
import { Button } from '@ui-kit/Button';
import { Icon } from '@ui-kit/Icon';
import { useMemo } from 'react';

export interface ReforgeSkippedSlotsViewProps {
	/** The slots whose item can take a proc enchant the sim implements. */
	procSlots: ReadonlyArray<string>;
	/** The slots no checked enchant preset lists. */
	unlistedSlots: ReadonlyArray<string>;
	/** Opens the Batch Sim tab. Without it the proc slots have no button. */
	onOpenBatchSim?: () => void;
}

const SlotList = ({ slots }: { slots: ReadonlyArray<string> }) => (
	<ul className="mb-0">
		{slots.map(slot => (
			<li key={slot}>{slot}</li>
		))}
	</ul>
);

export const ReforgeSkippedSlotsView = ({ procSlots, unlistedSlots, onOpenBatchSim }: ReforgeSkippedSlotsViewProps) => {
	if (!procSlots.length && !unlistedSlots.length) return null;

	return (
		<div className="mb-2 flex flex-col gap-2" data-testid="reforge-skipped-slots">
			{procSlots.length > 0 && (
				<div className="flex flex-col items-start gap-1" data-testid="reforge-skipped-slots-proc">
					<p className="mb-0">{i18n.t('sidebar.buttons.suggest_reforges.skipped_slots_description')}</p>
					<SlotList slots={procSlots} />
					{onOpenBatchSim && (
						<Button variant="link" size="sm" data-testid="reforge-open-batch-sim" onClick={onOpenBatchSim}>
							{i18n.t('sidebar.buttons.suggest_reforges.go_to_batch_sim')}
							<Icon name="arrow-right" />
						</Button>
					)}
				</div>
			)}
			{unlistedSlots.length > 0 && (
				<div className="flex flex-col gap-1" data-testid="reforge-skipped-slots-unlisted">
					<p className="mb-0">{i18n.t('sidebar.buttons.suggest_reforges.skipped_slots_unlisted')}</p>
					<SlotList slots={unlistedSlots} />
				</div>
			)}
		</div>
	);
};

export interface ReforgeSkippedSlotsProps {
	model: ReforgeOptimizerModel;
	player: Player<any>;
	/** Closes the popover the panel sits in once the Batch Sim tab opens. */
	onClose?: () => void;
}

/**
 * The equipped slots the optimizer leaves alone: those whose item can take a proc enchant the sim
 * implements, and with the enchant presets on, those no checked preset lists.
 */
export const ReforgeSkippedSlots = ({ model, player, onClose }: ReforgeSkippedSlotsProps) => {
	const host = useSimHost();
	const activateTab = useActivateTab();
	const subscribe = useMemo(
		() =>
			subscribeAll([
				subscribePlayerField(player, 'gear'),
				// The enchants the player can take depend on Enchanting.
				subscribePlayerField(player, 'profession1'),
				subscribePlayerField(player, 'profession2'),
				// The freezeItemSlots counter also moves when a single slot is frozen or unfrozen.
				subscribeReforgeField(model.settings, 'freezeItemSlots'),
				subscribeReforgeField(model.settings, 'useEnchantPresets'),
				// Checking a preset, or saving or deleting a checked one, bumps the batch settings counter.
				subscribeBulkField(player, 'settings'),
			]),
		[player, model],
	);
	const skipped = useStoreSubscribe(subscribe, () => model.getSkippedEnchantSlots(player.getGear()));
	const slotsFor = (reason: 'proc' | 'unlisted') => skipped.filter(skip => skip.reason === reason).map(skip => translateSlotName(skip.slot));
	const openBatchSim = host.simDisabled
		? undefined
		: () => {
				activateTab(BULK_TAB_ID);
				onClose?.();
			};

	return <ReforgeSkippedSlotsView procSlots={slotsFor('proc')} unlistedSlots={slotsFor('unlisted')} onOpenBatchSim={openBatchSim} />;
};
