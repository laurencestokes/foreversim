package database

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"

	_ "github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/tools/database/dbc"
	"github.com/wowsims/forever/tools/tooltip"
)

// Sets the minimum itemlevel that should be considered for this expansions
const MIN_EFFECT_ILVL = 50

// Enchantment IDs at or below this are not generated.
const MIN_ENCHANT_EFFECT_ID = 0

func isGeneratableEnchant(effectID int32) bool {
	return effectID > MIN_ENCHANT_EFFECT_ID
}

type ProcInfo struct {
	Outcome             core.HitOutcome
	Callback            core.AuraCallback
	ProcMask            core.ProcMask
	MaxCumulativeStacks int32
	RequireDamageDealt  bool
	ClassSpellsOnly     bool
	// The listener's Can Proc From Procs attribute: it also hears hits from proc spells.
	CanProcFromProcs bool
	// A "Chance on hit" item effect or a combat enchant, cast by the game off every eligible
	// weapon hit regardless of proc flags. Decided by the trigger type, not by the item slot.
	IsWeaponProc bool
	// An aura proc carrying Aura Is Weapon Proc, which skips hits that suppress weapon procs.
	HonoursWeaponProcSuppression bool
}

// A weapon proc ignores proc-ness and already skips hits that suppress weapon procs, so the two
// aura-side fields would be dead or redundant next to it. Cleared so the generated file states
// one rule per listener.
func (info *ProcInfo) setIsWeaponProc(isWeaponProc bool) {
	info.IsWeaponProc = isWeaponProc
	if isWeaponProc {
		info.CanProcFromProcs = false
		info.HonoursWeaponProcSuppression = false
	}
}

// Entry represents a effect with its Item ID, Spell ID and display name.
type Variant struct {
	ID      int
	SpellID int
	Name    string
}

type Entry struct {
	Variants  []*Variant
	Tooltip   []string
	ProcInfo  ProcInfo
	Supported bool
	// What adds a stack while the window is open, for the trinkets whose effect carries a
	// separate accumulating aura. Nil for everything else.
	StackProcInfo *ProcInfo
	// Set for an on-use whose window accumulates a separate aura, which needs the stacking
	// helper rather than the flat one. Carries what that helper cannot read from the database.
	StackingOnUse *StackingOnUse
	// Set for effects an ignore list deliberately excludes. These emit a comment only, so that
	// skipping them is visible in the generated file rather than silent.
	Skipped bool
	// Set when the proc is resolved from the client's own rows at run time, which is every proc
	// but the two shapes that need more than the rows state: a window that accumulates a second
	// aura, and an effect a hand-written constructor already covers.
	Proc *ProcRouting
	// Set when the effect deals flat damage instead of granting stats. Those resolve no stats, so
	// without this they are dropped before they are ever emitted.
	DealsDamage bool
}

// The literals a stacking on-use needs in the generated call. Everything else - stacks,
// per-stack stats, which aura is which - the helper reads from the database at runtime.
type StackingOnUse struct {
	Name       string
	CooldownMs int32
}

// The two spells an item or enchant proc is resolved from at run time, and what the rows say the
// sim cannot model. The reasons are the ones sim/core/spelldata answers, so the registration the
// generator writes and the listener the sim builds come from the same reading.
type ProcRouting struct {
	TriggerSpellID int
	// Zero where the trigger's own row is the buff.
	BuffSpellID int
	// A "Chance on hit" item effect or a combat enchant, cast by the game off every eligible weapon
	// hit whatever the row's proc flags say.
	IsWeaponProc bool
	// Set where the spell the proc applies deals damage instead of granting an aura, which is a
	// constructor of its own: there is no buff to build.
	Damage bool
	// Empty when the rows state enough to build the listener.
	Unsupported []string
	// What the rows resolve to, for the reader of the generated file.
	Summary string
}

func (r *ProcRouting) Supported() bool {
	return len(r.Unsupported) == 0
}

// Renders the reasons as the generated file states them, one clause per shape.
func (r *ProcRouting) Reason() string {
	return strings.Join(r.Unsupported, "; ")
}

// The rows behind an item effect, by the two ids the sim will look up: the spell carrying the proc
// and the spell it applies.
func routeProc(triggerSpellID int, buffSpellID int, isWeaponProc bool) *ProcRouting {
	routing := &ProcRouting{TriggerSpellID: triggerSpellID, IsWeaponProc: isWeaponProc}
	if buffSpellID != triggerSpellID {
		routing.BuffSpellID = buffSpellID
	}

	trigger := spelldata.Find(int32(triggerSpellID))
	routing.Unsupported = spelldata.ItemProcUnsupported(trigger, isWeaponProc)
	routing.Summary = procSummary(triggerSpellID, trigger, buffSpellID)

	return routing
}

// A proc whose spell deals damage rather than granting an aura. The spell is named separately
// because the client hangs it below the trigger rather than on it.
func (r *ProcRouting) asDamage(damageSpellID int32) {
	r.Damage = true
	r.BuffSpellID = int(damageSpellID)

	if damage := spelldata.Find(damageSpellID); damage == spelldata.Nil {
		r.Unsupported = append(r.Unsupported, "the damage spell has no row in the store")
	} else if damage.DamageEffect() == spelldata.NilEffect {
		r.Unsupported = append(r.Unsupported, "the damage spell's row states no damage")
	}

	r.Summary = procSummary(r.TriggerSpellID, spelldata.Find(int32(r.TriggerSpellID)), r.BuffSpellID)
}

// The buff the proc applies has to last for something: an aura of no duration is one the sim
// refuses to activate, and the client states it on either row.
func (r *ProcRouting) requireABuffDuration() {
	trigger := spelldata.Find(int32(r.TriggerSpellID))
	buff := trigger
	if r.BuffSpellID != 0 {
		buff = spelldata.Find(int32(r.BuffSpellID))
	}

	if buff == spelldata.Nil {
		r.Unsupported = append(r.Unsupported, "the buff has no row in the store")
		return
	}

	if buff.DurationMs == 0 && trigger.DurationMs == 0 {
		r.Unsupported = append(r.Unsupported, "neither row states how long the buff lasts")
	}
}

