package paladin

import (
	"testing"

	"github.com/wowsims/forever/sim/core/spelldata/parity"
)

// Every number this package's generated tables state, against the same spell as sim/core/spelldata
// carries it: the store is what replaces the tables, so it has to reproduce them first.
func TestParityWithTheFamilyTables(t *testing.T) {
	parity.Check(t, "paladin", spellData)
}
