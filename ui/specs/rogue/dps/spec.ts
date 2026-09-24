import * as OtherInputs from '@features/settings/model/other_inputs';
import { StatCapType } from '@generated/proto/api';
import { APLRotation } from '@generated/proto/apl';
import { Debuffs, IndividualBuffs, ItemSlot, PartyBuffs, PseudoStat, RaidBuffs, Spec, Stat, TristateEffect, WeaponType } from '@generated/proto/common';
import * as Mechanics from '@sim/constants/mechanics';
import { PlayerClasses } from '@sim/player/classes';
import { Player } from '@sim/player/player';
import { masterEpWeights } from '@sim/proto/master_ep_weights';
import { StatCap, Stats, UnitStat } from '@sim/proto/stats';
import { defineSpec } from '@sim/spec_config';

import * as Presets from './presets';

export default defineSpec<Spec.SpecRogue>({
	spec: Spec.SpecRogue,

	className: 'rogue-sim-ui',
	cssScheme: PlayerClasses.getCssScheme(PlayerClasses.Rogue),
	// List any known bugs / issues here and they'll be shown on the site.
	knownIssues: [
		'The APL is in constant flux due to bug fixes and new findings; if your DPS drops dramatically, reset it back to "Auto" in the Rotation tab!',
		'Mutilate does not have a default APL currently. It will not be automatically used when talented.',
	],

	// All stats for which EP should be calculated.
	epStats: [
		Stat.StatStamina,
		Stat.StatAgility,
		Stat.StatStrength,
		Stat.StatMeleeCritRating,
		Stat.StatMeleeHasteRating,
		Stat.StatMeleeHitRating,
		Stat.StatArmorPenetration,
		Stat.StatExpertiseRating,
		Stat.StatAttackPower,
		Stat.StatPhysicalDamage,
	],
	epPseudoStats: [PseudoStat.PseudoStatMainHandDps, PseudoStat.PseudoStatOffHandDps],
	// Reference stat against which to calculate EP.
	epReferenceStat: Stat.StatAttackPower,
	// Which stats to display in the Character Stats section, at the bottom of the left-hand sidebar.
	displayStats: UnitStat.createDisplayStatArray(
		[
			Stat.StatHealth,
			Stat.StatStamina,
			Stat.StatAgility,
			Stat.StatStrength,
			Stat.StatAttackPower,
			Stat.StatArmorPenetration,
			Stat.StatExpertiseRating,
			Stat.StatArcaneResistance,
			Stat.StatFireResistance,
			Stat.StatFrostResistance,
			Stat.StatNatureResistance,
			Stat.StatShadowResistance,
		],
		[PseudoStat.PseudoStatMeleeHitPercent, PseudoStat.PseudoStatMeleeCritPercent, PseudoStat.PseudoStatMeleeHastePercent],
	),

	defaults: {
		// Default equipped gear.
		gear: Presets.DEFAULT_GEAR.gear,
		// Default EP weights for sorting gear in the gear picker.
		// Master's weights (percent stats per 1%), in our ratings.
		epWeights: masterEpWeights({
			Agility: 2.38,
			Strength: 1.26,
			AttackPower: 1.0,
			SpellCrit: 0.41,
			SpellHit: 0.94,
			MeleeHit: 29.44,
			MeleeCrit: 17.92,
			FireResistance: 0.5,
			MainHandDps: 10.49,
			OffHandDps: 3.74,
			MeleeSpeedMultiplier: 18.56,
		}),
		statCaps: (() => {
			const expCap = new Stats().withStat(Stat.StatExpertiseRating, 6.5 * 4 * Mechanics.EXPERTISE_PER_QUARTER_PERCENT_REDUCTION);
			return expCap;
		})(),
		softCapBreakpoints: (() => {
			const meleeHitSoftCapConfig = StatCap.fromPseudoStat(PseudoStat.PseudoStatMeleeHitPercent, {
				breakpoints: [9, 28],
				capType: StatCapType.TypeSoftCap,
				postCapEPs: [3.06 * Mechanics.PHYSICAL_HIT_RATING_PER_HIT_PERCENT, 0],
			});

			return [meleeHitSoftCapConfig];
		})(),
		other: Presets.OtherDefaults,
		// Default consumes settings.
		consumables: Presets.DefaultConsumables,
		// Default talents.
		talents: Presets.DefaultTalents.data,
		// Default spec-specific settings.
		specOptions: Presets.DefaultOptions,
		// Default raid/party buffs settings.
		// Master's page (currentSettings on a fresh profile); its raid-wide Battle Shout, Trueshot,
		// Leader of the Pack and Fire Resistance Aura are party buffs here.
		raidBuffs: RaidBuffs.create({
			giftOfTheWild: TristateEffect.TristateEffectImproved,
		}),
		partyBuffs: PartyBuffs.create({
			battleShout: TristateEffect.TristateEffectImproved,
			trueshotAura: true,
			leaderOfThePack: TristateEffect.TristateEffectRegular,
			fireResistanceAura: true,
		}),
		individualBuffs: IndividualBuffs.create({
			blessingOfKings: true,
			blessingOfMight: true,
		}),
		debuffs: Debuffs.create({
			faerieFire: TristateEffect.TristateEffectRegular,
			sunderArmor: true,
			curseOfRecklessness: true,
		}),
	},

	playerInputs: {
		inputs: [],
	},
	// IconInputs to include in the 'Player' section on the settings tab.
	playerIconInputs: [],
	// Buff and Debuff inputs to include/exclude, overriding the EP-based defaults.
	includeBuffDebuffInputs: [Stat.StatSpellHitRating],
	excludeBuffDebuffInputs: [],
	// Inputs to include in the 'Other' section on the settings tab.
	otherInputs: {
		inputs: [OtherInputs.TotemTwisting, OtherInputs.InFrontOfTarget, OtherInputs.InputDelay],
	},
	itemSwapSlots: [ItemSlot.ItemSlotTrinket1, ItemSlot.ItemSlotTrinket2, ItemSlot.ItemSlotMainHand, ItemSlot.ItemSlotOffHand],
	encounterPicker: {
		// Whether to include 'Execute Duration (%)' in the 'Encounter' section of the settings tab.
		showExecuteProportion: false,
	},

	presets: {
		epWeights: [],
		// Preset talents that the user can quickly select.
		talents: Presets.TALENT_PRESETS,
		// Preset rotations that the user can quickly select.
		rotations: Presets.ROTATION_PRESETS,
		// Preset gear configurations that the user can quickly select.
		gear: Presets.GEAR_PRESETS,
		builds: Presets.BUILD_PRESETS,
	},

	// Master's: daggers get Mutilate, Hemorrhage or Backstab by talents, anything else Sinister Strike.
	autoRotation: (player: Player<Spec.SpecRogue>): APLRotation => {
		const talents = player.getTalents();
		if (player.getEquippedItem(ItemSlot.ItemSlotMainHand)?.effectiveWeaponType === WeaponType.WeaponTypeDagger) {
			if (talents.mutilate) return Presets.ROTATION_PRESET_MUTILATE.rotation.rotation!;
			if (talents.hemorrhage) return Presets.ROTATION_PRESET_HEMORRHAGE.rotation.rotation!;
			return Presets.ROTATION_PRESET_BACKSTAB.rotation.rotation!;
		}
		return Presets.ROTATION_PRESET_SINISTER_STRIKE.rotation.rotation!;
	},

	reforge: {},
});