// What the rows resolve to, as the sim's own constants, so the generated file states the reading
// rather than leaving it to be looked up.
func procSummary(triggerSpellID int, trigger *spelldata.Spell, buffSpellID int) string {
	if trigger == spelldata.Nil {
		return fmt.Sprintf("trigger %d is not in the store", triggerSpellID)
	}

	decoded := core.DecodeProcTypeMask(trigger.ProcFlags, trigger.ProcHint)
	summary := fmt.Sprintf("trigger %d (%s, %s, %s)", trigger.ID,
		procRateSummary(trigger), asCoreCallback(decoded.Callback), asCoreProcMask(decoded.ProcMask))

	if int(trigger.ID) != buffSpellID {
		summary += fmt.Sprintf(" -> buff %d", buffSpellID)
	}

	return summary
}

func procRateSummary(trigger *spelldata.Spell) string {
	switch {
	case trigger.RPPM > 0:
		return fmt.Sprintf("%v ppm", trigger.RPPM)
	case trigger.ProcChanceSource == spelldata.ProcChanceEffectN:
		return fmt.Sprintf("effect %d's chance", trigger.ProcChanceEffect)
	case trigger.ProcChanceSource == spelldata.ProcChanceAlways:
		return "every time"
	case trigger.ProcChanceSource == spelldata.ProcChancePPM:
		return "no stated rate"
	default:
		return fmt.Sprintf("%d%%", trigger.ProcChance)
	}
}

// Group holds a category of effects.
type Group struct {
	Name    string
	Entries []*Entry
}

type MissingItemEffect struct {
	ItemID  int32
	Name    string
	Effects []Variant
}

var missingEffectsMap = map[string]map[int32]MissingItemEffect{
	"EnchantEffects": {},
	"ItemEffects":    {},
}

type EffectParseResult byte

const (
	EffectParseResultInvalid     EffectParseResult = iota // Returned when the effect is invalid for the current parameters
	EffectParseResultUnsupported                          // Returned when the effect could be parsed but is not supported for effect generation
	EffectParseResultSuccess                              // Returned when the effect was parsed successfuly
	EffectParseResultRefused                              // Returned when the effect was parsed, is not supported, and said so in an entry of its own
)

func GenerateEffectsFile(groups []*Group, outFile string, templateString string) error {
	if _, err := os.Stat(outFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to check file %s: %w", outFile, err)
	}

	// Ensure groups and entries are sorted
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	for _, grp := range groups {
		sort.Slice(grp.Entries, func(i, j int) bool {
			if grp.Entries[i].Supported != grp.Entries[j].Supported {
				return !grp.Entries[i].Supported
			}

			return entryOrder(grp.Entries[i], grp.Entries[j])
		})
	}

	funcMap := map[string]any{
		"asCoreCallback": asCoreCallback,
		"asCoreProcMask": asCoreProcMask,
		"asCoreOutcome":  asCoreOutcome,
		"formatStrings":  formatStrings,
	}
	tmpl := template.Must(template.New("effects").Funcs(funcMap).Parse(templateString))

	// An empty generated file must not import anything: gen_db links sim/common, so unused
	// imports in a file it just wrote break the very build the next run needs.
	hasEntries := false
	for _, grp := range groups {
		if len(grp.Entries) > 0 {
			hasEntries = true
			break
		}
	}

	hasStacking := false
	// A registration resolved from the client's rows names no core constant, so a file whose live
	// entries are all of that shape must not import core: gen_db links the sim, and an unused import
	// in a file it just wrote breaks the build the next run needs.
	usesCore := false
	// And nothing at all is imported by a file whose every entry is commented out.
	hasLive := false
	for _, grp := range groups {
		for _, entry := range grp.Entries {
			if entry.StackingOnUse != nil {
				hasStacking = true
			}
			if entry.Skipped || !entry.Supported {
				continue
			}
			hasLive = true
			if entry.Proc == nil {
				usesCore = true
			}
		}
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, map[string]interface{}{"Groups": groups, "HasEntries": hasEntries, "HasStacking": hasStacking, "UsesCore": usesCore, "HasLive": hasLive}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// The template cannot indent commented-out blocks or blank lines the way gofmt wants, so
	// format the result. Otherwise every regeneration reverts whatever formatted the file last
	// and the diff is hundreds of whitespace-only lines.
	out := rendered.Bytes()
	if formatted, err := format.Source(out); err != nil {
		fmt.Printf("WARN: generated %s is not valid Go, writing unformatted: %v\n", outFile, err)
	} else {
		out = formatted
	}

	if err := os.WriteFile(outFile, out, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outFile, err)
	}

	return nil
}

// Whether two entries are resolved from the same rows, which is what lets them be emitted as one
// call with a variant list.
func sameProcRows(a *Entry, b *Entry) bool {
	if a.Proc == nil || b.Proc == nil {
		return a.Proc == b.Proc
	}

	return a.Proc.TriggerSpellID == b.Proc.TriggerSpellID && a.Proc.BuffSpellID == b.Proc.BuffSpellID
}

// A total order over entries. Sorting on the item or enchant ID alone is not one: an item with
// two effects yields two entries sharing that ID, and sort.Slice is not stable, so their order
// in the generated file flipped between runs.
func entryOrder(a *Entry, b *Entry) bool {
	if a.Variants[0].ID != b.Variants[0].ID {
		return a.Variants[0].ID < b.Variants[0].ID
	}
	return a.Variants[0].SpellID < b.Variants[0].SpellID
}

// Escapes a rendered tooltip for use inside a double-quoted TypeScript string. Tooltips
// routinely span several lines, which would otherwise produce a file that does not parse.
func jsString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return s
}

const missingEffectsFileName = "ui/sim/constants/missing_effects_auto_gen.ts"

