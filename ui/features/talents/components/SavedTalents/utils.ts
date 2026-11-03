import type { SpecPresets } from '@features/hooks/useSavedPanel';
import { SavedTalents } from '@generated/proto/ui';
import type { IndividualSimHost } from '@sim/sim_host';
import { currentTalentLayout, loadSavedTalentString } from '@sim/talents/factory';
import type { SavedDataPanelEntry } from '@ui-kit/SavedDataPanel';

export const serializeTalents = (data: SavedTalents): string => JSON.stringify(SavedTalents.toJson(data));

export const talentPresets = (presets: SpecPresets, { player }: IndividualSimHost<any>): Array<SavedDataPanelEntry<SavedTalents>> =>
	presets.talents.map(preset => {
		const data = SavedTalents.create({ ...preset.data, talentLayout: currentTalentLayout(player.getClass()) });
		return { name: preset.name, data, json: serializeTalents(data), isPreset: true, afterLoad: () => preset.onLoad?.(player) };
	});

export const loadTalents = ({ data, isPreset }: SavedDataPanelEntry<SavedTalents>, { player }: IndividualSimHost<any>) =>
	player.setTalentsString(isPreset ? data.talentsString : loadSavedTalentString(player.getClass(), data.talentsString, data.talentLayout));
