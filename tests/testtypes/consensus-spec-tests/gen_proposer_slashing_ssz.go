// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// Cached static size computed on package init.
var staticSizeCacheProposerSlashing = ssz.PrecomputeStaticSizeCache((*ProposerSlashing)(nil))

// SizeSSZ returns the total size of the static ssz object.
func (obj *ProposerSlashing) SizeSSZ(sizer *ssz.Sizer) (size uint32) {
	if fork := int(sizer.Fork()); fork < len(staticSizeCacheProposerSlashing) {
		return staticSizeCacheProposerSlashing[fork]
	}
	size = (*SignedBeaconBlockHeader)(nil).SizeSSZ(sizer) + (*SignedBeaconBlockHeader)(nil).SizeSSZ(sizer)
	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *ProposerSlashing) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticObject(codec, &obj.Header1) // Field  (0) - Header1 - ? bytes (SignedBeaconBlockHeader)
	ssz.DefineStaticObject(codec, &obj.Header2) // Field  (1) - Header2 - ? bytes (SignedBeaconBlockHeader)
}
