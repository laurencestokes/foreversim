import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { isDeepStrictEqual } from 'node:util';

import { inputFingerprint, validateCatalog } from './identity.mjs';

export const durations = [90, 180, 360];
export const targets = ['Mechanical', 'Elemental', 'Beast'];
const repoFile = relative => {
	const root = fs.realpathSync(new URL('../../', import.meta.url));
	assert.ok(typeof relative === 'string' && relative && !path.isAbsolute(relative), 'Expected a repository-relative file path');
	const file = fs.realpathSync(path.resolve(root, relative));
	assert.ok(file.startsWith(root + path.sep) && fs.statSync(file).isFile(), 'Expected a file inside the repository');
	const relativePath = path.relative(root, file).split(path.sep).join('/');
	// Local runs can validate a plan before it is committed. Published links must resolve in the workflow checkout.
	const localFiles = process.env.GITHUB_ACTIONS === 'true' ? [] : ['--others', '--exclude-standard'];
	const listed = execFileSync('git', ['--literal-pathspecs', 'ls-files', '-z', '--cached', ...localFiles, '--', relativePath], {
		cwd: root,
		encoding: 'utf8',
	});
	assert.equal(listed, relativePath + '\0', 'Selection and evidence must be repository files');
	return file;
};

export function selectBuilds({ plan, manifest, previous, catalog }) {
	validateCatalog(catalog);
	const ids = manifest.builds.map(build => build.id).sort();
	assert.deepEqual(ids, catalog.builds.map(build => build.id).sort(), 'Selection manifest must cover the catalog');
	if (plan === undefined) return { rerun: ids, reuse: [], raw: [], sources: [] };
	assert.ok(plan && typeof plan === 'object', 'Expected a selection plan');
	assert.equal(plan.schemaVersion, 1, 'Unsupported selection plan');
	assert.equal(plan.baseVerifiedCommit, previous?.verifiedCommit, 'Selection base differs from the previous publication');
	assert.match(plan.baseVerifiedCommit ?? '', /^[a-f0-9]{40}$/, 'Selection requires the previous verified commit');
	assert.ok(plan.reason?.trim() && plan.evidence?.trim(), 'Selection requires a reason and compatibility evidence');
	repoFile(plan.evidence);
	assert.ok(Array.isArray(plan.rerun) && Array.isArray(plan.reuse), 'Selection requires explicit rerun and reuse arrays');
	assert.deepEqual([...plan.rerun, ...plan.reuse].sort(), ids, 'Rerun and reuse must partition every current build exactly once');
	assert.equal(previous.schemaVersion, 1, 'Unsupported previous publication');
	assert.equal(previous.iterations, 50000, 'Reuse requires 50000 iterations');
	assert.deepEqual(previous.fight, { seconds: 180, durations, targetLevel: 63, targetArmor: 3731, targets }, 'Previous fight settings differ');
	assert.equal(new Set(previous.builds.map(build => build.id)).size, previous.builds.length, 'Duplicate previous build ID');
	const sources = new Map();
	for (const source of previous.sources) {
		assert.ok(!sources.has(source.id), 'Duplicate previous source');
		sources.set(source.id, source);
	}
	const used = new Set();
	const raw = [];
	for (const id of plan.reuse) {
		const current = manifest.builds.find(build => build.id === id);
		const old = previous.builds.find(build => build.id === id);
		assert.ok(old, `New build ${id} must be rerun`);
		assert.equal(inputFingerprint(old.request), old.inputFingerprint, `Invalid previous fingerprint for ${id}`);
		assert.equal(inputFingerprint(current.request), current.inputFingerprint, `Invalid current fingerprint for ${id}`);
		assert.equal(old.inputFingerprint, current.inputFingerprint, `Changed request fingerprint for reused build ${id}`);
		for (const key of ['request', 'setup', 'spec', 'class', 'name', 'gear', 'devGear', 'rotation', 'consumableNames']) {
			assert.ok(isDeepStrictEqual(old[key], current[key]), `Changed ${key} for reused build ${id}`);
		}
		assert.ok(
			current.legalRaces?.length && new Set(current.legalRaces.map(race => race.value)).size === current.legalRaces.length,
			`Missing or duplicate legal race for ${id}`,
		);
		if (old.consumables !== undefined) assert.equal(old.consumables, current.consumables, `Changed consumables label for reused build ${id}`);
		const rows = previous.rows.filter(row => row.buildId === id);
		assert.equal(rows.length, durations.length * targets.length * current.legalRaces.length, `Incomplete previous coverage for ${id}`);
		for (const seconds of durations) {
			for (const target of targets) {
				const races = current.legalRaces.map(race => {
					const matches = rows.filter(row => row.seconds === seconds && row.target === target && row.raceValue === race.value);
					assert.equal(matches.length, 1, `Missing or duplicate previous matchup for ${id}/${seconds}/${target}/${race.value}`);
					const row = matches[0];
					assert.equal(row.race, race.name, `Changed legal race for ${id}`);
					assert.equal(row.inputFingerprint, current.inputFingerprint, `Changed previous row input for ${id}`);
					assert.equal(row.iterations, 50000, 'Reuse requires 50000 iterations per row');
					assert.ok(Number.isFinite(row.dps) && row.dps > 0 && Number.isFinite(row.error) && row.error >= 0, 'Invalid previous result');
					const source = sources.get(row.sourceId);
					assert.ok(source, `Missing original source for ${id}`);
					used.add(source.id);
					return {
						race: row.race,
						raceValue: row.raceValue,
						dps: row.dps,
						error: row.error,
						iterations: row.iterations,
						commit: source.commit,
						sourceId: source.id,
					};
				});
				const dates = [...new Set(races.map(row => sources.get(row.sourceId).computedAt))];
				raw.push({
					buildId: id,
					inputFingerprint: current.inputFingerprint,
					request: current.request,
					setup: current.setup,
					spec: current.spec,
					class: current.class,
					build: current.name,
					gear: current.gear,
					rotation: current.rotation,
					...(current.consumables !== undefined ? { consumables: current.consumables } : {}),
					consumableNames: current.consumableNames,
					talents: current.request.raid.parties[0].players[0].talentsString,
					seconds,
					target,
					iterations: 50000,
					...(dates.length === 1 && dates[0] ? { computedAt: dates[0] } : {}),
					races,
				});
			}
		}
	}
	return {
		rerun: [...plan.rerun],
		reuse: [...plan.reuse],
		raw,
		sources: [...used].map(id => ({ ...sources.get(id), reused: true, publishedAt: sources.get(id).publishedAt ?? previous.publishedAt })),
	};
}

