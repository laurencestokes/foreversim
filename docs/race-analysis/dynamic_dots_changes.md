# What dynamic DoTs changed in the race analysis

Damage over time effects now recompute the caster's spell power, attack power and damage bonuses
every tick (`core.DynamicDoTs`, see `docs/forever_rules.md`). The breakdowns and Eureka! splits in
this folder were re-run at 50,000 iterations each on the same builds; the tables below compare them
with the snapshotted runs they replace. Percentages are vs each table's baseline race; noise is about
±0.05 points.

## Warlock racial totals (vs an Orc with Blood Fury off)

| Build | Human (1H sword) | Gnome | Undead | Orc (Blood Fury) | Troll |
|---|---|---|---|---|---|
| Deep Affliction | +1.93 → +2.00 | +1.78 → +1.56 | +1.48 → +1.47 | +0.66 → +0.45 | +0.34 → +0.33 |
| DS/Ruin Pandemic | +2.39 → +2.55 | +1.92 → +1.84 | +1.81 → +1.86 | +0.73 → +0.54 | +0.38 → +0.32 |
| Shadow and Flame | +2.14 → +2.32 | +1.87 → +1.64 | +1.98 → +2.03 | +0.69 → +0.53 | +0.30 → +0.18 |
| Demonic Pact | +1.81 → +1.94 | +2.04 → +1.54 | +1.45 → +1.52 | +0.73 → +0.50 | +0.12 → +0.15 |

- **Gnome falls** because Eureka!'s 10% no longer rides the opening DoTs for their whole duration; it
  reaches running DoTs only while Eureka! is up, which for a warlock is the few seconds until the third
  charge is spent.
- **Orc falls** for the same reason: Blood Fury's 10% spell power no longer stays in DoTs applied during
  it, it counts only for the ticks inside its 15 seconds.
- Human and Undead barely move (their racials are crit on every cast and a proc on hits).

## Eureka! split, warlock (whole / cost cut alone / damage bonus alone)

| Build | Before | After |
|---|---|---|
| Deep Affliction | +1.35% / +0.36% / +0.99% | +1.22% / +0.74% / +0.49% |
| DS/Ruin Pandemic | +1.41% / +0.43% / +0.99% | +1.19% / +0.78% / +0.43% |
| Shadow and Flame | +1.37% / +0.49% / +0.88% | +1.30% / +0.85% / +0.44% |
| Demonic Pact | +1.56% / +0.71% / +0.85% | +1.15% / +0.78% / +0.36% |

The damage half roughly halves, as expected. The cost-cut half roughly doubles, which this rule change
does not obviously explain; it is recorded here as observed and not yet understood.

## Unchanged

- **Rogue** breakdowns and Eureka! split: every figure within ±0.01 points. The rogue's race-sensitive
  damage is strikes and poisons, and Rupture/Garrote carry little of it.
- **Warrior** Eureka! split: identical. Rend was already computed live and Deep Wounds carries a fixed
  share of the crit that triggered it.
