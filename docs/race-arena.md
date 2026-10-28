# Race arena

The Race Arena compares every legal race on each upstream DPS build. Within a build, gear, talents,
resolved rotation, consumables and class options stay fixed. The [Build Arena](build-arena.md)
compares the complete presets against each other and explains the shared inputs, publication checks
and source provenance.

The simulator, game files and presets follow [wowsims/forever](https://github.com/wowsims/forever),
including the [October 26 racial tuning](https://www.wowhead.com/forever/news/racial-ability-tuning-pass-for-wow-forever-383597)
and [October 27 addendum](https://us.forums.blizzard.com/en/wow/t/2360696/8).
Human has 1% crit and 10% increased Spirit; Dwarf has 1% physical and spell hit; Tauren retains its
health bonus with no crit-damage bonus; Orc's Command grants 5% damage to hunter and warlock pets;
Skyborne has 2% haste. Earth Fury and Savage are removed. There are no fork-specific racial overrides.
Blood Fury has a 2-minute cooldown, Touch of the Grave has a 15-second internal cooldown, and Troll's
Berserking lasts 15 seconds. The raid baseline uses ordinary Read Leyline and Skysight casts.
Entering a raid removes Attuned and Charged, so the preset export rejects their enhanced casts.
As in upstream, Plainsrunning and health regeneration are not simulated.

This is a live-game baseline using upstream's level-60 presets with unequal gear levels. The game
files and hotfix cache are from live wowf build 1.60.1.70658. Items absent from that cache are omitted
from the simulator database; cache absence does not establish that an item is unavailable in game.
Upstream substitutes preset items without gear optimization, and some vanilla-ID replacements do
not activate the registered Forever set bonuses. See the [data limitations](build-arena.md#live-game-data-and-preset-limitations).

Warlock uses upstream's `preraid` set because its presets provide no gear, matching its regression
suites. Gear and rotations are not optimized separately for each race. DPS includes pets under full
raid, party and individual buffs and debuffs, with no value assigned to survival, utility or damage
a spec adds to other players.

Every race receives 50,000 iterations with seed 101 against a level 63 target with 3,731 armor. The
matrix covers 90-second, 3-minute and 6-minute fights against Mechanical, Elemental and Beast targets.
Both Skyborne races are included, and weapon types stay unchanged.

## Reading the page

The page opens on **Best race for your fights**. The reader chooses fight lengths and target types.
For each build, the recommendation minimizes a race's largest percentage gap behind the best race
across those fights. Races within the displayed noise estimate are named as level with it.
**Show detail** opens the race-by-fight heatmap and each race's gap by fight length at one target.
Per-fight tier lists follow.

Each build's **View setup** dialog shows its saved equipment, random suffixes, enchants, active weapon
swap gear and selected talent ranks. Item names come from the exported player's own database;
talent names and ranks come from the matching definitions. Item and spell links open current game
descriptions, which may change after the snapshot.

The race page's noise figure is the largest standard error of a race's mean, relative to the winning
DPS in its fight. Its difference estimate is twice the square root of two times that value. This is
an independent-error approximation, not a measured paired interval. A shared seed alone does not
establish covariance, and the estimate does not cover model errors or guarantee every comparison
simultaneously. Choosing a leader from noisy means also favors positive sampling noise. Races called
level should not be treated as proven identical or uniquely optimal.

## Rebuild and publication

The manual **Build Arenas** workflow computes one complete raw matrix and publishes both arenas
from it. See [the rebuild commands and validation rules](build-arena.md#rebuild-locally). The workflow
balances shards independently for each fight length using a bounded timing run, validates legal
coverage and fingerprints, and stages both results before committing them together. Smoke results
cannot pass publication's 50,000-iteration requirement.

The live-data refresh recomputes every legal race, build and encounter at 50,000 iterations. Its
presets and game data differ from the October 27 inputs, so those computations cannot be reused.

A change confined to one race can reuse the other races' raw results only when presets, encounter
and all other combat behavior are equivalent. Run the simulator with `-race Troll -commit REVISION`
to recompute every legal Troll matchup, then replace those rows in the complete raw matrix. Preserve
the source revisions and recorded dates, and validate the merged matrix before scoring either arena.
The Race Arena's `raceCommits` records a replacement revision separately from the original engine.
The Build Arena's sources retain each computation's provenance and any separate compatibility check.

For older results, attach setup snapshots using the original saved input:

```sh
node tools/race_arena/snapshot.mjs /path/to/original-input.json ui/app/race_arena/results.json
```

This reads talent definitions at the results' recorded commit and preserves the numbers and dates.
A fresh export is not evidence of what an older run equipped. The October 27 source dates and the
Troll replacement's unknown execution timestamp are documented in the
[historical October 27 publication](build-arena.md#historical-october-27-publication).

The fork deploys successful `master` publications at `/foreversim/forever/`. Its GitHub-hosted tests
and Pages workflow replace upstream's private runner and deployment credentials. `tools/site_base.sh`
adjusts built asset paths; the simulator source retains upstream's `/forever/` paths. Upstream's
scheduled maintenance, webhook, labeler and preview cleanup jobs run only in `wowsims/forever`.
