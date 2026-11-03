import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';

import { inputFingerprint, resolveBuildId, validateCatalog } from './identity.mjs';
import { publish, writeAtomic } from './publish.mjs';
import { combineResults, selectBuilds } from './select.mjs';

const commit = 'a'.repeat(40);
const date = '2026-10-28T12:00:00Z';
function fixture() {
	const catalog = { schemaVersion: 1, builds: [{ id: 'test-build', spec: 'mage', aliases: ['Test', 'Renamed'] }], excluded: [] };
	const request = {
		raid: {
			parties: [
				{
					players: [
						{
							name: 'Display',
							class: 'ClassMage',
							race: 'RaceHuman',
							talentsString: '1',
							equipment: { items: [{ id: 1 }] },
							rotation: { priorityList: [{ action: {} }] },
							buffs: { kings: true },
							bonusStats: { stats: [1, 0], pseudoStats: [2, 0] },
							database: { items: [{ id: 1, stats: [4, 0] }] },
						},
					],
					buffs: { strength: true },
				},
			],
			buffs: { intellect: true },
			debuffs: { armor: true },
		},
		encounter: {
			duration: 180,
			targets: [{ level: 63, stats: Array.from({ length: 28 }, (_, index) => (index === 27 ? 3731 : 0)), mobType: 'MobTypeMechanical' }],
		},
		simOptions: { iterations: 50000, randomSeed: '101' },
	};
	const build = {
		consumableNames: [],
		id: 'test-build',
		inputFingerprint: inputFingerprint(request),
		spec: 'mage',
		class: 'Mage',
		name: 'Arcane / Test',
		gear: 'Gear',
		rotation: 'Rotation',
		status: 'Alpha',
		devGear: false,
		caveats: [],
		setup: { equipment: [{ slot: 'Head' }], talents: [{ name: 'Tree' }] },
		request,
		legalRaces: [
			{ name: 'Human', value: 'RaceHuman' },
			{ name: 'Gnome', value: 'RaceGnome' },
		],
	};
	const raw = [90, 180, 360].flatMap(seconds =>
		['Mechanical', 'Elemental', 'Beast'].map(target => ({
			buildId: build.id,
			consumableNames: [],
			spec: build.spec,
			class: build.class,
			build: build.name,
			gear: build.gear,
			rotation: build.rotation,
			talents: '1',
			setup: structuredClone(build.setup),
			inputFingerprint: build.inputFingerprint,
			request: structuredClone(request),
			seconds,
			target,
			iterations: 50000,
			computedAt: date,
			races: build.legalRaces.map(race => ({
				race: race.name,
				raceValue: race.value,
				dps: 123.1234567890123,
				error: 0.12345678901234,
				iterations: 50000,
				commit,
			})),
		})),
	);
	return {
		manifest: { schemaVersion: 1, commit, iterations: 50000, builds: [build] },
		raw,
		catalog,
		provenance: { sources: [{ id: 'current', commit, computedAt: date, reused: false, note: 'Full run.' }] },
		upstreamCommit: commit,
		verifiedCommit: commit,
		publishedAt: date,
	};
}

test('publishes every matchup with raw precision and exact saved requests', () => {
	const input = fixture();
	const result = publish(input);
	assert.equal(result.rows.length, 18);
	assert.equal(result.rows[0].dps, 123.1234567890123);
	assert.equal(result.rows[0].error, 0.12345678901234);
	assert.deepEqual(result.builds[0].request, input.manifest.builds[0].request);
	assert.equal(result.builds[0].legalRaces, undefined);
	assert.equal(result.rows[0].sourceId, 'current');
});

test('rejects incomplete coverage and duplicate rows', () => {
	const missing = fixture();
	missing.raw.pop();
	assert.throws(() => publish(missing), /Missing matchup/);
	const duplicate = fixture();
	duplicate.raw.push(duplicate.raw[0]);
	assert.throws(() => publish(duplicate), /Duplicate raw list/);
	const race = fixture();
	race.raw[0].races.pop();
	assert.throws(() => publish(race), /Missing matchup/);
});

