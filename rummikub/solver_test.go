package rummikub

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestProblem struct {
	name                 string
	table                []BrickCombination
	hand                 []Brick
	expectedCombinations []BrickCombination
	expectedBricks       []Brick
}

func solveExampleProblems(t *testing.T, testProblems []TestProblem, space *ILPSolver, maxValue bool) {
	for _, prob := range testProblems {

		t.Run(prob.name, func(t *testing.T) {

			t.Logf("simulated table: %v", prob.table)
			t.Logf("simulated hand: %v", prob.hand)
			t.Logf("Solving...")

			startTime := time.Now()
			combinationsToPut, bricksToPut, solveError := space.Solve(prob.hand, prob.table, maxValue)
			if solveError != nil {
				panic(solveError)
			}
			duration := time.Since(startTime)
			t.Logf("Solving took %v seconds", duration.Seconds())

			t.Logf("Combinations suggested by the solver:")
			t.Logf("%v", combinationsToPut)

			t.Logf("bricks suggested by the solver:")
			t.Logf("%v", bricksToPut)

			// test if the right number of bricks to put was returned by the solver
			assert.Equal(t, len(prob.expectedBricks), len(bricksToPut), "Number of bricks to put suggested by the solver is not as expected.")

			// test if the solver returned the expected number of combinations to put
			assert.Equal(t, len(prob.expectedCombinations), len(combinationsToPut), "Number of combinations to put suggested by the solver is not as expected.")

			// test if the solver returned all expected combinations
			for _, expectation := range prob.expectedCombinations {
				found := false
				for _, suggestion := range combinationsToPut {
					if suggestion.Hash() == expectation.Hash() {
						found = true
						break
					}
				}
				assert.True(t, found, fmt.Sprintf("The following combination: %v was expected but not suggested by the solver", expectation))

			}

			// test if the solver returned all expected bricks
			for _, expectation := range prob.expectedBricks {
				found := false
				for _, suggestion := range bricksToPut {
					if suggestion.Hash() == expectation.Hash() {
						found = true
						break
					}
				}
				assert.True(t, found, fmt.Sprintf("The following brick: %v was expected but not suggested by the solver", expectation))

			}
		})
	}
}

func TestSolver_MaxValue_Deterministic(t *testing.T) {
	// test the maximum values solver by solving a a set of low-dimensional sample Rummikub problems.
	log.SetOutput(ioutil.Discard)

	gamerules := NewDefaultRules()
	space := NewILPSolver(gamerules)

	// Some low-dimensional sample Rummikub problems.
	var testProblems = []TestProblem{

		{
			"PROBLEM 1: contains two MaxBrick solutions of an equal number of bricks, one of which has the higher summed brick value.",
			[]BrickCombination{},
			[]Brick{
				// can be in both
				{Color: "blue", Value: 1},

				// higher yield
				{Color: "blue", Value: 3},
				{Color: "blue", Value: 2},

				// alternative:
				{Color: "red", Value: 1},
				{Color: "green", Value: 1},
			},
			[]BrickCombination{
				{
					Bricks: []Brick{
						{Color: "blue", Value: 3},
						{Color: "blue", Value: 2},
						{Color: "blue", Value: 1},
					},
				},
			},
			[]Brick{
				{Color: "blue", Value: 3},
				{Color: "blue", Value: 2},
				{Color: "blue", Value: 1},
			},
		},
		{
			"PROBLEM 2: can either make a group out of surplus from the dummy, winning a brick value of 4+4, or extend a run winning 4 + 5.",
			[]BrickCombination{
				{Bricks: []Brick{{Color: "blue", Value: 3},
					{Color: "blue", Value: 2},
					{Color: "blue", Value: 1}}},

				// dummy:
				{Bricks: []Brick{{Color: "red", Value: 4},
					{Color: "red", Value: 3},
					{Color: "red", Value: 2},
					{Color: "red", Value: 1}}},
			},
			[]Brick{
				// should be used to extend the run.
				{Color: "blue", Value: 4},
				{Color: "blue", Value: 5},

				//should not be placed
				{Color: "yellow", Value: 4},
			},
			[]BrickCombination{
				{
					Bricks: []Brick{
						{Color: "blue", Value: 5},
						{Color: "blue", Value: 4},
						{Color: "blue", Value: 3},
						{Color: "blue", Value: 2},
						{Color: "blue", Value: 1},
					},
				},
				{
					Bricks: []Brick{
						{Color: "red", Value: 4},
						{Color: "red", Value: 3},
						{Color: "red", Value: 2},
						{Color: "red", Value: 1},
					},
				},
			},
			[]Brick{
				{Color: "blue", Value: 4},
				{Color: "blue", Value: 5},
			},
		},
	}

	solveExampleProblems(t, testProblems, space, true)

}

