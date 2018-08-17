package main

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gitlab.com/jjhbarkeywolf/rummiGo/rummikub"
	"html/template"
	"net/http"
	"time"
)

func index(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// populate the log parser with the request context
	log := logger.WithFields(logrus.Fields{
		"user_ip": r.RemoteAddr,
		"url":     r.URL,
	})

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		log.Errorf("invalid method %v, want %v.", r.Method, http.MethodGet)
		return
	}

	t, _ := template.ParseFiles("index.html")
	t.Execute(w, nil)

	log.Info("served upload form")

	return
}

type NewGameSettings struct {
	AIplayerNames    []string `json:"ai_player_names"`
	HumanPlayerNames []string `json:"human_player_names"`
}

func newGame(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// populate the log parser with the request context
	log := logger.WithFields(logrus.Fields{
		"user_ip": r.RemoteAddr,
		"url":     r.URL,
	})

	var settings NewGameSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "error deserializing json", http.StatusBadRequest)
		log.Errorf("error deserializing json")
		return
	}

	// build a set of game rules
	rules := rummikub.NewDefaultRules()

	players := []rummikub.Player{}
	for _, name := range settings.AIplayerNames {
		players = append(players, rummikub.NewAIPlayer(name, rummikub.NewILPSolver(rules)))
	}

	for _, name := range settings.HumanPlayerNames {
		players = append(players, rummikub.NewHumanPlayer(name))
	}

	// initiate a new game
	game := rummikub.NewGame(
		rules,
		time.Now().Unix(),
		players...,
	)

	// store the new game under a new random ID
	gameId := gameDB.StoreNewGame(game)

	// send the game ID down
	json.NewEncoder(w).Encode(gameId)

	log.WithFields(logrus.Fields{
		"game_id": gameId,
	}).Info("started")

	return
}

func getHand(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// populate the log parser with the request context
	log := logger.WithFields(logrus.Fields{
		"user_ip": r.RemoteAddr,
		"url":     r.URL,
	})

	// get variables from URL
	vars := mux.Vars(r)

	gameId, ok := vars[GAME_RESOURCE]
	if !ok {
		http.Error(w, "game resource not specified", http.StatusBadRequest)
		log.Errorf("game resource not specified")
		return
	}

	playerName, ok := vars[PLAYER_RESOURCE]
	if !ok {
		http.Error(w, "player resource not specified", http.StatusBadRequest)
		log.Errorf("player resource not specified")
		return
	}

	log = log.WithFields(logrus.Fields{
		"game_id":     gameId,
		"player_name": playerName,
	})

	// find corresponding resources
	game := gameDB.GetGame(gameId)
	if game == nil {
		http.Error(w, "game not found", http.StatusNoContent)
		log.Errorf("game not found")
		return
	}

	player := game.GetPlayer(playerName)
	if player == nil {
		http.Error(w, "player not found", http.StatusNoContent)
		log.Errorf("player not found")
		return
	}

	// get the player's hand
	hand := player.Hand()

	// send the hand bricks down
	json.NewEncoder(w).Encode(hand)

	log.Info("hand served")

	return
}

const (
	// Note that any consumer of this API should use the provided HTTP status codes as much as possible and avoid relying on these messages.
	GAME_RESOURCE_NOT_SPECIFIED   = "game resource not specified"
	PLAYER_RESOURCE_NOT_SPECIFIED = "player resource not specified"
	GAME_NOT_FOUND                = "Game not found. Start one first."
	NO_HUMAN_PROVISIONED          = "No human player with this name has been provisioned"
	PLAYER_ALREADY_SUBSCRIBED     = "A player with this name has already subscribed to this game"
	ERROR_UPGRADING_CONNECTION    = "unexpected error upgrading connection"
)

// connect to a certain game by ID using a websocket.
func subscribeToGame(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// populate the log parser with the request context
	log := logger.WithFields(logrus.Fields{
		"user_ip": r.RemoteAddr,
		"url":     r.URL,
	})

	// get variables from URL
	vars := mux.Vars(r)

	gameID, ok := vars[GAME_RESOURCE]
	if !ok {
		http.Error(w, GAME_RESOURCE_NOT_SPECIFIED, http.StatusBadRequest)
		log.Error(GAME_RESOURCE_NOT_SPECIFIED)
		return
	}

	playAs, ok := vars[PLAYER_RESOURCE]
	if !ok {
		http.Error(w, PLAYER_RESOURCE_NOT_SPECIFIED, http.StatusBadRequest)
		log.Errorf(PLAYER_RESOURCE_NOT_SPECIFIED)
		return
	}

	log = log.WithFields(logrus.Fields{
		"game_id":     gameID,
		"player_name": playAs,
	})

	// // find corresponding game.
	// check if the game is already started. If so, reject the subscription.
	// If the game is not present in the ActiveGameStore, check if it exists in archived form in the database.
	var activeGame *ActiveGame
	activeGame = activeGamesStore.get(gameID)
	if activeGame == nil {
		// pull the game from the database
		game := gameDB.GetGame(gameID)
		if game == nil {
			http.Error(w, GAME_NOT_FOUND, http.StatusNoContent)
			log.Errorf(GAME_NOT_FOUND)
			return
		}
		// activate the game.
		activeGame = ActivateGame(game, gameID, func(aGame *ActiveGame) {
			// the cleanup function, invoked after the ActiveGame has been closed.

			// remove the game from the active games store
			activeGamesStore.remove(aGame.ID)

			// store the (updated) GameState in the long-term storage.
			gameDB.SaveGame(gameID, aGame.gameState)

			log.Info("Game inactivated")
		})

		// store the ActiveGame in the global store
		activeGamesStore.store(activeGame)

		log.Info("Game activated")
	}

	// check if the player name has been provisioned (human players only)
	player := activeGame.gameState.GetPlayer(playAs)
	if player == nil || !player.Human {
		http.Error(w, NO_HUMAN_PROVISIONED, http.StatusPartialContent)
		log.Error(NO_HUMAN_PROVISIONED)
		return
	}

	// check if the player has already been subscribed.
	if activeGame.IsPlayerSubscribed(playAs) {
		http.Error(w, PLAYER_ALREADY_SUBSCRIBED, http.StatusPartialContent)
		log.Error(PLAYER_ALREADY_SUBSCRIBED)
		return
	}

	// // If all checks passed, connect the player to the game.
	// upgrade the http connection to a websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, ERROR_UPGRADING_CONNECTION, 500)
		log.Errorf(ERROR_UPGRADING_CONNECTION)
		return
	}

	// Connect the player to the ActiveGame
	activeGame.connectPlayer(conn, player)
	log.Info("Player connected to game")

	// return 101 in case of success (default)
	return
}
