// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// Cached static size computed on package init.
var staticSizeCacheSignedBLSToExecutionChange = ssz.PrecomputeStaticSizeCache((*SignedBLSToExecutionChange)(nil))

// SizeSSZ returns the total size of the static ssz object.
func (obj *SignedBLSToExecutionChange) SizeSSZ(sizer *ssz.Sizer) uint32 {
	if fork := int(sizer.Fork()); fork < len(staticSizeCacheSignedBLSToExecutionChange) {
		return staticSizeCacheSignedBLSToExecutionChange[fork]
	}
	return ssz.Size((*BLSToExecutionChange)(nil)) + 96
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *SignedBLSToExecutionChange) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticObject(codec, &obj.Message)  // Field  (0) -   Message -  ? bytes (BLSToExecutionChange)
	ssz.DefineStaticBytes(codec, &obj.Signature) // Field  (1) - Signature - 96 bytes
}
