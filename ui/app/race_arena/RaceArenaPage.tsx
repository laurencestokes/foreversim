// The race tier list: every race each class can be, ranked on each DPS spec's community builds
// with everything but the race held the same.
//
// Like the build arena, nothing is simulated here. The lists run headless
// (sim/arenalib/race_arena.go), tools/race_arena sorts them into tiers, and this page renders the
// committed results file. The file is the cache and the commit that produced it is the key.

import { textClassName } from '@sim/proto/utils';
import { formatToNumber, formatToPercent } from '@sim/utils/format';
import clsx from 'clsx';
import { type ReactNode, useState } from 'react';

import { PageSection, ProductPage, SITE_REPO_URL } from '../ProductPage';
import results from './results.json';

export type RaceResult = { race: string; dps: number; behind: number; tier: string; relabelled?: string; racials: string };
export type RaceList = {
	spec: string;
	class: string;
	build: string;
	talents: string;
	gear: string;
	rotation: string;
	consumables: string;
	target: string;
	keptWeaponType?: boolean;
	weaponType?: string;
	races: Array<RaceResult>;
};
export type Tier = { name: string; maxBehind?: number };
export type RaceArenaResults = {
	commit: string;
	generated: string;
	iterations: number;
	clientBuild: string;
	fight: { seconds: number; targetLevel: number; targetArmor: number };
	tiers: Array<Tier>;
	noise: { maxError: number; maxDifference: number };
	lists: Array<RaceList>;
};

// Keyed by the arena's spec directories, which is what the runner writes. DPS specs only: a
// tank or healer's race is not worth what it adds to damage.
const SPEC_NAMES: Record<string, string> = {
	balance_druid: 'Balance Druid',
	feral_druid: 'Feral Druid',
	hunter: 'Hunter',
	mage: 'Mage',
	retribution_paladin: 'Retribution Paladin',
	shadow_priest: 'Shadow Priest',
	smite_priest: 'Smite Priest',
	rogue: 'Rogue',
	elemental_shaman: 'Elemental Shaman',
	enhancement_shaman: 'Enhancement Shaman',
	warlock: 'Warlock',
	warrior: 'Warrior',
};

// The creature types the lists were run against. Mechanical is the build arena's neutral target;
// the other two are the ones a racial pays extra against.
const TARGETS: Array<{ key: string; label: string; blurb: string }> = [
	{ key: 'Mechanical', label: 'Neutral target', blurb: 'A target no racial pays extra against. This is the main view.' },
	{ key: 'Elemental', label: 'Elemental target', blurb: "Adds the Skyborne's Elemental Insight: +5% damage against Elementals." },
	{ key: 'Beast', label: 'Beast target', blurb: "Adds the Dwarf's Big Game Hunter and the Troll's Beast Slaying: +5% damage against Beasts." },
];

// Existing theme tokens, gold at the top down to red at the bottom.
const TIER_COLOR: Record<string, string> = {
	S: 'border-brand text-brand',
	A: 'border-expansion text-expansion',
	B: 'border-evidence-forever text-evidence-forever',
	C: 'border-orange text-orange',
	D: 'border-danger text-danger',
};

const PILL = 'inline-flex cursor-pointer items-center gap-1 rounded-full border px-2.5 py-1 text-sm';
const PILL_OFF = 'border-white/18 text-gray-300 hover:border-brand hover:text-white';
const PILL_ON = 'border-brand bg-brand/15 text-white';

const percent = (value: number) => formatToPercent(value, { minimumFractionDigits: 1, maximumFractionDigits: 1 });

// What a tier holds, in words, from the thresholds the results file carries.
export const tierRange = (tiers: Array<Tier>, index: number): string => {
	const upper = tiers[index].maxBehind;
	const lower = index > 0 ? tiers[index - 1].maxBehind : undefined;
	if (upper === undefined) return `more than ${percent(lower ?? 0)} behind`;
	if (lower === undefined) return `within ${percent(upper)} of the best`;
	return `${percent(lower)} to ${percent(upper)} behind`;
};

const specName = (spec: string) => SPEC_NAMES[spec] ?? spec;
const consumablesName = (list: string) => list.replace('Arena-', '').replace('+class', ' + class imbues').toLowerCase();

