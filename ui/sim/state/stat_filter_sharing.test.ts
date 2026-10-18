import { DatabaseFilters, IndividualSimSettings, SimSettings, StatFilters } from '@generated/proto/ui';
import { makeIndividualSim, serveDatabase } from '@testing/sim/individual_sim';
import { envelope, senderStatFilters as senderFilters, viewerStatFilters as viewerFilters } from '@testing/sim/stat_filters';
import { beforeAll, describe, expect, it, vi } from 'vitest';

import { withoutStatFilters, withViewerStatFilters } from './stat_filter_sharing';

vi.mock('@sim/workers/worker_pool', async () => (await import('@testing/sim/worker_pool')).idleWorkerPool());

const settingsWith = (filters: DatabaseFilters | undefined) => envelope(SimSettings.create({ iterations: 1234, filters }));

describe('withoutStatFilters', () => {
	it('clears the stat filters on the envelope it is given and keeps every other filter', () => {
		const filters = DatabaseFilters.create({ minIlvl: 50, showAllEnchants: true, favoriteItems: [7], statFilters: senderFilters });
		const proto = settingsWith(filters);

		const stripped = withoutStatFilters(proto);

		expect(stripped).toBe(proto);
		expect(DatabaseFilters.equals(stripped.settings!.filters!, { ...filters, statFilters: undefined })).toBe(true);
	});

	it('passes a proto without settings or filters through', () => {
		expect(IndividualSimSettings.equals(withoutStatFilters(envelope()), envelope())).toBe(true);
		expect(IndividualSimSettings.equals(withoutStatFilters(settingsWith(undefined)), settingsWith(undefined))).toBe(true);
	});
});

describe('withViewerStatFilters', () => {
	beforeAll(() => serveDatabase());

	const viewerSim = async (statFilters: StatFilters | undefined) => {
		const { sim } = await makeIndividualSim();
		sim.setFilters(DatabaseFilters.create({ statFilters }));
		return sim;
	};

	it("puts the viewer's stat filters over the envelope's and keeps every other filter it carries", async () => {
		const sim = await viewerSim(viewerFilters);
		const carried = DatabaseFilters.create({ minIlvl: 50, oneHandedWeapons: true, favoriteItems: [7], statFilters: senderFilters });
		const proto = settingsWith(carried);

		const patched = withViewerStatFilters(proto, sim);

		expect(patched).toBe(proto);
		expect(DatabaseFilters.equals(patched.settings!.filters!, { ...carried, statFilters: viewerFilters })).toBe(true);
	});

	it("clears the envelope's stat filters when the viewer has none", async () => {
		const sim = await viewerSim(undefined);
		const carried = DatabaseFilters.create({ twoHandedWeapons: true, statFilters: senderFilters });

		const patched = withViewerStatFilters(settingsWith(carried), sim);

		expect(DatabaseFilters.equals(patched.settings!.filters!, { ...carried, statFilters: undefined })).toBe(true);
	});

	it('leaves an envelope without settings or filters as it is', async () => {
		const sim = await viewerSim(viewerFilters);
		expect(IndividualSimSettings.equals(withViewerStatFilters(envelope(), sim), envelope())).toBe(true);
		expect(IndividualSimSettings.equals(withViewerStatFilters(settingsWith(undefined), sim), settingsWith(undefined))).toBe(true);
	});
});

describe('Sim.applyDefaults', () => {
	beforeAll(() => serveDatabase());

	it("keeps the viewer's stat filters and favourites and resets every other filter", async () => {
		const { sim, host } = await makeIndividualSim();
		sim.setFilters(DatabaseFilters.create({ minIlvl: 50, favoriteItems: [7], statFilters: viewerFilters }));

		host.applyDefaults();

		const filters = sim.getFilters();
		expect(filters.statFilters && StatFilters.equals(filters.statFilters, viewerFilters)).toBe(true);
		expect(filters.favoriteItems).toEqual([7]);
		expect(filters.minIlvl).toBe(0);
	});
});
