# Race arena

The simulator, game data and preset builds follow [wowsims/forever](https://github.com/wowsims/forever),
with one explicit race-mechanics exception: Dwarf's Earth Fury grants +2% critical damage and healing
with any weapon or none, including through item swaps. This follows the unrestricted bonus in the
[October 15 development notes](https://us.forums.blizzard.com/en/wow/t/2360696/6) and the spell tooltip.
The race arena compares every legal race on each DPS build. It keeps the preset's gear, talents,
resolved rotation, consumables and class options fixed. Warlock presets contain no gear, so they use
the upstream `preraid` set, as the warlock regression suites do. Development gear sets are included.

The aim is a controlled level-60 baseline for race choice ahead of release. These are the upstream
presets, including development items, rather than a claim about final release gear. Only the race
changes within a build; gear and rotations are not optimized separately for each race. Rankings
measure damage, including pets, with no value assigned to survival or utility. Changes to gear,
balance, buffs or the encounter can change the ordering.

The page opens on **Best race for your fights**. A reader picks fight lengths and target types, and
each build names the race whose largest gap behind the best race, across those fights, is smallest:
the race that is never far off, rather than the best on average. Races within the noise figure of it
are named as level with it. **Show detail** opens a heatmap of every race against every fight (tier
letter in each cell, shaded on a one-hue ramp validated against the page surface; unpicked fights are
faded) and a chart of each race's gap by fight length at one target. The per-fight tier lists follow.

Each build's **View setup** dialog shows the equipped items, random suffixes, enchants, active weapon
swap gear and selected talent ranks. These are saved with the results. Item names come from the
exported player's own database; talent names and ranks come from the matching talent definitions.
The item and spell links open current game descriptions, which may change after the snapshot.

Every run uses the engine's full raid, party and individual buffs and debuffs, seed 101, and a fixed
length fight against a level 63 target with 3,731 armor: 90 seconds, 3 minutes and 6 minutes, with 3
minutes the page's main view. Each build is run against Mechanical, Elemental and Beast targets at
each length. The runner's `-durations` flag (default `90,180,360`) sets the lengths. Both Skyborne races are included. Weapon types are unchanged.

The preset export goes through the UI's existing defaults, build application and player serialization.
Its test checks that each build has gear, talents and a resolved rotation before writing any input.
The Go runner uses the same engine as the simulator, including the Earth Fury exception above.

## Rebuild

After installing the dependencies and generating Go and TypeScript sources:

```sh
RACE_ARENA_INPUT=/tmp/race-arena-input.json npx vitest run ui/specs/race_arena_export.test.ts
node tools/race_arena/snapshot.mjs /tmp/race-arena-input.json
go run -tags=with_db ./tools/race_arena/simulate -input /tmp/race-arena-input.json -out /tmp/race-arena-out
go run ./tools/race_arena /tmp/race-arena-out
npx oxfmt ui/app/race_arena/results.json
```

Use an empty output directory. The runner defaults to 50,000 iterations per race; `-iterations 2`
checks that every build can run, but those results must not replace the published rankings.
Three fight lengths make a full run three times as long as a single one.
`-workers` limits concurrent simulations. A failed simulation stops the run before writing results.

The runner carries each setup snapshot into its raw results, and the scorer saves one copy per build
across target types. Missing or inconsistent snapshots stop publication. Run
`node --test tools/race_arena/snapshot.test.mjs` to check the snapshot extraction.

To attach setups to older results, use the original saved input from that simulation run:
`node tools/race_arena/snapshot.mjs /path/to/original-input.json ui/app/race_arena/results.json`.
This reads talent definitions at the results' recorded commit and preserves the DPS, ranks, generation
date and engine revision. Do not substitute a fresh export after presets have changed.

The manual **Build Race Arena** workflow performs these steps, commits the result and starts the
Pages deployment on `master`. A concurrent branch update rejects the push so results cannot be
published under an engine revision they did not use.

The fork deploys on pushes to `master` at `/foreversim/forever/`. Its GitHub-hosted test and Pages
workflows replace upstream's private runner and deployment credentials. `tools/site_base.sh`
adjusts the built assets to this path; the source simulator retains upstream's `/forever/` paths.
Upstream's scheduled maintenance, webhook, labeler and preview cleanup jobs run only in `wowsims/forever`.
