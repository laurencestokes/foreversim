import { IndividualSimSettings, StatFilters } from '@generated/proto/ui';

import type { Sim } from '../sim';

// Stat filters belong to the viewer: an exported build carries none and an imported one keeps the viewer's.
// Both helpers change the envelope they are given and return it.

export const withoutStatFilters = (proto: IndividualSimSettings): IndividualSimSettings => {
	if (proto.settings?.filters) proto.settings.filters.statFilters = undefined;
	return proto;
};

// Loading settings replaces the filters whole, so filters the envelope carries take the viewer's stat filters.
// Settings without filters need nothing: the defaults they load keep the viewer's.
export const withViewerStatFilters = (proto: IndividualSimSettings, sim: Sim): IndividualSimSettings => {
	const filters = proto.settings?.filters;
	if (filters) {
		const { statFilters } = sim.store.getState().sim.filters;
		filters.statFilters = statFilters && StatFilters.clone(statFilters);
	}
	return proto;
};
