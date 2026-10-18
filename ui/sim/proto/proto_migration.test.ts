import { Player } from '@generated/proto/api';
import { ConsumesSpec } from '@generated/proto/common';
import { Consumable, SimDatabase, SimEnchant } from '@generated/proto/db';
import { describe, expect, it } from 'vitest';

import type { Database } from './database';
import { extendPlayerProtoWithMissingEffects } from './proto_migration';

const catalogue = (consumables: Consumable[]) => {
	const byId = new Map(consumables.map(consumable => [consumable.id, consumable]));
	return { getConsumable: (id: number) => byId.get(id), getSpellEffect: () => undefined } as unknown as Database;
};

const sentIds = (consumes: Partial<ConsumesSpec>, db: Database) => {
	const player = Player.create({ consumables: ConsumesSpec.create(consumes), database: SimDatabase.create() });
	extendPlayerProtoWithMissingEffects(player, db);
	return player.database!.consumables.map(consumable => consumable.id).sort((a, b) => a - b);
};

describe('extendPlayerProtoWithMissingEffects', () => {
	const db = catalogue([5206, 9088, 10305, 10306, 10308, 10309, 10310, 10646, 13452, 20749, 13444].map(id => Consumable.create({ id })));

	it('carries every item the extra ids and the elixir ids name', () => {
		expect(sentIds({ extraIds: [10309, 10310, 10308, 10306, 10305, 5206, 10646] }, db)).toEqual([5206, 10305, 10306, 10308, 10309, 10310, 10646]);
		expect(sentIds({ elixirIds: [13452, 9088] }, db)).toEqual([9088, 13452]);
	});

	it('carries an item two fields name once', () => {
		expect(sentIds({ resourcePotId: 13444, potions: [13444], extraIds: [10309] }, db)).toEqual([10309, 13444]);
	});

	it('carries the temporary enchant an imbue puts on the weapon', () => {
		const temporaryEnchant = SimEnchant.create({ effectId: 2628 });
		const withEnchant = catalogue([Consumable.create({ id: 20749, temporaryEnchant })]);
		const player = Player.create({ consumables: ConsumesSpec.create({ mhImbueId: 20749 }), database: SimDatabase.create() });
		extendPlayerProtoWithMissingEffects(player, withEnchant);
		expect(player.database!.consumables[0].temporaryEnchant).toEqual(temporaryEnchant);
	});
});
