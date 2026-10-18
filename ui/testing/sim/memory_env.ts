import type { Env } from '@sim/state/env';

export const memoryEnv = (location: Partial<Env['location']> = {}): Env => {
	const stored = new Map<string, string>();
	return {
		storage: {
			getItem: key => stored.get(key) ?? null,
			setItem: (key, value) => void stored.set(key, value),
			removeItem: key => void stored.delete(key),
		},
		location: { hash: '', search: '', href: 'http://localhost/', hostname: 'localhost', setHash: () => {}, ...location },
		hardwareConcurrency: 1,
		onPageHide: () => () => {},
	};
};
