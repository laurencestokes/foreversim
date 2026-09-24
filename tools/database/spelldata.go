package database

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"

	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/tools/database/dbc"
)

const RankLevel = 60

// Rage is stored in tenths: Heroic Strike costs 150, not 15. Mana, energy and focus are not.
func NormalizePowerCost(cost int32, powerType int32) int32 {
	if dbcenums.PowerType(powerType).InTenths() {
		return cost / 10
	}
	return cost
}

func IsWeaponDamageEffect(effect dbc.SpellEffectType) bool {
	return effect == dbcenums.E_WEAPON_DAMAGE_NOSCHOOL || effect == dbcenums.E_WEAPON_DAMAGE ||
		effect == dbcenums.E_NORMALIZED_WEAPON_DMG
}

// The client states a flat threat amount on the abilities whose point is threat: Feint and Cower
// shed it, Distracting Shot adds it. 22 ranked spells across four families carry one.
func IsThreatEffect(effect dbc.SpellEffectType) bool {
	return effect == dbcenums.E_THREAT || effect == dbcenums.E_THREAT_ALL
}

func IsPeriodicAura(aura dbc.EffectAuraType) bool {
	return aura == dbcenums.A_PERIODIC_DAMAGE || aura == dbcenums.A_PERIODIC_LEECH
}

type RankEffect struct {
	Index        int32
	Effect       dbc.SpellEffectType
	Aura         dbc.EffectAuraType
	BasePoints   int32
	PointsPerLvl float64
	Coefficient  float64
	APCoef       float64
	MiscValue    int32
	AuraPeriod   int32
	OwnerSpellID int32

	// SpellEffect.EffectChainAmplitude, which the client uses for more than chain falloff: Execute's
	// tooltip multiplies it by 10 for the damage each extra rage adds. 1 on nearly every effect.
	ChainAmplitude float64

	// SpellEffect.EffectTriggerSpell: the spell this effect fires, which for a seal's aura dummy is
	// the per-hit proc. Zero when it fires nothing.
	TriggerSpell int32

	// SpellLevels of the spell the effect belongs to, which is not always the rank's own: Blizzard
	// rank 1's damage sits on 1279976, a level 20-25 spell gaining 0.1 a level, whatever spell 10 says.
	SpellLevel int32
	MaxLevel   int32

	// Reached through the rank's description rather than stated by the rank or a same-name sibling -
	// see ReferencedEffects. A named dummy is the rank's own number, kept on another spell.
	Named bool
}

type RankSpell struct {
	SpellID    int32
	SpellLevel int32
	MaxLevel   int32
	ManaCost   sql.NullInt64
	PowerType  int32

	// SpellPower.PowerCostPct: the share of the pool the cast costs, as a percentage. Bloodrage
	// reads 20 against health (PowerType -2), Arcane Blast 15 against mana.
	PowerCostPct float64
	DurationMs   int32
	CastTimeMs   int32
	GCDMs        int32
	CooldownMs   int32
	MinRange     float64
	MaxRange     float64

	// How fast the projectile flies, in yards per second, which core turns into the delay between
	// the cast landing and the damage arriving. Zero for a spell that hits the instant it is cast.
	MissileSpeed float64

	// SpellAuraOptions.ProcChance, as a percentage. 100 means the aura fires on its own condition
	// rather than on a roll, which is how Flurry and Enrage read.
	ProcChance int32

	// SpellAuraOptions.ProcCharges: how many times the aura acts before it drops (Shield Block 2,
	// Retaliation 30). Zero is unlimited.
	ProcCharges int32

	// SpellTargetRestrictions.MaxTargets for an area effect (Whirlwind 4). Zero is unlimited.
	MaxTargets int32

	// SpellMisc.Attributes[1] carries Discount Power On Miss: the server refunds 80% of the cost
	// when the spell misses, which is what the rage specials' Refund models.
	RefundsOnMiss bool

	// SpellMisc.Attributes[8] carries Periodic Can Crit: the ticks of this spell's periodic effect
	// roll a critical strike.
	PeriodicCanCrit bool

	// SpellMisc.SchoolMask, in the client's bit order, which is not the sim's - see schoolName.
	SchoolMask int32

	// SpellCategories.DefenseType: 1 magic, 2 melee, 3 ranged, 0 none. Same order core uses.
	DefenseType int32

	Effects []RankEffect

	// Effects of other spells the description names, in its order - see ReferencedEffects. A tick
	// fills Periodic and a second one SecondaryPeriodic; a dummy-held number fills Direct.
	Referenced []RankEffect
}

