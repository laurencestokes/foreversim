import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

import { canonical, inputFingerprint, resolveBuildId, validateCatalog } from './identity.mjs';

const durations = [90, 180, 360];
const targets = ['Mechanical', 'Elemental', 'Beast'];
const tiers = [{ name: 'S', maxBehind: 5 }, { name: 'A', maxBehind: 10 }, { name: 'B', maxBehind: 20 }, { name: 'C', maxBehind: 30 }, { name: 'D' }];
const revision = value => assert.match(value ?? '', /^[a-f0-9]{40}$/, 'A full simulator commit is required');
const date = (value, label) =>
	assert.ok(typeof value === 'string' && /^\d{4}-\d\d-\d\dT/.test(value) && Number.isFinite(Date.parse(value)), `Invalid ${label}`);

export function publish({ manifest, raw, provenance, catalog, upstreamCommit, verifiedCommit, publishedAt = new Date().toISOString() }) {
	validateCatalog(catalog);
	revision(upstreamCommit);
	revision(verifiedCommit);
	date(publishedAt, 'publication date');
	assert.equal(manifest.schemaVersion, 1, 'Unsupported manifest schema');
	assert.equal(manifest.commit, verifiedCommit, 'Manifest revision differs from the verified engine');
	assert.equal(manifest.iterations, 50000, 'Publication requires 50000 iterations');
	assert.ok(Array.isArray(raw) && raw.length > 0, 'Missing raw lists');
	assert.ok(Array.isArray(manifest.builds) && manifest.builds.length > 0, 'Missing manifest builds');
	const builds = new Map();
	for (const build of manifest.builds) {
		assert.equal(resolveBuildId(catalog, build.spec, build.name), build.id, `Catalog ID differs for ${build.name}`);
		assert.ok(!builds.has(build.id), `Duplicate manifest build ${build.id}`);
		assert.equal(inputFingerprint(build.request), build.inputFingerprint, `Changed manifest input for ${build.id}`);
		assert.ok(build.setup?.equipment?.length && build.setup?.talents?.length, `Missing setup for ${build.id}`);
		assert.ok(
			typeof build.status === 'string' && typeof build.devGear === 'boolean' && Array.isArray(build.caveats),
			`Missing preset status for ${build.id}`,
		);
		assert.ok(
			Array.isArray(build.consumableNames) && build.consumableNames.every(item => Number.isInteger(item.id) && item.id > 0 && item.name),
			`Missing consumable names for ${build.id}`,
		);
		const request = build.request;
		const player = request.raid?.parties?.[0]?.players?.[0];
		assert.equal(request.raid?.parties?.length, 1, 'Expected one party');
		assert.equal(request.raid?.parties?.[0]?.players?.length, 1, 'Expected one player');
		assert.ok(player?.equipment?.items?.length && player.rotation?.priorityList?.length && player.talentsString, `Incomplete request for ${build.id}`);
		assert.ok(player.buffs && request.raid.buffs && request.raid.debuffs && request.raid.parties[0].buffs, `Missing baseline buffs for ${build.id}`);
		assert.equal(request.simOptions?.iterations, 50000, 'Request requires 50000 iterations');
		assert.equal(String(request.simOptions?.randomSeed), '101', 'Unexpected random seed');
		assert.equal(request.encounter?.duration, 180, 'Request template requires 180 seconds');
		assert.equal(request.encounter?.targets?.length, 1, 'Expected one target');
		assert.equal(request.encounter.targets[0].mobType, 'MobTypeMechanical', 'Request template requires Mechanical');
		assert.equal(request.encounter.targets[0].level, 63, 'Unexpected target level');
		assert.equal(request.encounter.targets[0].stats?.[27], 3731, 'Unexpected target armor');
		assert.ok(build.legalRaces?.length, `Missing legal races for ${build.id}`);
		assert.equal(new Set(build.legalRaces.map(race => race.value)).size, build.legalRaces.length, `Duplicate legal race for ${build.id}`);
		assert.equal(new Set(build.legalRaces.map(race => race.name)).size, build.legalRaces.length, `Duplicate race name for ${build.id}`);
		assert.ok(
			build.legalRaces.every(race => /^Race[A-Za-z]+$/.test(race.value) && race.name),
			`Invalid legal race for ${build.id}`,
		);
		assert.ok(
			build.legalRaces.some(race => race.value === player.race),
			`Illegal base race for ${build.id}`,
		);
		assert.equal(player.class, `Class${build.class}`, `Request class differs for ${build.id}`);
		builds.set(build.id, build);
	}
	assert.deepEqual([...builds.keys()].sort(), catalog.builds.map(build => build.id).sort(), 'Manifest does not cover the catalog');
	const sources = new Map();
	assert.ok(provenance.sources?.length, 'Missing source provenance');
	for (const source of provenance.sources) {
		assert.ok(source.id && !sources.has(source.id), `Duplicate source ${source.id}`);
		revision(source.commit);
		assert.equal(typeof source.reused, 'boolean', 'Source must state whether it was reused');
		assert.ok(source.note?.trim(), 'Source requires a provenance note');
		if (source.publishedAt) date(source.publishedAt, 'source publication date');
		if (source.computedAt) date(source.computedAt, 'source computation date');
		if (source.computedWindow) {
			date(source.computedWindow.start, 'computation window start');
			date(source.computedWindow.end, 'computation window end');
			assert.ok(Date.parse(source.computedWindow.start) <= Date.parse(source.computedWindow.end), 'Invalid computation window');
		}
		assert.ok(source.reused || source.computedAt, 'A new source requires its computation date');
		if (source.reused) assert.ok(source.publishedAt, 'A reused source requires its original publication date');
		sources.set(source.id, source);
	}
	if ([...sources.values()].some(source => source.commit !== verifiedCommit)) {
		const proof = provenance.compatibility;
		assert.equal(proof?.verifiedCommit, verifiedCommit, 'Mixed engines require current compatibility evidence');
		assert.ok(proof.method?.trim() && proof.evidence?.trim(), 'Missing compatibility method or evidence');
		assert.deepEqual(
			[...new Set(proof.sourceCommits)].sort(),
			[...new Set([...sources.values()].map(source => source.commit))].sort(),
			'Compatibility does not cover every source engine',
		);
		assert.deepEqual(
			[...proof.inputs].sort((a, b) => a.id.localeCompare(b.id)),
			[...builds.values()].map(build => ({ id: build.id, inputFingerprint: build.inputFingerprint })).sort((a, b) => a.id.localeCompare(b.id)),
			'Compatibility input fingerprints differ',
		);
	}
	const rows = [];
	const seen = new Set();
	const seenLists = new Set();
	const usedSources = new Set();
	for (const list of raw) {
		const build = builds.get(list.buildId);
		assert.ok(build, `Unknown build ${list.buildId}`);
		assert.equal(list.inputFingerprint, build.inputFingerprint, `Changed input fingerprint for ${build.id}`);
		assert.equal(inputFingerprint(list.request), build.inputFingerprint, `Changed raw request for ${build.id}`);
		assert.equal(list.spec, build.spec, `Raw spec differs for ${build.id}`);
		assert.equal(list.class, build.class, `Raw class differs for ${build.id}`);
		assert.equal(resolveBuildId(catalog, list.spec, list.build), build.id, `Raw build alias differs for ${build.id}`);
		assert.equal(list.gear, build.gear, `Raw gear preset differs for ${build.id}`);
		assert.equal(list.rotation, build.rotation, `Raw rotation preset differs for ${build.id}`);
		assert.equal(list.talents, build.request.raid.parties[0].players[0].talentsString, `Raw talents differ for ${build.id}`);
		assert.equal(canonical(list.setup), canonical(build.setup), `Raw setup differs for ${build.id}`);
		assert.equal(canonical(list.consumableNames), canonical(build.consumableNames), `Raw consumable names differ for ${build.id}`);
		assert.equal(list.iterations, 50000, 'Raw list requires 50000 iterations');
		assert.ok(durations.includes(list.seconds) && targets.includes(list.target), `Unexpected matchup for ${build.id}`);
		const listKey = `${build.id}|${list.seconds}|${list.target}`;
		assert.ok(!seenLists.has(listKey), `Duplicate raw list ${listKey}`);
		seenLists.add(listKey);
		assert.ok(Array.isArray(list.races) && list.races.length > 0, `Missing race rows for ${build.id}`);
		for (const row of list.races) {
			const race = build.legalRaces.find(race => race.name === row.race && race.value === row.raceValue);
			assert.ok(race, `Illegal race ${row.race} / ${row.raceValue} for ${build.id}`);
			assert.equal(row.iterations, 50000, 'Raw row requires 50000 iterations');
			assert.ok(Number.isFinite(row.dps) && row.dps > 0, 'Invalid raw DPS');
			assert.ok(Number.isFinite(row.error) && row.error >= 0, 'Invalid raw standard error');
			revision(row.commit);
			const sourceId = row.sourceId ?? list.sourceId;
			const matches = sourceId
				? [sources.get(sourceId)].filter(Boolean)
				: [...sources.values()].filter(
						source => source.commit === row.commit && (!list.computedAt || !source.computedAt || source.computedAt === list.computedAt),
					);
			assert.equal(matches.length, 1, `Ambiguous or missing source for ${build.id} / ${row.race}`);
			const source = matches[0];
			assert.equal(source.commit, row.commit, 'Row revision differs from its source');
			if (!source.reused) assert.equal(list.computedAt, source.computedAt, 'Computation date differs from its source');
			const key = `${build.id}|${list.seconds}|${list.target}|${race.value}`;
			assert.ok(!seen.has(key), `Duplicate matchup ${key}`);
			seen.add(key);
			usedSources.add(source.id);
			rows.push({
				buildId: build.id,
				inputFingerprint: build.inputFingerprint,
				seconds: list.seconds,
				target: list.target,
				race: race.name,
				raceValue: race.value,
				dps: row.dps,
				error: row.error,
				iterations: row.iterations,
				sourceId: source.id,
			});
		}
	}
	for (const build of builds.values())
		for (const seconds of durations)
			for (const target of targets)
				for (const race of build.legalRaces) {
					assert.ok(
						seen.has(`${build.id}|${seconds}|${target}|${race.value}`),
						`Missing matchup ${build.id} / ${seconds} / ${target} / ${race.name}`,
					);
				}
	assert.equal(usedSources.size, sources.size, 'Unused provenance source');
	return {
		schemaVersion: 1,
		publishedAt,
		upstreamCommit,
		verifiedCommit,
		iterations: 50000,
		fight: { seconds: 180, durations, targetLevel: 63, targetArmor: 3731, targets },
		tiers,
		sources: [...sources.values()],
		...(provenance.compatibility ? { compatibility: provenance.compatibility } : {}),
		builds: [...builds.values()].map(({ legalRaces, ...build }) => build),
		rows,
		excluded: catalog.excluded ?? [],
	};
}

