package buffmanifest

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestUniqueScopeField(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range Manifest {
		key := fmt.Sprintf("%s/%s", spec.Scope, spec.Field)
		if seen[key] {
			t.Errorf("duplicate (scope, field): %s", key)
		}
		seen[key] = true
	}
}

func TestUniqueScopeNumber(t *testing.T) {
	seen := map[string]string{}
	for _, spec := range Manifest {
		key := fmt.Sprintf("%s/%d", spec.Scope, spec.Number)
		if other, ok := seen[key]; ok {
			t.Errorf("duplicate (scope, number) %s: %s and %s", key, other, spec.Field)
		}
		seen[key] = spec.Field
	}
}

func TestScopeNumbersAreDense(t *testing.T) {
	for _, scope := range []BuffScope{ScopeRaid, ScopeParty, ScopeIndividual, ScopeDebuff} {
		rows := ByScope(scope)
		taken := make([]int32, len(rows))
		for i, spec := range rows {
			taken[i] = spec.Number
		}
		slices.Sort(taken)
		for i, number := range taken {
			if number != int32(i+1) {
				t.Errorf("%s takes the numbers %v, want 1..%d", scope, taken, len(rows))
				break
			}
		}
	}
}

func TestUniqueGoStem(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range Manifest {
		if spec.Go == "" {
			t.Errorf("%s has no Go stem", spec.Field)
			continue
		}
		if seen[spec.Go] {
			t.Errorf("duplicate Go stem: %s", spec.Go)
		}
		seen[spec.Go] = true
		if want := strings.ReplaceAll(spec.GoField(), "_", ""); spec.Go != want {
			t.Errorf("%s has Go stem %q, want %q", spec.Field, spec.Go, want)
		}
	}
}

func TestShellRowsHaveNotes(t *testing.T) {
	for _, spec := range Manifest {
		switch spec.Kind {
		case KindManual, KindAbsent, KindFlag:
			if spec.Notes == "" {
				t.Errorf("%s is %s and needs Notes", spec.Field, spec.Kind)
			}
		}
	}
}

func TestResolvableRowsNameASpell(t *testing.T) {
	for _, spec := range Manifest {
		switch spec.Kind {
		case KindAbsent, KindFlag:
			continue
		}
		if spec.SpellID == 0 {
			t.Errorf("%s (%s) names no spell", spec.Field, spec.Kind)
		}
	}
}

func TestProtoTypeMatchesKind(t *testing.T) {
	for _, spec := range Manifest {
		switch spec.Kind {
		case KindFlag:
			if spec.Proto != ProtoBool && spec.Proto != ProtoDouble {
				t.Errorf("%s is KindFlag and must be %s or %s, got %s", spec.Field, ProtoBool, ProtoDouble, spec.Proto)
			}
		case KindDebuffUptime:
			if spec.Proto != ProtoDouble {
				t.Errorf("%s is KindDebuffUptime and must be %s, got %s", spec.Field, ProtoDouble, spec.Proto)
			}
		case KindItemCount, KindExternalCD:
			if spec.Proto != ProtoInt32 {
				t.Errorf("%s is %s and must be %s, got %s", spec.Field, spec.Kind, ProtoInt32, spec.Proto)
			}
		}
	}
}

func TestTalentImpliesTristate(t *testing.T) {
	for _, spec := range Manifest {
		if spec.Talent != nil && spec.Proto != ProtoTristate {
			t.Errorf("%s carries talent %q but is %s", spec.Field, spec.Talent.Name, spec.Proto)
		}
	}
}

func TestFieldNaming(t *testing.T) {
	cases := []struct {
		field, goName, tsName string
	}{
		{"battle_shout", "BattleShout", "battleShout"},
		{"soe_enhancement_2pt4", "SoeEnhancement_2Pt4", "soeEnhancement2Pt4"},
		{"joc_retribution_2pt4", "JocRetribution_2Pt4", "jocRetribution2Pt4"},
		{"snapshot_bs_booming_voice_rank", "SnapshotBsBoomingVoiceRank", "snapshotBsBoomingVoiceRank"},
		{"isb_uptime", "IsbUptime", "isbUptime"},
		{"expose_weakness_hunter_agility", "ExposeWeaknessHunterAgility", "exposeWeaknessHunterAgility"},
	}
	for _, c := range cases {
		spec := BuffSpec{Field: c.field}
		if got := spec.GoField(); got != c.goName {
			t.Errorf("GoField(%q) = %q, want %q", c.field, got, c.goName)
		}
		if got := spec.TSField(); got != c.tsName {
			t.Errorf("TSField(%q) = %q, want %q", c.field, got, c.tsName)
		}
	}
}