func GenerateMissingEffectsFile() error {
	if _, err := os.Stat(missingEffectsFileName); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to check file %s: %w", missingEffectsFileName, err)
	}

	funcMap := map[string]any{
		"asCoreCallback": asCoreCallback,
		"asCoreProcMask": asCoreProcMask,
		"asCoreOutcome":  asCoreOutcome,
		"formatStrings":  formatStrings,
		"jsString":       jsString,
	}
	tmpl := template.Must(template.New("missingEffects").Funcs(funcMap).Parse(TmplStrMissingEffects))
	f, err := os.Create(missingEffectsFileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", missingEffectsFileName, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, missingEffectsMap); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func GenerateEnchantEffects(instance *dbc.DBC, db *WowDatabase) {
	groupMapProc := map[string]Group{}
	enchantSpellEffects := map[int]*dbc.SpellEffect{}

	// Several spells can grant the same enchant -- an enchant that was re-taught
	// by a later expansion's recipe has one spell per version, up to five here.
	// Map iteration order is randomized, so keep the lowest spell ID rather than
	// letting whichever one is visited last win and churn the generated file.
	for _, effect := range instance.SpellEffectsById {
		if effect.EffectType == dbcenums.E_ENCHANT_ITEM {
			enchantID := effect.EffectMiscValues[0]
			if existing, ok := enchantSpellEffects[enchantID]; ok && existing.SpellID <= effect.SpellID {
				continue
			}
			enchantSpellEffects[enchantID] = &effect
		}
		enchantID := effect.EffectMiscValues[0]
		if existing, ok := enchantSpellEffects[enchantID]; ok && existing.SpellID <= effect.SpellID {
			continue
		}
		enchantSpellEffects[enchantID] = &effect
	}

	for _, enchant := range instance.Enchants {
		parsed := enchant.ToProto()
		if _, ok := db.Enchants[EnchantToDBKey(parsed)]; !ok {
			continue
		}

		for _, enchantEffect := range parsed.EnchantEffects {
			TryParseEnchantEffect(parsed, enchantEffect, groupMapProc, instance, enchantSpellEffects)
		}
	}

	var procGroups []*Group
	for _, grp := range groupMapProc {
		procGroups = append(procGroups, &grp)
	}

	GenerateEffectsFile(procGroups, "sim/common/forever/enchants_auto_gen.go", TmplStrEnchant)
}

// Names the ignore-list rule that excluded an effect, for the comment emitted in the generated
// file. Returns "" when nothing excludes it.
func ignoredEffectReason(instance *dbc.DBC, effectID int) string {
	for _, effect := range instance.SpellEffectsInOrder(effectID) {
		if params, ok := IgnoreSpellEffectByAuraType[effect.EffectAura]; ok {
			if len(params) == 0 || slices.Contains(params, effect.EffectMiscValues[0]) {
				return fmt.Sprintf("ignored aura type %d", effect.EffectAura)
			}
		}

		if params, ok := IgnoreSpellEffectBySpellEffectType[effect.EffectType]; ok {
			if len(params) == 0 || slices.Contains(params, effect.EffectMiscValues[0]) {
				return fmt.Sprintf("ignored effect type %d", effect.EffectType)
			}
		}
	}

	return ""
}

// Records an effect excluded by an ignore list so the generated file documents it. Kept in its
// own group: variant merging is per-group, so these cannot affect whether a real effect's
// variant set is emitted live or commented.
func storeSkippedEffect(id int32, name string, buffID int32, instance *dbc.DBC, groupMap map[string]Group) {
	grp, exists := groupMap["Skipped"]
	if !exists {
		grp = Group{Name: "Skipped"}
	}

	buffName := instance.Spells[int(buffID)].NameLang
	grp.Entries = append(grp.Entries, &Entry{
		Skipped:  true,
		Variants: []*Variant{{ID: int(id), Name: name, SpellID: int(buffID)}},
		Tooltip: []string{fmt.Sprintf("%s: %q (%d) - %s",
			name, buffName, buffID, ignoredEffectReason(instance, int(buffID)))},
	})
	groupMap["Skipped"] = grp
}

func ItemEffectIsSupported(instance *dbc.DBC, effectID int) bool {
	supported := true
	if effects, ok := instance.SpellEffects[effectID]; ok {
		for _, effect := range effects {
			if params, ok := IgnoreSpellEffectByAuraType[effect.EffectAura]; ok {
				if len(params) == 0 {
					supported = false
					break
				} else {
					if slices.Contains(params, effect.EffectMiscValues[0]) {
						supported = false
					}
				}
			}

			if params, ok := IgnoreSpellEffectBySpellEffectType[effect.EffectType]; ok {
				if len(params) == 0 {
					supported = false
					break
				} else {
					if slices.Contains(params, effect.EffectMiscValues[0]) {
						supported = false
					}
				}
			}
		}
	}
	return supported
}

func GenerateItemEffects(instance *dbc.DBC, db *WowDatabase, itemSources map[int][]*proto.DropSource) {
	groupMapOnUse := map[string]Group{}
	groupMapProc := map[string]Group{}

	// Example loop over your items
	for _, parsed := range db.Items {
		parsed.ItemEffects = dbc.MergeItemEffectsForAllStates(parsed)

		// core.NewItemEffect takes one effect per item and panics on a second, so only an
		// item's first generated effect is written; the rest are listed as missing. Skullflame
		// Shield carries two client effects (18815, 18816) that both resolve to 18817.
		generated := false
		for _, itemEffect := range parsed.ItemEffects {
			if !ItemEffectIsSupported(instance, int(itemEffect.BuffId)) {
				// Commented into the generated file rather than dropped. These are deliberately
				// out of scope - summons, teleports, created items - but an item whose only
				// effect is skipped otherwise vanished with no trace, while a sibling marker
				// aura on the same item got reported as missing instead.
				skippedGroup := groupMapProc
				if itemEffect.GetOnUse() != nil {
					skippedGroup = groupMapOnUse
				}
				storeSkippedEffect(parsed.Id, parsed.Name, itemEffect.BuffId, instance, skippedGroup)
				continue
			}

			if generated {
				ParseTooltipForMissingEffect(parsed, itemEffect, instance, groupMapProc, "Procs")
				continue
			}
			if TryParseOnUseEffect(parsed, itemEffect, instance, groupMapOnUse) == EffectParseResultSuccess {
				generated = true
				continue
			}

			switch TryParseProcEffect(parsed, itemEffect, instance, groupMapProc) {
			case EffectParseResultSuccess:
				generated = true
			case EffectParseResultRefused:
			default:
				ParseTooltipForMissingEffect(parsed, itemEffect, instance, groupMapProc, "Procs")
			}
		}
	}

	// Sorting done in GenerateEffectsFile
	var onUseGroups []*Group
	for _, grp := range groupMapOnUse {
		onUseGroups = append(onUseGroups, &grp)
	}

	// Merge variants
	var procGroups []*Group
	needsStatPostfix := map[string]bool{}
	for _, grp := range groupMapProc {
		newEntries := []*Entry{}
		entryGroupings := map[string]*Entry{}

		// sort entries first to make tooltip generation consistent for variants
		sort.Slice(grp.Entries, func(i, j int) bool {
			return entryOrder(grp.Entries[i], grp.Entries[j])
		})

		for _, entry := range grp.Entries {
			var idx int64 = 0
			added := false

			// Make sure to only group by name and proc mask, each proc mask will create it's own sub group
			// A variant set is emitted as one call, so its members also have to name the same rows:
			// the reissued PvP shields share a name and a buff and carry different triggers.
			for _, group := range entryGroupings {
				if group.Variants[0].Name == entry.Variants[0].Name {
					idx++
					if group.ProcInfo.ProcMask == entry.ProcInfo.ProcMask && sameProcRows(group, entry) {
						group.AddVariant(entry.Variants[0])
						added = true
						break
					}
				}
			}

			if !added {
				groupName := entry.Variants[0].Name
				if idx > 0 {
					needsStatPostfix[groupName] = true
					groupName += "(" + strconv.FormatInt(idx, 10) + ")"
				}

				newEntries = append(newEntries, entry)
				entryGroupings[entry.Variants[0].Name] = entry
			}
		}

		grp.Entries = newEntries
		procGroups = append(procGroups, &grp)
	}

	updateNames := func(entries []*Entry) {
		for _, entry := range entries {
			for _, variant := range entry.Variants {
				if _, ok := needsStatPostfix[variant.Name]; ok {
					item := db.Items[int32(variant.ID)]
					for _, itemEffect := range item.ItemEffects {
						variant.Name += " - " + GetEffectStatString(itemEffect)
					}
				}

				variant.Name += BuildItemDifficultyPostfix(itemSources, variant.ID, instance)
			}
		}
	}

	// Update Item names
	for _, grp := range onUseGroups {
		updateNames(grp.Entries)
	}

	for _, grp := range procGroups {
		updateNames(grp.Entries)
	}

	GenerateEffectsFile(onUseGroups, "sim/common/forever/stat_bonus_cds_auto_gen.go", TmplStrOnUse)
	GenerateEffectsFile(procGroups, "sim/common/forever/stat_bonus_procs_auto_gen.go", TmplStrProc)
}