export function writeAtomic(destination, value) {
	fs.mkdirSync(path.dirname(destination), { recursive: true });
	const temporary = `${destination}.${process.pid}.tmp`;
	try {
		fs.writeFileSync(temporary, JSON.stringify(value, null, '\t') + '\n', { flag: 'wx' });
		fs.renameSync(temporary, destination);
	} finally {
		if (fs.existsSync(temporary)) fs.unlinkSync(temporary);
	}
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
	const args = Object.fromEntries(Array.from({ length: (process.argv.length - 2) / 2 }, (_, index) => process.argv.slice(2 + index * 2, 4 + index * 2)));
	assert.ok(
		args['--manifest'] && args['--raw'] && args['--provenance'] && args['--upstream'] && args['--verified'] && args['--out'],
		'Usage: node tools/build_arena/publish.mjs --manifest manifest.json --raw presets.json --provenance sources.json --upstream <commit> --verified <commit> --out results.json [--catalog catalog.json] [--published-at <date>]',
	);
	const read = file => JSON.parse(fs.readFileSync(file, 'utf8'));
	const result = publish({
		manifest: read(args['--manifest']),
		raw: read(args['--raw']),
		provenance: read(args['--provenance']),
		catalog: read(args['--catalog'] ?? new URL('./catalog.json', import.meta.url)),
		upstreamCommit: args['--upstream'],
		verifiedCommit: args['--verified'],
		publishedAt: args['--published-at'],
	});
	writeAtomic(args['--out'], result);
	console.log(`Published ${result.builds.length} builds and ${result.rows.length} verified matchups`);
}
