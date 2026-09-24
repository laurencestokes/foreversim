package forever

import (
	"time"

	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
)

func RegisterAllOnUseCds() {

	//
	// shared.NewSimpleStatActive(833) // Lifestone - https://www.wowhead.com/forever/spell=17712
	// shared.NewSimpleStatActive(1490) // Guardian Talisman - https://www.wowhead.com/forever/spell=1300364
	// shared.NewSimpleStatActive(2820) // Nifty Stopwatch - https://www.wowhead.com/forever/spell=14530
	// shared.NewSimpleStatActive(4130) // Smotts' Compass - https://www.wowhead.com/forever/spell=1317740
	// shared.NewSimpleStatActive(7734) // Six Demon Bag - https://www.wowhead.com/forever/spell=14537
	// shared.NewSimpleStatActive(8348) // Helm of Fire - https://www.wowhead.com/forever/spell=10578
	// shared.NewSimpleStatActive(8367) // Dragonscale Breastplate - https://www.wowhead.com/forever/spell=10618
	// shared.NewSimpleStatActive(11625) // Enthralled Sphere - https://www.wowhead.com/forever/spell=1300754
	// shared.NewSimpleStatActive(11702) // Grizzle's Skinner - https://www.wowhead.com/forever/spell=1300763
	// shared.NewSimpleStatActive(11808) // Circle of Flame - https://www.wowhead.com/forever/spell=17447
	// shared.NewSimpleStatActive(11905) // Linken's Boomerang - https://www.wowhead.com/forever/spell=15712
	// shared.NewSimpleStatActive(13143) // Mark of the Dragon Lord - https://www.wowhead.com/forever/spell=17252
	// shared.NewSimpleStatActive(13171) // Smokey's Lighter - https://www.wowhead.com/forever/spell=17283
	// shared.NewSimpleStatActive(13213) // Smolderweb's Eye - https://www.wowhead.com/forever/spell=1308917
	// shared.NewSimpleStatActive(13379) // Piccolo of the Flaming Fire - https://www.wowhead.com/forever/spell=18400
	// shared.NewSimpleStatActive(13515) // Ramstein's Lightning Bolts - https://www.wowhead.com/forever/spell=1299443
	// shared.NewSimpleStatActive(13544) // Spectral Essence - https://www.wowhead.com/forever/spell=1298497
	// shared.NewSimpleStatActive(13965) // Blackhand's Breadth - https://www.wowhead.com/forever/spell=1318944
	// shared.NewSimpleStatActive(13968) // Eye of the Beast - https://www.wowhead.com/forever/spell=1287840
	// shared.NewSimpleStatActive(14134) // Cloak of Fire - https://www.wowhead.com/forever/spell=18364
	// shared.NewSimpleStatActive(14152) // Robe of the Archmage - https://www.wowhead.com/forever/spell=18385
	// shared.NewSimpleStatActive(14153) // Robe of the Void - https://www.wowhead.com/forever/spell=18386
	// shared.NewSimpleStatActive(16768) // Furbolg Medicine Pouch - https://www.wowhead.com/forever/spell=20631
	// shared.NewSimpleStatActive(17744) // Heart of Noxxion - https://www.wowhead.com/forever/spell=21954
	// shared.NewSimpleStatActive(17759) // Mark of Resolution - https://www.wowhead.com/forever/spell=21956
	// shared.NewSimpleStatActive(18047) // Flame Walkers - https://www.wowhead.com/forever/spell=1301625
	// shared.NewSimpleStatActive(18340) // Eidolon Talisman - https://www.wowhead.com/forever/spell=1302362
	// shared.NewSimpleStatActive(18370) // Vigilance Charm - https://www.wowhead.com/forever/spell=1302264
	// shared.NewSimpleStatActive(18371) // Mindtap Talisman - https://www.wowhead.com/forever/spell=1302205
	// shared.NewSimpleStatActive(18406) // Onyxia Blood Talisman - https://www.wowhead.com/forever/spell=1287808
	// shared.NewSimpleStatActive(18537) // Counterattack Lodestone - https://www.wowhead.com/forever/spell=1302356
	// shared.NewSimpleStatActive(18634) // Gyrofreeze Ice Reflector - https://www.wowhead.com/forever/spell=23131
	// shared.NewSimpleStatActive(18637) // Major Recombobulator - https://www.wowhead.com/forever/spell=23064
	// shared.NewSimpleStatActive(18638) // Hyper-Radiant Flame Reflector - https://www.wowhead.com/forever/spell=23097
	// shared.NewSimpleStatActive(18639) // Ultra-Flash Shadow Reflector - https://www.wowhead.com/forever/spell=23132
	// shared.NewSimpleStatActive(18986) // Ultrasafe Transporter: Gadgetzan - https://www.wowhead.com/forever/spell=23453
	// shared.NewSimpleStatActive(19024) // Arena Grand Master - https://www.wowhead.com/forever/spell=23506
	// shared.NewSimpleStatActive(19141) // Luffa - https://www.wowhead.com/forever/spell=23595
	// shared.NewSimpleStatActive(19336) // Arcane Infused Gem - https://www.wowhead.com/forever/spell=23721
	// shared.NewSimpleStatActive(19339) // Mind Quickening Gem - https://www.wowhead.com/forever/spell=23723
	// shared.NewSimpleStatActive(19340) // Rune of Metamorphosis - https://www.wowhead.com/forever/spell=23724
	// shared.NewSimpleStatActive(19341) // Lifegiving Gem - https://www.wowhead.com/forever/spell=23725
	// shared.NewSimpleStatActive(19342) // Venomous Totem - https://www.wowhead.com/forever/spell=23726
	// shared.NewSimpleStatActive(19343) // Scrolls of Blinding Light - https://www.wowhead.com/forever/spell=23733
	// shared.NewSimpleStatActive(19344) // Natural Alignment Crystal - https://www.wowhead.com/forever/spell=23734
	// shared.NewSimpleStatActive(19948) // Zandalarian Hero Badge - https://www.wowhead.com/forever/spell=24574
	// shared.NewSimpleStatActive(19949) // Zandalarian Hero Medallion - https://www.wowhead.com/forever/spell=24661
	// shared.NewSimpleStatActive(19950) // Zandalarian Hero Charm - https://www.wowhead.com/forever/spell=24658
	// shared.NewSimpleStatActive(19955) // Wushoolay's Charm of Nature - https://www.wowhead.com/forever/spell=24542
	// shared.NewSimpleStatActive(19958) // Hazza'rah's Charm of Healing - https://www.wowhead.com/forever/spell=24546
	// shared.NewSimpleStatActive(19990) // Blessed Prayer Beads - https://www.wowhead.com/forever/spell=24354
	// shared.NewSimpleStatActive(19991) // Devilsaur Eye - https://www.wowhead.com/forever/spell=24352
	// shared.NewSimpleStatActive(19992) // Devilsaur Tooth - https://www.wowhead.com/forever/spell=24353
	// shared.NewSimpleStatActive(20036) // Fire Ruby - https://www.wowhead.com/forever/spell=24389
	// shared.NewSimpleStatActive(20071) // Talisman of Arathor - https://www.wowhead.com/forever/spell=23991
	// shared.NewSimpleStatActive(20072) // Defiler's Talisman - https://www.wowhead.com/forever/spell=23991
	// shared.NewSimpleStatActive(20084) // Hunting Net - https://www.wowhead.com/forever/spell=8312
	// shared.NewSimpleStatActive(20525) // Earthen Sigil - https://www.wowhead.com/forever/spell=24884
	// shared.NewSimpleStatActive(20534) // Abyss Shard - https://www.wowhead.com/forever/spell=25112
	// shared.NewSimpleStatActive(21115) // Defiler's Talisman - https://www.wowhead.com/forever/spell=25746
	// shared.NewSimpleStatActive(21117) // Talisman of Arathor - https://www.wowhead.com/forever/spell=25746
	// shared.NewSimpleStatActive(21181) // Grace of Earth - https://www.wowhead.com/forever/spell=25892
	// shared.NewSimpleStatActive(21488) // Fetish of Chitinous Spikes - https://www.wowhead.com/forever/spell=26168
	// shared.NewSimpleStatActive(21625) // Scarab Brooch - https://www.wowhead.com/forever/spell=26467
	// shared.NewSimpleStatActive(21647) // Fetish of the Sand Reaver - https://www.wowhead.com/forever/spell=26400
	// shared.NewSimpleStatActive(21685) // Petrified Scarab - https://www.wowhead.com/forever/spell=26463
	// shared.NewSimpleStatActive(21891) // Shard of the Fallen Star - https://www.wowhead.com/forever/spell=26789
	// shared.NewSimpleStatActive(22954) // Kiss of the Spider - https://www.wowhead.com/forever/spell=28866
	// shared.NewSimpleStatActive(23001) // Eye of Diminution - https://www.wowhead.com/forever/spell=28862
	// shared.NewSimpleStatActive(23027) // Warmth of Forgiveness - https://www.wowhead.com/forever/spell=28760
	// shared.NewSimpleStatActive(23040) // Glyph of Deflection - https://www.wowhead.com/forever/spell=28773
	// shared.NewSimpleStatActive(23558) // The Burrower's Shell - https://www.wowhead.com/forever/spell=29506
	// shared.NewSimpleStatActive(219345) // Infernal Lasso - https://www.wowhead.com/forever/spell=443265
	// shared.NewSimpleStatActive(221315) // Traveler's Symbols - https://www.wowhead.com/forever/spell=1306267
	// shared.NewSimpleStatActive(260819) // EZ-Thro Field Transporter: Gadgetzan - https://www.wowhead.com/forever/spell=23453
	// shared.NewSimpleStatActive(260821) // EZ and SAF Field Transporter: Mt. Hyjal - https://www.wowhead.com/forever/spell=1269339
	// shared.NewSimpleStatActive(260823) // Dimensional Transporter - Mt. Hyjal - https://www.wowhead.com/forever/spell=1269339
	// shared.NewSimpleStatActive(260824) // Gnomish Poultryizer - https://www.wowhead.com/forever/spell=1270941
	// shared.NewSimpleStatActive(269741) // Scented Runewood Brooch -  -  - https://www.wowhead.com/forever/spell=1296664
	// shared.NewSimpleStatActive(272437) // Adaptive Combat Assistant - https://www.wowhead.com/forever/spell=1291097
	// shared.NewSimpleStatActive(272438) // Weakness Analyzer - https://www.wowhead.com/forever/spell=1291101
	// shared.NewSimpleStatActive(272440) // Defender's Grip Stabilizer - https://www.wowhead.com/forever/spell=1291105
	// shared.NewSimpleStatActive(274386) // Toy Soldier - https://www.wowhead.com/forever/spell=1293820
	// shared.NewSimpleStatActive(274759) // Everlook Pathcarver - https://www.wowhead.com/forever/spell=1295270
	// shared.NewSimpleStatActive(274760) // Everlook Delivery Bot - https://www.wowhead.com/forever/spell=1295271
	// shared.NewSimpleStatActive(274761) // Parachute-Priest Pager - https://www.wowhead.com/forever/spell=1295272
	// shared.NewSimpleStatActive(275347) // Lichbane - https://www.wowhead.com/forever/spell=1296564
	// shared.NewSimpleStatActive(275729) // Rusty Propeller Blade - https://www.wowhead.com/forever/spell=1297762
	// shared.NewSimpleStatActive(276337) // Thaelemaches' Talisman - https://www.wowhead.com/forever/spell=1299440

	// Agility / Intellect / Spirit / Stamina / Strength
	shared.NewSimpleStatActive(270226) // Golden Banana - https://www.wowhead.com/forever/spell=1287571

	// Agility / Stamina / Strength
	shared.NewSimpleStatActive(15873) // Ragged John's Neverending Cup - https://www.wowhead.com/forever/spell=20587

	// ArcaneResistance / FireResistance / FrostResistance / NatureResistance / ShadowResistance
	shared.NewSimpleStatActive(15867) // Prismcharm - https://www.wowhead.com/forever/spell=19638
	shared.NewSimpleStatActive(23042) // Loatheb's Reflection - https://www.wowhead.com/forever/spell=28778

	// Armor
	shared.NewSimpleStatActive(11811) // Smoking Heart of the Mountain - https://www.wowhead.com/forever/spell=1300752
	shared.NewSimpleStatActive(12532) // Spire of the Stoneshaper - https://www.wowhead.com/forever/spell=16470
	shared.NewSimpleStatActive(19345) // Aegis of Preservation - https://www.wowhead.com/forever/spell=23780

	// ArmorPenetration
	// Badge of the Swarmguard - https://www.wowhead.com/forever/spell=26480
	shared.NewStackingStatBonusCD(shared.StackingStatBonusCD{
		Name:                  "Badge of the Swarmguard",
		ID:                    21670,
		CD:                    time.Millisecond * 180000,
		Callback:              core.CallbackOnSpellHitDealt,
		ProcMask:              core.ProcMaskMeleeMHAuto | core.ProcMaskMeleeOHAuto | core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeOHSpecial | core.ProcMaskRangedAuto | core.ProcMaskRangedSpecial,
		Outcome:               core.OutcomeLanded,
		RequireDamageDealt:    true,
		TrinketLimitsDuration: true,
	})

	// AttackPower / RangedAttackPower
	shared.NewSimpleStatActive(14554)  // Cloudkeeper Legplates - https://www.wowhead.com/forever/spell=18787
	shared.NewSimpleStatActive(21180)  // Earthstrike - https://www.wowhead.com/forever/spell=25891
	shared.NewSimpleStatActive(23041)  // Slayer's Crest - https://www.wowhead.com/forever/spell=28777
	shared.NewSimpleStatActive(249470) // Molten Heart of the Mountain - https://www.wowhead.com/forever/spell=1249113

	// FireResistance
	shared.NewSimpleStatActive(13164) // Heart of the Scale - https://www.wowhead.com/forever/spell=17275

	// FrostDamage / ShadowDamage
	shared.NewSimpleStatActive(249469) // Frozen Heart of the Mountain - https://www.wowhead.com/forever/spell=1249110

	// HealingPower
	shared.NewSimpleStatActive(20636) // Hibernation Crystal - https://www.wowhead.com/forever/spell=24998
	shared.NewSimpleStatActive(23047) // Eye of the Dead - https://www.wowhead.com/forever/spell=28780

	// HealingPower / SpellDamage
	shared.NewSimpleStatActive(18820) // Talisman of Ephemeral Power - https://www.wowhead.com/forever/spell=23271
	// Talisman of Ascendance - https://www.wowhead.com/forever/spell=28200
	shared.NewStackingStatBonusCD(shared.StackingStatBonusCD{
		Name:                  "Talisman of Ascendance",
		ID:                    22678,
		CD:                    time.Millisecond * 60000,
		Callback:              core.CallbackOnSpellHitDealt | core.CallbackOnHealDealt,
		ProcMask:              core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeOHSpecial | core.ProcMaskRangedAuto | core.ProcMaskRangedSpecial | core.ProcMaskSpellDamage | core.ProcMaskSpellHealing,
		Outcome:               core.OutcomeLanded,
		RequireDamageDealt:    false,
		TrinketLimitsDuration: true,
	})
	shared.NewSimpleStatActive(23046) // The Restrained Essence of Sapphiron - https://www.wowhead.com/forever/spell=28779

	// Health
	shared.NewSimpleStatActive(13966) // Mark of Tyranny - https://www.wowhead.com/forever/spell=1287842

	// MP5
	shared.NewSimpleStatActive(4696) // Lapidis Tankard of Tidesippe - https://www.wowhead.com/forever/spell=1135

	// Skipped
	// Not simulated: Lei of Lilies: "Conjure Lily Root" (18831) - ignored effect type 24
	// https://www.wowhead.com/forever/spell=18831
	// Not simulated: Staff of Conjuring: "Conjure Food" (8736) - ignored effect type 24
	// https://www.wowhead.com/forever/spell=8736
	// Not simulated: Orb of Deception: "Orb of Deception" (16739) - ignored aura type 56
	// https://www.wowhead.com/forever/spell=16739
	// Not simulated: Spider Belt: "Immune Root" (9774) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=9774
	// Not simulated: Cold Basilisk Eye: "Cold Eye" (1139) - ignored aura type 33
	// https://www.wowhead.com/forever/spell=1139
	// Not simulated: Enchanted Moonstalker Cloak: "Form of the Moonstalker" (6298) - ignored aura type 56
	// https://www.wowhead.com/forever/spell=6298
	// Not simulated: Ornate Mithril Boots: "Immune Root" (9774) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=9774
	// Not simulated: Signet of Expertise: "Summon Smithing Hammer" (11209) - ignored effect type 24
	// https://www.wowhead.com/forever/spell=11209
	// Not simulated: Glimmering Mithril Insignia: "Fearless" (12733) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=12733
	// Not simulated: Chained Essence of Eranikus: "Cloud of Chained Essence" (12766) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=12766
	// Not simulated: Stealthblade: "Stealth" (1300180) - ignored aura type 16
	// https://www.wowhead.com/forever/spell=1300180
	// Not simulated: Book of the Dead: "Summon Skeleton" (17490) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=17490
	// Not simulated: Cannonball Runner: "Summon Scarlet Cannon" (6251) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=6251
	// Not simulated: Barov Peasant Caller: "Death by Peasant" (18307) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=18307
	// Not simulated: Barov Peasant Caller: "Death by Peasant" (18308) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=18308
	// Not simulated: Arcanite Dragonling: "Arcanite Dragonling" (19804) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=19804
	// Not simulated: Ancient Cornerstone Grimoire: "Summon Skeleton" (17490) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=17490
	// Not simulated: Insignia of the Horde: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Horde: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Horde: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Horde: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Horde: "Immune Fear/Polymorph/Snare" (23274) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23274
	// Not simulated: Insignia of the Horde: "Immune Fear/Polymorph/Stun" (23276) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23276
	// Not simulated: Insignia of the Horde: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Horde: "Immune Charm/Fear/Stun" (23277) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23277
	// Not simulated: Insignia of the Alliance: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Alliance: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Alliance: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Alliance: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Alliance: "Immune Fear/Polymorph/Snare" (23274) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23274
	// Not simulated: Insignia of the Alliance: "Immune Fear/Polymorph/Stun" (23276) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23276
	// Not simulated: Insignia of the Alliance: "Immune Charm/Fear/Stun" (23277) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23277
	// Not simulated: Insignia of the Alliance: "Immune Fear/Polymorph/Stun" (23276) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23276
	// Not simulated: Dimensional Ripper - Everlook: "Everlook Transporter" (23442) - ignored effect type 252
	// https://www.wowhead.com/forever/spell=23442
	// Not simulated: Enamored Water Spirit: "Mana Spring Totem" (24854) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=24854
	// Not simulated: Defender of the Timbermaw: "Defender of the Timbermaw" (26066) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=26066
	// Not simulated: Vanquished Tentacle of C'Thun: "Tentacle Call" (26391) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=26391
	// Not simulated: Insignia of the Alliance: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Alliance: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Alliance: "Immune Fear/Polymorph/Stun" (23276) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23276
	// Not simulated: Insignia of the Alliance: "Immune Fear/Polymorph/Stun" (23276) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23276
	// Not simulated: Insignia of the Alliance: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Alliance: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Alliance: "Immune Charm/Fear/Stun" (23277) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23277
	// Not simulated: Insignia of the Alliance: "Immune Fear/Polymorph/Snare" (23274) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23274
	// Not simulated: Insignia of the Horde: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Horde: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Horde: "Immune Fear/Polymorph/Stun" (23276) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23276
	// Not simulated: Insignia of the Horde: "Immune Charm/Fear/Polymorph" (23273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23273
	// Not simulated: Insignia of the Horde: "Immune Fear/Polymorph/Snare" (23274) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23274
	// Not simulated: Insignia of the Horde: "Immune Charm/Fear/Stun" (23277) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=23277
	// Not simulated: Insignia of the Horde: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: Insignia of the Horde: "Immune Root/Snare/Stun" (5579) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=5579
	// Not simulated: SAF-T Emergency Ripper: Everlook: "Everlook Transporter" (23442) - ignored effect type 252
	// https://www.wowhead.com/forever/spell=23442
	// Not simulated: Defender of the Barkskin: "Defender of the Barkskin" (1294063) - ignored effect type 28
	// https://www.wowhead.com/forever/spell=1294063
	// Not simulated: Graverobber's Shovel: "Dig" (1292560) - ignored effect type 24
	// https://www.wowhead.com/forever/spell=1292560
	// Not simulated: Greater Insignia of the Horde: "PvP Trinket" (438273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=438273
	// Not simulated: Greater Insignia of the Alliance: "PvP Trinket" (438273) - ignored aura type 77
	// https://www.wowhead.com/forever/spell=438273
	// Not simulated: Field Agent Beverage: "Field Agent Beverage" (1297448) - ignored aura type 17
	// https://www.wowhead.com/forever/spell=1297448

	// SpellCritRating
	shared.NewSimpleStatActive(19952) // Gri'lek's Charm of Valor - https://www.wowhead.com/forever/spell=24498

	// SpellDamage / SpellPiercing
	shared.NewSimpleStatActive(21473) // Eye of Moam - https://www.wowhead.com/forever/spell=26166

	// Spirit
	shared.NewSimpleStatActive(272439) // Serenity Field - https://www.wowhead.com/forever/spell=1291103
}
