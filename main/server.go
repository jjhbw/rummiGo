package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/thisendout/apollo"
	"gitlab.com/jjhbarkeywolf/rummiGo/rummikub"
)

const (
	MB   = 1 << (10 * 2)
	PORT = ":8080"

	CONTENT_JSON = "application/json; charset=utf-8"

	// API resources
	GAME_RESOURCE   = "game_id"
	PLAYER_RESOURCE = "player_name"

	// endpoints
	GAME_ROOT = "/game"

	SUBSCRIBE = "/subscribe"
)

// declare the logger globally
var logger *logrus.Logger

//TODO store the archived games globally in-memory for now
var gameDB *GameDatabase

// store the running games in memory
var activeGamesStore *ActiveGameStore

// declare the upgrader globally
// TODO security; origin policy
var upgrader = websocket.Upgrader{}

// NewServer is the generator for the fully configured custom server object, for easier testing.
func NewServer() *http.Server {
	// start the server with the mux as the handler
	mux := buildServeMux()

	return &http.Server{
		Addr:    PORT,
		Handler: mux,

		//You should set Read, Write and Idle timeouts when dealing with untrusted clients and/or networks, so that a client can't hold up a connection by being slow to write or read.
		// An interesting (albeit slightly outdated) read regarding hardening Go HTTP servers for the open internet: https://blog.cloudflare.com/exposing-go-on-the-internet/
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 * MB,
	}
}

func buildServeMux() *mux.Router {
	mux := mux.NewRouter()

	// initiate the primary context
	ctx := context.Background()

	// build the middleware chain.
	// Apollo provides a Wrap function to inject normal http.Handler-based middleware into the chain.
	// The context will skip over the injected middleware and pass unharmed to the next context-aware handler in the chain.
	baseChain := apollo.New().With(ctx)

	// register the index handler
	mux.Handle("/", baseChain.Then(apollo.HandlerFunc(index))).Methods("GET")

	// register the game flow handlers
	mux.Handle(GAME_ROOT, baseChain.Then(apollo.HandlerFunc(newGame))).Methods("POST")
	mux.Handle(fmt.Sprintf("%v/{%v}/{%v}", GAME_ROOT, GAME_RESOURCE, PLAYER_RESOURCE), baseChain.Then(apollo.HandlerFunc(getHand))).Methods("GET")
	//mux.Handle(fmt.Sprintf("%v/{%v}", GAME_ROOT, GAME_RESOURCE), baseChain.Then(apollo.HandlerFunc(getState))).Methods("GET")

	// the upgrade route handler.
	mux.Handle(fmt.Sprintf("%v/{%v}/{%v}", SUBSCRIBE, GAME_RESOURCE, PLAYER_RESOURCE), baseChain.Then(apollo.HandlerFunc(subscribeToGame))) //.Methods("UPGRADE")

	return mux
}

func init() {
	gameDB = &GameDatabase{
		gameStore: make(map[string]*rummikub.GameState),
	}

	activeGamesStore = &ActiveGameStore{
		runningGames: make(map[string]*ActiveGame),
	}
}

func main() {

	// initiate a logger and point it to Stdout (for Docker)
	logger = logrus.New()
	logger.Out = os.Stdout

	// Log as JSON instead of the default ASCII formatter.
	logger.Formatter = &logrus.JSONFormatter{}

	// print startup message
	logger.Info("Initiating application state...")

	// start the server
	logger.Info("Starting http server at ", PORT)

	// initiate the server object
	server := NewServer()

	// run it, blocking the main thread.
	server.ListenAndServe()
}
