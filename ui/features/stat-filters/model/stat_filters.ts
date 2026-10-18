import { DatabaseFilters, StatFilters, StatSelection } from '@generated/proto/ui';
import { UnitStat } from '@sim/proto/stats';
import { memoLast } from '@sim/utils/misc';

export type StatFilterSlot = keyof StatFilters;

export interface StatFilterScope {
	/** The spec's list for the panel, shown while its slot is unset. */
	defaults: ReadonlyArray<UnitStat>;
	/** Every stat the panel can toggle, in display order. */
	universe: ReadonlyArray<UnitStat>;
}

const contains = (stats: ReadonlyArray<UnitStat>, stat: UnitStat) => stats.some(other => other.equals(stat));

export const statsKey = (stats: ReadonlyArray<UnitStat>): string => stats.map(stat => stat.getKey()).join(',');

export const isCustomised = (filters: DatabaseFilters, slot: StatFilterSlot): boolean => filters.statFilters?.[slot] !== undefined;

/** The slot's stored stats, or the defaults while it is unset, kept to the universe and in its order. */
export const shownStats = (filters: DatabaseFilters, slot: StatFilterSlot, scope: StatFilterScope): UnitStat[] => {
	const selection = filters.statFilters?.[slot];
	const wanted = selection ? selection.stats.map(stat => UnitStat.fromProto(stat)) : scope.defaults;
	return scope.universe.filter(stat => contains(wanted, stat));
};

/**
 * Reads the slot's shown stats off the filters, worked out again only for a new filters object, and hands back the
 * same array while the slot shows the same stats, so a write to another filter changes nothing downstream.
 */
export const shownStatsReader = (slot: StatFilterSlot, scope: StatFilterScope): ((filters: DatabaseFilters) => UnitStat[]) => {
	const byFilters = memoLast<UnitStat[]>();
	const byKey = memoLast<UnitStat[]>();
	return filters =>
		byFilters([filters], () => {
			const shown = shownStats(filters, slot, scope);
			return byKey([statsKey(shown)], () => shown);
		});
};

/**
 * Sets the slot to the given stats, or clears it back to the defaults for `undefined`. Changes the filters it is given
 * and returns them, so a caller hands it a copy (`sim.getFilters()` returns one).
 */
export const withSelection = (filters: DatabaseFilters, slot: StatFilterSlot, stats: ReadonlyArray<UnitStat> | undefined): DatabaseFilters => {
	if (!stats && !filters.statFilters) return filters;
	filters.statFilters ??= StatFilters.create();
	filters.statFilters[slot] = stats ? StatSelection.create({ stats: stats.map(stat => stat.toProto()) }) : undefined;
	return filters;
};
