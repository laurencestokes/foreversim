import fs from 'node:fs';
import path from 'node:path';

import { applyBuild } from '@features/settings/model/apply_build';
import { applyIndividualDefaults } from '@features/settings/model/apply_defaults';
import { Player as PlayerProto } from '@generated/proto/api';
import { Class, EquipmentSpec, Spec } from '@generated/proto/common';
import { UIDatabase } from '@generated/proto/ui';
import { LaunchStatus } from '@sim/constants/other';
import { PlayerSpecs } from '@sim/player/specs';
import type { PresetBuild } from '@sim/presets/types';
import type { SpecDefinition } from '@sim/spec_config';
import balance from '@specs/druid/balance/spec';
import feral from '@specs/druid/feralcat/spec';
import hunter from '@specs/hunter/dps/spec';
import mage from '@specs/mage/dps/spec';
import paladin from '@specs/paladin/retribution/spec';
import priest from '@specs/priest/dps/spec';
import rogue from '@specs/rogue/dps/spec';
import elemental from '@specs/shaman/elemental/spec';
import enhancement from '@specs/shaman/enhancement/spec';
import warlock from '@specs/warlock/dps/spec';
import warrior from '@specs/warrior/dps/spec';
import { makeIndividualSim, serveDatabase } from '@testing/sim/individual_sim';
import { expect, it, vi } from 'vitest';

import warlockGear from './warlock/dps/gear_sets/preraid.gear.json';

vi.mock('@sim/workers/worker_pool', async () => (await import('@testing/sim/worker_pool')).idleWorkerPool());

function requestsRegionalRacial(value: unknown): boolean {
	if (!value || typeof value !== 'object') return false;
	if ('spellId' in value && 'tag' in value && [1259705, 1259686].includes(Number(value.spellId)) && value.tag === 1) return true;
	return Object.values(value).some(requestsRegionalRacial);
}

// Export the same resolved players the simulator sends to Go, including simple rotations and build options.
// RACE_ARENA_INPUT turns this check into the input step for tools/race_arena/simulate.
it('resolves every DPS preset for the race arena with its gear and rotation', async () => {
	serveDatabase(UIDatabase.fromJson(JSON.parse(fs.readFileSync('assets/database/db.json', 'utf8')), { ignoreUnknownFields: true }));
	const definitions: Array<[string, SpecDefinition<any>]> = [
		['balance_druid', balance],
		['feral_druid', feral],
		['hunter', hunter],
		['mage', mage],
		['retribution_paladin', paladin],
		['priest', priest],
		['rogue', rogue],
		['elemental_shaman', elemental],
		['enhancement_shaman', enhancement],
		['warlock', warlock],
		['warrior', warrior],
	];
	const inputs = [];
	const dpsSpecs = Object.values(Spec)
		.filter((value): value is Spec => typeof value === 'number' && value !== Spec.SpecUnknown)
		.map(spec => PlayerSpecs.fromProto(spec))
		.filter(spec => !spec.isTankSpec && !spec.isHealingSpec);
	expect(definitions.map(([, config]) => config.spec).sort()).toEqual(dpsSpecs.map(spec => spec.specID).sort());
	for (const [key, config] of definitions) {
		const builds: PresetBuild[] = [...(config.presets.builds ?? [])];
		for (const talents of config.presets.talents) {
			if (!builds.some(build => build.talents?.data.talentsString === talents.data.talentsString)) {
				builds.push({ name: talents.name, talents });
			}
		}
		expect(builds.length, key).toBeGreaterThan(0);
		for (const build of builds) {
			const { player, host } = await makeIndividualSim({ spec: config.spec as Spec, config });
			applyIndividualDefaults(
				Object.assign(host, {
					statWeightActionSettings: { applyDefaults() {} },
					statSweepSettings: { applyDefaults() {} },
				}) as unknown as Parameters<typeof applyIndividualDefaults>[0],
			);
			applyBuild(build, host);
			// Warlock presets contain no gear; its regression suites use this upstream preraid set.
			if (key === 'warlock') player.setGear(host.sim.db.lookupEquipmentSpec(EquipmentSpec.fromJson(warlockGear)));
			const exported = player.toProto(false, true);
			const label = [build.group, build.name].filter(Boolean).join(' / ');
			const gear = build.gear ?? config.presets.gear.find(gear => gear.gear === config.defaults.gear);
			const devGear = Boolean(gear?.devOnly);
			const consumableIds = new Set(
				Object.values(exported.consumables ?? {})
					.flat()
					.filter((value): value is number => typeof value === 'number' && value > 0),
			);
			const consumableNames = host.sim.db
				.getConsumables()
				.filter(item => consumableIds.has(item.id))
				.map(({ id, name }) => ({ id, name }))
				.sort((a, b) => a.id - b.id);
			expect(consumableNames.length, `${key} / ${label} consumable names`).toBe(consumableIds.size);
			const caveats = [
				...(devGear ? ['Uses an upstream gear preset marked for development.'] : []),
				...(key === 'warlock' ? ['The build preset supplies no gear; uses the upstream preraid regression set.'] : []),
				...(key === 'hunter' && build.name.includes('Weave')
					? ['Weaving assumes uninterrupted access to melee range and the movement timing programmed in the preset.']
					: []),
				...(key === 'hunter' && build.name === 'SV Melee'
					? ['The preset moves into melee range and stays there; the result assumes continuous melee access.']
					: []),
				...(key === 'feral_druid' && build.name === 'Bearweave (45s fights)'
					? ['The preset is named for 45-second fights; this arena also measures it at 90, 180 and 360 seconds.']
					: []),
			];
			expect(requestsRegionalRacial(PlayerProto.toJson(exported)), `${key} / ${label}: Attuned and Charged are removed on entering a raid`).toBe(false);
			expect(exported.equipment?.items.filter(item => item.id).length, `${key} / ${label} gear`).toBeGreaterThan(10);
			expect(exported.rotation?.priorityList.length, `${key} / ${label} rotation`).toBeGreaterThan(0);
			expect(exported.talentsString, `${key} / ${label} talents`).not.toBe('');
			inputs.push({
				spec: key === 'priest' ? (label.toLowerCase().includes('smite') ? 'smite_priest' : 'shadow_priest') : key,
				class: Class[exported.class].replace('Class', ''),
				build: label,
				status: LaunchStatus[player.getPlayerSpec().launch.status],
				devGear,
				caveats,
				consumableNames,
				gear:
					key === 'warlock'
						? 'preraid'
						: (build.gear?.name ?? config.presets.gear.find(gear => gear.gear === config.defaults.gear)?.name ?? 'Default'),
				rotation: build.rotation?.name ?? 'Default',
				consumables: 'Preset consumables',
				player: PlayerProto.toJson(exported),
			});
		}
	}
	const destination = process.env.RACE_ARENA_INPUT;
	if (destination) {
		fs.mkdirSync(path.dirname(destination), { recursive: true });
		fs.writeFileSync(destination, JSON.stringify(inputs, null, 2) + '\n');
	}
}, 60000);