func GenerateItemEffectRandomPropPoints(instance *dbc.DBC, db *WowDatabase) {
	for id, allocMap := range instance.RandomPropertiesByIlvl {
		ilvl := int32(id)
		if ilvl < core.MinIlvl || ilvl > core.MaxIlvl {
			continue
		}
		db.ItemEffectRandPropPoints[ilvl] = &proto.ItemEffectRandPropPoints{
			Ilvl:           ilvl,
			RandPropPoints: allocMap[proto.ItemQuality_ItemQualityEpic][0],
		}
	}
}

func BuildItemDifficultyPostfix(itemSources map[int][]*proto.DropSource, itemId int, instance *dbc.DBC) string {
	difficultyPostfix := ""
	if sources, ok := itemSources[itemId]; ok {
		name := DifficultyToShortName(sources[0].Difficulty)
		if len(name) > 0 {
			difficultyPostfix += " " + name
		}
	}

	if item, ok := instance.Items[itemId]; ok {
		if len(item.NameDescription) > 0 && item.NameDescription != "Heroic" {
			difficultyPostfix += " (" + item.NameDescription + ")"
		}

		if item.Flags1.Has(dbc.HORDE_SPECIFIC) {
			difficultyPostfix += " (Horde)"
		}

		if item.Flags1.Has(dbc.ALLIANCE_SPECIFIC) {
			difficultyPostfix += " (Alliance)"
		}
	}

	return difficultyPostfix
}

func TryParseProcEffect(parsed *proto.UIItem, itemEffect *proto.ItemEffect, instance *dbc.DBC, groupMapProc map[string]Group) EffectParseResult {
	if itemEffect.GetProc() != nil && parsed.ScalingOptions[0].Ilvl >= MIN_EFFECT_ILVL {
		// Effect was already manually implemented
		if core.HasItemEffect(parsed.Id) {
			return EffectParseResultSuccess
		}

		tooltipString, id := dbc.GetItemEffectSpellTooltip(int(parsed.Id), int(itemEffect.BuffId))
		tooltip, _ := tooltip.ParseTooltip(tooltipString, tooltip.DBCTooltipDataProvider{DBC: instance}, int64(id))

		grp, exists := groupMapProc["Procs"]
		if !exists {
			grp = Group{Name: "Procs"}
		}

		if tooltip != nil {
			renderedTooltip := tooltip.String()
			entry := Entry{Tooltip: strings.Split(renderedTooltip, "\n"), Variants: []*Variant{{ID: int(parsed.Id), Name: parsed.Name, SpellID: int(itemEffect.BuffId)}}}
			entry.ProcInfo, entry.Supported = BuildProcInfo(parsed, int(itemEffect.BuffId), instance, renderedTooltip)

			entry.StackProcInfo = buildStackProcInfo(itemEffect, instance, renderedTooltip)

			// entry.Supported speaks only for the trigger that opens the window, so the stack side
			// has to be refused separately. Without this the template omits StackCallback,
			// attachStackTrigger early-returns on the empty callback and the stat aura activates at
			// zero stacks with no duration of its own - a trinket whose window opens and never gains
			// a stack, worth nothing and reported as implemented. The on-use path refuses the same
			// shape further down.
			if itemEffect.StackingAura != nil && itemEffect.GetStackProc() != nil && entry.StackProcInfo == nil {
				entry.Supported = false
			}

			// A stat-buff proc carries its two spell ids and nothing else: what it hears, how often
			// and for how long are the rows' to say, at run time, through the same decision this
			// reads here - and that decision is the whole of it, rather than the tooltip reading
			// BuildProcInfo does for the shapes below. The two that need more than the rows state
			// stay where they are: a window accumulating a second aura, and an effect with no stats.
			if itemEffect.StackingAura == nil && len(dbc.EffectStats(itemEffect)) > 0 {
				entry.Proc = routeItemProc(parsed, itemEffect)
				if entry.Proc != nil {
					entry.Proc.requireABuffDuration()
					entry.Supported = entry.Proc.Supported()
				}
			}

			// An effect that resolves no stats may still deal flat damage, which is a shape of its
			// own rather than a reason to refuse: there is no buff, so the proc casts the spell the
			// client hangs below its trigger, read from that spell's own row.
			if len(dbc.EffectStats(itemEffect)) == 0 {
				if damage := dbc.ResolveDamageEffect(int(itemEffect.BuffId)); damage != nil {
					entry.Proc = routeItemProc(parsed, itemEffect)
					if entry.Proc != nil {
						entry.Proc.asDamage(int32(damage.SpellID))
						entry.Supported = entry.Proc.Supported()
						entry.DealsDamage = true
					}
				}
			}

			if (len(dbc.EffectStats(itemEffect)) == 0 && !entry.DealsDamage) || !entry.Supported {
				StoreMissingEffect("ItemEffects", parsed.Name, Variant{
					ID:      int(parsed.Id),
					Name:    renderedTooltip,
					SpellID: int(itemEffect.BuffId),
				})

				// A proc the rows themselves refuse is emitted with the reason they gave, so the
				// generated file says why rather than leaving a shapeless commented block behind.
				if entry.Proc != nil && !entry.Proc.Supported() {
					grp.Entries = append(grp.Entries, &entry)
					groupMapProc["Procs"] = grp
					return EffectParseResultRefused
				}

				return EffectParseResultUnsupported
			}

			grp.Entries = append(grp.Entries, &entry)
			groupMapProc["Procs"] = grp

			return EffectParseResultSuccess
		} else {
			return EffectParseResultUnsupported
		}
	}

	// check if the item has any kind of proc as we only support stat proc parsing right now
	if effects, ok := instance.ItemEffectsByParentID[int(parsed.Id)]; ok && parsed.ScalingOptions[0].Ilvl >= MIN_EFFECT_ILVL {
		for _, effect := range effects {
			if SpellHasTriggerEffect(effect.SpellID, instance) {
				return EffectParseResultUnsupported
			}
		}
	}

	return EffectParseResultInvalid
}

