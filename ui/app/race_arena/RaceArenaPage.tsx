import { textClassName } from '@sim/proto/utils';
import { formatToNumber, formatToPercent } from '@sim/utils/format';
import { Button } from '@ui-kit/Button';
import clsx from 'clsx';
import { useState } from 'react';

import { LandingFooter } from '../landing/LandingFooter';
import { LandingHeader } from '../landing/LandingHeader';
import { type RaceArenaSetup, RaceArenaSetupDialog } from './RaceArenaSetupDialog';
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
	// The fight length the list was run at; absent on results from before fight lengths were split.
	seconds?: number;
	keptWeaponType?: boolean;
	weaponType?: string;
	races: Array<RaceResult>;
};
export type Tier = { name: string; maxBehind?: number };
export type RaceArenaResults = {
	commit: string;
	generated: string;
	iterations: number;
	fight: { seconds: number; durations?: Array<number>; targetLevel: number; targetArmor: number };
	tiers: Array<Tier>;
	noise: { maxError: number; maxDifference: number };
	lists: Array<RaceList>;
	setups: Record<string, RaceArenaSetup>;
};

// The runner groups DPS builds by these spec keys.
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

// The creature types the lists were run against. Mechanical is the race arena's neutral target;
// the other two are the ones a racial pays extra against.
const TARGETS: Array<{ key: string; label: string; blurb: string }> = [
	{ key: 'Mechanical', label: 'Neutral target', blurb: 'A target no racial pays extra against. This is the main view.' },
	{ key: 'Elemental', label: 'Elemental target', blurb: "Adds the Skyborne's Elemental Insight: +5% damage and crit damage against Elementals." },
	{
		key: 'Beast',
		label: 'Beast target',
		blurb: "Adds the Dwarf's Big Game Hunter and the Troll's Beast Slaying: +5% damage and crit damage against Beasts.",
	},
];

// Existing theme tokens, gold at the top down to red at the bottom.
const TIER_COLOR: Record<string, string> = {
	S: 'border-brand text-brand',
	A: 'border-expansion text-expansion',
	B: 'border-success text-success',
	C: 'border-orange text-orange',
	D: 'border-danger text-danger',
};

