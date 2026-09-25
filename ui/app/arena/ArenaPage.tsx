// The build arena: every build the sim can be given, ranked, computed ahead of time.
//
// The live rankings page runs one preset per spec in your browser, which is the ceiling of
// what a browser can be asked for and a long way short of a leaderboard. So the builds run
// headless and this page renders what they produced, with no sim in the browser at all. The
// results file is the cache and the commit that produced it is the key.

import { IndividualSimSettings } from '@generated/proto/ui';
import { CURRENT_API_VERSION } from '@sim/constants/other';
import type { PlayerSpec } from '@sim/player/player_spec';
import { PlayerSpecs } from '@sim/player/specs';
import { textClassNameForSpec } from '@sim/proto/utils';
import type { Composition } from '@sim/spells/rests';
import { classTalentsConfig } from '@sim/talents/factory';
import { formatToNumber, formatToPercent } from '@sim/utils/format';
import clsx from 'clsx';
import pako from 'pako';
import { useMemo, useState } from 'react';

import { PageSection, ProductPage, SITE_BASE, SITE_REPO_URL } from '../ProductPage';
import { RestsCell } from '../RestsCell';
import results from './results.json';

// Results are keyed by master's ui/ directory names, which is what the arena runner writes.
// The join to forever-next's specs happens here. Both priest builds are forever-next's one
// DPS priest spec, so the name is carried rather than derived.
const SPECS: Record<string, { spec: PlayerSpec<any>; name: string }> = {
	balance_druid: { spec: PlayerSpecs.BalanceDruid, name: 'Balance Druid' },
	feral_druid: { spec: PlayerSpecs.FeralCatDruid, name: 'Feral Druid' },
	feral_tank_druid: { spec: PlayerSpecs.FeralBearDruid, name: 'Feral Tank Druid' },
	elemental_shaman: { spec: PlayerSpecs.ElementalShaman, name: 'Elemental Shaman' },
	enhancement_shaman: { spec: PlayerSpecs.EnhancementShaman, name: 'Enhancement Shaman' },
	hunter: { spec: PlayerSpecs.Hunter, name: 'Hunter' },
	mage: { spec: PlayerSpecs.Mage, name: 'Mage' },
	protection_paladin: { spec: PlayerSpecs.ProtectionPaladin, name: 'Protection Paladin' },
	retribution_paladin: { spec: PlayerSpecs.RetributionPaladin, name: 'Retribution Paladin' },
	rogue: { spec: PlayerSpecs.Rogue, name: 'Rogue' },
	shadow_priest: { spec: PlayerSpecs.DpsPriest, name: 'Shadow Priest' },
	smite_priest: { spec: PlayerSpecs.DpsPriest, name: 'Smite Priest' },
	tank_warrior: { spec: PlayerSpecs.ProtectionWarrior, name: 'Protection Warrior' },
	warlock: { spec: PlayerSpecs.Warlock, name: 'Warlock' },
	warrior: { spec: PlayerSpecs.DpsWarrior, name: 'Warrior' },
};

type Build = {
	spec: string;
	build: string;
	talents: string;
	gear: string;
	rotation: string;
	consumables: string;
	dps: number;
	rests: Composition;
	ilvl: number;
	slots: number;
	optimised?: boolean;
	points?: number;
};

type Results = { sim?: string; commit: string; generated: string; iterations: number; host?: string; builds: Array<Build> };
const data = results as Results;
const byDps = [...data.builds].sort((a, b) => b.dps - a.dps);

// The arena runs every talent build on every gear set, but this page compares talents, and specs,
// so every row wears its spec's launch set: the best pre-raid gear from one shared item pool,
// built by the same rule for every spec (tools/launch_gear), all within a few item levels of
// each other. A spec with two (rogue daggers or swords, warrior dual wield or two-hander) lets
// each build take the better. Each talent build appears once, in its best rotation.
const isLaunchSet = (gear: string) => gear === 'launch' || gear.endsWith('_launch');
const builds = (() => {
	const seen = new Set<string>();
	return byDps.filter(build => {
		const talents = `${build.spec}|${build.talents}`;
		if (!isLaunchSet(build.gear) || seen.has(talents)) return false;
		seen.add(talents);
		return true;
	});
})();

