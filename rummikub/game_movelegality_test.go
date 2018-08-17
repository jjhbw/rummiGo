package rummikub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGame_IsLegalMove_IllegalCases(t *testing.T) {
	gamerules := NewDefaultRules()

	// the player hand
	player := NewAIPlayer("testplayer", NewILPSolver(gamerules))
	player.SetHand([]Brick{
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	})

	// the table
	game := NewEmptyGame(gamerules, player)
	a := NewBrickCombination(
		Brick{Color: "green", Value: 3},
		Brick{Color: "green", Value: 2},
		Brick{Color: "green", Value: 1},
	)
	b := NewBrickCombination(
		Brick{Color: "green", Value: 1},
		Brick{Color: "yellow", Value: 1},
		Brick{Color: "red", Value: 1},
	)
	c := NewBrickCombination(
		Brick{Color: "yellow", Value: 2},
		Brick{Color: "yellow", Value: 3},
		Brick{Color: "yellow", Value: 4},
	)
	// commit table state to move history
	game.commitMove(Move{Arrangement: []BrickCombination{a, b, c}})

	// simulate an illegal move (combination is invalid)
	suggestedAdditionalCombination := NewBrickCombination(
		Brick{Color: "green", Value: 2},
		Brick{Color: "blue", Value: 1},
		Brick{Color: "red", Value: 1},
	)
	move := NewMove("testplayer", []BrickCombination{a, b, c, suggestedAdditionalCombination})
	valid, why := game.IsLegalMove(move)
	t.Logf("\n %v", why)
	assert.Equal(t, valid, false, "Move is illegal at the combination level, but is still tagged as legal!!")
	assert.Equal(t, why, ILLEGAL_COMBINATION, "move was passed/rejected for the wrong reason")

	// simulate an illegal move (player doesnt own stones of new combination)
	suggestedAdditionalCombination = NewBrickCombination(
		Brick{Color: "green", Value: 5},
		Brick{Color: "blue", Value: 1},
		Brick{Color: "red", Value: 1},
	)
	move = NewMove("testplayer", []BrickCombination{a, b, c, suggestedAdditionalCombination})
	valid, why = game.IsLegalMove(move)
	t.Logf("\n %v", why)
	assert.Equal(t, valid, false, "Move consists of unowned, but is still tagged as legal!!")
	assert.Equal(t, why, NOT_OWNED, "move was passed/rejected for the wrong reason")

	// simulate an illegal move (proposed field is magically smaller than the current field)
	move = NewMove("testplayer", []BrickCombination{a, b})
	valid, why = game.IsLegalMove(move)
	t.Logf("\n %v", why)
	assert.Equal(t, valid, false, "Proposed move field is smaller than current field, but is still tagged as legal!!")
	assert.Equal(t, why, BRICKS_REMOVED, "move was passed/rejected for the wrong reason")

	// simulate a forfeiture
	move = NewMove("testplayer", []BrickCombination{a, b, c})
	valid, why = game.IsLegalMove(move)
	t.Logf("\n %v", why)
	assert.Equal(t, valid, true, "Player forfeited, but move tagged as illegal!!")
	assert.Equal(t, why, FORFEITED, "move was passed/rejected for the wrong reason")
}

func TestGame_IsLegalMove_NotYourTurn(t *testing.T) {
	// test if the game legality checker properly detects moves that are not made by the current player.
	// mostly a sanity check.
	gamerules := NewDefaultRules()

	// make the players
	playerA := NewAIPlayer("testplayerA", NewILPSolver(gamerules))
	playerA.SetHand([]Brick{
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	})

	playerB := NewAIPlayer("testplayerB", NewILPSolver(gamerules))
	playerB.SetHand([]Brick{
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	})

	// build the game struct
	game := NewEmptyGame(gamerules, playerA, playerB)

	// check that it is player A's turn
	assert.Equal(t, game.CurrentPlayer().getName(), playerA.getName(), "it is not player A's turn.")

	// make a move signed by Player B, while it is player A's turn.
	move := playerB.MakeMove(game.Table(), 0)

	// present the move to the game
	accepted, why := game.ProcessMove(move)
	assert.False(t, accepted, "move is accepted, but should not be; it is not this player's turn")
	assert.Equal(t, NOT_YOUR_TURN, why, "The wrong error message was returned trying to add this player's move: it is not this player's turn.")

}

func TestGame_IsLegalMove_FirstMoveValue(t *testing.T) {
	// test case for the first move value threshold (Rules.firstMoveValue)

	gamerules := NewDefaultRules()

	// the player
	player := NewAIPlayer("testplayer", NewILPSolver(gamerules))

	player.SetHand([]Brick{
		// add stones of sufficient value to the player hand
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 5},
		{Color: "blue", Value: 5},
		{Color: "red", Value: 5},

		// add low-value stones to the player's hand (for the illegal low-value first move)
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	})

	// the table
	game := NewEmptyGame(gamerules, player)
	a := NewBrickCombination(
		Brick{Color: "green", Value: 3},
		Brick{Color: "green", Value: 2},
		Brick{Color: "green", Value: 1},
	)
	b := NewBrickCombination(
		Brick{Color: "green", Value: 1},
		Brick{Color: "yellow", Value: 1},
		Brick{Color: "red", Value: 1},
	)
	c := NewBrickCombination(
		Brick{Color: "yellow", Value: 2},
		Brick{Color: "yellow", Value: 3},
		Brick{Color: "yellow", Value: 4},
	)

	// commit table state to move history
	game.commitMove(Move{Arrangement: []BrickCombination{a, b, c}})

	// simulate a LEGAL move of an appropriate value (for a first move)
	suggestedAdditionalCombination := NewBrickCombination(
		Brick{Color: "yellow", Value: 5},
		Brick{Color: "blue", Value: 5},
		Brick{Color: "red", Value: 5},
	)

	move := NewMove(player.getName(), []BrickCombination{a, b, c, suggestedAdditionalCombination})
	valid, why := game.IsLegalMove(move)
	t.Logf("\n %v", why)
	assert.Equal(t, true, valid, "Move is legal, but marked as illegal!")
	assert.Equal(t, LEGAL_MOVE, why, "move was passed/rejected for the wrong reason")

	// simulate a ILLEGAL move of an INappropriate value (for a first move)
	illegalCombination := NewBrickCombination(
		Brick{Color: "yellow", Value: 1},
		Brick{Color: "blue", Value: 1},
		Brick{Color: "red", Value: 1},
	)

	moveIll := NewMove(player.getName(), []BrickCombination{a, b, c, illegalCombination})
	valid, why = game.IsLegalMove(moveIll)
	t.Logf("\n %v", why)
	assert.Equal(t, false, valid, "Move is illegal, but marked as legal!")
	assert.Equal(t, VALUE_INSUFFICIENT, why, "move was passed/rejected for the wrong reason")
}
