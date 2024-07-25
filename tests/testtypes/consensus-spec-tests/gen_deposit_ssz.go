// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// Cached static size computed on package init.
var staticSizeCacheDeposit = ssz.PrecomputeStaticSizeCache((*Deposit)(nil))

// SizeSSZ returns the total size of the static ssz object.
func (obj *Deposit) SizeSSZ(sizer *ssz.Sizer) uint32 {
	if fork := int(sizer.Fork()); fork < len(staticSizeCacheDeposit) {
		return staticSizeCacheDeposit[fork]
	}
	return 33*32 + ssz.Size((*DepositData)(nil))
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *Deposit) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUnsafeArrayOfStaticBytes(codec, obj.Proof[:]) // Field  (0) - Proof - 1056 bytes
	ssz.DefineStaticObject(codec, &obj.Data)                // Field  (1) -  Data -    ? bytes (DepositData)
}
