package rummikub

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
)

type GameState struct {
	// contains the players and the contents of their respective hands (aka racks).
	Players []Player `json:"players"`

	// the collection of bricks (in randomized order) that remains after distributing bricks to the players.
	Pile []Brick `json:"pile"`

	// contains contains all successful(i.e. legal) moves made during the game.
	MoveHistory []Move `json:"move_history"`

	// points at index of player in players
	// Note that it relies heavily on the order of the GameState.players slice.
	CurrentTurn int `json:"current_turn"`

	// contains the game rules object.
	Rules Rules `json:"rules"`

	// the Source struct for the random number generator.
	Seed int64 `json:"seed"`
}

// Move processing outcomes.
const (
	LEGAL_MOVE         = "move is legal"
	FORFEITED          = "player forfeits his turn"
	NOT_YOUR_TURN      = "player named in the move object does not correspond to the name of the current player"
	NOT_OWNED          = "there are new bricks in the proposed combination that are not in the player's hand"
	BRICKS_REMOVED     = "bricks were removed from the field"
	VALUE_INSUFFICIENT = "cumulative value of bricks insufficient"
	GAME_WON           = "the game has been won"
)

// Possible outcomes of matching BrickCombinations to game rules.
const (
	LEGAL_COMBINATION              = "Combination is valid"
	TOO_MANY_JOKERS_IN_COMBINATION = "Too many Jokers in combination"
	ILLEGAL_COMBINATION            = "Combination is neither a valid group or a valid run"
	VALUE_OUT_OF_BOUNDS            = "Brick value invalid: brick value outside of game bounds"
	UNKNOWN_COLOR                  = "Brick color was not found in game rules"
)

// TODO separate display names from the names used in determining turns; slightly cleaner.

// NewEmptyGame initiates a game where the getBricks have not yet been randomized and distributed to the players.
// Useful for writing hard-coded test cases.
// Note that it does not contain any seed for random number generation.
func NewEmptyGame(rules Rules, ps ...Player) GameState {

	// TODO check for uniqueness of player names before creating game. Else; return err.
	//playerNames := []string{}
	//for _, p := range ps{
	//	playerNames = append(playerNames, p.getName())
	//}
	//if !noDuplicates(playerNames){
	//
	//}
	//// check if a slice of strings has no duplicates.
	//func noDuplicates(a []string) bool {
	//	m := make(map[string]bool)
	//
	//	for _, x := range a {
	//		m[x] = true
	//	}
	//
	//	if len(m) == len(a) {
	//		return true
	//	}
	//
	//	return false
	//}

	// make a (deterministically ordered) pile consisting of all unique bricks in the game.
	orderedPile := rules.AllBricks()

	return GameState{
		Players:     ps,
		MoveHistory: []Move{},
		Pile:        orderedPile,
		CurrentTurn: 0,
		Rules:       rules,
	}
}

//NewGame initiates a new game struct given a rules struct, a seed number for the random number generator, and a set of player interfaces.
func NewGame(rules Rules, seed int64, ps ...Player) *GameState {
	game := NewEmptyGame(rules, ps...)

	// Save the seed
	game.Seed = seed

	// refresh the deterministic pile with a randomly ordered one
	orderedPile := game.getRules().AllBricks()

	// build a random generator object
	randomSrc := rand.NewSource(game.Seed)
	rndm := rand.New(randomSrc)

	// generate a random permutation of the initial ordered pile.
	perm := rndm.Perm(len(orderedPile))

	// shuffle the pile of bricks
	shuffledPile := make([]Brick, len(orderedPile))
	for i, v := range perm {
		shuffledPile[v] = orderedPile[i]
	}
	game.Pile = shuffledPile

	// distribute the randomized bricks from the pile to the player racks.
	for i := range game.Players {
		tmp := []Brick{}
		for i := 0; i < rules.StartingHandSize; i++ {
			b, err := game.popFromPile()
			if err != nil {
				panic("pile empty before stones could be distributed")
			}
			tmp = append(tmp, *b)
		}
		game.Players[i].SetHand(tmp)
	}

	return &game
}

