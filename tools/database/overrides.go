package database

import (
	"regexp"
	"time"

	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
	"github.com/wowsims/forever/tools/database/dbc"
)

// Allows you to ignore certain Spell Effects that the Sim does not support.
// This prevents them from being added to the missing effects list to prevent confusion towards users.
// Empty array means ignore all effects of that type, otherwise it will be ignored
// based on EffectMiscValue_0
var IgnoreSpellEffectByAuraType = map[dbc.EffectAuraType][]int{
	dbcenums.A_MOD_MECHANIC_RESISTANCE: {},
	dbcenums.A_MOD_STEALTH:             {},
	dbcenums.A_MOD_STEALTH_DETECT:      {},
	dbcenums.A_MOD_STEALTH_LEVEL:       {},
	dbcenums.A_MOD_DECREASE_SPEED:      {},
	dbcenums.A_MOD_INVISIBILITY:        {},
	dbcenums.A_MOD_INVISIBILITY_DETECT: {},
	dbcenums.A_MOD_SKILL: {
		356, // Fishing Skill
		393, // Skinning Skill
	},
	dbcenums.A_MOD_INCREASE_MOUNTED_SPEED:        {},
	dbcenums.A_MOD_MOUNTED_SPEED_ALWAYS:          {},
	dbcenums.A_MOD_MOUNTED_SPEED_NOT_STACK:       {},
	dbcenums.A_MOD_INCREASE_MOUNTED_FLIGHT_SPEED: {},
	dbcenums.A_MOD_MOUNTED_FLIGHT_SPEED_ALWAYS:   {},
	dbcenums.A_TRANSFORM:                         {},
	dbcenums.A_MECHANIC_IMMUNITY:                 {},
	dbcenums.A_TRACK_CREATURES:                   {},
	dbcenums.A_TRACK_RESOURCES:                   {},
	dbcenums.A_FAR_SIGHT:                         {},
}

var IgnoreSpellEffectBySpellEffectType = map[dbc.SpellEffectType][]int{
	dbcenums.E_CREATE_ITEM:    {},
	dbcenums.E_SUMMON:         {},
	dbcenums.E_TELEPORT_UNITS: {},
}

// Spells that are flavour rather than mechanics, keyed by spell ID so that reporting them as
// missing effects can be suppressed without hiding anything real.
//
// Keyed per spell on purpose. The obvious shortcut - treat an effect made only of A_DUMMY auras
// as noise, which is what MoP does - does not hold in TBC, where A_DUMMY is how a whole class of
// real item effects is encoded: every relic, idol, libram and totem bonus, plus Zandalarian Hero
// Medallion, Thick Obsidian Breastplate and Ashtongue Talisman of Zeal. That rule removed 48
// entries here and only a handful of them were noise.
var IgnoreMissingEffectBySpellID = map[int]string{
	16372: "Seal of Ascension - no tooltip and no mechanic",
}

var OtherItemIdsToFetch = []string{}
var ConsumableOverrides = []*proto.Consumable{
	{Id: 23334, CooldownDuration: int32(time.Hour.Seconds())}, // Cracked Power Core
	{Id: 23381, CooldownDuration: int32(time.Hour.Seconds())}, // Chipped Power Core
}

// Empty: all 26 entries were TBC items absent from this client.
var ItemOverrides = []*proto.UIItem{}

// Keep these sorted by item ID.
var ItemAllowList = map[int32]struct{}{
	1168:  {}, // Skullflame Shield
	2140:  {},
	2505:  {},
	8345:  {}, // Wolfshead Helm
	11815: {}, // Hand of Justice
	18168: {}, // Force Reactive Disk
	19337: {}, // The Black Book
}

// Keep these sorted by item ID.
var ItemDenyList = map[int32]struct{}{
	17782: {}, // talisman of the binding shard
	17783: {}, // talisman of the binding fragment
	17802: {}, // Deprecated version of Thunderfury
	18582: {},
	18583: {},
	18584: {},
	22736: {},
	23363: {}, // Titanic Breastplate
	24265: {},
}

// Item icons to include in the DB, so they don't need to be separately loaded in the UI.
var ExtraItemIcons = []int32{
	// Demonic Rune
	12662,

	// Food IDs
	27655,
	27657,
	27658,
	27664,

	// Flask IDs
	13512,
	22854,
	22866,

	// Elixer IDs
	9224,
	13452,
	13454,
	22827,
	22833,
	22835,
	22840,

	// Potions / In Battle Consumes
	13442,
	22105,
	22788,
	22828,
	22837,
	22838,
	22849,

	// Thistle Tea
	7676,

	// Scrolls
	27498,
	27499,
	27503,
}

