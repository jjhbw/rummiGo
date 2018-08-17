package rummikub

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
)

func Test_BrickSliceUtils(t *testing.T) {
	combinationsA := []BrickCombination{
		{Bricks: []Brick{{Value: 10, Color: "yellow"}, {Value: 11, Color: "yellow"}, {Value: 12, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 11, Color: "green"}, {Value: 11, Color: "yellow"}, {Value: 11, Color: "red"}}},
		{Bricks: []Brick{{Value: 6, Color: "yellow"}, {Value: 7, Color: "yellow"}, {Value: 8, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 1, Color: "red"}, {Value: 1, Color: "blue"}, {Value: 1, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 5, Color: "red"}, {Value: 5, Color: "blue"}, {Value: 5, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 3, Color: "green"}, {Value: 4, Color: "green"}, {Value: 5, Color: "green"}}},
		{Bricks: []Brick{{Value: 4, Color: "yellow"}, {Value: 5, Color: "yellow"}, {Value: 6, Color: "yellow"}, {Value: 7, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 9, Color: "yellow"}, {Value: 10, Color: "yellow"}, {Value: 11, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 8, Color: "red"}, {Value: 8, Color: "green"}, {Value: 8, Color: "blue"}}},
	}

	combinationsB := []BrickCombination{
		{Bricks: []Brick{{Value: 10, Color: "yellow"}, {Value: 11, Color: "yellow"}, {Value: 12, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 6, Color: "yellow"}, {Value: 7, Color: "yellow"}, {Value: 8, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 11, Color: "green"}, {Value: 11, Color: "yellow"}, {Value: 11, Color: "red"}}},
	}

	a := DissolveCombinations(combinationsA)
	b := DissolveCombinations(combinationsB)

	// first test the dissolver
	assert.Equal(t, 28, len(a), "dissolver yields result of unexpected length")
	assert.Equal(t, 9, len(b), "dissolver yields result of unexpected length")

	// test differ (subtract b from a)
	diff := BrickSliceDiff(b, a)

	targetDiff := []Brick{
		{Value: 1, Color: "red"}, {Value: 1, Color: "blue"}, {Value: 1, Color: "yellow"},
		{Value: 5, Color: "red"}, {Value: 5, Color: "blue"}, {Value: 5, Color: "yellow"},
		{Value: 3, Color: "green"}, {Value: 4, Color: "green"}, {Value: 5, Color: "green"},
		{Value: 4, Color: "yellow"}, {Value: 5, Color: "yellow"}, {Value: 6, Color: "yellow"}, {Value: 7, Color: "yellow"},
		{Value: 9, Color: "yellow"}, {Value: 10, Color: "yellow"}, {Value: 11, Color: "yellow"},
		{Value: 8, Color: "red"}, {Value: 8, Color: "green"}, {Value: 8, Color: "blue"},
	}

	assert.Equal(t, len(targetDiff), len(diff), "length of diff is unexpected.")

	//assert.Equal(t, targetDiff, diff,"BrickSliceDiff yields unexpected result")
	for _, target := range targetDiff {
		contains := false
		for _, d := range diff {
			if d.Hash() == target.Hash() {
				contains = true
				break
			}
		}
		assert.True(t, contains, "diff does not contain: ", target)
	}

	for _, target := range diff {
		contains := false
		for _, d := range targetDiff {
			if d.Hash() == target.Hash() {
				contains = true
				break
			}
		}
		assert.True(t, contains, "Diff contains unexpected tile: ", target)
	}

}

func TestBrickCombination_HashOrderInsensitive(t *testing.T) {
	// test whether the BrickCombinations hashing function properly ignores the order in which the uniqueBricks have been entered.
	ComboA := NewBrickCombination(Brick{Color: "green", Value: 3}, Brick{Color: "green", Value: 2}, Brick{Color: "green", Value: 1})
	ComboB := NewBrickCombination(Brick{Color: "green", Value: 1}, Brick{Color: "green", Value: 2}, Brick{Color: "green", Value: 3})
	assert.Equal(t, ComboA.Hash(), ComboB.Hash(), "BrickCombination Hash function not insensitive to brick order!")
}

func TestBrickCombination_HashEquivalence(t *testing.T) {
	// test whether the BrickCombinations hashing function properly returns different hashes for different combinations.
	ComboA := NewBrickCombination(Brick{Color: "green", Value: 3}, Brick{Color: "green", Value: 2}, Brick{Color: "green", Value: 1})
	ComboB := NewBrickCombination(Brick{Color: "green", Value: 1}, Brick{Color: "red", Value: 2}, Brick{Color: "green", Value: 3})
	assert.NotEqual(t, ComboA.Hash(), ComboB.Hash(), "BrickCombination Hash function incorrectly returns the same hash for dissimilar combinations!")
}

