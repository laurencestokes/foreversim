// Package parity compares a class package's generated family tables with the same spells as
// sim/core/spelldata carries them, so that the store can be shown to reproduce every number the
// tables state before anything reads it instead of them.
//
// It lives in its own package because the tables are an unexported variable of each class package:
// the nine drivers are test files inside those packages, and the comparison they share is here.
package parity

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"time"

	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The level the family tables fold their amounts at, which is the level the sim plays.
const level = 60

// What a test hands the check. Named rather than taking *testing.T so that this package, which the
// class packages import, does not pull testing into a non-test build.
type Reporter interface {
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
}

// One value the table states and the store does not reproduce.
type Mismatch struct {
	Family  string
	Rank    int32
	SpellID int32
	Field   string
	Table   string
	Store   string
}

// How many mismatches are spelled out before the rest are counted: a systematic miss repeats, and
// ten rows of it say as much as a thousand.
const reported = 10

// Fields of shared.SpellData and shared.SpellDataEffect that no store row can answer for, with the
// reason each is not compared.
//
//	SpellData.Rank              the generator's position in the ladder, not a client column
//	SpellData.PPM               the client states none (SpellProcsPerMinuteID is 0 on every row);
//	                            the tables get theirs from WithSpellDataPPM at the call site, and
//	                            the store from an override, so neither side reads the other
//	SpellDataEffect.ValueMax    the spread: this client dropped EffectDieSides, so the generator
//	                            derives no high end at all and every table effect leaves it zero.
//	                            The store's spread is EffectVariance, which is a different number
//	SpellDataValue.TickMax      the same, for a tick
//
// AP coefficients are hand-supplied on both sides - one client effect in 38357 states one - so they
// are compared where the client does carry one and left alone where the call site adds it.
func Check(r Reporter, class string, tables any) {
	var mismatches []Mismatch
	rows := 0

	for _, family := range families(tables) {
		talent := isTalentLadder(family.table)
		seenSpells := map[int32]bool{}
		for _, row := range family.table {
			rows++
			mismatches = append(mismatches,
				compareRow(family.name, row, talent, int32(len(family.table)), seenSpells[row.SpellID])...)
			seenSpells[row.SpellID] = true
		}
	}

	seen := map[int]bool{}
	var unexpected []Mismatch
	for _, m := range mismatches {
		if i, ok := gapFor(class, m); ok {
			seen[i] = true
			r.Logf("%s %s (spell %d) %s: a known gap - %s",
				class, m.Family, m.SpellID, m.Field, knownGaps[i].Why)
			continue
		}
		unexpected = append(unexpected, m)
	}

	r.Logf("%s: %d rows across the family tables, %d mismatches, %d of them known gaps",
		class, rows, len(mismatches), len(mismatches)-len(unexpected))

	for i, m := range unexpected {
		if i == reported {
			r.Errorf("%s: %d further mismatches not listed", class, len(unexpected)-reported)
			break
		}
		r.Errorf("%s %s rank %d (spell %d) %s: the table says %s, the store says %s",
			class, m.Family, m.Rank, m.SpellID, m.Field, m.Table, m.Store)
	}

	for i, gap := range knownGaps {
		if gap.Class == class && !seen[i] {
			r.Errorf("%s spell %d %s is listed as a known gap and the store now answers it - take the "+
				"entry out of knownGaps", class, gap.SpellID, gap.Field)
		}
	}
}

// What a family table states and the store cannot answer, with what has to change for each. Empty:
// the store answers every number the nine tables state. An entry here is tolerated rather than
// failed, so the list also fails when a gap on it stops happening - a fix takes its entry with it.
var knownGaps []struct {
	Class   string
	SpellID int32
	Rank    int32
	Field   string
	Why     string
}

func gapFor(class string, m Mismatch) (int, bool) {
	for i, gap := range knownGaps {
		if gap.Class == class && gap.SpellID == m.SpellID && gap.Rank == m.Rank && gap.Field == m.Field {
			return i, true
		}
	}
	return 0, false
}

type family struct {
	name  string
	table shared.SpellDataTable
}

