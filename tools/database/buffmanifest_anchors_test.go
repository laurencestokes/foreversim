package database

// The manifest pins the spell each buff reads, so generation needs no client database. What pinned
// it is the rank resolution below, which reads SkillLineAbility, SkillLine and the rune enchantments -
// tables the store does not capture - so this is where a new client is checked against the pins.
// Skips without tools/database/wowsims.db.

import (
	"database/sql"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/tools/database/buffmanifest"
	"github.com/wowsims/forever/tools/database/dbc"
)

func openBuffTestDB(t *testing.T) *DBHelper {
	t.Helper()

	DatabasePath = "wowsims.db"
	if _, err := os.Stat(DatabasePath); err != nil {
		t.Skipf("no client database at %s - run `make db` from a local WoW install to enable this gate", DatabasePath)
	}

	helper, err := NewDBHelper()
	if err != nil {
		t.Fatalf("opening %s: %v", DatabasePath, err)
	}
	t.Cleanup(func() { helper.Close() })
	return helper
}

func TestManifestAnchorsMatchTheClient(t *testing.T) {
	db := openBuffTestDB(t).db

	runes, err := loadRuneGrantedSpells(db)
	if err != nil {
		t.Fatalf("%v", err)
	}

	for _, spec := range buffmanifest.Manifest {
		// A row without a castable family - an item's aura, an elixir's debuff - is pinned by hand.
		if spec.Name == "" {
			continue
		}

		classMask := ownerClassMask(spec.Owner)
		cands, err := anchorCandidates(db, spec.Name, classMask)
		if err != nil {
			t.Fatalf("%s: %v", spec.Field, err)
		}
		if len(cands) == 0 {
			t.Errorf("%s: no SkillLineAbility row grants %q to %s", spec.Field, spec.Name, spec.Owner)
			continue
		}

		cast := topRank(cands, classMask)
		aura := cast.SpellID
		if spec.AuraName != "" {
			if aura, err = auraFamilyMember(db, spec.AuraName, cast.Subtext, spec.Kind != buffmanifest.KindProc); err != nil {
				t.Fatalf("%s: %v", spec.Field, err)
			}
		}

		if spec.SpellID != aura {
			t.Errorf("%s: the manifest pins spell %d, the client resolves %d", spec.Field, spec.SpellID, aura)
		}
		if spec.CastID != 0 && spec.CastID != cast.SpellID {
			t.Errorf("%s: the manifest pins cast %d, the client resolves %d", spec.Field, spec.CastID, cast.SpellID)
		}
		if spec.Kind == buffmanifest.KindExternalCD && cast.SpellID != aura && spec.CastID == 0 {
			t.Errorf("%s: the cast %d times the cooldown and states what the aura %d does not, so the manifest has to pin it",
				spec.Field, cast.SpellID, aura)
		}
		if runes[aura] {
			t.Errorf("%s: spell %d is granted by a rune, which is a class rune and not a raid buff", spec.Field, aura)
		}
	}
}

// The two skill lines that grant runes rather than class abilities. Both are CategoryID 7, so the
// anchor query has to name them.
const (
	skillLineEngraving = 2851
	skillLineRunes     = 2853
)

var buffRankSubtext = regexp.MustCompile(`^Rank (\d+)$`)

type buffCandidate struct {
	SpellID       int32
	Subtext       string
	Rank          int32
	ClassMask     int32
	AcquireMethod int32
	Supercedes    int32
}