func TestFieldNamesRoundTrip(t *testing.T) {
	for _, spec := range Manifest {
		goName := spec.GoField()
		if goName == "" {
			t.Errorf("%s has an empty GoField", spec.Field)
			continue
		}
		want := strings.ReplaceAll(goName, "_", "")
		want = strings.ToLower(want[:1]) + want[1:]
		if got := spec.TSField(); got != want {
			t.Errorf("%s: TSField %q does not match GoField %q", spec.Field, got, goName)
		}
	}
}

// proto/buffs.proto is rendered from this manifest, so this test is not an independent oracle for
// the field set: it catches a hand edit to the committed proto drifting from the manifest, and
// nothing more. What the numbers and types are checked against is gen_buffs_proto's
// TestRenderMatchesCommittedFile: the committed file is what the manifest renders.
func TestCensusMatchesProto(t *testing.T) {
	messages := map[string]BuffScope{
		"RaidBuffs":       ScopeRaid,
		"PartyBuffs":      ScopeParty,
		"IndividualBuffs": ScopeIndividual,
		"Debuffs":         ScopeDebuff,
	}

	want := map[string]protoField{}
	for message, scope := range messages {
		for field, declared := range parseProtoMessage(t, message) {
			want[fmt.Sprintf("%s/%s", scope, field)] = declared
		}
	}

	got := map[string]bool{}
	for _, spec := range Manifest {
		key := fmt.Sprintf("%s/%s", spec.Scope, spec.Field)
		got[key] = true

		declared, ok := want[key]
		if !ok {
			t.Errorf("%s is in the manifest but not in buffs.proto", key)
			continue
		}
		if spec.Number != declared.number {
			t.Errorf("%s has number %d in the manifest and %d in buffs.proto", key, spec.Number, declared.number)
		}
		if !declared.allows(spec.Proto) {
			t.Errorf("%s is %s in the manifest and %s in buffs.proto", key, spec.Proto, declared.kind)
		}
	}
	for key := range want {
		if !got[key] {
			t.Errorf("%s is in buffs.proto but not in the manifest", key)
		}
	}
}

type protoField struct {
	kind   string
	number int32
}

// allows reports whether a manifest row may carry proto type p.
func (f protoField) allows(p BuffProtoType) bool {
	switch f.kind {
	case "bool":
		return p == ProtoBool
	case "int32":
		return p == ProtoInt32
	case "double":
		return p == ProtoDouble
	case "TristateEffect":
		return p == ProtoTristate
	}
	return false
}

var protoFieldRE = regexp.MustCompile(`^\s*([A-Za-z][\w.]*)\s+([a-z][a-z0-9_]*)\s*=\s*(\d+)\s*;`)

func parseProtoMessage(t *testing.T, message string) map[string]protoField {
	t.Helper()

	raw, err := os.ReadFile("../../../proto/buffs.proto")
	if err != nil {
		t.Fatalf("read buffs.proto: %v", err)
	}

	fields := map[string]protoField{}
	inMessage := false
	for _, line := range strings.Split(string(raw), "\n") {
		if !inMessage {
			inMessage = strings.HasPrefix(strings.TrimSpace(line), "message "+message+" {")
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "}") {
			break
		}
		match := protoFieldRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[3])
		if err != nil {
			t.Fatalf("field %s of %s: %v", match[2], message, err)
		}
		fields[match[2]] = protoField{kind: match[1], number: int32(number)}
	}

	if len(fields) == 0 {
		t.Fatalf("no fields parsed for message %s", message)
	}
	return fields
}