// A sim link carrying only the talents (?i=t), so opening it keeps the visitor's own gear and settings.
export const talentLink = (spec: PlayerSpec<any>, talents: string) => {
	const settings = IndividualSimSettings.create({ apiVersion: CURRENT_API_VERSION, player: { talentsString: talents } });
	const bytes = pako.deflate(IndividualSimSettings.toBinary(settings));
	return `${spec.simLink}?i=t#${btoa(String.fromCharCode(...bytes))}`;
};

// The tree a build puts most points in, which is what "a Fury build" means. A tie goes to the
// earlier tree.
const mainTree = (build: Build) => {
	const points = split(build.talents).split('/').map(Number);
	return points.indexOf(Math.max(...points));
};
const treeName = (build: Build) => {
	const spec = SPECS[build.spec]?.spec;
	return spec ? (classTalentsConfig[spec.classID as keyof typeof classTalentsConfig]?.[mainTree(build)]?.name ?? '') : '';
};

// The default view: each spec's best build in each of its trees, so the Arms, Fury and
// Protection answers all show rather than only whichever tree wins. from is best first, so the
// first row seen for a spec and tree is its best.
export const bestPerTree = (from: Array<Build>): Array<Build> => {
	const best = new Map<string, Build>();
	for (const build of from) {
		const at = `${build.spec}|${mainTree(build)}`;
		if (!best.has(at)) best.set(at, build);
	}
	return [...best.values()].sort((a, b) => b.dps - a.dps);
};

// What the talent search was worth on each searched row, against the best written build on the
// exact same gear set and rotation: across gear sets it would be part search, part item level.
const key = (build: Build) => `${build.spec}|${build.gear}|${build.rotation}`;
const gains = (() => {
	const written = new Map<string, number>();
	for (const build of data.builds.filter(b => !b.optimised)) written.set(key(build), Math.max(written.get(key(build)) ?? 0, build.dps));
	const out = new Map<string, number>();
	for (const searched of data.builds.filter(b => b.optimised)) {
		const base = written.get(key(searched));
		if (base) out.set(`${key(searched)}|${searched.talents}`, searched.dps / base - 1);
	}
	return out;
})();

// The x/y/z everyone reads a build as, straight off the talents string.
const split = (talents: string) => {
	const trees = talents.split('-');
	while (trees.length < 3) trees.push('');
	return trees
		.slice(0, 3)
		.map(tree => [...tree].reduce((total, char) => total + (parseInt(char) || 0), 0))
		.join('/');
};

// A searched build keeps the name of the build it started from; a point split in that name is
// no longer true of it, so it is dropped and the real split printed beside it.
const displayName = (build: Build) => (build.optimised ? build.build.replace(/\s*\d+\/\d+\/\d+/, '') : build.build);
const specName = (spec: string) => SPECS[spec]?.name ?? spec;
const consumablesName = (list: string) => (list || 'unknown consumables').replace('Arena-', '').replace('+class', ' + class imbues').toLowerCase();

const ALL_SPECS = [...new Set(builds.map(b => b.spec))].sort((a, b) => specName(a).localeCompare(specName(b)));

const PILL = 'inline-flex cursor-pointer items-center gap-1 rounded-full border px-2.5 py-1 text-sm';
const TAG = 'mt-0.5 cursor-help self-start rounded-full border px-1.5 text-xs';
const CELL = 'px-2 py-1 text-left align-middle whitespace-nowrap';
const SELECT = 'rounded-sm border border-white/18 bg-black px-2 py-1 text-sm text-gray-300';

