package database

// Resolves tools/database/buffmanifest against the spell store's captured client
// rows and renders sim/core/buffs/buffs_auto_gen.go and
// sim/core/buffs/debuffs_auto_gen.go from the result, in the same pass that
// renders the store.
//
// The manifest names the spell each proto field reads, and the generated file
// states it as a buffs.Meta: which of its effects are which stats, what they are
// worth and how they stack is read off the row at runtime by
// spelldata.ParseEffects. The generator asks the same parse, through
// spelldata.DryRun, whether the row attaches anything and what it leaves out, so
// a row it writes is one the sim builds the same way. A row the parse attaches
// nothing of renders as a commented shell carrying the reason.

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"text/template"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/tools/database/buffmanifest"
)

const buffsGenFile = "sim/core/buffs/buffs_auto_gen.go"
const debuffsGenFile = "sim/core/buffs/debuffs_auto_gen.go"

// ResolvedBuff is one manifest row plus everything the database states about it.
type ResolvedBuff struct {
	buffmanifest.BuffSpec

	SpellID     int32 // the spell the aura's numbers are read from
	CastSpellID int32 // the cast, where the manifest pins one; SpellID otherwise
	DurationMs  int32 // -1 or 0 never expires
	CooldownMs  int32 // 0 when the client states none; read from the cast

	// Spell is the row the store carries for SpellID, built from the same
	// captured client rows the store is rendered from.
	Spell *spelldata.Spell

	// SkipAuraTypes is the manifest's SkipAuras read as auras.
	SkipAuraTypes []dbcenums.EffectAuraType
	// FullComboPoints says a debuff's amount is stated per combo point, and the
	// raid config's copy is the finisher at full combo points.
	FullComboPoints bool

	// Applied is what the parse attaches for the row, and LeftOut the aura
	// effects it leaves out, one note each.
	Applied []spelldata.Applied
	LeftOut []string

	// TalentRanks is how many points the improving talent takes, 0 for a row
	// no talent prices.
	TalentRanks   int32
	TalentApplies buffmanifest.TalentApplies

	// TalentSpellID is the spell of the trait node that prices the improvement,
	// which is the icon the UI shows for the improved state. TalentPosition is
	// the effect of that spell the improvement is read from, counted the way
	// EffectN counts.
	TalentSpellID  int32
	TalentPosition int32

	ScopeFromClient buffmanifest.BuffScope

	DBName string

	Supported bool
	Reason    string
	Warnings  []string
}

func (r *ResolvedBuff) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *ResolvedBuff) unsupported(format string, args ...any) {
	r.Supported = false
	r.Reason = fmt.Sprintf(format, args...)
}

