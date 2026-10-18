import type { BulkEnchantPreset } from '@generated/proto/ui';
import { usePlayer } from '@sim/context/SimHostContext';
import { useStore } from 'zustand';

/** The saved enchant presets, which the batch sim and the gear optimizer share. */
export const useEnchantPresets = (): readonly BulkEnchantPreset[] => {
	const player = usePlayer();
	return useStore(player.sim.store, state => state.bulk[player.storeKey].enchantPresets);
};
