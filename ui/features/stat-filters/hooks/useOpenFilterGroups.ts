import type { UnitStat } from '@sim/proto/stats';
import { parseStringArray, useLocalStorageState } from '@ui-kit/hooks/useLocalStorageState';
import { useState } from 'react';

import { defaultOpenFilterGroups } from '../model/stat_groups';

/**
 * A stat filter popover's open groups, kept in the browser under `storageKey`. With none stored, each opening of the
 * popover opens the groups holding a shown stat at that moment and keeps them open until it closes, so unticking a
 * group's last stat does not fold the group away.
 */
export const useOpenFilterGroups = (storageKey: string, universe: ReadonlyArray<UnitStat>, shown: ReadonlyArray<UnitStat>) => {
	const [storedGroups, setStoredGroups] = useLocalStorageState<string[] | null>(storageKey, parseStringArray, () => null);
	const [openedGroups, setOpenedGroups] = useState<string[]>([]);
	return {
		openGroups: storedGroups ?? openedGroups,
		onOpenGroupsChange: setStoredGroups,
		onPopoverOpenChange: (open: boolean) => {
			if (open && storedGroups === null) setOpenedGroups(defaultOpenFilterGroups(universe, shown));
		},
	};
};
