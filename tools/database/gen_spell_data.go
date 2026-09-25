package database

import (
	"database/sql"
	"errors"
	"fmt"
	"go/format"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/tools/database/dbc"
)

var rankSubtext = regexp.MustCompile(`^Rank (\d+)$`)

type generatedRow struct {
	Rank            int32
	SpellID         int32
	Cost            int32
	PowerCostPct    float64
	CastTimeMs      int32
	GCDMs           int32
	CooldownMs      int32
	DurationMs      int32
	MinRange        float64
	MaxRange        float64
	MissileSpeed    float64
	ProcChance      int32
	ProcCharges     int32
	MaxTargets      int32
	RefundsOnMiss   bool
	PeriodicCanCrit bool
	FlatThreatBonus float64
	SchoolMask      int32
	DefenseType     int32
	Effects         []generatedEffect
	Direct          *generatedAmount
	Heal            *generatedAmount
	Periodic        *generatedAmount
	Energize        *generatedAmount

	SecondaryPeriodic *generatedAmount
}

type generatedEffect struct {
	Index          int32
	Effect         dbc.SpellEffectType
	Aura           dbc.EffectAuraType
	Misc           int32
	Value          float64
	ValueMax       float64
	ChainAmplitude float64
	Coef           float64
	APCoef         float64
}

type generatedAmount struct {
	Min    float64
	Max    float64
	Coef   float64
	APCoef float64

	PeriodMs int32
	Ticks    int32

	// The effect's spell where it is not the rank's own, which only a periodic value renders.
	SpellID int32
}

// SkillLineAbility.AcquireMethod: 0 trainer, 1 with the skill, 2 on level, 3 granted by another spell.
const (
	acquireOnLevel = 2
	acquireGranted = 3
)

type rankCandidate struct {
	SpellID       int32
	Rank          int32
	ClassMask     int
	SkillLine     int32
	AcquireMethod int32
}

type rankLadder struct {
	Name  string
	Field string
	Ranks map[int32]int32

	// What each rank's effects are worth, keyed by rank and then by the client's EffectIndex. Set
	// only on a ladder the talent tree supplied, where every rank is the same spell and the numbers
	// sit on the tree's curves rather than on a spell per rank. Nil means the rank's own spell states
	// its numbers, which is what every "Rank N" family does.
	Points map[int32]map[int32]float64
}

// One talent as the tree states it: a single spell, the rank cap the game enforces, and the value
// each rank gives. The rank spells the legacy Talent table still lists are gone from this client -
// Anticipation's 12750-12753 have no row in SpellName, SpellEffect or SpellMisc - so the tree is the
// only place a talent's ranks are described.
type traitLadder struct {
	SpellID  int32
	MaxRanks int32
	Points   map[int32]map[int32]float64
}

// SkillLineAbility.ClassMask is a bitmask over the class index dbc.Classes already carries, so the bit
// is that index shifted rather than a second table to keep in step with it.
func classMaskOf(class dbc.DbcClass) int {
	return 1 << (class.ID - 1)
}

// The Go identifier a family is reached by: "Shadow Word: Pain" -> ShadowWordPain. A "(DND)" suffix is
// the client's do-not-display marker, not part of the name.
func fieldNameOf(spellName string) string {
	spellName = strings.TrimSuffix(spellName, " (DND)")
	var b strings.Builder
	upper := true
	for _, r := range spellName {
		switch {
		case r == '\'' || r == '\u2019':
			// Dropped without breaking the word, so "Avenger's Shield" is AvengersShield rather than
			// AvengerSShield.
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if upper {
				b.WriteRune(unicode.ToUpper(r))
				upper = false
			} else {
				b.WriteRune(r)
			}
		default:
			upper = true
		}
	}

	name := b.String()
	if name == "" || unicode.IsDigit(rune(name[0])) {
		return ""
	}
	return name
}

