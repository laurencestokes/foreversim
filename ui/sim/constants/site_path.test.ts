import { afterEach, expect, it, vi } from 'vitest';

import { getSitePath, getSpecSitePath } from './other';

afterEach(() => vi.unstubAllEnvs());

it('keeps class links under the configured Pages path', () => {
	vi.stubEnv('VITE_SITE_BASE', '/foreversim/forever/');
	expect(getSpecSitePath('mage', 'dps')).toBe('/foreversim/forever/mage/dps/');
	expect(getSitePath('sim_worker.js')).toBe('/foreversim/forever/sim_worker.js');
	expect(getSitePath('lib.wasm.gz')).toBe('/foreversim/forever/lib.wasm.gz');
});

it('uses the upstream path when no fork path is configured', () => {
	vi.stubEnv('VITE_SITE_BASE', '');
	expect(getSpecSitePath('warrior', 'dps')).toBe('/forever/warrior/dps/');
});
