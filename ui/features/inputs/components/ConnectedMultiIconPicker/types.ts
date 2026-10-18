import type { Player } from '@sim/player/player';
import type { ActionId } from '@sim/proto/action_id';

import type { KeptInput } from '../../input';
import type { IconPickerConfig } from '../ConnectedIconPicker/types';

export type MultiIconPickerItemConfig<ModObject> = IconPickerConfig<ModObject, any>;

export interface MultiIconPickerConfig<ModObject> extends KeptInput<ModObject, boolean> {
	inputs: Array<MultiIconPickerItemConfig<ModObject>>;
	label?: string;
	categoryId?: ActionId;
	showWhen?: (obj: Player<any>) => boolean;
}
