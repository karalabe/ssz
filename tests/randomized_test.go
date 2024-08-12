package tests

import (
	"encoding/json"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
	types "github.com/karalabe/ssz/tests/testtypes/consensus-spec-tests"
	"github.com/prysmaticlabs/go-bitfield"
)

func roll(n int, r *rand.Rand) int {
	k := r.Intn(n)
	if k%2 == 0 {
		return 0
	}
	return k
}

func rbytes(n int, r *rand.Rand) []byte {
	rbs := make([]byte, n)
	k := r.Intn(n)
	if k%2 == 0 {
		r.Read(rbs)
	}
	return rbs
}

type BbbDeneb struct {
	types.BeaconBlockBodyDeneb
}

func (b *BbbDeneb) Generate(r *rand.Rand, _ int) reflect.Value {
	b = &BbbDeneb{}
	b.Eth1Data = &types.Eth1Data{
		DepositRoot:  types.Hash(rbytes(32, r)),
		DepositCount: r.Uint64(),
		BlockHash:    types.Hash(rbytes(32, r)),
	}
	b.Graffiti = [32]byte(rbytes(32, r))
	k := roll(16, r)
	b.ProposerSlashings = make([]*types.ProposerSlashing, k)
	for i := 0; i < k; i++ {
		header := types.BeaconBlockHeader{
			Slot:          r.Uint64(),
			ProposerIndex: r.Uint64(),
			ParentRoot:    types.Hash(rbytes(32, r)),
			StateRoot:     types.Hash(rbytes(32, r)),
			BodyRoot:      types.Hash(rbytes(32, r)),
		}
		b.ProposerSlashings[i] = &types.ProposerSlashing{
			Header1: &types.SignedBeaconBlockHeader{
				Header:    &header,
				Signature: [96]byte(rbytes(96, r)),
			},
			Header2: &types.SignedBeaconBlockHeader{
				Header:    &header,
				Signature: [96]byte(rbytes(96, r)),
			},
		}
	}
	k = roll(2048, r)
	attestationIndices := make([]uint64, k)
	for i := 0; i < k; i++ {
		attestationIndices[i] = r.Uint64()
	}
	k = roll(2, r)
	b.AttesterSlashings = make([]*types.AttesterSlashing, k)
	for i := 0; i < k; i++ {
		b.AttesterSlashings[i] = &types.AttesterSlashing{
			Attestation1: &types.IndexedAttestation{
				AttestationIndices: attestationIndices,
				Data: &types.AttestationData{
					Slot:            types.Slot(r.Uint64()),
					Index:           r.Uint64(),
					BeaconBlockHash: types.Hash(rbytes(32, r)),
					Source: &types.Checkpoint{
						Epoch: r.Uint64(),
						Root:  types.Hash(rbytes(32, r)),
					},
					Target: &types.Checkpoint{
						Epoch: r.Uint64(),
						Root:  types.Hash(rbytes(32, r)),
					},
				},
				Signature: [96]byte(rbytes(96, r)),
			},
			Attestation2: &types.IndexedAttestation{
				AttestationIndices: attestationIndices,
				Data: &types.AttestationData{
					Slot:            types.Slot(r.Uint64()),
					Index:           r.Uint64(),
					BeaconBlockHash: types.Hash(rbytes(32, r)),
					Source: &types.Checkpoint{
						Epoch: r.Uint64(),
						Root:  types.Hash(rbytes(32, r)),
					},
					Target: &types.Checkpoint{
						Epoch: r.Uint64(),
						Root:  types.Hash(rbytes(32, r)),
					},
				},
				Signature: [96]byte(rbytes(96, r)),
			},
		}
	}
	k = roll(128, r)
	b.Attestations = make([]*types.Attestation, k)
	for i := 0; i < k; i++ {
		b.Attestations[i] = &types.Attestation{
			AggregationBits: bitfield.NewBitlist(uint64(roll(2048, r))),
			Data: &types.AttestationData{
				Slot:            types.Slot(r.Uint64()),
				Index:           r.Uint64(),
				BeaconBlockHash: types.Hash(rbytes(32, r)),
				Source: &types.Checkpoint{
					Epoch: r.Uint64(),
					Root:  types.Hash(rbytes(32, r)),
				},
				Target: &types.Checkpoint{
					Epoch: r.Uint64(),
					Root:  types.Hash(rbytes(32, r)),
				},
			},
			Signature: [96]byte(rbytes(96, r)),
		}
	}
	k = roll(16, r)
	b.Deposits = make([]*types.Deposit, k)
	for i := 0; i < k; i++ {
		var proof [33][32]byte
		for j := 0; j < len(proof); j++ {
			proof[j] = [32]byte(rbytes(32, r))
		}
		b.Deposits[i] = &types.Deposit{
			Proof: proof,
			Data: &types.DepositData{
				Pubkey:                [48]byte(rbytes(48, r)),
				WithdrawalCredentials: [32]byte(rbytes(32, r)),
				Amount:                r.Uint64(),
				Signature:             [96]byte(rbytes(96, r)),
			},
		}
	}
	k = roll(16, r)
	b.VoluntaryExits = make([]*types.SignedVoluntaryExit, k)
	for i := 0; i < k; i++ {
		b.VoluntaryExits[i] = &types.SignedVoluntaryExit{
			Exit: &types.VoluntaryExit{
				Epoch:          r.Uint64(),
				ValidatorIndex: r.Uint64(),
			},
			Signature: [96]byte(rbytes(96, r)),
		}
	}
	b.SyncAggregate = &types.SyncAggregate{
		SyncCommiteeBits:      [64]byte(rbytes(64, r)),
		SyncCommiteeSignature: [96]byte(rbytes(96, r)),
	}
	k = roll(10, r) // 1048576 too big
	txs := make([][]byte, k)
	for i := 0; i < k; i++ {
		txs[i] = rbytes(1024, r) // 1073741824 too big
	}
	bg, _ := uint256.FromBig(big.NewInt(int64(r.Uint64())))
	k = roll(16, r)
	withdrawals := make([]*types.Withdrawal, k)
	for i := 0; i < k; i++ {
		withdrawals[i] = &types.Withdrawal{
			Index:     r.Uint64(),
			Validator: r.Uint64(),
			Address:   types.Address(rbytes(20, r)),
			Amount:    r.Uint64(),
		}
	}
	b.ExecutionPayload = &types.ExecutionPayloadDeneb{
		ParentHash:    types.Hash(rbytes(32, r)),
		FeeRecipient:  types.Address(rbytes(20, r)),
		StateRoot:     types.Hash(rbytes(32, r)),
		ReceiptsRoot:  types.Hash(rbytes(32, r)),
		LogsBloom:     types.LogsBloom(rbytes(256, r)),
		PrevRandao:    types.Hash(rbytes(32, r)),
		BlockNumber:   r.Uint64(),
		GasLimit:      r.Uint64(),
		GasUsed:       r.Uint64(),
		Timestamp:     r.Uint64(),
		ExtraData:     rbytes(32, r),
		BaseFeePerGas: bg,
		BlockHash:     types.Hash(rbytes(32, r)),
		Transactions:  txs,
		Withdrawals:   withdrawals,
		BlobGasUsed:   r.Uint64(),
		ExcessBlobGas: r.Uint64(),
	}
	k = roll(16, r)
	b.BlsToExecutionChanges = make([]*types.SignedBLSToExecutionChange, k)
	for i := 0; i < k; i++ {
		b.BlsToExecutionChanges[i] = &types.SignedBLSToExecutionChange{
			Message: &types.BLSToExecutionChange{
				ValidatorIndex:     r.Uint64(),
				FromBLSPubKey:      [48]byte(rbytes(48, r)),
				ToExecutionAddress: [20]byte(rbytes(20, r)),
			},
			Signature: [96]byte(rbytes(96, r)),
		}
	}
	k = roll(4096, r)
	b.BlobKzgCommitments = make([][48]byte, k)
	for i := 0; i < k; i++ {
		b.BlobKzgCommitments[i] = [48]byte(rbytes(48, r))
	}

	return reflect.ValueOf(b)
}

