package database

// A settings input's icon is the spell the settings tab shows a user as its tooltip, so it must be
// one the game files state stats for. Skips without tools/database/wowsims.db.

import (
	"testing"
)

func TestBuffInputActionIDsStateTheirTooltip(t *testing.T) {
	db := openBuffTestDB(t).db

	for _, row := range resolveCommittedBuffs(t) {
		if skipBuffInputReason(row) != "" {
			continue
		}
		id := buffInputActionID(row)
		var desc string
		if err := scanOptional(db, `SELECT COALESCE(Description_lang, '') FROM Spell WHERE ID = ?`, id, &desc); err != nil {
			t.Fatalf("%s: reading spell %d: %v", row.Field, id, err)
		}
		if desc == "" {
			t.Errorf("%s: action id %d states no Description_lang", row.Field, id)
		}
	}
}
