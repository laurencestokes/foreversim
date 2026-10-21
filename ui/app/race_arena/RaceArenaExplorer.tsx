import { textClassName } from '@sim/proto/utils';
import { formatToNumber, formatToPercent } from '@sim/utils/format';
import { Button } from '@ui-kit/Button';
import clsx from 'clsx';
import { type MouseEvent, type ReactNode, useMemo, useRef, useState } from 'react';

import {
	type BuildGroup,
	type Condition,
	conditionKey,
	durationsOf,
	fightLabel,
	groupBuilds,
	type RaceArenaResults,
	type RaceStanding,
	rankRaces,
	specName,
	TARGETS,
} from './raceArenaData';

// One hue, light to dark, for the tier a cell sits in: brand gold for the top tier, stepping toward the
// surface. An ordinal ramp, validated against the page surface (#1e2633): monotone lightness, a visible
// step between neighbours, and the faintest step still 2.3:1 on the surface. The tier letter rides in
// every cell, so the colour is never the only cue; ink is black on the three light steps and white on
// the two dark ones, each at least 4.5:1.
const TIER_RAMP = ['#f8b700', '#cc9a0a', '#a78113', '#896d1a', '#6f5c20'];
const TIER_INK = ['#000000', '#000000', '#000000', '#ffffff', '#ffffff'];

const PILL = 'inline-flex cursor-pointer items-center gap-1 rounded-full border px-2 py-1 text-sm';
const PILL_OFF = 'border-white/18 text-gray-300 hover:border-brand hover:text-white';
const PILL_ON = 'border-brand bg-brand/15 text-white';

