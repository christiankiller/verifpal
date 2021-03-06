/* SPDX-License-Identifier: GPL-3.0
 * Copyright © 2019-2020 Nadim Kobeissi, Symbolic Software <nadim@symbolic.software>.
 * All Rights Reserved. */

// 274578ab4bbd4d70871016e78cd562ad

package main

import (
	"fmt"
	"strings"
)

func sanity(model *verifpal) *knowledgeMap {
	var valKnowledgeMap *knowledgeMap
	principals := sanityDeclaredPrincipals(model)
	valKnowledgeMap = constructKnowledgeMap(model, principals)
	sanityQueries(model, valKnowledgeMap)
	return valKnowledgeMap
}

func sanityAssignmentConstants(right value, constants []constant, valKnowledgeMap *knowledgeMap) []constant {
	switch right.kind {
	case "constant":
		unique := true
		for _, c := range constants {
			if right.constant.name == c.name {
				unique = false
				break
			}
		}
		if unique {
			constants = append(constants, right.constant)
		}
	case "primitive":
		p := primitiveGet(right.primitive.name)
		if len(p.name) == 0 {
			errorCritical(fmt.Sprintf(
				"invalid primitive (%s)",
				right.primitive.name,
			))
		}
		if (len(right.primitive.arguments) == 0) || ((p.arity >= 0) && (len(right.primitive.arguments) != p.arity)) {
			plural := ""
			arity := fmt.Sprintf("%d", p.arity)
			if len(right.primitive.arguments) > 1 {
				plural = "s"
			}
			if p.arity < 0 {
				arity = "at least 1"
			}
			errorCritical(fmt.Sprintf(
				"primitive %s has %d input%s, expecting %s",
				right.primitive.name, len(right.primitive.arguments), plural, arity,
			))
		}
		for _, a := range right.primitive.arguments {
			switch a.kind {
			case "constant":
				unique := true
				for _, c := range constants {
					if a.constant.name == c.name {
						unique = false
						break
					}
				}
				if unique {
					constants = append(constants, a.constant)
				}
			case "primitive":
				constants = sanityAssignmentConstants(a, constants, valKnowledgeMap)
			case "equation":
				constants = sanityAssignmentConstants(a, constants, valKnowledgeMap)
			}
		}
	case "equation":
		for _, v := range right.equation.constants {
			unique := true
			for _, c := range constants {
				if v.name == c.name {
					unique = false
					break
				}
			}
			if unique {
				constants = append(constants, v)
			}
		}
	}
	return constants
}

func sanityQueries(model *verifpal, valKnowledgeMap *knowledgeMap) {
	for _, query := range model.queries {
		switch query.kind {
		case "confidentiality":
			i := sanityGetKnowledgeMapIndexFromConstant(valKnowledgeMap, query.constant)
			if i < 0 {
				errorCritical(fmt.Sprintf(
					"confidentiality query refers to unknown value (%s)",
					prettyConstant(query.constant),
				))
			}
		case "authentication":
			if len(query.message.constants) != 1 {
				errorCritical("authentication queries must only have one constant")
			}
			c := query.message.constants[0]
			i := sanityGetKnowledgeMapIndexFromConstant(valKnowledgeMap, c)
			if i >= 0 {
				knows := false
				if valKnowledgeMap.creator[i] == query.message.sender {
					knows = true
				}
				for _, m := range valKnowledgeMap.knownBy[i] {
					if _, ok := m[query.message.sender]; ok {
						knows = true
					}
				}
				if !knows {
					errorCritical(fmt.Sprintf(
						"authentication query depends on %s sending a constant (%s) that they do not know",
						query.message.sender,
						prettyConstant(c),
					))
				}
			} else {
				errorCritical(fmt.Sprintf(
					"authentication query refers to unknown constant (%s)",
					prettyConstant(c),
				))
			}
		}
	}
}

func sanityGetKnowledgeMapIndexFromConstant(valKnowledgeMap *knowledgeMap, c constant) int {
	var index int
	found := false
	for i := range valKnowledgeMap.constants {
		if valKnowledgeMap.constants[i].name == c.name {
			found = true
			index = i
			break
		}
	}
	if !found {
		index = -1
	}
	return index
}