func RequireSpellCastTimes(helper *DBHelper) error {
	if helper.tableExists("SpellCastTimes") {
		return nil
	}
	return errors.New("the client database has no SpellCastTimes table, so every generated cast time would " +
		"be zero - add \"SpellCastTimes\" to tools/database/generator-settings.json and re-run `make db`")
}

// The calibrated rule, shared with the regeneration check. Scored 86/87 against the hand tables.
//
// float32 is load-bearing: EffectRealPointsPerLevel is a float32 widened into the DB
// (3.79999995231628), and multiplying in float64 breaks 6 rows. Reproduces the tooltip, which is not
// the same as reproducing the server roll.
func DeriveRankAmount(e RankEffect, spellLevel, maxLevel int32) (min float64, max float64) {
	cap := maxLevel
	if cap <= 0 {
		cap = RankLevel
	}
	lvl := int32(RankLevel)
	if cap < lvl {
		lvl = cap
	}
	delta := lvl - spellLevel
	if delta < 0 {
		delta = 0
	}

	base := float32(e.BasePoints) + float32(float32(delta)*float32(e.PointsPerLvl))
	// EffectBasePointsF is the amount itself: Improved Battle Shout reads 5/10/15/20/25 and generates
	// as 5/10/15/20/25. The client states no die sides, so the amount has no spread.
	min = math.Floor(float64(base))
	return min, min
}

// Whether the spell carries the Passive attribute: never cast, only applied.
func SpellIsPassive(db *sql.DB, spellID int32) (bool, error) {
	return spellHasAttribute(db, spellID, dbcenums.ATTR_INDEX_BASE, dbcenums.ATTR_PASSIVE)
}

// Whether the flag is set in SpellMisc.Attributes[attrIndex]; false for a spell with no SpellMisc row.
func spellHasAttribute(db *sql.DB, spellID int32, attrIndex int, flag uint32) (bool, error) {
	var set bool
	err := scanOptional(db, fmt.Sprintf(
		`SELECT (COALESCE(json_extract(Attributes, '$[%d]'), 0) & %d) != 0 FROM SpellMisc WHERE SpellID = ? AND DifficultyID = 0`,
		attrIndex, flag), spellID, &set)
	return set, err
}

