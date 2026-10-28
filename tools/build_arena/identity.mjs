import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';

export function canonical(value) {
	if (Array.isArray(value)) return `[${value.map(canonical).join(',')}]`;
	if (value && typeof value === 'object') {
		return `{${Object.keys(value)
			.sort()
			.map(key => `${JSON.stringify(key)}:${canonical(value[key])}`)
			.join(',')}}`;
	}
	return JSON.stringify(value).replace(/[<>&\u2028\u2029]/g, char => `\\u${char.charCodeAt(0).toString(16).padStart(4, '0')}`);
}

// Only display names are omitted. The request includes the database and the complete baseline buffs.
export function inputFingerprint(request) {
	const input = structuredClone(request);
	for (const party of input.raid?.parties ?? []) for (const player of party.players ?? []) delete player.name;
	for (const target of input.encounter?.targets ?? []) delete target.name;
	trimStatVectors(input);
	return `sha256:${createHash('sha256').update(canonical(input)).digest('hex')}`;
}

// Missing stat-vector entries are zero in protobuf. Their serialized length has no combat effect.
function trimStatVectors(value) {
	if (!value || typeof value !== 'object') return;
	for (const [key, child] of Object.entries(value)) {
		if ((key === 'stats' || key === 'pseudoStats') && Array.isArray(child) && child.every(entry => typeof entry === 'number')) {
			while (child.at(-1) === 0) child.pop();
		} else trimStatVectors(child);
	}
}

export function resolveBuildId(catalog, spec, name) {
	assert.equal(catalog.schemaVersion, 1, 'Unsupported catalog schema');
	// Mage group headings are presentation; other groups distinguish builds with the same name.
	const alias = spec === 'mage' ? name.split(' / ').at(-1) : name;
	const matches = catalog.builds.filter(build => build.spec === spec && build.aliases.includes(alias));
	assert.equal(matches.length, 1, `Expected one catalog entry for ${spec} / ${name}`);
	return matches[0].id;
}

export function validateCatalog(catalog) {
	assert.equal(catalog.schemaVersion, 1, 'Unsupported catalog schema');
	assert.ok(Array.isArray(catalog.builds) && catalog.builds.length > 0, 'Missing catalog builds');
	const ids = new Set();
	const aliases = new Set();
	for (const build of catalog.builds) {
		assert.match(build.id, /^[a-z0-9]+(?:-[a-z0-9]+)*$/, 'Invalid stable build ID');
		assert.ok(!ids.has(build.id), `Duplicate catalog ID ${build.id}`);
		ids.add(build.id);
		assert.ok(build.spec && build.aliases?.length, `Missing aliases for ${build.id}`);
		for (const alias of build.aliases) {
			const key = `${build.spec}|${alias}`;
			assert.ok(!aliases.has(key), `Duplicate catalog alias ${key}`);
			aliases.add(key);
		}
	}
}
