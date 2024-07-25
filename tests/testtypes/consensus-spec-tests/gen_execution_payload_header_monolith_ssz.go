// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// SizeSSZ returns either the static size of the object if fixed == true, or
// the total size otherwise.
func (obj *ExecutionPayloadHeaderMonolith) SizeSSZ(sizer *ssz.Sizer, fixed bool) (size uint32) {
	size = 32 + 20 + 32 + 32 + 256 + 32 + 8 + 8 + 8 + 8 + 4 + 32 + 32 + 32
	if sizer.Fork() >= ssz.ForkCapella {
		size += 32
	}
	if sizer.Fork() >= ssz.ForkDeneb {
		size += 8 + 8
	}
	if fixed {
		return size
	}
	size += ssz.SizeDynamicBytes(sizer, obj.ExtraData)

	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *ExecutionPayloadHeaderMonolith) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineStaticBytes(codec, &obj.ParentHash)           // Field  ( 0) -       ParentHash -  32 bytes
	ssz.DefineStaticBytes(codec, &obj.FeeRecipient)         // Field  ( 1) -     FeeRecipient -  20 bytes
	ssz.DefineStaticBytes(codec, &obj.StateRoot)            // Field  ( 2) -        StateRoot -  32 bytes
	ssz.DefineStaticBytes(codec, &obj.ReceiptsRoot)         // Field  ( 3) -     ReceiptsRoot -  32 bytes
	ssz.DefineStaticBytes(codec, &obj.LogsBloom)            // Field  ( 4) -        LogsBloom - 256 bytes
	ssz.DefineStaticBytes(codec, &obj.PrevRandao)           // Field  ( 5) -       PrevRandao -  32 bytes
	ssz.DefineUint64(codec, &obj.BlockNumber)               // Field  ( 6) -      BlockNumber -   8 bytes
	ssz.DefineUint64(codec, &obj.GasLimit)                  // Field  ( 7) -         GasLimit -   8 bytes
	ssz.DefineUint64(codec, &obj.GasUsed)                   // Field  ( 8) -          GasUsed -   8 bytes
	ssz.DefineUint64(codec, &obj.Timestamp)                 // Field  ( 9) -        Timestamp -   8 bytes
	ssz.DefineDynamicBytesOffset(codec, &obj.ExtraData, 32) // Offset (10) -        ExtraData -   4 bytes
	ssz.DefineStaticBytes(codec, &obj.BaseFeePerGas)        // Field  (11) -    BaseFeePerGas -  32 bytes
	ssz.DefineStaticBytes(codec, &obj.BlockHash)            // Field  (12) -        BlockHash -  32 bytes
	ssz.DefineStaticBytes(codec, &obj.TransactionsRoot)     // Field  (13) - TransactionsRoot -  32 bytes
	if codec.Fork() >= ssz.ForkCapella {
		ssz.DefineStaticBytes(codec, &obj.WithdrawalRoot) // Field  (14) -   WithdrawalRoot -  32 bytes
	}
	if codec.Fork() >= ssz.ForkDeneb {
		ssz.DefineUint64(codec, &obj.BlobGasUsed)   // Field  (15) -      BlobGasUsed -   8 bytes
		ssz.DefineUint64(codec, &obj.ExcessBlobGas) // Field  (16) -    ExcessBlobGas -   8 bytes
	}

	// Define the dynamic data (fields)
	ssz.DefineDynamicBytesContent(codec, &obj.ExtraData, 32) // Field  (10) -        ExtraData - ? bytes
}
