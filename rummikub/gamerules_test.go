package rummikub

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLegalCombination(t *testing.T) {
	gamerules := NewDefaultRules()

	// contains wrong colors
	combinationToTest := NewBrickCombination()
	combinationToTest.AddBrick(Brick{Color: "yellow", Value: 1},
		Brick{Color: "blueish", Value: 1},
		Brick{Color: "blueish", Value: 1})
	valid, why := gamerules.IsLegalCombination(combinationToTest)
	t.Log(fmt.Sprintf("%+v\n", combinationToTest))
	assert.Equal(t, UNKNOWN_COLOR, why, "%+v\n", "combination was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, UNKNOWN_COLOR)
	assert.Equal(t, valid, false, "Brickcombination contains unknown colors, but still tagged as valid move!")

	// contains out-of-bounds brick values
	combinationToTest = NewBrickCombination()
	combinationToTest.AddBrick(Brick{Color: "green", Value: 100},
		Brick{Color: "green", Value: 100},
		Brick{Color: "green", Value: 100})
	valid, why = gamerules.IsLegalCombination(combinationToTest)
	t.Log(fmt.Sprintf("%+v\n", combinationToTest))
	assert.Equal(t, VALUE_OUT_OF_BOUNDS, why, "%+v\n", "combination was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, VALUE_OUT_OF_BOUNDS)
	assert.Equal(t, valid, false, "Brickcombination contains out of bounds brick values, but still tagged as valid move!")

	// contains too many jokers
	combinationToTest = NewBrickCombination()
	combinationToTest.AddBrick(MakeJoker(), MakeJoker(), MakeJoker(), MakeJoker())
	valid, why = gamerules.IsLegalCombination(combinationToTest)
	t.Log(fmt.Sprintf("%+v\n", combinationToTest))
	assert.Equal(t, TOO_MANY_JOKERS_IN_COMBINATION, why, "%+v\n", "combination was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, TOO_MANY_JOKERS_IN_COMBINATION)
	assert.Equal(t, valid, false, "Brickcombination contains too many jokers, but still tagged as valid move!")

	// is neither a valid set nor a valid row, but satisfies rest of legality constraints.
	combinationToTest = NewBrickCombination()
	combinationToTest.AddBrick(Brick{Color: "green", Value: 1},
		Brick{Color: "green", Value: 1},
		Brick{Color: "green", Value: 2})
	valid, why = gamerules.IsLegalCombination(combinationToTest)
	t.Log(fmt.Sprintf("%+v\n", combinationToTest))
	assert.Equal(t, ILLEGAL_COMBINATION, why, "%+v\n", "combination was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, ILLEGAL_COMBINATION)
	assert.Equal(t, valid, false, "Brickcombination is not a valid run or set, but still tagged as valid move!")

	// is legal run
	combinationToTest = NewBrickCombination()
	combinationToTest.AddBrick(
		Brick{Color: "yellow", Value: 4},
		Brick{Color: "blue", Value: 4},
		Brick{Color: "red", Value: 4},
	)
	valid, why = gamerules.IsLegalCombination(combinationToTest)
	t.Log(fmt.Sprintf("%+v\n", combinationToTest))
	assert.Equal(t, LEGAL_COMBINATION, why, "%+v\n", "combination was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, LEGAL_COMBINATION)
	assert.Equal(t, valid, true, "Brickcombination is valid but still tagged as invalid!")

	// is legal group
	combinationToTest = NewBrickCombination()
	combinationToTest.AddBrick(
		Brick{Color: "yellow", Value: 2},
		Brick{Color: "yellow", Value: 3},
		Brick{Color: "yellow", Value: 4},
	)
	valid, why = gamerules.IsLegalCombination(combinationToTest)
	t.Log(fmt.Sprintf("%+v\n", combinationToTest))
	assert.Equal(t, LEGAL_COMBINATION, why, "%+v\n", "combination was validated/rejected for the wrong reason: \n %v... \n should be: \n %s", why, LEGAL_COMBINATION)
	assert.Equal(t, valid, true, "Brickcombination is valid but still tagged as invalid!")
}
