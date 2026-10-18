import { usePlayer } from '@sim/context/SimHostContext';
import type { Player } from '@sim/player/player';
import { Database } from '@sim/proto/database';
import { useMemo } from 'react';

import { ImbueRows } from '../ConsumesPicker';
import { consumeConfigs } from '../ConsumesPicker/utils';

export const SwapConsumesPicker = () => {
	const player = usePlayer() as Player<any>;
	const configs = useMemo(() => consumeConfigs(player, Database.getSync()), [player]);

	return (
		<ImbueRows
			player={player}
			imbues={[configs.swapMhImbue, configs.swapOhImbue]}
			weaponImbues={[configs.swapMhWeaponImbue, configs.swapOhWeaponImbue]}
			imbueTestId="consumes-swap-imbue"
			weaponImbueTestId="consumes-swap-weapon-imbue"
		/>
	);
};
