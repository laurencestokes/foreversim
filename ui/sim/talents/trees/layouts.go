// Package trees exposes the same talent positions and rank limits to the native sim and the browser.
package trees

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/wowsims/forever/sim/core/proto"
)

//go:embed rogue.json warrior.json shaman.json history/prelaunch.json history/launch.json
var files embed.FS

type Talent struct {
	FieldName string `json:"fieldName"`
	MaxPoints int    `json:"maxPoints"`
}

type Layout struct {
	Current []struct {
		Talents []Talent `json:"talents"`
	}
	previous [][]string
	launch   [][]string
}

func decode(name string, into any) {
	data, err := files.ReadFile(name)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		panic(err)
	}
}

var layouts = func() map[string]*Layout {
	var previous map[string][][]string
	decode("history/prelaunch.json", &previous)
	var launch map[string][][]string
	decode("history/launch.json", &launch)
	current := make(map[string]*Layout, len(previous))
	for class, positions := range previous {
		layout := &Layout{previous: positions, launch: launch[class]}
		decode(class+".json", &layout.Current)
		current["Class"+strings.ToUpper(class[:1])+class[1:]] = layout
	}
	return current
}()

func ForClass(class string) *Layout { return layouts[class] }

func CurrentVersion(class proto.Class) proto.TalentLayout {
	if layout := ForClass(class.String()); layout != nil && layout.launch != nil {
		return proto.TalentLayout_TalentLayoutHotfix
	}
	if ForClass(class.String()) != nil {
		return proto.TalentLayout_TalentLayoutLaunch
	}
	return proto.TalentLayout_TalentLayoutLegacy
}

// An old digit belongs to the talent's field, even when that talent moves in the new tree.
func (l *Layout) Migrate(input string, version proto.TalentLayout) string {
	previous := l.previous
	if version == proto.TalentLayout_TalentLayoutLaunch {
		previous = l.launch
	}
	if len(previous) == 0 {
		return input
	}
	points := map[string]byte{}
	for ti, digits := range strings.Split(input, "-") {
		if ti >= len(previous) {
			break
		}
		for i := 0; i < len(digits) && i < len(previous[ti]); i++ {
			if digits[i] >= '1' && digits[i] <= '9' {
				points[previous[ti][i]] = digits[i] - '0'
			}
		}
	}
	out := make([]string, len(l.Current))
	for ti, tree := range l.Current {
		digits := make([]byte, len(tree.Talents))
		for i, talent := range tree.Talents {
			digits[i] = '0' + min(points[talent.FieldName], byte(talent.MaxPoints))
		}
		out[ti] = strings.TrimRight(string(digits), "0")
	}
	return strings.TrimRight(strings.Join(out, "-"), "-")
}
