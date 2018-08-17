package rummikub

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"gitlab.com/jjhbarkeywolf/ilp"
)

// TODO: vendor dependencies, especially my own
// TODO: make these functions adaptive to changing game rules (primarily the amount of jokers that are allowed per combination).

// The ILPSolver struct contains a validated (de-duplicated, legal, etc.) set of BrickCombinations.
type ILPSolver struct {
	combinations []BrickCombination
	uniqueBricks []Brick

	// store the Rules struct this solver is based upon.
	rules Rules

	// To facilitate quick lookup using the Contains method.
	combinationHashes map[CombinationIdentity]bool

	// some stats about the search solver.
	totalRuns   int
	totalGroups int
	groupSizes  map[int]int // how many sets of a certain length
	runSizes    map[int]int // how many rows of a certain length
}

// return the size of the ILPSolver instance
func (searchSpace *ILPSolver) Size() (int, int, int) {
	return len(searchSpace.uniqueBricks), searchSpace.totalRuns, searchSpace.totalGroups
}

// AllCombinations returns a copy of the combinations in the ILPSolver struct
func (searchSpace *ILPSolver) AllCombinations() []BrickCombination {
	//(slices are reference types).
	return searchSpace.combinations
}

// getBricks returns a copy of the uniqueBricks that the ILPSolver is based on.
func (searchSpace *ILPSolver) Bricks() []Brick {
	//(slices are reference types).
	return searchSpace.uniqueBricks
}

// addCombinations adds combinations to the ILPSolver object after checking their validity, simultaneously incrementing counters / lookup maps.
func (searchSpace *ILPSolver) addCombinations(combinations []BrickCombination) {
	for _, combo := range combinations {
		h := combo.Hash()
		if _, ok := searchSpace.combinationHashes[h]; ok {
			continue //ignore this combination
		} else {
			searchSpace.combinations = append(searchSpace.combinations, combo)
			searchSpace.combinationHashes[h] = true

			// validate the combinations and update the tallies
			isRun, _ := combo.IsValidRun()
			isGroup, _ := combo.IsValidGroup()
			if isRun {
				searchSpace.totalRuns++
				searchSpace.runSizes[len(combo.getBricks())]++
			} else if isGroup {
				searchSpace.totalGroups++
				searchSpace.groupSizes[len(combo.getBricks())]++
			} else {
				panic(fmt.Sprintf("invalid combination (not a valid run nor a valid group) supplied to ILPSolver: \n %v", combo))
			}
		}
	}
}

// Contains checks if a certain BrickCombination is present in the ILPSolver.
func (searchSpace *ILPSolver) Contains(combo BrickCombination) bool {
	// check if a combination is present in the search solver

	h := combo.Hash()
	_, ok := searchSpace.combinationHashes[h]
	return ok
}

// NewILPSolver maps all legal combinations that can be made given a set of getBricks and a Rules object that contains the game rules and returns a ILPSolver.
func NewILPSolver(gameRules Rules) *ILPSolver {

	brickSet := gameRules.BaseBricks()

	// compute all possible BrickCombinations using the game rules and the available base uniqueBricks.
	runs := ComputeAllRuns(brickSet)
	groups := ComputeAllGroups(brickSet)
	saltyGroups := saltWithJokers(groups, gameRules.JokersPerCombination)
	saltyRuns := saltWithJokers(runs, gameRules.JokersPerCombination)

	// build a search solver object
	space := &ILPSolver{
		combinationHashes: make(map[CombinationIdentity]bool),
		groupSizes:        make(map[int]int),
		runSizes:          make(map[int]int),
		rules:             gameRules,
	}

	// save all unique uniqueBricks used to build the search solver with.
	//Include ONE joker (jokers are considered non-unique) if the play set includes jokers.
	space.uniqueBricks = brickSet
	if gameRules.JokersInPlay > 0 {
		space.uniqueBricks = append(space.uniqueBricks, MakeJoker())
	}

	// add the combinations to the search solver.
	space.addCombinations(saltyGroups)
	space.addCombinations(saltyRuns)
	space.addCombinations(groups)
	space.addCombinations(runs)

	return space
}

