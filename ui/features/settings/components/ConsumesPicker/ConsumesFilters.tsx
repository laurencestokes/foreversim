import { Tabs } from '@base-ui/react/tabs';
import { FilterPopover, StatFilterList, useOpenFilterGroups, useStatFilter } from '@features/stat-filters';
import { ConsumableType } from '@generated/proto/common';
import i18n from '@i18n/config';
import { useSimHost } from '@sim/context/SimHostContext';
import { useStoreSubscribe } from '@sim/hooks/useStoreSubscribe';
import { Database } from '@sim/proto/database';
import { subscribeSimField } from '@sim/state/subscriptions';
import { Button } from '@ui-kit/Button';
import { TabNav } from '@ui-kit/TabNav';
import { NumberField } from '@ui-kit/TextField';
import { useMemo } from 'react';

import { consumableFilterScope } from '../../model/consumables';

const TYPE_KEYS: ReadonlyArray<[ConsumableType, string]> = [
	[ConsumableType.ConsumableTypePotion, 'potion'],
	[ConsumableType.ConsumableTypeConjured, 'conjured'],
	[ConsumableType.ConsumableTypeFlask, 'flask'],
	[ConsumableType.ConsumableTypeElixir, 'elixir'],
	[ConsumableType.ConsumableTypeScroll, 'scroll'],
	[ConsumableType.ConsumableTypeFood, 'food'],
	[ConsumableType.ConsumableTypeExplosive, 'explosive'],
	[ConsumableType.ConsumableTypeImbue, 'imbue'],
	[ConsumableType.ConsumableTypeWeaponImbue, 'weaponImbue'],
	[ConsumableType.ConsumableTypeDevice, 'device'],
	[ConsumableType.ConsumableTypeOther, 'other'],
];

const STATS_TAB = 'consumes-filters-stats-pane';
const MIN_LVL_TAB = 'consumes-filters-min-lvl-pane';

export const ConsumesFilters = () => {
	const { sim, player } = useSimHost();
	const filters = useStoreSubscribe(subscribeSimField(sim, 'filters'), () => sim.getFilters());
	const scope = useMemo(() => consumableFilterScope(player, Database.getSync()), [player]);
	const { groupsStorageKey, ...statFilter } = useStatFilter('consumables', scope);
	const { onPopoverOpenChange, ...groups } = useOpenFilterGroups(groupsStorageKey, statFilter.universe, statFilter.shown);
	const types = useMemo(() => {
		const ranges = new Map<ConsumableType, [number, number]>();
		for (const { type, lvl } of Database.getSync().getConsumables()) {
			const range = ranges.get(type);
			ranges.set(type, range ? [Math.min(range[0], lvl), Math.max(range[1], lvl)] : [lvl, lvl]);
		}
		return TYPE_KEYS.flatMap(([type, key]) => {
			const range = ranges.get(type);
			return range ? [{ type, key, range: `${range[0]}-${range[1]}` }] : [];
		});
	}, []);
	const setMinLvls = (minConsumableLvl: { [type: number]: number }) => sim.setFilters({ ...sim.getFilters(), minConsumableLvl });
	const minLvlsSet = Object.keys(filters.minConsumableLvl).length > 0;

	return (
		<FilterPopover
			ariaLabel={i18n.t('settings_tab.consumables.filters.trigger')}
			title={i18n.t('settings_tab.consumables.filters.title')}
			triggerClassName="ml-auto"
			triggerTestId="consumes-filters-button"
			testId="consumes-filters-popover"
			customised={statFilter.customised || minLvlsSet}
			onOpenChange={onPopoverOpenChange}>
			<Tabs.Root defaultValue={STATS_TAB} className="flex min-h-0 flex-col">
				<TabNav
					wrap={false}
					className="mb-2 shrink-0"
					tabs={[
						{ id: STATS_TAB, label: i18n.t('stat_filters.title') },
						{ id: MIN_LVL_TAB, label: i18n.t('settings_tab.consumables.filters.min_lvl') },
					]}
				/>
				<Tabs.Panel value={STATS_TAB} id={STATS_TAB} className="flex min-h-0 flex-col">
					<StatFilterList {...statFilter} {...groups} idPrefix="consumes-filters-stats" />
				</Tabs.Panel>
				<Tabs.Panel value={MIN_LVL_TAB} id={MIN_LVL_TAB} className="flex min-h-0 flex-col gap-2">
					<div className="ui-popover-body grid grid-cols-2 gap-2">
						{types.map(({ type, key, range }) => (
							<NumberField
								key={type}
								id={`consumes-filters-min-lvl-${key}`}
								label={i18n.t(`settings_tab.consumables.filters.types.${key}`)}
								placeholder={range}
								positive
								showZeroes={false}
								value={sim.getMinConsumableLvl(type)}
								onChange={value => setMinLvls({ ...filters.minConsumableLvl, [type]: value })}
							/>
						))}
					</div>
					<div className="flex shrink-0 justify-end">
						<Button variant="outline-primary" size="sm" data-testid="consumes-filters-reset" disabled={!minLvlsSet} onClick={() => setMinLvls({})}>
							{i18n.t('settings_tab.consumables.filters.reset')}
						</Button>
					</div>
				</Tabs.Panel>
			</Tabs.Root>
		</FilterPopover>
	);
};