// Every manifest row, read out of the client rows the store is built from.
func resolveBuffManifest(in *storeInputs) ([]ResolvedBuff, error) {
	t := in.tables()
	rows := make([]ResolvedBuff, 0, len(buffmanifest.Manifest))
	for _, spec := range buffmanifest.Manifest {
		row, err := resolveBuff(t, in.TraitNodes, spec)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.Field, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func resolveBuff(t *spellTables, nodes []traitNode, spec buffmanifest.BuffSpec) (ResolvedBuff, error) {
	row := ResolvedBuff{BuffSpec: spec, Supported: true, ScopeFromClient: spec.Scope}

	compiled, err := compiledProtoType(spec)
	if err != nil {
		return row, err
	}

	if spec.Kind == buffmanifest.KindAbsent || spec.Kind == buffmanifest.KindFlag {
		row.unsupported("%s", spec.Notes)
		return row, nil
	}

	if err := loadBuffSpell(t, &row); err != nil {
		return row, err
	}
	if err := resolveSkipAuras(&row); err != nil {
		return row, err
	}
	parseBuff(&row)
	if err := resolveTalent(t, nodes, &row); err != nil {
		return row, err
	}
	validateScope(&row)

	// The sim puts every resistance stat into its school's category by itself, so
	// a row whose manifest category is that school says the same thing twice and
	// has no exclusivity of its own beyond it.
	if isSchoolResistanceCategory(row.Category) {
		row.Category = ""
	}

	if compiled != buffProtoTypeNames[spec.Proto] && row.Supported {
		row.unsupported("proto field not yet retyped: the sim compiled against %s, the manifest declares %s",
			compiled, buffProtoTypeNames[spec.Proto])
	}
	return row, nil
}

// The spell's name, its row and duration, and the cooldown of a buff other
// players cast on their own: the shared timer the sim hands the next source is
// the cast's cooldown, and a totem's aura states neither how long the totem
// stands nor when the next one may be dropped. Mana Tide is that row - 17360
// carries the mana, the cast 17359 the 13 seconds and the 5 minutes.
func loadBuffSpell(t *spellTables, row *ResolvedBuff) error {
	if row.SpellID = row.BuffSpec.SpellID; row.SpellID == 0 {
		return fmt.Errorf("names no spell")
	}
	row.CastSpellID = row.SpellID
	if row.CastID != 0 {
		row.CastSpellID = row.CastID
	}
	for _, id := range []int32{row.SpellID, row.CastSpellID} {
		if _, named := t.Names[id]; !named {
			return fmt.Errorf("spell %d is no spell in this client", id)
		}
	}

	spell := t.row(row.SpellID)
	row.Spell = storeRowSpell(spell)
	row.DBName = spell.Name
	row.DurationMs = spell.DurationMs

	if row.Kind != buffmanifest.KindExternalCD {
		return nil
	}
	cast := t.row(row.CastSpellID)
	row.CooldownMs = max(cast.CooldownMs, cast.CategoryCooldownMs)
	if row.CooldownMs == 0 {
		row.unsupported("spell %d states no cooldown, which an external cooldown is scheduled by", row.CastSpellID)
		return nil
	}
	if row.DurationMs <= 0 && row.CastSpellID != row.SpellID {
		row.DurationMs = cast.DurationMs
	}
	if row.DurationMs <= 0 {
		row.unsupported("spell %d states no duration, which the external cooldown's aura needs", row.SpellID)
	}
	return nil
}

// The store's row as sim/core/spelldata states it, copied field by field from the generator's
// mirror of it. The parse reads this rather than the compiled store, which is the output the same
// pass rewrites.
func storeRowSpell(row storeSpell) *spelldata.Spell {
	s := &spelldata.Spell{}
	copyStoreFields(reflect.ValueOf(s).Elem(), reflect.ValueOf(row))
	return s
}

func copyStoreFields(to reflect.Value, from reflect.Value) {
	for i := 0; i < from.NumField(); i++ {
		field := from.Type().Field(i)
		dst := to.FieldByName(field.Name)
		if !field.IsExported() || !dst.IsValid() {
			continue
		}

		src := from.Field(i)
		if src.Kind() == reflect.Slice && src.Type().Elem().Kind() == reflect.Struct {
			dst.Set(reflect.MakeSlice(dst.Type(), src.Len(), src.Len()))
			for j := 0; j < src.Len(); j++ {
				copyStoreFields(dst.Index(j), src.Index(j))
			}
			continue
		}
		dst.Set(src.Convert(dst.Type()))
	}
}

// The manifest's SkipAuras, read as the auras dbcenums names.
func resolveSkipAuras(row *ResolvedBuff) error {
	for _, name := range row.SkipAuras {
		aura, ok := auraByName(name)
		if !ok {
			return fmt.Errorf("SkipAuras names %q, which is no aura", name)
		}
		row.SkipAuraTypes = append(row.SkipAuraTypes, aura)
	}
	return nil
}

func auraByName(name string) (dbcenums.EffectAuraType, bool) {
	for aura := dbcenums.EffectAuraType(0); aura < 1024; aura++ {
		if named, ok := dbcenums.Named(aura); ok && named == name {
			return aura, true
		}
	}
	return 0, false
}

// What the parse makes of the row with the options the generated Meta states:
// the amounts it attaches and the aura effects it leaves out. A kind whose
// behaviour is a driver's is written whatever the parse attaches, a damage
// shield needs its shield, and every other kind needs an amount.
func parseBuff(row *ResolvedBuff) {
	// The value a debuff states per combo point is 0 on the spell itself. The
	// raid config is one debuff that is simply on the target, which is the
	// finisher at full combo points; a caster spending fewer of them takes its
	// own value through a driver.
	if row.Kind == buffmanifest.KindDebuffStat {
		row.FullComboPoints = slices.ContainsFunc(row.Spell.Effects, func(e spelldata.Effect) bool {
			return e.PointsPerResource != 0 && e.Average(core.CharacterLevel) == 0
		})
	}

	// The constructors parse a buff with no character, as they do a debuff.
	parsed := spelldata.DryRun(row.Spell, row.parseOptions()...)
	row.Applied = parsed.Applied
	row.LeftOut = parsed.SkippedNotes(row.Spell)

	switch {
	case row.Kind == buffmanifest.KindDamageShield:
		if !row.hasDamageShield() {
			row.unsupported("no A_DAMAGE_SHIELD effect on spell %d", row.SpellID)
		}
	case isDriverKind(row.Kind):
		// Driver kinds: the hand-written driver decides what the numbers mean.
	default:
		if len(row.Applied) > 0 {
			return
		}
		if len(row.LeftOut) == 0 {
			row.unsupported("spell %d states no aura effect the parse attaches", row.SpellID)
			return
		}
		row.unsupported("spell %d states no aura effect the parse attaches: %s", row.SpellID,
			strings.Join(row.LeftOut, "; "))
	}
}

// The options buffs.Meta.Options states for the row, less the talent's scaling,
// which prices an amount and never changes what is attached. A damage shield's
// own effect is left to newDamageShield, and an item-count row counts its
// amounts, which is what a multiplier cannot be read through.
func (row *ResolvedBuff) parseOptions() []spelldata.ParseOpt {
	skip := row.SkipAuraTypes
	if row.Kind == buffmanifest.KindDamageShield {
		skip = append(slices.Clone(skip), dbcenums.A_DAMAGE_SHIELD)
	}
	opts := spelldata.RaidBuffOptions(skip, row.FullComboPoints)
	if row.Kind == buffmanifest.KindItemCount {
		opts = append(opts, spelldata.Count(1))
	}
	return opts
}

func (row *ResolvedBuff) hasDamageShield() bool {
	return slices.ContainsFunc(row.Spell.Effects, func(e spelldata.Effect) bool {
		return e.Aura == dbcenums.A_DAMAGE_SHIELD && spelldata.AppliesAura(e.Type)
	})
}

// The improving talent the manifest pins, which the store keeps as a ladder of
// its ranks. Its effect has to be a modifier whose class mask reaches the buff's
// spell; one that modifies misc 1 scales the duration, anything else the value.
func resolveTalent(t *spellTables, nodes []traitNode, row *ResolvedBuff) error {
	if row.Talent == nil {
		if row.Proto == buffmanifest.ProtoTristate && row.ImpAction == nil {
			return fmt.Errorf("declared ProtoTristate but the manifest names neither a talent nor an ImpAction")
		}
		return nil
	}
	if row.Proto != buffmanifest.ProtoTristate {
		return fmt.Errorf("names talent %q but is not ProtoTristate", row.Talent.Name)
	}

	id := row.Talent.SpellID
	node := slices.IndexFunc(nodes, func(n traitNode) bool { return n.SpellID == id })
	if node < 0 {
		return fmt.Errorf("talent %d is no node of a class tree", id)
	}
	talent := t.row(id)
	if talent.Name != row.Talent.Name {
		return fmt.Errorf("talent %d is %q, the manifest says %q", id, talent.Name, row.Talent.Name)
	}
	position := slices.IndexFunc(talent.Effects, func(e storeEffect) bool { return int32(e.Index) == row.Talent.Effect })
	if position < 0 {
		return fmt.Errorf("talent %d has no effect %d", id, row.Talent.Effect)
	}
	mod := talent.Effects[position]
	if mod.Aura != dbcenums.A_ADD_FLAT_MODIFIER && mod.Aura != dbcenums.A_ADD_PCT_MODIFIER {
		return fmt.Errorf("talent %d effect %d is aura %d, not a spell modifier", id, row.Talent.Effect, mod.Aura)
	}
	if !mod.ClassFlags.Matches(t.row(row.SpellID).ClassFlags) {
		return fmt.Errorf("talent %d effect %d does not reach spell %d", id, row.Talent.Effect, row.SpellID)
	}

	row.TalentSpellID = id
	row.TalentPosition = int32(position + 1)
	row.TalentApplies = row.Talent.Applies
	if mod.Misc == int32(dbcenums.SPELLMOD_DURATION) {
		row.TalentApplies = buffmanifest.TalentScalesDuration
	}
	if row.TalentApplies != buffmanifest.TalentScalesDuration && len(row.Applied) == 0 && !row.hasDamageShield() {
		row.warn("talent %q has nothing to scale: the parse attaches no amount of spell %d", row.Talent.Name, row.SpellID)
		return nil
	}
	row.TalentRanks = nodes[node].MaxRanks
	return nil
}

// Who the client says the aura reaches. The manifest wins - the sim's scopes are
// a UI grouping as much as a game fact - but a disagreement is worth printing.
func validateScope(row *ResolvedBuff) {
	scope, known := clientScope(*row)
	if !known {
		return
	}
	row.ScopeFromClient = scope
	if scope != row.Scope {
		row.warn("scope mismatch: the manifest says %s, spell %d reads as %s", row.Scope, row.SpellID, scope)
	}
}

// The group the client states a buff reaches: an area aura names it in the
// effect itself, and an aura the client applies over an area or on one ally
// names it in the effect's target.
func clientScope(row ResolvedBuff) (buffmanifest.BuffScope, bool) {
	if row.Spell == nil {
		return buffmanifest.ScopeIndividual, false
	}
	for _, effect := range row.Spell.Effects {
		switch effect.Type {
		case dbcenums.E_APPLY_AREA_AURA_RAID:
			return buffmanifest.ScopeRaid, true
		case dbcenums.E_APPLY_AREA_AURA_PARTY:
			return buffmanifest.ScopeParty, true
		case dbcenums.E_APPLY_AURA:
			switch effect.Target[0] {
			case dbcenums.TARGET_UNIT_CASTER_AREA_RAID:
				return buffmanifest.ScopeRaid, true
			case dbcenums.TARGET_UNIT_CASTER_AREA_PARTY:
				return buffmanifest.ScopeParty, true
			case dbcenums.TARGET_UNIT_TARGET_ALLY, dbcenums.TARGET_153:
				return buffmanifest.ScopeIndividual, true
			}
		}
	}
	return buffmanifest.ScopeIndividual, false
}

// The repository root, found by walking up from the working directory until
// go.mod appears. The generator runs from the root and the regeneration test
// from tools/database, and both have to reach sim/core/buffs.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod above the working directory, so the repository root is unknown")
		}
		dir = parent
	}
}

