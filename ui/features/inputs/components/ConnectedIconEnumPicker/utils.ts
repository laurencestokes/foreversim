import type { IconPickerConfig } from '../ConnectedIconPicker/types';
import type { MultiIconPickerConfig } from '../ConnectedMultiIconPicker/types';
import type { IconEnumPickerConfig, IconEnumValueConfig } from './types';

type PickerConfig<ModObject> = IconEnumPickerConfig<ModObject, any> | IconPickerConfig<ModObject, any> | MultiIconPickerConfig<ModObject>;

export const iconEnumValues = <ModObject, T>(config: IconEnumPickerConfig<ModObject, T>, modObject: ModObject): Array<IconEnumValueConfig<ModObject, T>> =>
	typeof config.values === 'function' ? config.values(modObject) : config.values;

export const iconEnumPickerShown = <ModObject, T>(config: IconEnumPickerConfig<ModObject, T>, modObject: ModObject): boolean =>
	(!config.showWhen || config.showWhen(modObject)) &&
	iconEnumValues(config, modObject).some(valueConfig => !!valueConfig.actionId && (!valueConfig.showWhen || valueConfig.showWhen(modObject)));

/** Whether a value the input shows is a pick: an icon enum's off its zero value, an icon's or a multi icon's when truthy and non-zero. */
export const valuePicked = <ModObject>(config: PickerConfig<ModObject>, value: unknown): boolean =>
	'values' in config ? !config.equals(value, config.zeroValue) : !!Number(value);

/** Whether the input holds a pick, read off the mod object; a multi icon holds one while any of its inputs does. */
export const inputPicked = <ModObject>(config: PickerConfig<ModObject>, modObject: ModObject): boolean =>
	'inputs' in config ? config.inputs.some(input => inputPicked(input, modObject)) : valuePicked(config, config.getValue(modObject));
