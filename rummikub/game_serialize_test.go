package rummikub

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Solely for manual inspection of serialized output.
func TestGame_Serialize_Inspect(t *testing.T) {
	gamerules := NewDefaultRules()

	// add two players
	playerA := NewAIPlayer("testplayerA", NewILPSolver(gamerules))

	// add two players
	playerB := NewAIPlayer("testplayerB", NewILPSolver(gamerules))

	// initiate the game
	game := NewGame(gamerules, 88, playerA, playerB)

	// run the game to completion
	game.RunAITurns()

	// serialize the game
	jsonifiedGame := game.Serialize()

	// pretty print for visual inspection
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, jsonifiedGame, "", "\t")
	if error != nil {
		t.Logf("JSON parse error: %v", error)
	}

	t.Logf(prettyJSON.String())

}

// Check whether game deserialization returns a viable game struct.
func TestGame_DeSerialize(t *testing.T) {
	gamerules := NewDefaultRules()

	// add two players
	playerA := NewAIPlayer("testplayerA", NewILPSolver(gamerules))

	// add two players
	playerB := NewAIPlayer("testplayerB", NewILPSolver(gamerules))

	// initiate the game
	gameA := NewGame(gamerules, 88, playerA, playerB)

	serializedYoungGame := gameA.Serialize()

	// run the game
	gameA.RunAITurns()

	// test if game.Serialize() yields same bytes from call to call.
	serializedOldGameA := gameA.Serialize()
	assert.Equal(t, serializedOldGameA, gameA.Serialize(), "GameState serialization yields different results from one call to the next")

	gameB := DeserializeGame(serializedYoungGame)

	gameB.RunAITurns()

	assert.Equal(t, gameA.Serialize(), gameB.Serialize(), "GameState serialization yields different results from one call to the next")

	// check whether the games have both been won
	assert.True(t, gameA.HasBeenWon(), "GameState A has not been won")
	assert.True(t, gameB.HasBeenWon(), "GameState B has not been won")

}

// test whether serializing a fresh game yields the same result (i.e. saving the seed works).
// Note that the resulting game states are not exactly bytewise equal, but should have:
// The same winner
// The same final table
// TODO re-evaluate whether we even care about this.
//func TestGame_Serialize_Determinism(t *testing.T) {
//	gamerules := NewDefaultRules()
//
//	// add two players
//	playerA := NewAIPlayer("testplayerA", NewILPSolver(gamerules))
//
//	// add two players
//	playerB := NewAIPlayer("testplayerB", NewILPSolver(gamerules))
//
//	// initiate the game
//	gameA := NewGame(gamerules, 88, playerA, playerB)
//
//	serializedYoungGame := gameA.Serialize()
//
//	// run the game
//	gameA.RunAITurns()
//
//	// test if game.Serialize() yields same bytes from call to call.
//	serializedOldGameA := gameA.Serialize()
//	assert.Equal(t, serializedOldGameA, gameA.Serialize(), "GameState serialization yields different results from one call to the next")
//
//	gameB := DeserializeGame(serializedYoungGame)
//
//	gameB.RunAITurns()
//
//	// check whether the games have both been won
//	assert.True(t, gameA.HasBeenWon(), "GameState A has not been won")
//	assert.True(t, gameB.HasBeenWon(), "GameState B has not been won")
//
//	// Check whether the games have been won by the same player
//	assert.Equal(t, gameB.CurrentPlayer().Name, gameA.CurrentPlayer().Name, "GameState won by different players after deserialization")
//
//	// check whether the final table arrangement is the same
//	assert.Equal(t, len(gameB.Table()), len(gameA.Table()), "final table arrangements of game a and b are not of the same size")
//	assert.Equal(t, len(DissolveCombinations(gameB.Table())), len(DissolveCombinations(gameA.Table())), "final table arrangements of game a and b are not of the same size")
//
//	// manually test for equality using the hashing methods
//	for _, b := range gameB.Table() {
//		foundA := false
//		for _, a := range gameA.Table() {
//			if a.Hash() == b.Hash() {
//				foundA = true
//				break
//			}
//		}
//		assert.True(t, foundA, "Final game table configuration is not equal. Cant find %v in gameA", b)
//	}
//
//	for _, a := range gameA.Table() {
//		foundB := false
//		for _, b := range gameB.Table() {
//			if a.Hash() == b.Hash() {
//				foundB = true
//				break
//			}
//		}
//		assert.True(t, foundB, "Final game table configuration is not equal. Cant find %v in gameB", a)
//	}
//
//}