// Every family a class can learn, from two sources: every spell in one of the class's skill lines
// whose subtext reads "Rank N", "Shapeshift" or is empty (a single-rank ability like Whirlwind or Cat
// Form is its own rank 1), and every talent in the class's tree. Nothing hand-maintained.
func discoverLadders(db *sql.DB, class dbc.DbcClass, treeID int) ([]rankLadder, []string, []string, error) {
	mask := classMaskOf(class)

	exclusive, err := exclusiveSkillLines(db, mask)
	if err != nil {
		return nil, nil, nil, err
	}

	// The Defense skill line carries the one-class passives that open a counterattack window - Offensive
	// State (DND) fires the 5 s Overpower aura 1282733 on a melee hit, Defensive State (DND) the Revenge
	// one - beside Parry and Block, which have several classes' bits.
	rows, err := db.Query(`
		SELECT n.Name_lang, sla.Spell, s.NameSubtext_lang, sla.ClassMask, sla.SkillLine, sla.AcquireMethod,
		       COALESCE(lv.BaseLevel, 0), (COALESCE(json_extract(sm.Attributes, '$[0]'), 0) & ?) != 0
		FROM SkillLineAbility sla
		JOIN SpellName n ON n.ID = sla.Spell
		JOIN Spell s ON s.ID = sla.Spell
		LEFT JOIN SpellLevels lv ON lv.SpellID = sla.Spell AND lv.DifficultyID = 0
		LEFT JOIN SpellMisc sm ON sm.SpellID = sla.Spell AND sm.DifficultyID = 0
		WHERE (sla.SkillLine IN (
			SELECT DISTINCT sla2.SkillLine
			FROM SkillLineAbility sla2
			JOIN SkillLine sl2 ON sl2.ID = sla2.SkillLine AND sl2.CategoryID = 7
			WHERE (sla2.ClassMask & ?) != 0
		) OR (sla.SkillLine = ? AND sla.ClassMask = ? AND sla.AcquireMethod = ?))
		AND (s.NameSubtext_lang LIKE 'Rank %' OR s.NameSubtext_lang IN ('', 'Shapeshift'))
		AND sla.SkillLine NOT IN (2851, 2853)
		AND NOT EXISTS (SELECT 1 FROM SpellEffect se WHERE se.SpellID = sla.Spell AND se.EffectAura = ?)
		ORDER BY n.Name_lang, sla.Spell`, dbcenums.ATTR_PASSIVE, mask, dbc.SkillLineDefense, mask, acquireOnLevel, dbcenums.A_MOUNTED)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	treeSpells, err := treeSpellIDs(db, treeID)
	if err != nil {
		return nil, nil, nil, err
	}
	legacyTalents, err := legacyTalentNames(db)
	if err != nil {
		return nil, nil, nil, err
	}

	byName := map[string]map[int32][]rankCandidate{}
	// A spell with no subtext is a single-rank family of its own only where no "Rank N" row carries
	// its name - Execute's granted 20647 is a sub-spell of the ranked Execute, not a rank - and only
	// when a trainer teaches it (Death Wish, Sweeping Strikes and the stance passives all have a
	// trainer row) or a talent the tree could not describe shares its name. A grant-only one is a
	// Season of Discovery rune ability or an internal (Quick Strike, Meathook). Skill lines 2851 and
	// 2853 are the runes themselves, and the Mounted aura keeps the paladin Mounts line out, which
	// is a class line too.
	unranked := map[string][]rankCandidate{}
	trainedUnranked := map[string]bool{}
	for rows.Next() {
		var name, subtext string
		var c rankCandidate
		var baseLevel int32
		var passive bool
		if err := rows.Scan(&name, &c.SpellID, &subtext, &c.ClassMask, &c.SkillLine, &c.AcquireMethod, &baseLevel, &passive); err != nil {
			return nil, nil, nil, err
		}
		m := rankSubtext.FindStringSubmatch(subtext)
		if m == nil {
			c.Rank = 1
			unranked[name] = append(unranked[name], c)
			// A passive the legacy Talent table names, with a trainer row at level 1 and no place in
			// the Forever tree, is a talent that no longer exists (Emberstorm, Deadliness); Anger
			// Management has the same rows but sits in the tree.
			legacy := passive && baseLevel <= 1 && legacyTalents[name] && !treeSpells[c.SpellID]
			if c.AcquireMethod != acquireGranted && !legacy {
				trainedUnranked[name] = true
			}
			continue
		}
		rank, _ := strconv.Atoi(m[1])
		c.Rank = int32(rank)
		if byName[name] == nil {
			byName[name] = map[int32][]rankCandidate{}
		}
		byName[name][c.Rank] = append(byName[name][c.Rank], c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	traits, traitSkipped, traitPartial, err := discoverTraitLadders(db, treeID)
	if err != nil {
		return nil, nil, nil, err
	}
	for name, cands := range unranked {
		if byName[name] != nil {
			continue
		}
		if _, talent := traits[name]; talent {
			continue // the tree describes it; its own spell and legacy ranks would only collide
		}
		_, talentSkipped := traitSkipped[name]
		if trainedUnranked[name] || talentSkipped {
			byName[name] = map[int32][]rankCandidate{1: cands}
		}
	}

	seen := map[string]bool{}
	var names []string
	for _, source := range []map[string]bool{keysOf(byName), keysOf(traits), keysOf(traitSkipped)} {
		for name := range source {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)

	var ladders []rankLadder
	var skipped, partial []string
	taken := map[string]string{}

	for _, name := range names {
		field := fieldNameOf(name)
		if field == "" {
			skipped = append(skipped, fmt.Sprintf("%s: no usable Go identifier", name))
			continue
		}
		if owner, clash := taken[field]; clash {
			skipped = append(skipped, fmt.Sprintf("%s: identifier %s already taken by %s", name, field, owner))
			continue
		}

		var ladder map[int32]int32
		var points map[int32]map[int32]float64
		var resolveErr error
		if claimedByClass(byName[name], mask, exclusive) {
			ladder, resolveErr = resolveLadder(db, name, byName[name], mask)
		}

		// The tree wins where it states more ranks than the "Rank N" spells resolved to, because the
		// rank cap it carries is the one the talent proto and the UI trees are built from: Precision
		// kept its subtext on rank 1 alone, and leaving it as that one rank would let a 3-point
		// talent be read past its own ladder. Below that the "Rank N" spells win - they state a cost
		// and a cast time per rank, which the tree does not - and a family the resolver refused
		// stays refused, since a tree node that grants an ability does not describe its ranks.
		if t, ok := traits[name]; ok && resolveErr == nil && int(t.MaxRanks) > len(ladder) {
			ladder, points = map[int32]int32{}, t.Points
			for rank := int32(1); rank <= t.MaxRanks; rank++ {
				ladder[rank] = t.SpellID
			}
			if note := traitPartial[name]; note != "" {
				partial = append(partial, fmt.Sprintf("%s: %s", name, note))
			}
		}

		if resolveErr != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %s", name, resolveErr))
			continue
		}
		if len(ladder) == 0 {
			if reason := traitSkipped[name]; reason != "" {
				skipped = append(skipped, fmt.Sprintf("%s: %s", name, reason))
			}
			continue
		}

		taken[field] = name
		ladders = append(ladders, rankLadder{Name: name, Field: field, Ranks: ladder, Points: points})
	}

	return ladders, skipped, partial, nil
}

// A $<spellID><token> in a description: "$12880d" on Enrage, "$12976s1" on Last Stand.
var descriptionSpellRef = regexp.MustCompile(`\$(?:/\d+;)?(\d{4,7})[a-z]`)

// The spells a rank triggers or reads its tooltip from: an EffectTriggerSpell edge (Intercept's
// stun 20615, Intimidating Shout's fear 20511), a $<id> token (Enrage's buff 12880, Flurry's
// 12966, Last Stand's 12976) or an overrides.HandTriggers entry. In id order, without the rank itself and
// without ids the client has no spell for.
func triggeredSpells(db *sql.DB, spellID int32) ([]int32, error) {
	seen := map[int32]bool{}
	for _, id := range handTriggered(spellID) {
		seen[id] = true
	}
	rows, err := db.Query(`SELECT EffectTriggerSpell FROM SpellEffect WHERE SpellID = ? AND EffectTriggerSpell > 0`, spellID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		seen[id] = true
	}
	rows.Close()

	var desc string
	if err := scanOptional(db, `SELECT COALESCE(Description_lang, '') FROM Spell WHERE ID = ?`, spellID, &desc); err != nil {
		return nil, err
	}
	for _, m := range descriptionSpellRef.FindAllStringSubmatch(desc, -1) {
		id, _ := strconv.Atoi(m[1])
		seen[int32(id)] = true
	}
	delete(seen, spellID)

	var ids []int32
	for id := range seen {
		var name string
		if err := scanOptional(db, `SELECT Name_lang FROM SpellName WHERE ID = ?`, id, &name); err != nil {
			return nil, err
		}
		if name != "" {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// The rows of a family's triggered table: one per spell the family's ranks trigger. Where every rank
// triggers the same spell there is one row, rank 1; where each rank triggers its own, the row takes
// the rank's number; anything else is numbered in id order.
func triggeredRows(db *sql.DB, l rankLadder, mask int) ([]generatedRow, error) {
	ranks := make([]int32, 0, len(l.Ranks))
	for rank := range l.Ranks {
		ranks = append(ranks, rank)
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })

	perRank := map[int32][]int32{}
	distinct := map[int32]bool{}
	var order []int32
	for _, rank := range ranks {
		ids, err := triggeredSpells(db, l.Ranks[rank])
		if err != nil {
			return nil, err
		}
		perRank[rank] = ids
		for _, id := range ids {
			if !distinct[id] {
				distinct[id] = true
				order = append(order, id)
			}
		}
	}
	if len(order) == 0 {
		return nil, nil
	}

	numbered := map[int32]int32{}
	oneEach := true
	for _, rank := range ranks {
		if len(perRank[rank]) != 1 {
			oneEach = false
			break
		}
		numbered[perRank[rank][0]] = rank
	}
	switch {
	case len(order) == 1:
		numbered[order[0]] = 1
	case oneEach && len(numbered) == len(order):
	default:
		sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
		for i, id := range order {
			numbered[id] = int32(i + 1)
		}
	}

	var out []generatedRow
	for _, id := range order {
		row, err := buildRow(db, numbered[id], id, mask, nil)
		if err != nil {
			return nil, fmt.Errorf("triggered spell %d: %w", id, err)
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rank < out[j].Rank })
	return out, nil
}

// Every spell a node of the class's Forever tree defines.
func treeSpellIDs(db *sql.DB, treeID int) (map[int32]bool, error) {
	rows, err := db.Query(`
		SELECT DISTINCT COALESCE(d.SpellID, 0)
		FROM TraitNode tn
		JOIN TraitNodeXTraitNodeEntry x ON x.TraitNodeID = tn.ID
		JOIN TraitNodeEntry e ON e.ID = x.TraitNodeEntryID
		JOIN TraitDefinition d ON d.ID = e.TraitDefinitionID
		WHERE tn.TraitTreeID = ?`, treeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int32]bool{}
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id != 0 {
			out[id] = true
		}
	}
	return out, rows.Err()
}

// The names the client's legacy Talent table still carries, tree or no tree.
func legacyTalentNames(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT DISTINCT n.Name_lang FROM Talent t JOIN SpellName n ON n.ID = t.SpellRank_0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func keysOf[V any](m map[string]V) map[string]bool {
	keys := make(map[string]bool, len(m))
	for k := range m {
		keys[k] = true
	}
	return keys
}

// Every talent in one class's tree, by name, plus the ones that had to be left out and the ones that
// came out short an effect. The tree states one spell per talent and a curve per effect, where the
// curve reads rank -> value, so a rank's numbers are the curve's and never the spell's own: those
// hold the top rank on some talents and a stale number on others, and 240 of the 640 curves in this
// build disagree with them.
func discoverTraitLadders(db *sql.DB, treeID int) (map[string]traitLadder, map[string]string, map[string]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT d.ID, COALESCE(d.SpellID, 0), e.MaxRanks,
		       COALESCE(NULLIF(sn.Name_lang, ''), d.OverrideName_lang, '')
		FROM TraitNode tn
		JOIN TraitNodeXTraitNodeEntry x ON x.TraitNodeID = tn.ID
		JOIN TraitNodeEntry e ON e.ID = x.TraitNodeEntryID
		JOIN TraitDefinition d ON d.ID = e.TraitDefinitionID
		LEFT JOIN SpellName sn ON sn.ID = d.SpellID
		WHERE tn.TraitTreeID = ?
		ORDER BY d.ID`, treeID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	type traitDef struct {
		ID       int32
		SpellID  int32
		MaxRanks int32
		Name     string
	}
	var defs []traitDef
	for rows.Next() {
		var d traitDef
		if err := rows.Scan(&d.ID, &d.SpellID, &d.MaxRanks, &d.Name); err != nil {
			return nil, nil, nil, err
		}
		defs = append(defs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	ladders := map[string]traitLadder{}
	skipped := map[string]string{}
	partial := map[string]string{}
	spellOf := map[string]int32{}

	for _, d := range defs {
		if d.Name == "" || d.SpellID == 0 {
			continue
		}
		// A node appears once per entry, and the client keeps retired twins around, so a second
		// definition on the same spell is the same talent and a second one on another spell is two
		// talents the generated file could only reach by one name.
		if first, ok := spellOf[d.Name]; ok {
			if first != d.SpellID {
				delete(ladders, d.Name)
				skipped[d.Name] = fmt.Sprintf(
					"two talent tree nodes carry the name, on spells %d and %d", first, d.SpellID)
			}
			continue
		}
		spellOf[d.Name] = d.SpellID

		// A one-rank node usually grants an ability rather than describing a ladder - Hemorrhage and
		// Water Shield are nodes on a spell the game teaches - and the one row it would yield names
		// the node's spell, which is not the spell the ability's own ranks are keyed on. A one-rank
		// node on a passive nothing teaches is the talent itself - Raging Blows, Vanguard - and
		// yields its one row at the spell's base points.
		oneRankPassive := false
		if d.MaxRanks <= 1 {
			passive, err := SpellIsPassive(db, d.SpellID)
			if err != nil {
				return nil, nil, nil, err
			}
			if !passive {
				continue
			}
			taught, err := taughtBySkillLine(db, d.SpellID)
			if err != nil {
				return nil, nil, nil, err
			}
			if taught {
				continue
			}
			oneRankPassive = true
		}

		effects, err := effectIndicesOf(db, d.SpellID)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(effects) == 0 {
			skipped[d.Name] = fmt.Sprintf("the client states no effects for spell %d", d.SpellID)
			continue
		}

		curves, err := traitCurves(db, d.ID)
		if err != nil {
			return nil, nil, nil, err
		}

		points := map[int32]map[int32]float64{}
		var uncovered []string
		for _, index := range effects {
			values, ok := curveRanks(curves[index], d.MaxRanks)
			if !ok {
				uncovered = append(uncovered, strconv.Itoa(int(index)))
				continue
			}
			for rank, value := range values {
				if points[rank] == nil {
					points[rank] = map[int32]float64{}
				}
				points[rank][index] = value
			}
		}

		if len(points) == 0 {
			if !oneRankPassive {
				skipped[d.Name] = fmt.Sprintf("the talent tree states no per-rank value for spell %d", d.SpellID)
				continue
			}
			points[1] = map[int32]float64{}
		}
		if len(uncovered) > 0 {
			partial[d.Name] = fmt.Sprintf("effect %s of spell %d has no rank curve and is held at its base points",
				strings.Join(uncovered, ", "), d.SpellID)
		}
		ladders[d.Name] = traitLadder{SpellID: d.SpellID, MaxRanks: d.MaxRanks, Points: points}
	}

	return ladders, skipped, partial, nil
}

// Whether a trainer, the skill itself or a level grants the spell (AcquireMethod 0, 1 or 2).
func taughtBySkillLine(db *sql.DB, spellID int32) (bool, error) {
	var taught bool
	err := scanOptional(db, fmt.Sprintf(`SELECT COUNT(*) > 0 FROM SkillLineAbility WHERE Spell = ? AND AcquireMethod != %d`, acquireGranted), spellID, &taught)
	return taught, err
}

func effectIndicesOf(db *sql.DB, spellID int32) ([]int32, error) {
	rows, err := db.Query(`SELECT EffectIndex FROM SpellEffect WHERE SpellID = ? ORDER BY EffectIndex`, spellID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indices []int32
	for rows.Next() {
		var index int32
		if err := rows.Scan(&index); err != nil {
			return nil, err
		}
		indices = append(indices, index)
	}
	return indices, rows.Err()
}

// Each curve as rank -> value. OperationType is 0 on all 640 rows in this build, meaning the point is
// the value the rank is set to; anything else would modify the spell's own number instead, and is
// dropped rather than guessed at - the effect then reads as having no curve.
func traitCurves(db *sql.DB, defID int32) (map[int32]map[int32]float64, error) {
	rows, err := db.Query(`
		SELECT p.EffectIndex, p.OperationType, cp.Pos_0, cp.Pos_1
		FROM TraitDefinitionEffectPoints p
		JOIN CurvePoint cp ON cp.CurveID = p.CurveID
		WHERE p.TraitDefinitionID = ?
		ORDER BY p.EffectIndex, cp.Pos_0`, defID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	curves := map[int32]map[int32]float64{}
	for rows.Next() {
		var index, operation int32
		var at, value float64
		if err := rows.Scan(&index, &operation, &at, &value); err != nil {
			return nil, err
		}
		rank := int32(at)
		if operation != 0 || float64(rank) != at {
			continue
		}
		if curves[index] == nil {
			curves[index] = map[int32]float64{}
		}
		curves[index][rank] = value
	}
	return curves, rows.Err()
}

func curveRanks(curve map[int32]float64, maxRanks int32) (map[int32]float64, bool) {
	values := make(map[int32]float64, maxRanks)
	for rank := int32(1); rank <= maxRanks; rank++ {
		value, ok := curve[rank]
		if !ok {
			return nil, false
		}
		values[rank] = value
	}
	return values, true
}

// The skill lines only this class appears in. 11 are shared: "Holy" carries both the paladin and the
// priest bit, which is how paladin Holy Shock reached the priest file. Inside an exclusive line a
// ClassMask-0 spell must be this class's, which is what makes Ignite attributable.
func exclusiveSkillLines(db *sql.DB, mask int) (map[int32]bool, error) {
	rows, err := db.Query(`
		SELECT SkillLine, group_concat(DISTINCT ClassMask)
		FROM SkillLineAbility WHERE ClassMask != 0 GROUP BY SkillLine`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := map[int32]bool{}
	for rows.Next() {
		var line int32
		var masks string
		if err := rows.Scan(&line, &masks); err != nil {
			return nil, err
		}
		union := 0
		for _, part := range strings.Split(masks, ",") {
			m, err := strconv.Atoi(part)
			if err != nil {
				continue
			}
			union |= m
		}
		if union == mask {
			lines[line] = true
		}
	}
	return lines, rows.Err()
}

// A family is this class's if one of its ranks carries the class bit, or if its ranks sit in a skill
// line no other class appears in.
func claimedByClass(byRank map[int32][]rankCandidate, mask int, exclusive map[int32]bool) bool {
	for _, cands := range byRank {
		for _, c := range cands {
			if c.ClassMask&mask != 0 || exclusive[c.SkillLine] {
				return true
			}
		}
	}
	return false
}

// Picks one spell per rank. Lightning Bolt's Elemental Overload twins (45284-45293) share the name,
// skill line and class set, differing only by ClassMask 0. So the class bit wins where it exists, and
// a ClassMask-0 candidate is taken only when no real one was found - that is how Holy Shield 1-3
// resolve.
// Where no rule can separate two candidates and none should try: Seal of Righteousness rank 1 is
// 20154 and 21084, same name, rank, effect shape and 20 mana. The sim has always used 21084.
var ladderPins = map[string]map[int32]int32{
	"Seal of Righteousness": {1: 21084},
}

// Forever re-issues a classic ability as a second spell in the 4xxxxx range, same name, same rank
// subtext, same class: Exorcism rank 1 is both 879 and 415068, and Raptor Strike rank 1 is 2973,
// 409691 and 415335. The copies are granted (AcquireMethod 3) where the original is trained (0) or
// learned (2), so the lowest method is the ability the player actually casts. Narrowing is relative
// and never empties the set, which leaves the 44 accepted ladder spells that are only ever granted
// exactly where they were.
func lowestAcquireMethod(cands []rankCandidate) []rankCandidate {
	perSpell := map[int32]int32{}
	for _, c := range cands {
		if best, seen := perSpell[c.SpellID]; !seen || c.AcquireMethod < best {
			perSpell[c.SpellID] = c.AcquireMethod
		}
	}

	best := int32(math.MaxInt32)
	for _, method := range perSpell {
		if method < best {
			best = method
		}
	}

	var kept []rankCandidate
	for _, c := range cands {
		if perSpell[c.SpellID] == best {
			kept = append(kept, c)
		}
	}
	return kept
}

func resolveLadder(db *sql.DB, name string, byRank map[int32][]rankCandidate, mask int) (map[int32]int32, error) {
	ladder := map[int32]int32{}

	// Sorted, so that when several ranks are ambiguous the one named in the skipped list is always the
	// lowest rather than whichever the map handed over first.
	rankNums := make([]int32, 0, len(byRank))
	for rank := range byRank {
		rankNums = append(rankNums, rank)
	}
	sort.Slice(rankNums, func(i, j int) bool { return rankNums[i] < rankNums[j] })

	var ambiguous []string
	for _, rank := range rankNums {
		cands := byRank[rank]
		var chosen []rankCandidate
		for _, c := range cands {
			if c.ClassMask&mask != 0 {
				chosen = append(chosen, c)
			}
		}
		if len(chosen) == 0 {
			chosen = cands
		}
		chosen = lowestAcquireMethod(chosen)

		ids := map[int32]bool{}
		for _, c := range chosen {
			ids[c.SpellID] = true
		}

		if len(ids) == 1 {
			ladder[rank] = chosen[0].SpellID
			continue
		}

		// Exorcism is one name over two full ladders: the trainer ranks 879-10314, and 415068-415073,
		// which the Season of Discovery passive Exorcist (415076) swaps onto the action bar so the
		// spell can hit any target. The stand-in copies the rank's name, subtext, class bit and mana,
		// so nothing above separates them, but the override aura names both sides. The overridden
		// spell is the one a trainer teaches and the one the ladder wants; the stand-in is only ever
		// reachable through its owner. Fire Blast, Drain Life, Renew and Raptor Strike carry the same
		// shape under Overheat, Master Channeler, Empowered Renew and Melee Specialist.
		replaced, err := overrideReplacements(db, ids)
		if err != nil {
			return nil, err
		}
		if len(replaced) > 0 && len(replaced) < len(ids) {
			for id := range replaced {
				delete(ids, id)
			}
			if len(ids) == 1 {
				for id := range ids {
					ladder[rank] = id
				}
				continue
			}
		}

		// Holy Shock is one name over three spells per rank: a dummy the player casts, plus a damage
		// and a heal spell the client never exposes. Only the castable one carries a SpellPower row,
		// and it is the one the sim registers, so that is the tie-break.
		castable, err := castableOf(db, ids)
		if err != nil {
			return nil, err
		}
		if castable == 0 {
			// Judgement of Command is the same dispatcher shape with no mana on either half - the
			// parent Judgement pays - so SpellPower cannot separate them. One E_DUMMY against one
			// E_SCHOOL_DAMAGE can: the dummy is what the seal triggers and what the sim registers,
			// and the damage spell rides along as a sibling, exactly as Holy Shock does.
			castable, err = dispatcherOf(db, ids)
			if err != nil {
				return nil, err
			}
		}
		if pinned, ok := ladderPins[name][rank]; ok && ids[pinned] {
			ladder[rank] = pinned
			continue
		}
		if castable == 0 {
			// Every ambiguous rank is collected rather than returning on the first, so the skipped
			// list in the class file names all of them. Judgement of Command is ambiguous on all six.
			var list []string
			for id := range ids {
				list = append(list, strconv.Itoa(int(id)))
			}
			sort.Strings(list)
			ambiguous = append(ambiguous, fmt.Sprintf("rank %d between spells %s", rank, strings.Join(list, ", ")))
			continue
		}
		ladder[rank] = castable
	}
	if len(ambiguous) > 0 {
		return nil, fmt.Errorf("ambiguous %s", strings.Join(ambiguous, "; "))
	}

	maxRank := int32(0)
	for rank := range ladder {
		if rank > maxRank {
			maxRank = rank
		}
	}
	for rank := int32(1); rank <= maxRank; rank++ {
		if _, ok := ladder[rank]; !ok {
			return nil, fmt.Errorf("missing rank %d of %d", rank, maxRank)
		}
	}
	return ladder, nil
}

// The one spell among these whose first effect is a dummy, when exactly one other is a direct
// damage effect. That pairing is the client's dispatcher shape: the dummy is the spell the game
// grants and triggers, the damage spell is never exposed. Anything else returns 0 rather than
// guessing - two talent auras that merely differ somewhere are not a dispatcher.
func dispatcherOf(db *sql.DB, ids map[int32]bool) (int32, error) {
	var dummy, damage int32
	for id := range ids {
		var effect dbc.SpellEffectType
		err := db.QueryRow(
			`SELECT Effect FROM SpellEffect WHERE SpellID = ? ORDER BY EffectIndex LIMIT 1`, id).Scan(&effect)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		switch effect {
		case dbcenums.E_DUMMY:
			if dummy != 0 {
				return 0, nil
			}
			dummy = id
		case dbcenums.E_SCHOOL_DAMAGE:
			if damage != 0 {
				return 0, nil
			}
			damage = id
		default:
			return 0, nil
		}
	}
	if dummy == 0 || damage == 0 {
		return 0, nil
	}
	return dummy, nil
}

// The one spell among these that has a mana cost, or 0 when that does not single one out.
func castableOf(db *sql.DB, ids map[int32]bool) (int32, error) {
	var found int32
	for id := range ids {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM SpellPower WHERE SpellID = ?`, id).Scan(&n); err != nil {
			return 0, err
		}
		if n == 0 {
			continue
		}
		if found != 0 {
			return 0, nil
		}
		found = id
	}
	return found, nil
}

// The candidates that another candidate is swapped for by an A_OVERRIDE_ACTIONBAR_SPELLS aura. On
// that aura the misc value is the spell being overridden and the base points is its replacement; a
// candidate is dropped only when the spell it stands in for is itself in the running, so a stand-in
// whose base rank was never a candidate is left alone.
func overrideReplacements(db *sql.DB, ids map[int32]bool) (map[int32]bool, error) {
	replaced := map[int32]bool{}
	for id := range ids {
		rows, err := db.Query(`
			SELECT CAST(EffectBasePointsF AS INTEGER)
			FROM SpellEffect
			WHERE EffectAura = ? AND EffectMiscValue_0 = ?`, int(dbcenums.A_OVERRIDE_ACTIONBAR_SPELLS), id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var replacement int32
			if err := rows.Scan(&replacement); err != nil {
				rows.Close()
				return nil, err
			}
			if ids[replacement] {
				replaced[replacement] = true
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return replaced, nil
}

// Which effect supplies which field. Derived from the effect types rather than declared per family,
// because the client data already says it: a SCHOOL_DAMAGE effect is direct damage, a HEAL effect is a
// heal, an ENERGIZE effect is Lay on Hands' mana restore, a periodic aura is a tick.
// Where points is set the rank's numbers come from the talent tree's curves. Only the effects it prices
// can fill a role; an effect the tree states no curve for is the same at every rank - Blood Craze's
// 20% health threshold - and is carried in Effects at the spell's own base points. The same-name
// sibling search is not wanted there either - it exists for the ability dispatchers, and a talent
// priced per rank has nothing to borrow.
func buildRow(db *sql.DB, rank int32, spellID int32, mask int, points map[int32]float64) (generatedRow, error) {
	var spell RankSpell
	var candidates []RankEffect
	var err error
	if points != nil {
		spell, err = LoadRankSpell(db, spellID)
		candidates = spell.Effects
	} else {
		spell, candidates, err = RankCandidates(db, spellID, mask)
	}
	if err != nil {
		return generatedRow{}, err
	}
	if points != nil {
		candidates = pricedEffects(spell.Effects, points)
	}

	derive := func(e RankEffect) (float64, float64) {
		if value, ok := points[e.Index]; ok {
			return value, value
		}
		return DeriveRankAmount(e, e.SpellLevel, e.MaxLevel)
	}

	row := generatedRow{
		Rank: rank, SpellID: spellID,
		CastTimeMs: spell.CastTimeMs, GCDMs: spell.GCDMs, CooldownMs: spell.CooldownMs,
		MinRange: spell.MinRange, MaxRange: spell.MaxRange, MissileSpeed: spell.MissileSpeed,
		ProcChance: spell.ProcChance, ProcCharges: spell.ProcCharges, MaxTargets: spell.MaxTargets, RefundsOnMiss: spell.RefundsOnMiss,
		PeriodicCanCrit: spell.PeriodicCanCrit,
		DurationMs:      spell.DurationMs, SchoolMask: spell.SchoolMask, DefenseType: spell.DefenseType,
	}
	if spell.ManaCost.Valid {
		row.Cost = NormalizePowerCost(int32(spell.ManaCost.Int64), spell.PowerType)
	}
	row.PowerCostPct = spell.PowerCostPct

	amountOf := func(e RankEffect) *generatedAmount {
		min, max := derive(e)
		a := &generatedAmount{Min: min, Max: max, Coef: e.Coefficient, APCoef: e.APCoef}
		if e.OwnerSpellID != spell.SpellID {
			a.SpellID = e.OwnerSpellID
		}

		// A tick count is the duration over the period, which is how every hand-written
		// NumberOfTicks in the sim was arrived at.
		if e.AuraPeriod > 0 {
			a.PeriodMs = e.AuraPeriod
			if spell.DurationMs > 0 {
				a.Ticks = spell.DurationMs / e.AuraPeriod
			}
		}
		return a
	}

	for _, e := range spell.Effects {
		min, max := derive(e)
		if max == min {
			max = 0
		}
		row.Effects = append(row.Effects, generatedEffect{
			Index: e.Index, Effect: e.Effect, Aura: e.Aura, Misc: e.MiscValue, Value: min, ValueMax: max,
			ChainAmplitude: e.ChainAmplitude, Coef: e.Coefficient, APCoef: e.APCoef,
		})
	}

	for _, e := range candidates {
		switch {
		// A damage effect with a period is one the description reached and the rank's periodic
		// dummy times, so it ticks; the rank's own damage effects never carry one. Consecration
		// names a second, the extra damage on the first few targets.
		case e.Effect == dbcenums.E_SCHOOL_DAMAGE && e.AuraPeriod > 0:
			switch {
			case row.Periodic == nil:
				row.Periodic = amountOf(e)
			case row.SecondaryPeriodic == nil:
				row.SecondaryPeriodic = amountOf(e)
			default:
				return generatedRow{}, fmt.Errorf("spell %d names a third ticking value on spell %d, and the row holds two",
					spellID, e.OwnerSpellID)
			}
		// A dummy the description names on a spell the rank's own dummy points at is the rank's number
		// kept there with its coefficient: Seal of Righteousness' per-hit damage, on its judgement.
		case e.Effect == dbcenums.E_DUMMY && e.Named && row.Direct == nil:
			row.Direct = amountOf(e)
		case (e.Effect == dbcenums.E_SCHOOL_DAMAGE || IsWeaponDamageEffect(e.Effect)) && row.Direct == nil:
			row.Direct = amountOf(e)
		case e.Effect == dbcenums.E_HEAL && row.Heal == nil:
			row.Heal = amountOf(e)
		case (e.Effect == dbcenums.E_ENERGIZE || e.Aura == dbcenums.A_PERIODIC_ENERGIZE) && row.Energize == nil:
			row.Energize = amountOf(e)
		case IsThreatEffect(e.Effect) && row.FlatThreatBonus == 0:
			min, _ := derive(e)
			row.FlatThreatBonus = min
		case IsPeriodicAura(e.Aura) && row.Periodic == nil:
			row.Periodic = amountOf(e)
		}
	}

	// An aura effect reached by the fallbacks below still lands in the role its shape says it has:
	// Frenzied Regeneration's aura ticks every second, and filing that under Direct would hand the
	// call site a periodic value through a field that promises a direct one.
	fallback := func(e RankEffect) {
		if e.AuraPeriod > 0 {
			row.Periodic = amountOf(e)
		} else {
			row.Direct = amountOf(e)
		}
	}

	// Holy Shield keeps its per-block damage on an aura effect that is none of the roles above, and it
	// is not the only aura effect on the spell: one index holds the block value and another the damage.
	// The damage is the one that scales with spell power, so a nonzero coefficient is what picks it.
	if !row.hasValue() {
		for _, e := range candidates {
			if e.Aura != 0 && e.BasePoints > 0 && e.Coefficient > 0 {
				fallback(e)
				break
			}
		}
	}
	if !row.hasValue() {
		for _, e := range candidates {
			if e.Aura != 0 && e.BasePoints > 0 {
				fallback(e)
				break
			}
		}
	}

	return row, nil
}

func pricedEffects(effects []RankEffect, points map[int32]float64) []RankEffect {
	var kept []RankEffect
	for _, e := range effects {
		if _, ok := points[e.Index]; ok {
			kept = append(kept, e)
		}
	}
	return kept
}

// A low rank can legitimately carry no numbers at all - Lay on Hands rank 1 heals a share of max health
// and restores no mana, so it has no ENERGIZE effect where ranks 2-4 do.
func (row generatedRow) hasValue() bool {
	return row.Direct != nil || row.Heal != nil || row.Periodic != nil || row.Energize != nil
}

// Every file the generator writes, by the path it is written to, rendered and none written: what
// happens to them is writeSpellDataFiles' business, and -check's business is that nothing does.
func renderSpellDataFiles(helper *DBHelper) (map[string][]byte, *storeInputs, error) {
	if err := RequireSpellCastTimes(helper); err != nil {
		return nil, nil, err
	}

	// Rendered in full before anything is written, so a class that fails validation cannot leave half
	// the packages regenerated and half stale.
	namer := newRankEnumNamer()

	// One tree per class, picked the same way the talent protos pick it, so the rank caps here and the
	// ones the sim's Talents message carries are the same numbers.
	trees, err := selectTraitTrees(helper)
	if err != nil {
		return nil, nil, err
	}

	rendered := map[string][]byte{}

	// The store carries every spell the class files are built from, so the discovery runs here and
	// both consumers read the one result rather than each rediscovering the ladders.
	var ladderIDs []int32
	for _, class := range dbc.Classes {
		pkg := strings.ToLower(dbc.ClassNameFromDBC(class))
		tree, ok := trees[classMaskOf(class)]
		if !ok {
			return nil, nil, fmt.Errorf("%s: the client database holds no talent tree for this class", pkg)
		}
		ladders, skipped, partial, err := discoverLadders(helper.db, class, tree)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", pkg, err)
		}
		out, err := renderClassFile(helper.db, pkg, class, namer, ladders, skipped, partial)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", pkg, err)
		}
		rendered[pkg] = out

		for _, l := range ladders {
			for _, id := range l.Ranks {
				ladderIDs = append(ladderIDs, id)
			}
		}

		// A node the ladders do not describe still grants a spell the sim registers - Hemorrhage is
		// one - so every spell the tree defines is a root of the store.
		treeSpells, err := treeSpellIDs(helper.db, tree)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", pkg, err)
		}
		for id := range treeSpells {
			if id != 0 {
				ladderIDs = append(ladderIDs, id)
			}
		}
	}

	// Rendered before any write too: it holds exactly the names the class files above turned out to
	// reference, and a class file naming a constant this file does not declare breaks the sim - and
	// with it gen_db, which imports the sim.
	enums, err := namer.render()
	if err != nil {
		return nil, nil, err
	}

	// The store's rows name the enum values through dbcenums, so nothing they reach is recorded on
	// the namer: the shared file above holds the class tables' names and no others.
	//
	// The client rows it is built from are captured on the way through and handed back, so the
	// caller can write them next to the store and the regeneration can be checked without the
	// database - see spelldata_inputs.go.
	inputs, err := loadStoreInputs(helper.db, ladderIDs, trees)
	if err != nil {
		return nil, nil, err
	}
	store, err := renderStore(inputs, namer)
	if err != nil {
		return nil, nil, err
	}

	formsFile, err := renderFormsFile(inputs.Forms)
	if err != nil {
		return nil, nil, err
	}

	// The buffs read the store's rows, so they are rendered from the same inputs, and checked and
	// type-checked with the rest.
	files, err := renderBuffOutputs(inputs)
	if err != nil {
		return nil, nil, fmt.Errorf("buffs: %w", err)
	}
	files["sim/core/dbcenums/forms_auto_gen.go"] = formsFile
	files["sim/common/shared/spell_data_enums_auto_gen.go"] = enums
	files["sim/core/spelldata/spells_auto_gen.go"] = store
	for pkg, out := range rendered {
		files[fmt.Sprintf("sim/%s/spell_data_auto_gen.go", pkg)] = out
	}
	return files, inputs, nil
}

func renderClassFile(db *sql.DB, pkg string, class dbc.DbcClass, namer *rankEnumNamer, ladders []rankLadder, skipped, partial []string) ([]byte, error) {
	// Named rather than dropped silently, so a family the resolver could not make sense of is visible
	// here instead of merely absent. Kept out of the body below, whose text decides which imports the
	// file needs - a family name containing "time." would otherwise add an unused one.
	var notGenerated strings.Builder
	for _, block := range []struct {
		header string
		lines  []string
	}{
		{"// Not generated:\n", skipped},
		{"// Generated with an effect held at its base points, the talent tree stating no rank value for it:\n", partial},
	} {
		if len(block.lines) == 0 {
			continue
		}
		notGenerated.WriteString(block.header)
		for _, s := range block.lines {
			fmt.Fprintf(&notGenerated, "//   %s\n", s)
		}
		notGenerated.WriteString("\n")
	}

	mask := classMaskOf(class)
	triggered := map[string][]generatedRow{}
	for _, l := range ladders {
		rows, err := triggeredRows(db, l, mask)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", l.Name, err)
		}
		if len(rows) > 0 {
			triggered[l.Field] = rows
		}
	}

	if storeBackedClasses[pkg] {
		return renderLadderClassFile(pkg, ladders, triggered, notGenerated.String())
	}

	var b strings.Builder
	b.WriteString("type generatedSpellData struct {\n")
	for _, l := range ladders {
		fmt.Fprintf(&b, "\t%s shared.SpellDataTable\n", l.Field)
		if triggered[l.Field] != nil {
			fmt.Fprintf(&b, "\t%sTriggered shared.SpellDataTable\n", l.Field)
		}
	}
	b.WriteString("}\n\nvar spellData = generatedSpellData{\n")
	for _, l := range ladders {
		ranks := make([]int32, 0, len(l.Ranks))
		for rank := range l.Ranks {
			ranks = append(ranks, rank)
		}
		sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })

		fmt.Fprintf(&b, "\t%s: shared.SpellDataTable{\n", l.Field)
		for _, rank := range ranks {
			row, err := buildRow(db, rank, l.Ranks[rank], mask, l.Points[rank])
			if err != nil {
				return nil, fmt.Errorf("%s rank %d: %w", l.Name, rank, err)
			}
			fmt.Fprintf(&b, "\t\t%s\n", formatRow(row, namer))
		}
		b.WriteString("\t},\n")
		if rows := triggered[l.Field]; rows != nil {
			fmt.Fprintf(&b, "\t%sTriggered: shared.SpellDataTable{\n", l.Field)
			for _, row := range rows {
				fmt.Fprintf(&b, "\t\t%s\n", formatRow(row, namer))
			}
			b.WriteString("\t},\n")
		}
	}
	b.WriteString("}\n")

	body := b.String()

	// Imports are gated on the body actually using them: an unused import does not compile, and this
	// file is one the generator itself needs in order to run again.
	var head strings.Builder
	fmt.Fprintf(&head, "// Code generated by tools/database/gen_spelldata. DO NOT EDIT.\n\n")
	fmt.Fprintf(&head, "package %s\n\n", pkg)
	var std, mod []string
	if strings.Contains(body, "time.") {
		std = append(std, `"time"`)
	}
	mod = append(mod, `"github.com/wowsims/forever/sim/common/shared"`)
	if strings.Contains(body, "core.") {
		mod = append(mod, `"github.com/wowsims/forever/sim/core"`)
	}
	head.WriteString("import (\n")
	for _, i := range std {
		fmt.Fprintf(&head, "\t%s\n", i)
	}
	if len(std) > 0 {
		head.WriteString("\n")
	}
	for _, i := range mod {
		fmt.Fprintf(&head, "\t%s\n", i)
	}
	head.WriteString(")\n\n")

	out, err := format.Source([]byte(head.String() + notGenerated.String() + body))
	if err != nil {
		return nil, fmt.Errorf("generated %s file does not parse, refusing to write it: %w", pkg, err)
	}
	return out, nil
}

