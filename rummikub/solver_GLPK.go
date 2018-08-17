package rummikub

// import (
// 	"fmt"

// 	"github.com/lukpank/go-glpk/glpk"
// )

// /////// OLD CODE

// // The core solver for the rummikub problem.
// // Given all the bricks present in the player hand and knowing the current combinations present on the table, it finds either:
// // 1) the maximum number of bricks that can be placed from the hand on to the table.
// // 2) the combination of bricks that can be placed on the table with highest summed values.
// // The combinations that constitute the proposed new arrangement of the table are also returned.
// // NOTE that the problem is always feasible (not returning any bricks i.e. y = 0 is always possible)
// func (searchSpace *ILPSolver) SolveGLPK(hand []Brick, table []BrickCombination, maxValue bool) ([]BrickCombination, []Brick, error) {
// 	allBricks := searchSpace.uniqueBricks
// 	allCombinations := searchSpace.combinations
// 	tableStones := DissolveCombinations(table)

// 	lp := glpk.New()           // new model object
// 	defer lp.Delete()          // delete the model on function return
// 	lp.SetProbName("Rummikub") // the problem name
// 	lp.SetObjName("Z")         // the name of the parameter to optimize. Lets call it Z.
// 	lp.SetObjDir(glpk.MAX)     // Whether to minimize or maximize the problem.

// 	//---- add the Y variables and their bounds.
// 	colIndicesToBrick := make(map[int32]Brick)
// 	brickToColIndices := make(map[Brick]int32)
// 	for i, bri := range allBricks {
// 		crickColInd := lp.AddCols(1)
// 		name := fmt.Sprintf("Y%d(%v)", i, bri) // name the column for debugging purposes
// 		lp.SetColName(crickColInd, name)
// 		lp.SetColKind(crickColInd, glpk.IV) // y is an integer variable

// 		// set y bounds
// 		//lp.SetColBnds(crickColInd, glpk.DB, 0, float64(solver.rules.replicates)) // y is double bounded (e.g. {0,1,2})
// 		lp.SetColBnds(crickColInd, glpk.DB, 0, 2) // y is double bounded (e.g. {0,1,2})

// 		// set the value of the y-variable (brick column) in the objective function
// 		if maxValue {
// 			// the coefficient of each variable y in the objective function is equal to its brick value.
// 			lp.SetObjCoef(crickColInd, float64(bri.Value))
// 		} else {
// 			// the coefficient of each variable y in the objective function is 1; all bricks have the same value.
// 			lp.SetObjCoef(crickColInd, 1)
// 		}

// 		// save the column index in maps
// 		colIndicesToBrick[int32(crickColInd)] = bri
// 		brickToColIndices[bri] = int32(crickColInd)
// 	}

// 	//---- add the x variables and their bounds
// 	//save column indices of the x variables for easier construction of the constraint matrix
// 	// NOTE: from the docs: " ind[0] and val[0] are ignored", so init with a 0 at index 0.
// 	xIndicesToCombo := make(map[int32]BrickCombination)
// 	comboHashToColumnIndex := make(map[uint32]int32)
// 	for _, combi := range allCombinations {
// 		xColInd := lp.AddCols(1)
// 		colIndex := int32(xColInd)

// 		// Add the x variable
// 		colName := fmt.Sprintf("comboX%v", xColInd) // name the column for debugging purposes
// 		lp.SetColName(int(colIndex), colName)       // name after the index in the possibleCombinations slice for easy retrieval
// 		lp.SetColKind(int(colIndex), glpk.IV)       // y is an integer variable
// 		lp.SetObjCoef(int(colIndex), 0)             // the coefficient of each variable x in the objective function is 0 (i.e. it doesn't count)

// 		// set x bounds
// 		//lp.SetColBnds(int(colIndex), glpk.DB, 0, float64(solver.rules.replicates)) // x is double bounded (e.g. {0,1,2})
// 		lp.SetColBnds(int(colIndex), glpk.DB, 0, 2) // x is double bounded (e.g. {0,1,2})

