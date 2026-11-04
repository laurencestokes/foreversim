# Build arena

[November 2 patch coverage by class](november-2-coverage.md) records which launch changes these results include and where the model still differs.

Results marked **November 4, 2026 launch projection** use the Hunter reactive-weaving presets, Feral and Shaman draft changes and the fork’s changes for the November 2 class notes. The Rogue, Warrior and Shaman presets spend 51 points in their revised trees. The setup inspector shows the actual allocation and rotation. Saved Shank identifiers resolve to its game spell. Reckless Swings uses its game spell identifier.

Both arenas show the official notes, the verified fork engine and the upstream source pins when their results carry `launchPreview` metadata. The pins below match [the projection source record](../tools/build_arena/launch-preview.json). A recorded proposal remains in that history after its changes enter master; the list is not a list of open drafts.

The upstream baseline is [`92c4c35b9`](https://github.com/wowsims/forever/commit/92c4c35b9e2364200ffb9e0f24573fb4904fac8f).

| Source                                                                                | Recorded revision                                                                                 | Relation to the baseline    |
| ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | --------------------------- |
| [#848: Reactive Hunter](https://github.com/wowsims/forever/pull/848)                  | [`5bff5deb7`](https://github.com/wowsims/forever/commit/5bff5deb7c812ce6b63b6f76b6163e4da46fdb05) | Additional patch source     |
| [#842: Warlock hotfix](https://github.com/wowsims/forever/pull/842)                   | [`c50111ec1`](https://github.com/wowsims/forever/commit/c50111ec1f06de09a8c0f3b09e06a933fa506710) | Included in baseline master |
| [#841: Warrior hotfix](https://github.com/wowsims/forever/pull/841)                   | [`dc46a392a`](https://github.com/wowsims/forever/commit/dc46a392a71761eed89beed6a1d7476eaf385f4e) | Included in baseline master |
| [#840: Rogue hotfix](https://github.com/wowsims/forever/pull/840)                     | [`22a3ea632`](https://github.com/wowsims/forever/commit/22a3ea632916d65b36704cc74343610d086bad96) | Included in baseline master |
| [#839: Shaman hotfix](https://github.com/wowsims/forever/pull/839)                    | [`bd53fce75`](https://github.com/wowsims/forever/commit/bd53fce75876b684ae2daa447a3f2956680121dd) | Included in baseline master |
| [#838: Hunter hotfix](https://github.com/wowsims/forever/pull/838)                    | [`194f58d5c`](https://github.com/wowsims/forever/commit/194f58d5c8c595cd79869e2145eb985e1d13107c) | Included in baseline master |
| [#836: Druid hotfix](https://github.com/wowsims/forever/pull/836)                     | [`d87fc5791`](https://github.com/wowsims/forever/commit/d87fc5791502d8abb78601a468bfe2ba4622b0c6) | Included in baseline master |
| [#835: Priest hotfix](https://github.com/wowsims/forever/pull/835)                    | [`e3a4df96a`](https://github.com/wowsims/forever/commit/e3a4df96a356710c6baa0b0a86b58745bac04eac) | Included in baseline master |
| [#834: Paladin hotfix](https://github.com/wowsims/forever/pull/834)                   | [`23dc9eefc`](https://github.com/wowsims/forever/commit/23dc9eefcd45950b61d8ffb835e9bc9f53da1c0e) | Included in baseline master |
| [#833: Mage hotfix](https://github.com/wowsims/forever/pull/833)                      | [`901c5b8cf`](https://github.com/wowsims/forever/commit/901c5b8cfd3d8b0d6a24d47fa28148f21618b134) | Included in baseline master |
| [#830: Target activity](https://github.com/wowsims/forever/pull/830)                  | [`47acdeee4`](https://github.com/wowsims/forever/commit/47acdeee47f1815f5104a11be4d09ae390d57e21) | Included in baseline master |
| [#828: Remaining cast time](https://github.com/wowsims/forever/pull/828)              | [`abcfd33da`](https://github.com/wowsims/forever/commit/abcfd33da90e0b8a7cbbf0ddeb549acae3f9d372) | Included in baseline master |
| [#827: Warlock presets](https://github.com/wowsims/forever/pull/827)                  | [`0d7b8145b`](https://github.com/wowsims/forever/commit/0d7b8145b7f16ee68e914c514479d048595e5ea4) | Included in baseline master |
| [#826: Target deaths and execute phases](https://github.com/wowsims/forever/pull/826) | [`631db9290`](https://github.com/wowsims/forever/commit/631db9290d60d4cc64b1b8ea5e3a1ddd6f71a022) | Included in baseline master |
| [#824: Hunter](https://github.com/wowsims/forever/pull/824)                           | [`7f77372e5`](https://github.com/wowsims/forever/commit/7f77372e5c314a72323f55276aba027984257921) | Included in baseline master |
| [#823: Feral](https://github.com/wowsims/forever/pull/823)                            | [`8f05366a7`](https://github.com/wowsims/forever/commit/8f05366a72cf2256919c45aeca8991e43b50f5bb) | Additional patch source     |
| [#822: Bear threat](https://github.com/wowsims/forever/pull/822)                      | [`c8a275b33`](https://github.com/wowsims/forever/commit/c8a275b336d5537404c7156a574324dd5008f91e) | Additional patch source     |
| [#821: Shaman](https://github.com/wowsims/forever/pull/821)                           | [`2fa331700`](https://github.com/wowsims/forever/commit/2fa33170006905ea188b1f9835a2bee89e90c874) | Included in baseline master |
| [#803: Mage Clearcasting](https://github.com/wowsims/forever/pull/803)                | [`393c297a9`](https://github.com/wowsims/forever/commit/393c297a9d0433d97d7ef33dbb53bb8f82385ad1) | Additional patch source     |

Warlock #842 remains in the source history. Its Wrack setup is superseded by #827, which is included in baseline master. Ranked Wrack uses #827's talents, rotation and sacrificed Imp while retaining the arena's existing pre-raid equipment.

The launch date describes the projection. Each measurement's source preserves the engine revision and computation date of its actual run, including reused measurements. The verified engine records the separate compatibility check. Results without projection metadata receive no launch-preview label.

The Build Arena compares complete DPS presets based on upstream setups at level 60. A result includes the preset's
gear, enchants, talents, resolved rotation, consumables and class options. Changing the build can change
all of them. These results help choose a setup to investigate; they do not isolate the value of a
single talent or compare classes in equal gear.

The comparison uses unequal preset gear levels. Every ranked Warlock uses the existing `preraid` set.
Wrack and Destruction use the revised upstream rotations. Their upstream regression tests use raid gear. A preset's
gear, incomplete rotation or missing mechanic can change a ranking substantially. The preset's
status and caveats belong beside its score; coverage alone does not establish that a spec is complete.
The saved setup shows the equipment, suffixes, enchants, weapon swaps and selected talent ranks.
Links open current item and spell descriptions, which can differ from the saved snapshot.

Rows labeled **Development gear** include the upstream test gear in the recorded fight. On the
public class simulator, selecting the same build leaves those gear slots as they were. Use the
arena’s setup inspector to see the equipped items, or replay its downloaded request for a like-for-like comparison.

## Live game data and preset limitations

The projection combines captured game data with the draft and fork changes listed in the coverage report. The class spell data comes from game build 1.60.1.70755 with hotfix push 113605.
Shaman's Lightning Overload uses its own rank-paired spell rows for damage and coefficients. Those rows differ from the talent tooltip's half-damage description; this still needs in-game verification. Shank always hits in the model and suppresses poison and item procs. Its proc suppression also needs in-game verification. Death Mark's cap follows the tooltip formula, using melee crit and attack power at payout.
A missing simulator database entry does not establish that an item is
unavailable in game. Later data updates can restore entries, while preset replacements may remain.

[Upstream's preset substitutions](https://github.com/wowsims/forever/pull/725) use available items
with the same name where possible, or a suitable slot, type and source otherwise. They have not
been gear-optimized. Some same-name replacements use vanilla item IDs whose set IDs differ from
the registered Forever sets. This affects the Soulforge, Sorcerer's, Heroism and Five Thunders
replacements. All eleven Rogue presets wear four pieces of the original Darkmantle set (512).
Both it and the reissue (1676) grant the four-piece Energy proc. The simulator uses a 1 PPM
estimate for both versions because the game files do not give a rate.

The Rogue roster keeps the original six alternatives and adds five updated upstream allocations, including Combat Venom. Each candidate retains its recorded gear and rotation; the two original Subtlety alternatives use the same talents with different builders.

Elemental's Mana Tide alternative keeps the page's default gear and rotation. Its score counts the Shaman's own DPS; mana support for other players is outside the comparison.

## Choosing a comparison

The page compares one fight length and target type at a time. It offers 90-second, 3-minute and
6-minute fights against Mechanical, Elemental and Beast targets. The target is level 63 with 3,731
armor. Every simulation uses seed 101 and the engine's shared raid, party and individual buff and
debuff baseline. Each legal race receives 50,000 iterations for each build and fight.

Hunter's presets draw reaction times uniformly between 150 and 350 milliseconds. Both bounds remain fixed across races and appear in the saved setup and request. The arena uses the three creature types above for every build, including Hunter; the class page's default creature type does not change this matrix.

The support baseline follows upstream's `FullRaidBuffs`, `FullPartyBuffs`, `FullIndividualBuffs` and
`FullDebuffs`. Here, "full" names that shared configuration, including support a build cannot supply
itself. It does not mean the highest book rank: Strength of Earth is rank 4, Grace of Air rank 2,
Greater Blessing of Might and Wisdom rank 1, and external Battle Shout rank 6. Salvation and Blessing
of Light apply no effect. Each build retains the spell ranks chosen by its resolved preset; some
presets cast book ranks. The recorded request shows the actual choices.

The default selects the highest observed mean DPS among each build's legal races. Choosing a race
instead compares that race and excludes builds its class cannot use. Class selection changes the
comparison population. Search filters the displayed rows after the leader has been selected, so
hiding a name does not silently change the reference score. The [Race Arena](race-arena.md) compares
races while keeping the build fixed.

DPS includes the player's pets and measures personal damage under the shared support baseline. Raid
buffs and debuffs that a spec provides are already present in that baseline. The comparison does not price
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

`ui/specs/race_arena_export.test.ts` applies the UI defaults and each preset through the simulator's
settings and serialization. A talent alternative without a complete build keeps the page's default
gear, rotation and options. `tools/race_arena/snapshot.mjs` attaches the
saved setup and a stable build identity from `tools/build_arena/catalog.json`. A build's identity
survives a display-label change. Its input fingerprint records the content of its resolved request;
a matching name or identity cannot substitute for matching inputs. Display aliases are kept separate
from fingerprints.

The runner's manifest records every exported build, its legal races and the request needed to
reproduce it. Publication requires the complete set of fight lengths, target types and legal races,
matching input fingerprints and finite results at 50,000 iterations. Duplicate or missing cells,
invalid provenance and smoke results fail validation. The same raw matrix supplies both arenas.

The manual **Build Arenas** workflow exports once and creates the manifest without simulating it.
A separate eight-iteration benchmark measures each selected build at every fight length, summing elapsed
times across all legal races and target types. For each fight length, the workflow assigns the
largest estimated build costs first to the least-loaded of 16 shards. These measurements include
concurrent CPU contention and are estimates; eight iterations do not predict every long-run cost.
Benchmark output is kept outside the publication inputs.

The workflow publishes a complete matrix at 50,000 iterations per cell. Each shard records its
simulator revision and computation date. After validation, both scorers write into a staging
directory. Only a complete successful pair is copied into the checkout and committed together. A
branch that has moved from the simulated revision fails publication, and a rejected push is never
retried by merging unrelated work. A successful publication on `master` starts Pages deployment.
Runs on other branches only commit their results there.

A full run computes every build. A selective run takes a checked-in selection file that names every build, the previous verified engine and the compatibility evidence. Only selected builds are benchmarked and simulated. The publisher verifies the old requests and combines their original scores and sources with the fresh rows. [November 4 selection and checks](november-4-arena-reuse.md)

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

## Historical October 27 publication

The following source dates and compatibility checks describe the October 27 computations and their
later reuse. They do not validate reuse with the live-cache data and substituted presets.

The October 27 matrix contained 34 builds, 306 fight lists and 2,025 legal race matchups. The original
cloud engine was `932ffdbf86205ac9f388ecd161e89b77206e1989`. GitHub's
[successful run](https://github.com/laurencestokes/foreversim/actions/runs/40545126193) records its
48 simulation steps between October 27, 2026, 16:54:32 and 18:17:49 UTC. The 261 Troll matchups were
replaced using engine `66ef260625b26bdfb388af40abe2c1616500a38d` for the 15-second Berserking
duration; their exact execution timestamp was not recorded. The combined rankings were scored at
18:21:25 UTC on October 27. That scoring time must not be presented as the Troll computation time.

That publication's preset comparison accounted for five display-only Mage group labels and a pseudo-stat
array shrinking from 42 to 37 entries. Every entry from index 36 onward was zero in both exports;
the remaining resolved inputs and all setup snapshots matched exactly. The upstream change it checked
extracted existing crit calculations for the character sheet and preserved the combat expressions.
A before-and-after run at two iterations matched unrounded DPS and standard error exactly in all
306 lists and 2,025 race matchups. A longer check at 256 iterations matched both values exactly in
333 matchups across 42 lists: 14 builds at 180 seconds against all three target types and every legal
race. It covered both feral builds, three weaving hunter builds, six rogue builds, Enhancement and
both warrior builds. These bounded checks supported that reuse together with the input and code
comparison; they did not test every possible roll sequence. Those estimates retained their original
50,000 iterations and source dates.

To reproduce that check, build the runner at `66ef260625b26bdfb388af40abe2c1616500a38d` and at
that publication's verified merged revision with the same Go version, platform and `with_db` tag. Keep each engine's
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
