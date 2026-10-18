import { PseudoStat, Stat } from '@generated/proto/common';
import { DatabaseFilters } from '@generated/proto/ui';
import { UnitStat } from '@sim/proto/stats';
import { describe, expect, it } from 'vitest';

import { isCustomised, shownStats, shownStatsReader, type StatFilterScope, withSelection } from './stat_filters';

const strength = UnitStat.fromStat(Stat.StatStrength);
const agility = UnitStat.fromStat(Stat.StatAgility);
const stamina = UnitStat.fromStat(Stat.StatStamina);
const meleeHit = UnitStat.fromPseudoStat(PseudoStat.PseudoStatMeleeHitPercent);
const spellDamage = UnitStat.fromStat(Stat.StatSpellDamage);
const intellect = UnitStat.fromStat(Stat.StatIntellect);

const scope: StatFilterScope = {
	defaults: [spellDamage, meleeHit, strength],
	universe: [strength, agility, stamina, meleeHit, spellDamage],
};

const keys = (stats: ReadonlyArray<UnitStat>) => stats.map(stat => stat.getKey());
const inUniverseOrder = (stats: ReadonlyArray<UnitStat>) => scope.universe.filter(stat => stats.some(other => other.equals(stat)));

describe('stat filters', () => {
	it('shows the defaults in universe order while the slot is unset', () => {
		const filters = DatabaseFilters.create();

		expect(isCustomised(filters, 'ep')).toBe(false);
		expect(keys(shownStats(filters, 'ep', scope))).toEqual(keys(inUniverseOrder(scope.defaults)));
	});

	it('shows the stored stats inside the universe, in universe order, once the slot is set', () => {
		const stored = [stamina, intellect, strength];
		const filters = withSelection(DatabaseFilters.create(), 'buffs', stored);

		const shown = shownStats(filters, 'buffs', scope);

		expect(keys(shown)).toEqual(keys(inUniverseOrder(stored)));
		expect(shown.some(stat => stat.equals(intellect))).toBe(false);
	});

	it('shows nothing for a slot set to an empty list and counts it as customised', () => {
		const filters = withSelection(DatabaseFilters.create(), 'debuffs', []);

		expect(isCustomised(filters, 'debuffs')).toBe(true);
		expect(shownStats(filters, 'debuffs', scope)).toEqual([]);
	});

	it('returns to the defaults when the slot is cleared', () => {
		const customised = withSelection(DatabaseFilters.create(), 'characterStats', [agility]);

		const cleared = withSelection(customised, 'characterStats', undefined);

		expect(isCustomised(cleared, 'characterStats')).toBe(false);
		expect(keys(shownStats(cleared, 'characterStats', scope))).toEqual(keys(inUniverseOrder(scope.defaults)));
	});

	it('leaves filters with no stat filters alone when a slot is cleared', () => {
		const filters = DatabaseFilters.create({ minIlvl: 50 });
		const before = DatabaseFilters.clone(filters);

		const cleared = withSelection(filters, 'ep', undefined);

		expect(cleared).toBe(filters);
		expect(cleared.statFilters).toBeUndefined();
		expect(DatabaseFilters.equals(cleared, before)).toBe(true);
	});

	it('changes only the given slot, on the filters it is given, and returns them', () => {
		const filters = withSelection(DatabaseFilters.create({ minIlvl: 50, showAllEnchants: true, favoriteItems: [7] }), 'ep', [agility]);
		const before = DatabaseFilters.clone(filters);

		const next = withSelection(filters, 'consumables', [stamina]);

		expect(next).toBe(filters);
		expect(keys(shownStats(next, 'consumables', scope))).toEqual(keys([stamina]));
		expect(keys(shownStats(next, 'ep', scope))).toEqual(keys(shownStats(before, 'ep', scope)));
		expect(isCustomised(next, 'buffs')).toBe(false);
		expect(
			DatabaseFilters.equals(DatabaseFilters.create({ ...next, statFilters: undefined }), DatabaseFilters.create({ ...before, statFilters: undefined })),
		).toBe(true);
	});

	it('keeps a set slot and an empty slot apart through binary serialisation', () => {
		const filters = withSelection(withSelection(DatabaseFilters.create(), 'ep', [meleeHit, agility]), 'buffs', []);

		const roundTripped = DatabaseFilters.fromBinary(DatabaseFilters.toBinary(filters));

		expect(isCustomised(roundTripped, 'ep')).toBe(true);
		expect(keys(shownStats(roundTripped, 'ep', scope))).toEqual(keys(shownStats(filters, 'ep', scope)));
		expect(isCustomised(roundTripped, 'buffs')).toBe(true);
		expect(shownStats(roundTripped, 'buffs', scope)).toEqual([]);
		expect(isCustomised(roundTripped, 'debuffs')).toBe(false);
	});

	it('keeps a set slot and an empty slot apart through JSON serialisation', () => {
		const filters = withSelection(withSelection(DatabaseFilters.create(), 'ep', [meleeHit, agility]), 'buffs', []);

		const roundTripped = DatabaseFilters.fromJsonString(DatabaseFilters.toJsonString(filters));

		expect(isCustomised(roundTripped, 'ep')).toBe(true);
		expect(keys(shownStats(roundTripped, 'ep', scope))).toEqual(keys(shownStats(filters, 'ep', scope)));
		expect(isCustomised(roundTripped, 'buffs')).toBe(true);
		expect(shownStats(roundTripped, 'buffs', scope)).toEqual([]);
		expect(isCustomised(roundTripped, 'debuffs')).toBe(false);
	});

	it('reads the same array while the slot shows the same stats, across new filters objects', () => {
		const read = shownStatsReader('buffs', scope);
		const shown = read(withSelection(DatabaseFilters.create(), 'buffs', [stamina]));

		expect(read(withSelection(withSelection(DatabaseFilters.create(), 'buffs', [stamina]), 'ep', [agility]))).toBe(shown);
		const changed = read(withSelection(DatabaseFilters.create(), 'buffs', [stamina, agility]));
		expect(changed).not.toBe(shown);
		expect(keys(changed)).toEqual(keys([agility, stamina]));
	});
});
