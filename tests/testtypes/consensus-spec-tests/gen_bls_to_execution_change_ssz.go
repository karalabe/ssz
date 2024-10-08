// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// SizeSSZ returns the total size of the static ssz object.
func (obj *BLSToExecutionChange) SizeSSZ(sizer *ssz.Sizer) uint32 {
	return 8 + 48 + 20
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *BLSToExecutionChange) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &obj.ValidatorIndex)          // Field  (0) -     ValidatorIndex -  8 bytes
	ssz.DefineStaticBytes(codec, &obj.FromBLSPubKey)      // Field  (1) -      FromBLSPubKey - 48 bytes
	ssz.DefineStaticBytes(codec, &obj.ToExecutionAddress) // Field  (2) - ToExecutionAddress - 20 bytes
}