func TestSolver_MaxBricks_Deterministic(t *testing.T) {
	// test the maximum bricks solver by solving a a set of low-dimensional sample Rummikub problems.
	// note that this test uses the brick maximizer, not the value maximizer.
	log.SetOutput(ioutil.Discard)

	gamerules := NewDefaultRules()
	space := NewILPSolver(gamerules)

	// Some low-dimensional sample Rummikub problems.
	var testProblems = []TestProblem{

		{
			"PROBLEM 1: some bricks can be put by extending present combinations.",
			[]BrickCombination{
				{Bricks: []Brick{{Color: "green", Value: 3},
					{Color: "green", Value: 2},
					{Color: "green", Value: 1}}},
				//-----
				{Bricks: []Brick{{Color: "green", Value: 1},
					{Color: "yellow", Value: 1},
					{Color: "red", Value: 1}}},
				//-----
				{Bricks: []Brick{{Color: "yellow", Value: 2},
					{Color: "yellow", Value: 3},
					{Color: "yellow", Value: 4}}},
				//-----
			},
			[]Brick{
				{Color: "yellow", Value: 2},
				{Color: "green", Value: 4},
				{Color: "yellow", Value: 5},
			},
			[]BrickCombination{
				{
					Bricks: []Brick{
						{Color: "yellow", Value: 2},
						{Color: "yellow", Value: 3},
						{Color: "yellow", Value: 4},
						{Color: "yellow", Value: 5},
					},
				},
				{
					Bricks: []Brick{
						{Color: "green", Value: 4},
						{Color: "green", Value: 3},
						{Color: "green", Value: 2},
						{Color: "green", Value: 1},
					},
				},
				{
					Bricks: []Brick{
						{Color: "green", Value: 1},
						{Color: "yellow", Value: 1},
						{Color: "red", Value: 1},
					},
				},
			},
			[]Brick{
				{Color: "yellow", Value: 5},
				{Color: "green", Value: 4},
			},
		},

		{
			"PROBLEM 1.5: some bricks can be put by relying solely on making a new run.",
			[]BrickCombination{},
			[]Brick{
				{Color: "yellow", Value: 2},
				{Color: "yellow", Value: 3},
				{Color: "yellow", Value: 4},
				//-----
				{Color: "green", Value: 4},
			},
			[]BrickCombination{
				{
					Bricks: []Brick{
						{Color: "yellow", Value: 2},
						{Color: "yellow", Value: 3},
						{Color: "yellow", Value: 4},
					},
				},
			},
			[]Brick{
				{Color: "yellow", Value: 2},
				{Color: "yellow", Value: 3},
				{Color: "yellow", Value: 4},
			},
		},

		{
			" PROBLEM 2: An unfeasible problem: no bricks can be put.",
			[]BrickCombination{
				{Bricks: []Brick{{Color: "green", Value: 3},
					{Color: "green", Value: 2},
					{Color: "green", Value: 1}}},
				//-----
				{Bricks: []Brick{{Color: "green", Value: 1},
					{Color: "yellow", Value: 1},
					{Color: "red", Value: 1}}},
				//-----
				{Bricks: []Brick{{Color: "yellow", Value: 2},
					{Color: "yellow", Value: 3},
					{Color: "yellow", Value: 4}}},
				//-----
			},
			[]Brick{
				{Color: "blue", Value: 2},
			},
			// The combinations already on the table should be returned.
			[]BrickCombination{
				{
					Bricks: []Brick{
						{Color: "green", Value: 3},
						{Color: "green", Value: 2},
						{Color: "green", Value: 1},
					},
				},
				{
					Bricks: []Brick{
						{Color: "green", Value: 1},
						{Color: "yellow", Value: 1},
						{Color: "red", Value: 1},
					},
				},
				{
					Bricks: []Brick{
						{Color: "yellow", Value: 2},
						{Color: "yellow", Value: 3},
						{Color: "yellow", Value: 4},
					},
				},
			},
			// No bricks can be put
			[]Brick{},
		},

		{
			"PROBLEM 3: A problem with a single joker.",
			[]BrickCombination{
				{Bricks: []Brick{{Color: "green", Value: 3},
					{Color: "green", Value: 2},
					{Color: "green", Value: 1}}},
			},
			[]Brick{
				{Color: JokerColor, Value: 1},
				{Color: "green", Value: 5},
			},
			// The combinations already on the table should be returned.
			[]BrickCombination{
				{
					Bricks: []Brick{
						{Color: "green", Value: 5},
						{Color: JokerColor, Value: 1},
						{Color: "green", Value: 3},
						{Color: "green", Value: 2},
						{Color: "green", Value: 1},
					},
				},
			},
			[]Brick{
				{Color: "green", Value: 5},
				{Color: JokerColor, Value: 1},
			},
		},
		{
			"PROBLEM 4: A problem with multiple jokers.",
			[]BrickCombination{
				{Bricks: []Brick{{Color: "green", Value: 3},
					{Color: "green", Value: 2},
					{Color: "green", Value: 1}}},
			},
			[]Brick{
				{Color: JokerColor, Value: 1},
				{Color: JokerColor, Value: 1},
				{Color: "green", Value: 5},
			},
			// The combinations already on the table should be returned.
			[]BrickCombination{
				{
					Bricks: []Brick{
						{Color: "green", Value: 5},
						{Color: "green", Value: 3},
						{Color: JokerColor, Value: 1},
					},
				},
				{
					Bricks: []Brick{
						{Color: JokerColor, Value: 1},
						{Color: "green", Value: 2},
						{Color: "green", Value: 1},
					},
				},
			},
			[]Brick{
				{Color: "green", Value: 5},
				{Color: JokerColor, Value: 1},
				{Color: JokerColor, Value: 1},
			},
		},
	}

	solveExampleProblems(t, testProblems, space, false)

}