// The proto message each scope's fields live on. The generated apply blocks read
// the field through the type the sim compiled against, so the manifest's declared
// type is checked against this rather than against proto/buffs.proto: a row
// whose proto field has not been retyped yet would otherwise generate code that
// cannot compile.
var buffScopeMessages = map[buffmanifest.BuffScope]reflect.Type{
	buffmanifest.ScopeRaid:       reflect.TypeOf(proto.RaidBuffs{}),
	buffmanifest.ScopeParty:      reflect.TypeOf(proto.PartyBuffs{}),
	buffmanifest.ScopeIndividual: reflect.TypeOf(proto.IndividualBuffs{}),
	buffmanifest.ScopeDebuff:     reflect.TypeOf(proto.Debuffs{}),
}

var buffProtoTypeNames = map[buffmanifest.BuffProtoType]string{
	buffmanifest.ProtoBool:     "bool",
	buffmanifest.ProtoTristate: "proto.TristateEffect",
	buffmanifest.ProtoInt32:    "int32",
	buffmanifest.ProtoDouble:   "float64",
}

// The Go type the compiled proto states for a field, spelled the way the manifest
// spells it.
func compiledProtoType(spec buffmanifest.BuffSpec) (string, error) {
	message, ok := buffScopeMessages[spec.Scope]
	if !ok {
		return "", fmt.Errorf("unknown scope %s", spec.Scope)
	}
	field, ok := message.FieldByName(spec.GoField())
	if !ok {
		return "", fmt.Errorf("%s has no field %s", message.Name(), spec.GoField())
	}
	switch field.Type.Kind() {
	case reflect.Bool:
		return "bool", nil
	case reflect.Float64:
		return "float64", nil
	case reflect.Int32:
		if name := field.Type.Name(); name != "int32" {
			return "proto." + name, nil
		}
		return "int32", nil
	}
	return field.Type.String(), nil
}