// The generated tables of a class, by the field name each is declared under, in declaration order.
func families(tables any) []family {
	v := reflect.ValueOf(tables)
	if v.Kind() != reflect.Struct {
		panic(fmt.Sprintf("parity: want the class package's table struct, got %T", tables))
	}

	var out []family
	tableType := reflect.TypeOf(shared.SpellDataTable{})
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Type != tableType {
			continue
		}
		out = append(out, family{name: v.Type().Field(i).Name, table: v.Field(i).Interface().(shared.SpellDataTable)})
	}
	return out
}

// A trait talent is one spell whose ranks are priced by a curve, so every row states the same spell
// id. A ranked spell states a different one per rank, and a table of one row states no ladder at
// all: the store may still price that spell's ranks, and reading it as a one-rank talent would be
// the table's rank cap overruling the tree's.
func isTalentLadder(table shared.SpellDataTable) bool {
	if len(table) < 2 {
		return false
	}
	for _, row := range table[1:] {
		if row.SpellID != table[0].SpellID {
			return false
		}
	}
	return true
}

func compareRow(name string, row shared.SpellData, talent bool, ranks int32, spellSeen bool) []Mismatch {
	s := spelldata.Find(row.SpellID)
	if s == spelldata.Nil {
		return []Mismatch{{Family: name, Rank: row.Rank, SpellID: row.SpellID, Field: "the row itself",
			Table: "a rank of " + name, Store: "no row - the spell is not reachable from the store's roots"}}
	}

	// A talent's rank values live on the store's curve rather than on ranks of their own, so the row
	// is read off the ladder the store builds from it. A curve of another length is the store and the
	// table disagreeing about the rank cap, which Talent panics on.
	rank := s
	if talent {
		var failed *Mismatch
		rank, failed = talentRank(name, row, ranks)
		if failed != nil {
			return []Mismatch{*failed}
		}
	}

	c := checker{name: name, row: row, spell: s, rank: rank}

	// The fields that belong to the spell rather than to the rank are compared once per spell: a
	// talent ladder is one spell over and over, and reporting its cost five times would bury the rank
	// that actually differs.
	if !spellSeen {
		c.cost()
		c.positiveDuration("CastTime", row.CastTime, s.CastTime())
		c.positiveDuration("GCD", row.GCD, s.GCD())
		c.positiveDuration("Cooldown", row.Cooldown, max(s.Cooldown(), s.CategoryCooldown()))
		c.durationOrPermanent("Duration", row.Duration, s)
		c.positiveFloat("MinRange", row.MinRange, float64(s.MinRange))
		c.positiveFloat("MaxRange", row.MaxRange, float64(s.MaxRange))
		c.positiveFloat("MissileSpeed", row.MissileSpeed, float64(s.Speed))
		c.positiveInteger("ProcChance", int64(row.ProcChance), int64(s.ProcChance))
		c.positiveInteger("ProcCharges", int64(row.ProcCharges), int64(s.ProcCharges))
		c.positiveInteger("MaxTargets", int64(row.MaxTargets), int64(s.MaxTargets))
		c.boolean("RefundsOnMiss", row.RefundsOnMiss, s.RefundsOnMiss())
		c.boolean("PeriodicCanCrit", row.PeriodicCanCrit, s.PeriodicCanCrit())
		c.integer("SpellSchool", int64(row.SpellSchool), int64(s.SpellSchool()))
		c.integer("DefenseType", int64(row.DefenseType), int64(s.DefenseTypeCore()))
	}

	c.effects()
	c.threat()
	c.handSupplied()
	if !talent && ranks == 1 {
		c.oneRankLadder()
	}

	c.role("Direct", row.Direct)
	c.role("Heal", row.Heal)
	c.role("Periodic", row.Periodic)
	c.role("Energize", row.Energize)
	c.role("SecondaryPeriodic", row.SecondaryPeriodic)

	return c.out
}

func talentRank(name string, row shared.SpellData, ranks int32) (rank *spelldata.Spell, failed *Mismatch) {
	defer func() {
		if p := recover(); p != nil {
			failed = &Mismatch{Family: name, Rank: row.Rank, SpellID: row.SpellID, Field: "the rank ladder",
				Table: fmt.Sprintf("%d ranks", ranks), Store: fmt.Sprint(p)}
		}
	}()
	return spelldata.Talent(row.SpellID, ranks).Rank(row.Rank), nil
}

