# ForeverSim

An unofficial DPS, tank and healing simulator for World of Warcraft®: Forever, the Classic+ relaunch
announced at BlizzCon 2026. Not affiliated with Blizzard or the WoWSims team.

**Live site:** https://laurencestokes.github.io/foreversim/forever/

## Where it comes from

This repository is built on:

- [wowsims/forever](https://github.com/wowsims/forever), the official WoWSims Forever sim, whose engine
  this is, and
- [ElliotWood/Forever](https://github.com/ElliotWood/Forever), which layers Forever's rule changes, data
  tooling, documentation and product pages on top of it.

Both are MIT licensed, as is this repository (see [LICENSE](LICENSE)). WoWSims asks that anyone using the
software keeps a user-visible link back to the original project; the site's landing page does.

What this repository adds so far:

- **Forever racials** from the beta client's own data (build 1.60.1.69977): the reworked racials, the
  Skyborne, Forever's race and class pairings, and Eureka! for warriors, warlocks and rogues. Every value
  is listed with its spell id in [docs/forever_rules.md](docs/forever_rules.md#racials).
- **Race analysis** tests that take each race's DPS apart, piece by piece, on identical gear
  ([results](docs/race-analysis/)).
- No analytics, and nothing a visitor drops on the site leaves their browser.

## Running it locally

The simplest way is Docker. From the repository folder:

```sh
docker build --target prod -t foreversim:local .
docker run -d --name foreversim -p 8080:8080 foreversim:local
```

Then open http://localhost:8080/forever/. Stop it with `docker rm -f foreversim`, and rebuild the image
after any change. For a development setup with live reload, see [docs/installation.md](docs/installation.md)
and [docs/commands.md](docs/commands.md).

## Debug: weapon type override

Every equipped main-hand or off-hand weapon has a "Weapon type (debug override)" selector in its gear
picker. Choosing a type (say, a sword relabelled as an axe) changes only what the weapon counts as for
effects that key on weapon type: racials such as Human Sword, Orc Axe and Dwarf Mace Specialization,
talents such as the rogue's Hack and Slash, and ability requirements such as Backstab and Mutilate
needing a dagger. Its stats, procs and every other effect are unchanged. An overridden weapon shows an
"as <type>" badge in the gear list. It exists to compare races fairly on otherwise identical gear; it
has no in-game equivalent.

## The race analysis tests

Opt-in Go tests, skipped unless their variable is set. With Go, protoc and the generated protos in place
(`make proto`):

```sh
RACE_BREAKDOWN=1 go test --tags=with_db ./sim/warrior/dps/ -run TestRaceBreakdown -v
RACE_BREAKDOWN=1 go test --tags=with_db ./sim/warlock/ -run TestRaceBreakdown -v
RACE_BREAKDOWN=1 go test --tags=with_db ./sim/rogue/ -run TestRaceBreakdown -v
EUREKA_SPLIT=1 go test --tags=with_db ./sim/rogue/ -run TestEurekaSplit -v
```

`RACE_BREAKDOWN_OUT` / `EUREKA_SPLIT_OUT` write the tables to a file. See `core.RacialBreakdown` in
[sim/core/racial_breakdown.go](sim/core/racial_breakdown.go) for how each column is measured.

## The race tier list

The [race tier list](https://laurencestokes.github.io/foreversim/forever/race_arena/) ranks every race
on each DPS spec's community builds, S to D, with the method and caveats at the top of the page. The page
only renders `ui/app/race_arena/results.json`; to regenerate it (about a quarter of an hour on 16 threads):

```sh
RACE_ARENA_OUT=race-arena-out go test --tags=with_db -count=1 -timeout 120m -run TestRaceArena $(go list ./sim/... | grep -v /sim/web)
go run ./tools/race_arena race-arena-out ui/app/race_arena/results.json
```

Without `RACE_ARENA_OUT` every `TestRaceArena` skips. `RACE_ARENA_ITERATIONS` overrides the 50,000
iterations for a quick check, and `RACE_ARENA_COMMIT` names the commit when git cannot (a container over a
worktree). The manual **Build Race Arena** workflow does the same run and commits the file. See
[sim/arenalib/race_arena.go](sim/arenalib/race_arena.go) for the method.

## Keeping up with upstream

Upstream changes are merged by hand, after review and a test run:

```sh
git remote add elliot https://github.com/ElliotWood/Forever.git
git remote add wowsims https://github.com/wowsims/forever.git
git fetch elliot wowsims
git merge elliot/master      # or wowsims/master
```

Engine fixes that belong upstream are offered to [wowsims/forever](https://github.com/wowsims/forever) as
pull requests of their own.

## Deploying

Every push to `master` runs the tests, builds the site and publishes it to this repository's GitHub Pages
(`.github/workflows/deploy.yml`). Pages must be set to deploy from GitHub Actions
(Settings > Pages > Source).
