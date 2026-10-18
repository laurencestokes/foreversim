import { Player } from '@generated/proto/api';
import { Consumable } from '@generated/proto/db';
import { SpellEffect } from '@generated/proto/spell';

import { Database } from './database';

export const extendPlayerProtoWithMissingEffects = (playerProto: Player, db: Database) => {
	const newConsumables: Consumable[] = [];
	const newSpellEffects: SpellEffect[] = [];
	const seenConsumableIds = new Set<number>();
	const seenEffectIds = new Set<number>();

	const { potions = [], conjuredItems = [], extraIds = [], elixirIds = [], ...consumables } = playerProto.consumables || {};
	const allConsumableIds = Object.values(consumables).filter((c): c is number => typeof c === 'number');
	const allConsumables = [...potions, ...conjuredItems, ...extraIds, ...elixirIds, ...allConsumableIds];

	allConsumables.forEach((cid: number) => {
		if (!cid || seenConsumableIds.has(cid)) return;
		const consume = db.getConsumable(cid);
		if (!consume) return;
		seenConsumableIds.add(consume.id);
		newConsumables.push(consume);
		for (const eid of consume.effectIds) {
			if (seenEffectIds.has(eid)) continue;
			const effect = db.getSpellEffect(eid);
			if (!effect) continue;

			seenEffectIds.add(effect.id);
			newSpellEffects.push(effect);
		}
	});

	if (playerProto.database) {
		// swap in the fresh arrays
		playerProto.database.consumables = newConsumables;
		playerProto.database.spellEffects = newSpellEffects;
	}
};