// The cost off the bar the generator read it from, which is the first SpellPower row; rage is the
// one bar the client states in tenths.
//
// A spell with no SpellLevels row states no cost in the table at all: the generator's rank loader
// reads the cost in the same query as the levels and stops at the missing row. Six family rows are
// on one - Silence, Ghostly Strike, Riposte, Divine Favor, Shadowform and Curse of Exhaustion - and
// each carries its cost in the store and none in the table. There is nothing to reproduce there, so
// it is only checked the other way.
//
// Named by id rather than read off the row, so that a seventh spell losing its levels row shows up
// as a mismatch instead of quietly joining the list.
var costlessInTheTable = []int32{14251, 14278, 15473, 15487, 18223, 20216}

func (c *checker) cost() {
	if slices.Contains(costlessInTheTable, c.row.SpellID) && c.row.Cost == 0 && c.row.PowerCostPct == 0 {
		return
	}
	c.positiveInteger("Cost", int64(c.row.Cost), int64(powerCost(c.spell)))
	c.float("PowerCostPct", c.row.PowerCostPct, float64(firstPower(c.spell).CostPct))
}

func powerCost(s *spelldata.Spell) float64 {
	return s.PowerCost(firstPower(s).Type)
}

func firstPower(s *spelldata.Spell) *spelldata.Power {
	if len(s.Powers) == 0 {
		return &spelldata.Power{}
	}
	return &s.Powers[0]
}

type checker struct {
	name  string
	row   shared.SpellData
	spell *spelldata.Spell
	rank  *spelldata.Spell
	out   []Mismatch
}

func (c *checker) add(field string, table, store any) {
	c.out = append(c.out, Mismatch{Family: c.name, Rank: c.row.Rank, SpellID: c.row.SpellID,
		Field: field, Table: fmt.Sprint(table), Store: fmt.Sprint(store)})
}

func (c *checker) integer(field string, table, store int64) {
	if table != store {
		c.add(field, table, store)
	}
}

func (c *checker) float(field string, table, store float64) {
	if table != store {
		c.add(field, table, store)
	}
}

func (c *checker) boolean(field string, table, store bool) {
	if table != store {
		c.add(field, table, store)
	}
}

func (c *checker) duration(field string, table, store time.Duration) {
	if table != store {
		c.add(field, table, store)
	}
}

// The generator emits these only where the client's number is positive, so a table zero says the
// store's is not positive rather than that it is zero: the client states an instant ranged shot as a
// cast time of -1000000.
func (c *checker) positiveInteger(field string, table, store int64) {
	if table == 0 {
		if store > 0 {
			c.add(field, "nothing", store)
		}
		return
	}
	c.integer(field, table, store)
}

func (c *checker) positiveFloat(field string, table, store float64) {
	if table == 0 {
		if store > 0 {
			c.add(field, "nothing", store)
		}
		return
	}
	c.float(field, table, store)
}

func (c *checker) positiveDuration(field string, table, store time.Duration) {
	if table == 0 {
		if store > 0 {
			c.add(field, "nothing", store)
		}
		return
	}
	c.duration(field, table, store)
}

// The table states nothing for an aura that never expires: the generator emits a duration only when
// it is positive, and the client's permanent aura is -1.
func (c *checker) durationOrPermanent(field string, table time.Duration, s *spelldata.Spell) {
	if table == 0 && s.DurationMs <= 0 {
		return
	}
	c.duration(field, table, s.Duration())
}

// Every effect the row states, against the effect at the same position in the store. The table
// counts positions too - the generator writes the client's EffectIndex into Index and emits the
// effects in that order - so position i answers position i.
func (c *checker) effects() {
	if len(c.row.Effects) != len(c.spell.Effects) {
		c.add("len(Effects)", len(c.row.Effects), len(c.spell.Effects))
		return
	}

	for i, te := range c.row.Effects {
		e := c.spell.EffectN(i + 1)
		where := fmt.Sprintf("Effects[%d]", i)

		c.integer(where+".Index", int64(te.Index), int64(e.Index))
		c.integer(where+".Effect", int64(te.Effect), int64(e.Type))
		c.integer(where+".Aura", int64(te.Aura), int64(e.Aura))
		c.integer(where+".Misc", int64(te.Misc), int64(e.Misc))

		if !c.statesValue(i, te.Value) {
			c.add(where+".Value", te.Value, c.valuesAt(i))
		}

		// The spread left the client with EffectDieSides, so the generator derives none and no table
		// effect carries one. A nonzero here would be a generator change the store has to answer.
		if te.ValueMax != 0 {
			c.add(where+".ValueMax", te.ValueMax, "the store's spread is Variance, not a high end")
		}

		// The client's default is 1 and the generator writes only what differs from it, rounding the
		// float32 the DB widened.
		amp := round6(float64(e.ChainAmp))
		if te.ChainAmplitude != 0 && te.ChainAmplitude != amp {
			c.add(where+".ChainAmplitude", te.ChainAmplitude, amp)
		}
		if te.ChainAmplitude == 0 && amp != 0 && amp != 1 {
			c.add(where+".ChainAmplitude", "the client's default", amp)
		}
	}
}

