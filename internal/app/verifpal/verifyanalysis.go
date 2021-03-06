/* SPDX-License-Identifier: GPL-3.0
 * Copyright © 2019-2020 Nadim Kobeissi, Symbolic Software <nadim@symbolic.software>.
 * All Rights Reserved. */

// bc668866bf7ad5972a2f8a9999e62fe7

package main

import (
	"fmt"
)

func verifyAnalysis(model *verifpal, valPrincipalState *principalState, valAttackerState *attackerState, depth int) int {
	valAttackerStateKnownInitLen := len(valAttackerState.known)
	for _, a := range valAttackerState.known {
		if a.kind == "constant" {
			depth = verifyAnalysisResolve(a, valPrincipalState, valAttackerState, depth)
			depth = verifyAnalysisEquivocate(a, valPrincipalState, valAttackerState, depth)
			i := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a.constant)
			if (i >= 0) && valPrincipalState.known[i] {
				depth = verifyAnalysisDeconstruct(a, valPrincipalState, valAttackerState, depth)
				depth = verifyAnalysisReconstruct(a, valPrincipalState, valAttackerState, depth)
			}
		} else {
			depth = verifyAnalysisResolve(a, valPrincipalState, valAttackerState, depth)
			depth = verifyAnalysisDeconstruct(a, valPrincipalState, valAttackerState, depth)
			depth = verifyAnalysisReconstruct(a, valPrincipalState, valAttackerState, depth)
			depth = verifyAnalysisEquivocate(a, valPrincipalState, valAttackerState, depth)
		}
	}
	for _, a := range valPrincipalState.assigned {
		depth = verifyAnalysisReconstruct(a, valPrincipalState, valAttackerState, depth)
	}
	if len(valAttackerState.known) > valAttackerStateKnownInitLen {
		depth = verifyAnalysis(model, valPrincipalState, valAttackerState, depth+1)
	}
	return depth
}

func verifyAnalysisResolve(a value, valPrincipalState *principalState, valAttackerState *attackerState, depth int) int {
	valAttackerStateKnownInitLen := len(valAttackerState.known)
	i := sanityValueInValues(a, &valPrincipalState.assigned, valPrincipalState)
	if i < 0 {
		return depth
	}
	ii := sanityExactSameValueInValues(valPrincipalState.assigned[i], &valAttackerState.known)
	if ii >= 0 {
		return depth
	}
	resolved := valPrincipalState.assigned[i]
	output := []value{}
	if resolved.kind == "primitive" {
		for _, v := range valAttackerState.known {
			switch v.kind {
			case "constant":
				if sanityEquivalentValues(v, resolved, valPrincipalState) {
					output = append(output, v)
				}
			}
		}
		if len(output) != primitiveGet(resolved.primitive.name).output {
			return depth
		}
	} else {
		output = append(output, a)
	}
	if sanityExactSameValueInValues(resolved, &valAttackerState.known) < 0 {
		if sanityExactSameValueInValues(resolved, &valAttackerState.conceivable) < 0 {
			prettyMessage(fmt.Sprintf(
				"%s resolves to %s",
				prettyValues(output), prettyValue(resolved),
			), depth, "deduction")
			valAttackerState.conceivable = append(valAttackerState.conceivable, resolved)
		}
		valAttackerState.known = append(valAttackerState.known, resolved)
		valAttackerState.mutatedTo = append(valAttackerState.mutatedTo, []string{})
	}
	if len(valAttackerState.known) > valAttackerStateKnownInitLen {
		depth = verifyAnalysisResolve(a, valPrincipalState, valAttackerState, depth+1)
	}
	return depth
}

func verifyAnalysisDeconstruct(a value, valPrincipalState *principalState, valAttackerState *attackerState, depth int) int {
	var r bool
	var revealed value
	var ar []value
	valAttackerStateKnownInitLen := len(valAttackerState.known)
	i := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a.constant)
	if i < 0 {
		return depth
	}
	switch a.kind {
	case "constant":
		a = valPrincipalState.assigned[i]
	}
	switch a.kind {
	case "primitive":
		r, revealed, ar = possibleToDeconstructPrimitive(a.primitive, valAttackerState, valPrincipalState)
	}
	if r {
		if sanityExactSameValueInValues(revealed, &valAttackerState.known) < 0 {
			if sanityExactSameValueInValues(revealed, &valAttackerState.conceivable) < 0 {
				prettyMessage(fmt.Sprintf(
					"%s found by attacker by deconstructing %s with %s",
					prettyValue(revealed), prettyValue(a), prettyValues(ar),
				), depth, "deduction")
				valAttackerState.conceivable = append(valAttackerState.conceivable, revealed)
			}
			valPrincipalState.sender[i] = "Attacker"
			valAttackerState.known = append(valAttackerState.known, revealed)
			valAttackerState.mutatedTo = append(valAttackerState.mutatedTo, []string{})
		}
	}
	if len(valAttackerState.known) > valAttackerStateKnownInitLen {
		depth = verifyAnalysisDeconstruct(a, valPrincipalState, valAttackerState, depth+1)
	}
	return depth
}