func sanityGetPrincipalStateIndexFromConstant(valPrincipalState *principalState, c constant) int {
	var index int
	found := false
	for i := range valPrincipalState.constants {
		if valPrincipalState.constants[i].name == c.name {
			found = true
			index = i
			break
		}
	}
	if !found {
		index = -1
	}
	return index
}

func sanityGetAttackerStateIndexFromConstant(valAttackerState *attackerState, c constant) int {
	for i, cc := range valAttackerState.known {
		switch cc.kind {
		case "constant":
			if cc.constant.name == c.name {
				return i
			}
		}
	}
	return -1
}

func sanityDeclaredPrincipals(model *verifpal) []string {
	var principals []string
	for _, block := range model.blocks {
		switch block.kind {
		case "principal":
			principals, _ = appendUnique(principals, block.principal.name)
		case "message":
			if !strInSlice(block.message.sender, principals) {
				errorCritical(fmt.Sprintf(
					"principal does not exist (%s)",
					block.message.sender,
				))
			}
			if !strInSlice(block.message.recipient, principals) {
				errorCritical(fmt.Sprintf(
					"principal does not exist (%s)",
					block.message.recipient,
				))
			}
		}
	}
	for _, query := range model.queries {
		switch query.kind {
		case "authentication":
			if !strInSlice(query.message.sender, principals) {
				errorCritical(fmt.Sprintf(
					"principal does not exist (%s)",
					query.message.sender,
				))
			}
			if !strInSlice(query.message.recipient, principals) {
				errorCritical(fmt.Sprintf(
					"principal does not exist (%s)",
					query.message.recipient,
				))
			}
		}
	}
	return principals
}

func sanityExactSameValue(a1 value, a2 value) bool {
	return strings.Compare(prettyValue(a1), prettyValue(a2)) == 0
}

func sanityEquivalentValues(a1 value, a2 value, valPrincipalState *principalState) bool {
	switch a1.kind {
	case "constant":
		i1 := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a1.constant)
		if i1 < 0 {
			return false
		}
		a1 = valPrincipalState.assigned[i1]
	}
	switch a2.kind {
	case "constant":
		i2 := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a2.constant)
		if i2 < 0 {
			return false
		}
		a2 = valPrincipalState.assigned[i2]
	}
	switch a1.kind {
	case "constant":
		switch a2.kind {
		case "constant":
			if a1.constant.name != a2.constant.name {
				return false
			}
			if a1.constant.output != a2.constant.output {
				return false
			}
		case "primitive":
			i1 := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a1.constant)
			if valPrincipalState.assigned[i1].kind != "primitive" {
				return false
			}
			return sanityEquivalentPrimitives(valPrincipalState.assigned[i1].primitive, a2.primitive, valPrincipalState)
		case "equation":
			i1 := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a1.constant)
			if valPrincipalState.assigned[i1].kind != "equation" {
				return false
			}
			return sanityEquivalentEquations(valPrincipalState.assigned[i1].equation, a2.equation, valPrincipalState)
		}
	case "primitive":
		switch a2.kind {
		case "constant":
			i2 := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a2.constant)
			if valPrincipalState.assigned[i2].kind != "primitive" {
				return false
			}
			return sanityEquivalentPrimitives(valPrincipalState.assigned[i2].primitive, a1.primitive, valPrincipalState)
		case "primitive":
			return sanityEquivalentPrimitives(a1.primitive, a2.primitive, valPrincipalState)
		case "equation":
			return false
		}
	case "equation":
		switch a2.kind {
		case "constant":
			i2 := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, a2.constant)
			if valPrincipalState.assigned[i2].kind != "equation" {
				return false
			}
			return sanityEquivalentEquations(valPrincipalState.assigned[i2].equation, a1.equation, valPrincipalState)
		case "primitive":
			return false
		case "equation":
			return sanityEquivalentEquations(a1.equation, a2.equation, valPrincipalState)
		}
	}
	return true
}

