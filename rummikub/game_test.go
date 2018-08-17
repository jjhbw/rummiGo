package rummikub

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGame_cycleTurns(t *testing.T) {
	gamerules := NewDefaultRules()

	// add two players
	playerA := NewAIPlayer("testplayerA", NewILPSolver(gamerules))
	playerAHand := []Brick{
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	}
	playerA.SetHand(playerAHand)

	// add two players
	playerB := NewAIPlayer("testplayerB", NewILPSolver(gamerules))
	playerBHand := []Brick{
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	}
	playerB.SetHand(playerBHand)

	// initiate the game
	game := NewEmptyGame(gamerules, playerA, playerB)

	// check that the turn counter starts at 0
	assert.Equal(t, 0, game.CurrentTurn, "turn counter does not start at zero")

	// check that the turn counter points to the right player
	assert.Equal(t, playerA.getName(), game.CurrentPlayer().getName(), "Turn counter does not point at the right player")

	// cycle the turn counter
	game.cycleTurn()

	// check that the turn counter is cyclically incremented.
	assert.Equal(t, playerB.getName(), game.CurrentPlayer().getName(), "Turn counter does not point at the right player")

	// cycle again
	game.cycleTurn()
	assert.Equal(t, playerA.getName(), game.CurrentPlayer().getName(), "Turn counter does not point at the right player")

	// negative assertion
	assert.NotEqual(t, playerB.getName(), game.CurrentPlayer().getName(), "Turn counter does not point at the right player")

}

func TestGame_ProcessMove_AcceptedMove(t *testing.T) {
	gamerules := NewDefaultRules()

	// add two players
	playerA := NewAIPlayer("testplayerA", NewILPSolver(gamerules))
	playerAHand := []Brick{
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 5},
		{Color: "blue", Value: 5},
		{Color: "red", Value: 5},
	}
	playerA.SetHand(playerAHand)

	// add two players
	playerB := NewAIPlayer("testplayerB", NewILPSolver(gamerules))
	playerBHand := []Brick{
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	}
	playerB.SetHand(playerBHand)

	// initiate the game
	game := NewEmptyGame(gamerules, playerA, playerB)

	// save the next brick to be popped from the pile and the size of the pile
	pileSize := len(game.Pile)

	// save the current position of the turn counter
	turn := game.CurrentTurn

	//A fully legal move.
	move := NewMove(playerA.getName(), []BrickCombination{
		{
			Bricks: []Brick{
				{Color: "yellow", Value: 5},
				{Color: "blue", Value: 5},
				{Color: "red", Value: 5},
			},
		},
	})

	accepted, why := game.ProcessMove(move)
	assert.True(t, accepted, "Move was incorrectly rejected")
	assert.Equal(t, LEGAL_MOVE, why, "Move was accepted/rejected for the wrong reason.")

	// - verify that the turn counter has been incremented
	assert.Equal(t, turn+1, game.CurrentTurn, "turn counter has not been incremented.")

	// - verify that the rearrangement history has changed accordingly.
	assert.Equal(t, 1, len(game.MoveHistory), "move history length is not as expected")
	assert.Equal(t, []Move{move}, game.MoveHistory, "move was not committed to history")

	// - verify that the current table arrangement has changed
	assert.Equal(t, move.Arrangement, game.Table())

	// - verify that the stones the player put on the field are properly removed from its hand.
	assert.Equal(t, []Brick{{Color: "green", Value: 2}}, game.Players[0].Hand(), "bricks not removed from player's hand.")

	// - verify that no bricks have been popped from the pile
	assert.Equal(t, pileSize, len(game.Pile), "No stones removed from pile.")

}

func TestGame_ProcessMove_GameWon(t *testing.T) {
	gamerules := NewDefaultRules()

	// add two players
	playerA := NewAIPlayer("testplayerA", NewILPSolver(gamerules))
	playerAHand := []Brick{
		{Color: "yellow", Value: 5},
		{Color: "blue", Value: 5},
		{Color: "red", Value: 5},
	}
	playerA.SetHand(playerAHand)

	// add two players
	playerB := NewAIPlayer("testplayerB", NewILPSolver(gamerules))
	playerBHand := []Brick{
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	}
	playerB.SetHand(playerBHand)

	// initiate the game
	game := NewEmptyGame(gamerules, playerA, playerB)

	// save the next brick to be popped from the pile and the size of the pile
	pileSize := len(game.Pile)

	// save the current position of the turn counter
	turn := game.CurrentTurn

	//A fully legal move.
	move := NewMove(playerA.getName(), []BrickCombination{
		{
			Bricks: []Brick{
				{Color: "yellow", Value: 5},
				{Color: "blue", Value: 5},
				{Color: "red", Value: 5},
			},
		},
	})
	accepted, why := game.ProcessMove(move)
	assert.True(t, accepted, "Move was incorrectly rejected")
	assert.Equal(t, GAME_WON, why, "Move was accepted/rejected for the wrong reason.")

	// - verify that the turn counter has not been incremented after the winning move.
	assert.Equal(t, turn, game.CurrentTurn, "turn counter has changed unexpectedly.")

	// - verify that the rearrangement history has changed
	assert.Equal(t, 1, len(game.MoveHistory), "move history length is not as expected")
	assert.Equal(t, []Move{move}, game.MoveHistory, "move was not committed to history")

	// - verify that the current table arrangement has changed
	assert.Equal(t, move.Arrangement, game.Table())

	// - verify that the stones the playerA put on the field are properly removed from its hand.
	assert.Equal(t, []Brick{}, game.CurrentPlayer().Hand(), "bricks not removed from player's hand.")

	// - verify that no bricks have been popped from the pile
	assert.Equal(t, pileSize, len(game.Pile), "No stones removed from pile.")

	// - verify that the game has been won
	assert.True(t, game.HasBeenWon(), "game has not been won...")
}

