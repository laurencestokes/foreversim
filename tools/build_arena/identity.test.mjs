import assert from 'node:assert/strict';
import fs from 'node:fs';
import { test } from 'node:test';

import { resolveBuildId, validateCatalog } from './identity.mjs';

const catalog = JSON.parse(fs.readFileSync(new URL('./catalog.json', import.meta.url), 'utf8'));

test('replacement presets have their own identities', () => {
	validateCatalog(catalog);
	for (const [spec, name, id] of [
		['feral_druid', 'Cat', 'feral-cat'],
		['mage', 'Arcane / Arcane', 'mage-arcane'],
		['mage', 'Frost / Frost Incin', 'mage-frost-incin'],
		['mage', 'Frost / Frost Med', 'mage-frost-med'],
		['retribution_paladin', 'Unsealed Wrath', 'paladin-unsealed-wrath'],
		['rogue', 'Combat Venom (Upstream)', 'rogue-combat-venom-upstream'],
		['elemental_shaman', 'Mana Tide', 'shaman-elemental-mana-tide'],
	]) {
		assert.equal(resolveBuildId(catalog, spec, name), id);
	}
	for (const [spec, name] of [
		['feral_druid', 'Bearweave (45s fights)'],
		['mage', 'Arcane / Arcane Frost'],
		['mage', 'Frost / Frost'],
		['retribution_paladin', 'Twist of Light'],
		['rogue', 'Subtlety BS (Upstream)'],
	]) {
		assert.throws(() => resolveBuildId(catalog, spec, name), /Expected one catalog entry/);
	}
});
