// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

// forkMapping maps fork names to fork values. This is used internally by the
// ssz codec generator to convert tags to values.
var forkMapping = map[string]string{
	"frontier":       "Frontier",
	"homestead":      "Homestead",
	"dao":            "DAO",
	"tangerine":      "Tangerine",
	"spurious":       "Spurious",
	"byzantium":      "Byzantium",
	"constantinople": "Constantinople",
	"istanbul":       "Istanbul",
	"muir":           "Muir",
	"phase0":         "Phase0",
	"berlin":         "Berlin",
	"london":         "London",
	"altair":         "Altair",
	"arrow":          "Arrow",
	"gray":           "Gray",
	"bellatrix":      "Bellatrix",
	"paris":          "Paris",
	"merge":          "Merge",
	"shapella":       "Shapella",
	"shanghai":       "Shanghai",
	"capella":        "Capella",
	"dencun":         "Dencun",
	"cancun":         "Cancun",
	"deneb":          "Deneb",
	"pectra":         "Pectra",
	"prague":         "Prague",
	"electra":        "Electra",
	"future":         "Future",
}