func LoadRankSpell(db *sql.DB, spellID int32) (RankSpell, error) {
	s := RankSpell{SpellID: spellID}
	err := db.QueryRow(`
		SELECT SpellLevel, MaxLevel FROM SpellLevels WHERE SpellID = ?`, spellID).Scan(&s.SpellLevel, &s.MaxLevel)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Passive talents such as the warrior's Blood Craze have no SpellLevels row at all. No level
		// data means no level scaling, which is what setting the spell's own level to the cap gives.
		s.SpellLevel = RankLevel
	case err != nil:
		return s, fmt.Errorf("levels for spell %d: %w", spellID, err)
	}

	// The cost stands on its own: Divine Favor has no SpellLevels row and still costs 4% of base
	// mana, so it cannot ride on the level query.
	if err := scanOptional(db, `
		SELECT ManaCost, COALESCE(PowerType, 0), COALESCE(PowerCostPct, 0)
		FROM SpellPower WHERE SpellID = ? ORDER BY OrderIndex LIMIT 1`, spellID, &s.ManaCost, &s.PowerType, &s.PowerCostPct); err != nil {
		return s, fmt.Errorf("power for spell %d: %w", spellID, err)
	}

	// Duration and its period are what a DoT's NumberOfTicks and TickLength are derived from.
	if err := scanOptional(db, `
		SELECT COALESCE(d.Duration, 0)
		FROM SpellMisc m LEFT JOIN SpellDuration d ON d.ID = m.DurationIndex
		WHERE m.SpellID = ?`, spellID, &s.DurationMs); err != nil {
		return s, fmt.Errorf("duration for spell %d: %w", spellID, err)
	}

	// Cooldown takes the longer of the two: Fire Blast and Cone of Cold use the shared
	// CategoryRecoveryTime, everything else its own RecoveryTime.
	if err := scanOptional(db, `
		SELECT COALESCE(max(RecoveryTime, CategoryRecoveryTime), 0), COALESCE(StartRecoveryTime, 0)
		FROM SpellCooldowns WHERE SpellID = ?`, spellID, &s.CooldownMs, &s.GCDMs); err != nil {
		return s, fmt.Errorf("cooldown for spell %d: %w", spellID, err)
	}

	// RangeMin is nonzero on only 212 spells in this build - the dead zone on a charge, and a handful
	// of ranged abilities - but where it exists core gates the cast on it exactly as it does MaxRange.
	if err := scanOptional(db, `
		SELECT COALESCE(r.RangeMin_0, 0), COALESCE(r.RangeMax_0, 0)
		FROM SpellMisc m JOIN SpellRange r ON r.ID = m.RangeIndex
		WHERE m.SpellID = ?`, spellID, &s.MinRange, &s.MaxRange); err != nil {
		return s, fmt.Errorf("range for spell %d: %w", spellID, err)
	}

	if err := scanOptional(db,
		`SELECT COALESCE(Speed, 0) FROM SpellMisc WHERE SpellID = ?`, spellID, &s.MissileSpeed); err != nil {
		return s, fmt.Errorf("missile speed for spell %d: %w", spellID, err)
	}

	// Both tables carry a row per DifficultyID on a few spells - 29213 caps 20 targets at 0 and 10 at
	// 186 - and the base row is the one wanted.
	if err := scanOptional(db,
		`SELECT COALESCE(ProcChance, 0), COALESCE(ProcCharges, 0) FROM SpellAuraOptions WHERE SpellID = ? ORDER BY DifficultyID`, spellID, &s.ProcChance, &s.ProcCharges); err != nil {
		return s, fmt.Errorf("proc chance for spell %d: %w", spellID, err)
	}

	if err := scanOptional(db,
		`SELECT COALESCE(MaxTargets, 0) FROM SpellTargetRestrictions WHERE SpellID = ? ORDER BY DifficultyID`, spellID, &s.MaxTargets); err != nil {
		return s, fmt.Errorf("max targets for spell %d: %w", spellID, err)
	}

	if s.RefundsOnMiss, err = spellHasAttribute(db, spellID, dbcenums.ATTR_INDEX_EX_1, dbcenums.ATTR_EX_1_DISCOUNT_POWER_ON_MISS); err != nil {
		return s, fmt.Errorf("miss refund for spell %d: %w", spellID, err)
	}

	if s.PeriodicCanCrit, err = spellHasAttribute(db, spellID, dbcenums.ATTR_INDEX_EX_8, dbcenums.ATTR_EX_8_PERIODIC_CAN_CRIT); err != nil {
		return s, fmt.Errorf("periodic crit for spell %d: %w", spellID, err)
	}

	if err := scanOptional(db,
		`SELECT COALESCE(SchoolMask, 0) FROM SpellMisc WHERE SpellID = ?`, spellID, &s.SchoolMask); err != nil {
		return s, fmt.Errorf("school for spell %d: %w", spellID, err)
	}

	if err := scanOptional(db,
		`SELECT COALESCE(DefenseType, 0) FROM SpellCategories WHERE SpellID = ?`, spellID, &s.DefenseType); err != nil {
		return s, fmt.Errorf("defense type for spell %d: %w", spellID, err)
	}

	if err := scanOptional(db, `
		SELECT COALESCE(ct.Base, 0)
		FROM SpellMisc m JOIN SpellCastTimes ct ON ct.ID = m.CastingTimeIndex
		WHERE m.SpellID = ?`, spellID, &s.CastTimeMs); err != nil {
		return s, fmt.Errorf("cast time for spell %d: %w", spellID, err)
	}

	if s.Effects, err = RankEffectsOf(db, spellID); err != nil {
		return s, err
	}
	s.Referenced, err = ReferencedEffects(db, s)
	return s, err
}

