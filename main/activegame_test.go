// +build !race
package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gitlab.com/jjhbarkeywolf/rummiGo/rummikub"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// remove the http prefix from the url given by net/http testserver for use with the websocket protocol.
func trimHTTPproto(s string) string {
	return strings.TrimPrefix(s, "http:")
}

func TestActiveGame_SubscriptionFlow(t *testing.T) {
	// shortcut a game into the database
	gamerules := rummikub.NewDefaultRules()
	playerName := "testplayer"
	player := rummikub.NewAIPlayer("AIplayer", rummikub.NewILPSolver(gamerules))

	// initiate the game and store it
	gamestate := rummikub.NewGame(gamerules, 88, player, rummikub.NewHumanPlayer(playerName))
	gameID := gameDB.StoreNewGame(gamestate)

	// initiate the global logger
	//var hook *test.Hook
	logger, _ = test.NewNullLogger()

	// run the test server
	ts := httptest.NewServer(buildServeMux())

	// // subscribe to the game using the subscription endpoint
	//send the upgrade request; create a new connection
	u := "ws:" + trimHTTPproto(ts.URL) + SUBSCRIBE + "/" + gameID + "/" + playerName
	t.Logf("connecting to %v", u)
	conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Logf("dial error:", err)
		assert.Fail(t, "dial failed")
	}
	assert.NotNil(t, resp, "response object is nil")
	assert.Equal(t, 101, resp.StatusCode, "Unexpected status code")

	// set a timer
	deadline := make(chan bool)
	time.AfterFunc(1*time.Second, func() {
		deadline <- true
	})

	// read messages coming from the server through the connection struct.
	bufferedMessages := make(chan []byte, 2)
	go func() {
		for {
			_, message, err := conn.ReadMessage()

			// if the websocket is closed (e.g. when the writePump closes it), this goroutine is returned.
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					t.Logf("error: %v", err)
					t.Fail()
				}
				break
			}

			// save the messages sent by the server
			bufferedMessages <- message
			if len(bufferedMessages) == 2 {
				close(bufferedMessages)
				return
			}
		}
	}()

	// wait until the deadline expires.
	<-deadline

	// shortcut the active games to compare the received data with the actual data
	aGame := activeGamesStore.get(gameID)

	// get the expected values
	snapshotBytes, _ := json.Marshal(Envelope{
		GAME_SNAPSHOT,
		aGame.snapshot(),
	})

	handBytes, _ := json.Marshal(Envelope{
		HAND_UPDATE,
		aGame.connectedClients[playerName].NewHandSnapshot(),
	})

	// inspect the saved messages and compare with saved values
	savedMessages := [][]byte{}
	for a := range bufferedMessages {
		savedMessages = append(savedMessages, a)
	}
	assert.Contains(t, savedMessages, snapshotBytes, "no game snapshot was received.")
	assert.Contains(t, savedMessages, handBytes, "No player hand was received.")

	// cleanly close the active game
	aGame.Close()

	// block until the game is closed and thus removed from the activeGamesStore.
	for activeGamesStore.contains(gameID) {
		time.Sleep(100)
	}

	// dump the saved messages
	t.Logf("--Received messages:")
	for _, msg := range savedMessages {
		t.Logf(string(msg))
	}

}

func TestActiveGame_Heartbeat_Cleanup(t *testing.T) {
	// test whether the heartbeat properly initiates a cleanup if the game is left without players.
	// shortcut a game into the database
	gamerules := rummikub.NewDefaultRules()
	playerName := "testplayer"
	player := rummikub.NewAIPlayer("AIplayer", rummikub.NewILPSolver(gamerules))

	// initiate the game and store it
	gamestate := rummikub.NewGame(gamerules, 88, player, rummikub.NewHumanPlayer(playerName))
	gameID := gameDB.StoreNewGame(gamestate)

	// // activate the game, but dont connect any players.
	// hook the cleanup function
	cleanupCalled := make(chan bool)
	activeGame := ActivateGame(gamestate, gameID, func(aGame *ActiveGame) {
		t.Log("cleanup function called.")
		// the cleanup function, invoked after the ActiveGame has been closed.
		// remove the game from the active games store
		activeGamesStore.remove(aGame.ID)

		// store the (updated) GameState in the long-term storage.
		gameDB.SaveGame(gameID, aGame.gameState)

		cleanupCalled <- true
	})
	// store the ActiveGame in the global store
	activeGamesStore.store(activeGame)

	// set a deadline
	deadline := make(chan bool)
	time.AfterFunc(6*time.Second, func() {
		t.Log("deadline expired")
		deadline <- true
	})

	// wait until either the cleanup function or the deadline has been called
listenLoop:
	for {
		select {
		case <-deadline:
			assert.Fail(t, "deadline reached")
			break listenLoop
		case <-cleanupCalled:
			t.Log("cleanup signal received")
			break listenLoop
		}
	}

	// check whether the game has been removed rom the active games store
	assert.False(t, activeGamesStore.contains(gameID), "game still in activeGameStore")

	// check whether the game state has been saved
	assert.True(t, gameDB.containsID(gameID), "gamestate has not been saved")
}

func TestActiveGame_Full_1v1_GameFlow(t *testing.T) {

	//TODO this test bares a race condition!!!

	// shortcut a game into the database
	gamerules := rummikub.NewDefaultRules()
	humanPlayerName := "testplayer"

	// provision the players in the game, initiate the game and store it.
	AIplayer := rummikub.NewAIPlayer("AIplayer1", rummikub.NewILPSolver(gamerules))
	humanPlayer := rummikub.NewHumanPlayer(humanPlayerName)
	gamestate := rummikub.NewGame(gamerules, 88, AIplayer, humanPlayer)
	gameID := gameDB.StoreNewGame(gamestate)

	// initiate the global logger
	//var hook *test.Hook
	logsink, hook := test.NewNullLogger()
	logger = logsink

	// run the test server
	ts := httptest.NewServer(buildServeMux())

	// // subscribe to the game using the subscription endpoint
	//send the upgrade request; create a new connection
	urlRoot := "ws:" + trimHTTPproto(ts.URL) + SUBSCRIBE + "/" + gameID + "/"
	mockClient := ConnectMockClient(humanPlayerName, urlRoot+humanPlayerName, gamerules, t)

	// loop: asynchronously wait for the player's turn and then let the client play a move
	go func() {
		for {
			// wait on the channel that tells us when the mock client GETS TOLD it is his turn.
			<-mockClient.turnAwaiter
			t.Logf("making move")
			mockClient.MakeMove()
		}
	}()

	// set a deadline to block the test while we wait for the move to be processed.
	deadline := make(chan bool)
	time.AfterFunc(5*time.Second, func() {
		deadline <- true
	})
	<-deadline

	// check if the game was won
	assert.True(t, mockClient.GetGameImage().HasBeenWon, "game has not been won")

	// Dump server logs
	t.Log("---Server logs:")
	for _, x := range hook.AllEntries() {
		t.Logf(x.String())
	}

}