// ComputeAllRuns retrieves all possible runs that can be made given a set of getBricks
func ComputeAllRuns(brickSet []Brick) []BrickCombination {
	perColor := map[string][]Brick{}

	for _, x := range brickSet {
		perColor[x.Color] = append(perColor[x.Color], x)
	}

	runs := []BrickCombination{}
	for _, v := range perColor {
		for _, runsize := range []int{3, 4, 5} {
			for i := 0; i <= (len(v) - runsize); i++ {
				run := NewBrickCombination()
				for q := i; q < i+runsize; q++ {
					run.AddBrick(v[q])
				}

				runs = append(runs, run)
			}
		}
	}
	return runs

}

// ComputeAllGroups returns all possible groups that can be made given a set of getBricks
func ComputeAllGroups(brickSet []Brick) []BrickCombination {
	perValue := map[int][]Brick{}

	for _, x := range brickSet {
		perValue[x.Value] = append(perValue[x.Value], x)
	}

	groups := []BrickCombination{}

	// add groups of size 4
	for _, v := range perValue {
		grp := NewBrickCombination(v...)
		// for _, b := range v {grp.AddBrick(b)}
		// grp.AddBrick(v...)
		groups = append(groups, grp)
	}

	// add groups of size 3
	for _, v := range perValue {
		rawCombinations := combinationsWithoutReplacement(v, 3)
		for _, c := range rawCombinations {
			grp := NewBrickCombination(c...)
			// grp.AddBrick(c...)
			groups = append(groups, grp)
		}
	}

	return groups

}

// combinationsWithoutReplacement is a recursive function that finds all possible (not necessarily valid) combinations of uniqueBricks, given a set of uniqueBricks.
func combinationsWithoutReplacement(brickSet []Brick, k int) [][]Brick {
	var (
		subI int
		ret  [][]Brick
		sub  [][]Brick
		next []Brick
	)
	for i := 0; i < len(brickSet); i++ {
		if k == 1 {

			ret = append(ret, []Brick{brickSet[i]})
		} else {
			sub = combinationsWithoutReplacement(brickSet[i+1:], k-1)
			for subI = 0; subI < len(sub); subI++ {
				next = sub[subI]
				next = append([]Brick{brickSet[i]}, next...) // equivalent of JS Array.unshift

				ret = append(ret, next)
			}
		}
	}
	return ret
}

// saltWithJokers generates new combinations with each stone replaced by a joker, given a set of combinations.
func saltWithJokers(combinations []BrickCombination, nJokersPerCombination int) []BrickCombination {
	// generate the combinations that would occur if each stone was replaced with a joker.
	for _, c := range combinations {
		combinations = append(combinations, saltCombination(c)...)
	}
	nJokersPerCombination--

	// redo for as many jokers as we allow per combination
	if nJokersPerCombination > 0 {
		combinations = saltWithJokers(combinations, nJokersPerCombination)
	}

	return combinations

}

func saltCombination(combination BrickCombination) []BrickCombination {
	// replace each tile in the combination with a joker
	jokerized := []BrickCombination{}
	joker := MakeJoker()
	for i := range combination.getBricks() {
		jkrzd := combination.Copy()
		jkrzd.getBricks()[i] = joker
		jokerized = append(jokerized, jkrzd)
	}

	return jokerized
}

func countBrickOccurrence(set []Brick, b Brick) int {
	count := 0
	for _, a := range set {
		if a.Color == b.Color && a.Value == b.Value {
			count++
		}
	}
	return count
}

