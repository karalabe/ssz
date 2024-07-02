// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package ssz is a simplified SSZ encoder/decoder.
package ssz

import (
	"bytes"
	"io"
	"sync"
)

// Object defines the methods a type needs to implement to be used as an SSZ
// encodable and decodable object.
type Object interface {
	// SizeSSZ returns the total size of an SSZ object.
	SizeSSZ() uint32

	// DefineSSZ defines how an object would be encoded/decoded.
	DefineSSZ(codec *Codec)
}

// encoderPool is a pool of SSZ encoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var encoderPool = sync.Pool{
	New: func() any {
		return &Codec{enc: new(Encoder)}
	},
}

// decoderPool is a pool of SSZ decoders to reuse some tiny internal helpers
// without hitting Go's GC constantly.
var decoderPool = sync.Pool{
	New: func() any {
		return &Codec{dec: new(Decoder)}
	},
}

// Encode serializes the object into an SSZ stream.
func Encode(w io.Writer, obj Object) error {
	codec := encoderPool.Get().(*Codec)
	defer encoderPool.Put(codec)

	codec.enc.out, codec.enc.err, codec.enc.dyn = w, nil, false
	obj.DefineSSZ(codec)
	return codec.enc.err
}

// EncodeToBytes serializes the object into a newly allocated byte buffer.
func EncodeToBytes(obj Object) ([]byte, error) {
	buffer := make([]byte, obj.SizeSSZ())
	if err := Encode(bytes.NewBuffer(buffer[:0]), obj); err != nil {
		return nil, err
	}
	return buffer, nil
}

// Decode parses an object with the given size out of an SSZ stream.
func Decode(r io.Reader, obj Object, size uint32) error {
	codec := decoderPool.Get().(*Codec)
	defer decoderPool.Put(codec)

	codec.dec.in, codec.dec.length, codec.dec.err, codec.dec.dyn = r, size, nil, false
	obj.DefineSSZ(codec)
	return codec.dec.err
}

// DecodeFromBytes parses an object from the given SSZ blob.
func DecodeFromBytes(blob []byte, obj Object) error {
	return Decode(bytes.NewReader(blob), obj, uint32(len(blob)))
}
