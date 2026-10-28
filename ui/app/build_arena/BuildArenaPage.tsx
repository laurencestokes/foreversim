import i18n from '@i18n/config';
import { BUILD_INFO } from '@sim/constants/build_info';
import { textClassName } from '@sim/proto/utils';
import { Button } from '@ui-kit/Button';
import { ChoiceField } from '@ui-kit/ChoiceField';
import { Dialog } from '@ui-kit/Dialog';
import { Table, TableCell, TableHeaderCell, TableRow } from '@ui-kit/Table';
import { SearchField } from '@ui-kit/TextField';
import clsx from 'clsx';
import { useState } from 'react';

import { LandingFooter } from '../landing/LandingFooter';
import { LandingHeader } from '../landing/LandingHeader';
import { specName } from '../race_arena/raceArenaData';
import { RaceArenaSetupDialog } from '../race_arena/RaceArenaSetupDialog';
import {
	type BuildArenaBuild,
	type BuildArenaResults,
	type BuildArenaRow,
	type BuildStanding,
	differenceIsNoise,
	matchesBuildSearch,
	rankBuilds,
	requestForRow,
	roundedGap,
} from './buildArenaData';
import results from './results.json';

const t = (key: string, values?: Record<string, unknown>) => i18n.t(`build_arena:${key}`, values);
const number = (value: number, digits = 1) =>
	new Intl.NumberFormat(i18n.language, { minimumFractionDigits: digits, maximumFractionDigits: digits }).format(value);
const sourceDate = (source: BuildArenaResults['sources'][number] | undefined) =>
	source?.computedAt ?? (source?.computedWindow ? t('computedWindow', source.computedWindow) : t('dateUnrecorded'));
const COLORS: Record<string, string> = {
	S: 'border-brand text-brand',
	A: 'border-expansion text-expansion',
	B: 'border-success text-success',
	C: 'border-orange text-orange',
	D: 'border-danger text-danger',
};
const MOBILE_CELL = clsx('max-md:block max-md:min-w-0 max-md:border-0 max-md:p-0');
const MobileLabel = ({ label }: { label: string }) => (
	<span aria-hidden="true" className="mb-1 block text-xs font-normal text-white/60 md:hidden">
		{label}
	</span>
);
const humanName = (value: string) =>
	value
		.replace(/([a-z0-9])([A-Z])/g, '$1 $2')
		.replace(/_/g, ' ')
		.replace(/^./, char => char.toUpperCase());
const displayBuildName = (build: BuildArenaBuild) => (build.class === 'Mage' ? build.name.split(' / ').slice(-1)[0] : build.name);
const buildTitle = (build: BuildArenaBuild) => `${specName(build.spec)}: ${displayBuildName(build)}`;
const record = (value: unknown): Record<string, unknown> =>
	value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : {};

type RecordedNames = { items: Map<number, string>; spells: Map<number, string> };
const recordedNames = (database: unknown): RecordedNames => {
	const names: RecordedNames = { items: new Map(), spells: new Map() };
	const visit = (value: unknown, kind: 'items' | 'spells') => {
		if (!value || typeof value !== 'object') return;
		if (Array.isArray(value)) {
			value.forEach(child => visit(child, kind));
			return;
		}
		const entry = record(value);
		if (typeof entry.id === 'number' && typeof entry.name === 'string') names[kind].set(entry.id, entry.name);
		if (typeof entry.buffId === 'number' && typeof entry.buffName === 'string') names.spells.set(entry.buffId, entry.buffName);
		Object.entries(entry).forEach(([key, child]) => visit(child, /spell/i.test(key) ? 'spells' : kind));
	};
	visit(database, 'items');
	return names;
};

