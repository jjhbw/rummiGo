package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"gitlab.com/jjhbarkeywolf/rummiGo/rummikub"
	"io"
	"io/ioutil"
)

func bodyToString(b io.ReadCloser) string {
	s, err := ioutil.ReadAll(b)
	if err != nil {
		panic(err)
	}
	return string(s)
}

func TestSubscriptionHandler_Errors(t *testing.T) {
	// initiate the global logger
	var hook *test.Hook
	logger, hook = test.NewNullLogger()

	// run the test server
	ts := httptest.NewServer(buildServeMux())

	//// CASE 1: game does not exist
	// attempt to subscribe to the game using the subscription endpoint
	//send the upgrade request; create a new connection
	u := "ws:" + trimHTTPproto(ts.URL) + SUBSCRIBE + "/" + "nonexistantgame" + "/" + "nonexistantplayer"
	_, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		assert.Error(t, err, "dial did not fail!")
	}
	assert.NotNil(t, resp, "response object is nil")
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Unexpected status code")
	assert.Equal(t, GAME_NOT_FOUND, hook.LastEntry().Message, "unexpected final log message")

	//// CASE 2: player not provisioned in game
	// shortcut a game into the database
	gamerules := rummikub.NewDefaultRules()
	playerName := "testplayer"
	player := rummikub.NewAIPlayer("AIplayer", rummikub.NewILPSolver(gamerules))

	// initiate the game and store it
	gamestate := rummikub.NewGame(gamerules, 88, player, rummikub.NewHumanPlayer(playerName))
	gameID := gameDB.StoreNewGame(gamestate)

	// attempt to subscribe
	u = "ws:" + trimHTTPproto(ts.URL) + SUBSCRIBE + "/" + gameID + "/" + "nonexistantplayer"
	_, resp, err = websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		assert.Error(t, err, "dial did not fail!")
	}
	assert.NotNil(t, resp, "response object is nil")
	assert.Equal(t, http.StatusPartialContent, resp.StatusCode, "Unexpected status code")
	assert.Equal(t, NO_HUMAN_PROVISIONED, hook.LastEntry().Message, "unexpected final log message")

	// log dump
	t.Logf("--Server logs:")
	for _, x := range hook.AllEntries() {
		t.Logf(x.Message)
	}
}

func TestSubscriptionHandler_NameCollision(t *testing.T) {
	// shortcut a game into the database
	gamerules := rummikub.NewDefaultRules()
	playerName := "testplayer"
	playerA := rummikub.NewAIPlayer("AIplayerA", rummikub.NewILPSolver(gamerules))

	// initiate the game and store it
	gamestate := rummikub.NewGame(gamerules, 88, playerA,
		rummikub.NewHumanPlayer(playerName))
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
	_, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Logf("dial error:", err)
		assert.Fail(t, "dial failed")
	}
	assert.NotNil(t, resp, "response object is nil")
	assert.Equal(t, 101, resp.StatusCode, "Unexpected status code")

	// Try to subscribe using the exact same name
	t.Logf("connecting to %v", u)
	_, resp, err = websocket.DefaultDialer.Dial(u, nil)
	assert.Error(t, err, "dial did not fail!")
	assert.NotNil(t, resp, "response object is nil")
	assert.Equal(t, http.StatusPartialContent, resp.StatusCode, "Unexpected status code")
}

func TestHandler_newGame(t *testing.T) {
	// initiate the global logger
	var hook *test.Hook
	logger, hook = test.NewNullLogger()

	// run the test server
	ts := httptest.NewServer(buildServeMux())
	targetURL := ts.URL + GAME_ROOT

	// initiate the desired settings
	settings := NewGameSettings{
		AIplayerNames:    []string{"jan", "kees"},
		HumanPlayerNames: []string{"henk"},
	}
	settingsBytes, err := json.Marshal(settings)
	assert.NoError(t, err, "error serializing settings")

	//send the request
	resp, err := http.Post(targetURL, CONTENT_JSON, bytes.NewBuffer(settingsBytes))
	assert.NoError(t, err, "Error sending request to mock server")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	defer resp.Body.Close()
	var gameID string
	err = json.NewDecoder(resp.Body).Decode(&gameID)

	game := gameDB.GetGame(gameID)

	assert.NotNil(t, game)

	t.Log("Log dump:")
	for _, x := range hook.AllEntries() {
		t.Logf(x.Message)
	}

}

func TestHandler_getHand(t *testing.T) {
	// shortcut a game into the database
	gamerules := rummikub.NewDefaultRules()
	playerName := "testplayer"
	player := rummikub.NewAIPlayer(playerName, rummikub.NewILPSolver(gamerules))

	// initiate the game and store it
	gamestate := rummikub.NewGame(gamerules, 88, player)
	gameID := gameDB.StoreNewGame(gamestate)

	// get the player hand
	playerHand := gamestate.GetPlayer(playerName).Hand()

	// initiate the global logger
	var hook *test.Hook
	logger, hook = test.NewNullLogger()

	// run the test server
	ts := httptest.NewServer(buildServeMux())
	targetURL := ts.URL + GAME_ROOT + "/" + gameID + "/" + playerName

	//send the request
	resp, err := http.Get(targetURL)
	assert.NoError(t, err, "Error sending request to mock server")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	defer resp.Body.Close()
	hand := []rummikub.Brick{}
	err = json.NewDecoder(resp.Body).Decode(&hand)
	assert.NotNil(t, hand)

	// check for differences
	assert.True(t, len(rummikub.BrickSliceDiff(playerHand, hand)) == 0, "hand request returned unexpected results")
	assert.True(t, len(rummikub.BrickSliceDiff(hand, playerHand)) == 0, "hand request returned unexpected results")

	t.Log("Log dump:")
	for _, x := range hook.AllEntries() {
		t.Logf(x.Message)
	}

}

func TestHandler_InvalidMethods(t *testing.T) {
	// initiate the global logger
	//var hook *test.Hook
	logger, _ = test.NewNullLogger()

	// run the test server
	ts := httptest.NewServer(buildServeMux())

	//// get table
	//resp, err := http.Post(ts.URL+GAME_ROOT+"/abcde", CONTENT_JSON, nil)
	//assert.NoError(t, err, "Error sending request to mock server")
	//assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "Unexpected status code")

	// get hand
	resp, err := http.Post(ts.URL+GAME_ROOT+"/abcde/efgh", CONTENT_JSON, nil)
	assert.NoError(t, err, "Error sending request to mock server")
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "Unexpected status code")

	// new game
	resp, err = http.Get(ts.URL + GAME_ROOT)
	assert.NoError(t, err, "Error sending request to mock server")
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "Unexpected status code")
}
