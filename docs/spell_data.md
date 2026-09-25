# Spell Data

The sim reads the client's numbers instead of hand-transcribed literals, and there are two ways to do
it. Which one a spell uses is a property of its class:

- **The store**, `sim/core/spelldata`: one generated file holding every spell the sim can reach, and
  resolvers that turn a row into a spell config, an aura, a dot, a talent's modifiers or a proc
  listener. The warrior reads it through the resolvers; rogue, warlock, mage, druid, priest, shaman
  and hunter read its rows by hand instead, one value at a time, with no resolver in between.
- **The family tables**, `sim/paladin/spell_data_auto_gen.go`: one generated table per spell family,
  read through `sim/common/shared`. Paladin reads them, and the section retires when paladin ports.

A new port uses the store. The family-table section is kept because paladin still depends on it, and
it retires when paladin ports.

The store:

- [The store](#the-store)
- [Building an ability](#building-an-ability)
- [Auras and dots](#auras-and-dots)
- [Talents and auras: ParseStatic and ParseEffects](#talents-and-auras-parsestatic-and-parseeffects)
- [Procs](#procs)
- [Overrides](#overrides)
- [Reading a row by hand](#reading-a-row-by-hand)

The family tables:

- [Using a rank](#using-a-rank)
- [The value shapes](#the-value-shapes)
- [Reaching a single effect](#reaching-a-single-effect)
- [A tick the client keeps on another spell](#a-tick-the-client-keeps-on-another-spell)
- [A number the client keeps on the judgement](#a-number-the-client-keeps-on-the-judgement)
- [A number the client keeps on the spell the rank fires](#a-number-the-client-keeps-on-the-spell-the-rank-fires)
- [Talents](#talents)
- [Worked examples](#worked-examples)
- [Attack power](#attack-power)
- [A row that is more than one spell](#a-row-that-is-more-than-one-spell)

Both:

- [Regenerating and checking](#regenerating-and-checking)
- [Porting a class to the store](#porting-a-class-to-the-store)
- [Traps](#traps)
- [Buffs and debuffs](#buffs-and-debuffs)

The accessor, spell config, aura and ladder snippets below are pinned by
`sim/core/spelldata/example_test.go`, which runs them against the committed store. The ones that need a
character - the parses, the proc triggers - are read off the warrior files they name.

## The store

`sim/core/spelldata/spells_auto_gen.go` holds every spell the sim can reach: the ones the class files
name, the ones items, enchants and set bonuses cast, and everything those reach in turn through a
trigger effect, an actionbar override or a tooltip reference. `sim/core/spelldata/snapshot_test.go`
pins what that comes to - 7048 rows carrying 9651 effects - so a regeneration that moves the universe
says so there.

Every field is the client's column in the client's units: a percentage is the integer 16, rage is on a
0-1000 bar, times are milliseconds. The conversion is in the accessors, so a row always matches what
the DBC says. `sim/core` must not import the package: the store imports core, and the import back
would be a cycle. That is why the raid buffs live in `sim/core/buffs` - see
[Buffs and debuffs](#buffs-and-debuffs).

### Finding a row

```go
frostbolt := spelldata.Find(116)      // Nil where the store does not carry the id
frostbolt = spelldata.MustFind(116)   // panics there instead
```

Every accessor answers on `Nil`, and `Nil` chains: `spelldata.Find(0).EffectN(1).Average(60)` is 0
rather than a crash, so a caller can ask about a spell this build does not have. `MustFind` is the
other bargain, and it is what a package-level variable takes - a regeneration that drops an id then
fails at startup, naming the id, instead of registering a spell with no numbers. `All()` and
`ByName(name)` exist for tests and debugging; a sim names its spells by id.

An id in hand-written code reads without grepping the table: `go run ./tools/spelldata 11574` prints the
row's header and one line per effect, each worded and then followed by the client's own columns. The
header names the school, the times and the cost, and also the stances by name, the weapon or slot the
row requires, the range, the target cap, the stacks and charges, the internal cooldown, the attributes
`attributes.go` exposes, the proc's rate and flags, the spells the tooltip references and the row's
labels. The title line names the ladder a class file reaches the id through - `warrior
spellData.Rend.Highest()` - scanned out of `sim/*/spell_data_auto_gen.go`, and says nothing for a class
the generator has not put on ladders yet. The same scan reads the other way too: `-family
warrior/Execute` prints a ladder's ranks with the call that reaches each, then its highest rank in full,
and `-expr 'spellData.Execute.Rank(3)' -package warrior` prints the row one ladder call names. A pick
followed by the store's own accessors is read too - `-expr
'spellData.Execute.Highest().EffectN(1).Average(core.CharacterLevel)'` answers the value, the doc comment
that accessor carries and the row with the effect it read marked `(read)`.

The worded line states each number in the unit the sim spends it in, which is also how it names the
accessor it was read with: a plain amount is `Average(60)` and a range is `Min`/`Max`, a percentage is
`Percent()`, rage is `Tenths()`, a time is `TimeValue()`, and a share of spell or attack
power is `Coeff()`/`APCoeff()`. A dummy line states the row's number and nothing about what it means,
which is the one shape that always needs the tooltip. An effect whose type or aura the printer has no
wording for says `unrecognised shape`, and the literal beneath it is then the whole answer.

A name instead of an id lists every row carrying it and then the highest rank, and
`-json` is the same answer for a tool to read. `-config 'spelldata.SpellConfig(&warrior.Unit,
executeRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))' -package warrior` prints the config the
resolver builds for that call, each field with the step that filled it - the row or the option - and
the row pick substituted through the package's declarations.

The same reading is an editor hover. `go run ./tools/spelldata -lsp` is a language server on stdio: a
hover on an id Go, APL JSON and TS state - `MustFind(11574)`, `SpellID: 11574`, `"spellId": 11574`,
`fromSpellId(23563)` - shows the row, with the header and the effects as tables; in Go a
`spellData.<Family>` token shows the ladder, a name bound to a ladder chain shows the rank or the value
it reads (substituted through the names it stands on, read from the open buffers first), each accessor
on a declaration line answers for itself, and the cursor on `spelldata.SpellConfig` shows the resolved
config. Each hover logs how it was resolved as a `window/logMessage`. `-hover sim/warrior/execute.go
12:40` prints the markdown a hover at that line and column shows, for scripts and tests. The server
compiles the store in, so a hover matches the checkout it was started in.

`.vscode/extensions/wowsims-spelldata` is its VS Code client, a workspace extension VS Code offers to
install when the repository is opened. It is built, not edited: the source is
`tools/vscode-spelldata`, and `make vscode-spelldata` bundles it with `vscode-languageclient` and writes
the manifest; `npm run check` there fails where the committed output is not a fresh build.
`tools/zed-spelldata` is the Zed extension (`zed: install dev extension`), and
`tools/spelldata/README.md` has the method table and the Neovim and Helix snippets.

### Reaching an effect

```go
damage := frostbolt.EffectN(2)                                 // the second effect the row carries
slow := frostbolt.Effect(dbcenums.A_MOD_DECREASE_SPEED, 0)     // the one effect with this aura and misc value
```

`EffectN` counts from **1, by position**. That is not the client's `EffectIndex`, which has gaps - 46
of the store's rows state an index that is not the position it sits at - and the row keeps the client's
own number in `Effect.Index`. A position the row does not have answers `NilEffect`.

`Effect(aura, misc)` names an effect by what it does rather than by where it sits. It panics when no
effect matches, and when two do: reading the first of several silently is the bug it exists to prevent,
and `EffectN` is the way past it. `FindEffect(type, aura, misc)` is the same question without the
panic, answering `NilEffect` instead.

The role readers each answer the first effect of their kind, or `NilEffect`: `DamageEffect()` (school
damage or any of the weapon-damage effects), `HealEffect()`, `EnergizeEffect()` and `PeriodicEffect()`
(the aura application whose aura ticks).

### Reading a value

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| `BaseValue()`             | the client's own number, unconverted                                  |
| `Percent()`               | over 100, because the client states a percentage as the integer 16    |
| `Tenths()`                | over 10, because rage sits on a 0-1000 bar                            |
| `TimeValue()`             | the number read as milliseconds, which is what a duration modifier is |
| `Period()`                | `EffectAuraPeriod`, the tick interval                                 |
| `Coeff()` / `APCoeff()`   | the spell power and attack power shares of the effect                 |
| `Average(level)`          | the amount at a caster level                                          |
| `Min(level)`/`Max(level)` | the ends of the roll                                                  |
| `Roll(sim, level)`        | the amount for one cast                                               |

`Average(level)` is the fold the family tables do: the base points truncated to a whole number, plus
`EffectRealPointsPerLevel` for each level between the spell's own `SpellLevel` and the caster's,
stopped at `MaxLevel` where the row states one, and floored. The arithmetic is float32 on purpose -
the client's per-level gain is a float32 widened into the database, and folding it in float64 moves
six rows off the tooltip.

`EffectVariance` is the spread the server rolls the amount over: `Min` and `Max` are the average times
`1 -/+ Variance/2`. `Roll` rolls it where the row states one and answers the average where it does
not, so a call site never has to ask whether this particular spell's amount is a range.

### Attributes, edges and class flags

`Spell.Attr` is the client's 17 attribute words. `HasAttr(word, bit)` reads one - a word past the end
answers unset rather than panicking - and `sim/core/spelldata/attributes.go` names the ones the sim
acts on: `IsPassive`, `IsChanneled`, `RefundsOnMiss` with `MissRefund()` (the 0.8 a
`RageCostOptions.Refund` takes), `PeriodicCanCrit`, `CanProcFromProcs`, `ClassSpellsOnly`,
`CannotCrit`, `IsAProc`, `SuppressesWeaponProcs` and `IsWeaponProcAura`; `IsBleed` reads
`SpellCategories.Mechanic` instead.

|                     |                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------------ |
| `effect.Trigger()`  | the spell `EffectTriggerSpell` names                                                                   |
| `spell.Triggered()` | every spell the row's effects fire, deduped, in effect order                                           |
| `spell.Drivers()`   | the spells whose effects fire this one, and the ones whose actionbar override replaces a spell with it |
| `spell.Refs()`      | the spells the tooltip names (`$12880d`, `$12966n`), in the order it names them                        |

An id the store does not carry is left out of those lists rather than answered as `Nil`. Lightning
Shield rank 1 shows why the two kinds of edge are separate: its aura effect triggers 26545, the
dispatcher every rank shares, while the rank's own damage spell 26364 is reachable only as a tooltip
reference.

`Spell.ClassFlags` is `SpellClassSet` with `SpellClassMask_0..3`: the family a spell files under and
its bit in it. `Effect.ClassFlags` is the same for `EffectSpellClassMask`, which is the set of spells a
modifier effect reaches, and `spell.AffectedBy(effect)` asks whether this spell is one of them.
`core.SpellConfig.ClassFlags` carries the row's own, which is what puts a registered spell inside the
talents that name it.

### A class file is ladders

A store-backed class keeps the same generated `spellData` global, with a `spelldata.Ladder` per family
in place of a table of rows (`sim/warrior/spell_data_auto_gen.go`):

```go
var spellData = generatedSpellData{
	AngerManagement: spelldata.Ranked(12296),
	Anticipation:    spelldata.Talent(12297, 5),
	BattleShout:     spelldata.Ranked(6673, 5242, 6192, 11549, 11550, 11551, 25289),
}
```

`Ranked` is one spell per rank, lowest first. `Talent` is a trait-tree talent, which the client states
as one spell whose per-rank numbers live in a curve: each rank is that spell with the curve's value on
the effects the curve covers, and an effect it has no row for keeps the spell's own base value. Both
`MustFind` every id, so a family that loses a spell fails at startup rather than registering nothing.

|                                          |                                                                                       |
| ---------------------------------------- | ------------------------------------------------------------------------------------- |
| `Rank(n)`                                | the spell at rank n. Rank 0 is untaken and answers `Nil`, so no `if rank > 0` guard   |
| `Highest()`                              | the top rank                                                                          |
| `ByID(id)`                               | the rank with this spell id; panics on one the ladder does not carry                  |
| `Len()`, `Each(fn)`                      | how many ranks, and each of them with its number                                      |
| `ValueAt(rank)`                          | the rank's only effect in the client's units; panics where the rank has more than one |
| `FractionAt`, `MultiplierAt`, `TenthsAt` | the same over 100, as `1 +` that, and over 10                                         |
| `EffectAt(n)`, `Effect(aura, misc)`      | one named effect across the ranks, carrying the same four readers                     |

`MultiplierAt` takes its sign from the data: a talent the client states as -2/-4/-6 gives 0.94 at rank
3 and nobody writes the minus.

`storeBackedClasses` in `tools/database/gen_spell_data.go` is how a class switches. Discovery, naming
and the `// Not generated:` header are the same either way, so the generated half of the move is one
line in that map and a regeneration; porting the call sites is the work.

## Building an ability

`spelldata.SpellConfig(unit, row, opts...)` fills the `core.SpellConfig` fields the row states, and
leaves everything else to the caller:

```go
var pummelRank = spellData.Pummel.ByID(6554)

config := spelldata.SpellConfig(&warrior.Unit, pummelRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) { /* ... */ }
warrior.RegisterSpell(config)
```

What the row fills: the `ActionID`, `Rank` (from "Rank 4", which is what flips `HasRanks` in the APL
UI), `SpellSchool`, `DefenseType`, `ClassFlags`, `MissileSpeed`, `MinRange`/`MaxRange`, the cast - cast
time, the GCD where the row sits in the global cooldown category, its own cooldown and the category one
it shares - and the cost out of the first bar it states, with rage divided off the 0-1000 bar and the
refund the Discount Power On Miss attribute states. `CastRequirement` gets the forms and caster auras
the client requires (see below), which is why Pummel needs no Berserker Stance condition. `Flags` gets
what the attributes and targets say:
`SpellFlagPassiveSpell`, `SpellFlagChanneled`, `SpellFlagSuppressWeaponProcs` and `SpellFlagHelpful`.

A bleed row also fills the damage and threat multipliers with 1.

What it does not: `ApplyEffects`, `ProcMask`, the multipliers on any other row, `ClassSpellMask`, `ExtraCastCondition`,
`Dot`, `RelatedSelfBuff`, the threat numbers the client does not carry - and `MaxTargets`, which has no
`SpellConfig` field at all, so a caller that caps an area effect reads `row.MaxTargets` itself. The same
goes for `RequiredAreas`, the area group a spell only works in: `row.AreaType()` names the kind of
terrain it stands for (Forest and Grassland, Mountainous, ...) and a caller gates on
`sim.Encounter.InArea(row.AreaType())`; a group that is a single zone reads as `AreaTypeUnknown`. A unit
is needed for the cooldown timers, so a config is built where the sim has a character rather than at
package init.

|               |                                                                                                                                                                        |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Melee(mask)` | the proc mask, `SpellFlagMeleeMetrics` and `SpellFlagAPL`, damage and threat multipliers of 1, and `IgnoreHaste`                                                       |
| `Magic(mask)` | the proc mask, `SpellFlagAPL`, the multipliers, and `BonusCoefficient` from the damage effect's spell power share - the heal's where the spell damages nothing         |
| `Proc()`      | `SpellFlagPassiveSpell` and `SpellFlagNoOnCastComplete`, clears `SpellFlagAPL`, and empties the cast - cast time, GCD, cooldowns - every cost and the cast requirement |
| `Flags(f)`    | ors in flags the client does not state                                                                                                                                 |
| `Tag(n)`      | splits one spell id into several actions                                                                                                                               |

The row is filled first and the options run on top in the order given, so an option sees what the row
put there.

### Where a spell can be cast

`core.CastRequirement` is what the client requires of the caster's form and auras, and core refuses a
cast, `CanCast` and `CanQueue` that break it ("wrong form", "missing caster aura", "excluded caster
aura"). It reads four client sources: `SpellShapeshift`'s mask and exclude mask (`StanceMask`,
`StanceExclude`), `SpellAuraRestrictions` (`CasterAura`, `ExcludeCasterAura` - Tiger's Fury states its
Cat Form requirement there, not in a mask), the not-shapeshifted and castable-in-caster-form attribute
bits, and `SpellShapeshiftForm`'s stance flag (`dbcenums.ShapeshiftForm.IsStance`, generated into
`sim/core/dbcenums/forms_auto_gen.go`).

For the caster's form `f`: an excluded form refuses; a form in the mask allows; in a shapeshift that is
not a stance (Cat, Bear) a spell with a mask or the not-shapeshifted bit is refused; with no form or in
a stance (the warrior's, Moonkin, Tree) a spell with a mask is refused unless it is castable in caster
form. A required caster aura must then be active on the caster, matched by `ActionID.SpellID`.

The class reports its form in `Unit.ShapeshiftForm` - the warrior's `setStance`, the druid's `setForm` -
and nothing else. A unit that sets `Unit.AutoUnshift` (the druid, to `ClearForm`) casts a spell its form
refuses but no form allows by leaving the form first, once, after every other check passed. That
holds in a stance-type form too: Healing Touch in Moonkin Form and Wrath in Tree of Life leave the form.

- A class that reads the store by hand takes `row.CastRequirement()`.
- A hand-built spell with no row states it by hand: `core.InForms(dbcenums.FORM_BATTLE_STANCE)`, with
  `.Excluding(...)` and `.OrCasterForm()`, or the struct literal for the attribute bits.
- A talent that swaps a spell on the action bar (`A_OVERRIDE_ACTIONBAR_SPELLS`) swaps its row too:
  Vanguard's Charge takes `chargeRank.OverriddenBy(spellData.Vanguard.Highest()).CastRequirement()`,
  whose row allows Defensive Stance.
- A form spell (subtext "Shapeshift") is a single-rank ladder like a warrior stance, so
  `spellData.CatForm` exists.

### The three shapes a registration takes

1. **The row, plus what no row states.** `sim/warrior/pummel.go` and `sim/warrior/slam.go`: the
   resolver, then the cast condition and `ApplyEffects`.
2. **The row, then an override of one of its fields.** `sim/warrior/hamstring.go` adds
   `ClassSpellMask` and pins the threat numbers with their review marker, because the client states no
   threat coefficient. Write the override after the call, on its own line, with the reason beside it.
3. **A hand-written `core.SpellConfig` reading the row's values.** The Sweeping Strikes hit spell in
   `sim/warrior/talents_arms.go` is a spell the sim registers and the store does not carry, so it is
   built by hand out of the parent's numbers.

### When not to take a default

- **A dot the client keeps on an `A_PERIODIC_DUMMY` stays hand-written.** `PeriodicEffect()` does not
  answer a dummy, and Deep Wounds' dummy states a spell power coefficient of 1 that `DotConfig` would
  put on a weapon-damage tick. Its spell config still resolves; only the dot is by hand
  (`sim/warrior/talents_arms.go`).
- **A sub-spell whose casts the sim reports does not take `Proc()`.** The option marks the spell
  passive and empties its cast, cooldown and cost, and the metrics aggregator counts no cast for a
  passive spell. Blood Craze's heal takes it on its own row, `BloodCrazeTriggered`
  (`sim/warrior/talents_fury.go`); Whirlwind's off-hand strike takes it on the Whirlwind row itself -
  `Melee(ProcMaskMeleeOHSpecial)`, `Proc()`, `Tag(2)` - which is what lets a spell with a cast time and
  a cooldown of its own lend its row to a sub-spell that has neither, writing only `ClassSpellMask`
  and `ApplyEffects` by hand (`sim/warrior/whirlwind.go`). Deep Wounds (`sim/warrior/talents_arms.go`)
  and Retaliation's counterattack (`sim/warrior/retaliation.go`) name the flags they need by hand
  instead - the bleed takes `SpellFlagNoOnCastComplete` with `SpellFlagIgnoreResists` and
  `SpellFlagProc`, the counterattack `SpellFlagMeleeMetrics`.
- **A sim-only copy of a spell carries the parent's `ClassFlags`.** Sweeping Strikes' hit spell and its
  normalized-attack copy (`sim/warrior/talents_arms.go`) have no row of their own, and without the
  parent's family mask the talents that name Sweeping Strikes would not reach them.

## Auras and dots

`spelldata.AuraConfig(row, opts...)` answers a `core.Aura` with the row's name as the label, its
`ActionID`, its duration - the client's -1 permanent aura becomes `core.NeverExpires` - and its stacks:
`CumulativeAura` where the row states one, `ProcCharges` otherwise, since the sim keeps stacks and
charges in one field. A row that states no duration at all resolves to 0, which core refuses to
activate: such an aura needs `Permanent()` or a duration of the caller's.

`Label(name)` renames one, which two auras of the same spell on one unit need. `Permanent()` makes the
aura last the iteration whatever the row says.

```go
aura := warrior.RegisterAura(spelldata.AuraConfig(shieldBlockRank))
spelldata.ParseEffects(&warrior.Character, aura, shieldBlockRank)
```

`spelldata.DotConfig(row, effect, opts...)` answers a `core.DotConfig`: that aura, `TickLength` from
the effect's period, `NumberOfTicks` from the row's duration over that period, `BonusCoefficient` from
the effect's spell power share, and an `OnTick` that deals the effect's own `Average` at the caster's
level, on the stats and multipliers in force at the tick - `OnSnapshot` is nil. It panics where the
effect states no period or the row no duration, rather than resolving a permanent aura's -1 into an
absurd number of ticks. A caller replaces `OnTick` for a heal or a value the client does not state.

`row.TickOutcome(dot)` picks the tick's outcome from the row: a tick that can crit where the client
marks Periodic Can Crit, on the magic hit table where the row's defense type is magic. A dot's `OnTick`
therefore never names an outcome itself.

## Talents and auras: ParseStatic and ParseEffects

A talent, a passive and a buff are all the same thing to the client: an aura whose `E_APPLY_AURA`
effects say what it does. The parse walks those effects and attaches each to the sim - a `SpellMod`
for the two modifier auras, a stat buff, a stat multiplier, a pseudo-stat multiplier or one of the
speed multipliers - so a port writes which row it is reading rather than what the row contains.

```go
spelldata.ParseStatic(&warrior.Character, spellData.Cruelty.Rank(warrior.Talents.Cruelty))
```

`ParseStatic` applies now and never comes off, which is what a talent or a class passive is.
`ParseEffects(character, aura, row, opts...)` makes the same attachments follow an aura: they turn on
when it is gained, off when it expires, and scale with its stacks where the row states
`CumulativeAura`.

A modifier effect is gated by the client's own class flags: `EffectSpellClassMask` decides which
registered spells the mod reaches, so a talent touches exactly the spells the client names it for. An
effect that names no spells at all, on a row of a class family, reaches that whole family - which is
what the client means by an unmasked class modifier.

**Call `ParseEffects` before the aura activates.** It attaches through `ApplyOnGain` and
`ApplyOnExpire`. An aura already up when the parse runs is caught up the way core's `Attach` helpers
catch one up, with one exception: the rows that need a `Simulation` to act - the cast, melee and attack
speed multipliers - stay off until the aura is applied again.

**An aura several ranks share takes a modifier once**, however many of those ranks the modifier's mask
names, because the mod is attached to the aura and not to each spell pointing at it.

|                   |                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------- |
| `Effects(1, 2)`   | only these effects, counted from 1 by position the way `EffectN` counts                                             |
| `SkipEffects(3)`  | everything but these                                                                                                |
| `Conditional(fn)` | a condition every attachment is gated on, read on gain and whenever the caller calls `Refresh(sim)`                 |
| `IgnoreStacks()`  | the values do not follow the aura's stacks, which is what a row stating charges rather than cumulative stacks means |

Defensive Stance is both shapes at once (`sim/warrior/stances.go`): the stance passive is parsed onto
the stance aura, and Defiance is parsed onto the same aura with a `Conditional` for the shield its
tooltip asks for and the row states nowhere, re-read through `Refresh` on an off-hand swap.

The call answers a `*Parsed`. `Applied` names each attachment - the effect, the sim kind it became and
the value in the sim's own units - and `Skipped` holds the aura effects the table has no row for, which
are exactly what the port still has to wire by hand. `SPELLDATA_REPORT=1` prints each of those on the
console with its position, the client's name for its aura, its misc value and its amount. Improved Slam
(`sim/warrior/talents_arms.go`) is the ordinary case: the cast time and global cooldown mods attach,
the five rank-swap effects behind them have no sim kind and are reported.

### What a value becomes

The table writes the sim's own units, and which unit that is differs per stat. The conversions:

|                                                             | the sim stores        | the parse writes                              |
| ----------------------------------------------------------- | --------------------- | --------------------------------------------- |
| Strength, Agility, Stamina, Intellect, Spirit               | flat points           | the value                                     |
| Health, Armor, the five resistances                         | flat points           | the value                                     |
| PhysicalDamage, SpellDamage and the per-school damage stats | flat power            | the value                                     |
| HealingPower, AttackPower, RangedAttackPower                | flat power            | the value                                     |
| MP5                                                         | mana per five seconds | the value                                     |
| PhysicalHitPercent, SpellHitPercent                         | percentage points     | the value                                     |
| PhysicalCritPercent, SpellCritPercent                       | percentage points     | the value                                     |
| **BlockPercent**                                            | **a fraction**        | **the value over 100**                        |
| DodgeRating                                                 | rating                | value x `DodgeRatingPerDodgePercent`          |
| ParryRating                                                 | rating                | value x `ParryRatingPerParryPercent`          |
| ExpertiseRating                                             | rating                | value x `ExpertisePerQuarterPercentReduction` |
| `PseudoStats.BonusHealingTaken`                             | flat healing          | the value                                     |
| every pseudo-stat multiplier and stat dependency            | a multiplier near 1   | `1 + value/100`                               |

Block is the one that reads differently from its name: core sums `stats.BlockPercent` with the rating
share already divided by 100, so the client's integer 5 is 0.05 there and a row handed over unconverted
is five hundred percent block. `TestParseStaticStatConventions` pins one row of each family against
what core reads back.

A talent that states the same percentage twice - once for the hit and once for the dot - has its dot
effect folded into the hit one, because a single `SpellMod_DamageDone_Flat` already reaches the ticks
and attaching both would double the bonus. The `Applied` entry says `folded-into`. A parse narrowed to
the dot effect alone attaches it as a dot mod instead.

### What the table does not take

Anything the table has no row for is skipped and reported rather than guessed at: the dummies the
client uses for "a script does this", the proc-driver auras (a proc is a `ProcTrigger`, not a
modifier), the crowd-control auras - stun, fear, root, silence - and `A_MOD_SKILL`. Those are named in
the report rather than numbered, so a port can see at a glance which ones are its own work.

Rows the table does know are skipped by the shape of the parse. A value the sim keeps as one number
per unit cannot be applied once per stack, and the static path has no aura to follow and no
`Simulation` to answer a later `Refresh` with:

|                                     | skips                                                                                                                       |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| a stacking aura (`MaxStack` > 0)    | the stat multipliers, the equipment scaling, the pseudo-stat multipliers, the speed multipliers and the cooldown multiplier |
| `ParseStatic`                       | the speed multipliers, which need the `Simulation` an aura's gain hands over                                                |
| `ParseStatic` with `Conditional`    | the stat multipliers and the equipment scaling as well                                                                      |
| an aura on a unit with no character | the equipment scaling, whose helper is a character's                                                                        |

`IgnoreStacks()` clears the first row of that table for a spell whose column counts charges rather
than cumulative stacks.

## Procs

The client's proc chance column is not always a chance, so the generator bakes what the tooltip says
about it into the row as a `ProcChanceSource`. The four shapes:

|                     |                                                                                                                                                                                                                               |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ProcChanceColumn`  | the tooltip renders `$h%`, so `SpellAuraOptions.ProcChance` is the roll                                                                                                                                                       |
| `ProcChanceEffectN` | the tooltip renders `$mN%`: the value of the effect at position `ProcChanceEffect` is the roll                                                                                                                                |
| `ProcChanceAlways`  | the column reads 100 or 101 and the trigger clause states no chance at all, so the aura fires whenever its own condition is met                                                                                               |
| `ProcChancePPM`     | the client states no chance anywhere, or a 100/101 column sits beside a trigger clause saying the effect only sometimes happens ("Chance to strike your ranged target"), so the rate has to come from an override into `RPPM` |

Reading `ProcChance` directly is the bug `ProcChanceSource` exists to prevent: 100 and 101 are the
client's "fires on its own condition" sentinel at least as often as they are a certainty.

```go
trigger := spelldata.ProcTrigger(&warrior.Character, spellData.Enrage.Rank(rank),
	func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
		warrior.EnrageAura.Activate(sim)
	})
trigger.Name = "Enrage - Trigger"
warrior.MakeProcTriggerAura(trigger)
```

`spelldata.ProcTrigger(character, row, handler, opts...)` fills the listener the row describes: the
name and action id, the `Callback`, `ProcMask`, `Outcome` and `RequireDamageDealt` the decoder reads
out of `ProcTypeMask`, the internal cooldown, `CanProcFromProcs` and `ClassSpellsOnly` from the
attributes, the class mask the proc effect names, and the rate the source above points at. A row that
is a weapon proc aura also excludes the hits of spells flagged Suppress Weapon Procs.

|                 |                                                                                                                                                             |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PPM(n)`        | a rate the client does not carry. It clears the chance and binds a manager to the trigger's proc mask, so an option that narrows the mask has to come first |
| `Chance(f)`     | a rate the caller states outright, manager included                                                                                                         |
| `ChanceFrom(e)` | the chance an effect states, for a `$mN` the generator could not resolve                                                                                    |

A trigger that ends with neither a chance nor a manager panics naming the spell. A listener that fires
on every qualifying hit is never what a row with no stated rate means, and a procs-per-minute rate with
no proc mask to measure it on panics for the same reason - on a weapon proc, which hits count is the
weapon's to say.

### The decoder and the tooltip hints

`core.DecodeProcTypeMask(row.ProcFlags, row.ProcHint)` turns the client's two proc words into a
`Callback`, a `ProcMask`, an `Outcome` and `RequireDamageDealt`, and names the bits it does not model
in `Unsupported`. `RequireDamageDealt` defaults to **true**, and a listener that fires on a dodge, a
parry, a miss or a block needs it false. Where the tooltip named that outcome the row carries
`ProcHintOutcomeTaken` and the decoder clears it already; a row whose wording yielded no hint - the
Battlegear of Wrath consume 23547 is one - has to be cleared by hand. Which outcome it is stays the
caller's either way, since no `ProcTypeMask` has a bit for any of them.

The mask states which hits reach the listener; it cannot state the condition around them, so the
generator reads that off the tooltip into `core.ProcHint`:

|                        |                                                                                                                                                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ProcHintCastTrigger`  | the tooltip names the cast itself, which turns a mask of the spell bits alone into `OnCastComplete` with no outcome                                     |
| `ProcHintCrit`         | the tooltip names a critical strike, which sets `Outcome = OutcomeCrit` and keeps the listener on the hit rather than the cast                          |
| `ProcHintHeals`        | the trigger clause names healing or an unrestricted spell, which is the evidence that a helpful-spell bit carries a real trigger rather than a leftover |
| `ProcHintPureHeal`     | the trigger is healing only, so the damage callbacks come off                                                                                           |
| `ProcHintNamedAbility` | the clause names one ability ("your Shock spells"), which no proc mask can state                                                                        |
| `ProcHintOutcomeTaken` | the clause names an outcome the mask has no bit for, which clears `RequireDamageDealt`                                                                  |

The last two are shapes the decode reads past rather than models, and they are what
`ProcTriggerUnsupported(character, row)` reports alongside the decoder's own unsupported bits. A
trigger is still built for all of them: a listener that hears fewer hits than the client's is
deliberately the narrower one.

A class mask naming **another class's** family is dropped rather than obeyed. An item every class can
wear states the filter of the one class it was written for, and on anyone else that mask matches
nothing and would silence the listener instead of narrowing it. A family the client states for no class
and the empty mask are evidence of nothing and are left alone.

### The shapes, as the warrior reads them

- **Enrage** (`sim/warrior/talents_fury.go`) is the column: its tooltip renders `$h%`, so the 30 in
  the column is a real roll on damage taken. Only the trigger's name is the caller's, because the
  driver and the buff share one.
- **Shield Specialization** (`sim/warrior/talents_protection.go`) is the effect ladder: the column
  reads 100 and the roll is effect 2. Its tooltip names an outcome, which the row carries as
  `ProcHintOutcomeTaken`, so the decoder already clears `RequireDamageDealt` - a block deals no damage.
  Which outcome it is stays the caller's: `Outcome = OutcomeBlock`.
- **Flurry** (`sim/warrior/talents_fury.go`) is the no-roll shape: the row states "always" and the
  crit is the condition. The buff row supplies the duration and the three charges through
  `AuraConfig`, the haste comes off the talent's ladder, and the mask stays the caller's because the
  row's reaches ranged and spell hits while the tooltip says melee. Where the row and the tooltip
  disagree on a number - 12966's flat 30 against the ladder's 25 at five points - the decision is
  written at the call site.
- **Lightning Shield** 324 is the same no-roll shape on a row nothing ports yet: 100 in the column,
  three charges, a 3.5 second internal cooldown, and its damage spell reachable only as a tooltip
  reference while its effect triggers the dispatcher every rank shares.

### Item and enchant procs

An item, enchant or set proc is registered from two spell ids and a shape:

```go
shared.NewSpellDataDamageProc(shared.SpellDataProc{TriggerSpellID: 7711, BuffSpellID: 7712},
	[]shared.ItemVariant{
		{ItemID: 17111, ItemName: "Blazefury Medallion"},
	})
```

`NewSpellDataProc` is the same for a proc whose buff is an aura. What the listener
hears, how often it fires and its internal cooldown come from the trigger's row; how long the buff
lasts and how it stacks come from the buff's; only the stats come from the item's effect entry, where
the sim's item level scaling lives. `IsWeaponProc` states the one shape no row describes: the game
casts a chance-on-hit effect off the weapon's hit without consulting a proc mask, and there the 100/101
sentinel is never a rate.

A buff the client states as a percentage of a stat (`A_MOD_PERCENT_STAT`, misc the stat index, -1 all
five) has no effect entry, which carries flat stats only, so the generator routes it as auras:
`NewSpellDataAuraProc` parses the buff's row, whose multiplier works through a dynamic stat dependency
and reports what it adds and removes to the temporary stats listeners. Insight 8216's 1299796 doubles
Spirit for 10 s. The aura proc registers the wearer's buff with the item swap, so an enchant's buff
drops with its weapon or shield, and, where the buff moves stats, with the stat-proc APL values. The
client has no item proc of this shape. `ParseStatic` reads the same aura as a multiplier applied once,
which reports nothing.

An enchant's procs are read one per slot of its `SpellItemEnchantment` row that casts a combat spell
(Effect 1) or hangs an equip aura off a hit (Effect 3), and at most one of them registers. A combat spell's chance, where the client states
one, is on the enchantment rather than on the spell - `EffectPointsMin`, Fiery Blaze's 15 - and the
store writes it onto the spell's row as its `ProcChance` column, noted beside the row. Every
enchantment casting one spell states the same chance for it; the generator stops on one that does
not. The chance is rolled on the hits of the enchanted weapon only
(`shared.statedWeaponProcChance`). An equip aura's own description is empty, so the store reads
the description of the spell granting the enchant in its place, for what no proc mask can state: the
named ability, outcome-taken, attack-dodged and attack-parried hints, and whether its 100 is a
sentinel ("often", "sometimes", "a chance to"), which makes it `ProcChancePPM`. The grant's cast,
crit and heal wording is left alone: which hits feed the aura is its own mask's to say.
A combat spell and an aura applying the same spell (Crusader) register once, as the combat spell.

`spelldata.ItemProcUnsupported(trigger, isWeaponProc)` is the single decision about whether the rows
say enough. The generator calls it when it writes the item files, the audit calls it, and a test pins
it, so a proc the generator emits is one the sim can build and a proc it comments out carries the
reason in the comment. `tools/database/unsupported_procs.txt` is the audit's census of every proc an
item, enchant or set can reach and what the rows refuse; `sim/common/forever/registered_effects.txt`
lists the item and enchant effects that do register. Neither is tracked: each is a local baseline its
test writes on the first run and then compares against, naming every line that moved.

## Overrides

`tools/database/overrides/spell_overrides.go` holds the numbers the store carries that the client
database does not state. One row per number:

```go
{16928, PPM, 1, "Annihilator: ProcChance 101 sentinel; TBC 1 PPM, unverified on Forever", "tbc-carryover"},
```

`Reason` and `Source` are not decoration: they are what the next reader weighs the number against, and
the generator refuses a row without a reason. `Source` names where it came from - `tbc-carryover`,
`tooltip`, `wcl:<report>/<fight>`, `issue #N`.

Every field names a rule the generator checks before it writes the value, so an override outlives its
reason no longer than the next regeneration:

|                                   |                                                         | stale when                               |
| --------------------------------- | ------------------------------------------------------- | ---------------------------------------- |
| `PPM`                             | procs per minute, onto `Spell.RPPM`                     | the row states a `SpellProcsPerMinuteID` |
| `FlatThreat`                      | threat the tooltip states without a number              | the spell gains a threat effect          |
| `APCoefDirect` / `APCoefPeriodic` | an attack power coefficient the client keeps in script  | the client states one on that effect     |
| `ProcChancePct`                   | a whole percentage onto `Spell.ProcChance`              | the tooltip states a chance of its own   |
| `DurationMs`                      | an aura duration                                        | the client states a `SpellDuration`      |
| `Hint`                            | the `ProcHint` bits the tooltip's wording did not yield | -                                        |

Two more refusals have nothing to do with the field: an override naming a spell the store does not
carry, and two overrides of the same field on one spell. Every one that is applied leaves an
`// override: <field> <value> -- <reason>` comment on the row it wrote to, so reading the generated
store says which numbers are not the client's.

An item or enchant proc refused as `states no rate` loses that refusal once a `PPM` override sits on
the spell it is routed through - the `TriggerSpellID` of its commented-out registration, the id on its
`// trigger N` line - and registers where it was the only one:

- A combat enchant (Effect 1) or a chance-on-hit item effect: the combat spell itself - Fiery Weapon's
  13897, Unholy Weapon's 20006 - not the spell that grants the enchant. The rate is measured on the
  hits of the weapon carrying it (`NewDynamicLegacyProcForEnchant(id, ppm, 0)`, or `...ForWeapon` for an
  item).
- An equip aura (Effect 3): the aura carrying the proc trigger - Revelation's 1248806 - not the spell it
  triggers (1248808) nor the grant (1248805). The rate is measured on the aura's own proc mask, so an
  aura whose flags decode to no mask stays refused as `no proc mask to measure its rate on`.

A procs-per-minute enchant proc rolls on weapon hits only: its mask keeps its melee and ranged bits,
and a spell or heal hit never procs it. `dpmForMask` strips the rest for every enchant, and
`spelldata.EnchantAuraUnsupported` refuses a rate on an equip aura that hears only spells as
`a procs-per-minute rate hears no weapon hits in this mask`, so it stays listed rather than
registering to never fire. An enchant the game does let spells proc is an exception to state there,
by enchant; there is none today. A stated chance (Fiery Blaze 36's 15%, Insight 8216's 35%) is not a
rate and hears what its row says.

Revelation 8217 stays listed. Its rate is scripted and the client does not state it: trigger 1248806's
`ProcChance` 100 is the sentinel beside the tooltip's "a chance", and its effect entry resolves no stats
from 1248808.

A rate follows the weapon the enchant sits on. A weapon enchant, ranged ones included
(`NewDynamicLegacyProcForEnchantWithMask`), rolls only on the hand carrying it, at that weapon's speed;
an item swap moves it with the weapon. An enchant on no weapon - armor, a cloak, a shield or a
held-in-off-hand item - is priced off the main hand (`NewLegacyPPMManager`, which prices every mask but
the off hand's and the ranged one's at the main hand's speed).

Where a combat spell and an aura apply the same spell (Crusader), the slot whose row states a rate is
the one kept, so the override goes on the combat spell to keep it the combat spell. After adding a row,
run `gen_spelldata` and then `gen_db`, which classifies the procs out of the store compiled into it.
`TestEnchantProcRoutingTakesAPPMOverride` in `tools/database` and `TestSpellDataProcTakesAPPMOverride`
in `sim/common/shared` pin both halves on those three rows with a rate that exists only in the test.
`TestEnchantAuraPPMFollowsItsWeapon` beside the latter pins the routing by slot, and
`TestEnchantPPMFollowsTheEnchantedWeapon` in `sim/core` the move across an item swap.

`overrides.AreaBonuses` is the second table in the same file, for an effect whose tooltip says it is
doubled in some kind of area while the client states no companion row for it ("This effect is doubled
in Volcanic areas" on Molten Fury). A row names the AreaGroup ids any of which counts, the factor on
the effect's amounts and the factor on its duration, onto `Spell.AreaBonusGroups`, `AreaMultiplier`
and `AreaDurationMultiplier`; `row.AreaBonus(&character.Env.Encounter)` reads them against the
encounter's area set, and is stale once the row states a `RequiredAreasID` of its own. The on-use stat
actives `shared.NewSimpleStatActive` registers read it; a proc chance the tooltip says is doubled has
no reader yet.

`tools/database/overrides/extra_spells.go` is the other hand-kept list: spells the generator
force-includes although no class, item, enchant or set reaches them. It is empty today. Each entry
states a reason and a source like an override, and the generator refuses one without a reason. The
extras are added while the store renders, so adding one without regenerating fails the check rather
than shipping a store that does not match its source.

## Reading a row by hand

Rogue, warlock, mage, druid, priest, shaman and hunter read the store the same way paladin reads its
family tables: a class file names the ladder it needs, picks a rank, and reads every field the
registration wants straight into a plain expression, with no resolver in between. **A row is a
source of numbers only** in these seven classes' files - no `spelldata.SpellConfig`, `AuraConfig`,
`DotConfig`, `ProcTrigger`, `ParseEffects` or `ParseStatic`. `core.SpellConfig` and `core.DotConfig`
are built by hand, field by field, the same shapes [Building an ability](#building-an-ability) and
[Auras and dots](#auras-and-dots) describe for the resolvers, just filled without one.

The generated file is still ladders (see [A class file is ladders](#a-class-file-is-ladders)):
`spellData.ShadowWordPain` is a `spelldata.Ladder`, `.Highest()`/`.ByID(id)`/`.Rank(n)` answer a
`*spelldata.Spell`, and `.Each(fn)` walks every rank in declaration order, the family tables'
`RegisterAll` under a new name:

```go
MindBlastRankMap.Each(func(_ int32, rank *spelldata.Spell) {
	priest.registerMindBlastSpell(rank, mindblastCDTimer)
})
```

What a registration reads off the row, since there is no resolver to read it for the caller:

|                                                    |                                                                                                                                                                                         |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `rank.ID`, `rank.RankNumber()`                     | the `ActionID` and the `Rank` field a `core.SpellConfig` wants; `RankNumber()` is `NameSubtext_lang`'s "Rank N", 0 where the client states none                                         |
| `rank.Cost()`                                      | the cost off the first power the row states, in the sim's units - rage already divided by ten; it answers a `float64`, so an `int32` field like `ManaCostOptions.FlatCost` needs a cast |
| `rank.GCD()`, `rank.CastTime()`                    | the global cooldown and the cast time                                                                                                                                                   |
| `max(rank.Cooldown(), rank.CategoryCooldown())`    | the row's own cooldown, or the one it shares with a category                                                                                                                            |
| `rank.Duration()`                                  | an aura or dot's length; the client's -1 becomes `core.NeverExpires`                                                                                                                    |
| `rank.SpellSchool()`, `rank.DefenseTypeCore()`     | already in core's enums - the accessor converts the client's byte                                                                                                                       |
| `float64(rank.MaxRange)`, `float64(rank.MinRange)` | the plain fields, in yards                                                                                                                                                              |

Straight off `sim/shaman/shocks.go`:

```go
func (shaman *Shaman) newShockSpellConfig(rank *spelldata.Spell, spellSchool core.SpellSchool, shockTimer *core.Timer) core.SpellConfig {
	return core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.ID},
		SpellSchool: spellSchool,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       SpellFlagShamanSpell | SpellFlagShock | core.SpellFlagAPL | SpellFlagInstant,
		MaxRange:    float64(rank.MaxRange),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: rank.GCD()},
			CD: core.Cooldown{
				Timer:    shockTimer,
				Duration: max(rank.Cooldown(), rank.CategoryCooldown()),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: rank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,
	}
}
```

**Every amount is `Average(core.CharacterLevel)`.** The family tables the seven classes replaced never
rolled - every value is `BasePoints`-derived and read once per rank - so a value read as `(low, high)`
there becomes one `Average` call, read for both ends, here too. A row can still carry a
spread (Frostbolt's `Variance` is 0.105) and `Roll(sim, level)`/`Min`/`Max` read it where a
store-backed class asks for one; `Average` is what keeps a ported number equal to the one it
replaces, not an absence of spread in the data.

**Name the effect by role where one applies; by position where the row hides it behind another
effect.** `DamageEffect()`, `HealEffect()`, `EnergizeEffect()` and `PeriodicEffect()` answer the first
effect of their kind, which is right wherever a spell has only one. Hellfire's tick is not:
`PeriodicEffect()` would find the `A_PERIODIC_TRIGGER_SPELL` at position 1, the trigger that fires
the Hellfire Effect spell each tick, rather than the damage tick itself at position 2:

```go
var hellfireRank = spellData.Hellfire.Highest()

// The tick sits at effect position 2: position 1 is the A_PERIODIC_TRIGGER_SPELL that fires the
// Hellfire Effect spell each tick, not the damage tick.
var hellfireTick = hellfireRank.EffectN(2)
var hellFireCoeff = hellfireTick.Coeff()
```

**A tick the description keeps on another spell is read off that spell, not the row that names it.**
Where the family has its own `XTriggered` ladder for the spell the trigger fires, that ladder is the
edge; failing that, `row.Triggered()[0]` (an actual `TriggerID` edge) or `row.Refs()[0]` (a tooltip
reference) reach the same spell, and a bare `spelldata.MustFind(id)` is the last resort, with a
comment naming why no other edge is unique. Hurricane's periodic damage sits on the spell its aura
triggers each tick, which the family's own `HurricaneTriggered` ladder names; the tick length still
comes off Hurricane's own row - the periodic dummy that carries no damage of its own:

```go
tickLength := hurricaneRank.Effect(dbcenums.A_PERIODIC_DUMMY, 0).Period()

// Hurricane's periodic damage is the spell HurricaneTriggered casts each tick.
hurricaneTickSpell := spellData.HurricaneTriggered.Highest()
hurricaneTick := hurricaneTickSpell.DamageEffect()
```

**`EffectAt(n)` counts from 1 by position, like `EffectN` - not the client's `EffectIndex`.**
`EffectAt(n+1)` lines up with client index `n` only where the row's indices run contiguously from 0;
where they do not, `Effect(aura, misc)` names the effect instead. Rogue's `PuncturingWounds` talent
needs it: its two crit modifiers share an aura and misc, so `Effect(aura, misc)` cannot tell them
apart, and sit at positions 1 and 3 around the proc trigger at position 2 (client index 1);
`EffectAt(3)` is the one Mutilate reads:

```go
// The proc trigger sits between the two crit modifiers, at effect position 2; the Mutilate
// crit bonus is the second of the two, at position 3, and they share an aura and misc so have
// to be indexed rather than named.
FloatValue: spellData.PuncturingWounds.EffectAt(3).ValueAt(rogue.Talents.PuncturingWounds),
```

**A `Ranked` family's rank can carry a per-level gain a `Ladder`'s per-rank readers do not add in.**
`EffectAt(n)` and `Effect(aura, misc)` answer base points across the ranks, which is right for a
`Talent`, whose curve states base points outright. A `Ranked` family - one spell per rank - can still
carry `EffectRealPointsPerLevel` on that effect, so `ValueAt`/`FractionAt`/`MultiplierAt`/`TenthsAt`
read low wherever the client scales it past the rank's own `SpellLevel`. Read the rank's own effect
through `Average` instead of the ladder's per-rank reader:

```go
X.Rank(rank).EffectN(1).Average(core.CharacterLevel)   // not X.EffectAt(1).ValueAt(rank)
```

**A dot ticks on current stats, so there is no `OnSnapshot`.** `OnTick` reads the effect's own
`Average` straight into `CalcAndDealPeriodicDamage` (or `CalcAndDealPeriodicHealing`) every time it
fires, on whatever stats and multipliers are in force then. An input the cast fixes - combo points
about to be spent, a stack about to be consumed - is read where the cast reads it and kept in a
variable the `OnTick` closure captures, never re-read at the tick. Rip's combo points are read in
`ApplyEffects`, before `SpendComboPoints` runs them to zero - not in `OnTick` after, where the count
would already be gone:

```go
var cp int32

// ...
	ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
		if result.Landed() {
			cp = druid.ComboPoints()
			spell.Dot(target).Apply(sim)
			druid.SpendComboPoints(sim, spell.ComboPointMetrics())
		}
		spell.DealOutcome(sim, result)
	},
	OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		ap := dot.Spell.MeleeAttackPower(target)
		var tickDamage float64
		switch {
		case cp <= 3:
			tickDamage = 990 + 0.18*ap
		case cp == 4:
			tickDamage = 1272 + 0.24*ap
		default: // 5
			tickDamage = 1554 + 0.24*ap
		}
		dot.Spell.CalcAndDealPeriodicDamage(sim, target, tickDamage/6, dot.OutcomeTick)
	},
```

**Hand numbers stay hand numbers - Go literals with the same review comment a family-table wrapper
carries, not a `WithSpellDataPPM`/`WithSpellDataFlatThreat`/`WithSpellDataAPCoef` call.** A
hand-supplied threat number, PPM or coefficient keeps the same marker a resolver-built config
carries. `sim/warrior/hamstring.go` writes it as an assignment on the config the resolver already
built:

```go
// TODO: Manual review needed -- the client states no threat coefficient; 1 until measured in game.
config.ThreatMultiplier = 1
```

A hand-built `core.SpellConfig`, with no resolver to build it first, carries the identical comment on
the identical field, as a struct-literal line instead:

```go
// TODO: Manual review needed -- the client states no threat coefficient; 1 until measured in game.
ThreatMultiplier: 1,
```

### Worked examples

#### Shadow Word: Pain's tick

`sim/priest/shadow_word_pain.go`:

```go
tick := rank.PeriodicEffect()
tickLength := tick.Period()

priest.RegisterSpell(core.SpellConfig{
	ActionID:       core.ActionID{SpellID: rank.ID},
	SpellSchool:    core.SpellSchoolShadow,
	DefenseType:    core.DefenseTypeMagic,
	ProcMask:       core.ProcMaskSpellDamage,
	Flags:          core.SpellFlagAPL,
	ClassSpellMask: PriestSpellShadowWordPain,
	Rank:           rank.RankNumber(),
	MaxRange:       float64(rank.MaxRange),

	ManaCost: core.ManaCostOptions{
		FlatCost: int32(rank.Cost()),
	},

	Cast: core.CastConfig{
		DefaultCast: core.Cast{GCD: rank.GCD()},
	},

	Dot: core.DotConfig{
		Aura: core.Aura{
			Label: fmt.Sprintf("ShadowWordPain-%d", rank.RankNumber()),
		},
		NumberOfTicks:       int32(rank.Duration() / tickLength),
		TickLength:          tickLength,
		AffectedByCastSpeed: false,
		BonusCoefficient:    tick.Coeff(),

		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.Spell.CalcAndDealPeriodicDamage(sim, target, tick.Average(core.CharacterLevel), dot.OutcomeTick)
		},
	},

	ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
		return spell.CalcPeriodicDamage(sim, target, tick.Average(core.CharacterLevel), spell.OutcomeExpectedMagicHit)
	},
})
```

No `OnSnapshot`, no `useSnapshot` branch: the tick is `tick.Average(core.CharacterLevel)` wherever it
is asked for, cast or projected.

#### Consecration's borrowed tick

The same shape as [A tick the client keeps on another spell](#a-tick-the-client-keeps-on-another-spell),
read off the store instead of a family table. Paladin has not ported, so there is no
`spellData.Consecration` ladder to reach it through yet; the row itself is already in the store -
every class family seeds it, ported or not - and reads by id in the meantime. Consecration rank 5's
tooltip names one spell for both ticks - `Refs()[0]` reaches it - and the two sit on its first two
effects by position, since both are school damage and `DamageEffect()` cannot tell them apart: the
base tick on effect 1, the bonus its first four targets take, with its own spell power share, on
effect 2. The periodic dummy on Consecration's own row states the target count, not a tick:

```go
consecrationRank := spelldata.MustFind(20924) // Consecration, rank 5
tickSpell := consecrationRank.Refs()[0]

tick := tickSpell.EffectN(1)
bonus := tickSpell.EffectN(2)
bonusTargets := int(consecrationRank.Effect(dbcenums.A_PERIODIC_DUMMY, 0).Average(core.CharacterLevel))

dealTick := func(sim *core.Simulation, dot *core.Dot) {
	for i, target := range sim.Encounter.ActiveTargetUnits {
		damage := tick.Average(core.CharacterLevel)
		if i < bonusTargets {
			damage += bonus.Average(core.CharacterLevel) + bonus.Coeff()*dot.Spell.BonusDamage(dot.Spell.Unit.AttackTables[target.UnitIndex])
		}
		dot.Spell.CalcAndDealPeriodicDamage(sim, target, damage, dot.OutcomeTickMagicHit)
	}
}
```

#### A heal and a mana restore

```go
heal := rank.HealEffect().Average(core.CharacterLevel)      // both ends: the tables carry no spread
mana := rank.EnergizeEffect().Average(core.CharacterLevel)  // 0 on a rank with no Energize effect at all
```

#### Registering several ranks

Forever's Starfire tops out at rank 7, so the spec's own subset ladder names ranks 6 and 7 by their
ids rather than a rank the table does not carry, `sim/druid/starfire.go`:

```go
var StarfireRankMap = spelldata.Ranked(spellData.Starfire.Rank(6).ID, spellData.Starfire.Rank(7).ID)
```

#### An attack power coefficient

Nothing in the client states an attack power coefficient per combo point, so Rupture's sits as a
literal table indexed by combo points, right beside the number it scales, `sim/rogue/rupture.go`:

```go
func (rogue *Rogue) ruptureDamage(target *core.Unit, comboPoints int32, baseDamage float64, damagePerComboPoint float64) float64 {
	return baseDamage +
		damagePerComboPoint*float64(comboPoints) +
		[]float64{0, 0.01, 0.02, 0.03, 0.03, 0.03}[comboPoints]*rogue.Rupture.MeleeAttackPower(target)
}
```

## The family tables

The sections from here to [A row that is more than one spell](#a-row-that-is-more-than-one-spell)
describe the generated tables paladin reads: one table per spell family in
`sim/paladin/spell_data_auto_gen.go`, with the accessors in `sim/common/shared`. A store-backed class
has none of this - its generated file is ladders into the store - so a new port reads
[The store](#the-store) instead, either through the resolvers or, per
[Reading a row by hand](#reading-a-row-by-hand), directly.

## Using a rank

Each class package has exactly one generated global, `spellData`, with a field per spell family:

```go
spellData.Exorcism          // the whole ladder, ranks 1-7
spellData.Fireball          // ranks 1-14
```

Pick the rank your spell registers **by spell ID**:

```go
var exorcismRanks = spellData.Exorcism.BySpellID(27138)
```

That is the identity the sim already uses everywhere - `ActionID`, saved APLs and the icon database all
key on the spell ID - so it cannot drift onto a different rank, and a regeneration that drops the ID
fails loudly instead of quietly substituting another.

The other accessors:

|                    |                                                                                               |
| ------------------ | --------------------------------------------------------------------------------------------- |
| `BySpellID(27138)` | the rank registered under that spell ID. Prefer this.                                         |
| `ByRank(6)`        | the rank numbered 6                                                                           |
| `Ranks(6, 8)`      | a subset, **in the order given**, which is registration order                                 |
| `HighestRank()`    | the highest rank _in the data_, which is not always one the game grants - see [Traps](#traps) |
| `RegisterAll(f)`   | calls `f` once per rank, in declaration order                                                 |

A single-rank ability (Whirlwind, Shield Wall, Taunt) is a one-row table of its own, rank 1, with the
same columns; nothing about it is hand-typed. Beside cost, cast time, cooldown and range a row carries
`Duration` (the aura or effect it leaves), `ProcCharges` (how many times that aura acts) and
`MaxTargets` (an area effect's cap), each zero where the client states none, and `RefundsOnMiss`, the
Discount Power On Miss attribute; `row.MissRefund()` turns it into the 0.8 a `RageCostOptions.Refund`
takes. `PeriodicCanCrit` is the Periodic Can Crit attribute; `shared.PeriodicTickOutcome(row, dot)` picks the tick
outcome it and the row's defense type call for, so a dot's `OnTick` never names one itself. `PowerCostPct`
is a cost stated as a share of the pool: Bloodrage reads 20, of health; Arcane Blast 15, of mana.

A family whose ranks trigger another spell, or whose tooltip reads a number off one, has a second
table beside it: `spellData.EnrageTriggered` holds the buff 12880 that Enrage's `$12880d` names,
`FlurryTriggered` the 12966 with its 3 charges, `LastStandTriggered` the 12976 with the 30% and 20 s,
`InterceptTriggered` the stun of each rank, `OffensiveStateTriggered` the 5 s Overpower window 1282733
that the Defense-line passive Offensive State (DND) fires on a melee hit, `DefensiveStateTriggered` the
Revenge one. A spell only a server-side handler casts, with no edge,
token or skill-line row naming it, is linked by hand in `tools/database/overrides.HandTriggers`:
`RetaliationTriggered` holds the counterattack 20240 that Retaliation's dummy aura fires. Where every
rank triggers the same spell the table has one row, rank 1; where each rank triggers its own, the row
takes the rank's number.

## The value shapes

A rank's value is discriminated by shape, so a variant only carries fields that mean something for it:

```go
shared.SpellDataFlat     {Value, Coef, APCoef}                          // a mana restore, a talent's number
shared.SpellDataRange    {Min, Max, Coef, APCoef}                       // damage or healing the client rolls
shared.SpellDataPeriodic {Tick, TickMax, TickLength, NumberOfTicks, Coef, APCoef, SpellID} // a tick and its schedule
```

They sit on the roles a rank can carry, any of which may be nil:

```go
rank.Direct             // Effect = SCHOOL_DAMAGE
rank.Heal               // Effect = HEAL
rank.Periodic           // a periodic aura
rank.Energize           // Effect = ENERGIZE, e.g. Lay on Hands' mana restore
rank.SecondaryPeriodic  // a second tick the description names - Consecration alone, see below
```

Asking what a value is worth on this cast is a single call, because the question means something for
all three shapes - a range rolls between its ends, a flat value and a tick are already the answer. The
method is named for what it produces rather than for how, so a static ability does not read as if it
rolled:

```go
baseDamage := rank.Direct.Damage(sim)     // instead of CalcAndRollDamageRange(sim, min, max)
tickDamage := rank.Periodic.Damage(sim)   // a tick is the answer unless the client rolls it
```

The coefficients are methods, named for the `core.SpellConfig` fields they feed:

```go
BonusCoefficient: rank.Direct.BonusCoefficient(),     // spell power
                  rank.Periodic.BonusCoefficient(),   // same on a tick
                  rank.Direct.APBonusCoefficient(),   // attack power
```

`Range()` gives both ends of a value at once:

```go
low, high := rank.Direct.Range()   // equal for a flat value or a tick
```

The methods assume the role is there. Where it may not be - `Energize` is nil on Lay on Hands rank 1 -
use the package helpers instead, which read a nil value as zero:

```go
shared.SpellDataMin(rank.Energize)     // 0 rather than a panic
shared.SpellDataMax(rank.Direct)
shared.SpellDataCoef(rank.Periodic)
shared.SpellDataAPCoef(rank.Direct)
```

Tick length and count live only on the periodic shape, so ask for that shape:

```go
p := rank.Periodic.AsPeriodic()
p.TickLength     // time.Duration, feeds core.DotConfig.TickLength
p.NumberOfTicks  // duration over the tick length, feeds core.DotConfig.NumberOfTicks
```

A rank also carries what the client knows about casting it:

|                               |                                                                                                                                            |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `Cost`                        | in the units the sim uses - see the rage trap below                                                                                        |
| `CastTime`, `GCD`, `Cooldown` | zero for a channel, whose duration carries it                                                                                              |
| `MinRange`, `MaxRange`        | `core` gates the cast on both; zero means ungated. `MinRange` is the dead zone on a charge, and is nonzero on only 212 spells in the build |
| `MissileSpeed`                | yards per second, which `core` turns into the delay before the damage lands. Zero is an instant hit                                        |
| `SpellSchool`, `DefenseType`  | as `core` names them. The client's school bits are `core`'s bits, so this is the number the DBC states rather than a translation of it     |

`MissileSpeed` is the one to be careful with: giving a spell a speed it did not have delays its damage
and moves goldens, so check the sim is not already modelling it elsewhere. Arcane Missiles is the case
to know - the channel carries no speed because the missile spell does, and that one is not a ranked row.

## Reaching a single effect

The role fields describe one effect each, which is all a castable spell needs. A talent routinely
carries two or three that the sim reads separately, and only one of them can be `Direct`:

```go
irf := spellData.ImprovedRighteousFury.ByRank(3)

irf.Effect(shared.A_ADD_PCT_MODIFIER, 8).Value    //  50  threat bonus
irf.Effect(shared.A_ADD_FLAT_MODIFIER, 12).Value  //  -6  damage taken
irf.Effects[1].Value                              //  -6  the same effect, by index
```

The aura and effect names are generated into `sim/common/shared/spell_data_enums_auto_gen.go`, mirrored
from `tools/database/dbc/enums.go` and holding only the values the tables use, so the two cannot drift.
`Misc` stays a plain int, because what it selects depends on the aura - see
[The Misc value](#the-misc-value).

Name the effect by aura rather than reading `Direct` whenever a spell has more than one. Which effect
lands in `Direct` is the generator's choice, not a promise, so a caller that depends on it breaks
silently the day the ordering changes.

`Effect` panics when nothing matches, and also when **two** effects match: 186 ranked spells carry a
duplicate aura/misc pair, and returning the first is how a caller ends up reading the wrong half of a
talent. Index into `Effects` where the pair cannot tell them apart.

**The value is in the client's units.** A percentage is an integer here - Improved Righteous Fury's
threat bonus reads `16`, not `0.16` - so the `/100` stays at the call site. It is deliberately not
folded into the generator the way the rage `/10` is: whether a value is a percentage depends on the
aura, so a blanket rule would be wrong for some rows and invisible when it was.

`ChainAmplitude` is the client's EffectChainAmplitude where it is not 1, which the client uses for more
than chain falloff: Execute's dummy carries 1.5, and its tooltip multiplies that by 10 for the damage
each extra rage adds. Zero on the effects that state 1.

## A tick the client keeps on another spell

Forever moves a ground effect's damage onto a spell of its own. Consecration rank 5 states a dummy,
the area trigger it creates, and a periodic dummy - no damage - and its tooltip reads
`${$1280349m1*8}`: the tick sits on 1280349, a spell that shares the name and rank subtext and that
the client links from nowhere but that description. Blizzard, Flamestrike, Rain of Fire, Hurricane
and Volley are shaped the same way, each rank naming its own sub-spell.

The generator follows the reference. When a rank carries a periodic dummy and no periodic damage of
its own, it reads the description for `$<spellID>m<n>` and `$<spellID>s<n>`, takes effect `n` of a
spell with the rank's name, and gives it the dummy's period, so it lands in `Periodic` with the tick
schedule the rank states. The tick says where it came from:

```go
p := spellData.Consecration.BySpellID(20924).Periodic.AsPeriodic()
p.SpellID   // 1280349; zero on a tick the rank's own effect states
```

Consecration's description names two, and the second lands in `SecondaryPeriodic`: the extra damage
its first few targets take, and the only part of the spell the client gives a spell power coefficient.
A third would fail the generator rather than be dropped.

**The periodic dummy's points are not a tick.** Consecration's reads 4, which is how many targets
take the second tick, and it stays where the client put it:

```go
bonusTargets := int(rank.Effect(shared.A_PERIODIC_DUMMY, 0).Value)   // 4
```

Before the generator followed the description, that 4 was filed as the tick and the AoE families
above had no tick at all.

## A number the client keeps on the judgement

Seal of Righteousness states no value. Rank 8 is an aura dummy at 1880, the damage each hit adds,
and a second aura dummy whose points are 20286, its judgement. The tooltip renders the hit off the
judgement - `$/87;20286s3 to $/25;20286s3` - and effect 3 of 20286 is a dummy the judgement does
nothing with itself: the same 1880, with the coefficient the seal's own copy lacks. The seal carries
0.1 on ranks 1-7 and nothing on rank 8; the judgement's dummy carries 0.058 on rank 1 rising to 0.2
from rank 4.

The generator follows that reference too. When a rank states no value and its description names an
effect of a spell one of its own dummies points at, a dummy at that index is the rank's number and
lands in `Direct`:

```go
d := spellData.SealOfRighteousness.BySpellID(20293).Direct.AsFlat()
d.Value   // 1880, which the seal's own effect 0 also says
d.Coef    // 0.2, which only the judgement's dummy states
```

The `/87` and `/25` are the tooltip's rendering and are not applied: the value is kept whole and the
proc's formula decides what a swing does with it. A named effect that is not a dummy is the pointed
spell's own - Seal of Fury and Seal of the Crusader both name their judgement's damage or aura - and
stays with it. A flat value does not say where it came from the way a tick does, so a Seal of
Righteousness row whose `Coef` is the seal's own 0.1, or 0, is one where the reference did not resolve.

## A number the client keeps on the spell the rank fires

Seal of Fury keeps its per-hit damage on the proc its aura dummy triggers. Rank 7's effect 0 is a
dummy at 1607 gaining 42 a level - Seal of Righteousness' number, left from when the seal was a copy
of it - whose `EffectTriggerSpell` is 20418, and the tooltip renders the hit off that spell:
`$20418s1 Holy damage`. 20418's effect 0 is school damage at 35 with a 0.1 coefficient; the seal's
own dummy carries 0.09 on ranks 1-6, 0.9 on rank 4 and nothing on rank 7. Before the generator
followed the trigger, the fallback took the dummy, and rank 7 generated at 1691 with `Coef: 0`.

A reference into a spell one of the rank's own effects triggers is followed to the named effect
whatever its shape, and the effect files by that shape: Seal of Fury's damage lands in `Direct`,
Seal of Light's heal in `Heal`, Seal of Wisdom's mana in `Energize`. Before this, Seal of Light and
Seal of Wisdom generated with the judgement's spell ID in `Direct`, read off the pointer dummy by the
last fallback.

```go
d := spellData.SealOfFury.BySpellID(20423).Direct.AsFlat()
d.Value    // 35, the proc's school damage
d.Coef     // 0.1, which only the proc states
```

A flat value does not name its source the way a tick does; the proc's spell ID is on the
`SealOfFuryTriggered` table beside it.

The trigger is read off every effect, not the dummy alone: rank 5 keeps it on the judgement pointer
and rank 7 on the damage dummy. The same rule reaches Arcane Missiles' per-missile damage, Intercept's
damage and the hunter pet abilities whose learn spell names the taught spell's number, so a rank that
used to carry no value in a role may carry one now.

## Talents

A talent is read by the points spent in it, not registered at a rank it has, so the ladder has its own
three readers. All of them answer the identity at rank 0 - an untaken talent - where `ByRank` would
panic:

```go
spellData.Moonfury.FractionAt(rank)        // 0.10 at 5/5 - the client's 10, over 100
spellData.NaturesReach.ValueAt(rank)       // 20 at 2/2  - the client's number as it stands
spellData.LivingSpirit.MultiplierAt(rank)  // 1.15 at 5/5 - 1 + the fraction
```

That replaces the `<literal> * float64(x.Talents.Y)` idiom, and with it the `if rank > 0` guard the
caller would otherwise need.

**`MultiplierAt` takes its sign from the data.** Improved Righteous Fury states its damage reduction as
-2 / -4 / -6, so rank 3 gives 0.94 and nobody writes the minus. Where the sim's parameter runs the other
way - `AddReducedCritTakenPercent` wants a positive amount for a reduction the client states negative -
negate at the call site so the disagreement is visible.

**Ladders are not always the per-point literal times the rank.** Most are: of 125 percent talents, 122
scale linearly, so `0.02 * rank` was already right and the table only adds provenance. The ones that do
not are the reason to read it - Improved Righteous Fury is 16 / 33 / 50, not 16 / 32 / 48, and shaman
Elemental Weapons is 7 / 14 / 20, not 7 / 14 / 21.

A single row's effect has the same readers without a rank: `row.Effect(aura, misc).Fraction()`,
`.Multiplier()` and `.Tenths()`, so Shield Wall reads `Effect(A_MOD_DAMAGE_PERCENT_TAKEN, 127).Multiplier()`
for its 0.4 and Bloodrage's energize reads `.Tenths()` for its 10 rage.

### Picking the effect

A talent with one effect per rank needs nothing further. One with several does, and `ValueAt` panics
rather than guess:

```go
spellData.ImprovedRighteousFury.
    Effect(shared.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_ALL_EFFECTS)).MultiplierAt(rank)   // 1.50 threat
spellData.ImprovedRighteousFury.
    Effect(shared.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_EFFECT2)).MultiplierAt(rank)      // 0.94 taken
```

**Do not pick the effect by which one matches the number.** Survival of the Fittest states +1/2/3% to
all stats and -1/-2/-3% crit taken; both ladders fit, and an automatic pass attached the stat effect to
the crit-taken call site. What the call site does decides it, and the mod's `Kind` usually says:
Improved Moonfire's two mods are `SpellMod_DamageDone_Flat` and `SpellMod_BonusCrit_Percent`, so one
takes `SPELLMOD_DAMAGE` and the other `SPELLMOD_CRITICAL_CHANCE`.

Where a talent modifies damage and its DoT with the same ladder, the sim has one mod against the
client's two. Either aura reads the same number; `SPELLMOD_DAMAGE` is the convention here.

An effect the tree states no curve for is the same at every rank, and sits in `Effects` at the spell's
own base points: Blood Craze's second effect is the 20% of maximum health a hit has to exceed, at 1/3
as at 3/3. Only the priced effects fill the role fields, so `ValueAt` and the ladder readers never
see it; reach it through `Effect` or `Effects[i]` on any rank. A one-rank node on a passive nothing
teaches - Raging Blows, Vanguard - is a table of one row built the same way, so
`spellData.RagingBlows.EffectAt(1).TenthsAt(1)` reads its -2 rage on Cleave.

### Proc chances

`SpellAuraOptions.ProcChance` is a separate source from the effects, and `ProcChanceAt` reads it as the
fraction a `ProcTrigger` takes. It is one input of three; the tooltip (`Spell.Description_lang`, colour
codes stripped) and the effect ladders are the others, and together they put every proc in one of four
shapes. Nothing about a proc is carried over from an earlier expansion on trust: a proc that was PPM
may state a percentage now, and the other way round.

1. **The tooltip carries `$h%`.** The column is the chance:
   `ProcChance: spellData.SealFate.ProcChanceAt(rogue.Talents.SealFate)` reads 0.20 at 1/5 and 1.00
   at 5/5. Enrage is this shape too: a real 30% roll on damage taken.
2. **The tooltip carries `$mN%` or `$sN%`.** The chance is that effect's ladder,
   `Effect(...).FractionAt(rank)`, and the column is noise: Unbridled Wrath states 12/24/36/48/60 by
   rank on its effect while the column reads a flat 60.
3. **No chance in the tooltip and the column reads 100 or 101.** The aura fires on its own condition
   and there is no roll: Flurry and Deep Wounds on a crit, Dual Wield Specialization on every hit. A
   101 on something that is not a proc at all (Sunder Armor, Demoralizing Shout) means nothing. A
   tooltip whose _trigger clause_ says the effect only happens sometimes - "Chance to strike your
   ranged target", "your melee swings have a chance to" - or that says "often", "sometimes" or
   "occasionally" (Darkmoon Card: Heroism's "Sometimes heals") is shape 4 rather than this one, and on a
   chance-on-hit weapon, where the game consults no condition at all, the 100 and 101 always are.
4. **No chance in the tooltip, and a column the tooltip's trigger clause contradicts.** A procs-per-minute
   proc the client does not carry (`SpellProcsPerMinuteID` is 0 on every row). The PPM is
   hand-supplied the way threat and attack power coefficients are, with the manual-review TODO
   quoting the raw column on the line:

    ```go
    var imbue = shared.WithSpellDataPPM(spellData.Imbue, 2)                          // one PPM, every rank
    var strike = shared.WithSpellDataPPMs(spellData.Strike, map[int32]float64{1: 1, 2: 1.5})
    dpm := character.NewLegacyPPMManager(imbue.PPMAt(rank), core.ProcMaskMelee)
    ```

    The per-rank form has to name every rank, and both panic on a table that already carries a PPM.

A `$<id>h` in a tooltip reads another spell's column, so the chance sits on that spell's table, not on
the one the tooltip belongs to.

### The high end of an effect

`SpellDataEffect.Value` is the low end. An aura with one die side and a fractional base has two ends a
whole number apart, and the game shows the higher: Seal of the Crusader rank 1 states 39.2 attack power
and buffs for 41. `ValueMax` holds it, and is zero on the 82% of effects where the two agree, so read
`High()` rather than `ValueMax` - rank 4's base is whole, so it has no `ValueMax` and its answer is
`Value`.

```go
spellData.SealOfTheCrusader.ByRank(rank).Effects[0].High()   // 41 at rank 1, 183 at rank 4
```

### The Misc value

`Misc` says what an effect applies to, and what it means depends on the aura: a modified spell property
under `A_ADD_PCT_MODIFIER` and `A_ADD_FLAT_MODIFIER`, a stat under `A_MOD_TOTAL_STAT_PERCENTAGE`, a
school mask under `A_MOD_DAMAGE_DONE`. There is no single enum for it, so it stays an int.

For the two modifier auras the `SPELLMOD_*` constants name it. Those are the `SpellModOp` values in
`sim/core/dbcenums/spellmods.go`, written by hand because the client ships no name list; each says
what it modifies. Ops 0 to 30 carry [TrinityCore's 3.3.5 names][tc], 31 to 40 its current ones.

[tc]: https://github.com/TrinityCore/TrinityCore/blob/3.3.5/src/server/game/Spells/SpellDefines.h

## Worked examples

### Direct damage

```go
var exorcismRanks = spellData.Exorcism.BySpellID(27138)

func (paladin *Paladin) registerExorcism() {
	paladin.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: exorcismRanks.SpellID},
		Rank:             exorcismRanks.Rank,
		ManaCost:         core.ManaCostOptions{FlatCost: exorcismRanks.Cost},
		BonusCoefficient: exorcismRanks.Direct.BonusCoefficient(),
		MaxRange:         exorcismRanks.MaxRange,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: exorcismRanks.GCD, CastTime: exorcismRanks.CastTime},
			CD:          core.Cooldown{Timer: paladin.getExorcismTimer(), Duration: exorcismRanks.Cooldown},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, exorcismRanks.Direct.Damage(sim), spell.OutcomeMagicHitAndCrit)
		},
	})
}
```

### A damage-over-time effect

The tick and its schedule both come from the table, so `NumberOfTicks` and `TickLength` stop being
hand-written:

```go
var swpRanks = spellData.ShadowWordPain.BySpellID(25368)

func (priest *Priest) registerShadowWordPain() {
	tick := swpRanks.Periodic.AsPeriodic()

	priest.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: swpRanks.SpellID},
		ManaCost: core.ManaCostOptions{FlatCost: swpRanks.Cost},

		Dot: core.DotConfig{
			Aura:             core.Aura{Label: "ShadowWordPain-" + swpRanks.GetRankLabel()},
			NumberOfTicks:    tick.NumberOfTicks,
			TickLength:       tick.TickLength,
			BonusCoefficient: tick.Coef,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Snapshot(target, tick.Tick)
			},
		},
	})
}
```

### A tick, and a second one for the first few targets

Consecration ticks on everyone in the area and again on the first four to enter it, with the spell
power coefficient on the second tick only, so the bonus is added to the base damage per target rather
than through the dot's coefficient:

```go
func (paladin *Paladin) registerConsecration(rankConfig shared.SpellData) {
	tick := rankConfig.Periodic.AsPeriodic()
	bonus := rankConfig.SecondaryPeriodic.AsPeriodic()
	bonusTargets := int(rankConfig.Effect(shared.A_PERIODIC_DUMMY, 0).Value)

	dealTick := func(sim *core.Simulation, dot *core.Dot) {
		for i, target := range sim.Encounter.ActiveTargetUnits {
			damage := tick.Tick
			if i < bonusTargets {
				damage += bonus.Tick + bonus.Coef*dot.Spell.BonusDamage(dot.Spell.Unit.AttackTables[target.UnitIndex])
			}
			dot.Spell.CalcAndDealPeriodicDamage(sim, target, damage, dot.OutcomeTickMagicHit)
		}
	}
	// ...
}
```

### A heal, and a mana restore

```go
holyLight := spellData.HolyLight.BySpellID(27136)
low, high := holyLight.Heal.Range()          // both ends in one call

layOnHands := spellData.LayOnHands.BySpellID(27154)
mana := shared.SpellDataMin(layOnHands.Energize)   // 900; rank 1 has no Energize at all, and reads 0
```

A restore that ticks is an `Energize` of the periodic shape, with the schedule a `Hot` or a periodic
action wants: Bloodrage's 29131 ticks 10 rage-tenths every second for 10 ticks.

```go
over := spellData.BloodrageTriggered.HighestRank().Energize.AsPeriodic()
over.Tenths(), over.TickLength, over.NumberOfTicks   // 1 rage, 1 s, 10
```

### Registering several ranks

Downranking registers more than one, and the spec chooses which:

```go
// Starfire's ladder runs 1-8; the sim registers only these two.
spellData.Starfire.Ranks(6, 8).RegisterAll(druid.registerStarfireSpell)

func (druid *Druid) registerStarfireSpell(rank shared.SpellData) {
	druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
		ActionID: core.ActionID{SpellID: rank.SpellID},
		Rank:     rank.Rank,
		ManaCost: core.ManaCostOptions{FlatCost: rank.Cost},
		Cast:     core.CastConfig{DefaultCast: core.Cast{GCD: rank.GCD, CastTime: rank.CastTime}},
		// ...
	})
}
```

Each rank is its own registered spell with its own ActionID, which is what lets an APL name a downrank.

## Attack power

Attack power scaling is **not** in the client data - one effect in 38357 carries a nonzero
`BonusCoefficientFromAP` - so melee coefficients live in server script and have to be supplied by hand:

One coefficient for the whole ladder:

```go
// Rupture's ranks are all periodic, so the coefficient goes on the tick.
var ruptureRanks = shared.WithSpellDataPeriodicAPCoef(spellData.Rupture, 0.18)
```

Or one per rank, the way the spell power coefficient already varies because each row carries its own:

```go
var ruptureRanks = shared.WithSpellDataPeriodicAPCoefs(spellData.Rupture, map[int32]float64{
	1: 0.04, 2: 0.06, 3: 0.08, 4: 0.10, 5: 0.12, 6: 0.15, 7: 0.18,
})
```

**Every rank in the table has to be named.** Leave one out and it panics rather than scaling that rank
off nothing, and naming a rank the ladder does not have panics too - so a ladder that gains a rank in a
later client build fails loudly instead of quietly mis-scaling.

|                                        |                                |
| -------------------------------------- | ------------------------------ |
| `WithSpellDataAPCoef(t, c)`            | one coefficient, on `Direct`   |
| `WithSpellDataPeriodicAPCoef(t, c)`    | one coefficient, on `Periodic` |
| `WithSpellDataAPCoefs(t, map)`         | per rank, on `Direct`          |
| `WithSpellDataPeriodicAPCoefs(t, map)` | per rank, on `Periodic`        |

All four return a copy, so the generated table keeps what the database said. All four panic if the role
is nil on any rank - check the generated table first, `spellData.Mangle` is the _learn-spell_ entry
(`Effect = 36`) and carries no value at all - and if the table already carries a coefficient, because a
value that appears upstream should be noticed, not silently shadowed.

## A row that is more than one spell

A seal is three spells for one rank - the aura, the proc it triggers and the judgement - so it keeps
its own row type rather than becoming a `SpellData`, and `SpellDataTableOf` is generic for exactly
that. What the client states is read from the tables; what it does not is passed in, and the shorter
name goes to the common case so a family that diverges reads differently from one that does not.

```go
// everything the client states, judgement damage included
sealOf(spellData.SealOfCommand, spellData.JudgementOfCommand, rank, proc{...})

// for the families whose judgement damage has to be supplied by hand
sealWithJudgement(spellData.SealOfLight, spellData.JudgementOfLight, rank, proc{...}, judge{...})
```

This is the pattern for any composite that follows - totems, poisons. Two things it taught: put the
reason for each literal on its own line rather than in a block at the top, and do not trust a golden
to verify it. The paladin goldens carry no seal spell ID at all, so the port was checked by dumping
every constructed row against the literals it replaced.

## Regenerating and checking

```
go run ./tools/database/gen_spelldata              # rewrite every generated spell data file
go run ./tools/database/gen_spelldata -check       # name the ones that are stale, write nothing
go run ./tools/database/gen_spelldata -unchecked   # write without the type-check below
make spelldata-check                               # the same check
```

The generator reads `tools/database/wowsims.db` and writes all of it in one pass:
`sim/core/spelldata/spells_auto_gen.go`, `sim/common/shared/spell_data_enums_auto_gen.go`, a
`sim/<class>/spell_data_auto_gen.go` for every class - ladders into the store for a store-backed class,
the family tables for the rest - `sim/core/dbcenums/forms_auto_gen.go`, and the raid buffs, which
[Buffs and debuffs](#buffs-and-debuffs) describes. `storeBackedClasses` in
`tools/database/gen_spell_data.go` lists eight
classes; paladin is the one it does not. It is its own binary rather than a mode of `gen_db` on purpose:
`gen_db` imports the sim and the sim reads these files, so a stale one would stop the generator that
fixes it from compiling. For the same reason nothing is written until all of it type-checks: the
rendered bytes go to a staging directory first and are compiled through `go build -overlay` in the
place of the committed files, and a failure leaves the tree exactly as it was and prints what the
compiler said. `-unchecked` skips that build, for a class just flipped to the store whose call sites
have not moved off the family table yet and so cannot type-check until they have - the reference file
is written first and the ports follow; `-check` closes the loop once the package builds again. `make
db` runs the store before `gen_db` for the same ordering reason - `gen_db`
classifies every item and enchant proc out of the store compiled into it.

Nothing lists which spells to generate. The class files walk `dbc.Classes`, take each class's own skill
lines and emit a family for every spell whose subtext reads `Rank N`; the store starts from what those
families name, from the item, enchant and set-bonus tables, and from the talent trees, and closes over
everything those reach. A family that could not be resolved is named in the `// Not generated:` comment
at the head of the class file.

`assets/db_inputs/spell_store_inputs.json` is the client rows the store was built from, committed
beside it, and every `SpellShapeshiftForm` row whole alongside them, since a form names no spell for
the store's closure to reach it by. It is gzip-compressed JSON despite the name, like everything under
`assets/db_inputs/dbc` - `zcat` it to read it. It exists so the store, the buffs and the shapeshift
forms can all be rebuilt without the client database, which is gitignored and comes from a local WoW
install, and `-check` compares it too: a capture that no longer matches the database would render the
committed files all the same, so nothing else would notice it going stale.

What guards the outputs:

- `TestStoreRegeneratesFromTheCommittedInputs` in `tools/database` re-derives the closure, the hand
  links, the curves, the tooltip hints, the overrides and the emitter from that capture and asserts the
  committed store is what comes out. No build tag and no database, so it runs everywhere.
  `TestBuffFilesRegenerateFromTheCommittedInputs` does the same for the buff files, and
  `TestFormsRegenerateFromTheCommittedInputs` for `sim/core/dbcenums/forms_auto_gen.go`.
- `sim/core/spelldata/snapshot_test.go` pins the shape and the counts of the committed store, and a
  handful of rows read out of the client by hand, so a regeneration that moves a number says so there
  instead of in a sim result.
- `sim/paladin/spell_data_parity_test.go` compares every value paladin's family table states with the
  same spell as the store carries it, through `sim/core/spelldata/parity`. That is what has to stay
  green for paladin before it ports; the eight store-backed classes carry no parity test of their
  own - it retires with the family table it checked.
- `go test ./tools/database/ -run GeneratedRankTables` re-derives amounts and coefficients for the 8
  families listed in `tools/database/spelldata_regen_test.go` from the database itself - 147 comparisons
  out of the rows paladin's family tables hold, so it is no substitute for regenerating and finding
  the diff empty. It skips when `wowsims.db` is absent.
- The repository's `pre-commit` hook runs `-check` when the database is present and the commit touches
  the generator or one of its outputs.

Goldens are the last check and the one that costs time: run the suites of the class that changed, and
diff the `.results` against the `.results.tmp` with a local goldendiff helper or by hand. A port that
was meant to be mechanical and moves a golden is a wrong port, not a new baseline.

## Porting a class to the store

1. **Flip the class** in `storeBackedClasses` (`tools/database/gen_spell_data.go`) and regenerate with
   `-unchecked`: the class file becomes one ladder per family, and nothing else changes until the call
   sites move, so the package cannot type-check until they have - the reference file is written first
   and the ports follow; `-check` closes the loop once the package builds again. Run
   `sim/<class>/spell_data_parity_test.go` first: every value the family table states has to match the
   store's before the class reads the store instead. The move from there forks in two: from the table
   to the same reads by hand, off a `spelldata.Ladder` instead of a `shared.SpellDataTable`, is what
   [Reading a row by hand](#reading-a-row-by-hand) describes, and it is the whole job for a class that
   stops there. From hand reads onto the resolvers - `SpellConfig`, `AuraConfig`, `DotConfig`,
   `ParseEffects`, `ProcTrigger` - is the warrior's move, and the rest of this checklist is written
   for it.
2. **Dump the rows before touching a call site**, three ways: the effects at the rank taken, which
   registered spells each modifier's `EffectSpellClassMask` names, and the whole `ProcTrigger` the row
   decodes to. The mask dump is what turns a golden move into something you knew before you ran it.
3. **Pass `Ladder.Rank(n)`, never `MustFind(id)`,** for anything the talent tree prices: a trait
   talent's per-rank numbers live in the curve, and the base row's effect can read 0.
4. **Check the cooldown categories.** A category cooldown resolves to `Unit.CategoryTimer`, which is
   one map per unit and the same one consumables and racials use for categories 4, 30, 1141, 1153 and 1190. A class row stating one of those would share a cooldown with a potion. Two of the class's own
   abilities sharing a category share the timer, which is what the client does: the warrior's Revenge
   and Overpower run off 65, Shield Bash and Pummel off 88, and Mortal Strike, Bloodthirst and Shield
   Slam off 971.
5. **Read what the resolver picked up.** `SpellFlagHelpful` follows the first effect's target and flips
   `IsFriendly` in the APL editor, so an attack whose first effect is a self side-effect needs it
   cleared. `Rank` comes from "Rank N" and flips `HasRanks` there. `IgnoreHaste` follows the physical
   school, not the defense type.
6. **A row with a `Variance` rolls.** Use `Roll(sim, level)` where the client states a spread; the
   extra draw reshuffles every later roll, which is a golden move with a known cause.
7. **Parse before you activate.** `ParseEffects` attaches on gain, so an aura already up when the parse
   runs loses the rows that need a `Simulation`.
8. **Compare value by value before deleting a hand modifier.** A mod that lands in a different bucket -
   a school pseudo-stat where the sim had a per-spell mod - is the same number in a different place,
   and a DPS diff will not show it. Diff the `.results` as well as the DPS numbers.
9. **`RequireDamageDealt` defaults true.** A listener that fires on a dodge, a parry, a miss or a block
   needs it false, which the decoder has already done where the tooltip named the outcome and the row
   carries `ProcHintOutcomeTaken`; clear it by hand only where the wording yielded no hint. The outcome
   itself is always the caller's.
10. **Rename a trigger whose driver and buff share a name**, and split a row that decodes to hits dealt
    _and_ taken into two listeners with a condition each.
11. **Say which source won.** Where the row and the tooltip disagree, the decision goes in a comment
    that names the tooltip's wording. Where a number stays by hand, its review marker stays with it.
12. **Isolate every golden move** by reverting exactly one change and re-running. Two causes in one
    commit look like one inexplicable cause.
13. **Delete the class masks last**, and only where the constant's sole reader is the `ClassSpellMask`
    write. A handler calling `spell.Matches` is a reader too.
14. **A gear proc is three calls, not one.** `ProcTrigger` for the listener, `AuraConfig` for the buff
    it applies and `ParseEffects` for what that buff does - the idiom `sim/warrior/items.go` uses for
    every set bonus and item buff the class owns.
15. **`Melee` and `Magic` put the spell in the rotation; `Flags` does not.** A stance-gated
    ability takes `Flags` so the APL editor does not offer it, and a config that writes
    `ThreatMultiplier = 1` or `FlatThreatBonus = 0` after the resolver is writing what the resolver
    already wrote - keep it only where its review marker says the number is still to be measured.

## Traps

**The data contains ranks the game never grants.** Fireball 38692 and Frostbolt 38697 are rank 14
entries at level 70 whose `SkillLineAbility` rows and spell attributes are byte-for-byte
indistinguishable from the real rank 13s, so the generator cannot filter them. Reaching for
`HighestRank()` on those families silently casts a spell that does not exist - it cost 1.2% DPS when it
happened during the mage port. Use `BySpellID`.

**`HighestRank()` is not "the rank my spec casts".** It is the largest rank number present, which is
also not the last element: Flamestrike is declared rank 7 then rank 6.

**A rank can carry nothing in a role.** Lay on Hands rank 1 restores no mana where ranks 2-4 do, so
`rank.Energize` is nil there. The `SpellData*` helpers read nil as zero; a direct field access does not.

**Never delete a generated file before regenerating it.** The class package stops compiling, and
`gen_db` - which imports the sim - then cannot build either. Regenerate over the top, or
`git checkout HEAD -- <path>` to get back.

**A melee ability states its bonus as weapon damage, not school damage.** Sinister Strike's +98 is
`Effect = 121` (normalised weapon damage); the generator reads 17, 58 and 121 alongside school damage,
so those land in `Direct`. `Effect = 31` (weapon percent damage) is deliberately excluded - it is a
multiplier on the swing, not an amount a rank can carry.

**Rage costs are divided by ten on the way in.** The client stores rage on a 0-1000 bar, so Heroic
Strike's cost reads 150 where the player sees 15. `Cost` is always in the units the sim uses; mana,
energy and focus need no conversion, and only rage does. An `Energize` effect that restores rage would
still be in tenths - nothing generated today does.

**A new client table needs a settings line.** `SpellCastTimes` was missing from
`generator-settings.json` and cast times read zero until it was added and `make db` re-run. Adding a
table is one line; the extractor needs no code.

The store's own:

**`EffectN` counts positions, `Effect.Index` is the client's number.** `EffectN(1)` is the first effect
the row carries, whatever index the client gave it; 46 rows state an index that is not the position it
sits at. A store row read with the client's number in hand is off by one wherever the two agree, and
wrong in a different way where they do not.

**`Average` truncates the base before it scales it.** The client resolves an amount to a whole number,
so an effect whose `EffectBasePointsF` is fractional - about one in 55 - contributes its integer part
and the per-level gain is added on top in float32. Reading `BaseValue()` and doing the arithmetic in
float64 gives a different answer on those rows.

**A row with more than one effect has to name the one it means.** `Ladder.ValueAt` panics rather than
read the first of several, and `Effect(aura, misc)` panics when two effects match. Both are the same
rule: `EffectAt(n)` or `EffectN(n)` is how a caller says which.

**A hand-built `Effect` scales from its own `SpellLevel` and `MaxLevel`.** The generator stamps both onto
every effect so `Average(level)` needs no lookup; an `Effect` literal in a test or a hand-written config
that leaves them zero scales `PPL` from level 0 up to the caster's level. Copy them from the spell's row,
or set `PPL` to 0.

**`MustFind` at package init fails at startup, not at the call site.** That is the point - a
regeneration that drops or renumbers an id stops the sim with the id in the message - but it means a
package-level `var` reaching for a spell this build does not carry takes the whole package down,
including tests that never touch that spell.

**`SPCoef` of exactly 1 is the column's filler.** 2,113 of the store's 9,651 effects carry it, on
weapon-damage effects, speed auras and shapeshifts among them, and no physical-school row in this
client states a fractional coefficient at all. `Magic()` and `DotConfig` hand the row's coefficient
to core as it stands, so a physical row read through either would take spell power per hit or per
tick: read the tooltip before you believe a coefficient of 1.

**`Spell.ProcChance` is not always a chance.** 100 and 101 are the client's "fires whenever its own
condition is met", and the tooltip is what says which the column is: `ProcChanceSource`, baked in at
generation, is the answer, and `spelldata.ProcTrigger` reads it rather than the column.

**A proc row with no stated rate panics when the trigger is built.** `ProcChancePPM` with no override
behind it means the client states nothing anywhere, so `spelldata.ProcTrigger` demands a `PPM()` rather
than build a listener that fires on every hit.

## Buffs and debuffs

Every raid, party, individual and enemy-debuff proto field has one row in
`tools/database/buffmanifest`. The manifest owns the field numbers and names the spell each row reads;
the store owns the values. The same `gen_spelldata` pass that renders the store renders, from the same
captured inputs:

- `sim/core/buffs/buffs_auto_gen.go` and `sim/core/buffs/debuffs_auto_gen.go`, a constructor per row;
- `ui/features/settings/model/buffs_debuffs_auto_gen.ts`, the settings inputs;
- `proto/buffs.proto`, the four messages.

They go through the same staging type-check and the same `-check` as the store, and regenerate without
the client database.

### How a buff reads the store

The generated constructors live in `sim/core/buffs`, above core, because the store imports core. Each
supported row holds a package-level `*Meta` and reads every number off it at runtime, through the same
`spelldata.ParseEffects` a class's own aura reads its stats through - a generated buff carries no effect
routing of its own:

```go
var battleShoutSpell = spelldata.MustFind(25289)
var battleShoutMeta = &Meta{
	Label:      "Battle Shout",
	Spell:      battleShoutSpell,
	Category:   BattleShoutCategory,
	SingleAura: true,
}

func BattleShoutValue(talentPoints int32) float64 {
	return battleShoutMeta.Value(talentPoints)
}
func BattleShoutDuration(talentPoints int32) time.Duration {
	return battleShoutMeta.Duration(talentPoints)
}
func BattleShoutAura(unit *core.Unit, isPlayer bool, talentPoints int32) *core.Aura {
	return newBuff(unit, battleShoutMeta, isPlayer, talentPoints)
}
```

`Meta.Options(talentPoints)` states the `spelldata.ParseOpt`s every row needs:
`spelldata.RaidBuffOptions` - `Level(core.CharacterLevel)`, `BuffAuras`, `SkipAuras` where the row
states any and `FullComboPoints` for a finisher, which the generator parses with too - and `ScaledBy`
the improving talent (left out where the talent scales the duration instead, since `Duration` reads
that separately). `newBuff` adds `SchoolResistances` and, where
`Category` is set, `Exclusive(Category, true)` for a `SingleAura` row or `ExclusivePerStat(Category)`
for any other, so a generated buff and a hand-written scroll of the same stat or resistance bid
against each other under the categories core's own exclusive stat buffs use. `newDebuff` adds only
`Exclusive(Category, SingleAura)`, since a debuff never carries a resistance of its own;
`newItemCountBuff` adds `Count` so that a party with several of the same item is worth that many
copies of the amount. `Value` is a damage shield's own effect where the row states one, and otherwise
the first thing `spelldata.DryRun` attaches for those options. `Duration` reads the spell, or `Cast` where the
spell states no duration of its own; `Cooldown` reads `Cast` where the row pins one, else the spell
itself; both go through the helpers in `sim/core/buffs/amounts.go`, and a talent that scales the
duration truncates it the way `talentScaled` truncates an amount everywhere else. A row whose aura is
a damage shield skips the parse outright: `newDamageShield` calls `core.NewDamageShield` with the
spell's school and `Value`.

`sim/core/spelldata/buff.go` holds a second table, read only when a parse states `BuffAuras()`:
`A_PERIODIC_ENERGIZE` on a mana row becomes MP5 there, and `A_MOD_ATTACK_POWER_PCT` a percentage on
attack power. Neither sits in the parse's shared table, because warrior rows carry them for a value the
class wires itself - Bloodrage's rage tick is an `A_PERIODIC_ENERGIZE`, Berserker Stance Passive an
`A_MOD_ATTACK_POWER_PCT` of 0 - and reading either off the shared table would change what those rows
mean there.

The generator asks the same parse for every manifest row: `spelldata.DryRun`, given the row's options
and no character, answers what it attaches, and `SkippedNotes` on that answer what it leaves out, so a
row the generator writes is one the sim builds the same way. A row a driver decides
the meaning of - `KindExternalCD`, `KindProc`, `KindManual`, `KindDebuffUptime` - is written whatever
the parse attaches; every other kind needs an amount, or for `KindDamageShield` the shield effect
itself. A row the parse attaches nothing of renders as a commented shell naming the reason, and a row
it does write states what it could not read as a `// Left out:` note above it in the generated file,
one per aura effect the parse has no row for.

Core applies the buffs through `core.BuffHooks`, which `sim/core/buffs` registers from its `init`:
`ApplyBuffs`, `ApplyDebuffs` and the Gift of Arthas elixir's debuff. A sim that
never imported the package panics naming the import. `sim/common` imports it, which links it into every
sim and every class test; core's own tests reach it through the generated-buff tests, which sit in
`package core_test` beside `sim/core/export_test.go`.

### Adding a buff

1. Add the proto field's row to `buffmanifest.Manifest`: field, number, scope, proto type, kind, Go stem,
   and the `SpellID` its numbers are read from. Pin a `CastID` where the cast states the timing the
   aura does not, and a `Talent` with its `SpellID` where a talent improves it.
2. `go run ./tools/gen_buffs_proto` and `make proto`, so the compiled protos carry the field. The pass
   below type-checks against them and writes nothing while the field is missing.
3. `go run ./tools/database/gen_spelldata`. The spell becomes a root of the store, and the constructor,
   the apply block and the settings input are written.
4. A row the kind cannot express outright states `Driver: true`, and `sim/core/buffs/drivers.go`
   declares `drive<Go>`.
5. With a database, `TestManifestAnchorsMatchTheClient` checks the pin is the top rank the owner's skill
   lines grant.

### The manifest row

`BuffSpec` in `tools/database/buffmanifest/manifest.go`:

| Field                      | What it is                                                                                                                                                                                                                                                                                                                                                                                                                    |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Field`, `Number`, `Scope` | the proto field, its number and the message it lives on. Each scope's numbers are dense from 1, so a row that goes away is a renumber of the rows after it                                                                                                                                                                                                                                                                    |
| `Proto`                    | `ProtoBool`, `ProtoTristate`, `ProtoInt32` or `ProtoDouble`. Declared, not derived, so the emitter runs while the compiled protos are stale; the generator checks it against the compiled message                                                                                                                                                                                                                             |
| `Kind`                     | what the generator emits, below                                                                                                                                                                                                                                                                                                                                                                                               |
| `Go`                       | the identifier stem: `BattleShout` gives `BattleShoutAura`, `BattleShoutValue`, `BattleShoutDuration`, `BattleShoutCategory`, `battleShoutSpell` and `battleShoutMeta`                                                                                                                                                                                                                                                        |
| `SpellID`                  | the spell the numbers are read from, and a root of the store: the top rank of the castable family, or the aura that family's cast applies when the cast is a summon or a dummy                                                                                                                                                                                                                                                |
| `CastID`                   | the cast, for a row whose timing only the cast states: Mana Tide's aura 17360 carries the mana, the cast 17359 the 13 seconds and the 5 minutes                                                                                                                                                                                                                                                                               |
| `Name`, `AuraName`         | the castable family's `SpellName.Name_lang` and, for a summon or a dummy, the aura family's. `Name` is the label's default; both are what `TestManifestAnchorsMatchTheClient` resolves the pins from                                                                                                                                                                                                                          |
| `Owner`                    | the class that casts it, which narrows the rank lookup and marks the row "(External)" on that class's settings tab                                                                                                                                                                                                                                                                                                            |
| `Talent`                   | the improving talent's `SpellID`, name, effect index, and whether it scales the value, the duration or adds a stat. Only a `ProtoTristate` row may state one, and its effect has to be a modifier whose class mask reaches the row's spell. The generated `Meta.TalentEffect` names the same effect by its `EffectN` position rather than by this index                                                                       |
| `Category`                 | the exclusive-effect category the aura bids in, `""` for none                                                                                                                                                                                                                                                                                                                                                                 |
| `SharedCategory`           | a second category the aura joins without an effect of its own, which is how the paladin auras exclude each other across schools. Applied to the player's copy only, and declared once in the generated file as `<Name>Category`                                                                                                                                                                                               |
| `SingleAura`               | the category holds one aura at a time, so the loser is deactivated rather than outbid                                                                                                                                                                                                                                                                                                                                         |
| `Driver`                   | the apply block hands the row to `drive<Go>` instead of activating the aura outright                                                                                                                                                                                                                                                                                                                                          |
| `SkipAuras`                | aura names, spelled the way `sim/core/dbcenums` spells them, for an effect of the row's spell that sits beside the buff and that the raid's copy does not apply. Resolved to `dbcenums.EffectAuraType`s while the row is built; the six paladin auras skip `A_MOD_HEALING_PCT`, which each states as a healing-taken row of 0, and `A_MECHANIC_DURATION_MOD`, which only Concentration Aura states, as two mechanic rows of 0 |
| `Stats`                    | the UI relevance tags a spec's `epStats` and `displayStats` are matched against                                                                                                                                                                                                                                                                                                                                               |
| `ImpAction`                | the improved state's source when it is not a talent - an item, or the spell an item set grants at a piece threshold - and the icon that state shows. A `ProtoTristate` row states this or a `Talent`                                                                                                                                                                                                                          |
| `Label`                    | a UI label override; the client's name is the default                                                                                                                                                                                                                                                                                                                                                                         |
| `Notes`                    | why a `KindManual`, `KindAbsent` or `KindFlag` row is one. Required for those three                                                                                                                                                                                                                                                                                                                                           |

### The kinds

`KindStatFlat`, `KindStatPct` and `KindResistance` are stat buffs; `KindPseudoMult` moves a
pseudo-stat; `KindDamageShield` is a retaliation proc; `KindProc` and `KindExternalCD` need a driver
for the trigger or the cooldown; `KindItemCount` takes a count and applies its amounts per item;
`KindDebuffStat`, `KindDebuffStacking`, `KindDebuffDamageTaken`, `KindDebuffAtkSpeed` and
`KindDebuffUptime` are the debuff shapes. `KindManual` is a row the sim models by hand, `KindFlag`
is a sim input rather than a buff (a toggle, or a number such as `retribution_aura_spell_power`), and
`KindAbsent` is a field the Forever client describes no spell for. The last two resolve to a commented
shell naming the reason, as does a row whose spell states no aura effect the parse attaches. No `Proto`
value is an enum, and a field that wants one would add its own value and a name for it in both emitters.

### What stays hand-written

Judgement of the Crusader states its effect in a shape no manifest row can carry - a holy school
alone - so its row is a shell and `applyDebuffs` in `sim/core/buffs/debuffs.go` applies the
hand-written aura after the generated ones. The paladin's own aura and judgement ranks
(`PaladinAuraRank`, `JudgementRank`) live in `sim/core/buffs/paladin.go` beside the generated
categories they join.

### Drivers

A row the generator cannot express outright states `Driver: true`, or is a kind that always needs one,
and the apply block calls `drive<Go>` instead. The contract is in `sim/core/buffs/drivers.go`: a buff
row's driver takes the `*Character` and the whole scope message, a debuff row's takes the `*Unit`, the
debuffs and the raid. Handing over the message rather than the one field is what lets Grace of Air read
`party.TotemTwisting`. The driver builds the generated aura with `<Go>Aura(...)` and adds what the
client does not state: a proc trigger, a cooldown, a delay, a regen. Declaring the function is what
makes the row compile.

### Regenerating

`go run ./tools/database/gen_spelldata` writes the buff files with everything else, and `-check` names
them when they are stale. `tools/database` imports `sim/core`, so the generator cannot run while the
tree it generates into does not compile. A change that breaks it - retiring a proto field, deleting a
driver the generated file still calls - needs the generated file cut by hand first: delete the lines
that name the field or symbol that is going away, `go build ./...`, then run the generator, which
writes the whole file back. `tools/gen_buffs_proto` renders `proto/buffs.proto` alone, with nothing but
the manifest, for the one moment the sim cannot build: before protoc has seen a new field.

### The guard tests

`go test ./tools/database/...` needs no client database and runs in CI; only
`TestManifestAnchorsMatchTheClient`, `TestGeneratedRankTablesMatchTheDatabase` and
`TestProcShapeOfNamedSpells` skip without one.

| Test                                                                                                              | What it holds                                                                                                                                        |
| ----------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TestUniqueScopeField`, `TestUniqueScopeNumber`, `TestUniqueGoStem`                                               | no two rows collide                                                                                                                                  |
| `TestScopeNumbersAreDense`                                                                                        | every scope's field numbers are 1..N with no gap                                                                                                     |
| `TestProtoTypeMatchesKind`, `TestTalentImpliesTristate`, `TestShellRowsHaveNotes`, `TestResolvableRowsNameASpell` | the schema rules above                                                                                                                               |
| `TestFieldNaming`, `TestFieldNamesRoundTrip`                                                                      | `GoField()` and `TSField()` reproduce protoc's and protobuf-ts's camel case                                                                          |
| `TestRenderProtoNextIndex`, `TestRenderProtoHeader`                                                               | the next free number above each message, and the header protoc reads                                                                                 |
| `TestBuffFilesRegenerateFromTheCommittedInputs`                                                                   | the two Go files, the settings inputs and `proto/buffs.proto` are what the committed store inputs render                                             |
| `TestRenderedBuffFilesMatchTheFixtures`, `TestRenderedBuffFilesCompile`                                           | synthetic rows of every shape render to the committed fixtures, and those fixtures compile against the real `sim/core/buffs` through a build overlay |
| `TestRenderBuffsDebuffsTS*`                                                                                       | the settings inputs each proto type and kind renders                                                                                                 |
| `TestResolvedBuffInvariants`                                                                                      | the pinned talent, categories and stat amounts                                                                                                       |
| `TestScopeMatchesTheClientTargeting`                                                                              | a row whose spell states an area aura, or an aura aimed over an area, sits in the scope that targeting names                                         |
| `TestManifestAnchorsMatchTheClient`                                                                               | with a database, each pinned spell is still the top rank, aura or cast the client's skill lines grant                                                |

The generated constructors' behaviour - categories, stacks, drivers - is held by
`sim/core/buffs_generated_test.go` and `sim/core/debuffs_generated_test.go`. Rewrite the synthetic
fixtures with `UPDATE_BUFF_FIXTURES=1 go test ./tools/database/`.

### Traps

**A negative amount that scales by level floors away from zero.** `Average` floors the folded amount,
so Demoralizing Shout's -196 and -1.4 a level read -205 at 60, and Demoralizing Roar's -193 does too;
a client that truncated the per-level part toward zero would state -204. The store's reading is the one
the sim takes until the game says otherwise.

**A debuff prices at the character's level, not the target's.** `Meta.Options` states `Level`
unconditionally, so `newDebuff` reads the row at `core.CharacterLevel` (60) whatever level the target
carries. Demoralizing Shout's -196 and -1.4 a level reads -205 against a level 60 target and -205 still
against a level 63 raid boss, not the -209 pricing the boss's own level would give: a raid's debuff is
cast by a player of the sim's level, and the target has no character for the parse to read a level off
in the first place.

**`A_PERIODIC_ENERGIZE` and `A_MOD_ATTACK_POWER_PCT` read only for a buff.** Both sit in the table
`BuffAuras()` adds rather than the shared one. Bloodrage reads its rage tick by hand
(`sim/warrior/bloodrage.go`); Berserker Stance Passive's own `A_MOD_ATTACK_POWER_PCT` of 0 is already
parsed onto the stance aura through the shared table (`sim/warrior/stances.go`), which skips and
reports it today because the shared table names no row for it. A shared-table entry for either aura
would start attaching a stat change there instead, changing what the class's own row means.

**Dropping a row renumbers the ones after it.** Each scope's numbers are dense from 1, so a field that
goes away shifts every later number down by one and `TestScopeNumbersAreDense` holds that. Nothing is
live, so no saved payload rides on the old numbers.

**A proto change here breaks whatever a browser holds.** Nothing migrates a saved payload: a field the
manifest retypes makes `fromJson` throw on the enum name an older save spells out. Every `fromJson` of
a settings envelope passes `ignoreUnknownFields`, so a field that only goes away costs nothing.

**Drums are not a Forever consumable.** The client describes no row for the TBC drums - 35476, 35475
and 35478, Battle, War and Restoration, nor their Greater variants - so there is nothing to model and
no manifest row to hang them on. The only "Drums of War" it knows is 1259907, fifteen seconds of party
movement speed, which is no stat buff at all.

**A talent curve only scales the row's first stat.** A row whose talent improves a second amount would
need the generator extended; nothing in the manifest does today.

**A set bonus is not a talent, but it can be a tristate.** The resolver reads `SkillLineAbility`, the
trait trees and the spell effects; it does not read `ItemSetSpell`, so a set that modifies a buff
cannot be a manifest `Talent`. What it can be is the row's `ImpAction`, which is the improved state a
tristate row needs when no trait node prices one, and the icon that state shows. Battlegear of Wrath
is the case: item set 218's `ItemSetSpell` at three pieces (2336) is 23563, an `A_ADD_FLAT_MODIFIER`
of 30 against every effect of the Battle Shout family, which makes the shout worth 169 attack power
rather than 139. The two halves of that are modelled separately. The warrior's own cast reads the
`has_bs_t2` class option - the user's word that this warrior wears the set, not the equipped gear -
and the party's copy is `battle_shout`'s improved state, which `driveBattleShout` reads because the
resolver has no curve to give it. Both call `AddGeneratedFlatBonus`, which raises what the aura
applies and what it bids for its category together, so the stronger of the two copies is the one the
character sheet shows. It is told what the buff is worth without the bonus, because the aura belongs
to the unit rather than to whoever raised it: two warriors in a party wearing the same set ask for
the same total and the second call does nothing. The 30 itself lives in
`buffs.BattleShoutT2Bonus`, with the set and the spell it came from written next to it.

**A party or raid flag means an external caster provides the buff.** The generated apply block builds
the row's `isPlayer=false` copy whenever the proto field is set, so a class port that registers its own
`isPlayer=true` copy has two copies on the character. It either stops setting the flag in
`AddPartyBuffs`/`AddRaidBuffs`, or the row states a `Category` with `SingleAura` and the two copies bid
against each other, so the character sheet shows the buff once. The higher bid deactivates the other
copy; on a tie the incumbent keeps the category when its remaining duration is the longer one, which
is why a druid casting its own Thorns is turned away while the raid's permanent copy is up - both deal
the same 22, so the character strikes back for the same either way. Battle Shout is the worked example:
neither copy is permanent, and the two are worth the same unless one side wears the tier 2 set, so
`TestPlayerBattleShoutTakesTheCategoryOnATie` holds the player's own to the tie and
`TestTheStrongerBattleShoutTakesTheCategory` holds the stronger one to the rest. The rows this decides
are `thorns`, `leader_of_the_pack`, `moonkin_aura` and `trueshot_aura`: `sim/druid/druid.go`,
`sim/druid/feralcat` and `sim/druid/feralbear` raise the party's Leader of the Pack or Moonkin Aura
from a talent, `sim/hunter/hunter.go` raises Trueshot Aura, and the druid's own Thorns waits on the
druid port. `thorns`, `battle_shout`, `leader_of_the_pack` and `moonkin_aura` carry a category, so a
druid registering its own copy of any of them has nothing to decide: the copy joins the same category
and the two bid. The last two share one, `DruidCritAura`, because spell 17007 calls Leader of the
Pack exclusive with Moonkin Aura - a druid's own cast of either joins it, so a party that ticks both
and a druid who casts one are worth the client's 3 and not two threes. Only `trueshot_aura` is left
without a category, so the hunter port has to choose.