// DeserializeGame builds a new game state from a serialized game.
// This is necessary as the Solver structs are not serialized (they are too big and mostly constant), and thus need to be 're-armed' on deserialization.
// TODO: currently works with only one type of solver.
func DeserializeGame(serializedGame []byte) *GameState {
	var game GameState
	json.Unmarshal(serializedGame, &game)

	for i := range game.Players {
		if game.Players[i].isHuman() {
			game.Players[i].solver = &DummySolver{}
		} else {
			game.Players[i].solver = NewILPSolver(game.getRules())
		}

	}

	return &game
}

// GetPlayer finds a player by name. Returns a pointer to the Player. Returns nil if Player was not found.
func (game *GameState) GetPlayer(name string) *Player {
	for i, p := range game.Players {
		if p.Name == name {
			return &game.Players[i]
		}
	}
	return nil
}

// pops a stone from the ('left' of) pile
// Note that it will always return a brick by refreshing the pile if it is empty.
func (game *GameState) popFromPile() (*Brick, error) {
	if len(game.Pile) == 1 {
		lastBrick := game.Pile[0]
		game.Pile = []Brick{}
		return &lastBrick, nil
	}

	if len(game.Pile) == 0 {
		return nil, errors.New("pile empty")
	}

	x, newPile := game.Pile[len(game.Pile)-1], game.Pile[:len(game.Pile)-1]
	game.Pile = newPile
	return &x, nil
}

// getRules returns a copy of the game rules.
func (game *GameState) getRules() Rules {
	return game.Rules
}

// returns a pointer to the player object whose turn it is.
func (game *GameState) CurrentPlayer() *Player {
	return &game.Players[game.CurrentTurn]
}

// Table returns a slice containing all BrickCombinations currently on the table by querying the move history.
func (game *GameState) Table() []BrickCombination {
	nMoves := len(game.MoveHistory)

	if nMoves == 0 {
		return []BrickCombination{}
	}

	if nMoves == 1 {
		return game.MoveHistory[0].Arrangement
	}

	return game.MoveHistory[len(game.MoveHistory)-1].Arrangement
}

// Set the new table state (after validation)
func (game *GameState) commitMove(m Move) {
	//game.table = m.proposedTable
	game.MoveHistory = append(game.MoveHistory, m)
}

func (game *GameState) IsLegalMove(move Move) (bool, string) {

	// Get the player
	player := game.CurrentPlayer()

	// check if the name in the move corresponds to the player name (check if it is the player's turn)
	// formality; edge case to strengthen API.
	if player.getName() != move.PlayerName {
		return false, NOT_YOUR_TURN
	}

	// get the bricks of the table arrangement that the player proposes and of the current table arrangement.
	proposedTableBricks := move.Bricks()
	currentTableBricks := DissolveCombinations(game.Table())

	if len(BrickSliceDiff(proposedTableBricks, currentTableBricks)) > 0 {
		return false, BRICKS_REMOVED
	}

	// compute the set difference of the proposed field and the current field
	newBricks := BrickSliceDiff(currentTableBricks, proposedTableBricks)

	// check if any additional bricks are put on the table. If not, the player opted to draw a stone and forfeit.
	if len(newBricks) == 0 {
		return true, FORFEITED
	}

	// check if the difference between the old and the new field is at least a subset of the player's hand
	madeUpBricks := BrickSliceDiff(player.Hand(), newBricks)
	if len(madeUpBricks) > 0 {
		return false, NOT_OWNED
	}

	// check if all proposed combinations are legal according to the game rules
	for _, x := range move.Arrangement {
		if isLegal, why := game.Rules.IsLegalCombination(x); !isLegal {
			return false, why
		}
	}

	// Check if it is the player's first move by searching for its name in the table history
	// If so; it should maximize the value of its proposed new arrangements according to the game rules.
	minValueConstraint := game.getRules().FirstMoveValue
	for _, m := range game.MoveHistory {
		if m.PlayerName == player.getName() {
			minValueConstraint = 0
			break
		}
	}

	cumulativeBrickValue := 0
	for _, b := range newBricks {
		cumulativeBrickValue += b.Value
	}

	if cumulativeBrickValue < minValueConstraint {
		return false, VALUE_INSUFFICIENT
	}

	return true, LEGAL_MOVE
}

