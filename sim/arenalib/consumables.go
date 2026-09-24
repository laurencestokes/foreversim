package arenalib

import (
	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// What every build in the arena drinks and is buffed by, decided here rather than by each
// spec's own test file.
//
// The file header claims every build meets the same fixed environment, and until this existed
// that was false. Fifteen specs brought four different consumable sets from four different
// content phases, and the gaps were not small ones: both paladins and the feral tank had their
// weapon imbue commented out entirely, while warrior, hunter, rogue, tank warrior and
// enhancement all carried Windfury. Measured on the warrior, stripping its imbues cost 14.2%
// of its damage - so the leaderboard was reporting a 19.6% gap between warrior and retribution
// while handing one of them a weapon buff and the other a bare weapon.
//
// Windfury deserves a note, because it is the biggest single item here and it is not really a
// consumable at all. The old engine had nowhere to put a totem and applied it as a weapon
// imbue; this one has PartyBuffs.WindfuryTotem, so the melee and ranged lists carry it there.
// Giving it to every melee matches what the arena does everywhere else - every spec gets the
// same blessings whatever its faction. The arena is a fixed environment, not a raid roster, and
// it has never modelled faction.
//
// Three sets rather than one, because one list for all fifteen would be a different lie.
// Elemental Sharpening Stone is +2% melee crit and -2% RANGED crit, so handing the hunter the
// melee list would equalise the shopping and quietly tax the one spec that shoots. Within a
// role every spec gets the identical list; what it is worth to you is your class's business,
// which is why Mighty Rage Potion stays in the melee list even though only warriors can spend
// it. Same shopping list, not same benefit - that is the part that makes two numbers comparable.
//
// The consumables are the Classic items the Forever client ships, in the Classic slots #421
// added to ConsumesSpec. The buffs are the old engine's ForeverBuffs, field for field where the
// new protos have the field: the resistance auras and totems, Blessing of Sanctuary, Curse of
// Shadow, Curse of Weakness and Stormstrike have no slot here, and none of them moves the damage
// of a build hitting a boss that swings physical.
type Role int

const (
	Melee Role = iota
	Ranged
	Caster
)

var arenaRaidBuffs = &proto.RaidBuffs{
	ArcaneBrilliance:   true,
	PowerWordFortitude: proto.TristateEffect_TristateEffectImproved,
	ShadowProtection:   true,
	DivineSpirit:       proto.TristateEffect_TristateEffectRegular,
	GiftOfTheWild:      proto.TristateEffect_TristateEffectImproved,
	Thorns:             proto.TristateEffect_TristateEffectImproved,
}

// Every role's party buffs. The melee and ranged lists add Windfury Totem on top.
var arenaPartyBuffs = &proto.PartyBuffs{
	BattleShout:          proto.TristateEffect_TristateEffectImproved,
	BloodPact:            proto.TristateEffect_TristateEffectImproved,
	DevotionAura:         true,
	RetributionAura:      true,
	GraceOfAirTotem:      proto.TristateEffect_TristateEffectImproved,
	LeaderOfThePack:      proto.TristateEffect_TristateEffectRegular,
	ManaSpringTotem:      proto.TristateEffect_TristateEffectImproved,
	MoonkinAura:          proto.TristateEffect_TristateEffectRegular,
	StrengthOfEarthTotem: proto.TristateEffect_TristateEffectImproved,
	TrueshotAura:         true,
}

// World buffs do not work inside Forever raids, so the player buffs are the blessings alone.
var arenaPlayerBuffs = &proto.IndividualBuffs{
	BlessingOfKings:  true,
	BlessingOfMight:  true,
	BlessingOfWisdom: true,
}

// Improved Shadow Bolt, Shadow Weaving, Improved Scorch and Winter's Chill became personal
// buffs on their caster in Forever, so nobody applies them to the raid.
var arenaDebuffs = &proto.Debuffs{
	CurseOfElements:        proto.TristateEffect_TristateEffectRegular,
	CurseOfRecklessness:    true,
	DemoralizingRoar:       proto.TristateEffect_TristateEffectImproved,
	DemoralizingShout:      proto.TristateEffect_TristateEffectImproved,
	ExposeArmor:            proto.TristateEffect_TristateEffectImproved,
	FaerieFire:             proto.TristateEffect_TristateEffectRegular,
	InsectSwarm:            true,
	JudgementOfLight:       true,
	JudgementOfWisdom:      true,
	JudgementOfTheCrusader: true,
	ScorpidSting:           true,
	SunderArmor:            true,
	ThunderClap:            proto.TristateEffect_TristateEffectImproved,
}

func withWindfury(party *proto.PartyBuffs) *proto.PartyBuffs {
	party = googleProto.Clone(party).(*proto.PartyBuffs)
	party.WindfuryTotem = proto.TristateEffect_TristateEffectRegular
	return party
}

var consumesMelee = core.BuffsCombo{
	Label:   "Arena-Melee",
	Raid:    arenaRaidBuffs,
	Party:   withWindfury(arenaPartyBuffs),
	Player:  arenaPlayerBuffs,
	Debuffs: arenaDebuffs,
	Consumables: &proto.ConsumesSpec{
		BattleElixirId:    13452, // Elixir of the Mongoose
		AttackPowerBuffId: 12460, // Juju Might
		StrengthBuffId:    12451, // Juju Power
		PotId:             13442, // Mighty Rage Potion
		DragonbreathChili: true,
		FlaskId:           13510, // Flask of the Titans
		FoodId:            20452, // Smoked Desert Dumplings
		OhImbueId:         18262, // Elemental Sharpening Stone
	},
}

// The melee list without the off-hand stone, whose ranged crit penalty is a real cost to the
// one spec that does its damage from thirty yards, and without the rage potion.
var consumesRanged = core.BuffsCombo{
	Label:   "Arena-Ranged",
	Raid:    arenaRaidBuffs,
	Party:   withWindfury(arenaPartyBuffs),
	Player:  arenaPlayerBuffs,
	Debuffs: arenaDebuffs,
	Consumables: &proto.ConsumesSpec{
		BattleElixirId:    13452, // Elixir of the Mongoose
		AttackPowerBuffId: 12460, // Juju Might
		StrengthBuffId:    12451, // Juju Power
		DragonbreathChili: true,
		FlaskId:           13510, // Flask of the Titans
		FoodId:            20452, // Smoked Desert Dumplings
	},
}

var consumesCaster = core.BuffsCombo{
	Label:   "Arena-Caster",
	Raid:    arenaRaidBuffs,
	Party:   arenaPartyBuffs,
	Player:  arenaPlayerBuffs,
	Debuffs: arenaDebuffs,
	Consumables: &proto.ConsumesSpec{
		PotId:              13444, // Major Mana Potion
		FlaskId:            13512, // Flask of Supreme Power
		FoodId:             20452, // Smoked Desert Dumplings
		MhImbueId:          20749, // Brilliant Wizard Oil
		SpellPowerElixirId: 13454, // Greater Arcane Elixir
		SchoolElixirId:     6373,  // Elixir of Firepower
	},
}

// A weapon imbue a class grants itself, which the role list must not overwrite.
//
// The role lists equalise what a character BUYS. Some imbues are not bought: Windfury Weapon
// is an enhancement shaman casting on their own weapons, and Instant Poison is a rogue's
// poison. Equalising those does not make a comparison fairer, it takes a class ability away -
// the first pass did exactly that and cost enhancement 23.6% of its damage, which is not a
// shaman being measured honestly, it is a shaman being disarmed.
//
// Poisons are items here (ConsumesSpec's imbue ids). Windfury Weapon is a spec option on this
// engine, so a class that casts its own sets Windfury, which takes the totem away exactly as
// the old list's imbue override did.
type ClassImbues struct {
	MainHand int32
	OffHand  int32
	Windfury bool
}

func consumesFor(role Role, imbues ClassImbues) core.BuffsCombo {
	var combo core.BuffsCombo
	switch role {
	case Ranged:
		combo = consumesRanged
	case Caster:
		combo = consumesCaster
	default:
		combo = consumesMelee
	}
	if imbues == (ClassImbues{}) {
		return combo
	}

	// Cloned, not mutated: the role lists are package level and shared by every spec, so
	// writing a shaman's Windfury into one would hand it to the next spec that ran. Cloned
	// rather than copied because a proto carries a mutex.
	combo.Consumables = googleProto.Clone(combo.Consumables).(*proto.ConsumesSpec)
	if imbues.MainHand != 0 {
		combo.Consumables.MhImbueId = imbues.MainHand
	}
	if imbues.OffHand != 0 {
		combo.Consumables.OhImbueId = imbues.OffHand
	}
	if imbues.Windfury {
		combo.Party = googleProto.Clone(combo.Party).(*proto.PartyBuffs)
		combo.Party.WindfuryTotem = proto.TristateEffect_TristateEffectMissing
	}
	combo.Label += "+class"
	return combo
}