func TestSolver_Regression_RemoveStones(t *testing.T) {
	// Test for the presence of a bug where the solver suggested a move that removed stones from the table.
	// in this case the solver produces a move that yields the following error:
	// "getBricks removed from table: [{10 yellow} {11 yellow}]"

	gamerules := NewDefaultRules()

	table := []BrickCombination{
		{Bricks: []Brick{{Value: 10, Color: "yellow"}, {Value: 11, Color: "yellow"}, {Value: 12, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 6, Color: "yellow"}, {Value: 7, Color: "yellow"}, {Value: 8, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 1, Color: "red"}, {Value: 1, Color: "blue"}, {Value: 1, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 5, Color: "red"}, {Value: 5, Color: "blue"}, {Value: 5, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 3, Color: "green"}, {Value: 4, Color: "green"}, {Value: 5, Color: "green"}}},
		{Bricks: []Brick{{Value: 4, Color: "yellow"}, {Value: 5, Color: "yellow"}, {Value: 6, Color: "yellow"}, {Value: 7, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 9, Color: "yellow"}, {Value: 10, Color: "yellow"}, {Value: 11, Color: "yellow"}}},
		{Bricks: []Brick{{Value: 8, Color: "red"}, {Value: 8, Color: "green"}, {Value: 8, Color: "blue"}}},
	}

	hand := []Brick{
		{Value: 2, Color: "green"},
		{Value: 10, Color: "blue"},
		{Value: 3, Color: "blue"},
		{Value: 12, Color: "yellow"},
		{Value: 12, Color: "green"},
		{Value: 7, Color: "red"},
		{Value: 13, Color: "green"},
	}

	solver := NewILPSolver(gamerules)
	player := NewAIPlayer("testplayer", solver)
	player.SetHand(hand)
	move := player.MakeMove(table, 0)

	// has the solver added unowned bricks to play?
	addedBricks := BrickSliceDiff(DissolveCombinations(table), move.Bricks())
	unownedNewBricks := BrickSliceDiff(player.Hand(), addedBricks)
	assert.True(t, len(unownedNewBricks) == 0, "Player added unowned bricks: %v", unownedNewBricks)

	// has the solver removed bricks from play?)
	illegalDelta := BrickSliceDiff(move.Bricks(), DissolveCombinations(table))
	assert.True(t, len(move.Bricks()) >= len(DissolveCombinations(table)), "Size of move is not larger than or equal to size of previous table ")
	assert.True(t, len(illegalDelta) == 0, "getBricks removed from table: %v", illegalDelta)
	t.Log(move.Arrangement)

	// Test outside of GameState context
	combinationsToPut, bricksToPut, solveError := solver.Solve(hand, table, false)
	assert.NoError(t, solveError, "solver produced an error")

	// sanity check: do the contents of combinationsToPut and bricksToPut match?
	assert.Equal(t, len(DissolveCombinations(combinationsToPut)), len(bricksToPut)+len(DissolveCombinations(table)), "suggested bricks to put do not match those in the suggested combinations")
	t.Logf("suggested bricks to put: %v", bricksToPut)

	// check for an illegal delta
	illegalDeltaOutsideGame := BrickSliceDiff(DissolveCombinations(combinationsToPut), DissolveCombinations(table))
	assert.True(t, len(DissolveCombinations(combinationsToPut)) >= len(DissolveCombinations(table)), "Size of move is not larger than or equal to size of previous table ")
	assert.True(t, len(illegalDeltaOutsideGame) == 0, "getBricks removed from table: %v ", illegalDeltaOutsideGame)
	t.Log(combinationsToPut)

}