test('keeps draft sources with fresh launch results and refuses a different upstream base', () => {
	const input = fixture();
	assert.equal(publish(input).launchPreview, undefined);
	const preview = { launchDate: '2026-11-04', upstreamCommit: commit, drafts: [{ pullRequest: 824, commit: 'b'.repeat(40) }] };
	input.manifest.launchPreview = preview;
	assert.deepEqual(publish(input).launchPreview, preview);
	preview.notesUrl = 'https://us.forums.blizzard.com/en/wow/t/2360696/9';
	assert.deepEqual(publish(input).launchPreview, preview);
	preview.notesUrl = 'https://example.com/tuning';
	assert.throws(() => publish(input), /Blizzard source/);
	delete preview.notesUrl;
	input.provenance.sources[0].reused = true;
	assert.throws(() => publish(input), /fresh measurements/);
	input.provenance.sources[0].reused = false;
	preview.upstreamCommit = 'c'.repeat(40);
	assert.throws(() => publish(input), /upstream revision differs/);
});

test('rejects duplicate lists even when their race rows are disjoint', () => {
	const input = fixture();
	const duplicate = structuredClone(input.raw[0]);
	input.raw[0].races = [input.raw[0].races[0]];
	duplicate.races = [duplicate.races[1]];
	input.raw.push(duplicate);
	assert.throws(() => publish(input), /Duplicate raw list/);
});

test('rejects raw setup and preset metadata that disagree with the measured request', () => {
	for (const mutate of [
		input => {
			input.raw[0].setup.equipment[0].slot = 'Wrong';
		},
		input => {
			input.raw[0].talents = '2';
		},
		input => {
			input.raw[0].gear = 'Other';
		},
		input => {
			input.raw[0].rotation = 'Other';
		},
	]) {
		const input = fixture();
		mutate(input);
		assert.throws(() => publish(input), /Raw .* differ/);
	}
});

test('rejects unknown builds, illegal races and non-finite metrics', () => {
	for (const mutate of [
		input => {
			input.raw[0].buildId = 'missing';
		},
		input => {
			input.raw[0].races[0].raceValue = 'RaceOrc';
		},
		input => {
			input.raw[0].races[0].dps = NaN;
		},
		input => {
			input.raw[0].races[0].error = Infinity;
		},
		input => {
			input.raw[0].races[0].error = -1;
		},
	]) {
		const input = fixture();
		mutate(input);
		assert.throws(() => publish(input));
	}
});

test('rejects smoke data and changed requests even with a copied fingerprint', () => {
	for (const mutate of [
		input => {
			input.manifest.iterations = 8;
		},
		input => {
			input.raw[0].iterations = 8;
		},
		input => {
			input.raw[0].races[0].iterations = 8;
		},
		input => {
			input.raw[0].request.raid.parties[0].players[0].database.items[0].stats[0] = 99;
		},
		input => {
			input.raw[0].inputFingerprint = 'sha256:copied';
		},
	]) {
		const input = fixture();
		mutate(input);
		assert.throws(() => publish(input));
	}
});

test('catalog IDs survive deliberate aliases and Mage group headings', () => {
	const { catalog } = fixture();
	assert.equal(resolveBuildId(catalog, 'mage', 'Other group / Renamed'), 'test-build');
	assert.throws(() => resolveBuildId(catalog, 'mage', 'Unknown'));
	assert.throws(() => validateCatalog({ ...catalog, builds: [...catalog.builds, catalog.builds[0]] }), /Duplicate catalog/);
	const changed = fixture();
	changed.catalog.builds.push({ id: 'new-build', spec: 'warrior', aliases: ['New'] });
	assert.throws(() => publish(changed), /Manifest does not cover the catalog/);
});

