# November 4 arena result reuse

This update takes WoWSims master `92c4c35b9e2364200ffb9e0f24573fb4904fac8f`, Hunter proposal #848 at `5bff5deb7c812ce6b63b6f76b6163e4da46fdb05`, and Feral proposal #823 at `8f05366a72cf2256919c45aeca8991e43b50f5bb`. The baseline publication is fork master `3a336c9dd233ed52053af2a3fc95e80787803793`, verified against engine `6ae736635d68f1c89629cf8d5dc7e98d2a4a6940`.

The selection in `tools/build_arena/nov4-upstream.json` reruns 29 builds and reuses 12. The catalog has 41 builds and 2,529 race, target and duration combinations. Of these, 1,971 receive new measurements and 558 retain their original measurements and provenance. Every published matchup has 50,000 iterations.

## Unchanged builds

| Builds                                                                          | Evidence                                                                                                                                                                                                                                                 |
| ------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Mage FFB, Fire, Frost Incineration and Frost Meditation                         | Their requests and spell implementations are unchanged. Improved Frostbolt gains a proc hint, which its static talent parser does not use. The changed Arcane prepull actions belong to separate rotations.                                              |
| Paladin Unsealed Wrath and Holy Shock                                           | Neither build takes Holy Shield, whose ladder gains a rank. Vengeance and Two-Handed Weapon Specialization retain the same effective row values.                                                                                                         |
| Shadow and Smite Priest                                                         | Damage calculations are unchanged. Health costs move to resource metrics. The retained Shadow Word: Death rule still applies backlash on a nonlethal hit. Neither rotation reads player health. Disease-removal rows are outside their damage rotations. |
| Demonology Fire Brand, Demonology Shadow Brand, Destruction and SM/Ruin Warlock | Their requests, rotations and spell rows are unchanged. Life Tap and Hellfire retain their costs under the revised health metrics. Amplify Curse remains manual, and Shadow and Flame reads the same rank amounts.                                       |

For all 12 builds, the full request, fingerprint, setup, gear, rotation, consumable names and legal races match the previous publication. Both arenas have exactly one original measurement for each legal race in every combination of 90, 180 and 360 seconds and Mechanical, Elemental and Beast targets.

Differential probes ran the old verified engine and the integrated engine for eight iterations at 180 seconds, across all legal races and all three target types for these 12 builds. All 186 comparisons produced exactly equal DPS and sampling error. These short probes support the code and input checks. The retained published measurements keep their original 50,000 iterations, source revisions and dates.

## Shared inputs and behavior

The reaction range is disabled when the upper bound is absent. All 12 reused builds omit that bound, so it adds no random rolls and their aura checks retain the fixed reaction time. Movement logging and stat-weight presentation do not alter their combat. Disabled cooldowns preserve their manual activation, and non-tanks retain their defensive cooldown restriction. The encounters contain no stuns that would activate the changed PvP trinket behavior.

Ten item rows change: Antipodean Rod, Glimmering Mail Breastplate and Vanadium Loop are added; Earthshaker, Choker of the Fire Lord, Quick Strike Ring, Heavy Dark Iron Ring, Band of Sulfuras, Cauterizing Band and Ring of Spell Power change. None is equipped or swapped by any current arena build. Enchants, consumables, random suffixes and item-effect data remain unchanged for this comparison. The End of Beta event consumables and weapons are excluded upstream.

The spell-store changes that affect damage belong to Druid and Shaman. Other changed rows cover disease removal, Improved Frostbolt metadata, Holy Shield and excluded event consumables. Racial implementations remain unchanged. The arena still uses the same controlled encounters even though the upstream Hunter default now selects Dragonkin.

## New measurements

- Balance, Feral Cat and Feral Hybrid use the captured hotfix values. Feral bleed and Bite attack-power overrides replace the earlier preview multipliers.
- All seven Hunters use reactions drawn between 150 and 350 ms. The weave rotations also change.
- Arcane and Arcane Fire change prepull timing.
- All 11 Rogues use the updated rotation and mechanics. Six original allocations remain. Four upstream allocations change, and Combat Venom replaces the older upstream Subtlety Backstab candidate.
- Both existing Shaman builds use revised data and mechanics. The new Mana Tide allocation receives its first measurements.
- Affliction Wrack changes talents, rotation, consumables and demon settings. It and Destruction keep the existing pre-raid gear.
- Arms and Fury change talents and rotations.

The publication validator checks the unchanged requests and complete coverage again before accepting reuse. A later change to any reused setup or shared behavior requires a new compatibility assessment.
