import { WeaponType } from '@generated/proto/common';
import i18n from '@i18n/config';
import { translateWeaponType } from '@i18n/localization';
import { Tooltip, tooltipAnchorProps } from '@ui-kit/Tooltip';
import { useId } from 'react';

export interface WeaponTypeOverrideLabelProps {
	weaponTypeOverride: WeaponType;
}

/**
 * Small marker shown on an equipped weapon whose type has been debug-relabelled (see
 * EquippedItem.withWeaponTypeOverride), so gear lists and results never silently look like the
 * character is holding a real axe/sword/etc when the stats say otherwise.
 */
export const WeaponTypeOverrideLabel = ({ weaponTypeOverride }: WeaponTypeOverrideLabelProps) => {
	const tooltipId = useId();

	// Falsy rather than an exact WeaponTypeUnknown check: some test doubles for EquippedItem leave
	// this field undefined rather than defaulting it, and undefined means the same thing -- no override.
	if (!weaponTypeOverride) return null;

	const typeName = translateWeaponType(weaponTypeOverride);

	return (
		<>
			<small className="ml-1 text-ui text-link-warning" data-testid="weapon-type-override-badge" {...tooltipAnchorProps(tooltipId)}>
				{i18n.t('gear_tab.gear_picker.weapon_type_override.badge', { type: typeName })}
			</small>
			<Tooltip id={tooltipId} content={i18n.t('gear_tab.gear_picker.weapon_type_override.badge_tooltip', { type: typeName })} />
		</>
	);
};