// The rows an item effect names: the client's ItemEffect row carries the spell with the proc on it,
// and the shipped entry carries the spell that applies the stats.
func routeItemProc(parsed *proto.UIItem, itemEffect *proto.ItemEffect) *ProcRouting {
	effect := dbc.GetItemEffectForBuffID(int(parsed.Id), int(itemEffect.BuffId))
	if effect == nil {
		return nil
	}

	return routeProc(effect.SpellID, int(itemEffect.BuffId), effect.TriggerType == dbc.ITEM_SPELLTRIGGER_CHANCE_ON_HIT)
}

func TryParseOnUseEffect(parsed *proto.UIItem, itemEffect *proto.ItemEffect, instance *dbc.DBC, groupMap map[string]Group) EffectParseResult {
	// Effect was already manually implemented
	if core.HasItemEffect(parsed.Id) {
		return EffectParseResultSuccess
	}

	if itemEffect.GetOnUse() != nil && parsed.ScalingOptions[0].Ilvl >= MIN_EFFECT_ILVL {
		if itemEffect.GetOnUse().CooldownMs < 0 && itemEffect.GetOnUse().CategoryCooldownMs < 0 {
			return EffectParseResultUnsupported
		}

		groupName := GetEffectStatString(itemEffect)
		grp, exists := groupMap[groupName]
		if !exists {
			grp = Group{Name: groupName}
		}

		entry := &Entry{Variants: []*Variant{{ID: int(parsed.Id), Name: parsed.Name, SpellID: int(itemEffect.BuffId)}}, Supported: true}
		grp.Entries = append(grp.Entries, entry)
		groupMap[groupName] = grp

		// A stacking on-use keeps its stats on the accumulating aura, so the flat check below
		// would call it unsupported, and the flat helper would grant nothing.
		stacking := itemEffect.StackingAura
		if stacking != nil && len(stacking.GetScalingOptions()[0].GetStats()) > 0 {
			entry.StackProcInfo = buildStackProcInfo(itemEffect, instance, "")
			if entry.StackProcInfo == nil {
				entry.Supported = false
				return EffectParseResultUnsupported
			}
			entry.StackingOnUse = &StackingOnUse{
				Name:       parsed.Name,
				CooldownMs: itemEffect.GetOnUse().CooldownMs,
			}
			return EffectParseResultSuccess
		}

		if len(dbc.EffectStats(itemEffect)) == 0 {
			entry.Supported = false
			return EffectParseResultUnsupported
		}

		return EffectParseResultSuccess
	}

	return EffectParseResultInvalid
}

func TryParseEnchantEffect(enchant *proto.UIEnchant, enchantEffect *proto.ItemEffect, groupMapProc map[string]Group, instance *dbc.DBC, enchantSpellEffects map[int]*dbc.SpellEffect) EffectParseResult {
	if (enchantEffect.GetProc() != nil || EnchantHasDummyEffect(enchant, instance)) && isGeneratableEnchant(enchant.EffectId) {

		// Effect was already manually implemented
		if core.HasEnchantEffect(enchant.EffectId) {
			return EffectParseResultSuccess
		}

		if enchantingSpell, ok := enchantSpellEffects[int(enchant.EffectId)]; ok {
			tooltipString := instance.Spells[enchantingSpell.SpellID].Description
			tooltip, _ := tooltip.ParseTooltip(tooltipString, tooltip.DBCTooltipDataProvider{DBC: instance}, int64(enchantingSpell.SpellID))

			grp, exists := groupMapProc["Enchants"]
			if !exists {
				grp = Group{Name: "Enchants"}
			}

			renderedTooltip := tooltip.String()
			entry := Entry{Tooltip: strings.Split(renderedTooltip, "\n"), Variants: []*Variant{{ID: int(enchant.EffectId), Name: enchant.Name, SpellID: int(enchantingSpell.SpellID)}}}
			entry.ProcInfo, entry.Supported = BuildEnchantProcInfo(enchant, instance, renderedTooltip)

			// The same two ids an item proc carries. An enchant's trigger is the spell the client
			// hangs on the enchantment; the buff is what the shipped entry says it applies.
			entry.Proc = routeEnchantProc(enchant, instance)
			if entry.Proc != nil {
				entry.Supported = entry.Proc.Supported()
			}

			grp.Entries = append(grp.Entries, &entry)
			groupMapProc["Enchants"] = grp

			if !entry.Supported {
				StoreMissingEffect("EnchantEffects", enchant.Name, Variant{
					ID:      int(enchant.EffectId),
					Name:    renderedTooltip,
					SpellID: int(enchant.SpellId),
				})
				return EffectParseResultUnsupported
			}

			return EffectParseResultSuccess
		}
	}

	return EffectParseResultInvalid
}

