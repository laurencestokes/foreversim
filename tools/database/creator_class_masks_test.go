package database

// A soulbound item only class spells create belongs to that class, though the game files leave it
// open to every class; a tradable one stays open. Runs the query on a fixture shaped like the game
// files, holding only the columns it reads.

import (
	"database/sql"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/tools/database/dbc"
)

func TestCreatorClassMasks(t *testing.T) {
	const mage, warlock = 1 << 7, 1 << 8
	type creator struct {
		spellID   int
		effect    dbcenums.SpellEffectType
		classMask int
	}
	cases := []struct {
		name           string
		bonding        int
		allowableClass int
		creators       []creator
		want           []proto.Class
	}{
		{"soulbound, open as -1, one warlock spell", dbc.BIND_ON_ACQUIRE, -1,
			[]creator{{1001, dbcenums.E_CREATE_ITEM, warlock}}, []proto.Class{proto.Class_ClassWarlock}},
		{"soulbound, open as 0, one mage spell", dbc.BIND_ON_ACQUIRE, 0,
			[]creator{{1002, dbcenums.E_CREATE_ITEM, mage}}, []proto.Class{proto.Class_ClassMage}},
		{"soulbound, two ranks of one class", dbc.BIND_ON_ACQUIRE, -1,
			[]creator{{1003, dbcenums.E_CREATE_ITEM, warlock}, {1004, dbcenums.E_CREATE_ITEM, warlock}}, []proto.Class{proto.Class_ClassWarlock}},
		{"soulbound, spells of two classes", dbc.BIND_ON_ACQUIRE, -1,
			[]creator{{1005, dbcenums.E_CREATE_ITEM, mage}, {1006, dbcenums.E_CREATE_ITEM, warlock}},
			[]proto.Class{proto.Class_ClassMage, proto.Class_ClassWarlock}},
		{"tradable", 0, -1, []creator{{1007, dbcenums.E_CREATE_ITEM, warlock}}, nil},
		{"soulbound, already restricted", dbc.BIND_ON_ACQUIRE, mage, []creator{{1008, dbcenums.E_CREATE_ITEM, warlock}}, nil},
		{"soulbound, by a spell no class learns", dbc.BIND_ON_ACQUIRE, -1, []creator{{1009, dbcenums.E_CREATE_ITEM, 0}}, nil},
		{"soulbound, named by an effect that creates nothing", dbc.BIND_ON_ACQUIRE, -1, []creator{{1010, dbcenums.E_DUMMY, warlock}}, nil},
	}

	var effects, abilities, items []string
	for i, c := range cases {
		itemID := 100 + i
		items = append(items, fmt.Sprintf("(%d, %d, %d)", itemID, c.bonding, c.allowableClass))
		for _, cr := range c.creators {
			effects = append(effects, fmt.Sprintf("(%d, %d, %d, %d)", len(effects)+1, cr.spellID, cr.effect, itemID))
			abilities = append(abilities, fmt.Sprintf("(%d, %d, %d)", len(abilities)+1, cr.spellID, cr.classMask))
		}
	}
	fixture := fmt.Sprintf(`
CREATE TABLE SpellEffect (ID INTEGER PRIMARY KEY, SpellID INTEGER, Effect INTEGER, EffectItemType INTEGER);
CREATE TABLE SkillLineAbility (ID INTEGER PRIMARY KEY, Spell INTEGER, ClassMask INTEGER);
CREATE TABLE ItemSparse (ID INTEGER PRIMARY KEY, Bonding INTEGER, AllowableClass INTEGER);
INSERT INTO SpellEffect (ID, SpellID, Effect, EffectItemType) VALUES %s;
INSERT INTO SkillLineAbility (ID, Spell, ClassMask) VALUES %s;
INSERT INTO ItemSparse (ID, Bonding, AllowableClass) VALUES %s;`,
		strings.Join(effects, ", "), strings.Join(abilities, ", "), strings.Join(items, ", "))

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "creators.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(fixture); err != nil {
		t.Fatalf("building the fixture: %v", err)
	}
	masks, err := LoadCreatorClassMasks(&DBHelper{db: db})
	if err != nil {
		t.Fatal(err)
	}

	for i, c := range cases {
		itemID := 100 + i
		mask, derived := masks[itemID]
		if derived != (c.want != nil) {
			t.Errorf("%s: class mask %d derived: %v, want %v", c.name, mask, derived, c.want != nil)
			continue
		}
		if got := dbc.GetClassesFromClassMask(mask); derived && !slices.Equal(got, c.want) {
			t.Errorf("%s: classes %v, want %v", c.name, got, c.want)
		}
	}
}

// On the game files, every mask derived names at least one class.
func TestCreatorClassMasksNameAClass(t *testing.T) {
	masks, err := LoadCreatorClassMasks(openBuffTestDB(t))
	if err != nil {
		t.Fatalf("%v", err)
	}
	for _, id := range slices.Sorted(maps.Keys(masks)) {
		if len(dbc.GetClassesFromClassMask(masks[id])) == 0 {
			t.Errorf("item %d: class mask %d names no class", id, masks[id])
		}
	}
}
