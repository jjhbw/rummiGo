package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"gitlab.com/jjhbarkeywolf/rummiGo/rummikub"
	"sync"
	"testing"
	"time"
)

// TODO clean up race conditions in this code

type MockClient struct {
	playerImage *rummikub.Player
	isFirstMove bool
	connection  *websocket.Conn
	rules       rummikub.Rules

	// the unbuffered channel to store the moves on
	movesToSend chan rummikub.Move

	// buffered (1) channel that is sent on the moment it is this player's turn
	turnAwaiter chan bool

	//lockable image of the game
	gameSnap struct {
		sync.Mutex
		state GameSnapshot
	}
}

// Initiate a mock client instance by connecting to an ActiveGame server.
func ConnectMockClient(name, url string, gamerules rummikub.Rules, t *testing.T) *MockClient {
	// initiate a MockClient instance
	m := MockClient{rules: gamerules, movesToSend: make(chan rummikub.Move), turnAwaiter: make(chan bool, 1)}

	// initiate a Player to hold the hand state and to back the move-making methods
	// Also serves to hold the player name
	p := rummikub.NewAIPlayer(name, rummikub.NewILPSolver(gamerules))
	m.playerImage = &p

	// // subscribe to the game using the subscription endpoint
	//send the upgrade request; create a new connection
	t.Logf("connecting to %v", url)
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Logf("dial error:", err)
		assert.Fail(t, "dial failed")
		panic(err)
	}
	assert.NotNil(t, resp, "response object is nil")
	assert.Equal(t, 101, resp.StatusCode, "Unexpected status code")

	// store the connection
	m.connection = conn

	// start the listener
	go func() {
		for {
			_, messageBytes, err := conn.ReadMessage()

			// if the websocket is closed (e.g. when the writePump closes it), this goroutine is returned.
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					t.Logf("error: %v", err)
					t.Fail()
				}
				break
			}

			// update the hand and game views based on the server's response
			var msg json.RawMessage
			env := Envelope{
				Payload: &msg,
			}

			if err := json.Unmarshal(messageBytes, &env); err != nil {
				t.Fatal(err)
			}

			switch env.MessageType {
			case HAND_UPDATE:
				var s HandSnapshot
				if err := json.Unmarshal(msg, &s); err != nil {
					t.Fatal(err)
				}

				m.playerImage.SetHand(s.Hand)
				m.isFirstMove = s.IsFirstMove

			case GAME_SNAPSHOT:
				var s GameSnapshot
				if err := json.Unmarshal(msg, &s); err != nil {
					t.Fatal(err)
				}

				m.gameSnap.Lock()
				m.gameSnap.state = s
				t.Log(m.gameSnap.state.Table)
				m.gameSnap.Unlock()
				myTurn := m.gameSnap.state.CurrentPlayer == m.playerImage.Name
				if myTurn {
					m.turnAwaiter <- true
				}

			default:
				t.Fatalf("unknown message type: %q", env.MessageType)
			}
		}
	}()

	// start the sender (to make sure there is only one goroutine writing to the channel
	go func() {
		for {
			move, ok := <-m.movesToSend
			if !ok {
				t.Logf("Move sending channel closed. Terminating mock client.")
				return
			}

			w, err := m.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			msg, err := json.Marshal(Envelope{
				MessageType: MOVE_PROPOSAL,
				Payload: struct {
					Table []rummikub.BrickCombination `json:"table"`
				}{move.Arrangement},
			})

			fmt.Println(string(msg))
			if err != nil {
				panic(err)
			}
			w.Write(msg)
			if err := w.Close(); err != nil {
				return
			}
		}
	}()

	return &m
}

// Block until it is the MockClients turn
func (m *MockClient) AwaitTurn() {

	yourTurn := make(chan bool)

	go func() {
	waitLoop:
		for {
			m.gameSnap.Lock()
			letsgo := m.gameSnap.state.CurrentPlayer == m.playerImage.Name
			m.gameSnap.Unlock()
			if letsgo {
				break waitLoop
			}

			time.Sleep(100)
		}
		yourTurn <- true
	}()

	<-yourTurn
}

// Makes a move based on the reflection of the game state in the struct.
func (m *MockClient) MakeMove() {
	gameImage := m.GetGameImage()
	// if the game has been won, close the send channel
	if gameImage.HasBeenWon {
		close(m.movesToSend)
		return
	}

	// check if it is the player's first move
	minVal := 0
	if m.isFirstMove {
		minVal = m.rules.FirstMoveValue
	}

	// build the move using the solver.
	move := m.playerImage.MakeMove(gameImage.Table, minVal)

	// put the move in the send queue
	m.movesToSend <- move
}

// retrieve the image that the mock client has of the game state
func (m *MockClient) GetGameImage() GameSnapshot {
	m.gameSnap.Lock()
	defer m.gameSnap.Unlock()
	return m.gameSnap.state
}
