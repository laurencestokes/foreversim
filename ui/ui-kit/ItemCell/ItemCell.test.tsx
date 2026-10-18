import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { ItemCell, ItemCellIcon } from './ItemCell';

describe('ItemCellIcon', () => {
	it('keeps an empty item-level badge, which an empty gear slot renders', () => {
		const { container } = render(
			<ItemCellIcon ilvl={null}>
				<a />
			</ItemCellIcon>,
		);

		expect(container.querySelector('[data-testid="item-picker-ilvl"]')?.textContent).toBe('');
	});

	it('drops the badge when the caller omits it', () => {
		const { container } = render(
			<ItemCellIcon>
				<a />
			</ItemCellIcon>,
		);

		expect(container.querySelector('[data-testid="item-picker-ilvl"]')).toBeNull();
	});
});

describe('ItemCell', () => {
	it('opens with the framed icon, ahead of the name and labels', () => {
		const { container } = render(
			<ItemCell
				icon={
					<ItemCellIcon ilvl="66">
						<a />
					</ItemCellIcon>
				}
				name="Head"
			/>,
		);

		expect(container.firstElementChild!.firstElementChild!.getAttribute('data-testid')).toBe('item-picker-icon-wrapper');
	});

	it('puts the trailing action after the labels, outside the label stack', () => {
		const { container } = render(<ItemCell icon={<a />} name="Head" labels={<a className="test-reforge" />} action={<button className="favourite" />} />);

		const root = container.firstElementChild!;
		expect(root.lastElementChild?.classList.contains('favourite')).toBe(true);
		expect(root.querySelector('[data-testid="item-picker-labels-container"] .favourite')).toBeNull();
	});

	it('marks the name and labels with their alignment, and leaves them unmarked without one', () => {
		const { container, rerender } = render(<ItemCell icon={<a />} name="Head" align="end" />);
		const labels = () => container.querySelector<HTMLElement>('[data-testid="item-picker-labels-container"]')!;
		expect(labels().dataset.align).toBe('end');
		expect(container.firstElementChild!.hasAttribute('data-align')).toBe(false);

		rerender(<ItemCell icon={<a />} name="Head" />);
		expect(labels().hasAttribute('data-align')).toBe(false);
	});
});