export function combineResults({ manifest, fresh, selection, plan, workflowUrl }) {
	const rerun = new Set(selection.rerun);
	const sources = new Map(selection.sources.map(source => [source.id, source]));
	const measured = fresh.map(list => {
		assert.ok(rerun.has(list.buildId), `Undeclared fresh build ${list.buildId}`);
		assert.ok(list.computedAt && Number.isFinite(Date.parse(list.computedAt)), 'Fresh result requires its computation time');
		const id = `${manifest.commit}@${list.computedAt}`;
		assert.ok(!sources.get(id)?.reused, 'Fresh and reused source IDs collide');
		if (list.sourceId !== undefined) assert.equal(list.sourceId, id, 'Fresh result identifies a different source');
		sources.set(id, { id, commit: manifest.commit, computedAt: list.computedAt, reused: false, note: workflowUrl });
		const races = list.races.map(row => {
			assert.equal(row.commit, manifest.commit, 'Fresh result revision differs');
			if (row.sourceId !== undefined) assert.equal(row.sourceId, id, 'Fresh row identifies a different source');
			return { ...row, sourceId: id };
		});
		return { ...list, races };
	});
	const provenance = { sources: [...sources.values()] };
	if (plan?.reuse.length) {
		provenance.reuse = plan;
		provenance.compatibility = {
			verifiedCommit: manifest.commit,
			sourceCommits: [...new Set(provenance.sources.map(source => source.commit))],
			inputs: manifest.builds.map(({ id, inputFingerprint }) => ({ id, inputFingerprint })),
			method: plan.reason,
			evidence: plan.evidence,
		};
	}
	return { raw: [...selection.raw, ...measured], provenance };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
	const args = Object.fromEntries(Array.from({ length: (process.argv.length - 2) / 2 }, (_, index) => process.argv.slice(2 + index * 2, 4 + index * 2)));
	const read = file => JSON.parse(fs.readFileSync(file, 'utf8'));
	const manifest = read(args['--manifest']);
	const catalog = read(new URL('./catalog.json', import.meta.url));
	let plan, previous;
	if (args['--plan']) {
		assert.ok(args['--plan'].endsWith('.json'), 'Selection plan must be a repository JSON file');
		plan = read(repoFile(args['--plan']));
		previous = read(args['--previous'] ?? 'ui/app/build_arena/results.json');
	}
	const selection = selectBuilds({ plan, manifest, previous, catalog });
	if (args['--fresh']) {
		const result = combineResults({ manifest, fresh: read(args['--fresh']), selection, plan, workflowUrl: args['--workflow'] });
		fs.writeFileSync(args['--raw-out'], JSON.stringify(result.raw));
		fs.writeFileSync(args['--provenance-out'], JSON.stringify(result.provenance));
	} else {
		const inputs = read(args['--inputs']);
		assert.deepEqual(inputs.map(build => build.id).sort(), manifest.builds.map(build => build.id).sort(), 'Export and manifest coverage differs');
		fs.writeFileSync(args['--out'], JSON.stringify(inputs.filter(build => selection.rerun.includes(build.id))));
		console.log(`Rerun ${selection.rerun.length} builds across all nine scenarios; reuse ${selection.reuse.length}`);
	}
}