// Item Ids of consumables to allow
var ConsumableAllowList = []int32{
	7676,  // Thisle Tea
	9088,  // Gift of Arthas
	9155,  // Arcane Elixir
	9224,  // Elixir of Demonslaying
	13442, // Migty Rage Potion
	13452, // Elixir of the Mongoose
	13454, // Greater Arcane Elixir
	12662, // Demonic Rune
	22105, // Master Healthstone
	22788, // Flamecap
	22797, // Nightmare Seed
	23334, // Cracked Power Core
	23381, // Chipped Power Core
	5206,  // Bogling Root
}

// Classic consumables and the picker each goes in. The client has no battle/guardian split for
// them, and most sit below the loader's level-50 floor, so without this list the sim silently
// treats them as empty. The set is master's consumables picker plus its world buffs.
var ClassicConsumableTypes = map[int32]proto.ConsumableType{
	// Potions
	1710: proto.ConsumableType_ConsumableTypePotion, 3928: proto.ConsumableType_ConsumableTypePotion,
	13446: proto.ConsumableType_ConsumableTypePotion, 3827: proto.ConsumableType_ConsumableTypePotion,
	6149: proto.ConsumableType_ConsumableTypePotion, 13443: proto.ConsumableType_ConsumableTypePotion,
	13444: proto.ConsumableType_ConsumableTypePotion, 13442: proto.ConsumableType_ConsumableTypePotion,
	5633: proto.ConsumableType_ConsumableTypePotion, 5631: proto.ConsumableType_ConsumableTypePotion,
	9036: proto.ConsumableType_ConsumableTypePotion, 13461: proto.ConsumableType_ConsumableTypePotion,
	13457: proto.ConsumableType_ConsumableTypePotion, 13456: proto.ConsumableType_ConsumableTypePotion,
	13460: proto.ConsumableType_ConsumableTypePotion, 13458: proto.ConsumableType_ConsumableTypePotion,
	13459: proto.ConsumableType_ConsumableTypePotion, 13455: proto.ConsumableType_ConsumableTypePotion,
	4623: proto.ConsumableType_ConsumableTypePotion,
	// Flasks
	13510: proto.ConsumableType_ConsumableTypeFlask, 13511: proto.ConsumableType_ConsumableTypeFlask,
	13512: proto.ConsumableType_ConsumableTypeFlask, 13513: proto.ConsumableType_ConsumableTypeFlask,
	// Food
	21023: proto.ConsumableType_ConsumableTypeFood, 13928: proto.ConsumableType_ConsumableTypeFood,
	20452: proto.ConsumableType_ConsumableTypeFood, 18254: proto.ConsumableType_ConsumableTypeFood,
	13810: proto.ConsumableType_ConsumableTypeFood, 13813: proto.ConsumableType_ConsumableTypeFood,
	13931: proto.ConsumableType_ConsumableTypeFood, 18045: proto.ConsumableType_ConsumableTypeFood,
	21217: proto.ConsumableType_ConsumableTypeFood, 13851: proto.ConsumableType_ConsumableTypeFood,
	21072: proto.ConsumableType_ConsumableTypeFood,
	// Battle elixir slot: agility elixirs
	13452: proto.ConsumableType_ConsumableTypeBattleElixir, 9187: proto.ConsumableType_ConsumableTypeBattleElixir,
	8949: proto.ConsumableType_ConsumableTypeBattleElixir, 3390: proto.ConsumableType_ConsumableTypeBattleElixir,
	// Spell power elixirs
	13454: proto.ConsumableType_ConsumableTypeSpellPowerElixir, 9155: proto.ConsumableType_ConsumableTypeSpellPowerElixir,
	// Guardian elixir slot: health and mana regen (Forever adds the Greater Mageblood Elixir)
	3825: proto.ConsumableType_ConsumableTypeGuardianElixir, 2458: proto.ConsumableType_ConsumableTypeGuardianElixir,
	20007: proto.ConsumableType_ConsumableTypeGuardianElixir, 250341: proto.ConsumableType_ConsumableTypeGuardianElixir,
	// Armor elixirs
	13445: proto.ConsumableType_ConsumableTypeDefenseElixir, 8951: proto.ConsumableType_ConsumableTypeDefenseElixir,
	3389: proto.ConsumableType_ConsumableTypeDefenseElixir, 5997: proto.ConsumableType_ConsumableTypeDefenseElixir,
	// School power elixirs
	21546: proto.ConsumableType_ConsumableTypeSchoolElixir, 6373: proto.ConsumableType_ConsumableTypeSchoolElixir,
	17708: proto.ConsumableType_ConsumableTypeSchoolElixir, 9264: proto.ConsumableType_ConsumableTypeSchoolElixir,
	250343: proto.ConsumableType_ConsumableTypeSchoolElixir, // Forever's Elixir of Nature Power
	// Strength: Juju Power, Elixir of Giants, Elixir of Ogre's Strength
	12451: proto.ConsumableType_ConsumableTypeStrengthBuff, 9206: proto.ConsumableType_ConsumableTypeStrengthBuff,
	3391: proto.ConsumableType_ConsumableTypeStrengthBuff,
	// Attack power: Juju Might, Winterfall Firewater
	12460: proto.ConsumableType_ConsumableTypeAttackPowerBuff, 12820: proto.ConsumableType_ConsumableTypeAttackPowerBuff,
	// Blasted Lands and Zanza buffs
	8410: proto.ConsumableType_ConsumableTypeZanza, 8412: proto.ConsumableType_ConsumableTypeZanza,
	8423: proto.ConsumableType_ConsumableTypeZanza, 8424: proto.ConsumableType_ConsumableTypeZanza,
	8411: proto.ConsumableType_ConsumableTypeZanza, 20079: proto.ConsumableType_ConsumableTypeZanza,
	// Alcohol: Rumsey Rum Black Label/Dark/Light, Gordok Green Grog, Kreeg's Stout Beatdown
	21151: proto.ConsumableType_ConsumableTypeAlcohol, 21114: proto.ConsumableType_ConsumableTypeAlcohol,
	20709: proto.ConsumableType_ConsumableTypeAlcohol, 18269: proto.ConsumableType_ConsumableTypeAlcohol,
	18284: proto.ConsumableType_ConsumableTypeAlcohol,
}
var ConsumableDenyList = []int32{}