const SavedValues = ({ value, names, kind }: { value: unknown; names: RecordedNames; kind?: 'items' | 'spells' }) => {
	if (value === null || value === undefined) return <span>{t('none')}</span>;
	if (typeof value === 'boolean') return <span>{t(value ? 'yes' : 'no')}</span>;
	if (typeof value === 'number' && kind)
		return <span>{value === 0 ? t('none') : (names[kind].get(value) ?? t(kind === 'items' ? 'unknownItem' : 'unknownSpell', { id: value }))}</span>;
	if (typeof value !== 'object') return <span className="wrap-break-word">{typeof value === 'string' ? humanName(value) : String(value)}</span>;
	if (Array.isArray(value) && value.every(child => child === 0)) return <span>{t('none')}</span>;
	if (Array.isArray(value) && value.every(child => typeof child !== 'object'))
		return (
			<span className="wrap-break-word">
				{value
					.map(child =>
						typeof child === 'number' && kind
							? child === 0
								? t('none')
								: (names[kind].get(child) ?? t(kind === 'items' ? 'unknownItem' : 'unknownSpell', { id: child }))
							: typeof child === 'string'
								? humanName(child)
								: String(child),
					)
					.join(', ')}
			</span>
		);
	const entries = Object.entries(value).filter(([key]) => key !== 'apiVersion');
	if (!entries.length) return <span>{t('none')}</span>;
	return (
		<dl className="m-0 flex flex-col gap-2 border-l border-surface-border pl-4">
			{entries.map(([key, child]) => (
				<div key={key} className="min-w-0">
					<dt className="text-sm text-white/60">
						{Array.isArray(value) ? Number(key) + 1 : i18n.t(`build_arena:fields.${key}`, { defaultValue: humanName(key) })}
					</dt>
					<dd className="m-0">
						<SavedValues
							value={child}
							names={names}
							kind={
								key === 'consumes' || key === 'consumables' || /itemId$/i.test(key)
									? 'items'
									: /^(spellId|auraId|buffId)$/i.test(key)
										? 'spells'
										: kind
							}
						/>
					</dd>
				</div>
			))}
		</dl>
	);
};
const Estimate = ({ row }: { row: BuildArenaRow }) => (
	<span className="whitespace-nowrap tabular-nums">
		{number(row.dps)} <span className="text-white/60">± {number(1.96 * row.error)}</span>
	</span>
);

