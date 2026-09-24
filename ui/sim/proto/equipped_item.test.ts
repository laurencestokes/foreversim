import { GemColor, ItemType, WeaponType } from '@generated/proto/common';
import { UIEnchant as Enchant, UIGem as Gem, UIItem as Item } from '@generated/proto/ui';
import { describe, expect, it, vi } from 'vitest';

vi.mock('@i18n/localization', () => ({
	translateStat: (stat: unknown) => String(stat),
	translatePseudoStat: (pseudoStat: unknown) => String(pseudoStat),
}));

import { EquippedItem } from './equipped_item';

const socketedItem = () =>
	Item.create({
		id: 30107,
		name: 'Vestments of the Sea-Witch',
		type: ItemType.ItemTypeChest,
		gemSockets: [GemColor.GemColorRed],
		scalingOptions: { 0: { ilvl: 141, randPropPoints: 0, weaponDamageMin: 0, weaponDamageMax: 0, stats: [] } },
	});

const swordItem = () =>
	Item.create({
		id: 19324,
		name: 'The Lobotomizer',
		type: ItemType.ItemTypeWeapon,
		weaponType: WeaponType.WeaponTypeSword,
		scalingOptions: { 0: { ilvl: 76, randPropPoints: 0, weaponDamageMin: 59, weaponDamageMax: 111, stats: [] } },
	});

const shieldItem = () =>
	Item.create({
		id: 17066,
		name: 'Drillborer Disk',
		type: ItemType.ItemTypeWeapon,
		weaponType: WeaponType.WeaponTypeShield,
		scalingOptions: { 0: { ilvl: 76, randPropPoints: 0, weaponDamageMin: 0, weaponDamageMax: 0, stats: [] } },
	});

const gem = (id: number) => Gem.create({ id, color: GemColor.GemColorRed });
const enchant = (effectId: number) => Enchant.create({ effectId, itemId: effectId });

// TBC's signature is equals(other, ignoreEnchants?, ignoreGems?) where MoP's is
// equals(other, ignoreReforge?, ignoreEnchants?, ignoreGems?, ignoreUpgrades?). A call written
// against one silently means something else against the other, so pin the positions.
describe('EquippedItem.equals', () => {
	const base = new EquippedItem({ item: socketedItem(), enchant: enchant(2661), gems: [gem(32196)] });

	it('takes enchants and gems into account by default', () => {
		expect(base.equals(new EquippedItem({ item: socketedItem(), enchant: enchant(2661), gems: [gem(32196)] }))).toBe(true);
		expect(base.equals(new EquippedItem({ item: socketedItem(), enchant: enchant(2669), gems: [gem(32196)] }))).toBe(false);
		expect(base.equals(new EquippedItem({ item: socketedItem(), enchant: enchant(2661), gems: [gem(32215)] }))).toBe(false);
	});

	it('reads the second parameter as ignoreEnchants', () => {
		const otherEnchant = new EquippedItem({ item: socketedItem(), enchant: enchant(2669), gems: [gem(32196)] });
		expect(base.equals(otherEnchant, true)).toBe(true);
		expect(base.equals(otherEnchant, false)).toBe(false);
	});

	it('reads the third parameter as ignoreGems, and the second does not cover gems', () => {
		const otherGem = new EquippedItem({ item: socketedItem(), enchant: enchant(2661), gems: [gem(32215)] });
		expect(base.equals(otherGem, true)).toBe(false);
		expect(base.equals(otherGem, false, true)).toBe(true);
	});

	it('ignores both when both flags are set', () => {
		const other = new EquippedItem({ item: socketedItem(), enchant: enchant(2669), gems: [gem(32215)] });
		expect(base.equals(other, true, true)).toBe(true);
	});
});