// Raid buffs / debuffs
var SharedSpellsIcons = []int32{
	// Registered CD's
	10060,
	16190,
	29166,
	2825,

	17051,

	25898,

	20140,
	8071,

	14767,

	8075,

	20045,

	30808,
	19506,

	12861,
	18696,

	20245,
	5675,
	16206,

	17007,

	8512,
	29193,

	24907,

	3738,
	8227,

	6562,
	16840,

	// Raid Debuffs
	8647,

	770,
	702,
	18180,

	12879,
	16862,

	20337,

	3043,

	1490,

	20271,

	11374,
	15235,

	// Raid buffs, debuffs and shared consumables.
	603,   // Curse of Doom
	688,   // Summon Imp
	691,   // Summon Felhunter
	697,   // Summon Voidwalker
	704,   // Curse of Recklessness
	706,   // Demon Armor
	712,   // Summon Succubus
	6117,  // Mage Armor
	7302,  // Ice Armor
	8024,  // Flametongue Weapon
	8033,  // Frostbrand Weapon
	8232,  // Windfury Weapon
	10538, // Fire Resistance Totem
	12470, // Fire Nova
	13339, // Fire Blast
	13376, // Fire Shield
	13889, // Minor Speed
	14325, // Hunter's Mark
	17768, // Wolfshead Helm
	18803, // Focus
	19615, // Frenzy Effect
	20574, // Axe Specialization
	20575, // Command
	20576, // Command
	20594, // Stoneform
	20595, // Gun Specialization
	20597, // Sword Specialization
	20864, // Mace Specialization
	23110, // Dash
	23563, // Enhanced Battle Shout
	25076, // Cobra Reflexes
	25894, // Greater Blessing of Wisdom
	25895, // Greater Blessing of Salvation
	26654, // Sweeping Strikes
	28142, // Power of the Guardian
	28143, // Power of the Guardian
	29414, // Haste
}