// No row means no level scaling, the way LoadRankSpell reads it.
func levelsOf(db *sql.DB, spellID int32) (spellLevel, maxLevel int32, err error) {
	err = db.QueryRow(`SELECT SpellLevel, MaxLevel FROM SpellLevels WHERE SpellID = ?`, spellID).Scan(&spellLevel, &maxLevel)
	if errors.Is(err, sql.ErrNoRows) {
		return RankLevel, 0, nil
	}
	return spellLevel, maxLevel, err
}

// A $<spellID><token><n> in a description, the "$1280349m1" in Consecration's, with or without the
// divisor Seal of Righteousness renders its "$/87;20286s3" through. Four digits is the shortest
// real spell ID, which keeps the plain $s3 and $m2 out.
var descriptionValueRef = regexp.MustCompile(`\$(?:/\d+;)?(\d{4,7})[ms](\d)`)

// The client links a rank to a number kept on another spell in two ways, and both are followed here.
//
// Forever moves a ground effect's damage onto a spell of its own - Consecration rank 5 ticks through
// 1280349, Blizzard rank 1 through 1279976 - that the client links from nowhere but the description.
// What the rank itself states is a dummy, the area trigger, and a periodic dummy whose period is the
// tick and whose points are not damage: Consecration's 4 is how many targets take the extra damage.
// The description's "$1280349m1" is followed to the named damage effect of a spell sharing the rank's
// name, and it is given the dummy's period, so it files as the periodic value.
//
// Seal of Righteousness states no value at all: an aura dummy at the number each hit adds, and a
// second whose points are the rank's judgement, 20286 on rank 8. Its description renders the hit off
// the judgement - "$/87;20286s3" - and effect 3 of 20286 is a dummy the judgement does nothing with
// itself: the same number, with the coefficient the seal's own copy lacks (0.1 on ranks 1-7, none on
// rank 8, where the judgement's dummy says 0.2). So a reference into a spell the rank's own dummy
// points at is followed to the named effect when that is a dummy, and it files as the direct value.
// A named damage or aura effect on the pointed spell is that spell's own - Seal of Fury and Seal of
// the Crusader both name their judgement's - and stays with it.
//
// Seal of Fury keeps its per-hit damage on the spell its aura dummy fires. Rank 7's effect 0 is a
// dummy at 1607 gaining 42 a level - Seal of Righteousness' number, left over from when the seal
// was a copy of it - with EffectTriggerSpell 20418, and the description renders the hit off that
// spell: "$20418s1 Holy damage", a school damage effect at 35 with the 0.1 coefficient the dummy
// lacks (0.09 on ranks 1-6, none on rank 7). So a reference into a spell one of the rank's own
// effects triggers is followed to the named effect whatever its shape, and it files by that shape:
// the proc's damage is the seal's direct value. Rank 4's judgement pointer names the judgement as
// its trigger too, so a spell the rank points at is read by the paragraph above first, and its
// damage stays with it.
//
// Nil for a rank that is neither shape, which is everything outside the ground-effect families and
// the seals.
func ReferencedEffects(db *sql.DB, spell RankSpell) ([]RankEffect, error) {
	var period int32
	pointed := map[int32]bool{}
	triggered := map[int32]bool{}
	for _, e := range spell.Effects {
		if IsPeriodicAura(e.Aura) {
			return nil, nil
		}
		if e.Aura == dbcenums.A_PERIODIC_DUMMY && e.AuraPeriod > 0 {
			period = e.AuraPeriod
		}
		// A dummy whose points do not scale with level is read as a spell ID; it only matters once
		// the description names that spell.
		if (e.Effect == dbcenums.E_DUMMY || e.Aura == dbcenums.A_DUMMY) && e.PointsPerLvl == 0 && e.BasePoints > 0 {
			pointed[e.BasePoints] = true
		}
		// The client is not consistent about which effect carries the trigger - Seal of Fury rank 5
		// keeps it on the judgement pointer, rank 7 on the damage dummy - so every effect is read.
		if e.TriggerSpell > 0 {
			triggered[e.TriggerSpell] = true
		}
	}
	if period == 0 && (len(pointed)+len(triggered) == 0 || HasValueEffect(spell.Effects)) {
		return nil, nil
	}

	var name, description string
	if err := db.QueryRow(`
		SELECT n.Name_lang, COALESCE(s.Description_lang, '')
		FROM SpellName n JOIN Spell s ON s.ID = n.ID
		WHERE n.ID = ?`, spell.SpellID).Scan(&name, &description); err != nil {
		return nil, fmt.Errorf("description of spell %d: %w", spell.SpellID, err)
	}

	var out []RankEffect
	seen := map[[2]int32]bool{}
	for _, m := range descriptionValueRef.FindAllStringSubmatch(description, -1) {
		id, _ := strconv.Atoi(m[1])
		n, _ := strconv.Atoi(m[2])
		key := [2]int32{int32(id), int32(n) - 1}
		if int32(id) == spell.SpellID || n < 1 || seen[key] {
			continue
		}
		seen[key] = true

		// The effect type the reference must land on; anyShape for a triggered spell, whose named
		// effect is taken as it is.
		var want dbc.SpellEffectType
		anyShape := false
		switch {
		case period > 0:
			var sameName int
			if err := db.QueryRow(`SELECT count(*) FROM SpellName WHERE ID = ? AND Name_lang = ?`,
				id, name).Scan(&sameName); err != nil {
				return nil, err
			}
			if sameName == 0 {
				continue
			}
			want = dbcenums.E_SCHOOL_DAMAGE
		case pointed[int32(id)]:
			want = dbcenums.E_DUMMY
		case triggered[int32(id)]:
			anyShape = true
		default:
			continue
		}

		effects, err := RankEffectsOf(db, int32(id))
		if err != nil {
			return nil, err
		}
		for _, e := range effects {
			if e.Index == key[1] && (anyShape || e.Effect == want) {
				e.AuraPeriod = period
				e.Named = true
				out = append(out, e)
			}
		}
	}
	return out, nil
}