// Whether a manifest category names a resistance school, which is the category
// spelldata.SchoolResistances puts that school's stat into anyway.
func isSchoolResistanceCategory(category string) bool {
	switch category {
	case core.ResistanceCategoryArcane, core.ResistanceCategoryFire, core.ResistanceCategoryFrost,
		core.ResistanceCategoryNature, core.ResistanceCategoryShadow:
		return true
	}
	return false
}

func isDriverKind(kind buffmanifest.BuffKind) bool {
	switch kind {
	case buffmanifest.KindExternalCD, buffmanifest.KindProc, buffmanifest.KindManual, buffmanifest.KindDebuffUptime:
		return true
	}
	return false
}

func lowerFirst(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

///////////////////////////////////////////////////////////////////////////
//							Rendering
///////////////////////////////////////////////////////////////////////////

// buffRow is what the templates see: every expression the generated file needs,
// already spelled as Go source.
type buffRow struct {
	Go          string
	Field       string
	Label       string
	SpellID     int32
	Kind        string
	Reason      string
	LeftOut     []string
	Supported   bool
	HasSpell    bool
	SpellVar    string
	MetaVar     string
	MetaFields  string
	Category    string
	CategoryVar string
	HasValue    bool
	HasCooldown bool
	ExtraParams string
	Constructor string
	ApplyIf     string
	ApplyBody   string
	HasApply    bool
}

// Every file the manifest renders, by the path it is written to: the two Go
// files, the settings inputs and the proto messages.
func renderBuffOutputs(in *storeInputs) (map[string][]byte, error) {
	rows, err := resolveBuffManifest(in)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		for _, warning := range row.Warnings {
			fmt.Fprintf(progress, "buffs: %s: %s\n", row.Field, warning)
		}
	}

	files, err := renderBuffFiles(rows)
	if err != nil {
		return nil, err
	}
	if files[buffsDebuffsTSFile], err = RenderBuffsDebuffsTS(rows); err != nil {
		return nil, err
	}
	files[buffsProtoFile] = buffmanifest.RenderProto(buffmanifest.Manifest)
	return files, nil
}