func TestGame_ProcessMove_Forfeited(t *testing.T) {
	gamerules := NewDefaultRules()

	// the player hand
	player := NewAIPlayer("testplayer", NewILPSolver(gamerules))
	playerHand := []Brick{
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	}
	player.SetHand(playerHand)

	// initiate the game
	game := NewEmptyGame(gamerules, player)

	// manually populate the table
	tmpTable := []BrickCombination{
		{
			Bricks: []Brick{
				{Color: "yellow", Value: 2},
				{Color: "blue", Value: 1},
				{Color: "red", Value: 1},
			},
		},
	}
	// commit table state to move history
	initialMove := Move{Arrangement: tmpTable}
	game.commitMove(initialMove)

	// save the size of the pile for later verification
	pileSize := len(game.Pile)

	// save the last brick of the pile for later verification
	savedPileBrick := game.Pile[len(game.Pile)-1]

	// CASE: a turn forfeiture (player proposes the same arrangement as is currently on the table)
	move := NewMove(player.getName(), tmpTable)
	accepted, why := game.ProcessMove(move)
	assert.True(t, accepted, "Move was incorrectly rejected")
	assert.Equal(t, FORFEITED, why, "Move was accepted/rejected for the wrong reason.")

	// - verify that the rearrangement history has not changed
	assert.Equal(t, 2, len(game.MoveHistory), "move history length is not as expected")
	assert.Equal(t, []Move{initialMove, move}, game.MoveHistory, "move was not committed to history")

	// - verify that the current table arrangement has not changed
	assert.Equal(t, tmpTable, game.Table())

	// - verify that the player hand has drawn the last brick from the pile
	assert.Equal(t, game.CurrentPlayer().Hand(), append(playerHand, savedPileBrick), "player has not drawn a penalty brick.")

	// - verify that a brick has been popped from the pile
	assert.Equal(t, pileSize-1, len(game.Pile), "No stones removed from pile.")
}

func TestGame_ProcessMove_RejectedMove(t *testing.T) {
	gamerules := NewDefaultRules()

	// the player hand
	player := NewAIPlayer("testplayer", NewILPSolver(gamerules))
	playerHand := []Brick{
		{Color: "green", Value: 2},
		{Color: "yellow", Value: 1},
		{Color: "blue", Value: 1},
		{Color: "red", Value: 1},
	}
	player.SetHand(playerHand)

	// initiate the game
	game := NewEmptyGame(gamerules, player)

	// CASE: an illegal move (player does not have the bricks required to make the move)
	move := NewMove(player.getName(), []BrickCombination{
		{
			Bricks: []Brick{
				{Color: "red", Value: 2},
			},
		},
	})
	accepted, why := game.ProcessMove(move)
	assert.False(t, accepted, "Move was incorrectly accepted")
	assert.Equal(t, NOT_OWNED, why, "Move was accepted/rejected for the wrong reason.")

	// - verify that the rearrangement history has not changed
	assert.Equal(t, 0, len(game.MoveHistory), "move history length is not 0")

	// - verify that the current table arrangement has not changed
	assert.Equal(t, []BrickCombination{}, game.Table())

	// - verify that the player hand has not changed
	assert.Equal(t, game.CurrentPlayer().Hand(), playerHand)
}

func TestGame_Pile_Deterministic(t *testing.T) {
	// test whether the game seed fully governs reproducibility
	gamerules := NewDefaultRules()

	playerA := NewAIPlayer("A", NewILPSolver(gamerules))

	seedA := int64(20)
	gameAa := NewGame(gamerules, seedA, playerA)
	gameAb := NewGame(gamerules, seedA, playerA)

	assert.Equal(t, gameAa.Pile, gameAb.Pile, "Initiating a game with the same seed does not lead to the same pile")

	seedB := int64(10)
	gameBa := NewGame(gamerules, seedB, playerA)

	assert.NotEqual(t, gameBa.Pile, gameAa.Pile, "Initiating a game with a different seed does not lead to a different pile")
	assert.NotEqual(t, gameBa.Pile, gameAb.Pile, "Initiating a game with a different seed does not lead to a different pile")
}

