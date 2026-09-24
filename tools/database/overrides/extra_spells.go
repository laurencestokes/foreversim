package overrides

// A spell the store carries although no class, item, enchant or set bonus reaches it. Reason and
// Source weigh the same as an Override's, and the generator refuses an entry without a reason.
type Extra struct {
	SpellID int32
	Reason  string
	Source  string
}

// The spells the generator adds to the store's roots by hand.
var ExtraSpells = []Extra{}
