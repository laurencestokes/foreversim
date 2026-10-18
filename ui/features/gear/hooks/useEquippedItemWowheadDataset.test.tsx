import { ItemSlot } from '@generated/proto/common';
import { UIItem } from '@generated/proto/ui';
import { SimHostProvider } from '@sim/context/SimHostContext';
import { EquippedItem } from '@sim/proto/equipped_item';
import { Gear } from '@sim/proto/gear';
import { createSimStore, patchKeyed, type PlayerSlice, seedKeyed } from '@sim/state/sim_store';
import { act, render, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { useEquippedItemWowheadDataset } from './useEquippedItemWowheadDataset';

const KEY = 3;

const equipped = (id: number) => new EquippedItem({ item: UIItem.create({ id, scalingOptions: { 0: { ilvl: 58 } } }) });

// Lightforge Belt and Lightforge Bracers, two pieces of one set.
const belt = equipped(16723);
const bracers = equipped(16722);

const Probe = ({ item }: { item: EquippedItem }) => {
	const wowhead = useEquippedItemWowheadDataset(item);
	return <a {...wowhead} />;
};

describe('useEquippedItemWowheadDataset', () => {
	it('lists the set pieces equipped in other slots after this one', async () => {
		const store = createSimStore();
		seedKeyed(store, 'players', KEY, { gear: new Gear({ [ItemSlot.ItemSlotWaist]: belt }), v: {} } as unknown as PlayerSlice);
		const host = { player: { sim: { store }, storeKey: KEY } } as never;
		const { container } = render(
			<SimHostProvider host={host}>
				<Probe item={belt} />
			</SimHostProvider>,
		);
		const anchor = container.querySelector('a')!;
		await waitFor(() => expect(anchor.dataset.wowhead).toContain('pcs=16723'));

		act(() =>
			patchKeyed(store, 'players', KEY, { gear: new Gear({ [ItemSlot.ItemSlotWaist]: belt, [ItemSlot.ItemSlotWrist]: bracers }) }, ['gear'] as never),
		);

		await waitFor(() => expect(anchor.dataset.wowhead).toContain('pcs=16722:16723'));
	});
});
