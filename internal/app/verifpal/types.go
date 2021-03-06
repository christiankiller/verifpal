/* SPDX-License-Identifier: GPL-3.0
 * Copyright © 2019-2020 Nadim Kobeissi, Symbolic Software <nadim@symbolic.software>.
 * All Rights Reserved. */

// 6dc5ca957dc5760bba97d4d8a0fe4adf

package main

type verifpal struct {
	attacker string
	blocks   []block
	queries  []query
}
type block struct {
	kind      string
	principal principal
	message   message
}
type principal struct {
	name        string
	expressions []expression
}
type message struct {
	sender    string
	recipient string
	constants []constant
}
type query struct {
	kind     string
	constant constant
	message  message
	resolved bool
}
type expression struct {
	kind      string
	qualifier string
	constants []constant
	left      []constant
	right     value
}
type value struct {
	kind      string
	constant  constant
	primitive primitive
	equation  equation
}
type constant struct {
	name      string
	guard     bool
	output    int
	qualifier string
	fresh     bool
}
type primitive struct {
	name      string
	arguments []value
	check     bool
}
type equation struct {
	constants []constant
}
type knowledgeMap struct {
	principals []string
	constants  []constant
	assigned   []value
	creator    []string
	knownBy    [][]map[string]string
}
type decomposeRule struct {
	hasRule bool
	given   []int
	reveal  int
}

type rewriteRule struct {
	hasRule  bool
	name     string
	from     int
	to       int
	matching []int
	filter   func(value, int, *principalState) (value, bool)
}
type primitiveSpec struct {
	name      string
	arity     int
	output    int
	decompose decomposeRule
	rewrite   rewriteRule
	check     bool
}

type principalState struct {
	name          string
	constants     []constant
	assigned      []value
	guard         []bool
	known         []bool
	creator       []string
	sender        []string
	wasRewritten  []bool
	beforeRewrite []value
	wasMutated    []bool
	beforeMutate  []value
}
type attackerState struct {
	active      bool
	known       []value
	conceivable []value
	mutatedTo   [][]string
}
type verifyResult struct {
	query   query
	summary string
}
type replacementMap struct {
	constants     []constant
	replacements  [][]value
	combination   []value
	depthIndex    []int
	injectCounter int
}
