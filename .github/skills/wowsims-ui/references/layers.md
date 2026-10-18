# Layers: where code goes and what may import what

**Source of truth:** `.oxlintrc.json` (the bans) and `tsconfig.json` `paths` (the aliases). `npm run lint:js` and
`npm run type-check` enforce both on every run, so trust them over this file. Dump the current bans:

```
node -e "const c=require('./.oxlintrc.json'); c.overrides.forEach(o => { for (const k of ['no-restricted-imports','no-restricted-globals']) if (o.rules?.[k]) console.log(JSON.stringify(o.files), k, JSON.stringify(o.rules[k])); })"
```

`ui/README.md` ("Dependency direction", "Placement rules") has the orientation prose. Nothing enforces the README, so
where it and `.oxlintrc.json` disagree, the lint config is right.

## The bans, as configured

| Scope                       | May not import                                                                                       | Notes                                                                                                                                                                                                                 |
| --------------------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ui/sim/**`                 | `@ui-kit` `@features` `@app` `@specs`, `@testing`                                                    | **`allowTypeImports: true`**, see below                                                                                                                                                                               |
| `ui/i18n/**/*.{ts,tsx}`     | `@app` `@features` `@ui-kit` `@specs/**`, `@testing`                                                 | `@sim` is allowed; no type-import escape hatch                                                                                                                                                                        |
| `ui/ui-kit/**`              | `@features` `@app` `@specs`, `react-dom/client`, `@sim` except `utils/**` `constants/**`, `@testing` | only the shell creates a React root; the `@sim` group is below                                                                                                                                                        |
| `ui/features/**`            | `@app` `@specs`, `react-dom/client`, `@testing`                                                      |                                                                                                                                                                                                                       |
| `ui/features/*/model/**`    | `@app` `@specs`, `react` and `react-dom`, `@testing`                                                 | a model is framework-free, not only DOM-free                                                                                                                                                                          |
| `ui/features/**/*.{ts,tsx}` | `patchSlice` `patchKeyed` `seedKeyed` `deleteKeyed` from `**/state/sim_store*`                       | write through a facade                                                                                                                                                                                                |
| `ui/features/**/*.test.*`   | `@app` `@specs`, `react-dom/client`                                                                  | a test may seed the store with the writers; a model test may use React                                                                                                                                                |
| `ui/app/**`                 | `@specs`, `@testing`                                                                                 | specs import the app, never the reverse                                                                                                                                                                               |
| tests and stories           | the bans of their layer, without `@testing`                                                          | `ui/{sim,i18n}/**/*.test.*`, `ui/{ui-kit,app}/**/*.{test,stories}.*` and `ui/features/**/*.stories.tsx` restate their layer's groups; the two features test blocks also drop the store writers, as the row above says |
| `ui/design-system/**`       | `@features` `@app` `@specs`                                                                          | the docs pages sit beside the kit: `@sim` and `@ui-kit` allowed                                                                                                                                                       |
| `ui/testing/**`             | `@app` `@specs`                                                                                      | the story fixtures and decorators may reach features                                                                                                                                                                  |

`no-restricted-globals` bans `window` `document` `localStorage` `location` `navigator` in `ui/sim/**` and in
`ui/features/*/model/**`. Code there that needs one takes the injected `Env` (`ui/sim/state/env.ts`): `sim.env` at
runtime, `ui/app/browser_env.ts` in the browser, `memoryEnv` (`ui/testing/sim/memory_env.ts`) in tests.

The kit's `@sim` group is a positive list, because a negated glob inside a group is unproven in oxlint: it names every
`ui/sim` directory and top-level module except `utils` and `constants`. A new top-level entry under `ui/sim` has to join
it, and a directory with an `index.ts` needs both `@sim/<dir>` and `@sim/<dir>/**`, since the bare specifier resolves
to it. A store binding belongs in `ui/features/inputs`, which is what the ban's message says.

**Naming a DOM type is not being a view.** A sim-layer function that takes an `HTMLElement` it is handed and enriches it
(an icon, an href, a Wowhead dataset) never creates an element and touches no browser global, so it satisfies the rule.
The test for `ui/sim` is browser globals and element creation, not whether `HTMLElement` appears in a signature, so
leave such a function where it is.

## Where tests and fixtures go

Only `ui/specs/` itself may import `@specs`: every layer block above bans it. A test that needs every real spec config
(the shipped defaults, the display stats of each spec) therefore lives in `ui/specs/*.test.ts(x)`, not beside the code
it tests.

A fixture or stub that a test or story needs lives with the tests, never in production code (`AGENTS.md`):
`ui/testing/sim/` for sim-level helpers that import neither `@features` nor `@ui-kit` (`makeIndividualSim` in
`ui/testing/sim/individual_sim.ts`, `memoryEnv`, the stat-filter helpers), `ui/testing/storybook/fixtures/` for story
fixtures, `ui/testing/react.tsx` for `renderWithHost`. A test-only export, prop or barrel re-export added to `ui/` breaks
that rule even when it is small.

## Four ways these rules surprise people

**`import type` helps in `ui/sim/**` and nowhere else.** Only that block sets `allowTypeImports: true`, so `ui/sim` may
name an `@app`/`@features`/`@ui-kit` type as long as the import is erased. `no-restricted-imports` bans the specifier,
not the binding, so `import type { SimHostObject } from '@app/individual_sim_ui'` inside `ui/ui-kit` is still a
violation although nothing survives to runtime.

**Patterns must use `**`, not `*`.** oxlint matches one path segment per `*`, so `@features/*` silently misses
`@features/gear/components/SelectorModal/ItemList` and the ban looks as if it works.

**Scoping is an explicit file list.** Negated globs in `overrides.files` break scoping in oxlint, so a new top-level
`ui/` directory needs its own override block. And `categories` is `{"correctness": "off"}`: a rule that is not listed
by name does not run, which is why `react-hooks/rules-of-hooks` and `react-hooks/exhaustive-deps` sit in the root
`rules` block. Deleting either line silently disables it.

**A later block replaces a rule; it does not add to it.** When two blocks match a file and both set
`no-restricted-imports`, the later block's options win whole. `excludeFiles` exists, but a block that excludes the
tests leaves them with none of its bans. So the `@testing` ban sits in each production block, and each test and story
block after it repeats its layer's groups without it. A new group in a layer block goes into that layer's test and story
blocks too.

## Aliases

`tsconfig.json` `paths` maps `@<layer>/*` to `ui/<layer>/*` for every layer above plus `@testing`. The same table is
`UI_ALIASES` in `vite.config.mts`, which the site build, the worker build, `vitest.config.mts` and
`.storybook/vite.config.ts` spread into `resolve.alias`. The two must agree: a mismatch shows up as a type-check that
passes and a build that cannot resolve.

```
node -e "console.log(require('./tsconfig.json').compilerOptions.paths)"
```

The aliases are `tsconfig` `paths` rather than `package.json` `imports` because `tsc` under
`moduleResolution: "bundler"` does not resolve `#foo/*` subpaths.

## Crossing layers without importing downward

A lower layer that needs the shell uses a narrow host interface, never the class. They live in `ui/sim/sim_host.ts`:
`SimWarning`, `SimHost`, `IndividualSimHost<Spec>`, and `isIndividualSimHost()`, which stands in for
`instanceof SimHostObject`. `SimHostObject implements IndividualSimHost` (`ui/app/individual_sim_ui.tsx`), so add to the
interface and let the compiler find the host. From React the host arrives through context: `useSimHost()`, `usePlayer()`
and `useSim()` in `ui/sim/context/SimHostContext.tsx`, which is why `ui/sim` owns React context although it is the
bottom layer.

The per-spec config schema is `ui/sim/spec_config.ts` (`IndividualSimUIConfig`, `InputConfig`, `CustomSection`,
`SpecDefinition`, `defineSpec`, …). It names the picker configs of `ui/features/inputs/` and the kit's `SavedDataConfig`
through `import type` only, which the `ui/sim/**` block permits; a value import there would break the build. List what
`ui/sim` names upward:

```
/usr/bin/grep -rn "@ui-kit\|@features\|@app" ui/sim --include=*.ts --include=*.tsx | /usr/bin/grep -v "\.test\."
```

## The module-evaluation-order trap

`ui/sim/player/specs/<class>.ts` calls `getSpecSitePath()` at module scope to build each `PlayerSpec`'s `simLink`. If
one of those files imports a value from a non-leaf module, the resulting cycle lets a bundler evaluate
`ui/sim/player/specs/index.ts` before the class modules it aggregates, `specToPlayerSpec` is built from `undefined`s,
and `PlayerSpecs.fromProto` silently returns `undefined`. That surfaces as a crash inside `new Player`, far from the
cause. So `player/specs/*` and `player/classes/*` import leaf modules only.

`import/no-cycle` is not enabled on purpose: it reports dozens of cycles, most in `ui/sim`, which is a triage project
rather than a switch to flip. Cycles are caught by review. Count them before arguing either way:

```
RTK_DISABLED=1 npx oxlint --import-plugin -D import/no-cycle ./ui 2>&1 | /usr/bin/grep -c 'import(no-cycle)'
```

## Moving a file

There is no move tool; `ui/README.md` ("How to move a file") has the steps. The part that bites is that a move passes
`type-check` and can still fail at runtime, so run every gate in `verification.md` and load a spec page afterwards.