// The classes whose file names the store's ladders instead of restating the client's rows. Their
// numbers come out of sim/core/spelldata, so the file holds one line per family and no data at all.
var storeBackedClasses = map[string]bool{
	"warrior": true,
	"rogue":   true,
	"warlock": true,
	"mage":    true,
	"druid":   true,
	"priest":  true,
	"shaman":  true,
	"hunter":  true,
}

// A class file as references into the store: the same struct, the same field names, and a ladder per
// family in place of the rows.
//
// A family the talent tree prices is one spell whose per-rank numbers live in a curve, which is what
// Talent reads; every other family is one spell per rank, which is Ranked. Both index by position,
// so a ladder whose ranks are not 1..n would silently misnumber and is refused instead.
func renderLadderClassFile(pkg string, ladders []rankLadder, triggered map[string][]generatedRow, notGenerated string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("type generatedSpellData struct {\n")
	for _, l := range ladders {
		fmt.Fprintf(&b, "\t%s spelldata.Ladder\n", l.Field)
		if triggered[l.Field] != nil {
			fmt.Fprintf(&b, "\t%sTriggered spelldata.Ladder\n", l.Field)
		}
	}
	b.WriteString("}\n\nvar spellData = generatedSpellData{\n")

	for _, l := range ladders {
		ranks := make([]int32, 0, len(l.Ranks))
		for rank := range l.Ranks {
			ranks = append(ranks, rank)
		}
		sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })

		ids := make([]int32, 0, len(ranks))
		for i, rank := range ranks {
			if rank != int32(i+1) {
				return nil, fmt.Errorf("%s: rank %d where rank %d was expected, and a ladder indexes by position",
					l.Name, rank, i+1)
			}
			ids = append(ids, l.Ranks[rank])
		}

		if l.Points != nil {
			fmt.Fprintf(&b, "\t%s: spelldata.Talent(%d, %d),\n", l.Field, ids[0], len(ids))
		} else {
			fmt.Fprintf(&b, "\t%s: spelldata.Ranked(%s),\n", l.Field, joinIDs(ids))
		}

		rows := triggered[l.Field]
		if rows == nil {
			continue
		}
		triggeredIDs := make([]int32, 0, len(rows))
		for i, row := range rows {
			if row.Rank != int32(i+1) {
				return nil, fmt.Errorf("%s triggered: rank %d where rank %d was expected, and a ladder indexes by position",
					l.Name, row.Rank, i+1)
			}
			triggeredIDs = append(triggeredIDs, row.SpellID)
		}
		fmt.Fprintf(&b, "\t%sTriggered: spelldata.Ranked(%s),\n", l.Field, joinIDs(triggeredIDs))
	}
	b.WriteString("}\n")

	var head strings.Builder
	fmt.Fprintf(&head, "// Code generated by tools/database/gen_spelldata. DO NOT EDIT.\n\n")
	fmt.Fprintf(&head, "package %s\n\n", pkg)
	head.WriteString("import (\n\t\"github.com/wowsims/forever/sim/core/spelldata\"\n)\n\n")

	out, err := format.Source([]byte(head.String() + notGenerated + b.String()))
	if err != nil {
		return nil, fmt.Errorf("generated %s file does not parse, refusing to write it: %w", pkg, err)
	}
	return out, nil
}