// Returns whether the game has been won.
func (game *GameState) HasBeenWon() bool {
	for _, p := range game.Players {
		if len(p.Hand()) == 0 {
			return true
		}
	}
	return false
}

// Move to the next turn: increment or reset the turn counter to point to the next player.
func (game *GameState) cycleTurn() {
	game.CurrentTurn++
	if game.CurrentTurn > len(game.Players)-1 {
		game.CurrentTurn = 0
	}
}

func (game *GameState) IsFirstMove(playerName string) bool {
	for _, m := range game.MoveHistory {
		if m.PlayerName == playerName {
			return false
		}
	}
	return true
}

// RunAITurns cycles (by recursion) through the players, running each AI player's turn.
// It stops when it encounters a non-AI player or when a player has won the game..
func (game *GameState) RunAITurns() {
	// get the player object whose turn it is.
	player := game.CurrentPlayer()
	playerName := player.getName()

	// check if it is an AI, if not; return.
	if player.isHuman() {
		return
	}

	// Check if it is the player's first move by searching for its name in the table history
	// If so; it should maximize the value of its proposed new arrangements according to the game rules.
	valueConstraint := game.getRules().FirstMoveValue
	if game.IsFirstMove(playerName) {
		valueConstraint = 0
	}

	// run the AI player's decision making logic, producing a Move object.
	move := player.MakeMove(game.Table(), valueConstraint)

	// sanity check: check if the name in the Move corresponds to the player whose turn it is.
	if !(move.PlayerName == playerName) {
		panic(fmt.Sprintf("expected player name %v got player name %s", playerName, move.PlayerName))
	}

	// process the player's move. Stop if game has been won.
	// If the move is not processed due to being illegal: panic hard.
	// AI players should never produce illegal moves.
	accepted, reason := game.ProcessMove(move)
	if !accepted {
		if reason == GAME_WON {
			return
		}
		msg := fmt.Sprintf("AI player %v's move not accepted: \n why: %v. \n Offending move: %v \n current table: %v \n player hand : %v", playerName, reason, move, game.Table(), player.Hand())
		panic(msg)
	}

	// recurse until a non-AI player is encountered or the game is won.
	game.RunAITurns()

}

// ProcessMove checks if the move is legal. If so: change the game state accordingly.
// Returns whether the move has been successfully processed.
func (game *GameState) ProcessMove(m Move) (bool, string) {
	_, reason := game.IsLegalMove(m)

	player := game.CurrentPlayer()

	// check if the game has already been won. If so; reject the new move.
	if game.HasBeenWon() {
		return false, GAME_WON
	}

	// process the move
	moveAccepted := false

	switch reason {

	case LEGAL_MOVE:
		// update player hand.
		// get the move delta; the stones to remove from the player's hand.
		bricksPut := BrickSliceDiff(DissolveCombinations(game.Table()), m.Bricks())

		// update the player hand.
		newHand := BrickSliceDiff(bricksPut, player.Hand())
		player.SetHand(newHand)

		// commit the move
		game.commitMove(m)

		// check if this move means the game has been won. If so, return early.
		if game.HasBeenWon() {
			return true, GAME_WON
		}

		moveAccepted = true

	case FORFEITED:
		// save the move in the move history
		game.commitMove(m)

		// Pop stone to hand from pile as penalty for turn forfeiture.
		// If the pile is empty, the player does not have to draw.
		b, err := game.popFromPile()
		if err == nil {
			player.SetHand(append(player.Hand(), *b))
		}

		moveAccepted = true
	}

	// If the game has not been won, increment the cyclic turn counter before returning.
	game.cycleTurn()

	return moveAccepted, reason
}

func (game *GameState) Serialize() []byte {
	bytes, err := json.Marshal(game)
	if err != nil {
		panic(err)
	}
	return bytes
}