// The rows an enchant's proc is resolved from. An enchant applies its effect through one spell, so
// the trigger is that spell and the buff is whatever the shipped entry names - the same spell again
// where the client grants the stats through it directly.
func routeEnchantProc(enchant *proto.UIEnchant, instance *dbc.DBC) *ProcRouting {
	if enchant.SpellId == 0 {
		return nil
	}

	raw, ok := instance.EnchantsByEffectId[int(enchant.EffectId)]
	isWeaponProc := ok && raw.IsCombatSpell(int(enchant.SpellId))

	buffSpellID := int(enchant.SpellId)
	for _, effect := range enchant.EnchantEffects {
		if effect.GetProc() != nil {
			buffSpellID = int(effect.BuffId)
			break
		}
	}

	routing := routeProc(int(enchant.SpellId), buffSpellID, isWeaponProc)

	if len(enchant.EnchantEffects) == 0 {
		if damage := dbc.ResolveDamageEffect(int(enchant.SpellId)); damage != nil {
			routing.asDamage(int32(damage.SpellID))
			return routing
		}

		routing.Unsupported = append(routing.Unsupported, "the enchant grants neither stats nor damage")
		return routing
	}

	routing.requireABuffDuration()

	return routing
}

func ParseTooltipForMissingEffect(parsed *proto.UIItem, itemEffect *proto.ItemEffect, instance *dbc.DBC, groupMap map[string]Group, groupMapName string) {
	if parsed.ScalingOptions[0].Ilvl >= MIN_EFFECT_ILVL {
		// Effect was already manually implemented
		if core.HasItemEffect(parsed.Id) {
			return
		}

		tooltipString, id := dbc.GetItemEffectSpellTooltip(int(parsed.Id), int(itemEffect.BuffId))
		tooltip, _ := tooltip.ParseTooltip(tooltipString, tooltip.DBCTooltipDataProvider{DBC: instance}, int64(id))

		grp, exists := groupMap[groupMapName]
		if !exists {
			grp = Group{Name: groupMapName}
		}

		if tooltip != nil {
			renderedTooltip := tooltip.String()
			entry := Entry{
				Tooltip:   strings.Split(renderedTooltip, "\n"),
				Supported: false,
				Variants: []*Variant{
					{
						ID:      int(parsed.Id),
						Name:    parsed.Name,
						SpellID: int(itemEffect.BuffId),
					},
				},
			}

			grp.Entries = append(grp.Entries, &entry)
			groupMap[groupMapName] = grp

			// Flavour auras carry no mechanic worth implementing and only add noise to the
			// report. Suppressing the report only, never the group entry: those
			// Supported: false entries take part in the variant grouping that decides whether
			// a whole variant set is emitted live or commented, so dropping one can flip a
			// real registration.
			if _, ignored := IgnoreMissingEffectBySpellID[int(itemEffect.BuffId)]; ignored {
				return
			}

			if len(dbc.EffectStats(itemEffect)) == 0 || !entry.Supported {
				StoreMissingEffect("ItemEffects", parsed.Name, Variant{
					ID:      int(parsed.Id),
					Name:    renderedTooltip,
					SpellID: int(itemEffect.BuffId),
				})
			}
		}
	}
}

// A critical strike named as the trigger. The first clause reads the wording that puts something
// after the crit ("critical strikes have a", "critical hits grant"), which is why the character
// after it may not open "by", "for" or "chance" - those three are how a tooltip states a magnitude
// instead. The second reads the trigger stated from the caster's side, "your critical strikes",
// where the plural is what separates it from the magnitudes: those all read "critical strike
// chance", "critical strike damage" or "critical strike rating", in the singular.
var critMatcher = regexp.MustCompile(`critical ([^\s]+|damage,?)( chance)? [^fbc]|[Yy]our [a-z ]{0,20}critical strikes`)
var pureHealMatcher = regexp.MustCompile(`healing spells`)
var hasHealMatcher = regexp.MustCompile(`heal(ing)?[^,]`)
var hasGenericMatcher = regexp.MustCompile(`a spell`)

// A trigger condition stated as an attack outcome. Deliberately matches the condition clause rather
// than the words themselves: "increases your dodge rating" is a stat on hundreds of items, while
// "when one of your spells is resisted" is a trigger. "After a block" is the third clause the
// client uses, on the Battlegear of Wrath parry (23548); "after you dodge" is not in here because
// the only rows wording it that way are Counterattack's, where it gates the button rather than a
// proc. A miss is not in here either: the one row naming one, 456394, means its own attack missing
// rather than an attack on it.
var outcomeConditionMatcher = regexp.MustCompile(`(?i)when .{0,60}?(is|are) resisted|((each|every) time|when|whenever) you (block|dodge|parry)|after a (block|dodge|parry)`)

// A tooltip stating that the effect only happens sometimes. Where the data pairs that with a 100%
// rate, the real rate is the one thing the data does not carry.
var statedChanceMatcher = regexp.MustCompile(`(?i)chance (to|of|when)|has a chance`)

// What a sentence names as feeding the proc. A clause stating a chance says nothing about a rate
// unless it also says what the chance is rolled on or what it does: Force Reactive Disk's "This also
// has a chance of damaging the shield" is the shield's durability, in a sentence of its own, and the
// -ing is what keeps it apart from the damage a proc deals.
var procTriggerClauseMatcher = regexp.MustCompile(`(?i)melee|ranged|attack|swing|strike|cast|spell|whenever|each time|on hit|block|weapon|damage\b`)

// A chance that is the magnitude rather than the rate: "increases the critical effect chance of your
// Lesser Healing Wave", "grants increased chance to Block", "Chance to trigger Overload increased by
// an additional 5%". The word the effect modifies sits within a clause of the chance either way.
var increasedChanceMatcher = regexp.MustCompile(`(?i)increase[sd]?[^.]{0,40}?chance|chance[^.]{0,30}?increase[sd]?`)

// Where one statement ends and the next begins. Crude on purpose: an abbreviation or a decimal
// splits a sentence in two, and neither changes which half carries the trigger.
var sentenceBreak = regexp.MustCompile(`[.!?](?:\s|$)|\r?\n`)

// Whether the tooltip says the effect only happens sometimes and leaves the rate unsaid. Read from
// the sentence that states the trigger rather than from the whole text: "increases your critical
// strike chance" is a magnitude on hundreds of spells, and a chance in a sentence that names nothing
// the proc fires on is about something else entirely.
func tooltipStatesAnUnknownRate(description string) bool {
	for _, sentence := range sentenceBreak.Split(description, -1) {
		if !statedChanceMatcher.MatchString(sentence) {
			continue
		}
		if !procTriggerClauseMatcher.MatchString(sentence) {
			continue
		}
		if increasedChanceMatcher.MatchString(sentence) {
			continue
		}

		return true
	}

	return false
}

