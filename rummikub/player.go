package rummikub

// The Player struct contains the state of the player during the game.
type Player struct {
	// may be an empty slice.
	HandHistory [][]Brick `json:"hand_history"`
	Name        string    `json:"name"`
	Human       bool      `json:"human"`

	//Can be equipped with different solvers.
	// TODO neater handling of serialization (solver state currently not serialized)
	solver Solver `json:"-"`
}

type Move struct {
	Arrangement []BrickCombination `json:"arrangement"`
	PlayerName  string             `json:"player_name"`
}

type Solver interface {
	Solve(hand []Brick, table []BrickCombination, maximizeValue bool) (proposedArrangement []BrickCombination, bricksToPut []Brick, solveError error)
}

// NewAIPlayer returns a new AI player given a name and a search space struct which it will use as a solver.
func NewAIPlayer(name string, solver Solver) Player {

	return Player{
		Name:        name,
		HandHistory: [][]Brick{},
		solver:      solver,
		Human:       false,
	}
}

// NewHumanPlayer creates a new human player state container.
// Note that the solver method of the Human Player returns forfeits as per default. It should not be called as part of a game simulation.
// This struct functions just to keep track of the human player's state.
func NewHumanPlayer(name string) Player {
	return Player{
		Name:        name,
		HandHistory: [][]Brick{},
		solver:      &DummySolver{},
		Human:       true,
	}
}

// DummySolver implements the Solver interfaces and always forfeits moves. To be used in testing.
type DummySolver struct{}

func (s *DummySolver) Solve(hand []Brick, table []BrickCombination, maximizeValue bool) (proposedArrangement []BrickCombination, bricksToPut []Brick, solveError error) {
	return table, []Brick{}, nil
}

// Check if the Player object holds
func (p *Player) isHuman() bool {
	return p.Human
}

// SetHand sets the player hand state to a new unordered collection of bricks.
func (p *Player) SetHand(b []Brick) {
	p.HandHistory = append(p.HandHistory, b)
}

// Hand gets the last-added brick collection from the handHistory field.
func (p *Player) Hand() []Brick {
	if len(p.HandHistory) > 1 {
		return p.HandHistory[len(p.HandHistory)-1]
	}

	if len(p.HandHistory) == 0 {
		return []Brick{}
	}

	return p.HandHistory[0]
}

func (p *Player) getName() string {
	return p.Name
}

// MakeMove is the AI player's decision making logic.
// Given the combinations on the table and an optionally nonzero minimum aggregate stone value threshold,
// it will construct a move.
func (p *Player) MakeMove(table []BrickCombination, minValue int) Move {

	// check if there is any constraint in place on the aggregate minimum value of the stones put on the table.
	maxValue := false
	if minValue > 0 {
		maxValue = true
	}

	// Solve the rummikub problem given the hand and the table.
	// Note that the selection between MaxBrick and MaxVal is made based on the firstMove argument.
	proposedTable, _, solveError := p.solver.Solve(p.Hand(), table, maxValue)

	// TODO: better error handling
	if solveError != nil {
		panic(solveError)
	}

	// build a new move object from the proposed table configuration.
	candidateMove := NewMove(p.Name, proposedTable)

	// get the stones that are currently on the table
	tableStones := DissolveCombinations(table)

	// check if the stones that are going to be put on the table satisfy the minimum value constraint.
	if len(BrickSliceDiff(tableStones, candidateMove.Bricks())) > minValue {
		return candidateMove
	}

	// if the minValue constraint is not satisfied: return a forfeiting move (propose an unchanged table).
	return NewMove(p.Name, table)

}

func (move *Move) Bricks() []Brick {
	proposedStones := []Brick{}
	for _, c := range move.Arrangement {
		proposedStones = append(proposedStones, c.getBricks()...)
	}
	return proposedStones
}

func NewMove(playerName string, proposedTable []BrickCombination) Move {
	return Move{Arrangement: proposedTable, PlayerName: playerName}
}