const buffsProtoFile = "proto/buffs.proto"

func renderBuffFiles(rows []ResolvedBuff) (map[string][]byte, error) {
	buffs, err := renderBuffFile(rows, false)
	if err != nil {
		return nil, err
	}
	debuffs, err := renderBuffFile(rows, true)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{buffsGenFile: buffs, debuffsGenFile: debuffs}, nil
}

func renderBuffFile(resolved []ResolvedBuff, debuffs bool) ([]byte, error) {
	var rows []buffRow
	var shared []sharedCategoryRow
	needsSpells, needsEnums := false, false
	for _, row := range resolved {
		if (row.Scope == buffmanifest.ScopeDebuff) != debuffs {
			continue
		}
		rendered := renderRow(row)
		if rendered.Supported {
			needsSpells = true
			needsEnums = needsEnums || len(row.SkipAuraTypes) > 0
			if row.SharedCategory != "" && !slices.ContainsFunc(shared, func(c sharedCategoryRow) bool { return c.Name == row.SharedCategory }) {
				shared = append(shared, sharedCategoryRow{Var: sharedCategoryVar(row.SharedCategory), Name: row.SharedCategory})
			}
		}
		rows = append(rows, rendered)
	}
	slices.SortFunc(shared, func(a, b sharedCategoryRow) int { return strings.Compare(a.Name, b.Name) })

	name := "buffs"
	tmplStr := TmplStrBuffs
	if debuffs {
		name, tmplStr = "debuffs", TmplStrDebuffs
	}

	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return nil, err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, map[string]any{
		"Rows": rows, "NeedsSpells": needsSpells, "NeedsEnums": needsEnums,
		"SharedCategories": shared,
	}); err != nil {
		return nil, fmt.Errorf("rendering %s: %w", name, err)
	}

	// The template cannot indent a commented-out shell the way gofmt wants, and
	// writing unformatted Go would make every later run diff on whitespace.
	out, err := format.Source(rendered.Bytes())
	if err != nil {
		return nil, fmt.Errorf("generated %s is not valid Go: %w", name, err)
	}
	return out, nil
}