// Wording that names the cast itself as the trigger rather than the spell landing:
// "each time you cast a spell", "chance on successful spellcast", "chance on spell cast".
var castTriggerMatcher = regexp.MustCompile(`(?i)you cast|on spell ?cast|spellcast`)

// A trigger clause restricted to one named ability: "Your Shock spells", "Your Moonfire ability",
// "Your casts of Greater Heal", "Your Shadow Bolt has", "When you cast Flash of Light". The capital
// is what carries the meaning - an unrestricted trigger reads "your spell critical strikes" or
// "each time you cast a spell", with nothing capitalized to name.
// The client writes a conditional list where the item shows one name - Eternal Power's "Your casts
// of $?s2060[Greater Heal]..." - so the "casts of" clause reads the possessive alone and leaves what
// follows to the tooltip renderer. The other two clauses need the capital: "when you cast a spell"
// and "your melee attacks have" name no ability.
var namedAbilityMatcher = regexp.MustCompile(`[Yy]our [A-Z][A-Za-z']*( [A-Z][A-Za-z']*)* (spell|spells|ability|abilities|has|have)` +
	`|[Yy]our casts of` +
	`|[Ww]hen you cast [A-Z]`)

// What core.DecodeProcTypeMask cannot read off the mask: the trigger wording around it. The named
// ability and outcome condition bits are read here as well, for the store's rows, and say nothing
// new to the item generator: BuildSpellProcInfo refuses a tooltip carrying either before it decodes
// anything, so the outcome bit the decode does read never reaches an item.
func procTooltipHints(tooltip string) core.ProcHint {
	var hints core.ProcHint

	if castTriggerMatcher.MatchString(tooltip) {
		hints |= core.ProcHintCastTrigger
	}

	if critMatcher.MatchString(tooltip) {
		hints |= core.ProcHintCrit
	}

	// An unrestricted "a spell" counts as heal evidence too: next to a helpful-spell bit it is the
	// wording of a proc that fires off any spell the character casts, healing included.
	if hasHealMatcher.MatchString(tooltip) || hasGenericMatcher.MatchString(tooltip) {
		hints |= core.ProcHintHeals
	}

	if pureHealMatcher.MatchString(tooltip) {
		hints |= core.ProcHintPureHeal
	}

	if namedAbilityMatcher.MatchString(tooltip) {
		hints |= core.ProcHintNamedAbility
	}

	if outcomeConditionMatcher.MatchString(tooltip) {
		hints |= core.ProcHintOutcomeTaken
	}

	return hints
}

// Derives what adds a stack to an accumulating aura, from the container spell rather than from
// the one that opens the window. buff_id is the container by then: the parser rebases the effect
// onto it precisely because that is where the duration and these proc flags live.
func buildStackProcInfo(itemEffect *proto.ItemEffect, instance *dbc.DBC, tooltip string) *ProcInfo {
	if itemEffect.StackingAura == nil || itemEffect.GetStackProc() == nil {
		return nil
	}

	container, ok := instance.Spells[int(itemEffect.BuffId)]
	if !ok {
		return nil
	}

	info, supported := BuildSpellProcInfo(&container, tooltip, proto.ItemType_ItemTypeUnknown)
	if !supported {
		return nil
	}
	return &info
}

func BuildProcInfo(parsed *proto.UIItem, itemEffectID int, instance *dbc.DBC, tooltip string) (ProcInfo, bool) {
	itemEffect := dbc.GetItemEffectForBuffID(int(parsed.Id), itemEffectID)
	if itemEffect == nil {
		return ProcInfo{}, false
	}

	// if we have multiple spells find the first that has a proc aura assigned
	procId := itemEffect.SpellID
	procSpell, ok := instance.Spells[int(procId)]
	if !ok {
		panic(fmt.Sprintf("Could not find proc aura %d spell for item effect %d.\n", procId, parsed.Id))
	}

	// A "Chance on hit" effect is a weapon proc: the game casts it off the hit itself and never
	// consults a ProcTypeMask, so it takes the on-hit shape whatever the proc spell says.
	isWeaponProc := itemEffect.TriggerType == dbc.ITEM_SPELLTRIGGER_CHANCE_ON_HIT

	itemType := proto.ItemType_ItemTypeUnknown
	if isWeaponProc {
		itemType = proto.ItemType_ItemTypeWeapon
	}

	procInfo, supported := BuildSpellProcInfo(&procSpell, tooltip, itemType)
	procInfo.setIsWeaponProc(isWeaponProc)

	if SpellHasDummyEffect(int(procId), instance) {
		return procInfo, false
	}

	return procInfo, supported
}

func BuildEnchantProcInfo(enchant *proto.UIEnchant, instance *dbc.DBC, tooltip string) (ProcInfo, bool) {
	procSpellID := enchant.SpellId
	if procSpellID == 0 {
		fmt.Printf("WARN: Enchant %d with no spell id", enchant.EffectId)
		return ProcInfo{}, false
	}

	procSpell, ok := instance.Spells[int(procSpellID)]
	if !ok {
		panic(fmt.Sprintf("Could not find proc aura %d spell for item effect %d.\n", procSpellID, enchant.EffectId))
	}

	procInfo, supported := BuildSpellProcInfo(&procSpell, tooltip, enchant.Type)
	raw, ok := instance.EnchantsByEffectId[int(enchant.EffectId)]
	procInfo.setIsWeaponProc(ok && raw.IsCombatSpell(int(procSpellID)))

	if SpellHasDummyEffect(int(procSpellID), instance) {
		return procInfo, false
	}

	return procInfo, supported
}