describe('EquippedItem weapon type override', () => {
	it('defaults effectiveWeaponType to the item’s own type', () => {
		const sword = new EquippedItem({ item: swordItem() });
		expect(sword.weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
		expect(sword.effectiveWeaponType).toBe(WeaponType.WeaponTypeSword);
	});

	it('withWeaponTypeOverride relabels effectiveWeaponType without touching the item itself', () => {
		const sword = new EquippedItem({ item: swordItem() });
		const asAxe = sword.withWeaponTypeOverride(WeaponType.WeaponTypeAxe);

		expect(asAxe.weaponTypeOverride).toBe(WeaponType.WeaponTypeAxe);
		expect(asAxe.effectiveWeaponType).toBe(WeaponType.WeaponTypeAxe);
		// The underlying item is untouched -- it's still, physically, a sword.
		expect(asAxe.item.weaponType).toBe(WeaponType.WeaponTypeSword);
		expect(asAxe.item.id).toBe(sword.item.id);
	});

	it('clears the override when relabelled to WeaponTypeUnknown', () => {
		const overridden = new EquippedItem({ item: swordItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeAxe);
		const cleared = overridden.withWeaponTypeOverride(WeaponType.WeaponTypeUnknown);
		expect(cleared.weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
		expect(cleared.effectiveWeaponType).toBe(WeaponType.WeaponTypeSword);
	});

	it('is a no-op for non-weapons', () => {
		const chest = new EquippedItem({ item: socketedItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeAxe);
		expect(chest.weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
	});

	it('is a no-op when the item itself is a shield', () => {
		const shield = new EquippedItem({ item: shieldItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeSword);
		expect(shield.weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
		expect(shield.effectiveWeaponType).toBe(WeaponType.WeaponTypeShield);
	});

	it('is a no-op when relabelling into OffHand or Shield', () => {
		const sword = new EquippedItem({ item: swordItem() });
		expect(sword.withWeaponTypeOverride(WeaponType.WeaponTypeShield).weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
		expect(sword.withWeaponTypeOverride(WeaponType.WeaponTypeOffHand).weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
	});

	it('is dropped by withItem, since a new item is not a relabel of the old one', () => {
		const overridden = new EquippedItem({ item: swordItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeAxe);
		const replaced = overridden.withItem(swordItem());
		expect(replaced.weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
	});

	it('survives withEnchant, withGem and withRandomSuffix', () => {
		const overridden = new EquippedItem({ item: swordItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeAxe);
		expect(overridden.withEnchant(enchant(2661)).weaponTypeOverride).toBe(WeaponType.WeaponTypeAxe);
		expect(overridden.withRandomSuffix(null).weaponTypeOverride).toBe(WeaponType.WeaponTypeAxe);
	});

	it('feeds into equals, so a different override is a different EquippedItem', () => {
		const asAxe = new EquippedItem({ item: swordItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeAxe);
		const asMace = new EquippedItem({ item: swordItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeMace);
		const plain = new EquippedItem({ item: swordItem() });

		expect(asAxe.equals(asMace)).toBe(false);
		expect(asAxe.equals(plain)).toBe(false);
		expect(asAxe.equals(new EquippedItem({ item: swordItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeAxe))).toBe(true);
	});

	// The override has to round-trip through the ItemSpec proto (save/share links), same as
	// enchant/gems/randomSuffix.
	it('round-trips through asSpec()', () => {
		const overridden = new EquippedItem({ item: swordItem() }).withWeaponTypeOverride(WeaponType.WeaponTypeAxe);
		const spec = overridden.asSpec();
		expect(spec.weaponTypeOverride).toBe(WeaponType.WeaponTypeAxe);

		const rebuilt = new EquippedItem({ item: swordItem(), weaponTypeOverride: spec.weaponTypeOverride });
		expect(rebuilt.effectiveWeaponType).toBe(WeaponType.WeaponTypeAxe);
		expect(rebuilt.equals(overridden)).toBe(true);
	});

	it('omits the override from asSpec() when unset', () => {
		const plain = new EquippedItem({ item: swordItem() });
		expect(plain.asSpec().weaponTypeOverride).toBe(WeaponType.WeaponTypeUnknown);
	});
});