const BuildDetails = ({ standing, data, onClose }: { standing: BuildStanding; data: BuildArenaResults; onClose: () => void }) => {
	const [race, setRace] = useState(standing.row.race);
	const [setupOpen, setSetupOpen] = useState(false);
	const row = standing.races.find(candidate => candidate.race === race) ?? standing.row;
	const source = data.sources.find(candidate => candidate.id === row.sourceId);
	const request = requestForRow(standing.build, row);
	const raid = record(request.raid);
	const party = record((raid.parties as unknown[])?.[0]);
	const player = record((party.players as unknown[])?.[0]);
	const names = recordedNames(player.database ?? request.database);
	standing.build.consumableNames?.forEach(item => names.items.set(item.id, item.name));
	const settings = Object.fromEntries(
		Object.entries(player).filter(
			([key]) => !['equipment', 'itemSwap', 'talentsString', 'name', 'race', 'rotation', 'database', 'apiVersion', 'class'].includes(key),
		),
	);
	return (
		<Dialog
			open
			onOpenChange={open => !open && onClose()}
			title={buildTitle(standing.build)}
			size="lg"
			bodyClassName="min-w-0 wrap-anywhere"
			scrollContents>
			<p className="m-0 text-white/70">
				{specName(standing.build.spec)} · {standing.build.gear} · {standing.build.rotation}
			</p>
			{displayBuildName(standing.build) !== standing.build.name && (
				<p className="m-0 text-sm text-white/60">{t('recordedLabel', { name: standing.build.name })}</p>
			)}
			{!!standing.build.caveats.length && (
				<ul className="m-0 pl-4 text-orange">
					{standing.build.caveats.map(caveat => (
						<li key={caveat}>{caveat}</li>
					))}
				</ul>
			)}
			<div className="flex flex-wrap items-end gap-4">
				<Button variant="secondary" onClick={() => setSetupOpen(true)}>
					{t('gearTalents')}
				</Button>
				<ChoiceField
					label={t('downloadRace')}
					value={race}
					onChange={setRace}
					options={standing.races.map(candidate => ({ value: candidate.race, label: candidate.race }))}
				/>
				<Button
					variant="secondary"
					href={`data:application/json;charset=utf-8,${encodeURIComponent(JSON.stringify(request, null, 2))}`}
					download={`${standing.build.id}-${row.raceValue}-${row.seconds}-${row.target}.raid-sim-request.json`}>
					{t('downloadRequest')}
				</Button>
			</div>
			<p className="m-0 text-sm text-white/60">{t('requestHelp', { race: row.race, seconds: row.seconds, target: t(`targets.${row.target}`) })}</p>
			<section className="flex flex-col gap-2">
				<h3 className="m-0 text-lg">{t('raceResults')}</h3>
				<p className="m-0 text-sm text-white/70">{t('raceResultsHelp')}</p>
				<div className="min-w-0 md:overflow-x-auto">
					<Table aria-label={t('raceResults')} dense className="max-md:block">
						<thead className="max-md:sr-only">
							<TableRow>
								<TableHeaderCell scope="col">{t('race')}</TableHeaderCell>
								<TableHeaderCell scope="col">{t('dps')}</TableHeaderCell>
								<TableHeaderCell scope="col">{t('raceGap')}</TableHeaderCell>
								<TableHeaderCell scope="col">{t('precision')}</TableHeaderCell>
							</TableRow>
						</thead>
						<tbody className="max-md:grid max-md:gap-4">
							{standing.races.map(candidate => (
								<TableRow
									key={candidate.race}
									data-selected={candidate.race === row.race ? '' : undefined}
									className="max-md:grid max-md:grid-cols-2 max-md:gap-4 max-md:border max-md:border-surface-border max-md:p-4">
									<TableHeaderCell scope="row" wrap className="text-left max-md:col-span-2 max-md:border-0 max-md:p-0">
										{candidate.race}
									</TableHeaderCell>
									<TableCell className={MOBILE_CELL}>
										<MobileLabel label={t('dps')} />
										<Estimate row={candidate} />
									</TableCell>
									<TableCell className={MOBILE_CELL}>
										<MobileLabel label={t('raceGap')} />
										{number(roundedGap(candidate.dps, standing.races[0].dps), 2)}%
									</TableCell>
									<TableCell className={clsx(MOBILE_CELL, 'text-sm text-white/70 max-md:col-span-2')}>
										<MobileLabel label={t('precision')} />
										{differenceIsNoise(candidate, standing.races[0]) ? t('indistinguishableRace') : t('lowerEstimate')}
									</TableCell>
								</TableRow>
							))}
						</tbody>
					</Table>
				</div>
			</section>
			<section className="flex flex-col gap-2">
				<h3 className="m-0 text-lg">{t('settings')}</h3>
				<p className="m-0 text-sm text-white/70">{t('settingsHelp')}</p>
				<SavedValues value={{ ...settings, raidBuffs: raid.buffs, partyBuffs: party.buffs, targetDebuffs: raid.debuffs }} names={names} />
			</section>
			<details>
				<summary className="cursor-pointer text-brand">{t('recordedRotation')}</summary>
				<div className="min-w-0 pt-4">
					<SavedValues value={player.rotation} names={names} />
				</div>
			</details>
			<p className="m-0 text-xs text-white/60">
				{t('rowProvenance', {
					source: source?.id,
					iterations: number(row.iterations, 0),
					date: sourceDate(source),
				})}
			</p>
			<RaceArenaSetupDialog
				open={setupOpen}
				onOpenChange={setSetupOpen}
				title={t('setupTitle', { name: buildTitle(standing.build) })}
				gear={standing.build.gear}
				rotation={standing.build.rotation}
				setup={standing.build.setup}
				commit={source?.commit ?? data.verifiedCommit}
			/>
		</Dialog>
	);
};