test('fingerprints ignore display names, object ordering and stat-vector zero tails', () => {
	const request = fixture().manifest.builds[0].request;
	const renamed = structuredClone(request);
	renamed.raid.parties[0].players[0].name = 'Renamed';
	renamed.raid.parties[0].players[0].bonusStats.stats.push(0, 0);
	renamed.raid.parties[0].players[0].bonusStats.pseudoStats.push(0, 0);
	renamed.raid.parties[0].players[0].database.items[0].stats.push(0);
	assert.equal(inputFingerprint(renamed), inputFingerprint(request));
	assert.equal(inputFingerprint(Object.fromEntries(Object.entries(request).reverse())), inputFingerprint(request));
	renamed.raid.parties[0].players[0].bonusStats.pseudoStats.push(1);
	assert.notEqual(inputFingerprint(renamed), inputFingerprint(request));
	const changed = structuredClone(request);
	changed.raid.buffs.intellect = false;
	assert.notEqual(inputFingerprint(changed), inputFingerprint(request));
});

test('historical engines require explicit evidence covering every input and source commit', () => {
	const input = fixture();
	const original = 'b'.repeat(40);
	input.provenance.sources = [{ id: 'old', commit: original, reused: true, publishedAt: date, note: 'Exact computation time was not recorded.' }];
	for (const list of input.raw) {
		delete list.computedAt;
		for (const row of list.races) row.commit = original;
	}
	assert.throws(() => publish(input), /compatibility evidence/);
	input.provenance.compatibility = {
		verifiedCommit: commit,
		sourceCommits: [original],
		inputs: input.manifest.builds.map(({ id, inputFingerprint }) => ({ id, inputFingerprint })),
		method: 'Audited inputs and engine, then checked results.',
		evidence: 'docs/evidence.md',
	};
	assert.equal(publish(input).sources[0].computedAt, undefined);
	input.provenance.compatibility.inputs[0].inputFingerprint = 'changed';
	assert.throws(() => publish(input), /fingerprints differ/);
});

test('rejects source revision and computation-date mismatches', () => {
	const input = fixture();
	input.raw[0].computedAt = '2026-10-27T12:00:00Z';
	assert.throws(() => publish(input), /source/);
	const revision = fixture();
	revision.raw[0].races[0].commit = 'short';
	assert.throws(() => publish(revision), /full simulator commit/);
	const unknown = fixture();
	unknown.raw[0].sourceId = 'missing';
	assert.throws(() => publish(unknown), /source/);
});

test('writes a complete JSON file and replaces an existing publication', () => {
	const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'build-arena-'));
	try {
		const destination = path.join(directory, 'results.json');
		writeAtomic(destination, { before: true });
		const result = publish(fixture());
		writeAtomic(destination, result);
		assert.deepEqual(JSON.parse(fs.readFileSync(destination, 'utf8')), result);
		assert.deepEqual(fs.readdirSync(directory), ['results.json']);
	} finally {
		fs.rmSync(directory, { recursive: true, force: true });
	}
});

function mixedFixture() {
	const input = fixture();
	const previous = structuredClone(publish(input));
	const nextCommit = 'b'.repeat(40);
	const nextDate = '2026-11-03T12:00:00Z';
	input.manifest.commit = nextCommit;
	input.manifest.launchPreview = { launchDate: '2026-11-04', upstreamCommit: commit, drafts: [{ pullRequest: 824, commit: nextCommit }] };
	const newBuild = { ...structuredClone(input.manifest.builds[0]), id: 'new-build', name: 'Fire / Other' };
	input.manifest.builds.push(newBuild);
	input.catalog.builds.push({ id: newBuild.id, spec: 'mage', aliases: ['Other'] });
	const fresh = input.raw.map(list => ({
		...list,
		buildId: newBuild.id,
		build: newBuild.name,
		computedAt: nextDate,
		races: list.races.map(row => ({ ...row, dps: row.dps + 10, commit: nextCommit })),
	}));
	const plan = {
		schemaVersion: 1,
		baseVerifiedCommit: commit,
		rerun: [newBuild.id],
		reuse: ['test-build'],
		reason: 'The engine change affects the new build only.',
		evidence: 'docs/build-arena.md',
	};
	return { ...input, fresh, plan, previous, verifiedCommit: nextCommit, publishedAt: nextDate };
}

