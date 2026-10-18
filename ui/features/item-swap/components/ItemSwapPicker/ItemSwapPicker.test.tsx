import { OpenSelectorModalContext } from '@features/gear/hooks/useSelectorModal';
import { ItemSlot } from '@generated/proto/common';
import i18n from '@i18n/config';
import { SimHostProvider } from '@sim/context/SimHostContext';
import type { Player } from '@sim/player/player';
import { ItemSwapGear } from '@sim/proto/gear';
import { createSimStore, patchKeyed, PLAYER_FIELDS, seedKeyed, zeroVersions } from '@sim/state/sim_store';
import { fakeHost } from '@testing/sim/fake_host';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('@sim/state/subscriptions', async () => (await import('@testing/sim/subscriptions')).mockSubscriptions());
vi.mock('@sim/hooks/useActionId', () => ({ useActionId: () => ({ iconUrl: '', name: '', href: '', ready: true }) }));
vi.mock('@features/inputs/components/ConnectedBooleanPicker', () => ({ ConnectedBooleanPicker: () => <div className="boolean-picker-root" /> }));

const { ItemSwapPicker, ItemSwapPickerView } = await import('./ItemSwapPicker');

const SLOTS = [ItemSlot.ItemSlotMainHand, ItemSlot.ItemSlotOffHand, ItemSlot.ItemSlotTrinket1];

const setup = (enabled: boolean) => {
	const store = createSimStore();
	const storeKey = 0;
	seedKeyed(store, 'players', storeKey, { itemSwapEnabled: enabled, itemSwapGear: new ItemSwapGear({}), v: zeroVersions(PLAYER_FIELDS) } as never);
	const player = {
		storeKey,
		sim: { store },
		itemSwapSettings: { getEnableItemSwap: () => enabled, getItem: () => null },
	} as unknown as Player<any>;
	const host = fakeHost({ player });
	const view = render(
		<SimHostProvider host={host}>
			<OpenSelectorModalContext value={vi.fn()}>
				<ItemSwapPicker itemSlots={SLOTS}>
					<div data-testid="swap-rows" />
				</ItemSwapPicker>
			</OpenSelectorModalContext>
		</SimHostProvider>,
	);
	const setEnabled = (next: boolean) => act(() => patchKeyed(store, 'players', storeKey, { itemSwapEnabled: next }, ['itemSwap']));
	return { ...view, setEnabled };
};

describe('ItemSwapPicker', () => {
	it('puts one icon per swap slot directly in the icon group', () => {
		const { container } = setup(true);

		const group = container.querySelector('[data-testid="item-swap-icon-grid"]')!;
		expect(group.children).toHaveLength(SLOTS.length);
		expect([...group.children].every(child => child.getAttribute('data-testid') === 'item-picker-icon-wrapper')).toBe(true);
	});

	it('renders no picker row while item swap is off, and hides the box', () => {
		const { container } = setup(false);

		expect(container.querySelector('[data-testid="input-item-swap-container"]')).toBeNull();
		expect(container.querySelector('[data-testid="item-swap-icon-grid"]')).toBeNull();
		expect(screen.getByTestId('item-swap-box').hasAttribute('hidden')).toBe(true);
	});

	it('puts the extra rows in the box under the picker row', () => {
		setup(true);

		const box = screen.getByTestId('item-swap-box');
		expect(box.hasAttribute('hidden')).toBe(false);
		expect(box.children[0]).toBe(screen.getByTestId('input-item-swap-container'));
		expect(box.children[1]).toBe(screen.getByTestId('swap-rows'));
	});

	it('keeps the extra rows mounted as item swap turns off and on', () => {
		const { setEnabled } = setup(true);
		const rows = screen.getByTestId('swap-rows');

		setEnabled(false);
		expect(screen.getByTestId('item-swap-box').hasAttribute('hidden')).toBe(true);
		expect(screen.getByTestId('swap-rows')).toBe(rows);

		setEnabled(true);
		expect(screen.getByTestId('item-swap-box').hasAttribute('hidden')).toBe(false);
		expect(screen.getByTestId('swap-rows')).toBe(rows);
	});
});

const renderView = (props: Partial<Parameters<typeof ItemSwapPickerView>[0]> = {}) => {
	const onSwap = vi.fn();
	render(
		<ItemSwapPickerView onSwap={onSwap} {...props}>
			<span data-testid="slot-icon" />
			<span data-testid="slot-icon" />
		</ItemSwapPickerView>,
	);
	return { onSwap };
};

describe('ItemSwapPickerView', () => {
	it('renders the row as a split field', () => {
		renderView();
		const row = screen.getByTestId('input-item-swap-container');
		expect(row.getAttribute('data-layout')).toBe('split');
	});

	it('names the row, swap button and icons, as one group with the row label', () => {
		renderView();
		const group = screen.getByRole('group', { name: i18n.t('settings_tab.other.item_swap.label') });
		expect(group).toBe(screen.getByTestId('input-item-swap-container'));
		expect(group.querySelectorAll('[data-testid="slot-icon"]')).toHaveLength(2);
		expect(group.contains(screen.getByTestId('gear-swap-icon'))).toBe(true);
	});

	it('swaps on the swap button', () => {
		const { onSwap } = renderView();
		fireEvent.click(screen.getByRole('button', { name: i18n.t('settings_tab.other.item_swap.tooltip') }));
		expect(onSwap).toHaveBeenCalledOnce();
	});

	it.each([undefined, 'Swap before the pull.'])('shows the note %j under the row only when there is one', note => {
		renderView({ note });
		expect(document.querySelector('p')?.textContent ?? null).toBe(note ?? null);
	});
});
