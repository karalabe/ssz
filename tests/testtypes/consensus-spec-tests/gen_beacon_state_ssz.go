// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// Cached static size computed on package init.
var staticSizeCacheBeaconState = 8 + 32 + 8 + new(Fork).SizeSSZ() + new(BeaconBlockHeader).SizeSSZ() + 8192*32 + 8192*32 + 4 + new(Eth1Data).SizeSSZ() + 4 + 8 + 4 + 4 + 65536*32 + 8192*8 + 4 + 4 + 1 + new(Checkpoint).SizeSSZ() + new(Checkpoint).SizeSSZ() + new(Checkpoint).SizeSSZ()

// SizeSSZ returns either the static size of the object if fixed == true, or
// the total size otherwise.
func (obj *BeaconState) SizeSSZ(fixed bool) uint32 {
	var size = uint32(staticSizeCacheBeaconState)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticBytes(obj.HistoricalRoots)
	size += ssz.SizeSliceOfStaticObjects(obj.Eth1DataVotes)
	size += ssz.SizeSliceOfStaticObjects(obj.Validators)
	size += ssz.SizeSliceOfUint64s(obj.Balances)
	size += ssz.SizeSliceOfDynamicObjects(obj.PreviousEpochAttestations)
	size += ssz.SizeSliceOfDynamicObjects(obj.CurrentEpochAttestations)

	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *BeaconState) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineUint64(codec, &obj.GenesisTime)                                    // Field  ( 0) -                 GenesisTime -       8 bytes
	ssz.DefineStaticBytes(codec, obj.GenesisValidatorsRoot[:])                   // Field  ( 1) -       GenesisValidatorsRoot -      32 bytes
	ssz.DefineUint64(codec, &obj.Slot)                                           // Field  ( 2) -                        Slot -       8 bytes
	ssz.DefineStaticObject(codec, &obj.Fork)                                     // Field  ( 3) -                        Fork -       ? bytes (Fork)
	ssz.DefineStaticObject(codec, &obj.LatestBlockHeader)                        // Field  ( 4) -           LatestBlockHeader -       ? bytes (BeaconBlockHeader)
	ssz.DefineArrayOfStaticBytes(codec, obj.BlockRoots[:])                       // Field  ( 5) -                  BlockRoots -  262144 bytes
	ssz.DefineArrayOfStaticBytes(codec, obj.StateRoots[:])                       // Field  ( 6) -                  StateRoots -  262144 bytes
	ssz.DefineSliceOfStaticBytesOffset(codec, &obj.HistoricalRoots)              // Offset ( 7) -             HistoricalRoots -       4 bytes
	ssz.DefineStaticObject(codec, &obj.Eth1Data)                                 // Field  ( 8) -                    Eth1Data -       ? bytes (Eth1Data)
	ssz.DefineSliceOfStaticObjectsOffset(codec, &obj.Eth1DataVotes)              // Offset ( 9) -               Eth1DataVotes -       4 bytes
	ssz.DefineUint64(codec, &obj.Eth1DepositIndex)                               // Field  (10) -            Eth1DepositIndex -       8 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &obj.Validators)                 // Offset (11) -                  Validators -       4 bytes
	ssz.DefineSliceOfUint64sOffset(codec, &obj.Balances)                         // Offset (12) -                    Balances -       4 bytes
	ssz.DefineArrayOfStaticBytes(codec, obj.RandaoMixes[:])                      // Field  (13) -                 RandaoMixes - 2097152 bytes
	ssz.DefineArrayOfUint64s(codec, obj.Slashings[:])                            // Field  (14) -                   Slashings -   65536 bytes
	ssz.DefineSliceOfDynamicObjectsOffset(codec, &obj.PreviousEpochAttestations) // Offset (15) -   PreviousEpochAttestations -       4 bytes
	ssz.DefineSliceOfDynamicObjectsOffset(codec, &obj.CurrentEpochAttestations)  // Offset (16) -    CurrentEpochAttestations -       4 bytes
	ssz.DefineStaticBytes(codec, obj.JustificationBits[:])                       // Field  (17) -           JustificationBits -       1 bytes
	ssz.DefineStaticObject(codec, &obj.PreviousJustifiedCheckpoint)              // Field  (18) - PreviousJustifiedCheckpoint -       ? bytes (Checkpoint)
	ssz.DefineStaticObject(codec, &obj.CurrentJustifiedCheckpoint)               // Field  (19) -  CurrentJustifiedCheckpoint -       ? bytes (Checkpoint)
	ssz.DefineStaticObject(codec, &obj.FinalizedCheckpoint)                      // Field  (20) -         FinalizedCheckpoint -       ? bytes (Checkpoint)

	// Define the dynamic data (fields)
	ssz.DefineSliceOfStaticBytesContent(codec, &obj.HistoricalRoots, 16777216)          // Field  ( 7) -             HistoricalRoots - ? bytes
	ssz.DefineSliceOfStaticObjectsContent(codec, &obj.Eth1DataVotes, 2048)              // Field  ( 9) -               Eth1DataVotes - ? bytes
	ssz.DefineSliceOfStaticObjectsContent(codec, &obj.Validators, 1099511627776)        // Field  (11) -                  Validators - ? bytes
	ssz.DefineSliceOfUint64sContent(codec, &obj.Balances, 1099511627776)                // Field  (12) -                    Balances - ? bytes
	ssz.DefineSliceOfDynamicObjectsContent(codec, &obj.PreviousEpochAttestations, 4096) // Field  (15) -   PreviousEpochAttestations - ? bytes
	ssz.DefineSliceOfDynamicObjectsContent(codec, &obj.CurrentEpochAttestations, 4096)  // Field  (16) -    CurrentEpochAttestations - ? bytes
}
