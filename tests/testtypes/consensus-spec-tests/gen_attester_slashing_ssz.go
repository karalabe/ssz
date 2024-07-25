// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// SizeSSZ returns either the static size of the object if fixed == true, or
// the total size otherwise.
func (obj *AttesterSlashing) SizeSSZ(sizer *ssz.Sizer, fixed bool) (size uint32) {
	size = 4 + 4
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(sizer, obj.Attestation1)
	size += ssz.SizeDynamicObject(sizer, obj.Attestation2)

	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *AttesterSlashing) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineDynamicObjectOffset(codec, &obj.Attestation1) // Offset (0) - Attestation1 - 4 bytes
	ssz.DefineDynamicObjectOffset(codec, &obj.Attestation2) // Offset (1) - Attestation2 - 4 bytes

	// Define the dynamic data (fields)
	ssz.DefineDynamicObjectContent(codec, &obj.Attestation1) // Field  (0) - Attestation1 - ? bytes
	ssz.DefineDynamicObjectContent(codec, &obj.Attestation2) // Field  (1) - Attestation2 - ? bytes
}