// 		//save the column index of the x variable
// 		xIndicesToCombo[colIndex] = combi
// 		comboHashToColumnIndex[combi.Hash()] = colIndex
// 	}

// 	//---- add the constraints per brick i
// 	for _, brick := range allBricks {

// 		// //CONSTRAINT 1 the hand (aka rack) constraint
// 		r := countBrickOccurrence(hand, brick)
// 		handRow := lp.AddRows(1)                                                     // returns the index of the added row
// 		lp.SetRowName(handRow, fmt.Sprintf("rack_%v", brick))                        // name the row for debugging purposes
// 		lp.SetMatRow(handRow, []int32{0, brickToColIndices[brick]}, []float64{0, 1}) // NOTE: from the docs: " ind[0] and val[0] are ignored, so a leading 0 is given in both vectors."
// 		lp.SetRowBnds(handRow, glpk.UP, 0, float64(r))

// 		// //CONSTRAINT 2 the "tiles must be on rack or on table" constraint (build the constraint vector (SijXj - Yi))
// 		//(sum(sij * xj) = ti + yi) rewritten as (sum(sij *xj) - yi = ti).
// 		// save the column indices of all combinations that contain the brick and the values they should take
// 		// again, note the leading 0 which is ignored by the spare matrix builder.
// 		colIndices := []int32{0}
// 		containsBrick := []float64{0}
// 		for _, combination := range allCombinations {

// 			// retrieve the index corresponding to the combination
// 			combinationIndex := comboHashToColumnIndex[combination.Hash()]

// 			// does the combination contain the brick?
// 			SijXj := float64(countBrickOccurrence(combination.getBricks(), brick))
// 			colIndices = append(colIndices, combinationIndex)
// 			containsBrick = append(containsBrick, float64(SijXj))

// 		}

// 		// add an index indicating subtraction of the number of times this brick is placed on the table. (-Yi)
// 		colIndices = append(colIndices, brickToColIndices[brick])
// 		containsBrick = append(containsBrick, float64(-1))

// 		// add the row to the matrix
// 		tableRackRow := lp.AddRows(1)
// 		lp.SetRowName(tableRackRow, fmt.Sprintf("tablerack_%v", brick)) // name the row for debugging purposes
// 		lp.SetMatRow(tableRackRow, colIndices, containsBrick)

// 		// count the number of times this brick occurs on the table
// 		t := countBrickOccurrence(tableStones, brick)
// 		lp.SetRowBnds(tableRackRow, glpk.FX, float64(t), float64(t))

// 	}

// 	// DEBUGGING: write out the problem definition to a file
// 	//lp.WriteLP(nil, "ilp_problem_definition.cplex")

// 	// solve the problem with the integer solver
// 	iocp := glpk.NewIocp()
// 	iocp.SetPresolve(true)
// 	solveError := lp.Intopt(iocp)
// 	if solveError != nil {
// 		return nil, nil, solveError
// 	}

// 	// parse the solutions
// 	// note that both the combinations and the bricks can be put multiple times.
// 	bricksToPut := []Brick{}
// 	for _, b := range allBricks {
// 		bTimesToPut := int(lp.MipColVal(int(brickToColIndices[b])))
// 		if bTimesToPut > 0 {
// 			for nb := 0; nb < bTimesToPut; nb++ {
// 				bricksToPut = append(bricksToPut, b)
// 			}
// 		}
// 	}

// 	// get the sets
// 	combinationsToPut := []BrickCombination{}
// 	for colInd, _ := range xIndicesToCombo {
// 		cTimesToPut := int(lp.MipColVal(int(colInd)))
// 		if cTimesToPut > 0 {
// 			for cput := 0; cput < cTimesToPut; cput++ {
// 				combinationsToPut = append(combinationsToPut, xIndicesToCombo[colInd])
// 			}
// 		}
// 	}
// 	return combinationsToPut, bricksToPut, solveError
// }
