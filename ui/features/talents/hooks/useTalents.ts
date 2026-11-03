import { SavedTalents } from '@generated/proto/ui';
import { usePlayer } from '@sim/context/SimHostContext';
import { usePlayerStore } from '@sim/hooks/usePlayerStore';
import { currentTalentLayout } from '@sim/talents/factory';
import { useMemo } from 'react';

export const useTalents = (): SavedTalents => {
	const talentsString = usePlayerStore('talentsString');
	const player = usePlayer();
	return useMemo(() => SavedTalents.create({ talentsString, talentLayout: currentTalentLayout(player.getClass()) }), [player, talentsString]);
};