// sharedCategoryRow is one `var <Name>Category = "<Name>"` line: several rows
// name the same shared category, so the generated file declares it once.
type sharedCategoryRow struct {
	Var  string
	Name string
}

// The identifier the generated file gives a shared category, which every row
// that joins it and every caller in sim/core/buffs name.
func sharedCategoryVar(category string) string {
	return category + "Category"
}

func buffScopeField(scope buffmanifest.BuffScope) string {
	switch scope {
	case buffmanifest.ScopeRaid:
		return "raid"
	case buffmanifest.ScopeParty:
		return "party"
	case buffmanifest.ScopeDebuff:
		return "debuffs"
	}
	return "individual"
}

func renderRow(row ResolvedBuff) buffRow {
	out := buffRow{
		Go: row.Go, Field: row.Field, Label: buffLabel(row),
		SpellID: row.SpellID, Kind: row.Kind.String(), Reason: row.Reason, LeftOut: row.LeftOut,
		Supported: row.Supported, HasSpell: row.SpellID != 0,
	}

	// A row whose amounts are worth one item each takes the number of them, so
	// that a party with three of the same staff gets three times the aura.
	if row.Kind == buffmanifest.KindItemCount {
		out.ExtraParams = ", count float64"
	}

	if !row.Supported {
		if out.Reason == "" {
			out.Reason = "no reason given"
		}
		return out
	}

	if row.Category != "" {
		out.Category = row.Category
		out.CategoryVar = row.Go + "Category"
	}
	out.SpellVar = lowerFirst(row.Go) + "Spell"
	out.MetaVar = lowerFirst(row.Go) + "Meta"
	out.MetaFields = buffMetaFields(row, out)
	out.HasValue = len(row.Applied) > 0 || row.Kind == buffmanifest.KindDamageShield
	out.HasCooldown = row.CooldownMs > 0
	out.Constructor = buffConstructor(row, out)
	out.ApplyIf, out.ApplyBody, out.HasApply = buffApply(row)
	return out
}

// The UI label, which is the client's name for the spell unless the manifest
// overrides it.
func buffLabel(row ResolvedBuff) string {
	if row.Label != "" {
		return row.Label
	}
	if row.Name != "" {
		return row.Name
	}
	return row.DBName
}