func BuildSpellProcInfo(procSpell *dbc.Spell, tooltip string, itemType proto.ItemType) (ProcInfo, bool) {
	var info = ProcInfo{
		RequireDamageDealt:  true,
		MaxCumulativeStacks: procSpell.MaxCumulativeStacks,
	}

	requiresOutcome := true
	onHitProc := false

	// On hit proc
	if itemType == proto.ItemType_ItemTypeWeapon {
		onHitProc = true
		info.Callback |= core.CallbackOnSpellHitDealt
		info.ProcMask |= core.ProcMaskUnknown
	}

	if itemType == proto.ItemType_ItemTypeRanged {
		info.Callback |= core.CallbackOnSpellHitDealt
		info.ProcMask |= core.ProcMaskRanged
	}

	if procSpell.OnlyProcsFromClassAbilities() {
		info.ClassSpellsOnly = true
	}

	// A spell-family filter the generated proc cannot reproduce. Two sources of evidence for the
	// same thing: the spell names a family in SpellClassMask, or its tooltip names one. On item
	// procs TBC stores the mask as either nil or all-zero and keeps the real filter server-side,
	// so for those the tooltip is the only evidence there is - "Your Shock spells", "Your Moonfire
	// ability". Generating one anyway would proc it off every spell instead of that one.
	if slices.ContainsFunc(procSpell.SpellClassMask, func(mask int) bool { return mask != 0 }) ||
		namedAbilityMatcher.MatchString(tooltip) {
		return info, false
	}

	// An outcome the client does not record. ProcTypeMask has no dodge, parry, block or resist bit -
	// TrinityCore keeps those in a HitMask of its own - so a tooltip stating one as the trigger
	// condition is the only evidence there is. Eye of Magtheridon fires on a resisted spell, and
	// without this it generates as a 100%-per-cast buff. Crit is deliberately not in here: it is the
	// one outcome the generated shape can express, and critMatcher below assigns it.
	if outcomeConditionMatcher.MatchString(tooltip) {
		return info, false
	}

	// A buff the game spends by charges rather than by time: World Breaker's 900 crit rating is
	// consumed by the next two melee hits, which is ProcCharges 2 on 36111. The generated shape
	// knows only durations, so it would hold the buff for the whole window instead. On-use effects
	// are unaffected - their charges are uses of the item, and they do not come through here.
	if procSpell.ProcCharges > 0 {
		return info, false
	}

	// The bit table, decoded in core so the sim reads the same mask the same way. Bits the decode
	// does not model come back in Unsupported and are dropped here rather than refused: a mask
	// naming one next to bits that do model something still generates a listener, and that
	// listener is deliberately the narrower trigger: it hears the hits the sim knows and stays
	// silent on the rest.
	//
	// One rule of the bit table cannot live in core. The heal branch's cleanup for a helpful-only
	// mask also has to clear the OnSpellHitDealt seeded above for a ranged item, which the decode
	// cannot see and this merge cannot undo. No ranged item carries a helpful-only mask with a
	// healing tooltip, and the pureHeal strip below still covers the "healing spells" wording.
	if !onHitProc && len(procSpell.ProcTypeMask) > 0 {
		decoded := core.DecodeProcTypeMask(
			[2]uint32{uint32(procSpell.ProcTypeMask[0]), uint32(procSpell.ProcTypeMask[1])},
			procTooltipHints(tooltip),
		)

		info.Callback |= decoded.Callback
		info.ProcMask |= decoded.ProcMask
		info.RequireDamageDealt = decoded.RequireDamageDealt
		info.Outcome = decoded.Outcome
		requiresOutcome = false
	}

	// Proc-ness is a flag on the listener, not a hit kind in the mask. ProcMaskSpellDamageProc is
	// deliberately never emitted: that is the weapon-imbue shape (Flametongue, rogue poisons), kept
	// a distinct hit kind so that Shiffar's Nexus-Horn, Wrath of Cenarius and Robe of the Elder
	// Scribes proc off Ignite, Elemental Overload and Hurricane's DoT rather than off imbue crits.
	info.CanProcFromProcs = procSpell.CanProcFromProcs()
	info.HonoursWeaponProcSuppression = procSpell.IsWeaponProcAura()

	if requiresOutcome {
		if critMatcher.MatchString(tooltip) {
			info.Outcome = core.OutcomeCrit
		} else {
			info.Outcome = core.OutcomeLanded
		}
	}

	// check for pure healing spell
	if pureHealMatcher.MatchString(tooltip) {
		info.Callback &= ^core.CallbackOnSpellHitDealt
		info.Callback &= ^core.CallbackOnPeriodicDamageDealt
	}

	// A trigger with no callback never fires, and factory_ProcStatBonusEffect returns early on
	// exactly that, so such an effect cannot be generated. The test used to also require an
	// empty Outcome and ProcMask, which it can never have: when requiresOutcome is true the
	// Outcome is set a few lines up, and when it is false the whole term is false, so nothing
	// was ever refused here and empty-callback effects were emitted as live registrations that
	// silently did nothing.
	return info, info.Callback != core.CallbackEmpty
}

func StoreMissingEffect(effectType string, name string, variant Variant) {
	if missingEffectsMap[effectType] == nil {
		missingEffectsMap[effectType] = map[int32]MissingItemEffect{}
	}
	id := int32(variant.ID)
	if missingEffectsMap[effectType][id].Effects == nil {
		missingEffectsMap[effectType][id] = MissingItemEffect{
			ItemID:  id,
			Name:    name,
			Effects: []Variant{},
		}
	}
	itemEntry := missingEffectsMap[effectType][id]
	haveEffect := false
	for _, effect := range itemEntry.Effects {
		if effect.SpellID == variant.SpellID {
			haveEffect = true
			break
		}
	}
	if haveEffect {
		return
	}

	itemEntry.Effects = append(
		itemEntry.Effects,
		variant,
	)
	missingEffectsMap[effectType][id] = itemEntry
}

func asCoreCallback(callback core.AuraCallback) string {
	callbacks := []string{}
	for i := range 32 {
		callbackFlag := core.AuraCallback(1 << i)
		if callbackFlag >= core.CallbackLast {
			break
		}

		if callback.Matches(callbackFlag) {
			callbacks = append(callbacks, "core."+callbackFlag.String())
		}
	}

	if len(callbacks) == 0 {
		return "core.CallbackEmpty"
	}

	return strings.Join(callbacks, " | ")
}

func asCoreProcMask(procMask core.ProcMask) string {
	procs := []string{}
	for i := range 32 {
		procFlag := core.ProcMask(1 << i)
		if procFlag >= core.ProcMaskLast {
			break
		}

		if procMask.Matches(procFlag) {
			procs = append(procs, "core."+procFlag.String())
		}
	}

	if len(procs) == 0 {
		return "core.ProcMaskUnknown"
	}
	return strings.Join(procs, " | ")
}

func asCoreOutcome(outcome core.HitOutcome) string {
	if outcome == core.OutcomeCrit {
		return "core.OutcomeCrit"
	}

	if outcome.Matches(core.OutcomeLanded) {
		return "core.OutcomeLanded"
	}

	return "core.OutcomeEmpty"
}

func (entry *Entry) AddVariant(variant *Variant) {
	entry.Variants = append(entry.Variants, variant)
	sort.Slice(entry.Variants, func(i, j int) bool {
		return entry.Variants[i].ID < entry.Variants[j].ID
	})
}
