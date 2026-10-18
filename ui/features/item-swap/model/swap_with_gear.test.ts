import { ConsumesSpec, ItemSlot } from '@generated/proto/common';
import type { Player } from '@sim/player/player';
import { describe, expect, it } from 'vitest';

import { swapWithGear } from './swap_with_gear';

type Items = Partial<Record<ItemSlot, string>>;

/** A gear set reduced to the two calls the exchange makes, with item names standing in for equipped items. */
const gearOf = (items: Items) => ({
	items,
	getEquippedItem: (slot: ItemSlot) => items[slot] ?? null,
	withEquippedItem: (slot: ItemSlot, item: string | null) => gearOf({ ...items, [slot]: item ?? undefined }),
});
type FakeGear = ReturnType<typeof gearOf>;

const playerOf = (gear: Items, swapGear: Items, consumes: ConsumesSpec) => {
	const state = { gear: gearOf(gear), swapGear: gearOf(swapGear), consumes };
	const player = {
		getGear: () => state.gear,
		setGear: (next: FakeGear) => {
			state.gear = next;
		},
		getConsumes: () => ConsumesSpec.clone(state.consumes),
		setConsumes: (next: ConsumesSpec) => {
			state.consumes = ConsumesSpec.clone(next);
		},
		itemSwapSettings: {
			getGear: () => state.swapGear,
			setGear: (next: FakeGear) => {
				state.swapGear = next;
			},
		},
	} as unknown as Player<any>;
	return { player, state };
};

const CONSUMES = ConsumesSpec.create({
	mhImbueId: 1,
	ohImbueId: 2,
	mhWeaponImbueId: 3,
	ohWeaponImbueId: 4,
	swapMhImbueId: 5,
	swapOhImbueId: 6,
	swapMhWeaponImbueId: 7,
	swapOhWeaponImbueId: 8,
	foodId: 9,
});

const imbuesOf = (consumes: ConsumesSpec) => [
	consumes.mhImbueId,
	consumes.ohImbueId,
	consumes.mhWeaponImbueId,
	consumes.ohWeaponImbueId,
	consumes.swapMhImbueId,
	consumes.swapOhImbueId,
	consumes.swapMhWeaponImbueId,
	consumes.swapOhWeaponImbueId,
];

describe('swapWithGear', () => {
	it('exchanges the items and the imbues of both hands with the swap set', () => {
		const { player, state } = playerOf(
			{ [ItemSlot.ItemSlotMainHand]: 'sword', [ItemSlot.ItemSlotOffHand]: 'shield', [ItemSlot.ItemSlotTrinket1]: 'trinket' },
			{ [ItemSlot.ItemSlotMainHand]: 'mace', [ItemSlot.ItemSlotOffHand]: 'dagger' },
			CONSUMES,
		);

		swapWithGear(player, [ItemSlot.ItemSlotMainHand, ItemSlot.ItemSlotOffHand, ItemSlot.ItemSlotTrinket1]);

		expect(state.gear.items).toEqual({ [ItemSlot.ItemSlotMainHand]: 'mace', [ItemSlot.ItemSlotOffHand]: 'dagger' });
		expect(state.swapGear.items).toEqual({
			[ItemSlot.ItemSlotMainHand]: 'sword',
			[ItemSlot.ItemSlotOffHand]: 'shield',
			[ItemSlot.ItemSlotTrinket1]: 'trinket',
		});
		expect(imbuesOf(state.consumes)).toEqual([5, 6, 7, 8, 1, 2, 3, 4]);
		expect(state.consumes.foodId).toBe(9);
	});

	it('leaves the imbues of a hand it does not swap', () => {
		const { player, state } = playerOf({ [ItemSlot.ItemSlotMainHand]: 'sword' }, { [ItemSlot.ItemSlotMainHand]: 'mace' }, CONSUMES);

		swapWithGear(player, [ItemSlot.ItemSlotMainHand]);

		expect(imbuesOf(state.consumes)).toEqual([5, 2, 7, 4, 1, 6, 3, 8]);
	});
});
