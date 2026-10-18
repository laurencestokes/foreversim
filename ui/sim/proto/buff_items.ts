import { PartyBuffs } from '@generated/proto/buffs';
import { SimDatabase } from '@generated/proto/db';

import type { Database } from './database';
import { Gear } from './gear';

// The items whose data the sim reads for a party buff. Without the full item database the sim
// knows only the items the request carries, so the request carries these while the buff is on.
export const PARTY_BUFF_ITEM_IDS: Partial<Record<keyof PartyBuffs, readonly number[]>> = {
	darkspear: [12802],
};

export const addPartyBuffItems = (database: SimDatabase, db: Database, partyBuffs: PartyBuffs | undefined) => {
	const itemIds = Object.entries(PARTY_BUFF_ITEM_IDS).flatMap(([field, ids]) => (partyBuffs?.[field as keyof PartyBuffs] ? ids : []));
	for (const id of itemIds) {
		const item = db.getItemById(id);
		if (item && !database.items.some(carried => carried.id === id)) database.items.push(Gear.itemToDB(item));
	}
};
