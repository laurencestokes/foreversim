import { PartyBuffs } from '@generated/proto/buffs';
import { SimDatabase, SimItem } from '@generated/proto/db';
import { UIItem } from '@generated/proto/ui';
import { describe, expect, it } from 'vitest';

import { addPartyBuffItems, PARTY_BUFF_ITEM_IDS } from './buff_items';
import type { Database } from './database';

const darkspearItemIds = PARTY_BUFF_ITEM_IDS.darkspear!;

const catalogue = (ids: readonly number[]) => {
	const byId = new Map(ids.map(id => [id, UIItem.create({ id, name: `Item ${id}` })]));
	return { getItemById: (id: number) => byId.get(id) } as unknown as Database;
};

const sentItemIds = (buffs: PartyBuffs | undefined, carried: readonly number[] = []) => {
	const database = SimDatabase.create({ items: carried.map(id => SimItem.create({ id })) });
	addPartyBuffItems(database, catalogue(darkspearItemIds), buffs);
	return database.items.map(item => item.id).sort((a, b) => a - b);
};

const sorted = (ids: readonly number[]) => [...ids].sort((a, b) => a - b);

describe('addPartyBuffItems', () => {
	it('carries the items a party buff reads while the buff is on', () => {
		expect(sentItemIds(PartyBuffs.create({ darkspear: true }))).toEqual(sorted(darkspearItemIds));
	});

	it('carries nothing while the buff is off', () => {
		expect(sentItemIds(PartyBuffs.create({ darkspear: false }))).toEqual([]);
		expect(sentItemIds(undefined)).toEqual([]);
	});

	it('carries an item the request already holds once', () => {
		expect(sentItemIds(PartyBuffs.create({ darkspear: true }), darkspearItemIds)).toEqual(sorted(darkspearItemIds));
	});

	it('names the Darkspear item for the party Darkspear', () => {
		expect(darkspearItemIds).toEqual([12802]);
	});
});