export const BuildArenaPageView = ({ data }: { data: BuildArenaResults }) => {
	const [seconds, setSeconds] = useState(data.fight.seconds);
	const [target, setTarget] = useState('Mechanical');
	const [className, setClassName] = useState('');
	const [race, setRace] = useState('');
	const [search, setSearch] = useState('');
	const [detail, setDetail] = useState<BuildStanding>();
	const classes = [...new Set(data.builds.map(build => build.class))].sort();
	const races = [...new Set(data.rows.map(row => row.race))].sort();
	const ranking = rankBuilds(data, { seconds, target, className, race });
	const leader = ranking.standings[0];
	const visible = ranking.standings.filter(standing => matchesBuildSearch(standing, search));
	return (
		<div className="flex min-h-full flex-col">
			<LandingHeader />
			<main className="ui-landing-container flex min-w-0 grow flex-col gap-6 py-8 wrap-anywhere">
				<nav className="flex flex-wrap gap-4" aria-label={t('arenaNavigation')}>
					<Button href="/forever/" variant="link">
						{t('back')}
					</Button>
					<Button href="/forever/race_arena/" variant="link">
						{t('raceArena')}
					</Button>
				</nav>
				<header>
					<h1 className="m-0 text-3xl font-bold text-brand">{t('title')}</h1>
					<p className="mb-0">{t('intro', { count: data.builds.length })}</p>
				</header>
				<aside className="flex flex-col gap-2 border border-orange bg-orange/10 p-4" aria-label={t('limitationsTitle')}>
					<h2 className="m-0 text-lg text-orange">{t('limitationsTitle')}</h2>
					<p className="m-0">{t('limitations')}</p>
					<p className="m-0">{t('raidSupport')}</p>
				</aside>
				<section className="flex flex-col gap-4 border border-surface-border p-4" aria-label={t('controls')}>
					<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
						<ChoiceField
							label={t('duration')}
							value={seconds}
							onChange={setSeconds}
							options={data.fight.durations.map(value => ({ value, label: t('seconds', { count: value }) }))}
						/>
						<ChoiceField
							label={t('target')}
							value={target}
							onChange={setTarget}
							options={data.fight.targets.map(value => ({ value, label: t(`targets.${value}`) }))}
						/>
						<ChoiceField
							label={t('comparison')}
							value={className}
							onChange={setClassName}
							options={[{ value: '', label: t('allBuilds') }, ...classes.map(value => ({ value, label: value }))]}
						/>
						<ChoiceField
							label={t('raceMode')}
							value={race}
							onChange={setRace}
							options={[{ value: '', label: t('bestRace') }, ...races.map(value => ({ value, label: value }))]}
						/>
					</div>
					<p className="m-0 text-sm text-white/70">{race ? t('fixedRaceHelp') : t('bestRaceHelp')}</p>
					<SearchField label={t('search')} value={search} onChange={setSearch} placeholder={t('searchPlaceholder')} />
				</section>
				<section className="flex flex-col gap-4" aria-label={t('rankings')}>
					<div aria-live="polite" className="flex flex-col gap-2">
						<h2 className="m-0 text-xl">{t('encounterTitle', { seconds, target: t(`targets.${target}`) })}</h2>
						{leader && (
							<p className="m-0 text-white/70">
								{t('reference', {
									name: buildTitle(leader.build),
									race: leader.row.race,
									dps: number(leader.row.dps),
									count: visible.length,
									total: ranking.standings.length,
								})}
							</p>
						)}
						{!!ranking.omitted.length && (
							<p className="m-0 text-orange">
								{t('omitted', { count: ranking.omitted.length, race })} {ranking.omitted.map(build => build.name).join(', ')}.
							</p>
						)}
					</div>
					<ul className="m-0 flex list-none flex-wrap gap-x-6 gap-y-2 p-0" aria-label={t('tierLegend')}>
						{data.tiers.map((tier, index) => (
							<li key={tier.name} className="flex items-center gap-2">
								<span className={clsx('rounded-sm border px-2 font-bold', COLORS[tier.name])}>{tier.name}</span>
								<span className="text-sm text-white/70">
									{tier.maxBehind === undefined
										? t('tierLast', { lower: data.tiers[index - 1]?.maxBehind })
										: index === 0
											? t('tierFirst', { upper: tier.maxBehind })
											: t('tierRange', { lower: data.tiers[index - 1]?.maxBehind, upper: tier.maxBehind })}
								</span>
							</li>
						))}
					</ul>
					<p className="m-0 text-sm text-white/60">{t('estimateHelp')}</p>
					{visible.length ? (
						<div className="min-w-0 md:overflow-x-auto">
							<Table hover aria-label={t('rankings')} className="max-md:block">
								<thead className="max-md:sr-only">
									<TableRow>
										<TableHeaderCell scope="col">{t('tier')}</TableHeaderCell>
										<TableHeaderCell scope="col">{t('build')}</TableHeaderCell>
										<TableHeaderCell scope="col">{t('dps')}</TableHeaderCell>
										<TableHeaderCell scope="col">{t('gap')}</TableHeaderCell>
										<TableHeaderCell scope="col">{t('race')}</TableHeaderCell>
										<TableHeaderCell scope="col">{t('spread')}</TableHeaderCell>
										<TableHeaderCell scope="col">{t('details')}</TableHeaderCell>
									</TableRow>
								</thead>
								<tbody className="max-md:grid max-md:gap-4">
									{visible.map(standing => (
										<TableRow
											key={standing.build.id}
											data-tier={standing.tier}
											data-build={standing.build.id}
											className="max-md:grid max-md:grid-cols-2 max-md:gap-4 max-md:border max-md:border-surface-border max-md:p-4">
											<TableCell className={MOBILE_CELL}>
												<MobileLabel label={t('tier')} />
												<span className={clsx('inline-flex rounded-sm border px-2 py-1 text-xl font-bold', COLORS[standing.tier])}>
													{standing.tier}
												</span>
											</TableCell>
											<TableHeaderCell
												scope="row"
												wrap
												className="text-left max-md:-order-1 max-md:col-span-2 max-md:border-0 max-md:p-0 md:min-w-48">
												<span className={textClassName(standing.build.class.toLowerCase())}>{displayBuildName(standing.build)}</span>
												<span className="block text-xs font-normal text-white/60">
													{specName(standing.build.spec)} · {standing.build.gear}
												</span>
												{standing.build.class === 'Hunter' &&
													standing.build.caveats
														.filter(caveat => /melee|weav|movement/i.test(caveat))
														.map(caveat => (
															<span key={caveat} className="block max-w-72 text-xs font-normal text-orange">
																{caveat}
															</span>
														))}
												{standing.build.devGear && (
													<span className="block text-xs font-normal text-orange">{t('developmentGear')}</span>
												)}
												{standing.build.status !== 'Launched' && (
													<span className="block text-xs font-normal text-white/50">
														{t('status', { status: standing.build.status })}
													</span>
												)}
											</TableHeaderCell>
											<TableCell className={MOBILE_CELL}>
												<MobileLabel label={t('dps')} />
												<Estimate row={standing.row} />
												{standing.indistinguishable && <span className="block text-xs text-white/60">{t('indistinguishable')}</span>}
											</TableCell>
											<TableCell className={clsx(MOBILE_CELL, 'tabular-nums')}>
												<MobileLabel label={t('gap')} />
												{number(standing.behind, 2)}%
												{standing.boundaryUncertain && (
													<span
														className="block text-xs text-orange"
														title={t('boundaryHelp', {
															low: number(standing.gapInterval[0], 2),
															high: number(standing.gapInterval[1], 2),
														})}>
														{t('boundary')}
													</span>
												)}
											</TableCell>
											<TableCell className={MOBILE_CELL}>
												<MobileLabel label={t('race')} />
												{standing.row.race}
												{!race && standing.closeRaces.length > 1 && (
													<span className="block text-xs text-white/60">
														{t('closeRaces', { count: standing.closeRaces.length })}
													</span>
												)}
											</TableCell>
											<TableCell className={clsx(MOBILE_CELL, 'whitespace-nowrap tabular-nums')}>
												<MobileLabel label={t('spread')} />
												{number(standing.raceSpread, 2)}%
											</TableCell>
											<TableCell className={MOBILE_CELL}>
												<Button
													variant="secondary"
													size="sm"
													aria-label={t('inspectBuild', { name: buildTitle(standing.build) })}
													onClick={() => setDetail(standing)}>
													{t('inspect')}
												</Button>
											</TableCell>
										</TableRow>
									))}
								</tbody>
							</Table>
						</div>
					) : (
						<p role="status" className="border border-surface-border p-6">
							{t(search ? 'noSearch' : 'noResults')}
						</p>
					)}
				</section>
				<section className="flex flex-col gap-2 border border-surface-border p-4" aria-label={t('method')}>
					<h2 className="m-0 text-xl">{t('method')}</h2>
					<p className="m-0">
						{t('methodFight', { level: data.fight.targetLevel, armor: number(data.fight.targetArmor, 0), iterations: number(data.iterations, 0) })}
					</p>
					<p className="m-0">{t('methodNoise')}</p>
					<p className="m-0">{t('methodSelection')}</p>
					<p className="m-0">{t('methodTiers')}</p>
					{data.compatibility && (
						<p className="m-0">
							{data.compatibility.method}{' '}
							<Button
								variant="link"
								href={`https://github.com/laurencestokes/foreversim/blob/${BUILD_INFO.commit || 'master'}/${data.compatibility.evidence}`}
								target="_blank">
								{t('compatibilityEvidence')}
							</Button>
						</p>
					)}
					<p className="m-0 text-sm text-white/70">
						{t('published', { date: data.publishedAt })}{' '}
						<Button variant="link" href={`https://github.com/wowsims/forever/commit/${data.upstreamCommit}`} target="_blank">
							{t('upstream', { commit: data.upstreamCommit.slice(0, 7) })}
						</Button>
						{' · '}
						<Button variant="link" href={`https://github.com/laurencestokes/foreversim/commit/${data.verifiedCommit}`} target="_blank">
							{t('verified', { commit: data.verifiedCommit.slice(0, 7) })}
						</Button>
					</p>
					<ul className="m-0 flex flex-col gap-2 pl-4 text-sm text-white/70">
						{data.sources.map(source => (
							<li key={source.id}>
								{t(source.reused ? 'reusedSource' : 'computedSource', { id: source.id, date: sourceDate(source) })}{' '}
								<Button variant="link" href={`https://github.com/laurencestokes/foreversim/commit/${source.commit}`} target="_blank">
									{source.commit.slice(0, 7)}
								</Button>
								. {source.note}
								{source.publishedAt && <> {t('sourcePublished', { date: source.publishedAt })}</>}
							</li>
						))}
					</ul>
				</section>
				{!!data.excluded.length && (
					<section>
						<h2 className="m-0 text-lg">{t('excluded')}</h2>
						<ul className="mb-0 pl-4 text-sm text-white/70">
							{data.excluded.map(item => (
								<li key={item.spec}>
									{humanName(item.spec)}: {item.reason}
								</li>
							))}
						</ul>
					</section>
				)}
				{detail && (
					<BuildDetails
						key={`${detail.build.id}|${detail.row.race}|${detail.row.seconds}|${detail.row.target}`}
						standing={detail}
						data={data}
						onClose={() => setDetail(undefined)}
					/>
				)}
			</main>
			<LandingFooter />
		</div>
	);
};

export const BuildArenaPage = () => <BuildArenaPageView data={results as BuildArenaResults} />;