// If any of these match the item name, don't include it.
var DenyListNameRegexes = []*regexp.Regexp{
	regexp.MustCompile(`30 Epic`),
	regexp.MustCompile(`130 Epic`),
	regexp.MustCompile(`63 Blue`),
	regexp.MustCompile(`63 Green`),
	regexp.MustCompile(`66 Epic`),
	regexp.MustCompile(`90 Epic`),
	regexp.MustCompile(`90 Green`),
	regexp.MustCompile(`Boots 1`),
	regexp.MustCompile(`Boots 2`),
	regexp.MustCompile(`Boots 3`),
	regexp.MustCompile(`Bracer 1`),
	regexp.MustCompile(`Bracer 2`),
	regexp.MustCompile(`Bracer 3`),
	regexp.MustCompile(`DB\d`),
	regexp.MustCompile(`DEPRECATED`),
	regexp.MustCompile(`OLD`),
	regexp.MustCompile(`Deprecated`),
	regexp.MustCompile(`Deprecated: Keanna`),
	regexp.MustCompile(`Indalamar`),
	regexp.MustCompile(`Monster -`),
	regexp.MustCompile(`NEW`),
	regexp.MustCompile(`PH`),
	regexp.MustCompile(`QR XXXX`),
	regexp.MustCompile(`TEST`),
	regexp.MustCompile(`Test`),
	regexp.MustCompile(`Enchant Template`),
	regexp.MustCompile(`Arcane Amalgamation`),
	regexp.MustCompile(`Deleted`),
	regexp.MustCompile(`DELETED`),
	regexp.MustCompile(`zOLD`),
	regexp.MustCompile(`Archaic Spell`),
	regexp.MustCompile(`Well Repaired`),
	regexp.MustCompile(`Boss X`),
	regexp.MustCompile(`Adventurine`),
	regexp.MustCompile(`Sardonyx`),
	regexp.MustCompile(`Zyanite`),
	regexp.MustCompile(`zzold`),
	regexp.MustCompile(`Tom's`),
	regexp.MustCompile(`Stabilized Eternium Scope`),
}

// Allows manual overriding for Gem fields in case WowHead is wrong.
// Empty: this build ships no gem system -- GemProperties has zero rows.
var GemOverrides = []*proto.UIGem{}
var GemAllowList = map[int32]struct{}{}
var EnchantDenyListSpells = map[int32]struct{}{}
var EnchantDenyListItems = map[int32]struct{}{}
var GemDenyList = map[int32]struct{}{}

var EnchantDenyList = map[int32]struct{}{
	3269: {}, // Truesilver Fishing Line
	3289: {}, // Skybreaker Whip/Riding Crop
	3315: {}, // Carrot on a Stick
	4671: {}, // Kyle's Test Enchantment
	4687: {}, // Enchant Weapon - Ninja (TEST VERSION)
	4717: {}, // Enchant Weapon - Pandamonium (DNT)
	5029: {}, // Custom - Jaina - Crackling Lightning
	5110: {}, // Lightweave Embroidery - Junk
}

var EnchantAllowList = []int32{
	368,  // Enchant Cloak - Greater Agility
	804,  // Enchant Cloak - Lesser Shadow Resistance
	369,  // Enchant Bracer - Major Intellect
	684,  // Enchant Gloves - Major Strength
	963,  // Enchant Weapon - Major Striking
	1593, // Bracer 24 AP
	1594, // Gloves 26 AP
	1900, // Enchant Weapon - Crusader
	2564, // Weapon 15 Agi
	2583, // Presence of Might
	2588, // Presence of Sight
	2647, // Enchant Bracer - Brawn
	2659, // Enchant Chest - Exceptional Health
}

// Note: EffectId is required for all enchants, because they are
// used by various importers/exporters
// Note: ItemId, SpellId and Name are part of the enchant DB key, so they must
// match the generated enchant or the override is added as a separate entry.
var EnchantOverrides = []*proto.UIEnchant{
	// The head/leg resistance kits grant their stats through an equip spell that no
	// longer exists in the client data, so they have to be filled in by hand.
	{EffectId: 2681, ItemId: 22635, SpellId: 28162, Name: "Savage Guard", Stats: stats.Stats{stats.NatureResistance: 10}.ToProtoArray()},
	{EffectId: 2682, ItemId: 22636, SpellId: 28164, Name: "Ice Guard", Stats: stats.Stats{stats.FrostResistance: 10}.ToProtoArray()},
	{EffectId: 2683, ItemId: 22638, SpellId: 28166, Name: "Shadow Guard", Stats: stats.Stats{stats.ShadowResistance: 10}.ToProtoArray()},
}