const PILL = 'inline-flex cursor-pointer items-center gap-1 rounded-full border px-2 py-1 text-sm';
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
		<span className="flex flex-wrap items-baseline gap-x-1">
			<strong>{race.race}</strong>
			<span className="text-sm text-white/60 tabular-nums">
				{race.behind ? formatToPercent(-race.behind, { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : 'best'}
			</span>
			{race.relabelled && (
				<span
					className="cursor-help rounded-full border border-brand px-1 text-xs text-brand"
					title={`This race's weapons were relabelled ${race.relabelled} (same stats and procs), which beat the list's own weapons.`}
					data-testid="relabelled">
					as {race.relabelled}
				</span>
			)}
		</span>
		<span className="text-xs text-white/50">{race.racials}</span>
	</span>
);

const TierList = ({ list, tiers, setup, commit }: { list: RaceList; tiers: Array<Tier>; setup?: RaceArenaSetup; commit: string }) => {
	const [setupOpen, setSetupOpen] = useState(false);
	return (
		<section className="flex flex-col gap-2 border border-surface-border bg-black/30 p-2" data-testid="race-list">
			<header className="flex flex-col">
				<span className="text-xs text-white/60">{specName(list.spec)}</span>
				<h3 className={clsx('m-0 text-lg', textClassName(list.class.toLowerCase()))}>{list.build}</h3>
				<span className="text-xs text-white/50">
					{list.gear} gear, {list.rotation} rotation, {consumablesName(list.consumables)}
					{list.keptWeaponType && ', weapons kept as shipped for every race'}
					{list.weaponType && `, weapons run as ${list.weaponType} for every race`}
				</span>
				{setup && (
					<Button
						variant="secondary"
						size="sm"
						className="mt-2 self-start"
						aria-label={`View setup for ${specName(list.spec)}: ${list.build}`}
						onClick={() => setSetupOpen(true)}>
						View setup
					</Button>
				)}
			</header>
			{setup && (
				<RaceArenaSetupDialog
					open={setupOpen}
					onOpenChange={setSetupOpen}
					title={`${specName(list.spec)}: ${list.build}`}
					gear={list.gear}
					rotation={list.rotation}
					setup={setup}
					commit={commit}
				/>
			)}
			<div className="flex flex-col">
				{tiers.map(tier => {
					const races = list.races.filter(race => race.tier === tier.name);
					return (
						<div
							key={tier.name}
							className="flex items-stretch gap-2 border-t border-surface-border py-1"
							data-testid="tier-row"
							data-tier={tier.name}>
							<span className={clsx('flex w-8 shrink-0 items-center justify-center rounded-sm border text-lg font-bold', TIER_COLOR[tier.name])}>
								{tier.name}
							</span>
							<div className="flex min-w-0 flex-wrap gap-1">
								{races.length ? (
									races.map(race => <RaceChip key={race.race} race={race} />)
								) : (
									<span className="self-center text-white/30">-</span>
								)}
							</div>
						</div>
					);
				})}
			</div>
		</section>
	);
};

const secondsOf = (data: RaceArenaResults, list: RaceList) => list.seconds ?? data.fight.seconds;

const fightLabel = (seconds: number) => (seconds % 60 === 0 ? `${seconds / 60} min` : `${seconds} s`);

const Lists = ({ data }: { data: RaceArenaResults }) => {
	const targets = TARGETS.filter(target => data.lists.some(list => list.target === target.key));
	const [target, setTarget] = useState(targets[0]?.key ?? 'Mechanical');
	const durations = data.fight.durations ?? [data.fight.seconds];
	const [seconds, setSeconds] = useState(durations.includes(data.fight.seconds) ? data.fight.seconds : durations[0]);
	const shown = data.lists.filter(list => list.target === target && secondsOf(data, list) === seconds);
	const classes = [...new Set(shown.map(list => list.class))];

	return (
		<div className="flex flex-col gap-4">
			<div className="flex flex-col gap-2">
				<ul className="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0" data-testid="tier-legend">
					{data.tiers.map((tier, index) => (
						<li key={tier.name} className="flex items-center gap-1">
							<span className={clsx('rounded-sm border px-1 font-bold', TIER_COLOR[tier.name])}>{tier.name}</span>
							<span className="text-sm text-white/70">{tierRange(data.tiers, index)}</span>
						</li>
					))}
				</ul>
				{durations.length > 1 && (
					<div className="flex flex-wrap items-center gap-2" data-testid="fight-lengths">
						{durations.map(d => (
							<Button
								variant="unstyled"
								key={d}
								className={clsx(PILL, d === seconds ? PILL_ON : PILL_OFF)}
								type="button"
								aria-pressed={d === seconds}
								onClick={() => setSeconds(d)}>
								{fightLabel(d)} fight
							</Button>
						))}
						<span className="text-sm text-white/50">
							Short fights favour burst cooldowns and opening racials; long ones favour passives that pay every second.
						</span>
					</div>
				)}
				{targets.length > 1 && (
					<div className="flex flex-wrap items-center gap-2">
						{targets.map(t => (
							<Button
								variant="unstyled"
								key={t.key}
								className={clsx(PILL, t.key === target ? PILL_ON : PILL_OFF)}
								type="button"
								aria-pressed={t.key === target}
								onClick={() => setTarget(t.key)}>
								{t.label}
							</Button>
						))}
						<span className="text-sm text-white/50">{targets.find(t => t.key === target)?.blurb}</span>
					</div>
				)}
			</div>
			{classes.map(playerClass => (
				<div key={playerClass} className="flex flex-col gap-2" data-testid="race-class">
					<h2 className={clsx('m-0 text-xl', textClassName(playerClass.toLowerCase()))}>{playerClass}</h2>
					<div className="grid grid-cols-[repeat(auto-fit,minmax(min(26rem,100%),1fr))] gap-2">
						{shown
							.filter(list => list.class === playerClass)
							.map(list => (
								<TierList
									key={`${list.spec}|${list.build}`}
									list={list}
									tiers={data.tiers}
									setup={data.setups[`${list.spec}|${list.build}`]}
									commit={data.commit}
								/>
							))}
					</div>
				</div>
			))}
		</div>
	);
};

export const RaceArenaPage = ({ data = results as RaceArenaResults }: { data?: RaceArenaResults }) => (
	<div className="flex min-h-full flex-col">
		<LandingHeader />
		<main className="ui-landing-container flex grow flex-col gap-6 py-8">
			<Button href="/forever/" variant="link">
				Back to the simulator
			</Button>
			<header>
				<h1 className="text-3xl font-bold text-brand">Race tier list</h1>
				<p>
					Every legal race, compared on{' '}
					{data.lists.filter(list => list.target === 'Mechanical' && secondsOf(data, list) === data.fight.seconds).length} DPS builds from the
					upstream simulator.
				</p>
			</header>
			<section className="flex flex-col gap-2 border border-surface-border p-4">
				<h2 className="m-0 text-xl">Method</h2>
				<p className="m-0">
					This is a pre-release baseline for choosing a race at level 60. Each build holds gear, enchants, talents, rotation, consumables and buffs
					constant across every legal race. Changing the race includes its racial abilities, base attributes and effects on pets. Gear and rotations
					are not individually optimized for each race.
				</p>
				<p className="m-0">
					We use upstream level-60 presets, including pre-raid and development gear. Warlock uses upstream's preraid test set. View setup on each
					build shows the gear, enchants, weapon swaps and talent ranks saved with these results. These presets provide a common starting point; they
					are not a prediction of final release gear.
				</p>
				<p className="m-0">
					Every simulation is a fixed-length fight ({(data.fight.durations ?? [data.fight.seconds]).map(fightLabel).join(', ')};{' '}
					{fightLabel(data.fight.seconds)} is the main view) against one level {data.fight.targetLevel} target with{' '}
					{formatToNumber(data.fight.targetArmor)} armor, with full raid, party and individual buffs and debuffs. The neutral target is the main
					comparison; Elemental and Beast views show the relevant creature-type racials. Both Skyborne races appear wherever their class is legal.
					Cooldown racials follow the preset rotation. Rankings measure damage, including pets; survival and utility are outside this comparison.
				</p>
				<p className="m-0">
					The gear keeps its actual weapon types for every race. The engine follows wowsims/forever with one exception: Dwarf's Earth Fury grants +2%
					critical damage and healing with any weapon or none, including through swaps, just like Tauren's crit bonus.
				</p>
				<p className="m-0">
					{formatToNumber(data.iterations)} iterations per race use the same random seed (101). The largest standard error is{' '}
					{formatToNumber(data.noise.maxError, { maximumFractionDigits: 3 })}% of a list's best; differences below about{' '}
					{formatToNumber(data.noise.maxDifference, { maximumFractionDigits: 2 })}% may be simulation noise. Tiers compare races within a build, not
					damage between builds.
				</p>
				<p className="m-0">
					Read these as estimates under shared assumptions. Release balance, gear, buffs, fight length and encounter mechanics can change the
					ordering. Close races may be effectively tied, even when they appear in different tiers.
				</p>
			</section>
			<Lists data={data} />
			<p className="m-0 text-sm text-white/60" data-testid="race-arena-source">
				Computed from sim{' '}
				<Button variant="link" href={`https://github.com/laurencestokes/foreversim/commit/${data.commit}`}>
					<code>{data.commit.slice(0, 7)}</code>
				</Button>{' '}
				on {data.generated.slice(0, 10)}, {formatToNumber(data.iterations)} iterations per race.
			</p>
		</main>
		<LandingFooter />
	</div>
);