const Row = ({ build, rank, top }: { build: Build; rank: number; top: number }) => {
	const spec = SPECS[build.spec]?.spec;
	const color = spec ? textClassNameForSpec(spec) : 'text-white';
	const share = (build.dps / top) * 100;
	const gain = gains.get(`${key(build)}|${build.talents}`);

	return (
		<tr className="even:bg-white/3" data-testid="arena-row">
			<td className={clsx(CELL, 'w-[3ch] text-right text-white/50 tabular-nums')}>{rank}</td>
			<td className={CELL}>
				{spec && <img className="mr-2 inline-block size-8 align-middle" src={spec.getIcon('medium')} alt="" />}
				<span className="inline-flex flex-col align-middle">
					<span className="text-xs text-white/60">{specName(build.spec)}</span>
					<span className={clsx('font-semibold', color)}>{displayName(build)}</span>
					<span className="text-xs text-white/50 tabular-nums">
						{split(build.talents)} {treeName(build)}
						{spec && (
							<>
								{' · '}
								<a
									className="text-brand hover:underline"
									href={talentLink(spec, build.talents)}
									target="_blank"
									rel="noreferrer"
									title={`${build.talents}

Opens the sim with these talents. Your gear and other settings are kept.`}
									data-testid="arena-talents-link">
									open talents in the sim
								</a>
							</>
						)}
					</span>
					{build.optimised && (
						<span
							className={clsx(TAG, 'border-brand text-brand')}
							title={`${build.talents}\n\nClimbed from every distinct build this spec has on file for this gear and rotation.`}>
							found by search
							{gain !== undefined && ` ${formatToPercent(gain * 100, { maximumFractionDigits: 1, signDisplay: 'always' })}`}
						</span>
					)}
					{!!build.points && (
						<span className={clsx(TAG, 'border-danger text-danger')} title="This build does not spend every talent point a level 60 character has.">
							{build.points} of 51 points
						</span>
					)}
				</span>
			</td>
			<td className={clsx(CELL, 'text-sm')}>
				<span className="block text-gray-300">{build.gear}</span>
				<span className="block text-white/50">{build.rotation || 'default rotation'}</span>
				{build.rotation.endsWith('_lowrank') && (
					<span
						className={clsx(TAG, 'block border-brand text-brand')}
						title="Casts lower spell ranks to save mana. The sim gives lower ranks full spell-power scaling, as the beta does at level 20; whether Forever penalises them at level 60 is not yet known.">
						low ranks, unconfirmed at 60
					</span>
				)}
				<span
					className="block text-white/50"
					title="The consumable list this build drank. Every spec in a role drinks the same one; it is set by the arena, not by the spec.">
					{consumablesName(build.consumables)}
				</span>
			</td>
			<td className={clsx(CELL, 'text-right tabular-nums')}>
				<span className="block">{build.ilvl ? build.ilvl.toFixed(1) : '?'}</span>
				{!!build.slots && build.slots < 15 && (
					<span className="block cursor-help text-xs text-danger" title="This gear set leaves slots empty, so the character is not fully equipped.">
						{build.slots} slots
					</span>
				)}
			</td>
			<td className={clsx(CELL, 'text-right tabular-nums')}>{formatToNumber(build.dps, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}</td>
			<td className={clsx(CELL, 'w-full min-w-40')}>
				<div className="flex items-center gap-2">
					<div className="relative h-3 min-w-10 flex-1 bg-white/5">
						<div className={clsx('absolute inset-y-0 left-0 bg-current', color)} style={{ width: `${share}%` }} />
					</div>
					<span className="w-[5ch] shrink-0 text-right tabular-nums">{formatToPercent(share, { maximumFractionDigits: 1 })}</span>
				</div>
			</td>
			<td className={CELL}>
				<RestsCell rests={build.rests} />
			</td>
		</tr>
	);
};

const Controls = ({
	showAll,
	setShowAll,
	spec,
	setSpec,
}: {
	showAll: boolean;
	setShowAll: (value: boolean) => void;
	spec: string;
	setSpec: (value: string) => void;
}) => (
	<div className="flex flex-wrap items-center gap-x-4 gap-y-2">
		<button className={clsx(PILL, 'border-brand text-white hover:bg-brand/15')} type="button" onClick={() => setShowAll(!showAll)}>
			{showAll ? 'Show the best build in each tree' : 'Show every talent build'}
		</button>
		<select className={SELECT} value={spec} onChange={event => setSpec(event.target.value)} aria-label="Spec">
			<option value="">Every spec</option>
			{ALL_SPECS.map(s => (
				<option key={s} value={s}>
					{specName(s)}
				</option>
			))}
		</select>
	</div>
);

const Leaderboard = () => {
	const [showAll, setShowAll] = useState(false);
	const [spec, setSpec] = useState('');

	const shown = useMemo(() => {
		const kept = builds.filter(b => !spec || b.spec === spec);
		return showAll ? kept : bestPerTree(kept);
	}, [showAll, spec]);
	const top = shown[0]?.dps || 1;
	const specCount = new Set(shown.map(b => b.spec)).size;

	// The one thing a ranking table has to admit when it is not true: that the rows are not
	// wearing comparable gear. Three item levels is about a fifth of a tier.
	const levels = shown.map(b => b.ilvl).filter(ilvl => ilvl > 0);
	const low = Math.min(...levels);
	const high = Math.max(...levels);
	const wide = levels.length > 1 && high - low > 3;

	return (
		<div className="flex flex-col gap-3">
			<Controls {...{ showAll, setShowAll, spec, setSpec }} />
			<p className="m-0 text-white/50" data-testid="arena-count">
				{showAll
					? `All ${shown.length} talent builds, best first. Each spec wears one gear set throughout, so within a spec only the talents differ.`
					: `The best build in each talent tree of ${specCount} spec${specCount === 1 ? '' : 's'}: ${shown.length} builds, best first.`}
			</p>
			{levels.length > 0 && (
				<p className={clsx('m-0 text-sm', wide ? 'text-brand' : 'text-white/50')} data-wide={wide || undefined}>
					These rows span item level {low.toFixed(1)} to {high.toFixed(1)}
					{wide ? ', so some of the gap between them is gear rather than spec.' : '.'}
				</p>
			)}
			<table className="block w-full border-collapse overflow-x-auto">
				<thead>
					<tr className="border-b border-surface-border">
						<th className={clsx(CELL, 'text-right')}>#</th>
						<th className={CELL}>Build</th>
						<th className={CELL}>Gear and rotation</th>
						<th className={clsx(CELL, 'text-right')}>ilvl</th>
						<th className={clsx(CELL, 'text-right')}>DPS</th>
						<th className={CELL}>Share of top</th>
						<th className={CELL}>Rests on a guess</th>
					</tr>
				</thead>
				<tbody>
					{shown.map((build, index) => (
						<Row key={`${key(build)}|${build.talents}|${build.consumables}|${build.build}`} build={build} rank={index + 1} top={top} />
					))}
				</tbody>
			</table>
		</div>
	);
};

const SimSource = () => {
	const commit = data.commit ? data.commit.slice(0, 7) : 'unknown';
	const commitLink = data.commit ? (
		<a className="text-brand hover:underline" href={`${SITE_REPO_URL}/commit/${data.commit}`} target="_blank" rel="noreferrer">
			<code>{commit}</code>
		</a>
	) : (
		<code>{commit}</code>
	);
	return (
		<>
			Computed from sim {commitLink} on {data.generated.slice(0, 10)}
			{data.host ? `, on ${data.host}` : ''}.{' '}
			{data.sim === 'master' ? (
				<strong className="text-brand" data-testid="arena-sim">
					These are the old master sim&apos;s numbers (the Classic Era engine this site used to run), not the forever-next engine the rest of this
					site now runs on. They move to forever-next&apos;s once the arena is rerun on it.
				</strong>
			) : (
				<span data-testid="arena-sim">
					Produced by the {data.sim ?? 'forever-next'} sim.
					{builds.every(b => b.gear.startsWith('parity')) &&
						' Until the arena runner is ported to it, each row is one build per spec from the parity check (tools/parity): equal bonus stats and statless weapons, no gear set and no consumables beyond what a class grants itself, so the gear, bracket and search columns have nothing to show yet.'}
				</span>
			)}
		</>
	);
};

export const ArenaPage = () => (
	<ProductPage
		title="The build arena"
		subtitle={`Every talent build this sim has on file, each on its spec's launch gear: ${builds.length} talent builds across ${new Set(builds.map(b => b.spec)).size} specs, each run on its own at ${formatToNumber(data.iterations)} iterations against the same target, with the same buffs and the same consumables, plus the builds a talent search found on top of those. Nothing is simulated in your browser.`}>
		<div className="grid grid-cols-[repeat(auto-fit,minmax(min(26rem,100%),1fr))] gap-4">
			<PageSection title="How a number gets onto this page">
				<p className="m-0">
					Every build here was simulated: a character is assembled, given a talent build, a gear set and a rotation, and run against the same target
					for {formatToNumber(data.iterations)} iterations. What comes out is the average damage per second of those runs. Nothing is estimated,
					interpolated or predicted - each row is the outcome of that build being played out {formatToNumber(data.iterations)} times.
				</p>
				<p className="m-0">
					The gear sets and rotations are files in the repository, written by people. The talent builds are the community ones from each spec&apos;s
					own page, plus whatever a search found on top of them. All of it runs headless when the sim changes, and the site ships the results, which
					is why the table is instant and why nothing is simulated in your browser.
				</p>
			</PageSection>
			<PageSection title="Where AI comes into it, and where it does not">
				<p className="m-0">
					<strong>Not into any number on this page.</strong> The damage figures come from a simulator - an open-source engine, forked and adjusted for
					Forever. It is ordinary code doing arithmetic on the client&apos;s own data tables. No language model produces, adjusts or estimates a DPS
					figure, and the talent search is a hill climb that measures builds rather than reasons about them.
				</p>
				<p className="m-0">
					<strong>Into the code, heavily.</strong> This sim&apos;s Forever changes, the talent search, this page and most of what surrounds them were
					written by an AI assistant working to one person&apos;s direction. That is worth saying plainly, because it is exactly the situation where
					confident-sounding output is cheap and being wrong is easy.
				</p>
				<p className="m-0">
					So the checking is the point rather than an afterthought. <a href={`${SITE_BASE}evidence/`}>Every ability the sim registers</a> records
					where its numbers came from, and a test refuses to let one be added without that. The <strong>rests on a guess</strong> column carries it
					through to here: it is how much of a build&apos;s damage depends on something nobody has confirmed.
				</p>
			</PageSection>
		</div>

		<Leaderboard />

		<ul className="m-0 flex max-w-landing-lg flex-col gap-2 pl-6 text-sm text-white/60">
			<li>
				<SimSource />
			</li>
			<li>
				<strong>Every build meets the same conditions.</strong> One target, one encounter length, one buff set, one consumable list for its role. That
				is what makes two numbers comparable - the live <a href={`${SITE_BASE}dps_rankings/`}>rankings page</a> achieves the same thing by putting
				everyone in one raid, which stops being possible at this count.
			</li>
			<li>
				<strong>The consumables are the arena&apos;s, and they did not used to be.</strong> Every spec brought its own list from its own test file, and
				the gaps were not small ones: both paladins and the feral tank had their weapon imbue commented out entirely, while warrior, hunter, rogue and
				tank warrior carried Windfury. Stripping the warrior&apos;s imbues costs it 14.2% - so this table was reporting a 19.6% gap between warrior and
				retribution while handing one of them a weapon buff and the other a bare weapon. It is 2.3% now, and the difference was never about the specs.
				Each row says which list it drank.
			</li>
			<li>
				<strong>Three lists, not one.</strong> Elemental Sharpening Stone is +2% melee crit and -2% <em>ranged</em> crit, so a single list for all
				fifteen would equalise the shopping and quietly tax the only spec that shoots. Within a role the list is identical - the same shopping list, not
				the same benefit, which is why Mighty Rage Potion stays in the melee list even though only warriors can spend it. What a class grants itself is
				not a consumable and is left alone: an enhancement shaman keeps Windfury Weapon and a rogue keeps its poisons. Equalising those took 23.6% off
				the shaman, which is not a shaman measured fairly, it is a shaman disarmed.
			</li>
			<li>
				<strong>Every spec wears the same standard of gear.</strong> Each row uses its spec&apos;s launch set: the best pre-raid gear from one shared
				item pool (dungeons, crafting, quests and world drops, with every raid and world boss left out), picked by the same rule for every spec. They
				land within about three item levels of each other, and the line under the filter says exactly how far apart the rows you are looking at are. The
				arena sims the spec&apos;s other gear sets too; they are not shown, because they would compare gear rather than specs.
			</li>
			<li>
				<strong>Rests on a guess</strong> is what the build&apos;s damage is made of, not a verdict on it. Each ability is weighted by its share of that
				build&apos;s damage and looked up in the <a href={`${SITE_BASE}evidence/`}>evidence manifest</a>. A build ten DPS ahead means something
				different if a quarter of it is unconfirmed. Hover the bar for the breakdown.
			</li>
			<li>
				<strong>Rotations tagged low ranks</strong> drop to cheaper spell ranks as mana runs down. The sim gives every rank full spell-power scaling,
				which is what beta players see at level 20; nobody has confirmed whether Forever penalises low ranks at level 60, and if it does those rows will
				overstate the spec. The spec&apos;s normal rotation is always ranked beside them.
			</li>
			<li>
				<strong>Gear and rotations come from what is already here</strong> - the sets and priority lists on each spec&apos;s page. Nothing invents a
				better rotation than the ones people have written down, so a spec with one rotation on file gets one rotation ranked. That is a gap in the data,
				not a finding about the spec.
			</li>
			<li>
				<strong>Talents are searched, because they cannot be enumerated.</strong> A warrior has <strong>89,776,730,783,606,094</strong> builds it could
				actually spend - counted from the trees, enforcing rank caps, row gates and the prerequisite arrows. Count only the all-or-nothing ones, every
				talent maxed or untouched, and a warrior still has 57,341,667 and a mage 1,261,940,421 - eighteen months and forty years at a second a build. So
				each spec&apos;s best known build is improved one point at a time instead: price every point that could come out, price every point that could
				go in, make the best trade, repeat until no single move helps. Rows marked{' '}
				<span className="rounded-full border border-brand px-1.5 text-xs text-brand">found by search</span> came out of that, and hovering one shows its
				talent string.
			</li>
			<li>
				<strong>What that does and does not promise.</strong> It climbs from every distinct build the spec has on file rather than only its best one,
				because a climb goes to the nearest peak. Several starts agreeing is the cheapest evidence available that the peak is not merely nearby - it is
				still not proof that nothing higher exists. It also only knows what this sim models: a talent flagged as unimplemented is worth zero here, so
				the search will happily empty it, and that is a fact about the sim rather than advice. Every build it reaches is checked against rank caps, row
				gates and prerequisites first.
			</li>
			<li>
				<strong>A build that does not spend 51 points says so.</strong> The mage Frost community build spends 49. It is left as written rather than
				quietly corrected - it is somebody else&apos;s build - but the searched row beside it shows what those two points are worth.
			</li>
			<li>
				Tank specs are measured on damage alone and healing specs are absent, because damage is the only axis this table has. A protection paladin at
				the bottom is not a bad tank - and a searched tank build is a tank build with the mitigation optimised out of it, so read those rows as what the
				spec can do to a target dummy and nothing else.
			</li>
		</ul>
	</ProductPage>
);
