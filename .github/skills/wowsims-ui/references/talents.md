# Talents

**Source of truth:** `ui/sim/talents/talents_string.ts` (the string codec),
`ui/sim/talents/broken_talents.ts` (the tier and prerequisite rules), `ui/sim/talents/config.ts`
(the tree shape and `newTalentsConfig`), `ui/features/talents/model/can_set_points.ts` (which clicks
are accepted), `ui/features/talents/model/tree_graph.ts` (prerequisite arrows and grid placement) and
the views in `ui/features/talents/components/TalentsPicker/`. Each file opens with a comment that
states its rule; read it there if this file and the code disagree:

```
sed -n '1,12p' ui/sim/talents/talents_string.ts ui/sim/talents/broken_talents.ts ui/features/talents/model/can_set_points.ts
```

## The talents string

- One run of digits per tree, joined with `-`. Digit `i` of tree `t` is `config[t].talents[i]`, in the
  order the config lists the talents, **not** a row or column index.
- Trailing zeroes in each tree and trailing dashes are trimmed, so an empty middle tree reads
  `2500052300030150330125--053500031003001`. `serializeTalentsString(parseTalentsString(s))` hands a
  real preset back byte for byte; `TalentsPicker.test.tsx` holds that.
- The string is the only stored form: `SavedTalents` and every preset carry it and nothing else (see
  `references/conventions.md`). `Player.getTalents()` parses it into the spec's talents proto on
  demand.

## The tree rules

- **Tier:** a talent in row N needs 5 × N points in the rows **above** it. Points in its own row do
  not count.
- **Prerequisite:** a talent with `prereqLocation` needs that talent at its maximum rank. A
  prerequisite is usually in an earlier row, but the game files also put one beside its child in
  the same row, on either side: Paladin Holy's Divine Precision (row 4, column 0) needs Holy Shock
  (row 4, column 1).
- **Order:** `newTalentsConfig` throws unless a tree lists its talents row by row, left to right.
  `findBrokenTalents` relies on that to walk each tree in one pass.
- **Cap:** `MAX_POINTS_PLAYER` is `CHARACTER_LEVEL - 9`, across all three trees.

## Broken talents

A point can always be taken back, even when that leaves a talent in a lower row short of its tier
or a child without a full prerequisite. That talent is **broken**:

- It keeps its points in the string, so putting the missing point back restores the build.
- Its points unlock nothing: lower rows count only points in talents that are not broken, and a
  broken prerequisite breaks its children. The breakage cascades down the tree.
- The picker draws it with a red border, glow and rank badge (`data-broken="true"`).
- The sidebar lists it through the `sidebar.warnings.broken_talents` warning in
  `ui/features/results/model/default_warnings.ts`, which reads `Player.getBrokenTalentNames()`.
- **The sim still runs and still applies it.** This is deliberate: taking points out of a talent
  without losing the talents below it is how a build is tested without that talent. Do not add a run
  guard for broken talents.

A point is only added where the talent would not be broken afterwards, so a broken talent cannot
take another point until its gap is filled.

## The picker

- **Input:** left mousedown spends a point, right mousedown takes one back, and the context menu is
  suppressed so the right click reaches it. Mousedown commits, not click, so the click only cancels
  the Wowhead link. A tap spends a point and wraps a full talent back to 0; a tap on a broken talent
  clears it, since it cannot take another point. A long press (`LONG_TOUCH_MS`) clears the talent.
- **Touch listeners are native**, attached once in an effect and reading the current handlers
  through a ref: React attaches `touchstart` as passive, where `preventDefault` does nothing.
- **A rejected click still fires `onChange`** with the unchanged points; the facade's equality guard
  turns it into a no-op write. Tests assert on the last written string, not on whether a write
  happened.
- **Layout:** each tree is ONE CSS grid. Talents and prerequisite arrows are both grid items placed by
  `gridRow`/`gridColumn`, and an arrow spans the cells from parent to child. The z-index ladder (`ROOT_Z_INDEX` at
  a root, each child 2 lower, each arrow 1 lower than its parent) makes a deeper chain paint under the one
  above it.
- **Arrows** light when the child can take a point, or when it holds points and is not broken. A
  child holding points proves its prerequisite is full, which matters once every point is spent and no
  talent can take a point.
- **Background art** fills a wrapper below the header, so it starts where the header ends whatever
  height the header has. Do not position it at a fixed offset from the top of the tree.
- **Icons** are looked up at rank 1 whatever the points, so spending a point does not re-seed the
  lookup and blank the cell for a frame. Only the Wowhead link carries the rank.
- **The trees** sit side by side from `lg` up. Below `lg` they scroll-snap horizontally, and the
  picker scrolls the middle tree into view on mount.

## Styling states

`TalentPicker.css` keys every state off data attributes on `.ui-talent-picker-root`:

| State                         | Attribute                                | Look                            |
| ----------------------------- | ---------------------------------------- | ------------------------------- |
| Empty and cannot take a point | `data-points="0"` without `data-can-add` | greyscale icon                  |
| Partly spent                  | (default)                                | green border and badge          |
| Full                          | `data-full="true"`                       | gold border and badge           |
| Broken                        | `data-broken="true"`                     | red border, glow and bold badge |
| Empty with every point spent  | `data-points="0"` and `data-all-spent`   | badge hidden                    |

Broken uses no `filter`. A filter applies to the whole element, border and badge included, so a
greyscale filter would turn the red grey and a broken talent would look the same as one that cannot
take a point yet.

## Tips

- **Check a new or changed talent preset for broken talents.** A preset that breaks one shows the
  sidebar warning and a red talent as soon as it loads. A scratch test prints what is wrong:

    ```ts
    brokenTalentNames(config, parseTalentsString(config, presetString));
    ```

    Count the points too: `totalPointsSpent` under `MAX_POINTS_PLAYER` also raises the unspent points
    warning.

- **Write fixtures that tell the rules apart.** A tier test passes by accident when the raw point
  count is under the threshold either way. To prove broken points do not count, put the raw count
  over the threshold (`'4000' + '55'`: 14 points above row 2, only 4 of them live). To cover the
  arrow's `!broken` clause, use a _full_ broken child, not a partly spent one.
- **Unit tests build their own trees** with `newTalentsConfig` (`can_set_points.test.ts`,
  `broken_talents.test.ts`, `TalentTreePicker.test.tsx`), so a game files change cannot move them. `TalentTreePicker.test.tsx` mocks `useActionId`, since no unit test
  has the database.
- **Stories:** `TalentTreePicker` and `TalentPicker` each have a `Broken` story. Both opt out of the
  smoke test (`smoke: false`) because the icons come from the database.
