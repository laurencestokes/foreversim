import { EquippedItemIcon } from '@features/gear/components/ItemCell';
import { useEquippedItemWowheadDataset } from '@features/gear/hooks/useEquippedItemWowheadDataset';
import { useOpenSelectorModal } from '@features/gear/hooks/useSelectorModal';
import { SelectorModalTabs } from '@features/gear/types';
import type { ItemSlot } from '@generated/proto/common';
import { translateSlotName } from '@i18n/localization';
import { useSimHost } from '@sim/context/SimHostContext';
import { useActionId } from '@sim/hooks/useActionId';
import { usePlayerStore } from '@sim/hooks/usePlayerStore';
import type { WowheadDatasetProps } from '@sim/hooks/useWowheadDataset';
import type { EquippedItem } from '@sim/proto/equipped_item';
import { useMemo } from 'react';

import { createItemSwapGearData } from '../../model/gear_data';

export interface ItemSwapIconViewProps {
	slot: ItemSlot;
	/** The swap item in the slot; `null` draws the slot's empty art. */
	item: EquippedItem | null;
	iconUrl: string;
	name: string;
	href: string;
	/** The swap item's wowhead tooltip, on the icon. */
	wowhead?: WowheadDatasetProps;
	onOpen: () => void;
}

export const ItemSwapIconView = ({ slot, item, iconUrl, name, href, wowhead, onOpen }: ItemSwapIconViewProps) => (
	<EquippedItemIcon
		slot={slot}
		item={item}
		iconUrl={iconUrl}
		href={href}
		wowhead={wowhead}
		onOpen={onOpen}
		label={name || translateSlotName(slot) || undefined}
		className="focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-1 focus-visible:outline-link"
	/>
);

export interface ItemSwapIconProps {
	slot: ItemSlot;
}

export const ItemSwapIcon = ({ slot }: ItemSwapIconProps) => {
	const host = useSimHost();
	const openSelectorModal = useOpenSelectorModal();
	const player = host.player;

	const item = usePlayerStore('itemSwapGear').getEquippedItem(slot);

	const actionId = useMemo(() => item?.asActionId(), [item]);
	const { iconUrl, name, href } = useActionId(actionId);

	const wowheadProps = useEquippedItemWowheadDataset(item);

	return (
		<ItemSwapIconView
			slot={slot}
			item={item}
			iconUrl={iconUrl}
			name={name}
			href={href}
			wowhead={wowheadProps}
			onOpen={() => openSelectorModal(slot, SelectorModalTabs.Items, createItemSwapGearData(player, slot))}
		/>
	);
};
