![Obligatory xkcd](http://imgs.xkcd.com/comics/standards.png)

# ssz [![API Reference](https://pkg.go.dev/badge/github.com/karalabe/ssz)](https://pkg.go.dev/github.com/karalabe/ssz?tab=doc)

Package `ssz` provides a zero-allocation, opinionated toolkit for working with Ethereum's [Simple Serialize (SSZ)](https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md) format through Go. The primary focus is on code maintainability, only secondarily striving towards raw performance.

***Please note, this repository is a work in progress. The API is unstable and breaking changes will regularly be made. Hashing is not yet implemented. Do not depend on this in publicly available modules.***

## Goals and objectives

- **Elegant API surface:** Binary protocols are low level constructs and writing encoders/decoders entails boilerplate and fumbling with details. Code generators can do a good job in achieving performance, but with a too low level API, the generated code becomes impossible to humanly maintain. That isn't an issue, until you throw something at the generator it cannot understand (e.g. multiplexed types), at which point you'll be in deep pain. By defining an API that is elegant from a dev perspective, we can create maintainable code for the special snowflake types, yet still generate it for the rest of boring types.
- **Reduced redundancies:** The API aims to make the common case easy and the less common case possible. Redundancies in user encoding/decoding code are deliberately avoided to remove subtle bugs (even at a slight hit on performance). If the user's types require some asymmetry, explicit encoding and decoding code paths are still supported.
- **Support existing types:** Serialization libraries often assume the user is going to define a completely new, isolated type-set for all the things they want to encode. That is simply not the case, and to introduce a new encoding library into a pre-existing codebase, it must play nicely with the existing types. That means common Go typing and aliasing patterns should be supported without annotating everything with new methods.
- **Performant, as meaningful:** Encoding/decoding code should be performant, even if we're giving up some of it to cater for the above goals. Language constructs that are known to be slow (e.g. reflection) should be avoided, and code should have performance similar to low level generated ones, including 0 needing allocations. That said, a meaningful application of the library will *do* something with the encoded data, which will almost certainly take more time than generating/parsing a binary blob.

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

In order to encode/decode such a (standalone) object via SSZ, it needs to implement the `ssz.StaticObject` interface:

```go
type StaticObject interface {
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
	ssz.DefineUint64(codec, &w.Index)        // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)    // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineStaticBytes(codec, &w.Address) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)       // Field (3) - Amount         -  8 bytes
}
```

- The `DefineXYZ` methods should feel self-explanatory. They spill out what fields to encode in what order and into what types. The interesting tidbit is the addressing of the fields. Since this code is used for *both* encoding and decoding, it needs to be able to instantiate any `nil` fields during decoding, so pointers are needed.
- Another interesting part is that we haven't defined an encoder/decoder for `Address`, rather just passed it in, and it works. This is because the `DefineStaticBytes` is actually a generic function that can operate on a variety of `[N]byte` arrays.

To encode the above `Witness` into an SSZ stream, use either `ssz.EncodeToStream` or `ssz.EncodeToBytes`. The former will write into a stream directly, whilst the latter will write into a bytes buffer directly. In both cases you need to supply the output location to avoid GC allocations in the library.

```go
func main() {
	out := new(bytes.Buffer)
	if err := ssz.EncodeToStream(out, new(Withdrawal)); err != nil {
		panic(err)
	}
	fmt.Printf("ssz: %#x\n", blob)
}
```

To decode an SSZ blob, use `ssz.DecodeFromStream` and `ssz.DecodeFromBytes` with the same disclaimers about allocations. Note, decoding requires knowing the *size* of the SSZ blob in advance. Unfortunately, this is a limitation of the SSZ format.

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

In order to encode/decode such a (standalone) object via SSZ, it needs to implement the `ssz.DynamicObject` interface:

```go
type DynamicObject interface {
	// SizeSSZ returns either the static size of the object if fixed == true, or
	// the total size otherwise.
	SizeSSZ(fixed bool) uint32

	// DefineSSZ defines how an object would be encoded/decoded.
	DefineSSZ(codec *Codec)
}
```

If you look at it more closely, you'll notice that it's almost the same as `ssz.StaticObject`, except the type of `SizeSSZ` is different, here taking an extra boolean argument. The method name/type clash is deliberate: it guarantees compile time that dynamic objects cannot end up in static ssz slots and vice versa.

```go
func (e *ExecutionPayload) SizeSSZ(fixed bool) uint32 {
	// Start out with the static size
	size := uint32(512)
	if fixed {
		return size
	}
	// Append all the dynamic sizes
	size += ssz.SizeDynamicBytes(e.ExtraData)           // Field (10) - ExtraData    - max 32 bytes (not enforced)
	size += ssz.SizeSliceOfDynamicBytes(e.Transactions) // Field (13) - Transactions - max 1048576 items, 1073741824 bytes each (not enforced)
	size += ssz.SizeSliceOfStaticObjects(e.Withdrawals) // Field (14) - Withdrawals  - max 16 items, 44 bytes each (not enforced)

	return size
}
```

Opposed to the static `Withdrawal` from the previous section, `ExecutionPayload` has both static and dynamic fields, so we can't just return a pre-computed literal number.

- First up, we will still need to know the static size of the object to avoid costly runtime calculations over and over. Just for reference, that would be the size of all the static fields in the object + 4 bytes for each dynamic field (offset encoding). Feel free to verify the number `512` above.
  - If the caller requested only the static size via the `fixed` parameter, return early.
- If the caller, however, requested the total size of the object, we need to iterate over all the dynamic fields and accumulate all their sizes too.
  - For all the usual Go suspects like slices and arrays of bytes; 2D sliced and arrays of bytes (i.e. `ExtraData` and `Transactions` above), there are helper methods available in the `ssz` package. 
  - For types implementing `ssz.StaticObject / ssz.DynamicObject` (e.g. one item of `Withdrawals` above), there are again helper methods available to use them as single objects, static array of objects, of dynamic slice of objects.

The codec itself is very similar to the static example before:

```go
func (e *ExecutionPayload) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineStaticBytes(codec, &e.ParentHash)                 // Field  ( 0) - ParentHash    -  32 bytes
	ssz.DefineStaticBytes(codec, &e.FeeRecipient)               // Field  ( 1) - FeeRecipient  -  20 bytes
	ssz.DefineStaticBytes(codec, &e.StateRoot)                  // Field  ( 2) - StateRoot     -  32 bytes
	ssz.DefineStaticBytes(codec, &e.ReceiptsRoot)               // Field  ( 3) - ReceiptsRoot  -  32 bytes
	ssz.DefineStaticBytes(codec, &e.LogsBloom)                  // Field  ( 4) - LogsBloom     - 256 bytes
	ssz.DefineStaticBytes(codec, &e.PrevRandao)                 // Field  ( 5) - PrevRandao    -  32 bytes
	ssz.DefineUint64(codec, &e.BlockNumber)                     // Field  ( 6) - BlockNumber   -   8 bytes
	ssz.DefineUint64(codec, &e.GasLimit)                        // Field  ( 7) - GasLimit      -   8 bytes
	ssz.DefineUint64(codec, &e.GasUsed)                         // Field  ( 8) - GasUsed       -   8 bytes
	ssz.DefineUint64(codec, &e.Timestamp)                       // Field  ( 9) - Timestamp     -   8 bytes
	ssz.DefineDynamicBytesOffset(codec, &e.ExtraData)           // Offset (10) - ExtraData     -   4 bytes
	ssz.DefineUint256(codec, &e.BaseFeePerGas)                  // Field  (11) - BaseFeePerGas -  32 bytes
	ssz.DefineStaticBytes(codec, &e.BlockHash)                  // Field  (12) - BlockHash     -  32 bytes
	ssz.DefineSliceOfDynamicBytesOffset(codec, &e.Transactions) // Offset (13) - Transactions  -   4 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &e.Withdrawals) // Offset (14) - Withdrawals   -   4 bytes

	// Define the dynamic data (fields)
	ssz.DefineDynamicBytesContent(codec, &e.ExtraData, 32)                                 // Field (10) - ExtraData
	ssz.DefineSliceOfDynamicBytesContent(codec, &e.Transactions, 1_048_576, 1_073_741_824) // Field (13) - Transactions
	ssz.DefineSliceOfStaticObjectsContent(codec, &e.Withdrawals, 16)                       // Field (14) - Withdrawals
}
```

Most of the `DefineXYZ` methods are similar as before. However, you might spot two distinct sets of method calls, `DefineXYZOffset` and `DefineXYZContent`. You'll need to use these for dynamic fields:
  - When SSZ encodes a dynamic object, it encodes it in two steps.
    - A 4-byte offset pointing to the dynamic data is written into the static SSZ area.
    - The dynamic object's actual encoding are written into the dynamic SSZ area.
  - Encoding libraries can take two routes to handle this scenario:
    - Explicitly require the user to give one command to write the object offset, followed by another command later to write the object content. This is fast, but leaks out encoding detail into user code.
    - Require only one command from the user, under the hood writing the object offset immediately, and stashing the object itself away for later serialization when the dynamic area is reached. This keeps the offset notion hidden from users, but entails a GC hit to the encoder.
  - This package was decided to be allocation free, thus the user is needs to be aware that they need to define the dynamic offset first and the dynamic content later. It's a tradeoff to achieve 50-100% speed increase.
  - You might also note that dynamic fields also pass in size limits that the decoder can enforce.

To encode the above `ExecutionPayload` do just as we have done with the static `Witness` object.

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
		ssz.EncodeStaticBytes(enc, &w.Address)   // Field (2) - Address        - 20 bytes
		ssz.EncodeUint64(enc, w.Amount)          // Field (3) - Amount         -  8 bytes
	})
	codec.DefineDecoder(func(dec *ssz.Decoder) {
		ssz.DecodeUint64(dec, &w.Index)          // Field (0) - Index          -  8 bytes
		ssz.DecodeUint64(dec, &w.Validator)      // Field (1) - ValidatorIndex -  8 bytes
		ssz.DecodeStaticBytes(dec, &w.Address)   // Field (2) - Address        - 20 bytes
		ssz.DecodeUint64(dec, &w.Amount)         // Field (3) - Amount         -  8 bytes
	})
}
```

- As you can see, we piggie-back on the already existing `ssz.Object`'s `DefineSSZ` method, and do *not* require implementing new functions. This is good because we want to be able to seamlessly use unified or split encoders without having to tell everyone about it.
- Whereas previously we had a bunch of `DefineXYZ` method to enumerate the fields for the unified encoding/decoding, here we replaced them with separate definitions for the encoder and decoder via `codec.DefineEncoder` and `codec.DefineDecoder`.
- The implementation of the encoder and decoder follows the exact same pattern and naming conventions as with the `codec` but instead of operating on a `ssz.Codec` object, we're operating on an `ssz.Encoder`/`ssz.Decoder` objects; and instead of calling methods named `ssz.DefineXYZ`, we're calling methods named `ssz.EncodeXYZ` and `ssz.DecodeXYZ`.
- Perhaps note, the `EncodeXYZ` methods do not take pointers to everything anymore, since they do not require the ability to instantiate the field during operation.
  - One exception is the `[N]byte` array types, which need to pointer to avoid an escape to the heap (and inherently, an allocation) within the encoder.

Encoding the above `Witness` into an SSZ stream, you use the same thing as before. Everything is seamless.

### Checked types

If your types are using strongly typed arrays (e.g. `[32]byte`, and not `[]byte`) for static lists, the above codes work just fine. However, some types might want to use `[]byte` as the field type, but have it still *behave* as if it was `[32]byte`. This poses an issue, because if the decoder only sees `[]byte`, it cannot figure out how much data you want to decode into it. For those scenarios, we have *checked methods*.

The previous `Withdrawal` is a good example. Let's replace the `type Address [20]byte` alias, with a plain `[]byte` slice (not a `[20]byte` array, rather an opaque `[]byte` slice).

```go
type Withdrawal struct {
    Index     uint64  `ssz-size:"8"`
    Validator uint64  `ssz-size:"8"`
    Address   []byte  `ssz-size:"20"`
    Amount    uint64  `ssz-size:"8"`
}
```

The code for the `SizeSSZ` remains the same. The code for `DefineSSZ` changes ever so slightly:

```go
func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &w.Index)                   // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)               // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineCheckedStaticBytes(codec, &w.Address, 20) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)                  // Field (3) - Amount         -  8 bytes
}
```

Notably, the `ssz.DefineStaticBytes` call from our old code (which got given a `[20]byte` array), is replaced with `ssz.DefineCheckedStaticBytes`. The latter method operates on an opaque `[]byte` slice, so if we want it to behave like a static sized list, we need to tell it how large it's needed to be. This will result in a runtime check to ensure that the size is correct before decoding.

Note, *checked methods* entail a runtime cost. When decoding such opaque slices, we can't blindly fill the fields with data, rather we need to ensure that they are allocated and that they are of the correct size.  Ideally only use *checked methods* for prototyping or for pre-existing types where you just have to run with whatever you have and can't change the field to an array.

### Generated types

TODO

## Quick reference

The table below is a summary of the methods available for `DefineSSZ`:

- The *Symmetric API* is to be used if the encoding/decoding doesn't require specialised logic.
- The *Asymmetric API* is to be used if encoding or decoding requires special casing

|         Type          |                                                                                                               Symmetric API                                                                                                               |                                                                                                            Asymmetric Encoding                                                                                                            |                                                                                                            Asymmetric Decoding                                                                                                            |
|:---------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|
|        `bool`         |                                                                                   [`DefineBool`](https://pkg.go.dev/github.com/karalabe/ssz#DefineBool)                                                                                   |                                                                                   [`EncodeBool`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeBool)                                                                                   |                                                                                   [`DecodeBool`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeBool)                                                                                   |
|       `uint64`        |                                                                                 [`DefineUint64`](https://pkg.go.dev/github.com/karalabe/ssz#DefineUint64)                                                                                 |                                                                                 [`EncodeUint64`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeUint64)                                                                                 |                                                                                 [`DecodeUint64`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeUint64)                                                                                 |
|      `[]uint64`       |               [`DefineSliceOfUint64sOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfUint64sOffset) [`DefineSliceOfUint64sContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfUint64sContent)               |               [`EncodeSliceOfUint64sOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfUint64sOffset) [`EncodeSliceOfUint64sContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfUint64sContent)               |               [`DecodeSliceOfUint64sOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfUint64sOffset) [`DecodeSliceOfUint64sContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfUint64sContent)               |
|    `*uint256.Int`¹    |                                                                                [`DefineUint256`](https://pkg.go.dev/github.com/karalabe/ssz#DefineUint256)                                                                                |                                                                                [`EncodeUint256`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeUint256)                                                                                |                                                                                [`DecodeUint256`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeUint256)                                                                                |
|      `[N]byte`²       |                                                                            [`DefineStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DefineStaticBytes)                                                                            |                                                                            [`EncodeStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeStaticBytes)                                                                            |                                                                            [`DecodeStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeStaticBytes)                                                                            |
| `[N]byte` in `[]byte` |                                                                     [`DefineCheckedStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DefineCheckedStaticBytes)                                                                     |                                                                     [`EncodeCheckedStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeCheckedStaticBytes)                                                                     |                                                                     [`DecodeCheckedStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeCheckedStaticBytes)                                                                     |
|       `[]byte`        |                   [`DefineDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesOffset) [`DefineDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesContent)                   |                   [`EncodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesOffset) [`EncodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesContent)                   |                   [`DecodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesOffset) [`DecodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesContent)                   |
|     `[M][N]byte`²     |                                                                     [`DefineArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DefineArrayOfStaticBytes)                                                                     |                                                                     [`EncodeArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeArrayOfStaticBytes)                                                                     |                                                                     [`DecodeArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeArrayOfStaticBytes)                                                                     |
|     `[][N]byte`²      |       [`DefineSliceOfStaticBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticBytesOffset) [`DefineSliceOfStaticBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticBytesContent)       |       [`EncodeSliceOfStaticBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticBytesOffset) [`EncodeSliceOfStaticBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticBytesContent)       |       [`DecodeSliceOfStaticBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticBytesOffset) [`DecodeSliceOfStaticBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticBytesContent)       |
|      `[][]byte`       |     [`DefineSliceOfDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicBytesOffset) [`DefineSliceOfDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicBytesContent)     |     [`EncodeSliceOfDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicBytesOffset) [`EncodeSliceOfDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicBytesContent)     |     [`DecodeSliceOfDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicBytesOffset) [`DecodeSliceOfDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicBytesContent)     |
|  `ssz.StaticObject`   |                                                                           [`DefineStaticObject`](https://pkg.go.dev/github.com/karalabe/ssz#DefineStaticObject)                                                                           |                                                                           [`EncodeStaticObject`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeStaticObject)                                                                           |                                                                           [`DecodeStaticObject`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeStaticObject)                                                                           |
| `[]ssz.StaticObject`  |   [`DefineSliceOfStaticObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticObjectsOffset) [`DefineSliceOfStaticObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticObjectsContent)   |   [`EncodeSliceOfStaticObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticObjectsOffset) [`EncodeSliceOfStaticObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticObjectsContent)   |   [`DecodeSliceOfStaticObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticObjectsOffset) [`DecodeSliceOfStaticObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticObjectsContent)   |
|  `ssz.DynamicObject`  |                   [`DefineDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesOffset) [`DefineDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesContent)                   |                   [`EncodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesOffset) [`EncodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesContent)                   |                   [`DecodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesOffset) [`DecodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesContent)                   |
| `[]ssz.DynamicObject` | [`DefineSliceOfDynamicObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicObjectsOffset) [`DefineSliceOfDynamicObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicObjectsContent) | [`EncodeSliceOfDynamicObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicObjectsOffset) [`EncodeSliceOfDynamicObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicObjectsContent) | [`DecodeSliceOfDynamicObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicObjectsOffset) [`DecodeSliceOfDynamicObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicObjectsContent) |

*¹Type is from `github.com/holiman/uint256`.* \
*²`N` can be `{20, 32, 48, 96, 256}`, open an issue if you need another size.*

## Performance

The goal of this package is to be close in performance to low level generated encoders, without sacrificing maintainability. It should, however, be significantly faster than runtime reflection encoders.

There are some benchmarks that you can run via `go test ./tests --run=NONE --bench=.`, but no extensive measurement effort was made yet until the APIs are finalized, nor has the package been compared to any other implementation.
