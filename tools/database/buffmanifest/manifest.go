// Package buffmanifest is the checked-in census of every raid, party, individual
// and enemy-debuff proto field the sim knows about. It is a leaf package: it may
// import the standard library and sim/core/proto and nothing else, so the buff
// generator and the proto emitter can both read it while the generated proto and
// sim/core files are stale.
package buffmanifest

import (
	"strings"

	"github.com/wowsims/forever/sim/core/proto"
)

type BuffScope int

const (
	ScopeRaid BuffScope = iota
	ScopeParty
	ScopeIndividual
	ScopeDebuff
)

func (s BuffScope) String() string {
	switch s {
	case ScopeRaid:
		return "ScopeRaid"
	case ScopeParty:
		return "ScopeParty"
	case ScopeIndividual:
		return "ScopeIndividual"
	case ScopeDebuff:
		return "ScopeDebuff"
	}
	return "BuffScope(unknown)"
}

type BuffProtoType int

const (
	ProtoBool BuffProtoType = iota
	ProtoTristate
	ProtoInt32
	ProtoDouble
)

func (t BuffProtoType) String() string {
	switch t {
	case ProtoBool:
		return "ProtoBool"
	case ProtoTristate:
		return "ProtoTristate"
	case ProtoInt32:
		return "ProtoInt32"
	case ProtoDouble:
		return "ProtoDouble"
	}
	return "BuffProtoType(unknown)"
}

type BuffKind int

const (
	KindStatFlat BuffKind = iota
	KindStatPct
	KindResistance
	KindPseudoMult
	KindDamageShield
	KindDebuffStat
	KindDebuffStacking
	KindDebuffDamageTaken
	KindDebuffAtkSpeed
	KindDebuffUptime
	KindExternalCD
	KindProc
	KindItemCount
	KindManual
	KindFlag
	KindAbsent
)

func (k BuffKind) String() string {
	switch k {
	case KindStatFlat:
		return "KindStatFlat"
	case KindStatPct:
		return "KindStatPct"
	case KindResistance:
		return "KindResistance"
	case KindPseudoMult:
		return "KindPseudoMult"
	case KindDamageShield:
		return "KindDamageShield"
	case KindDebuffStat:
		return "KindDebuffStat"
	case KindDebuffStacking:
		return "KindDebuffStacking"
	case KindDebuffDamageTaken:
		return "KindDebuffDamageTaken"
	case KindDebuffAtkSpeed:
		return "KindDebuffAtkSpeed"
	case KindDebuffUptime:
		return "KindDebuffUptime"
	case KindExternalCD:
		return "KindExternalCD"
	case KindProc:
		return "KindProc"
	case KindItemCount:
		return "KindItemCount"
	case KindManual:
		return "KindManual"
	case KindFlag:
		return "KindFlag"
	case KindAbsent:
		return "KindAbsent"
	}
	return "BuffKind(unknown)"
}

type TalentApplies int

const (
	TalentScalesValue TalentApplies = iota
	TalentScalesDuration
	TalentAddsStat
)

func (a TalentApplies) String() string {
	switch a {
	case TalentScalesValue:
		return "TalentScalesValue"
	case TalentScalesDuration:
		return "TalentScalesDuration"
	case TalentAddsStat:
		return "TalentAddsStat"
	}
	return "TalentApplies(unknown)"
}

type TalentMod struct {
	// SpellID is the spell of the trait node that prices the improvement, which the store carries as
	// a talent ladder.
	SpellID int32
	Name    string
	// Effect is the index of the modifying effect inside the talent spell.
	Effect  int32
	Applies TalentApplies
}

type ActionRef struct {
	SpellID int32
	ItemID  int32
}

type BuffSpec struct {
	Field  string // proto field name, snake_case, owns the number
	Number int32  // proto field number
	Scope  BuffScope
	Proto  BuffProtoType // bool when nothing prices an improved state
	Kind   BuffKind
	Go     string // identifier stem: "BattleShout"
	// SpellID is the spell the aura's numbers are read from: the top rank of the castable family, or
	// the aura that family's cast applies when the cast is a summon or a dummy. CastID is the cast
	// itself, for a row whose timing only the cast states. Both are roots of the spell store.
	SpellID        int32
	CastID         int32
	Name           string      // SpellName.Name_lang of the castable family ("" for rows with no spell)
	AuraName       string      // aura family when the cast is a summon or dummy (totems: "Strength of Earth")
	Owner          proto.Class // class that casts it; ClassUnknown for none
	Talent         *TalentMod  // improving talent family, nil when none
	Category       string      // exclusive-effect category value, "" = none
	SharedCategory string      // second exclusive category the aura also joins, "" = none
	SingleAura     bool
	Driver         bool // apply block hands the field to drive<Go>; the aura is not simply always up
	// SkipAuras names auras of the spell the buff leaves out, spelled the way
	// sim/core/dbcenums spells them, for a row the client states beside the
	// buff that the raid's copy does not apply. The names are checked while the
	// row is resolved.
	SkipAuras []string
	Stats     []proto.Stat // UI relevance tags
	// ImpAction names the improved state's source when it is not a talent: an
	// item, or the spell an item set grants at a piece threshold. It is the icon
	// the improved state shows, and a tristate row states it or a Talent.
	ImpAction *ActionRef
	Label     string // UI label override, "" = DB name
	Notes     string // reason for manual/absent rows; emitted as a comment
}

// GoField is the field name protoc-gen-go generates for Field. It ports
// protobuf-go's strs.GoCamelCase, including the branch that keeps the underscore
// in front of a digit ("soe_enhancement_2pt4" gives "SoeEnhancement_2Pt4").
func (s BuffSpec) GoField() string {
	var b []byte
	for i := 0; i < len(s.Field); i++ {
		c := s.Field[i]
		switch {
		case c == '_' && i+1 < len(s.Field) && isASCIILower(s.Field[i+1]):
		case isASCIIDigit(c):
			b = append(b, c)
		default:
			if isASCIILower(c) {
				c -= 'a' - 'A'
			}
			b = append(b, c)
			for ; i+1 < len(s.Field) && isASCIILower(s.Field[i+1]); i++ {
				b = append(b, s.Field[i+1])
			}
		}
	}
	return string(b)
}

// TSField is the property name protobuf-ts generates for Field: lower camel case
// in which a digit also capitalises the character that follows it
// ("soe_enhancement_2pt4" gives "soeEnhancement2Pt4").
func (s BuffSpec) TSField() string {
	var b strings.Builder
	capNext := false
	for i := 0; i < len(s.Field); i++ {
		c := s.Field[i]
		switch {
		case c == '_':
			capNext = true
		case isASCIIDigit(c):
			b.WriteByte(c)
			capNext = true
		case capNext:
			if isASCIILower(c) {
				c -= 'a' - 'A'
			}
			b.WriteByte(c)
			capNext = false
		case i == 0:
			if !isASCIILower(c) {
				c += 'a' - 'A'
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func ByScope(scope BuffScope) []BuffSpec {
	var out []BuffSpec
	for _, spec := range Manifest {
		if spec.Scope == scope {
			out = append(out, spec)
		}
	}
	return out
}

func isASCIILower(c byte) bool { return c >= 'a' && c <= 'z' }

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }
