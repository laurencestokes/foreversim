# Crit rage and the warrior racials

On 2026-09-30 Blizzard re-added bonus rage on critical strikes: "Critical strike will now give 75%
more Rage than non-critical strikes." In the sim a critical auto attack now pays 1.75 times the
normalised rage of one that does not crit (`CritRageMultiplier`, `sim/core/rage.go`).

The same racial breakdown (`TestRaceBreakdown` in `sim/warrior/dps`, dual wield, Pre-BiS, identical
weapons, 50,000 iterations, client build 1.60.1.70124), run with the bonus off and on:

| Racial | Without crit rage | With crit rage | Change |
|---|---|---|---|
| Human, Sword Specialization (+2% crit) | +12.7 DPS (+2.41%) | +15.1 (+2.69%) | +19% |
| Night Elf, Elune's Light (+10% crit, 15 s) | +4.2 (+0.80%) | +5.5 (+0.99%) | +31% |
| Orc, Axe Specialization (+1% crit) | +6.5 (+1.22%) | +7.6 (+1.35%) | +17% |
| Dwarf, Mace Specialization (+1% crit) | +6.4 (+1.22%) | +7.5 (+1.34%) | +17% |
| Undead, Touch of the Grave | +11.0 (+2.09%) | +11.4 (+2.03%) | flat |
| Tauren, Endurance (+1% hit) | +4.4 (+0.83%) | +3.8 (+0.67%) | down |
| Gnome, Eureka! | +5.6 (+1.05%) | +4.4 (+0.79%) | down |

Crit now pays twice, in damage and in the rage that buys more Heroic Strikes, so every crit racial is
worth more; Elune's Light most, because its 10% lands while the warrior's other cooldowns are up.
Eureka!'s rage discount and Tauren's hit matter less once rage is plentiful.

## Bonus off

Baseline: Human, 527.5 ± 0.1 DPS (50000 iterations per run). Differences under about 0.4 DPS are noise, shown as ≈0.

| Race | Total vs baseline | Base stats | Passive racials | Cooldown racial | Weapon type (no racial) | Weapon racial |
|---|---|---|---|---|---|---|
| Human | +12.7 (+2.41%) | +0.0 (≈0) | +0.0 (≈0) | - | +0.0 (≈0) | +12.7 (+2.41%) |
| Orc | +11.6 (+2.21%) | +0.3 (≈0) | +0.0 (≈0) | +4.9 (+0.92%) | +0.0 (≈0) | +6.5 (+1.22%) |
| Undead | +10.0 (+1.89%) | -1.0 (-0.19%) | +11.0 (+2.09%) | - | - | - |
| Dwarf | +6.0 (+1.13%) | -0.5 (-0.09%) | +0.0 (≈0) | - | +0.0 (≈0) | +6.4 (+1.22%) |
| Tauren | +4.8 (+0.91%) | +0.4 (+0.08%) | +4.4 (+0.83%) | - | - | - |
| Skyborne (either) | +4.8 (+0.91%) | +0.2 (≈0) | +4.5 (+0.86%) | - | - | - |
| NightElf | +4.5 (+0.86%) | +0.3 (≈0) | +0.0 (≈0) | +4.2 (+0.80%) | - | - |
| Gnome | +4.4 (+0.84%) | -1.1 (-0.22%) | +0.0 (≈0) | +5.6 (+1.05%) | - | - |
| Troll | +3.9 (+0.74%) | +1.4 (+0.27%) | +0.0 (≈0) | +2.4 (+0.46%) | - | - |

Orc, the other order: weapon racial alone +6.4 (+1.21%), cooldown racial on top of it +4.9 (+0.94%).

## Bonus on

Baseline: Human, 561.8 ± 0.1 DPS (50000 iterations per run). Differences under about 0.4 DPS are noise, shown as ≈0.

| Race | Total vs baseline | Base stats | Passive racials | Cooldown racial | Weapon type (no racial) | Weapon racial |
|---|---|---|---|---|---|---|
| Human | +15.1 (+2.69%) | +0.0 (≈0) | +0.0 (≈0) | - | +0.0 (≈0) | +15.1 (+2.69%) |
| Orc | +12.8 (+2.27%) | +0.1 (≈0) | +0.0 (≈0) | +5.1 (+0.90%) | +0.0 (≈0) | +7.6 (+1.35%) |
| Undead | +10.2 (+1.81%) | -1.2 (-0.22%) | +11.4 (+2.03%) | - | - | - |
| Dwarf | +6.8 (+1.22%) | -0.7 (-0.12%) | +0.0 (≈0) | - | +0.0 (≈0) | +7.5 (+1.34%) |
| NightElf | +6.0 (+1.07%) | +0.5 (+0.09%) | +0.0 (≈0) | +5.5 (+0.99%) | - | - |
| Skyborne (either) | +4.7 (+0.84%) | +0.3 (≈0) | +4.4 (+0.79%) | - | - | - |
| Troll | +4.1 (+0.73%) | +1.6 (+0.28%) | +0.0 (≈0) | +2.5 (+0.45%) | - | - |
| Tauren | +4.0 (+0.71%) | +0.2 (≈0) | +3.8 (+0.67%) | - | - | - |
| Gnome | +3.3 (+0.59%) | -1.1 (-0.19%) | +0.0 (≈0) | +4.4 (+0.79%) | - | - |

Orc, the other order: weapon racial alone +7.5 (+1.34%), cooldown racial on top of it +5.1 (+0.91%).