func queryEach(db *sql.DB, scan func(*sql.Rows) error, query string, args ...any) error {
	rows, err := db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// The spells a rune grants. An `Engrave <slot> - <name>` spell enchants the item with a
// SpellItemEnchantment whose EffectArg columns hold the spells the rune teaches.
func loadRuneGrantedSpells(db *sql.DB) (map[int32]bool, error) {
	granted := map[int32]bool{}
	err := queryEach(db, func(rows *sql.Rows) error {
		var a, b, c int32
		if err := rows.Scan(&a, &b, &c); err != nil {
			return err
		}
		for _, id := range []int32{a, b, c} {
			if id != 0 {
				granted[id] = true
			}
		}
		return nil
	}, `
		SELECT COALESCE(sie.EffectArg_0, 0), COALESCE(sie.EffectArg_1, 0), COALESCE(sie.EffectArg_2, 0)
		FROM SpellEffect se
		JOIN SpellName n ON n.ID = se.SpellID
		JOIN SpellItemEnchantment sie ON sie.ID = se.EffectMiscValue_0
		WHERE se.Effect = 54 AND n.Name_lang LIKE 'Engrave % - %'`)
	return granted, err
}

// Every spell of the named family the owner can learn. CategoryID 7 is the class and profession
// skill lines.
func anchorCandidates(db *sql.DB, name string, classMask int32) ([]buffCandidate, error) {
	var out []buffCandidate
	err := queryEach(db, func(rows *sql.Rows) error {
		var c buffCandidate
		if err := rows.Scan(&c.SpellID, &c.Subtext, &c.ClassMask, &c.AcquireMethod, &c.Supercedes); err != nil {
			return err
		}
		if m := buffRankSubtext.FindStringSubmatch(c.Subtext); m != nil {
			rank, _ := strconv.Atoi(m[1])
			c.Rank = int32(rank)
		}
		out = append(out, c)
		return nil
	}, `
		SELECT sla.Spell, COALESCE(s.NameSubtext_lang, ''), sla.ClassMask,
		       sla.AcquireMethod, COALESCE(sla.SupercedesSpell, 0)
		FROM SkillLineAbility sla
		JOIN SpellName n ON n.ID = sla.Spell
		JOIN Spell s ON s.ID = sla.Spell
		JOIN SkillLine sl ON sl.ID = sla.SkillLine AND sl.CategoryID = 7
		  AND sl.ID NOT IN (?, ?)
		WHERE n.Name_lang = ? AND ((sla.ClassMask & ?) != 0 OR sla.ClassMask = 0)
		ORDER BY sla.Spell`, skillLineEngraving, skillLineRunes, name, classMask)
	return out, err
}

// The rank the player ends up with. A family with rank subtexts is ordered by them, each rank
// narrowed to its lowest acquire method, since Forever re-issues single ranks as granted copies of a
// trained ability. A family without one - Blessing of Kings, Innervate - is ordered by
// SupercedesSpell, where the spell no other row supersedes is the last one learnt.
func topRank(cands []buffCandidate, classMask int32) buffCandidate {
	var owned []buffCandidate
	for _, c := range cands {
		if classMask != 0 && c.ClassMask&classMask != 0 {
			owned = append(owned, c)
		}
	}
	if len(owned) == 0 {
		owned = cands
	}

	top := int32(0)
	for _, c := range owned {
		top = max(top, c.Rank)
	}
	if top > 0 {
		var ranked []buffCandidate
		best := int32(math.MaxInt32)
		for _, c := range owned {
			if c.Rank == top {
				ranked = append(ranked, c)
				best = min(best, c.AcquireMethod)
			}
		}
		return ranked[slices.IndexFunc(ranked, func(c buffCandidate) bool { return c.AcquireMethod == best })]
	}

	superseded := map[int32]bool{}
	for _, c := range owned {
		superseded[c.Supercedes] = true
	}
	for _, c := range owned {
		if !superseded[c.SpellID] {
			return c
		}
	}
	return owned[0]
}

// The aura of a totem or of a dummy passive, which the client only ties to the cast by name. The
// rank subtext of the cast picks the matching rank; where neither carries one - Leader of the Pack
// is 17007 and 24932, both rankless - the spell that applies a party or raid aura is the one other
// players see.
// preferShared picks the aura that reaches the party over a same-named one that only its caster
// holds. A proc row wants the reverse: since client build 70009 Windfury Totem's party aura (10612)
// shares the name of the proc aura it hands out (10610), and the proc aura is the one that states it.
func auraFamilyMember(db *sql.DB, name string, subtext string, preferShared bool) (int32, error) {
	type auraCandidate struct {
		SpellID int32
		Subtext string
		Shared  bool
	}
	seen := map[int32]int{}
	var cands []auraCandidate
	err := queryEach(db, func(rows *sql.Rows) error {
		var id, target int32
		var sub string
		var effect dbc.SpellEffectType
		if err := rows.Scan(&id, &sub, &effect, &target); err != nil {
			return err
		}
		shared := effect != dbcenums.E_APPLY_AURA || isSharedTarget(dbc.ImplicitTarget(target))
		if index, ok := seen[id]; ok {
			cands[index].Shared = cands[index].Shared || shared
			return nil
		}
		seen[id] = len(cands)
		cands = append(cands, auraCandidate{SpellID: id, Subtext: sub, Shared: shared})
		return nil
	}, `
		SELECT n.ID, COALESCE(s.NameSubtext_lang, ''), e.Effect, COALESCE(e.ImplicitTarget_0, 0)
		FROM SpellName n
		JOIN Spell s ON s.ID = n.ID
		JOIN SpellEffect e ON e.SpellID = n.ID
		WHERE n.Name_lang = ? AND e.Effect IN (?, ?, ?)
		ORDER BY n.ID, e.EffectIndex`,
		name, dbcenums.E_APPLY_AURA, dbcenums.E_APPLY_AREA_AURA_PARTY, dbcenums.E_APPLY_AREA_AURA_RAID)
	if err != nil || len(cands) == 0 {
		return 0, err
	}

	var matching []auraCandidate
	for _, c := range cands {
		if c.Subtext == subtext {
			matching = append(matching, c)
		}
	}
	if len(matching) == 0 {
		matching = cands
	}
	if len(matching) > 1 {
		for _, c := range matching {
			if c.Shared == preferShared {
				return c.SpellID, nil
			}
		}
	}
	return matching[0].SpellID, nil
}

func isSharedTarget(target dbc.ImplicitTarget) bool {
	switch target {
	case dbc.TARGET_UNIT_CASTER_AREA_PARTY, dbc.TARGET_UNIT_CASTER_AREA_RAID,
		dbc.TARGET_UNIT_TARGET_ALLY, dbc.TARGET_UNIT_TARGET_RAID, dbc.TARGET_153:
		return true
	}
	return false
}

// SkillLineAbility.ClassMask is a bitmask over the client's class ids, which is not the order
// proto.Class uses.
func ownerClassMask(class proto.Class) int32 {
	for _, c := range dbc.Classes {
		if c.ProtoClass == class {
			return int32(1) << (c.ID - 1)
		}
	}
	return 0
}
