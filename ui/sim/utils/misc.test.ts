import { describe, expect, it, vi } from 'vitest';

import { memoLast } from './misc';

describe('memoLast', () => {
	it('computes once while the deps keep their identities, and again when one changes', () => {
		const memo = memoLast<object>();
		const compute = vi.fn(() => ({}));
		const dep = {};
		const first = memo([dep, 1], compute);
		expect(memo([dep, 1], compute)).toBe(first);
		expect(compute).toHaveBeenCalledTimes(1);

		const second = memo([{}, 1], compute);
		expect(second).not.toBe(first);
		expect(memo([dep, 2], compute)).not.toBe(second);
		expect(compute).toHaveBeenCalledTimes(3);
	});

	it('holds one entry only', () => {
		const memo = memoLast<number>();
		const compute = vi.fn(() => compute.mock.calls.length);
		memo(['a'], compute);
		memo(['b'], compute);
		memo(['a'], compute);
		expect(compute).toHaveBeenCalledTimes(3);
	});
});