// Solve runs the ILP-based solver for the rummikub problem.
// Given all the bricks present in the player hand and knowing the current combinations present on the table, it finds either:
// 1) the maximum number of bricks that can be placed from the hand on to the table.
// 2) the combination of bricks that can be placed on the table with highest summed values.
// The combinations that constitute the proposed new arrangement of the table are also returned.
// NOTE that the problem is always feasible (not returning any bricks i.e. y = 0 is always possible)
func (searchSpace *ILPSolver) Solve(hand []Brick, table []BrickCombination, maxValue bool) ([]BrickCombination, []Brick, error) {

	allBricks := searchSpace.uniqueBricks
	allCombinations := searchSpace.combinations
	tableStones := DissolveCombinations(table)

	// initiate a new problem
	prob := ilp.NewProblem()

	// set it to maximize the objective function
	prob.Maximize()

	// add the x variables (the brick combinations) and their bounds, storing their references.
	comboVars := make(map[CombinationIdentity]*ilp.Variable)
	varNameToCombo := make(map[string]*BrickCombination)

	for i, combi := range allCombinations {
		name := fmt.Sprintf("combi_%v", i)
		comboVar := prob.AddVariable(name).
			SetCoeff(0).
			IsInteger().
			LowerBound(0).
			UpperBound(2)

		comboVars[combi.Hash()] = comboVar
		varNameToCombo[name] = &allCombinations[i]

	}

	// add the Y variables; one for each brick
	varNameToBrick := make(map[string]*Brick)
	for i, bri := range allBricks {

		// decide on the coefficient of yi in the objective function
		// if not overridden, the coefficient of each variable y in the objective function is 1; all bricks have the same value.
		var yiCoef float64 = 1
		if maxValue {
			// the coefficient of each variable y in the objective function is equal to its brick value.
			yiCoef = float64(bri.Value)
		}

		// create the variable struct
		name := fmt.Sprintf("%s_%v", bri.Color, bri.Value)
		yi := prob.AddVariable(name).
			SetCoeff(yiCoef).
			IsInteger().
			LowerBound(0).
			UpperBound(2)

		// save it to the name-brick mapping
		varNameToBrick[name] = &allBricks[i]

		// //CONSTRAINT 1 the hand (aka rack) constraint
		// Specifies that the brick to put on the table must first be in the player's hand
		// NOTE that we do this by overriding the variable's upper bound. This is more efficient in light of the presolve procedure.
		ri := countBrickOccurrence(hand, bri)
		yi.UpperBound(float64(ri))

		// //CONSTRAINT 2 the "tiles must be on rack or on table" constraint
		// (sum(sij * xj) = ti + yi) rewritten as (sum(sij * xj) - yi = ti).
		// count the number of times this brick occurs on the table
		t := countBrickOccurrence(tableStones, bri)

		// build the constraint
		constraintTwo := prob.AddConstraint().
			AddExpression(-1, yi).
			EqualTo(float64(t))

		// add an expression for each combination xj that includes brick yi
		for _, combi := range allCombinations {

			// how many times does this combination contain this type of brick?
			Sij := float64(countBrickOccurrence(combi.getBricks(), bri))

			xj := comboVars[combi.Hash()]
			constraintTwo.AddExpression(Sij, xj)

		}

	}

	// TODO: set the number of workers to the number of CPUs
	prob.SetWorkers(4)

	//TODO: REMOVEME add tree logger instrumentation
	tl := ilp.NewTreeLogger()
	prob.SetInstrumentation(tl)

	// TODO: REMOVEME dump the debug log to a file regardless of whether the solver returns
	defer dumpToDot(tl)

	// run the solver
	soln, err := prob.Solve()
	if err != nil {
		return nil, nil, err
	}

	// get the coefficients for each combination
	var combinationsToPut []BrickCombination
	for name, combi := range varNameToCombo {

		cTimesToPut, err := soln.GetValueFor(name)
		if err != nil {
			// This should never happen as it would indicate a problem with the ILP lib and thus never fail silently.
			panic(err)
		}

		for cput := 0; cput < int(cTimesToPut); cput++ {
			combinationsToPut = append(combinationsToPut, *combi)
		}

	}

	// get the coefficients for each brick
	var bricksToPut []Brick
	for name, brick := range varNameToBrick {

		cTimesToPut, err := soln.GetValueFor(name)
		if err != nil {
			// This should never happen as it would indicate a problem with the ILP lib and thus never fail silently.
			panic(err)
		}

		for cput := 0; cput < int(cTimesToPut); cput++ {
			bricksToPut = append(bricksToPut, *brick)
		}

	}

	return combinationsToPut, bricksToPut, nil

}

// TODO: REMOVEME
func dumpToDot(tree *ilp.TreeLogger) {
	var buffer bytes.Buffer
	tree.ToDOT(&buffer)
	err := ioutil.WriteFile("__debugviz__.dot", buffer.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}
