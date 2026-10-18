# Race arena

The simulator, game data and preset builds follow [wowsims/forever](https://github.com/wowsims/forever).
The race arena compares every legal race on each DPS build. It keeps the preset's gear, talents,
resolved rotation, consumables and class options fixed. Warlock presets contain no gear, so they use
the upstream `preraid` set, as the warlock regression suites do. Development gear sets are included.

Every run uses the engine's full raid, party and individual buffs and debuffs, seed 101, and a fixed
180 second fight against a level 63 target with 3,731 armor. Each build is run against Mechanical,
Elemental and Beast targets. Both Skyborne races are included. Weapon types are unchanged.

The preset export goes through the UI's existing defaults, build application and player serialization.
Its test checks that each build has gear, talents and a resolved rotation before writing any input.
The Go runner uses the canonical engine without changes to its race mechanics.

## Rebuild

After installing the dependencies and generating Go and TypeScript sources:

```sh
RACE_ARENA_INPUT=/tmp/race-arena-input.json npx vitest run ui/specs/race_arena_export.test.ts
go run -tags=with_db ./tools/race_arena/simulate -input /tmp/race-arena-input.json -out /tmp/race-arena-out
go run ./tools/race_arena /tmp/race-arena-out
npx oxfmt ui/app/race_arena/results.json
```

Use an empty output directory. The runner defaults to 50,000 iterations per race; `-iterations 2`
checks that every build can run, but those results must not replace the published rankings.
`-workers` limits concurrent simulations. A failed simulation stops the run before writing results.

The manual **Build Race Arena** workflow performs these steps, commits the result and starts the
Pages deployment on `master`. A concurrent branch update rejects the push so results cannot be
published under an engine revision they did not use.

The fork deploys on pushes to `master` at `/foreversim/forever/`. Its GitHub-hosted test and Pages
workflows replace upstream's private runner and deployment credentials. `tools/site_base.sh`
adjusts the built assets to this path; the source simulator retains upstream's `/forever/` paths.
Upstream's scheduled maintenance, webhook, labeler and preview cleanup jobs run only in `wowsims/forever`.
