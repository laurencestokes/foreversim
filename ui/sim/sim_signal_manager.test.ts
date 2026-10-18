import { describe, expect, it, vi } from 'vitest';

import { RequestTypes, SimSignalManager } from './sim_signal_manager';

describe('SimSignalManager.abortType', () => {
	it('trips only the requests whose type is in the mask', async () => {
		const manager = new SimSignalManager();
		const individual = manager.registerRunning(RequestTypes.IndividualSim);
		const statWeights = manager.registerRunning(RequestTypes.StatWeights);
		const stopped: string[] = [];
		individual.abort.onTrigger(async () => void stopped.push('individual'));
		statWeights.abort.onTrigger(async () => void stopped.push('stat-weights'));

		await manager.abortType(RequestTypes.IndividualSim);

		expect(stopped).toEqual(['individual']);
	});

	it('trips everything registered when the mask is All', async () => {
		const manager = new SimSignalManager();
		const stopped: string[] = [];
		for (const [name, type] of [
			['individual', RequestTypes.IndividualSim],
			['stat-weights', RequestTypes.StatWeights],
			['reforge', RequestTypes.ReforgeOptimize],
			['bulk', RequestTypes.BulkSim],
		] as const) {
			manager.registerRunning(type).abort.onTrigger(async () => void stopped.push(name));
		}

		await manager.abortType(RequestTypes.All);

		expect(stopped.sort()).toEqual(['bulk', 'individual', 'reforge', 'stat-weights']);
	});

	// Each run kind has its own type, so cancelling a batch leaves a stat-weights or optimizer run alone.
	it('tells a batch apart from a single sim, a stat-weights run and a gear optimizer run', async () => {
		const manager = new SimSignalManager();
		const stopped: string[] = [];
		for (const [name, type] of [
			['individual', RequestTypes.IndividualSim],
			['bulk', RequestTypes.BulkSim],
			['stat-weights', RequestTypes.StatWeights],
			['reforge', RequestTypes.ReforgeOptimize],
		] as const) {
			manager.registerRunning(type).abort.onTrigger(async () => void stopped.push(name));
		}

		await manager.abortType(RequestTypes.BulkSim);

		expect(stopped).toEqual(['bulk']);
	});

	it('does not trip a request that already finished', async () => {
		const manager = new SimSignalManager();
		const signals = manager.registerRunning(RequestTypes.IndividualSim);
		const onAbort = vi.fn(async () => {});
		signals.abort.onTrigger(onAbort);
		manager.unregisterRunning(signals);

		await manager.abortType(RequestTypes.All);

		expect(onAbort).not.toHaveBeenCalled();
	});
});