// The fields of the row's buffs.Meta literal, one per line, leaving out the
// ones at their zero value.
func buffMetaFields(row ResolvedBuff, rendered buffRow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Label: %q,\n", rendered.Label)
	fmt.Fprintf(&b, "Spell: %s,\n", rendered.SpellVar)
	if row.CastSpellID != row.SpellID && row.CooldownMs > 0 {
		fmt.Fprintf(&b, "Cast: spelldata.MustFind(%d),\n", row.CastSpellID)
	}
	if rendered.CategoryVar != "" {
		fmt.Fprintf(&b, "Category: %s,\n", rendered.CategoryVar)
	}
	if row.SharedCategory != "" {
		fmt.Fprintf(&b, "SharedCategory: %s,\n", sharedCategoryVar(row.SharedCategory))
	}
	if row.SingleAura {
		b.WriteString("SingleAura: true,\n")
	}
	if row.TalentRanks > 0 {
		fmt.Fprintf(&b, "Talent: spelldata.Talent(%d, %d),\n", row.TalentSpellID, row.TalentRanks)
		fmt.Fprintf(&b, "TalentEffect: %d,\n", row.TalentPosition)
		if row.TalentApplies == buffmanifest.TalentScalesDuration {
			b.WriteString("TalentScalesDuration: true,\n")
		}
	}
	if len(row.SkipAuraTypes) > 0 {
		names := make([]string, len(row.SkipAuraTypes))
		for i, aura := range row.SkipAuraTypes {
			name, _ := dbcenums.Named(aura)
			names[i] = "dbcenums." + name
		}
		fmt.Fprintf(&b, "SkipAuras: []dbcenums.EffectAuraType{%s},\n", strings.Join(names, ", "))
	}
	if row.FullComboPoints {
		b.WriteString("FullComboPoints: true,\n")
	}
	return b.String()
}

// The call the constructor makes into sim/core/buffs/meta.go.
func buffConstructor(row ResolvedBuff, rendered buffRow) string {
	switch {
	case row.Kind == buffmanifest.KindDamageShield:
		return fmt.Sprintf("return newDamageShield(unit, %s, isPlayer, talentPoints)", rendered.MetaVar)
	case row.Scope == buffmanifest.ScopeDebuff:
		return fmt.Sprintf("return newDebuff(unit, %s, isPlayer, talentPoints)", rendered.MetaVar)
	case row.Kind == buffmanifest.KindItemCount:
		return fmt.Sprintf("return newItemCountBuff(unit, %s, isPlayer, count)", rendered.MetaVar)
	}
	return fmt.Sprintf("return newBuff(unit, %s, isPlayer, talentPoints)", rendered.MetaVar)
}

// The apply block: the condition the proto field is read by, and the call that
// puts the buff on the unit. A kind whose behaviour is a cooldown, a proc or an
// uptime calls a driver of a fixed name that sim/core/buffs/drivers.go declares,
// and so does a row the manifest marks as driven. A driver is handed the whole
// scope message rather than its own field, because a driven buff often reads a
// second one: Grace of Air is 9 seconds long while the party is twisting totems.
func buffApply(row ResolvedBuff) (string, string, bool) {
	unit, field := "char", buffScopeField(row.Scope)
	if row.Scope == buffmanifest.ScopeDebuff {
		unit = "target"
	}
	access := field + "." + row.GoField()

	var cond, points string
	switch row.Proto {
	case buffmanifest.ProtoBool:
		cond, points = access, "0"
	case buffmanifest.ProtoTristate:
		cond = access + " != proto.TristateEffect_TristateEffectMissing"
		points = fmt.Sprintf("core.GetTristateValueInt32(%s, 0, %d)", access, row.TalentRanks)
	case buffmanifest.ProtoInt32, buffmanifest.ProtoDouble:
		cond, points = access+" > 0", "0"
	default:
		return "", "", false
	}

	var body string
	switch {
	case row.Driver, isDriverKind(row.Kind), row.Kind == buffmanifest.KindItemCount:
		scope := field
		if row.Scope == buffmanifest.ScopeDebuff {
			scope = "debuffs, raid"
		}
		body = fmt.Sprintf("drive%s(%s, %s)", row.Go, unit, scope)
	default:
		target := "&char.Unit"
		if row.Scope == buffmanifest.ScopeDebuff {
			target = "target"
		}
		body = fmt.Sprintf("core.MakePermanent(%sAura(%s, false, %s))", row.Go, target, points)
	}
	return cond, body, true
}