func sanityEquivalentPrimitives(p1 primitive, p2 primitive, valPrincipalState *principalState) bool {
	if p1.name != p2.name {
		return false
	}
	if len(p1.arguments) != len(p2.arguments) {
		return false
	}
	for i := range p1.arguments {
		equiv := sanityEquivalentValues(p1.arguments[i], p2.arguments[i], valPrincipalState)
		if !equiv {
			return false
		}
	}
	return true
}

func sanityDeconstructEquationValues(e equation, valPrincipalState *principalState) []value {
	var values []value
	for _, c := range e.constants {
		i := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, c)
		if i < 0 {
			return []value{}
		}
		values = append(values, valPrincipalState.assigned[i])
	}
	return values
}

func sanityEquivalentEquations(e1 equation, e2 equation, valPrincipalState *principalState) bool {
	e1Values := sanityDeconstructEquationValues(e1, valPrincipalState)
	e2Values := sanityDeconstructEquationValues(e2, valPrincipalState)
	if (len(e1Values) == 0) || (len(e2Values) == 0) {
		return false
	}
	if e1Values[0].kind == "equation" && e2Values[0].kind == "equation" {
		e1Base := sanityDeconstructEquationValues(e1Values[0].equation, valPrincipalState)
		e2Base := sanityDeconstructEquationValues(e2Values[0].equation, valPrincipalState)
		if sanityEquivalentValues(e1Base[1], e2Values[1], valPrincipalState) && sanityEquivalentValues(e1Values[1], e2Base[1], valPrincipalState) {
			return true
		}
		if sanityEquivalentValues(e1Base[1], e2Base[1], valPrincipalState) && sanityEquivalentValues(e1Values[1], e2Values[1], valPrincipalState) {
			return true
		}
		return false
	}
	if !sanityEquivalentValues(e1Values[0], e2Values[0], valPrincipalState) {
		return false
	}
	if !sanityEquivalentValues(e1Values[1], e2Values[1], valPrincipalState) {
		return false
	}
	return true
}

func sanityExactSameValueInValues(v value, assigneds *[]value) int {
	index := -1
	for i, a := range *assigneds {
		if sanityExactSameValue(v, a) {
			index = i
			break
		}
	}
	return index
}

func sanityValueInValues(v value, assigneds *[]value, valPrincipalState *principalState) int {
	index := -1
	for i, a := range *assigneds {
		if sanityEquivalentValues(v, a, valPrincipalState) {
			index = i
			break
		}
	}
	return index
}

func sanityPerformRewrites(valPrincipalState *principalState) ([]primitive, []int) {
	var failedRewrites []primitive
	var failedRewritesIndices []int
	for i, a := range valPrincipalState.assigned {
		switch a.kind {
		case "constant":
			continue
		case "primitive":
			prim := primitiveGet(a.primitive.name)
			if prim.rewrite.hasRule {
				wasRewritten, rewrite := possibleToPrimitivePassRewrite(a.primitive, valPrincipalState)
				if wasRewritten {
					valPrincipalState.wasRewritten[i] = true
					valPrincipalState.assigned[i] = rewrite
					valPrincipalState.beforeMutate[i] = rewrite
				} else {
					failedRewrites = append(failedRewrites, a.primitive)
					failedRewritesIndices = append(failedRewritesIndices, i)
				}
			}
		case "equation":
			continue
		}
	}
	return failedRewrites, failedRewritesIndices
}

func sanityGetEquationRootGenerator(e equation, valPrincipalState *principalState) constant {
	i := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, e.constants[0])
	if valPrincipalState.assigned[i].kind == "equation" {
		return sanityGetEquationRootGenerator(valPrincipalState.assigned[i].equation, valPrincipalState)
	}
	return valPrincipalState.assigned[i].constant
}

func sanityCheckEquationGenerators(valPrincipalState *principalState) {
	for _, a := range valPrincipalState.assigned {
		if a.kind == "equation" {
			c := sanityGetEquationRootGenerator(a.equation, valPrincipalState)
			if c.name != "g" {
				errorCritical(fmt.Sprintf(
					"equation does not use 'g' as generator (%s)",
					prettyEquation(a.equation),
				))
			}
		}
	}
}
