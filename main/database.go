package main

import (
	"crypto/md5"
	"fmt"
	"gitlab.com/jjhbarkeywolf/rummiGo/rummikub"
	"io"
	"strconv"
	"sync"
	"time"
)

// TODO This is bullshit. Access to the ActiveGameStore should be mutex-locked in the handler!!!!!

// ActiveGameStore represents the in-memory datastructure used to store active games.
type ActiveGameStore struct {
	runningGames map[string]*ActiveGame
	sync.Mutex
}

func (db *ActiveGameStore) contains(ID string) bool {
	db.Lock()
	defer db.Unlock()
	_, ok := db.runningGames[ID]
	return ok
}

// get returns a nil pointer if the game has not been stored yet.
func (db *ActiveGameStore) get(ID string) *ActiveGame {
	db.Lock()
	defer db.Unlock()
	return db.runningGames[ID]
}

func (db *ActiveGameStore) store(game *ActiveGame) {
	if db.contains(game.ID) {
		panic("ActiveGame already stored in ActiveGameStore")
	}
	db.Lock()
	defer db.Unlock()
	db.runningGames[game.ID] = game
}

// ensures that the game is removed. Will fail silently if the provided gameID is not found.
func (db *ActiveGameStore) remove(gameID string) {
	db.Lock()
	defer db.Unlock()
	delete(db.runningGames, gameID)
}

// GameDatabase supplies the interface to the archived games.
// TODO: for now nothing more than an in-memory k:v store.
type GameDatabase struct {
	sync.Mutex
	gameStore map[string]*rummikub.GameState
}

func (db *GameDatabase) GetGame(ID string) *rummikub.GameState {
	db.Lock()
	defer db.Unlock()
	return db.gameStore[ID]
}

func (db *GameDatabase) containsID(id string) bool {
	db.Lock()
	defer db.Unlock()
	_, ok := db.gameStore[id]
	return ok
}

func (db *GameDatabase) SaveGame(id string, game *rummikub.GameState) {
	db.Lock()
	defer db.Unlock()
	db.gameStore[id] = game
}

// StoreNewGame stores the game under a new ID and returns the ID
func (db *GameDatabase) StoreNewGame(game *rummikub.GameState) string {
	// generate an ID for the game. Small built in check ensures ID is unique.
	gameID := getRandomShortString()
	for db.containsID(gameID) {
		gameID = getRandomShortString()
	}
	db.Lock()
	defer db.Unlock()
	db.gameStore[gameID] = game
	return gameID
}

// use an md5 hash to produce a random string of a fixed size.
func getRandomShortString() string {
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(time.Now().Unix(), 10))
	return fmt.Sprintf("%x", h.Sum(nil))[0:6]
}