const RaceChip = ({ race }: { race: RaceResult }) => (
	<span
		className="flex flex-col rounded-sm border border-white/18 bg-black/40 px-2 py-1"
		title={`${formatToNumber(race.dps, { minimumFractionDigits: 1, maximumFractionDigits: 1 })} DPS`}
		data-testid="race-chip">
		<span className="flex flex-wrap items-baseline gap-x-1.5">
			<strong>{race.race}</strong>
			<span className="text-sm text-white/60 tabular-nums">
				{race.behind ? formatToPercent(-race.behind, { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : 'best'}
			</span>
			{race.relabelled && (
				<span
					className="cursor-help rounded-full border border-brand px-1.5 text-xs text-brand"
					title={`This race's weapons were relabelled ${race.relabelled} (same stats and procs), which beat the list's own weapons.`}
					data-testid="relabelled">
					as {race.relabelled}
				</span>
			)}
		</span>
		<span className="text-xs text-white/50">{race.racials}</span>
	</span>
);

const TierList = ({ list, tiers }: { list: RaceList; tiers: Array<Tier> }) => (
	<section className="flex flex-col gap-2 border border-surface-border bg-black/30 p-3" data-testid="race-list">
		<header className="flex flex-col">
			<span className="text-xs text-white/60">{specName(list.spec)}</span>
			<h3 className={clsx('m-0 text-lg', textClassName(list.class.toLowerCase()))}>{list.build}</h3>
			<span className="text-xs text-white/50">
				{list.gear} gear, {list.rotation} rotation, {consumablesName(list.consumables)}
				{list.keptWeaponType && ', weapons kept as shipped for every race'}
				{list.weaponType && `, weapons run as ${list.weaponType} for every race`}
			</span>
		</header>
		<div className="flex flex-col">
			{tiers.map(tier => {
				const races = list.races.filter(race => race.tier === tier.name);
				return (
					<div
						key={tier.name}
						className="flex items-stretch gap-2 border-t border-surface-border py-1.5"
						data-testid="tier-row"
						data-tier={tier.name}>
						<span className={clsx('flex w-8 shrink-0 items-center justify-center rounded-sm border text-lg font-bold', TIER_COLOR[tier.name])}>
							{tier.name}
						</span>
						<div className="flex min-w-0 flex-wrap gap-1.5">
							{races.length ? races.map(race => <RaceChip key={race.race} race={race} />) : <span className="self-center text-white/30">-</span>}
						</div>
					</div>
				);
			})}
		</div>
	</section>
);

const Lists = ({ data }: { data: RaceArenaResults }) => {
	const targets = TARGETS.filter(target => data.lists.some(list => list.target === target.key));
	const [target, setTarget] = useState(targets[0]?.key ?? 'Mechanical');
	const shown = data.lists.filter(list => list.target === target);
	const classes = [...new Set(shown.map(list => list.class))];

	return (
		<div className="flex flex-col gap-4">
			<div className="flex flex-col gap-2">
				<ul className="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0" data-testid="tier-legend">
					{data.tiers.map((tier, index) => (
						<li key={tier.name} className="flex items-center gap-1.5">
							<span className={clsx('rounded-sm border px-1.5 font-bold', TIER_COLOR[tier.name])}>{tier.name}</span>
							<span className="text-sm text-white/70">{tierRange(data.tiers, index)}</span>
						</li>
					))}
				</ul>
				{targets.length > 1 && (
					<div className="flex flex-wrap items-center gap-2">
						{targets.map(t => (
							<button
								key={t.key}
								className={clsx(PILL, t.key === target ? PILL_ON : PILL_OFF)}
								type="button"
								aria-pressed={t.key === target}
								onClick={() => setTarget(t.key)}>
								{t.label}
							</button>
						))}
						<span className="text-sm text-white/50">{targets.find(t => t.key === target)?.blurb}</span>
					</div>
				)}
			</div>
			{classes.map(playerClass => (
				<div key={playerClass} className="flex flex-col gap-2" data-testid="race-class">
					<h2 className={clsx('m-0 text-xl', textClassName(playerClass.toLowerCase()))}>{playerClass}</h2>
					<div className="grid grid-cols-[repeat(auto-fit,minmax(min(26rem,100%),1fr))] gap-3">
						{shown
							.filter(list => list.class === playerClass)
							.map(list => (
								<TierList key={`${list.spec}|${list.build}`} list={list} tiers={data.tiers} />
							))}
					</div>
				</div>
			))}
		</div>
	);
};

const Commit = ({ commit }: { commit: string }) =>
	commit ? (
		<a className="text-brand hover:underline" href={`${SITE_REPO_URL}/commit/${commit}`} target="_blank" rel="noreferrer">
			<code>{commit.slice(0, 7)}</code>
		</a>
	) : (
		<code>unknown</code>
	);

const RulesLink = ({ children }: { children: ReactNode }) => (
	<a href={`${SITE_REPO_URL}/blob/master/docs/forever_rules.md#racials`} target="_blank" rel="noreferrer">
		{children}
	</a>
);

const Method = ({ data }: { data: RaceArenaResults }) => (
	<PageSection title="Method">
		<p className="m-0">
			<strong>One build, every race.</strong> Each list is one community build of a DPS spec: the talent presets named with their point split on the
			spec&apos;s own page, the same builds the build arena runs. Each wears the page&apos;s default gear set and plays the rotation the page picks for
			those talents. Where the default gear cannot play the build, the list names the set it wears instead: the Arms warrior&apos;s two-hander, the
			Mutilate rogue&apos;s daggers, the Smite priest&apos;s own set, and the feral&apos;s Launch set, because its default set has no weapon.
		</p>
		<p className="m-0">
			<strong>The build arena&apos;s environment.</strong> One buff set, the consumable list for the spec&apos;s role, and one target: a level{' '}
			{data.fight.targetLevel} boss with {formatToNumber(data.fight.targetArmor)} armor, for {data.fight.seconds} seconds. Every race of a list is run on
			the same random seed, so they meet the same rolls, and a race&apos;s result includes its base attributes as well as its racials.
		</p>
		<p className="m-0">
			<strong>Identical gear, weapon type aside.</strong> Human sword, Orc axe and Dwarf mace racials pay crit only while that weapon type is held, so
			ranking races on the gear as shipped would rank the gear set&apos;s weapons. So those three races are run twice: on the gear as shipped, and with
			the racial&apos;s weapon type laid over the same main-hand and off-hand weapons through the sim&apos;s weapon type override, which changes what the
			weapon counts as and nothing else - its stats, procs and item are unchanged. The better run counts, and a chip marked{' '}
			<span className="rounded-full border border-brand px-1.5 text-xs text-brand">as Sword</span> is one where the relabel won. Held off-hands, shields
			and ranged weapons are never relabelled; a weapon is only relabelled to a type the class can wield in that hand (a two-hander only to a type the
			class uses two-handed); and a build whose rotation needs its weapon type keeps it (the Mutilate rogue keeps its daggers). A relabel also changes
			what weapon type talents see, such as the rogue&apos;s Hack and Slash, and that counts toward the race&apos;s result as it would in game.
		</p>
		<p className="m-0">
			<strong>One exception to the gear as shipped.</strong> The Arms warrior takes Weaponmaster, which pays very differently by weapon type, and its
			set&apos;s two-hander is an axe. Left as an axe, only the Human&apos;s relabel would reach Weaponmaster&apos;s sword bonus, and the Human would be
			credited with a talent any race can have. A sword is the best type for that build for every race measured (about 3.5% over the axe for a race with
			no weapon racial), so its two-hander is run as a sword for every race, and the Orc and Dwarf try their own types on top.
		</p>
		<p className="m-0">
			<strong>{formatToNumber(data.iterations)} iterations per race.</strong> One standard error on a race&apos;s mean is at most{' '}
			{formatToNumber(data.noise.maxError, { maximumFractionDigits: 3 })}% of its list&apos;s best, so two races less than about{' '}
			{formatToNumber(data.noise.maxDifference, { maximumFractionDigits: 2 })}% apart may be level. That is small next to a tier, but a race sitting right
			on a tier edge could fall either side of it.
		</p>
		<p className="m-0">
			<strong>Tiers are measured from each list&apos;s best race:</strong>{' '}
			{data.tiers.map((tier, index) => `${tier.name} ${tierRange(data.tiers, index)}`).join('; ')}. The tiers compare races within one list, never one
			spec with another.
		</p>
		<p className="m-0">
			<strong>Where the racials come from.</strong> The Forever beta client, build {data.clientBuild}: each race&apos;s racial skill line and the spell
			data behind it. Every value is listed with its spell id in <RulesLink>docs/forever_rules.md</RulesLink>.
		</p>
	</PageSection>
);

const Caveats = () => (
	<PageSection title="Caveats">
		<ul className="m-0 flex list-disc flex-col gap-2 pl-5">
			<li>The racials come from a beta client. Blizzard can change any of them, and a new client build can move any list here.</li>
			<li>
				Touch of the Grave (Undead) is modelled as a drain of 5% of the caster&apos;s maximum health as Shadow damage. It can be resisted, cannot crit,
				and procs from hits and from applying a damage over time spell, not from its ticks: a 5% chance for warriors, paladins and rogues, 10% for
				priests, mages and warlocks, at most once a second. Its damage follows maximum health, so the environment&apos;s health buffs (Fortitude, Kings,
				and Flask of the Titans on the melee list) raise it. The client&apos;s tooltip says &ldquo;up to 5%&rdquo;, so the real drain may be smaller
				or vary; this is the least certain number on the page, and it is why Undead tops so many lists.
			</li>
			<li>
				Eureka! (Gnome) is modelled for warriors, rogues and warlocks only. Gnome mages and priests are ranked without it, on Expansive Mind alone. A
				rogue&apos;s Eureka! is assumed not to spend a charge on Blade Flurry, which deals no damage itself.
			</li>
			<li>
				Cooldown racials - Blood Fury, Berserking, Elune&apos;s Light, Eureka! - fire when each page&apos;s rotation fires its cooldowns, not at an
				ideal moment. The warrior rotation, for example, only uses them in roughly the first 40 seconds and the last 30.
			</li>
			<li>
				Survival and utility racials are not valued: Stoneform, Shatter Curse, Will to Survive, Will of the Forsaken and the like. Neither is the Night
				Elf priest&apos;s Starshards, which neither priest rotation casts.
			</li>
			<li>
				The +5% creature type racials only apply against a matching target: the Skyborne against Elementals, Dwarves and Trolls against Beasts. The main
				view is a neutral target; the Elemental and Beast views show those fights.
			</li>
			<li>
				Damage over time effects are modelled as dynamic, recomputing the caster&apos;s bonuses on every tick, as beta reports describe (
				<RulesLink>docs/forever_rules.md</RulesLink>). A short racial cooldown therefore raises only the ticks that land while it is up.
			</li>
			<li>
				One gear set per list. A set with more or less hit or crit can reorder races whose racials are hit (Tauren) or crit (the weapon racials,
				Elune&apos;s Light).
			</li>
			<li>
				Healers and tanks are not ranked. This page measures damage, and neither role is valued by its damage; the build arena shows tank specs on
				damage alone.
			</li>
			<li>
				The Skyborne are one row, because the High Order and the Windshapers share every racial. A Skyborne mage can only be High Order and a Skyborne
				shaman only a Windshaper; warriors, hunters, rogues and druids can be either.
			</li>
		</ul>
	</PageSection>
);

export const RaceArenaPage = ({ data = results as RaceArenaResults }: { data?: RaceArenaResults }) => (
	<ProductPage
		title="Race tier list"
		subtitle={`Every race each class can be, ranked on ${data.lists.filter(list => list.target === 'Mechanical').length} DPS builds with everything else held the same. Computed ahead of time from the sim; nothing is simulated in your browser.`}>
		<div className="grid grid-cols-[repeat(auto-fit,minmax(min(26rem,100%),1fr))] gap-4">
			<Method data={data} />
			<Caveats />
		</div>

		<Lists data={data} />

		<p className="m-0 text-sm text-white/60" data-testid="race-arena-source">
			Computed from sim <Commit commit={data.commit} /> on {data.generated.slice(0, 10)}, {formatToNumber(data.iterations)} iterations per race, racials
			from client build {data.clientBuild}.
		</p>
	</ProductPage>
);
