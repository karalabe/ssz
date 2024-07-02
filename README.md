![Obligatory xkcd](http://imgs.xkcd.com/comics/standards.png)

# ssz [![API Reference](https://pkg.go.dev/badge/github.com/karalabe/ssz)](https://pkg.go.dev/github.com/karalabe/ssz?tab=doc)

Package `ssz` provides an opinionated toolkit for working with Ethereum's [Simple Serialize (SSZ)](https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md) format through Go. The primary focus is on code maintainability, only secondarily striving towards raw performance.

***Please note, this repository is a work in progress. The API is unstable and breaking changes will regularly be made. Hashing is not yet implemented. Do not depend on this in publicly available modules.***

## Goals and objectives

- **Elegant API surface:** Binary protocols are low level constructs and writing encoders/decoders entails boilerplate and fumbling with details. Code generators can do a good job in achieving performance, but with a too low level API, the generated code becomes impossible to humanly maintain. That isn't an issue, until you throw something at the generator it cannot understand (e.g. multiplexed types), at which point you'll be in deep pain. By defining an API that is elegant from a dev perspective, we can create maintainable code for the special snowflake types, yet still generate it for the rest of boring types.
- **Reduced redundancies:** The API aims to make the common case easy and the less common case possible. Redundancies in user encoding/decoding code are deliberately avoided to remove subtle bugs (even at a slight hit on performance). If the user's types require some asymmetry, explicit encoding and decoding code paths are still supported.
- **Support existing types:** Serialization libraries often assume the user is going to define a completely new, isolated type-set for all the things they want to encode. That is simply not the case, and to introduce a new encoding library into a pre-existing codebase, it must play nicely with the existing types. That means common Go typing and aliasing patterns should be supported without annotating everything with new methods.
- **Performant, as meaningful:** Encoding/decoding code should be performant, even if we're giving up some of it to cater for the above goals. Language constructs that are known to be slow (e.g. reflection) should be avoided, and code should have performance similar to low level generated ones. That said, a meaningful application of the library will *do* something with the encoded data, which will almost certainly take more time than generating/parsing a binary blob.

## Expectations

Whilst we aim to be a become the SSZ encoder of `go-ethereum` - and more generally, a go-to encoder for all Go applications requiring to work with Ethereum data blobs - there is no guarantee that this outcome will occur. At the present moment, this package is still in the design and experimentation phase and is not ready for a formal proposal.

There are several possible outcomes from this experiment:

- We determine the effort required to implement all current and future SSZ features are not worth it, abandoning this package.
- All the needed features are shipped, but the package is rejected in favor of some other design that is considered superior.
- The API design of this package get merged into some other existing library and this work gets abandoned in its favor.
- The package turns out simple enough, performant enough and popular enough to be accepted into `go-ethereum` beyond a test.
- Some other unforeseen outcome of the infinite possibilities.

## Development

This module is primarily developed by @karalabe (and possibly @fjl and @rjl493456442). For discussions, please reach out on the Ethereum Research Discord server.

## How to use

First up, you need to add the packag eto your ptoject:

```go
go get github.com/karalabe/ssz
```

### Static types

Some data types in Ethereum will only contain a handful of statically sized fields. One such example would be a `Withdrawal` as seen below.

```go
type Address [20]byte

type Withdrawal struct {
    Index     uint64  `ssz-size:"8"`
    Validator uint64  `ssz-size:"8"`
    Address   Address `ssz-size:"20"`
    Amount    uint64  `ssz-size:"8"`
}
```

In order to encode/decode such a (standalone) object via SSZ, it needs to implement the `ssz.Object` interface:

```go
type Object interface {
	// SizeSSZ returns the total size of an SSZ object.
	SizeSSZ() uint32

	// DefineSSZ defines how an object would be encoded/decoded.
	DefineSSZ(codec *Codec)
}
```

- The `SizeSSZ` seems self-explanatory. It returns the total size of the final SSZ, and for static types such as a `Withdrawal`, you need to calculate this by hand (or by a code generator, more on that later).
- The `DefineSSZ` is more involved. It expects you to define what fields, in what order and with what types are going to be encoded. Essentially, it's the serialization format.

```go
func (w *Withdrawal) SizeSSZ() uint32 { return 44 }

func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &w.Index)          // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)      // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineStaticBytes(codec, w.Address[:]) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)         // Field (3) - Amount         -  8 bytes
}
```

- The `DefineXYZ` methods should feel self-explanatory. They spill out what fields to encode in what order and into what types. The interesting tidbit is the addressing of the fields. Since this code is used for *both* encoding and decoding, it needs to be able to instantiate any `nil` fields during decoding, so pointers are needed.
- Another interesting part is that we haven't defined an encoder/decoder for `Address`, rather just sliced it into `[]byte`. It is common in Go world that byte slices or arrays are aliased into various types, so instead of requiring the user to annotate all those tiny utility types, they can just use them directly.

To encode the above `Witness` into an SSZ stream, use either `ssz.Encode` or `ssz.EncodeToBytes`. The former will write into a stream directly (use this in prod), whilst the latter will allocate a new output byte slice (will thrash the GC).

```go
func main() {
	blob, err := ssz.EncodeToBytes(new(Withdrawal))
	if err != nil {
		panic(err)
	}
	fmt.Printf("ssz: %#x\n", blob)
}
```

To decode an SSZ blob, use `ssz.Decode` and `ssz.DecodeFromBytes` with the same disclaimers about allocations. Note, decoding requires knowing the *size* of the SSZ blob in advance. Unfortunately, this is a limitation of the SSZ format.

### Dynamic types

Most data types in Ethereum will contain a cool mix of static and dynamic data fields. Encoding those is much more interesting, yet still proudly simple. One such a data type would be an `ExecutionPayload` as seen below:

```go
type Hash      [32]byte
type LogsBLoom [256]byte

type ExecutionPayload struct {
	ParentHash    Hash          `ssz-size:"32"`
	FeeRecipient  Address       `ssz-size:"20"`
	StateRoot     Hash          `ssz-size:"32"`
	ReceiptsRoot  Hash          `ssz-size:"32"`
	LogsBloom     LogsBLoom     `ssz-size:"256"`
	PrevRandao    Hash          `ssz-size:"32"`
	BlockNumber   uint64        `ssz-size:"8"`
	GasLimit      uint64        `ssz-size:"8"`
	GasUsed       uint64        `ssz-size:"8"`
	Timestamp     uint64        `ssz-size:"8"`
	ExtraData     []byte        `ssz-max:"32"`
	BaseFeePerGas *uint256.Int  `ssz-size:"32"`
	BlockHash     Hash          `ssz-size:"32"`
	Transactions  [][]byte      `ssz-max:"1048576,1073741824"`
	Withdrawals   []*Withdrawal `ssz-max:"16"`
}
```

Do note, we've reused the previously defined `Address` and `Withdrawal` types. You'll need those too to make this part of the code work. The `uint256.Int` type is from the `github.com/holiman/uint256` package.

As before, we need this type to implement `ssz.Object` to make an encoder/decoder, but the code will be more interesting due to the dynamic fields container within.

```go
func (e *ExecutionPayload) SizeSSZ() uint32 {
	// Start out with the static size
	size := uint32(512)

	// Append all the dynamic sizes
	size += ssz.SizeDynamicBytes(e.ExtraData)           // Field (10) - ExtraData    - max 32 bytes (not enforced)
	size += ssz.SizeSliceOfDynamicBytes(e.Transactions) // Field (13) - Transactions - max 1048576 items, 1073741824 bytes each (not enforced)
	size += ssz.SizeSliceOfStaticObjects(e.Withdrawals) // Field (14) - Withdrawals  - max 16 items, 44 bytes each (not enforced)

	return size
}
```

Opposed to the static `Withdrawal` from the previous section, `ExecutionPayload` has both static and dynamic fields, so we can't just return a pre-computed literal number.

- First up, we will still need to know the static size of the object to avoid costly runtime calculations over and over. Just for reference, that would be the size of all the static fields in the object + 4 bytes for each dynamic field (offset encoding). Feel free to verify the number `512` above.
- Secondly, we need to iterate over all the dynamic fields and accumulate all their sizes too.
  - For all the usual Go suspects like slices and arrays of bytes; 2D sliced and arrays of bytes (i.e. `ExtraData` and `Transactions` above), there are helper methods available in the `ssz` package. 
  - For types implementing `ssz.Object` (i.e. one item of `Withdrawals` above), there are again helper methods available to use them as single objects, static array of objects, of dynamic slice of objects. You need to know if the object you encode is static or dynamic though as the encoding changes!

The codec itself is very similar to the static example before, with a few quirks:

```go
func (e *ExecutionPayload) DefineSSZ(codec *ssz.Codec) {
	// Signal to the codec that we have dynamic fields
	defer codec.OffsetDynamics(512)()

	// Enumerate all the fields we need to code
	ssz.DefineStaticBytes(codec, e.ParentHash[:])                                   // Field  ( 0) - ParentHash    -  32 bytes
	ssz.DefineStaticBytes(codec, e.FeeRecipient[:])                                 // Field  ( 1) - FeeRecipient  -  20 bytes
	ssz.DefineStaticBytes(codec, e.StateRoot[:])                                    // Field  ( 2) - StateRoot     -  32 bytes
	ssz.DefineStaticBytes(codec, e.ReceiptsRoot[:])                                 // Field  ( 3) - ReceiptsRoot  -  32 bytes
	ssz.DefineStaticBytes(codec, e.LogsBloom[:])                                    // Field  ( 4) - LogsBloom     - 256 bytes
	ssz.DefineStaticBytes(codec, e.PrevRandao[:])                                   // Field  ( 5) - PrevRandao    -  32 bytes
	ssz.DefineUint64(codec, &e.BlockNumber)                                         // Field  ( 6) - BlockNumber   -   8 bytes
	ssz.DefineUint64(codec, &e.GasLimit)                                            // Field  ( 7) - GasLimit      -   8 bytes
	ssz.DefineUint64(codec, &e.GasUsed)                                             // Field  ( 8) - GasUsed       -   8 bytes
	ssz.DefineUint64(codec, &e.Timestamp)                                           // Field  ( 9) - Timestamp     -   8 bytes
	ssz.DefineDynamicBytes(codec, &e.ExtraData, 32)                                 // Offset (10) - ExtraData     -   4 bytes
	ssz.DefineUint256(codec, &e.BaseFeePerGas)                                      // Field  (11) - BaseFeePerGas -  32 bytes
	ssz.DefineStaticBytes(codec, e.BlockHash[:])                                    // Field  (12) - BlockHash     -  32 bytes
	ssz.DefineSliceOfDynamicBytes(codec, &e.Transactions, 1_048_576, 1_073_741_824) // Offset (13) - Transactions  -   4 bytes
	ssz.DefineSliceOfStaticObjects(codec, &e.Withdrawals, 16)                       // Offset (14) - Withdrawals   -   4 bytes
}
```

- First up, all dynamic objects must start their codec by running `defer codec.OffsetDynamics(size)()` (note, there's an extra `()` at the end).
  - A bit unorthodox, notation, but what happens under the hood is that we're telling the encoder/decoder that there will be `512` bytes of fixed data, after which the dynamic content begins. Telling the codec the size heads-up is needed to allow running the encoder/decoder in a streaming way, without having to skip encoding fields and later backfilling them.
  - The weird `()` is because `codec.OffsetDynamics` actually starts stashing away encountered dynamic fields (they are encoded at the end of the SSZ container) and will encode them when the `defer` runs.
- The `DefineXYZ` methods are used exactly the same as before, only we used more variations this time due to the more complex data structure. You might also note that dynamic fields also pass in size limits that the decoder can enforce.

To encode the above `ExecutionPayload` do just as we have done with the static `Witness` object. Perhaps 

```go
func main() {
	blob, err := ssz.EncodeToBytes(new(ExecutionPayload))
	if err != nil {
		panic(err)
	}
	fmt.Printf("ssz: %#x\n", blob)
}
```

### Asymmetric types

For types defined in perfect isolation - dedicated for SSZ - it's easy to define the fields with the perfect types, and perfect sizes, and perfect everything. Generating or writing an elegant encoder for those, is easy.

In reality, often you'll need to encode/decode types which already exist in a codebase, which might not map so cleanly onto the SSZ defined structure spec you want (e.g. you have one union type of `ExecutionPayload` that contains all the Bellatrix, Capella, Deneb, etc fork fields together) and you want to encode/decode them differently based on the context.

Most SSZ libraries will not permit you to do such a thing. Reflection based libraries *cannot* infer the context in which they should switch encoders and can neither can they represent multiple encodings at the same time. Generator based libraries again have no meaningful way to specify optional fields based on different constraints and contexts. 

The only way to handle such scenarios is to write the encoders by hand, and furthermore, encoding might be dependent on what's in the struct, whilst decoding might be dependent on what's it contained within. Completely asymmetric, so our unified *codec definition* approach from the previous sections cannot work.

For these scenarios, this package has support for asymmetric encoders/decoders, where the caller can independently implement the two paths with their unique quirks.

To avoid having a real-world example's complexity overshadow the point we're trying to make here, we'll just convert the previously demoed `Withdrawal` encoding/decoding from the unified `codec` version to a separate `encoder` and `decoder` version.

```go
func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	codec.DefineEncoder(func(enc *ssz.Encoder) {
		ssz.EncodeUint64(enc, w.Index)           // Field (0) - Index          -  8 bytes
		ssz.EncodeUint64(enc, w.Validator)       // Field (1) - ValidatorIndex -  8 bytes
		ssz.EncodeStaticBytes(enc, w.Address[:]) // Field (2) - Address        - 20 bytes
		ssz.EncodeUint64(enc, w.Amount)          // Field (3) - Amount         -  8 bytes
	})
	codec.DefineDecoder(func(dec *ssz.Decoder) {
		ssz.DecodeUint64(dec, &w.Index)          // Field (0) - Index          -  8 bytes
		ssz.DecodeUint64(dec, &w.Validator)      // Field (1) - ValidatorIndex -  8 bytes
		ssz.DecodeStaticBytes(dec, w.Address[:]) // Field (2) - Address        - 20 bytes
		ssz.DecodeUint64(dec, &w.Amount)         // Field (3) - Amount         -  8 bytes
	})
}
```

- As you can see, we piggie-back on the already existing `ssz.Object`'s `DefineSSZ` method, and do *not* require implementing new functions. This is good because we want to be able to seamlessly use unified or split encoders without having to tell everyone about it.
- Whereas previously we had a bunch of `DefineXYZ` method to enumerate the fields for the unified encoding/decoding, here we replaced them with separate definitions for the encoder and decoder via `codec.DefineEncoder` and `codec.DefineDecoder`.
- The implementation of the encoder and decoder follows the exact same pattern and naming conventions as with the `codec` but instead of operating on a `ssz.Codec` object, we're operating on an `ssz.Encoder`/`ssz.Decoder` objects; and instead of calling methods named `ssz.DefineXYZ`, we're calling methods named `ssz.EncodeXYZ` and `ssz.DecodeXYZ`.
- Perhaps note, the `EncodeXYZ` methods do not take pointers to everything anymore, since they do not require the ability to instantiate the field during operation.

Encoding the above split-encoder `Witness` into an SSZ stream, you use the same thing as before. Everything is seamless.

```go
func main() {
	blob, err := ssz.EncodeToBytes(new(Withdrawal))
	if err != nil {
		panic(err)
	}
	fmt.Printf("ssz: %#x\n", blob)
}
```

### Generated types

TODO

## Performance

The goal of this package is to be close in performance to low level generated encoders, without sacrificing maintainability. It should, however, be significantly faster than runtime reflection encoders.

There are some benchmarks that you can run via `go test ./tests --run=NONE --bench=.`, but no extensive measurement effort was made yet until the APIs are finalized, nor has the package been compared to anything other implementation.
