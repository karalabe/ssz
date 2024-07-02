package consensus_spec_tests

import "github.com/karalabe/ssz"

type BeaconBlock struct {
	Slot          Slot
	ProposerIndex uint64
	ParentRoot    Hash
	StateRoot     Hash
	Body          *BeaconBlockBody
}

func (b *BeaconBlock) SizeSSZ() uint32 {
	size := uint32(84)
	size += ssz.SizeDynamicObject(b.Body)
	return size
}
func (b *BeaconBlock) DefineSSZ(codec *ssz.Codec) {
	codec.OffsetDynamics(84)
	defer codec.FinishDynamics()

	ssz.DefineUint64(codec, &b.Slot)
	ssz.DefineUint64(codec, &b.ProposerIndex)
	ssz.DefineStaticBytes(codec, b.ParentRoot[:])
	ssz.DefineStaticBytes(codec, b.StateRoot[:])
	ssz.DefineDynamicObject(codec, &b.Body)
}