func pprint(o any) string {
	s, _ := json.MarshalIndent(o, "", "\t")
	return string(s)
}

func TestSSZRoundTripBeaconBodyDeneb(t *testing.T) {
	f := func(body *BbbDeneb) bool {
		bz := make([]byte, body.SizeSSZ(false))
		if err := ssz.EncodeToBytes(bz, body); err != nil {
			t.Log("Serialize: could not serialize body --", err)
			return false
		}
		destBody := &BbbDeneb{}
		if err := ssz.DecodeFromBytes(bz, destBody); err != nil {
			t.Log("Deserialize: could not deserialize back the serialized body --", err)
			return false
		}

		if !reflect.DeepEqual(body, destBody) {
			t.Log("Deserialize: deserialized body different than former body after serialization")
			t.Log(pprint(body))
			t.Log("***********")
			t.Log(pprint(destBody))
			return false
		}

		destBz := make([]byte, destBody.SizeSSZ(false))
		if err := ssz.EncodeToBytes(destBz, destBody); err != nil {
			t.Log("Serialize: could not serialize back the body after deserialization --", err)
			return false
		}

		if !reflect.DeepEqual(bz, destBz) {
			t.Log("Serialize: serialized body different after a serialization-deserialization-serialization trip")
			return false
		}
		return true
	}

	c := quick.Config{MaxCount: 100000}
	if err := quick.Check(f, &c); err != nil {
		t.Error(err)
	}
}

var concurrencyThreshold uint64 = 65536

type Container struct {
	Withdrawals []*types.Withdrawal
}

func (c *Container) SizeSSZ() uint32 {
	return ssz.SizeSliceOfStaticObjects(c.Withdrawals)
}

func (c *Container) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineSliceOfStaticObjectsOffset(codec, &c.Withdrawals, concurrencyThreshold)
	ssz.DefineSliceOfStaticObjectsContent(codec, &c.Withdrawals, concurrencyThreshold)
}

func (c *Container) Generate(r *rand.Rand, size int) reflect.Value {
	withdrawals := make([]*types.Withdrawal, uint32(concurrencyThreshold)/(&types.Withdrawal{}).SizeSSZ()+1)
	for i := 0; i < len(withdrawals); i++ {
		withdrawals[i] = &types.Withdrawal{
			Index:     r.Uint64(),
			Validator: r.Uint64(),
			Address:   types.Address(rbytes(20, r)),
			Amount:    r.Uint64(),
		}
	}
	c = &Container{Withdrawals: withdrawals}

	return reflect.ValueOf(c)
}

func TestHashConcurrent(t *testing.T) {
	f := func(c *Container) bool {
		htrSeq := ssz.HashSequential(c)
		htrC := ssz.HashConcurrent(c)
		if !reflect.DeepEqual(htrSeq, htrC) {
			t.Log("Sequential hash != Concurrent hash")
			t.Log(pprint(c))
			return false
		}
		return true
	}

	c := quick.Config{MaxCount: 100000}
	if err := quick.Check(f, &c); err != nil {
		t.Error(err)
	}
}