func TestBrickCombination_GroupValidityChecker(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	// combination is too small
	groupToTest := NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "red", Value: 1},
		Brick{Color: "red", Value: 1})
	valid, why := groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, COMBINATION_TOO_SMALL, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, COMBINATION_TOO_SMALL))
	assert.Equal(t, valid, false, "IsValidGroup: combination has duplicates but was still tagged as a group")

	// not all colors are unique
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "red", Value: 1},
		Brick{Color: "red", Value: 1},
		Brick{Color: "green", Value: 1})
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, COLORS_NOT_UNIQUE, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, COLORS_NOT_UNIQUE))
	assert.Equal(t, valid, false, "IsValidGroup: combination has duplicates but was still tagged as a group")

	// not all values are the same (1/2)
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "red", Value: 1},
		Brick{Color: "black", Value: 1},
		Brick{Color: "green", Value: 2})
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, CONTAINS_MULTIPLE_VALUES, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, CONTAINS_MULTIPLE_VALUES))
	assert.Equal(t, valid, false, "IsValidGroup: brickcombination has multiple unique values but was still tagged as a group (1/2)")

	// not all values are the same (2/2)
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "green", Value: 1},
		Brick{Color: "green", Value: 1},
		Brick{Color: "green", Value: 2})
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, CONTAINS_MULTIPLE_VALUES, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, CONTAINS_MULTIPLE_VALUES))
	assert.Equal(t, valid, false, "IsValidGroup: brickcombination has multiple unique values but was still tagged as a group (2/2)")

	// contains only jokers
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: JokerColor, Value: 1},
		Brick{Color: JokerColor, Value: 1},
		Brick{Color: JokerColor, Value: 1},
		Brick{Color: JokerColor, Value: 1})
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, CONTAINS_ONLY_JOKERS, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, CONTAINS_ONLY_JOKERS))
	assert.Equal(t, valid, false, "IsValidGroup: brickcombination contains only jokers but was still tagged as a group")

	// is too small
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1})
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, COMBINATION_TOO_SMALL, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, COMBINATION_TOO_SMALL))
	assert.Equal(t, valid, false, "IsValidGroup: brickcombination is too small but was still tagged as a group")

	// is VALID
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1},
		Brick{Color: "green", Value: 1})
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, VALID_GROUP, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, VALID_GROUP))
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidGroup: %v should be valid!", groupToTest))

	// is VALID with joker
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1},
		Brick{Color: "green", Value: 1},
		MakeJoker())
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, VALID_GROUP, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, VALID_GROUP))
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidGroup: %v should be valid!", groupToTest))

	// Regression test: is VALID with joker (combos starting with a joker broke previous tests)
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(MakeJoker(),
		Brick{Color: "yellow", Value: 2},
		Brick{Color: "blue", Value: 2},
		Brick{Color: "green", Value: 2},
	)
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, VALID_GROUP, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, VALID_GROUP))
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidGroup: %v should be valid!", groupToTest))

	// is VALID with 2 jokers
	groupToTest = NewBrickCombination()
	groupToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1},
		Brick{Color: "green", Value: 1},
		MakeJoker(),
		MakeJoker())
	valid, why = groupToTest.IsValidGroup()
	t.Logf("testing: %v", groupToTest)
	assert.Equal(t, why, VALID_GROUP, fmt.Sprintf("group was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, VALID_GROUP))
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidGroup: %v should be valid!", groupToTest))
}

func TestBrickCombination_RunValidityChecker(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	// contains only jokers
	runToTest := NewBrickCombination()
	runToTest.AddBrick(Brick{Color: JokerColor, Value: 1},
		Brick{Color: JokerColor, Value: 1},
		Brick{Color: JokerColor, Value: 1},
		Brick{Color: JokerColor, Value: 1})
	valid, why := runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, false, "IsValidRun: brickcombination contains only jokers but was still tagged as a run")

	// is too small
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1})
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, false, "IsValidRun: brickcombination is too small but was still tagged as a run")

	// contains multiple colors
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1},
		Brick{Color: "blueish", Value: 1})
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, false, "IsValidRun: brickcombination contains multiple colors but was still tagged as a run")

	// contains non-unique numbers
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "green", Value: 1},
		Brick{Color: "green", Value: 1},
		Brick{Color: "green", Value: 2})
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, false, "IsValidRun: brickcombination contains non-unique numbers but was still tagged as run")

	// does not contain consecutive numbers
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1},
		Brick{Color: "blueish", Value: 1})
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, false, "IsValidRun: brickcombination does not contain consecutive values but was still tagged as a run")

	// is VALID
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "blue", Value: 1},
		Brick{Color: "blue", Value: 2},
		Brick{Color: "blue", Value: 3})
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidRun: %v should be valid!", runToTest))

	// is VALID
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "blue", Value: 2},
		Brick{Color: "blue", Value: 3},
		Brick{Color: "blue", Value: 4})
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidRun: %v should be valid!", runToTest))

	// is VALID with joker
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "blue", Value: 1},
		Brick{Color: "blue", Value: 2},
		MakeJoker())
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidRun: %v should be valid!", runToTest))

	// is VALID with 2 jokers
	runToTest = NewBrickCombination()
	runToTest.AddBrick(Brick{Color: "blue", Value: 1},
		Brick{Color: "blue", Value: 2},
		MakeJoker(),
		MakeJoker())
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidRun: %v should be valid!", runToTest))

	// regression test: is VALID but starts with a joker. These runs broke tests in the past.
	runToTest = NewBrickCombination()
	runToTest.AddBrick(MakeJoker(),
		Brick{Color: "yellow", Value: 2},
		Brick{Color: "yellow", Value: 3},
		Brick{Color: "yellow", Value: 4},
	)
	valid, why = runToTest.IsValidRun()
	t.Logf("testing: %v", runToTest)
	t.Logf(why)
	assert.Equal(t, valid, true, fmt.Sprintf("IsValidRun: %v should be valid!", runToTest))

}