const gap = (behind: number) => (behind ? formatToPercent(-behind, { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : 'best');
const gapPlain = (behind: number) => formatToPercent(behind, { minimumFractionDigits: 2, maximumFractionDigits: 2 });

type Tooltip = { x: number; y: number; content: ReactNode } | null;

// A tooltip that follows the pointer inside a relatively positioned box.
const useTooltip = () => {
	const box = useRef<HTMLDivElement>(null);
	const [tip, setTip] = useState<Tooltip>(null);
	const show = (event: MouseEvent, content: ReactNode) => {
		const rect = box.current?.getBoundingClientRect();
		if (!rect) return;
		setTip({ x: event.clientX - rect.left, y: event.clientY - rect.top, content });
	};
	const node = tip && (
		<div
			role="tooltip"
			className="pointer-events-none absolute z-10 max-w-64 rounded-sm border border-surface-border bg-black/90 px-2 py-1 text-xs text-white shadow"
			style={{ left: tip.x + 12, top: tip.y + 12 }}>
			{tip.content}
		</div>
	);
	return { box, show, hide: () => setTip(null), node };
};

const Toggle = ({ on, onClick, children }: { on: boolean; onClick: () => void; children: ReactNode }) => (
	<Button variant="unstyled" type="button" className={clsx(PILL, on ? PILL_ON : PILL_OFF)} aria-pressed={on} onClick={onClick}>
		{children}
	</Button>
);

// The tier a gap falls in, as an index into the ramp.
const tierIndex = (data: RaceArenaResults, tierName: string) =>
	Math.max(
		0,
		data.tiers.findIndex(tier => tier.name === tierName),
	);

// Every race against every fight, for one build. Rows follow the ranking for the selected fights; the
// fights not selected stay visible but faded, so a reader sees what a pick gives up elsewhere.
const Heatmap = ({
	data,
	group,
	ranking,
	durations,
	targets,
	selected,
}: {
	data: RaceArenaResults;
	group: BuildGroup;
	ranking: Array<RaceStanding>;
	durations: Array<number>;
	targets: typeof TARGETS;
	selected: Set<string>;
}) => {
	const tooltip = useTooltip();
	const ordered = [...ranking.map(s => s.race), ...group.races.filter(race => !ranking.some(s => s.race === race))];
	return (
		<div ref={tooltip.box} className="relative overflow-x-auto">
			<table className="m-0 border-separate border-spacing-px text-sm" data-testid="race-heatmap">
				<caption className="pb-1 text-left text-xs text-white/60">
					Tier and gap behind the best race in each fight. Rows sorted by the largest gap in the selected fights.
				</caption>
				<thead>
					<tr>
						<th scope="col" rowSpan={2} className="pr-2 text-left font-normal text-white/60">
							Race
						</th>
						{durations.map(seconds => (
							<th key={seconds} scope="colgroup" colSpan={targets.length} className="px-1 text-center font-normal text-white/80">
								{fightLabel(seconds)}
							</th>
						))}
					</tr>
					<tr>
						{durations.flatMap(seconds =>
							targets.map(target => (
								<th key={`${seconds}|${target.key}`} scope="col" className="px-1 text-center text-xs font-normal text-white/50">
									{target.short}
								</th>
							)),
						)}
					</tr>
				</thead>
				<tbody>
					{ordered.map(race => (
						<tr key={race}>
							<th scope="row" className="pr-2 text-left font-normal whitespace-nowrap">
								{race}
							</th>
							{durations.flatMap(seconds =>
								targets.map(target => {
									const condition = { seconds, target: target.key };
									const row = group.lists.get(conditionKey(condition))?.races.find(r => r.race === race);
									const key = conditionKey(condition);
									if (!row) {
										return (
											<td key={key} className="text-center text-white/30">
												-
											</td>
										);
									}
									const index = tierIndex(data, row.tier);
									const label = `${race}, ${fightLabel(seconds)}, ${target.short.toLowerCase()} target: tier ${row.tier}, ${gap(row.behind)}`;
									return (
										<td
											key={key}
											aria-label={label}
											className={clsx('h-8 w-12 rounded-sm text-center font-bold tabular-nums', !selected.has(key) && 'opacity-35')}
											style={{ background: TIER_RAMP[index], color: TIER_INK[index] }}
											onMouseMove={event =>
												tooltip.show(
													event,
													<>
														<strong>{race}</strong>, {fightLabel(seconds)}, {target.short.toLowerCase()} target
														<br />
														Tier {row.tier}, {gap(row.behind)} ({formatToNumber(row.dps, { maximumFractionDigits: 1 })} DPS)
														<br />
														<span className="text-white/60">{row.racials}</span>
													</>,
												)
											}
											onMouseLeave={tooltip.hide}>
											{row.tier}
										</td>
									);
								}),
							)}
						</tr>
					))}
				</tbody>
			</table>
			{tooltip.node}
		</div>
	);
};

// Each race's gap behind the best across the fight lengths, at one target. Lines that cross are where the
// best race changes with fight length. Every line is muted except the pick and the one under the pointer;
// races are named at the line ends, so no line depends on colour to be told apart.
const FightLengthChart = ({
	data,
	group,
	durations,
	target,
	pick,
}: {
	data: RaceArenaResults;
	group: BuildGroup;
	durations: Array<number>;
	target: string;
	pick?: string;
}) => {
	const tooltip = useTooltip();
	const [hovered, setHovered] = useState<string | null>(null);
	const width = 560;
	const height = 260;
	const margin = { top: 16, right: 140, bottom: 40, left: 48 };
	const series = group.races
		.map(race => ({
			race,
			points: durations.map(seconds => ({
				seconds,
				row: group.lists.get(conditionKey({ seconds, target }))?.races.find(r => r.race === race),
			})),
		}))
		.filter(s => s.points.every(p => p.row));
	if (series.length === 0 || durations.length < 2) return null;

	const worst = Math.max(...series.flatMap(s => s.points.map(p => p.row!.behind)));
	const top = Math.max(0.5, Math.ceil(worst * 2) / 2);
	const step = top <= 1 ? 0.25 : top <= 3 ? 0.5 : 1;
	const ticks = Array.from({ length: Math.floor(top / step) + 1 }, (_, i) => i * step);
	const plotW = width - margin.left - margin.right;
	const plotH = height - margin.top - margin.bottom;
	const x = (i: number) => margin.left + (durations.length === 1 ? plotW / 2 : (i * plotW) / (durations.length - 1));
	const y = (behind: number) => margin.top + (behind / top) * plotH;

	// End labels, pushed apart so names never overlap.
	const labels = series.map(s => ({ race: s.race, y: y(s.points[s.points.length - 1].row!.behind) })).sort((a, b) => a.y - b.y);
	for (let i = 1; i < labels.length; i++) labels[i].y = Math.max(labels[i].y, labels[i - 1].y + 13);
	const overflow = labels.length ? labels[labels.length - 1].y - (height - margin.bottom + 4) : 0;
	if (overflow > 0) labels.forEach(l => (l.y -= overflow));

	const emphasis = (race: string) => race === hovered || (!hovered && race === pick);
	const noiseBand = Math.min(data.noise.maxDifference, top);

	return (
		<div ref={tooltip.box} className="relative">
			<svg
				viewBox={`0 0 ${width} ${height}`}
				className="w-full max-w-160"
				role="img"
				aria-label={`Gap behind the best race by fight length, ${TARGETS.find(t => t.key === target)?.short.toLowerCase() ?? target} target. The same figures are in the table above.`}
				data-testid="fight-length-chart">
				<rect x={margin.left} y={y(0)} width={plotW} height={y(noiseBand) - y(0)} fill="currentColor" className="text-white/8" />
				{ticks.map(tick => (
					<g key={tick}>
						<line x1={margin.left} x2={margin.left + plotW} y1={y(tick)} y2={y(tick)} stroke="currentColor" className="text-white/10" />
						<text x={margin.left - 6} y={y(tick) + 4} textAnchor="end" fontSize={11} fill="currentColor" className="text-white/60">
							{tick === 0 ? 'best' : `-${formatToNumber(tick, { maximumFractionDigits: 2 })}%`}
						</text>
					</g>
				))}
				{durations.map((seconds, i) => (
					<text key={seconds} x={x(i)} y={height - 12} textAnchor="middle" fontSize={11} fill="currentColor" className="text-white/70">
						{fightLabel(seconds)}
					</text>
				))}
				{[...series]
					.sort((a, b) => Number(emphasis(a.race)) - Number(emphasis(b.race)))
					.map(s => {
						const path = s.points.map((p, i) => `${i ? 'L' : 'M'}${x(i)},${y(p.row!.behind)}`).join(' ');
						const strong = emphasis(s.race);
						return (
							<g key={s.race}>
								<path
									d={path}
									fill="none"
									stroke={strong ? '#f8b700' : 'currentColor'}
									strokeWidth={2}
									className={strong ? '' : 'text-white/30'}
								/>
								{s.points.map((p, i) => (
									<circle
										key={p.seconds}
										cx={x(i)}
										cy={y(p.row!.behind)}
										r={4}
										fill={strong ? '#f8b700' : '#1e2633'}
										stroke={strong ? '#1e2633' : 'currentColor'}
										strokeWidth={2}
										className={strong ? '' : 'text-white/40'}
									/>
								))}
								<path
									d={path}
									fill="none"
									stroke="transparent"
									strokeWidth={12}
									onMouseMove={event => {
										setHovered(s.race);
										tooltip.show(
											event,
											<>
												<strong>{s.race}</strong>
												{s.points.map(p => (
													<div key={p.seconds}>
														{fightLabel(p.seconds)}: tier {p.row!.tier}, {gap(p.row!.behind)}
													</div>
												))}
											</>,
										);
									}}
									onMouseLeave={() => {
										setHovered(null);
										tooltip.hide();
									}}
								/>
							</g>
						);
					})}
				{labels.map(label => (
					<text
						key={label.race}
						x={margin.left + plotW + 8}
						y={label.y + 4}
						fontSize={11}
						fill="currentColor"
						className={emphasis(label.race) ? 'text-white' : 'text-white/60'}
						fontWeight={emphasis(label.race) ? 700 : 400}>
						{label.race}
					</text>
				))}
			</svg>
			<p className="m-0 text-xs text-white/50">
				Shaded band at the top: within {gapPlain(data.noise.maxDifference)} of the best, where two races may be level once simulation noise is counted.
			</p>
			{tooltip.node}
		</div>
	);
};

const BuildRow = ({
	data,
	group,
	conditions,
	selected,
	durations,
	targets,
}: {
	data: RaceArenaResults;
	group: BuildGroup;
	conditions: Array<Condition>;
	selected: Set<string>;
	durations: Array<number>;
	targets: typeof TARGETS;
}) => {
	const [open, setOpen] = useState(false);
	const topTier = data.tiers[0]?.name ?? 'S';
	const ranking = rankRaces(group, conditions, topTier);
	const [chartTarget, setChartTarget] = useState<string | null>(null);
	if (!ranking.length) return null;
	const pick = ranking[0];
	const tied = ranking.slice(1).filter(s => s.worst - pick.worst <= data.noise.maxDifference);
	const next = ranking[1 + tied.length];
	const selectedTargets = targets.filter(t => conditions.some(c => c.target === t.key));
	const shownTarget = chartTarget && selectedTargets.some(t => t.key === chartTarget) ? chartTarget : (selectedTargets[0]?.key ?? 'Mechanical');

	return (
		<li className="flex flex-col gap-2 border-t border-surface-border py-2" data-testid="best-race-row">
			<div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
				<span className="w-56 shrink-0">
					<span className="block text-xs text-white/60">{specName(group.spec)}</span>
					<span className={clsx('font-bold', textClassName(group.class.toLowerCase()))}>{group.build}</span>
				</span>
				<span className="min-w-0 grow">
					<strong className="text-lg text-brand" data-testid="best-race">
						{pick.race}
					</strong>{' '}
					<span className="text-sm text-white/70">
						{topTier} in {pick.topTier} of {pick.conditions} {pick.conditions === 1 ? 'fight' : 'fights'},{' '}
						{pick.worst ? `never more than ${gapPlain(pick.worst)} behind` : 'best in every one'}
					</span>
					{tied.length > 0 && (
						<span className="block text-sm text-white/70">
							Tied within noise: {tied.map(s => (s.worst > pick.worst ? `${s.race} (worst ${gapPlain(s.worst)})` : s.race)).join(', ')}
						</span>
					)}
					{next && (
						<span className="block text-sm text-white/50">
							Next: {next.race} (worst {gapPlain(next.worst)})
						</span>
					)}
				</span>
				<Button variant="secondary" size="sm" aria-expanded={open} onClick={() => setOpen(!open)}>
					{open ? 'Hide detail' : 'Show detail'}
				</Button>
			</div>
			{open && (
				<div className="flex flex-col gap-4 pl-0 md:pl-56">
					<Heatmap data={data} group={group} ranking={ranking} durations={durations} targets={targets} selected={selected} />
					{durations.length > 1 && (
						<div className="flex flex-col gap-1">
							<div className="flex flex-wrap items-center gap-2">
								<span className="text-sm text-white/70">Gap behind the best by fight length:</span>
								{selectedTargets.length > 1 &&
									selectedTargets.map(t => (
										<Toggle key={t.key} on={t.key === shownTarget} onClick={() => setChartTarget(t.key)}>
											{t.short}
										</Toggle>
									))}
							</div>
							<FightLengthChart data={data} group={group} durations={durations} target={shownTarget} pick={pick.race} />
						</div>
					)}
				</div>
			)}
		</li>
	);
};

// The best race for each build over the fights a reader picks, with every race against every fight
// behind it. The pick is the race whose largest gap behind the best is smallest across the picked
// fights: never far off, whichever of them it meets, rather than best on average.
export const RaceArenaExplorer = ({ data }: { data: RaceArenaResults }) => {
	const durations = durationsOf(data);
	const targets = TARGETS.filter(target => data.lists.some(list => list.target === target.key));
	const groups = useMemo(() => groupBuilds(data), [data]);
	const [lengths, setLengths] = useState<Set<number>>(() => new Set(durations));
	const [targetKeys, setTargetKeys] = useState<Set<string>>(() => new Set([targets[0]?.key ?? 'Mechanical']));

	const flip = <T,>(set: Set<T>, value: T, update: (next: Set<T>) => void) => {
		const next = new Set(set);
		if (next.has(value)) {
			if (next.size === 1) return;
			next.delete(value);
		} else {
			next.add(value);
		}
		update(next);
	};

	const conditions: Array<Condition> = durations
		.filter(seconds => lengths.has(seconds))
		.flatMap(seconds => targets.filter(t => targetKeys.has(t.key)).map(t => ({ seconds, target: t.key })));
	const selected = new Set(conditions.map(conditionKey));
	const classes = [...new Set(groups.map(group => group.class))].sort();

	return (
		<section className="flex flex-col gap-4 border border-surface-border p-4" data-testid="race-explorer">
			<header className="flex flex-col gap-1">
				<h2 className="m-0 text-xl">Best race for your fights</h2>
				<p className="m-0 text-sm text-white/70">
					Pick the fights you care about. Each build names the race that is never far behind the best in any of them: the smallest worst-case gap, not
					the best average. Show detail lays out every race against every fight, and how the gaps move with fight length.
				</p>
			</header>
			<div className="flex flex-wrap items-center gap-x-4 gap-y-2" data-testid="explorer-filters">
				{durations.length > 1 && (
					<div className="flex flex-wrap items-center gap-2" role="group" aria-label="Fight lengths">
						<span className="text-sm text-white/60">Fight length</span>
						{durations.map(seconds => (
							<Toggle key={seconds} on={lengths.has(seconds)} onClick={() => flip(lengths, seconds, setLengths)}>
								{fightLabel(seconds)}
							</Toggle>
						))}
					</div>
				)}
				{targets.length > 1 && (
					<div className="flex flex-wrap items-center gap-2" role="group" aria-label="Targets">
						<span className="text-sm text-white/60">Target</span>
						{targets.map(target => (
							<Toggle key={target.key} on={targetKeys.has(target.key)} onClick={() => flip(targetKeys, target.key, setTargetKeys)}>
								{target.short}
							</Toggle>
						))}
					</div>
				)}
			</div>
			{classes.map(playerClass => (
				<div key={playerClass} className="flex flex-col">
					<h3 className={clsx('m-0 text-lg', textClassName(playerClass.toLowerCase()))}>{playerClass}</h3>
					<ul className="m-0 list-none p-0">
						{groups
							.filter(group => group.class === playerClass)
							.map(group => (
								<BuildRow
									key={group.key}
									data={data}
									group={group}
									conditions={conditions}
									selected={selected}
									durations={durations}
									targets={targets}
								/>
							))}
					</ul>
				</div>
			))}
		</section>
	);
};