// A spell with no row in one of the optional tables is ordinary - most spells have no cooldown - and
// leaves the destination at its zero, which is what core reads as "ungated". Any other error means the
// schema moved, and silently zeroing a cast time or a range on that is the failure this loader exists
// to avoid.
func scanOptional(db *sql.DB, query string, spellID int32, dest ...any) error {
	err := db.QueryRow(query, spellID).Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func RankEffectsOf(db *sql.DB, spellID int32) ([]RankEffect, error) {
	rows, err := db.Query(`
		-- EffectBasePointsF is a REAL, and Variance carries the spread. BasePoints stays integral
		-- here because the dummy-target heuristic below reads the base as a spell id.
		--
		-- TODO: ~1.8% of SpellEffect rows have a fractional EffectBasePointsF and lose it to
		-- this cast. TODO: the client states no die sides, so DeriveRankAmount answers the same
		-- min and max and the generated rank tables carry no damage range.
		SELECT EffectIndex, Effect, EffectAura, CAST(EffectBasePointsF AS INTEGER),
		       EffectRealPointsPerLevel, EffectBonusCoefficient, BonusCoefficientFromAP, EffectAuraPeriod,
		       COALESCE(EffectMiscValue_0, 0), COALESCE(EffectTriggerSpell, 0), COALESCE(EffectChainAmplitude, 0)
		FROM SpellEffect WHERE SpellID = ? ORDER BY EffectIndex`, spellID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spellLevel, maxLevel, err := levelsOf(db, spellID)
	if err != nil {
		return nil, fmt.Errorf("levels for spell %d: %w", spellID, err)
	}

	var out []RankEffect
	for rows.Next() {
		e := RankEffect{OwnerSpellID: spellID, SpellLevel: spellLevel, MaxLevel: maxLevel}
		if err := rows.Scan(&e.Index, &e.Effect, &e.Aura, &e.BasePoints, &e.PointsPerLvl, &e.Coefficient, &e.APCoef, &e.AuraPeriod, &e.MiscValue, &e.TriggerSpell, &e.ChainAmplitude); err != nil {
			return nil, err
		}
		// Stored as a float32, so 0.7 arrives as 0.699999988079071.
		e.ChainAmplitude = math.Round(e.ChainAmplitude*1e6) / 1e6
		out = append(out, e)
	}
	return out, rows.Err()
}

// Judgement of Command's rows sit on the Retribution skill line with ClassMask 0, so the sibling
// search by name finds nothing. The dummy names its target itself: its base points derive to the
// damage spell's ID - 20425 states 20467 - and that spell shares the name and the rank subtext, which
// is what is checked before its effects are taken. Following the pointer
// rather than widening the name search keeps Blizzard's tick spell out of its parent's Direct.
func dummyTargetEffects(db *sql.DB, spell RankSpell) ([]RankEffect, error) {
	if len(spell.Effects) != 1 || spell.Effects[0].Effect != dbcenums.E_DUMMY {
		return nil, nil
	}
	target, _ := DeriveRankAmount(spell.Effects[0], spell.SpellLevel, spell.MaxLevel)
	id := int32(target)
	if float64(id) != target || id <= 0 {
		return nil, nil
	}

	var same int
	if err := db.QueryRow(`
		SELECT count(*)
		FROM Spell s
		JOIN SpellName n ON n.ID = s.ID
		WHERE s.ID = ?
		  AND n.Name_lang = (SELECT Name_lang FROM SpellName WHERE ID = ?)
		  AND s.NameSubtext_lang = (SELECT NameSubtext_lang FROM Spell WHERE ID = ?)`,
		id, spell.SpellID, spell.SpellID).Scan(&same); err != nil {
		return nil, err
	}
	if same == 0 {
		return nil, nil
	}
	return RankEffectsOf(db, id)
}

// Holy Shock's registered spells carry only Effect=3 (dummy) and have no EffectTriggerSpell edge to the
// damage and heal spells that share their name and rank - the association exists nowhere but the name.
// Restricted to the family's class so an NPC copy of the name cannot be picked up.
func SiblingRankEffects(db *sql.DB, spellID int32, classBit int) ([]RankEffect, error) {
	rows, err := db.Query(`
		SELECT DISTINCT sla.Spell
		FROM SkillLineAbility sla
		JOIN SpellName n ON n.ID = sla.Spell
		JOIN Spell s ON s.ID = sla.Spell
		WHERE n.Name_lang = (SELECT Name_lang FROM SpellName WHERE ID = ?)
		  AND s.NameSubtext_lang = (SELECT NameSubtext_lang FROM Spell WHERE ID = ?)
		  AND (sla.ClassMask & ?) != 0
		  AND sla.Spell != ?
		ORDER BY sla.Spell`, spellID, spellID, classBit, spellID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []RankEffect
	for _, id := range ids {
		effs, err := RankEffectsOf(db, id)
		if err != nil {
			return nil, err
		}
		out = append(out, effs...)
	}
	return out, nil
}

func HasValueEffect(effects []RankEffect) bool {
	for _, e := range effects {
		if e.Effect == dbcenums.E_SCHOOL_DAMAGE || e.Effect == dbcenums.E_HEAL || e.Effect == dbcenums.E_ENERGIZE ||
			IsWeaponDamageEffect(e.Effect) || IsPeriodicAura(e.Aura) {
			return true
		}
	}
	return false
}

// Every effect that could carry the numbers a rank row wants, including the ones reached only through
// the description or a same-name sibling.
func RankCandidates(db *sql.DB, spellID int32, classBit int) (RankSpell, []RankEffect, error) {
	spell, err := LoadRankSpell(db, spellID)
	if err != nil {
		return spell, nil, err
	}

	candidates := slices.Concat(spell.Effects, spell.Referenced)
	if !HasValueEffect(candidates) {
		sibs, err := SiblingRankEffects(db, spellID, classBit)
		if err != nil {
			return spell, nil, err
		}
		if len(sibs) == 0 {
			sibs, err = dummyTargetEffects(db, spell)
			if err != nil {
				return spell, nil, err
			}
		}
		candidates = slices.Concat(spell.Effects, spell.Referenced, sibs)
	}
	return spell, candidates, nil
}
