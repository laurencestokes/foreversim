import { useResolvedIcon } from '@sim/hooks/useResolvedIcon';
import { useStoreSubscribe } from '@sim/hooks/useStoreSubscribe';
import type { Player } from '@sim/player/player';
import type { ActionId } from '@sim/proto/action_id';
import type { StoreSubscribe } from '@sim/state/subscriptions';
import { MultiIconPicker } from '@ui-kit/MultiIconPicker';

import { keptTooltipOf } from '../../input';
import { valuePicked } from '../ConnectedIconEnumPicker/utils';
import { ConnectedIconPicker } from '../ConnectedIconPicker';
import type { MultiIconPickerConfig } from './types';

export interface ConnectedMultiIconPickerProps<ModObject> {
	modObject: ModObject;
	config: MultiIconPickerConfig<ModObject>;
	subscribe: StoreSubscribe;
	onClear: () => void;
	testId?: string;
}

const firstActiveActionId = <ModObject,>(config: MultiIconPickerConfig<ModObject>, modObject: ModObject): ActionId | null => {
	for (const input of config.inputs) {
		const value = Number(input.getValue(modObject));
		if (!valuePicked(input, value)) continue;
		if (value === 2 && input.improvedId) return input.improvedId;
		if (value === 3 && input.improvedId2) return input.improvedId2;
		return input.actionId;
	}
	return null;
};

export const ConnectedMultiIconPicker = <ModObject,>({ modObject, config, subscribe, onClear, testId }: ConnectedMultiIconPickerProps<ModObject>) => {
	// Subscribed only to re-render on each notification: the reads below take `config` from the render, so a new
	// config is followed without waiting for one.
	useStoreSubscribe(subscribe, () => ({}));
	const actionId = firstActiveActionId(config, modObject);
	const hidden = !!config.showWhen && !config.showWhen(modObject as unknown as Player<any>);

	const icon = useResolvedIcon(config.categoryId ?? actionId ?? undefined);

	if (hidden) return null;

	return (
		<MultiIconPicker
			label={config.label}
			icon={icon}
			active={!!actionId}
			keptTooltip={keptTooltipOf(config, modObject, !!actionId)}
			onClear={onClear}
			testId={testId}>
			{config.inputs.map((input, index) => (
				<ConnectedIconPicker key={index} modObject={modObject} config={input} />
			))}
		</MultiIconPicker>
	);
};