func joinIDs(ids []int32) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(int(id))
	}
	return strings.Join(parts, ", ")
}

// core's SpellSchool bits are the client's, so this is a rendering and not a translation: the name
// emitted for a bit holds that same bit. TestSpellSchoolBitsMatchTheClient asserts the pairing.
var schoolBitNames = map[int32]string{
	1:  "core.SpellSchoolPhysical",
	2:  "core.SpellSchoolHoly",
	4:  "core.SpellSchoolFire",
	8:  "core.SpellSchoolNature",
	16: "core.SpellSchoolFrost",
	32: "core.SpellSchoolShadow",
	64: "core.SpellSchoolArcane",
}

// Nine spells in the build carry two schools - Frostfire and the Nature/Shadow plagues - and the OR of
// the two names is the value core's own named combination holds, so they are spelled out rather than
// matched against a list that would need keeping in step.
func schoolName(mask int32) string {
	var names []string
	for bit := int32(1); bit <= 64; bit <<= 1 {
		if mask&bit != 0 {
			name, ok := schoolBitNames[bit]
			if !ok {
				return ""
			}
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, " | ")
}

func defenseTypeName(t int32) string {
	switch t {
	case 1:
		return "core.DefenseTypeMagic"
	case 2:
		return "core.DefenseTypeMelee"
	case 3:
		return "core.DefenseTypeRanged"
	}
	return ""
}

func formatRow(row generatedRow, namer *rankEnumNamer) string {
	parts := []string{fmt.Sprintf("Rank: %d", row.Rank), fmt.Sprintf("SpellID: %d", row.SpellID)}
	if row.Cost > 0 {
		parts = append(parts, fmt.Sprintf("Cost: %d", row.Cost))
	}
	if row.PowerCostPct != 0 {
		parts = append(parts, fmt.Sprintf("PowerCostPct: %s", num(row.PowerCostPct)))
	}
	if row.CastTimeMs > 0 {
		parts = append(parts, fmt.Sprintf("CastTime: %s", millis(row.CastTimeMs)))
	}
	if row.GCDMs > 0 {
		parts = append(parts, fmt.Sprintf("GCD: %s", millis(row.GCDMs)))
	}
	if row.CooldownMs > 0 {
		parts = append(parts, fmt.Sprintf("Cooldown: %s", millis(row.CooldownMs)))
	}
	if row.DurationMs > 0 {
		parts = append(parts, fmt.Sprintf("Duration: %s", millis(row.DurationMs)))
	}
	if row.MinRange > 0 {
		parts = append(parts, fmt.Sprintf("MinRange: %s", num(row.MinRange)))
	}
	if row.MaxRange > 0 {
		parts = append(parts, fmt.Sprintf("MaxRange: %s", num(row.MaxRange)))
	}
	if row.MissileSpeed > 0 {
		parts = append(parts, fmt.Sprintf("MissileSpeed: %s", num(row.MissileSpeed)))
	}
	if row.ProcChance > 0 {
		parts = append(parts, fmt.Sprintf("ProcChance: %d", row.ProcChance))
	}
	if row.ProcCharges > 0 {
		parts = append(parts, fmt.Sprintf("ProcCharges: %d", row.ProcCharges))
	}
	if row.MaxTargets > 0 {
		parts = append(parts, fmt.Sprintf("MaxTargets: %d", row.MaxTargets))
	}
	if row.RefundsOnMiss {
		parts = append(parts, "RefundsOnMiss: true")
	}
	if row.PeriodicCanCrit {
		parts = append(parts, "PeriodicCanCrit: true")
	}
	if row.FlatThreatBonus != 0 {
		parts = append(parts, fmt.Sprintf("FlatThreatBonus: %s", num(row.FlatThreatBonus)))
	}
	if name := schoolName(row.SchoolMask); name != "" {
		parts = append(parts, "SpellSchool: "+name)
	}
	if name := defenseTypeName(row.DefenseType); name != "" {
		parts = append(parts, "DefenseType: "+name)
	}
	if len(row.Effects) > 0 {
		var es []string
		for _, e := range row.Effects {
			f := fmt.Sprintf("{Index: %d, Effect: %s, Aura: %s, Misc: %d, Value: %s",
				e.Index, namer.Effect(e.Effect), namer.Aura(e.Aura), e.Misc, num(e.Value))
			if e.ValueMax > 0 {
				f += ", ValueMax: " + num(e.ValueMax)
			}
			if e.ChainAmplitude != 0 && e.ChainAmplitude != 1 {
				f += ", ChainAmplitude: " + num(e.ChainAmplitude)
			}
			if e.Coef != 0 {
				f += ", Coef: " + num(e.Coef)
			}
			if e.APCoef != 0 {
				f += ", APCoef: " + num(e.APCoef)
			}
			es = append(es, f+"}")
		}
		parts = append(parts, "Effects: []shared.SpellDataEffect{"+strings.Join(es, ", ")+"}")
	}
	for _, role := range []struct {
		name  string
		value *generatedAmount
	}{
		{"Direct", row.Direct},
		{"Heal", row.Heal},
		{"Periodic", row.Periodic},
		{"Energize", row.Energize},
		{"SecondaryPeriodic", row.SecondaryPeriodic},
	} {
		if role.value != nil {
			parts = append(parts, role.name+": "+formatValue(*role.value))
		}
	}
	return "{" + strings.Join(parts, ", ") + "},"
}

// Picks the variant from the shape of the data: a periodic value if it ticks, a range if the client
// rolls it, a flat number otherwise.
func formatValue(a generatedAmount) string {
	tail := fmt.Sprintf("Coef: %s", num(a.Coef))
	if a.APCoef > 0 {
		tail += fmt.Sprintf(", APCoef: %s", num(a.APCoef))
	}

	switch {
	case a.PeriodMs > 0:
		tick := num(a.Min)
		if a.Max > a.Min {
			tick += fmt.Sprintf(", TickMax: %s", num(a.Max))
		}
		out := fmt.Sprintf("shared.SpellDataPeriodic{Tick: %s, %s, TickLength: %s", tick, tail, millis(a.PeriodMs))
		if a.Ticks > 0 {
			out += fmt.Sprintf(", NumberOfTicks: %d", a.Ticks)
		}
		if a.SpellID > 0 {
			out += fmt.Sprintf(", SpellID: %d", a.SpellID)
		}
		return out + "}"
	case a.Max > a.Min:
		return fmt.Sprintf("shared.SpellDataRange{Min: %s, Max: %s, %s}", num(a.Min), num(a.Max), tail)
	default:
		return fmt.Sprintf("shared.SpellDataFlat{Value: %s, %s}", num(a.Min), tail)
	}
}

// Written as a time.Duration expression rather than a bare number, so the generated file reads the way
// a hand-written cast time or tick length does.
func millis(ms int32) string {
	return fmt.Sprintf("%d * time.Millisecond", ms)
}

func num(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
