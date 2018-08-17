package rummikub

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCombinationSpace_NoDuplicates(t *testing.T) {
	// test whether any duplicates are generated in the combination searchSpace

	gamerules := NewDefaultRules()
	space := NewILPSolver(gamerules)

	toTest := space.AllCombinations()

	tmp := map[CombinationIdentity]BrickCombination{}

	for _, combo := range toTest {
		h := combo.Hash()
		if _, ok := tmp[h]; ok {
			assert.Fail(t, "Collision: %v --and-- %v ", tmp[h], combo)

		} else {
			tmp[h] = combo
		}
	}
}

func TestCombinationSpace_GetPossibleCombinations_ComboSizes(t *testing.T) {
	// test whether the numbers of possible legal combinations per combination size in the search searchSpace is as expected.

	gamerules := NewDefaultRules()
	space := NewILPSolver(gamerules)

	assert.Equal(t, 65, space.groupSizes[4], "Unexpected number of groups of size 4")
	assert.Equal(t, 130, space.groupSizes[3], "Unexpected number of groups of size 3")

	assert.Equal(t, 136, space.runSizes[3], "Unexpected number of runs of size 3")
	assert.Equal(t, 164, space.runSizes[4], "Unexpected number of runs of size 4")
	assert.Equal(t, 184, space.runSizes[5], "Unexpected number of runs of size 5")

}

func TestCombinationSpace_GetPossibleCombinations_Deterministic(t *testing.T) {
	// Ensure that the result of AllCombinations does not vary between calls.
	// This is a regression test. Once upon a time, a bug made the produced search spaces inconsistent between calls.

	gamerules := NewDefaultRules()
	spaceA := NewILPSolver(gamerules)

	// test if result of AllCombinations is an exact copy of the contents of the combinations field.
	assert.Equal(t, spaceA.AllCombinations(), spaceA.combinations, "AllCombinations method does not return exact copy of the combinations field.")
	assert.True(t, reflect.DeepEqual(spaceA.AllCombinations(), spaceA.combinations), "AllCombinations method does not return exact copy of the combinations field according to reflect.DeepEqual.")

	// build another independent search searchSpace.
	gamerules = NewDefaultRules()
	spaceB := NewILPSolver(gamerules)

	//t.Logf("%v", spaceB.AllCombinations())

	// manually test for equality using the hashing methods
	for _, a := range spaceA.combinations {
		assert.True(t, spaceB.Contains(a), "Combination that is present in search solver A is not present in search solver B")
	}
	for _, b := range spaceB.combinations {
		assert.True(t, spaceA.Contains(b), "Combination that is present in search solver B is not present in search solver A")
	}

}

func TestCombinationSpace_Contains(t *testing.T) {
	// test whether the hashing-based presence checker returns the expected results

	gamerules := NewDefaultRules()
	spaceA := NewILPSolver(gamerules)

	validCombo := NewBrickCombination(Brick{Color: "yellow", Value: 2}, Brick{Color: "yellow", Value: 3}, Brick{Color: "yellow", Value: 4})
	inValidCombo := NewBrickCombination(Brick{Color: "green", Value: 2}, Brick{Color: "yellow", Value: 3}, Brick{Color: "yellow", Value: 4})

	assert.True(t, spaceA.Contains(validCombo), "SearchSpace Contains method says valid combination is NOT present in the search searchSpace, which SHOULD be completely valid and EXHAUSTIVE.")
	assert.False(t, spaceA.Contains(inValidCombo), "SearchSpace Contains method says an invalid combination is present in the search searchSpace, which SHOULD be completely valid and exhaustive.")
}
