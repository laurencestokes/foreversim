# Build arena

The Build Arena compares complete upstream DPS presets at level 60. A result includes the preset's
gear, enchants, talents, resolved rotation, consumables and class options. Changing the build can change
all of them. These results help choose a setup to investigate; they do not isolate the value of a
single talent or compare classes in equal gear.

Presets include mixed gear levels and development items. Warlock uses upstream's `preraid` gear
because its build presets provide no gear, matching the upstream regression suites. A development
preset, incomplete rotation or missing mechanic can change a ranking substantially. The preset's
status and caveats belong beside its score; coverage alone does not establish that a spec is complete.
The saved setup shows the equipment, suffixes, enchants, weapon swaps and selected talent ranks.
Links open current item and spell descriptions, which can differ from the saved snapshot.

## Choosing a comparison

The page compares one fight length and target type at a time. It offers 90-second, 3-minute and
6-minute fights against Mechanical, Elemental and Beast targets. The target is level 63 with 3,731
armor. Every simulation uses seed 101 and the engine's full raid, party and individual buffs and
debuffs. Each legal race receives 50,000 iterations for each build and fight.

The default selects the highest observed mean DPS among each build's legal races. Choosing a race
instead compares that race and excludes builds its class cannot use. Class selection changes the
comparison population. Search filters the displayed rows after the leader has been selected, so
hiding a name does not silently change the reference score. The [Race Arena](race-arena.md) compares
races while keeping the build fixed.

DPS includes the player's pets and measures personal damage under full buffs. Raid buffs and debuffs
that a spec provides are already present in that baseline. The comparison therefore does not price
that spec's contribution to other players, survival, utility, movement or target mechanics. The
presets and races are not optimized separately for these encounters.

## Scores and uncertainty

The leader is the highest observed mean DPS in the selected population. A build's gap is
`100 × (1 − build DPS / leader DPS)`. The page rounds the gap to two decimal places before assigning
S at up to 5%, A at up to 10%, B at up to 20%, C at up to 30%, and D above 30%. The tiers describe
this comparison and have no fixed DPS threshold. Race spread is `100 × (1 − lowest race DPS / highest
race DPS)` within the same build and fight.

Each saved mean has standard error `SE = iteration standard deviation / sqrt(iterations)`. The
DPS interval is the mean plus or minus `1.96 × SE`. Two rows are marked indistinguishable at this
precision when their mean difference is at most `1.96 × sqrt(SE₁² + SE₂²)`.

For a build mean `A`, leader mean `L`, and their standard errors `sA` and `sL`, the gap interval is
centered on `100 × (1 − A/L)` with half-width
`196 × sqrt((sA/L)² + (A × sL/L²)²)`. The leader compared with itself has a gap of zero.
The page marks a tier as uncertain when the interval crosses a tier boundary.

These are normal and delta-method approximations using independent errors. The shared seed does
not supply paired covariance measurements. An interval describes simulation sampling error; it does
not cover inaccurate mechanics, uncertain tuning or unrepresentative presets. Picking the best race
or leader from the observed means also favors positive sampling noise. The intervals make no
simultaneous guarantee over all the comparisons, and an indistinguishable result does not establish
that two builds are identical.

## Inputs and publication

`ui/specs/race_arena_export.test.ts` applies the UI defaults and each preset through the same build
application and serialization used by the simulator. `tools/race_arena/snapshot.mjs` attaches the
saved setup and a stable build identity from `tools/build_arena/catalog.json`. A build's identity
survives a display-label change. Its input fingerprint records the content of its resolved request;
a matching name or identity cannot substitute for matching inputs. Display aliases are kept separate
from fingerprints.

The runner's manifest records every exported build, its legal races and the request needed to
reproduce it. Publication requires the complete set of fight lengths, target types and legal races,
matching input fingerprints and finite results at 50,000 iterations. Duplicate or missing cells,
invalid provenance and smoke results fail validation. The same raw matrix supplies both arenas.

The manual **Build Arenas** workflow exports once and creates the manifest without simulating it.
A separate eight-iteration benchmark measures each build at every fight length, summing elapsed
times across all legal races and target types. For each fight length, the workflow assigns the
largest estimated build costs first to the least-loaded of 16 shards. These measurements include
concurrent CPU contention and are estimates; eight iterations do not predict every long-run cost.
Benchmark output is kept outside the publication inputs.

The workflow computes the full matrix once at 50,000 iterations per cell. Each shard records its
simulator revision and computation date. After validation, both scorers write into a staging
directory. Only a complete successful pair is copied into the checkout and committed together. A
branch that has moved from the simulated revision fails publication, and a rejected push is never
retried by merging unrelated work. A successful publication on `master` starts Pages deployment.
Runs on other branches only commit their results there.

## Rebuild locally

After installing dependencies and generating the Go and TypeScript sources, use new output paths:

