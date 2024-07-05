package consensus_spec_tests

import "github.com/karalabe/ssz"

type BeaconBlock struct {
	Slot          Slot
	ProposerIndex uint64
	ParentRoot    Hash
	StateRoot     Hash
	Body          *BeaconBlockBody
}

func (b *BeaconBlock) SizeSSZ(fixed bool) uint32 {
	size := uint32(84)
	if !fixed {
		size += ssz.SizeDynamicObject(b.Body)
	}
	return size
}
func (b *BeaconBlock) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &b.Slot)
	ssz.DefineUint64(codec, &b.ProposerIndex)
	ssz.DefineStaticBytes(codec, b.ParentRoot[:])
	ssz.DefineStaticBytes(codec, b.StateRoot[:])
	ssz.DefineDynamicObjectOffset(codec, &b.Body)

	ssz.DefineDynamicObjectContent(codec, &b.Body)
}
