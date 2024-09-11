// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package ssz

// Fork is an enum with all the hard forks that Ethereum mainnet went through,
// which can be used to multiplex monolith types that can encode/decode across
// a range of forks, not just for one specific.
//
// These enums are only meaningful in relation to one another, but are completely
// meaningless numbers otherwise. Do not persist them across code versions.
type Fork int

const (
	ForkUnknown Fork = iota // Placeholder if forks haven't been specified (must be index 0)

	ForkFrontier       // https://ethereum.org/en/history/#frontier
	ForkHomestead      // https://ethereum.org/en/history/#homestead
	ForkDAO            // https://ethereum.org/en/history/#dao-fork
	ForkTangerine      // https://ethereum.org/en/history/#tangerine-whistle
	ForkSpurious       // https://ethereum.org/en/history/#spurious-dragon
	ForkByzantium      // https://ethereum.org/en/history/#byzantium
	ForkConstantinople // https://ethereum.org/en/history/#constantinople
	ForkIstanbul       // https://ethereum.org/en/history/#istanbul
	ForkMuir           // https://ethereum.org/en/history/#muir-glacier
	ForkPhase0         // https://ethereum.org/en/history/#beacon-chain-genesis
	ForkBerlin         // https://ethereum.org/en/history/#berlin
	ForkLondon         // https://ethereum.org/en/history/#london
	ForkAltair         // https://ethereum.org/en/history/#altair
	ForkArrow          // https://ethereum.org/en/history/#arrow-glacier
	ForkGray           // https://ethereum.org/en/history/#gray-glacier
	ForkBellatrix      // https://ethereum.org/en/history/#bellatrix
	ForkParis          // https://ethereum.org/en/history/#paris
	ForkShapella       // https://ethereum.org/en/history/#shapella
	ForkDencun         // https://ethereum.org/en/history/#dencun
	ForkPectra         // https://ethereum.org/en/history/#pectra

	ForkFuture // Use this for specifying future features (must be last index, no gaps)

	ForkMerge    = ForkParis    // Common alias for Paris
	ForkShanghai = ForkShapella // EL alias for Shapella
	ForkCapella  = ForkShapella // CL alias for Shapella
	ForkCancun   = ForkDencun   // EL alias for Dencun
	ForkDeneb    = ForkDencun   // CL alias for Dencun
	ForkPrague   = ForkPectra   // EL alias for Pectra
	ForkElectra  = ForkPectra   // CL alias for Pectra
)

// ForkMapping maps fork names to fork values. This is used internally by the
// ssz codec generator to convert tags to values.
var ForkMapping = map[string]Fork{
	"frontier":       ForkFrontier,
	"homestead":      ForkHomestead,
	"dao":            ForkDAO,
	"tangerine":      ForkTangerine,
	"spurious":       ForkSpurious,
	"byzantium":      ForkByzantium,
	"constantinople": ForkConstantinople,
	"istanbul":       ForkIstanbul,
	"muir":           ForkMuir,
	"phase0":         ForkPhase0,
	"berlin":         ForkBerlin,
	"london":         ForkLondon,
	"altair":         ForkAltair,
	"arrow":          ForkArrow,
	"gray":           ForkGray,
	"bellatrix":      ForkBellatrix,
	"paris":          ForkParis,
	"merge":          ForkMerge,
	"shapella":       ForkShapella,
	"shanghai":       ForkShanghai,
	"capella":        ForkCapella,
	"dencun":         ForkDencun,
	"cancun":         ForkCancun,
	"deneb":          ForkDeneb,
	"pectra":         ForkPectra,
	"prague":         ForkPrague,
	"electra":        ForkElectra,
	"future":         ForkFuture,
}

// ForkFilter can be used by the XXXOnFork methods inside monolithic types to
// define certain fields appearing only in certain forks.
type ForkFilter struct {
	Added   Fork
	Removed Fork
}