func TestGame_RunAITurns(t *testing.T) {
	gamerules := NewDefaultRules()

	// build the players
	playerA := NewAIPlayer("A", NewILPSolver(gamerules))
	playerB := NewAIPlayer("B", NewILPSolver(gamerules))
	playerC := NewAIPlayer("C", NewILPSolver(gamerules))

	seed := int64(20)
	game := NewGame(gamerules, seed, playerA, playerB, playerC)

	// run the game
	game.RunAITurns()

	// as all players are AI's, the recursive RunAITurns function should run until the game has been won
	assert.True(t, game.HasBeenWon(), "GameState has not been won after full RunAITurns recursion")

	// TODO check all move deltas?
	//for _, m := range game.moveHistory{
	//
	//	for i := range game.players{
	//		if m.playerName == game.players[i].getName(){
	//			for _, savedHand := range game.players[i].handHistory{
	//
	//			}
	//		}
	//	}
	//}

	// test the length of the contents of the history trackers
	allHandMutations := 0
	for i := range game.Players {
		p := game.Players[i]
		mutations := len(p.HandHistory)
		t.Logf("Hand of player %v has had %v mutations", p.getName(), mutations)
		allHandMutations = allHandMutations + len(p.HandHistory)
	}

	// Note that each player has an initial state, which is counted as a mutation.
	assert.Equal(t, allHandMutations, len(game.MoveHistory)+len(game.Players), "Fields that track move history don't match.")

}

func TestGame_PileExhaustion(t *testing.T) {
	gamerules := NewDefaultRules()

	// build the players
	playerA := NewAIPlayer("A", NewILPSolver(gamerules))

	seed := int64(8)
	game := NewGame(gamerules, seed, playerA)

	assert.True(t, len(game.Pile) > 1, "game pile size at game start too small.")
	expectedSize := len(gamerules.AllBricks()) - (gamerules.StartingHandSize * len(game.Players))
	assert.True(t, len(game.Pile) == expectedSize, "game pile size at game start (%v) not as expected(%v).", len(game.Pile), expectedSize)

	// artificially truncate the pile so it is likely to be emptied
	goalPileSize := 1
	for len(game.Pile) > goalPileSize {
		game.popFromPile()
	}
	assert.True(t, len(game.Pile) == goalPileSize, "game pile not truncated by popping.")

	// pop the pile past the refresh checkpoint
	_, err := game.popFromPile()
	assert.NoError(t, err, "Could not draw last brick from the pile")

	_, err = game.popFromPile()
	assert.Error(t, err, "Attempting to draw from an empty pile does not yield an error")

	// check that the pile is empty
	assert.True(t, len(game.Pile) == 0, "Pile is not empty")

}

func TestGame_RunAITurns_StopAtNonAI(t *testing.T) {
	gamerules := NewDefaultRules()

	// build the players
	playerAIa := NewAIPlayer("AI_1", NewILPSolver(gamerules))
	playerAIb := NewAIPlayer("AI_2", NewILPSolver(gamerules))
	playerHuman := NewHumanPlayer("Human_1")

	seed := int64(8)
	game := NewGame(gamerules, seed, playerAIa, playerAIb, playerHuman)

	// AI was added first, so the first turn is for the AI.
	game.RunAITurns()

	// check that the AIs have run its turn.
	assert.Equal(t, game.MoveHistory[0].PlayerName, playerAIa.getName(), "First moves were not made by the AI player")
	assert.Equal(t, game.MoveHistory[1].PlayerName, playerAIb.getName(), "First moves were not made by the AI player")

	// construct a move for the human player
	m := playerHuman.MakeMove(game.Table(), 0)
	accepted, _ := game.ProcessMove(m)
	assert.True(t, accepted, "Human player's move was not accepted.")
	assert.Equal(t, game.MoveHistory[2].PlayerName, playerHuman.getName(), "Third move was not made by the human player")

	// run the AIs again.
	game.RunAITurns()

	assert.Equal(t, game.MoveHistory[3].PlayerName, playerAIa.getName(), "Third move was not made by the AI player")
	assert.Equal(t, game.MoveHistory[4].PlayerName, playerAIb.getName(), "Third move was not made by the AI player")

}

//func TestGame_VaryingGamerules(t *testing.T){
//	assert.Fail(t, "TODO permute the game rules and test a series of randomized games per permutation.")
//}
