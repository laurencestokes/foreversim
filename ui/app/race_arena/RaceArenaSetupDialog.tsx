import { Tabs } from '@base-ui/react/tabs';
import { ActionId } from '@sim/proto/action_id';
import { Button } from '@ui-kit/Button';
import { CopyButton } from '@ui-kit/CopyButton';
import { Dialog } from '@ui-kit/Dialog';
import { TabNav, TabPanel, TabPanels } from '@ui-kit/TabNav';

type EquipmentSlot = { slot: string; item?: { id: number; name: string; suffixId?: number; enchant?: string } };
export type RaceArenaSetup = {
	equipment: Array<EquipmentSlot>;
	itemSwap?: Array<EquipmentSlot>;
	talentsString: string;
	talents: Array<{ name: string; points: number; talents: Array<{ id: number; definitionId?: number; name: string; rank: number; maxRank: number }> }>;
};

const Equipment = ({ slots }: { slots: Array<EquipmentSlot> }) => (
	<dl className="m-0 grid gap-x-6 gap-y-2 md:grid-cols-2">
		{slots.map(({ slot, item }) => (
			<div key={slot} className="min-w-0 border-b border-surface-border pb-2">
				<dt className="text-xs text-white/60">{slot}</dt>
				<dd className="m-0 flex flex-col items-start">
					{item ? (
						<>
							<Button variant="link" className="text-left" href={ActionId.makeItemUrl(item.id, item.suffixId)} target="_blank">
								{item.name}
							</Button>
							{item.enchant && <span className="text-sm text-success">{item.enchant}</span>}
						</>
					) : (
						<span className="text-sm text-white/50">Empty</span>
					)}
				</dd>
			</div>
		))}
	</dl>
);

export const RaceArenaSetupDialog = ({
	open,
	onOpenChange,
	title,
	gear,
	rotation,
	setup,
	commit,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	title: string;
	gear: string;
	rotation: string;
	setup: RaceArenaSetup;
	commit: string;
}) => (
	<Dialog open={open} onOpenChange={onOpenChange} title={title} size="lg" scrollContents>
		<p className="m-0 text-sm text-white/70">
			{gear} gear · {rotation} rotation. This saved setup was used for every race and target type in this build. Item and talent links open their current
			game descriptions in a new tab.
		</p>
		<Tabs.Root defaultValue="gear">
			<TabNav
				tabs={[
					{ id: 'gear', label: 'Gear & enchants' },
					{ id: 'talents', label: `Talents (${setup.talents.map(tree => tree.points).join('/')})` },
				]}
			/>
			<TabPanels>
				<TabPanel value="gear">
					<div className="flex flex-col gap-4 pt-4">
						<Equipment slots={setup.equipment} />
						{!!setup.itemSwap?.length && (
							<section className="flex flex-col gap-2">
								<h3 className="m-0 text-lg">Weapon swaps</h3>
								<p className="m-0 text-sm text-white/70">The rotation swaps these items in during the fight. Every race uses the same swaps.</p>
								<Equipment slots={setup.itemSwap} />
							</section>
						)}
					</div>
				</TabPanel>
				<TabPanel value="talents">
					<div className="flex flex-col gap-4 pt-4">
						<div className="grid gap-4 md:grid-cols-3">
							{setup.talents.map(tree => (
								<section key={tree.name} className="min-w-0">
									<h3 className="m-0 border-b border-surface-border pb-2 text-base">
										{tree.name} · {tree.points} points
									</h3>
									{tree.talents.length ? (
										<ul className="m-0 flex list-none flex-col gap-2 p-0 pt-2">
											{tree.talents.map(talent => (
												<li key={talent.id} className="flex items-baseline justify-between gap-2 text-sm">
													<Button
														variant="link"
														className="text-left"
														href={ActionId.makeSpellUrl(talent.id, talent.rank, talent.definitionId)}
														target="_blank">
														{talent.name}
													</Button>
													<span className="shrink-0 text-white/70 tabular-nums">
														{talent.rank}/{talent.maxRank}
													</span>
												</li>
											))}
										</ul>
									) : (
										<p className="mb-0 text-sm text-white/50">No points spent.</p>
									)}
								</section>
							))}
						</div>
						<div className="flex flex-wrap items-center gap-2 border-t border-surface-border pt-4">
							<code className="min-w-0 text-xs break-all select-all">{setup.talentsString}</code>
							<CopyButton getContent={() => setup.talentsString} text="Copy talent string" variant="secondary" size="sm" />
						</div>
					</div>
				</TabPanel>
			</TabPanels>
		</Tabs.Root>
		<p className="m-0 text-xs text-white/50">
			Setup recorded with engine{' '}
			<Button variant="link" href={`https://github.com/laurencestokes/foreversim/commit/${commit}`} target="_blank">
				<code>{commit.slice(0, 7)}</code>
			</Button>
			.
		</p>
	</Dialog>
);