// The two ways a table effect's number is arrived at: a rank priced by the talent tree states the
// curve's value, and everything else the amount at level 60.
//
// Which of the two applies is read off the rank rather than allowed either way: the client's base on
// a curved effect is usually the top rank's number - King of the Jungle's 60 is rank 3 of 20/40/60 -
// so accepting the base everywhere would let a low rank state the highest one's value. Only a
// position the curve prices at exactly the base is ambiguous, and there the two answers agree anyway
// unless the base is fractional.
func (c *checker) valuesAt(position int) []float64 {
	e := c.spell.EffectN(position + 1)
	if c.rank == c.spell {
		return []float64{e.Average(level)}
	}

	priced := c.rank.EffectN(position + 1).BaseValue()
	if priced != e.BaseValue() {
		return []float64{priced}
	}
	return []float64{e.Average(level), priced}
}

func (c *checker) statesValue(position int, value float64) bool {
	for _, v := range c.valuesAt(position) {
		if v == value {
			return true
		}
	}
	return false
}

// The client states a flat threat on the abilities whose point is threat, as an E_THREAT effect the
// generator files into FlatThreatBonus; the store keeps that effect and carries its own FlatThreat
// only where an override supplies one.
func (c *checker) threat() {
	if c.row.FlatThreatBonus == 0 {
		return
	}
	if c.spell.FlatThreat != 0 {
		c.float("FlatThreatBonus", c.row.FlatThreatBonus, float64(c.spell.FlatThreat))
		return
	}
	for _, owner := range c.owners(nil) {
		for i := range owner.Effects {
			e := &owner.Effects[i]
			if e.Type != dbcenums.E_THREAT && e.Type != dbcenums.E_THREAT_ALL {
				continue
			}
			if e.Average(level) == c.row.FlatThreatBonus {
				return
			}
		}
	}
	c.add("FlatThreatBonus", c.row.FlatThreatBonus, "no threat effect of that amount, and no override")
}

// A table of one row is how a one-rank passive talent is generated, and the store has to answer for
// it through a ladder as well as through Find: the sim reads a talent as Talent(id, 1).Rank(1)
// whether the tree prices it or not. A spell the tree does price more ranks of is not one of these,
// and says so by panicking rather than by answering a rank it does not have.
func (c *checker) oneRankLadder() {
	rank, ok := oneRank(c.row.SpellID)
	if !ok {
		return
	}
	for i := range c.spell.Effects {
		if got := rank.EffectN(i + 1).BaseValue(); got != c.spell.Effects[i].BaseValue() {
			c.add(fmt.Sprintf("Talent(id, 1).Rank(1).EffectN(%d)", i+1), c.spell.Effects[i].BaseValue(), got)
		}
	}
}

// The single rank of a one-rank ladder, and whether there is one to read. A table of one row is also
// what a talent's triggered spell generates as, and the tree may well price that spell's own ranks -
// Tactical Mastery's buff has five - which Talent says by panicking rather than by answering a rank
// it does not have. The recover covers that one call and nothing else.
func oneRank(spellID int32) (rank *spelldata.Spell, ok bool) {
	defer func() {
		if recover() != nil {
			rank, ok = nil, false
		}
	}()
	return spelldata.Talent(spellID, 1).Rank(1), true
}

// Procs per minute reach a table through WithSpellDataPPM at the call site, never through the
// generator, so a generated row carrying one is the client having gained a rate the store would
// have to read too.
func (c *checker) handSupplied() {
	if c.row.PPM != 0 {
		c.add("PPM", c.row.PPM, "the generator states none; the store's rate is RPPM, from an override")
	}
}

