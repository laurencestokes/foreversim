package rogue

import (
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Saved rotations can carry the rank labels from the four-rank Mutilate ladder.
// The surviving spells keep their IDs and each moves down one rank; the removed first rank
// names the current first rank. Current rank filters pass through unchanged.
func normalizeMutilateRanks(message protoreflect.Message) {
	if id, ok := message.Interface().(*proto.ActionID); ok {
		switch {
		case id.GetSpellId() == 1310707:
			id.RawId = &proto.ActionID_SpellId{SpellId: 399956}
			id.Rank = 1
		case id.GetSpellId() == 399956 && id.Rank == 2,
			id.GetSpellId() == 1241582 && id.Rank == 3,
			id.GetSpellId() == 1241584 && id.Rank == 4:
			id.Rank--
		}
		return
	}
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case field.IsList() && field.Message() != nil:
			for i := 0; i < value.List().Len(); i++ {
				normalizeMutilateRanks(value.List().Get(i).Message())
			}
		case field.Message() != nil && !field.IsMap():
			normalizeMutilateRanks(value.Message())
		}
		return true
	})
}