func verifyAnalysisReconstruct(a value, valPrincipalState *principalState, valAttackerState *attackerState, depth int) int {
	var r bool
	var ar []value
	valAttackerStateKnownInitLen := len(valAttackerState.known)
	aBackup := a
	i := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a.constant)
	if i < 0 {
		return depth
	}
	switch a.kind {
	case "constant":
		a = valPrincipalState.assigned[i]
	}
	switch a.kind {
	case "primitive":
		r, ar = possibleToReconstructPrimitive(a.primitive, valAttackerState, valPrincipalState)
		for _, aa := range a.primitive.arguments {
			verifyAnalysisReconstruct(aa, valPrincipalState, valAttackerState, depth)
		}
	case "equation":
		r, ar = possibleToReconstructEquation(a.equation, valAttackerState, valPrincipalState)
	}
	if r {
		if sanityExactSameValueInValues(aBackup, &valAttackerState.known) < 0 {
			if sanityExactSameValueInValues(aBackup, &valAttackerState.conceivable) < 0 {
				prettyMessage(fmt.Sprintf(
					"%s found by attacker by reconstructing with %s",
					prettyValue(aBackup), prettyValues(ar),
				), depth, "deduction")
				valAttackerState.conceivable = append(valAttackerState.conceivable, aBackup)
			}
			valPrincipalState.sender[i] = "Attacker"
			valAttackerState.known = append(valAttackerState.known, aBackup)
			valAttackerState.mutatedTo = append(valAttackerState.mutatedTo, []string{})
		}
	}
	if len(valAttackerState.known) > valAttackerStateKnownInitLen {
		depth = verifyAnalysisReconstruct(aBackup, valPrincipalState, valAttackerState, depth+1)
	}
	return depth
}

func verifyAnalysisEquivocate(a value, valPrincipalState *principalState, valAttackerState *attackerState, depth int) int {
	valAttackerStateKnownInitLen := len(valAttackerState.known)
	for _, aa := range valPrincipalState.assigned {
		if sanityEquivalentValues(a, aa, valPrincipalState) {
			if sanityExactSameValueInValues(aa, &valAttackerState.known) < 0 {
				if sanityExactSameValueInValues(aa, &valAttackerState.conceivable) < 0 {
					prettyMessage(fmt.Sprintf(
						"%s found by attacker by equivocating with %s",
						prettyValue(aa), prettyValue(a),
					), depth, "deduction")
					valAttackerState.conceivable = append(valAttackerState.conceivable, aa)
				}
				valAttackerState.known = append(valAttackerState.known, aa)
				valAttackerState.mutatedTo = append(valAttackerState.mutatedTo, []string{})
			}
		}
		switch aa.kind {
		case "primitive":
			for _, aaa := range aa.primitive.arguments {
				if sanityEquivalentValues(a, aaa, valPrincipalState) {
					if sanityExactSameValueInValues(aaa, &valAttackerState.known) < 0 {
						if sanityExactSameValueInValues(aaa, &valAttackerState.conceivable) < 0 {
							prettyMessage(fmt.Sprintf(
								"%s found by attacker by equivocating with %s",
								prettyValue(aaa), prettyValue(a),
							), depth, "deduction")
							valAttackerState.conceivable = append(valAttackerState.conceivable, aaa)
						}
						valAttackerState.known = append(valAttackerState.known, aaa)
						valAttackerState.mutatedTo = append(valAttackerState.mutatedTo, []string{})
					}
				}
			}
		}
	}
	if len(valAttackerState.known) > valAttackerStateKnownInitLen {
		depth = verifyAnalysisEquivocate(a, valPrincipalState, valAttackerState, depth+1)
	}
	return depth
}