const merge = input => combineResults({ ...input, selection: selectBuilds(input), workflowUrl: 'Fresh workflow run.' });

test('selects only the new build and publishes every fight with its original source', () => {
	const input = mixedFixture();
	const selected = selectBuilds(input);
	assert.deepEqual(selected.rerun, ['new-build']);
	assert.deepEqual(selectBuilds({ ...input, plan: undefined }).rerun, ['new-build', 'test-build']);
	const result = publish({ ...input, ...merge(input) });
	assert.deepEqual(
		result.rows.filter(row => row.buildId === 'test-build'),
		input.previous.rows,
	);
	assert.equal(result.rows.filter(row => row.buildId === 'new-build').length, 18);
	assert.deepEqual(
		result.sources.find(source => source.id === 'current'),
		{ ...input.previous.sources[0], reused: true, publishedAt: input.previous.publishedAt },
	);
	assert.deepEqual(result.reuse, input.plan);
	const again = selectBuilds({ ...input, previous: result, plan: { ...input.plan, baseVerifiedCommit: result.verifiedCommit } });
	assert.deepEqual(again.sources, selected.sources);
});

test('rejects undeclared, incomplete and changed inputs before benchmarking', () => {
	for (const mutate of [
		input => {
			input.plan.baseVerifiedCommit = input.verifiedCommit;
		},
		input => {
			input.plan.rerun = [];
		},
		input => {
			input.plan.reuse.push('new-build');
		},
		input => {
			[input.plan.rerun, input.plan.reuse] = [input.plan.reuse, input.plan.rerun];
		},
		input => {
			input.plan.evidence = '';
		},
		input => {
			input.plan.evidence = '.git';
		},
		input => {
			input.manifest.builds[0].request.raid.parties[0].players[0].name = 'Other';
		},
		input => {
			input.manifest.builds[0].inputFingerprint = 'changed';
		},
		input => {
			input.manifest.builds[0].gear = 'Other';
		},
		input => {
			input.manifest.builds[0].setup.equipment[0].slot = 'Other';
		},
		input => {
			input.manifest.builds[0].legalRaces.pop();
		},
		input => {
			input.manifest.builds[0].legalRaces[0].name = 'Other';
		},
		input => {
			input.previous.rows[0].sourceId = 'unknown';
		},
		input => {
			input.previous.rows[0].iterations = 8;
		},
		input => {
			input.previous.rows[0] = input.previous.rows[1];
		},
	]) {
		const input = mixedFixture();
		mutate(input);
		assert.throws(() => selectBuilds(input));
	}
});

test('rejects altered measurements, missing fresh fights and reuse outside the plan', () => {
	const input = mixedFixture();
	assert.throws(() => combineResults({ ...input, fresh: input.raw, selection: selectBuilds(input) }), /Undeclared fresh build/);
	input.fresh[0].races[0].sourceId = 'current';
	assert.throws(() => merge(input), /identifies a different source/);
	for (const mutate of [
		result => {
			result.raw.find(list => list.buildId === 'test-build').races[0].dps += 1;
		},
		result => {
			result.provenance.sources.find(source => source.reused).computedAt = input.publishedAt;
		},
		result => {
			result.raw.pop();
		},
		result => {
			result.raw.at(-1).races.pop();
		},
		result => {
			delete result.provenance.reuse;
		},
		result => {
			result.provenance.compatibility.method = 'Different';
		},
	]) {
		const run = mixedFixture();
		const combined = merge(run);
		mutate(combined);
		assert.throws(() => publish({ ...run, ...combined }));
	}
});