```sh
revision=$(git rev-parse HEAD)
RACE_ARENA_INPUT=/tmp/arena-input.json npx vitest run ui/specs/race_arena_export.test.ts
node tools/race_arena/snapshot.mjs /tmp/arena-input.json
go run -tags=with_db ./tools/race_arena/simulate -input /tmp/arena-input.json -commit "$revision" -iterations 50000 -manifest /tmp/arena-manifest.json
go run -tags=with_db ./tools/race_arena/simulate -input /tmp/arena-input.json -commit "$revision" -iterations 50000 -out /tmp/arena-raw
```

Create a provenance file whose `sources` array records each source's `id`, `commit`, `computedAt`,
`reused` flag and note. Its dates must match the raw lists' recorded computation dates. Then validate
and stage both results before replacing the published files:

```sh
node tools/build_arena/publish.mjs --manifest /tmp/arena-manifest.json --raw /tmp/arena-raw/presets.json --provenance /tmp/arena-sources.json --upstream UPSTREAM_SHA --verified "$revision" --out /tmp/build-results.json
RACE_ARENA_COMMIT="$revision" go run ./tools/race_arena /tmp/arena-raw /tmp/race-results.json
node --test tools/build_arena/*.test.mjs tools/race_arena/snapshot.test.mjs
go test ./tools/race_arena/...
```

`UPSTREAM_SHA` identifies the upstream source included in the engine. Keep raw results, original
inputs and provenance together. `-workers` limits concurrency and `-durations` selects fight lengths;
publication still requires the full expected matrix. A low-iteration run checks that inputs can run
and must never replace published results.

## Replay a downloaded request

Download a row's request from its setup dialog, then run it with the matching simulator revision:

```sh
go run -tags=with_db ./tools/build_arena/replay -input /tmp/build-request.json -out /tmp/build-result.json
```

The command uses the saved request's settings, including its race, encounter, buffs, iterations and
seed. It writes the complete `RaidSimResult` and prints mean raid DPS, standard error and completed
iterations. Arena requests contain one player, so raid DPS includes that player and their pets.
Simulation errors remain in the result file and make the command fail.

## Reusing existing computations

Reuse requires the same resolved inputs, encounter, legal races and combat behavior. Compare the
full inputs and setup snapshots, audit every intervening engine and data change, and run bounded
before-and-after comparisons on the changed paths. A source change confined to one race requires
that race's complete replacement matrix. An unexplained numerical difference requires new
computations for the affected cells.

Preserve the source engine revision and original computation date when data is reused. Record a
compatibility check against a newer revision separately. A publication date or file modification
date is not a computation date. Unknown dates remain unknown.

The October 27 matrix contains 34 builds, 306 fight lists and 2,025 legal race matchups. The original
cloud engine was `932ffdbf86205ac9f388ecd161e89b77206e1989`. GitHub's
[successful run](https://github.com/laurencestokes/foreversim/actions/runs/40545126193) records its
48 simulation steps between October 27, 2026, 16:54:32 and 18:17:49 UTC. The 261 Troll matchups were
replaced using engine `66ef260625b26bdfb388af40abe2c1616500a38d` for the 15-second Berserking
duration; their exact execution timestamp was not recorded. The combined rankings were scored at
18:21:25 UTC on October 27. That scoring time must not be presented as the Troll computation time.

The current preset comparison accounts for five display-only Mage group labels and a pseudo-stat
array shrinking from 42 to 37 entries. Every entry from index 36 onward was zero in both exports;
the remaining resolved inputs and all setup snapshots matched exactly. The merged upstream change
extracts existing crit calculations for the character sheet and preserves the combat expressions.
A before-and-after run at two iterations matched unrounded DPS and standard error exactly in all
306 lists and 2,025 race matchups. A longer check at 256 iterations matched both values exactly in
333 matchups across 42 lists: 14 builds at 180 seconds against all three target types and every legal
race. It covered both feral builds, three weaving hunter builds, six rogue builds, Enhancement and
both warrior builds. These bounded checks support reuse together with the input and code comparison;
they do not test every possible roll sequence. Published estimates retain their original 50,000
iterations and source dates.

To reproduce that check, build the runner at `66ef260625b26bdfb388af40abe2c1616500a38d` and at the
verified merged revision with the same Go version, platform and `with_db` tag. Keep each engine's
saved full export, and select the 14 builds above into matching sample inputs. Run each pair into
separate new directories:

```sh
"$OLD_RUNNER" -input "$OLD_INPUT" -out /tmp/arena-old-smoke -iterations 2 -workers 2
"$NEW_RUNNER" -input "$NEW_INPUT" -out /tmp/arena-new-smoke -iterations 2 -workers 2
"$OLD_RUNNER" -input "$OLD_SAMPLE" -out /tmp/arena-old-sample -iterations 256 -workers 2 -durations 180
"$NEW_RUNNER" -input "$NEW_SAMPLE" -out /tmp/arena-new-sample -iterations 256 -workers 2 -durations 180
```

Match lists by build identity, duration and target, then compare each race's raw `dps` and `error`
with exact numeric equality. Check the expected list and race counts before interpreting a pass.
Map only the five known Mage labels, and assert that every removed pseudo-stat entry is zero before
normalizing the saved inputs. A mismatch fails the reuse proof and needs an explanation or new
computations; it must not be hidden by display rounding or a statistical tolerance.
