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