// A role value against the effect the generator filed under it. The table does not record which
// effect that was, so it is recovered by matching the amount and the coefficients, which is how the
// rank-table regeneration check reads one back too.
func (c *checker) role(field string, value shared.SpellDataValue) {
	if value == nil {
		return
	}

	owners := c.owners(value)
	if owners == nil {
		c.add(field+".SpellID", value.(shared.SpellDataPeriodic).SpellID,
			"no row - the named spell is not in the store")
		return
	}

	low, high := value.Range()
	periodic, ticks := value.(shared.SpellDataPeriodic)

	// The two ends of a role value always agree in this client: the generator derives the high end
	// from EffectDieSides, which the client dropped, so it writes a flat value and leaves TickMax
	// empty. One that did differ would be a spread the store answers out of Variance instead.
	if high != low {
		c.add(field+" spread", fmt.Sprintf("%v to %v", low, high),
			"the store's spread is Variance, and no generated role value carries one")
	}
	if ticks && periodic.TickMax != 0 {
		c.add(field+".TickMax", periodic.TickMax, "the generator derives no high end for a tick")
	}

	// A tick is looked for among the ticking effects first: Scorpid Poison's threat effect states the
	// same number as its tick does, and reading the amount back off the threat one would then check
	// the schedule against an effect that has none.
	for _, ticking := range []bool{true, false} {
		if ticking && !ticks {
			continue
		}
		for _, owner := range owners {
			for i := range owner.Effects {
				e := owner.Effects[i]
				if ticking && e.PeriodMs == 0 {
					continue
				}
				if !amountIs(&e, low) || e.SPCoef != value.BonusCoefficient() {
					continue
				}
				if e.APCoef != 0 && e.APCoef != value.APBonusCoefficient() {
					continue
				}
				if ticks {
					c.periodic(field, periodic, &e)
				}
				return
			}
		}
	}

	c.add(field, fmt.Sprintf("%v (coefficient %v)", low, value.BonusCoefficient()),
		fmt.Sprintf("no effect of spell %d or of a spell it reads from states that amount with that coefficient",
			c.row.SpellID))
}

// The spells a role value can have come from. The generator reads a rank that states no amount of
// its own off a spell its tooltip names, off a spell it triggers, or off a rank of the same name -
// Intimidating Shout's stun takes its damage from the rank that casts it - and the table records
// which one only for a tick.
func (c *checker) owners(value shared.SpellDataValue) []*spelldata.Spell {
	if periodic, ok := value.(shared.SpellDataPeriodic); ok && periodic.SpellID != 0 {
		named := spelldata.Find(periodic.SpellID)
		if named == spelldata.Nil {
			return nil
		}
		return []*spelldata.Spell{named}
	}

	owners := []*spelldata.Spell{c.rank}
	owners = append(owners, c.spell.Refs()...)
	owners = append(owners, c.spell.Triggered()...)
	return append(owners, spelldata.ByName(c.spell.Name)...)
}

// The tick schedule the generator derived: the period the tick runs on, and the rank's duration over
// it. A tick read off another spell has no period of its own - Consecration's damage sits on a spell
// the client links only from the tooltip - and takes the rank's periodic dummy's, which is what times
// it in the game.
func (c *checker) periodic(field string, value shared.SpellDataPeriodic, e *spelldata.Effect) {
	period := e.PeriodMs
	if period == 0 {
		period = c.timingPeriod()
	}
	c.duration(field+".TickLength", value.TickLength, time.Duration(period)*time.Millisecond)

	ticks := int32(0)
	if c.spell.DurationMs > 0 && period > 0 {
		ticks = c.spell.DurationMs / period
	}
	c.integer(field+".NumberOfTicks", int64(value.NumberOfTicks), int64(ticks))
}

func (c *checker) timingPeriod() int32 {
	for i := range c.spell.Effects {
		e := &c.spell.Effects[i]
		if e.Aura == dbcenums.A_PERIODIC_DUMMY && e.PeriodMs > 0 {
			return e.PeriodMs
		}
	}
	return 0
}

// A role value is the effect's amount at level 60, or the curve's value where a talent tree priced
// the rank.
func amountIs(e *spelldata.Effect, value float64) bool {
	return e.Average(level) == value || e.BaseValue() == value
}

func round6(f float64) float64 {
	return math.Round(f*1e6) / 1e6
}
