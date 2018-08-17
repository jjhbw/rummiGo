package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"gitlab.com/jjhbarkeywolf/rummiGo/rummikub"
	"sync"
	"time"
)

// TODO process proposed moves
// TODO write test for user inputs (proposed moves)
// TODO improve type-safety by emphasizing channel directions where possible
// TODO do we want panic (with recovery) or only soft errors?
// TODO write a cleanup protocol (storing games and closing connections) that fires when the SERVER is terminated (using a builtin hook of http.Server?)
// TODO watch out for blockages caused by congested channels
// TODO keep channels simple by having only one writer
// TODO implement rate limiting in the readPump
//see: "- However if you have several senders and several receivers on the "quit" channel, then you have a problem: closing a closed channel will panic."

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1 << (10 * 2) //1MB //TODO is not sensible...

	// the period of the game's heartbeat
	gameHeartBeatPeriod = 5 * time.Second

	// outgoing message types
	ERROR_MESSAGE  = "error_message"
	HAND_UPDATE    = "hand_update"
	GAME_SNAPSHOT  = "game_snapshot"
	MOVE_REJECTION = "move_rejected"

	// incoming messages
	MOVE_PROPOSAL = "move"

	// error responses
	UNKNOWN_MESSAGE_TYPE = "unknown message type"
)

// ActiveGame contains a GameState struct and the websocket connections of the involved players.
// Note that at this point, the game is not yet running.
type ActiveGame struct {
	// the game ID
	ID string

	// A mutex to synchronize access to the gameState
	sync.Mutex

	// the logger with several pre-populated fields
	logSink *logrus.Entry

	// the game state
	gameState *rummikub.GameState

	// contains a single client for each player (by player name).
	connectedClients map[string]*Client

	// the cleanup function, which is called when the gameManager returns.
	onClose func(aGame *ActiveGame)

	// Subscription requests from the clients.
	subscribe chan *Client

	// unsubscribe requests from clients.
	unsubscribe chan *Client

	// the channel used to inactivate the game and all relevant connections without concurrency issues.
	closer chan bool

	// this channel contains the move proposals: i.e. candidate moves pending approval
	moveCandidates chan MoveProposal
}

func (aGame *ActiveGame) IsPlayerSubscribed(name string) bool {
	aGame.Lock()
	defer aGame.Unlock()
	_, present := aGame.connectedClients[name]
	return present
}

// ActivateGame wraps a rummikub.GameState into an ActiveGame.
// The cleanup function is called when the game ActiveGame is terminated to allow for removal of any external references.
func ActivateGame(g *rummikub.GameState, gameID string, cleanupFunc func(aGame *ActiveGame)) *ActiveGame {
	aGame := &ActiveGame{
		ID:               gameID,
		gameState:        g,
		logSink:          logger.WithField("game_id", gameID),
		connectedClients: make(map[string]*Client),
		onClose:          cleanupFunc,
		subscribe:        make(chan *Client),
		unsubscribe:      make(chan *Client),
		closer:           make(chan bool),

		// TODO double check whether we want this channel to be buffered.
		moveCandidates: make(chan MoveProposal, 10),
	}

	// activate the gameManager.
	go aGame.gameManager()

	return aGame
}

// A wrapper around the server messages containing an identifier for the type of payload.
type Envelope struct {
	// identifier so the client knows what type of message this is
	MessageType string      `json:"message_type"`
	Payload     interface{} `json:"payload"`
}

type GameSnapshot struct {
	// the current state of the table
	Table []rummikub.BrickCombination `json:"table"`

	// player whose turn it is
	CurrentPlayer string `json:"current_player"`

	// all players in the game and whether they are subscribed (true or false).
	PlayerStatuses map[string]bool `json:"player_statuses"`

	// whether the game has been won. If true, the winner is the current player.
	HasBeenWon bool `json:"has_been_won"`
}

// snapshot generates a snapshot of the current ActiveGame to be sent down to the clients.
func (aGame *ActiveGame) snapshot() *GameSnapshot {
	// avoid concurrency issues and false-positive race conditions in tests.
	aGame.Lock()
	defer aGame.Unlock()

	statuses := make(map[string]bool)
	for _, p := range aGame.gameState.Players {
		//check if the player is subscribed (i.e. a Client exists)
		_, subbed := aGame.connectedClients[p.Name]
		statuses[p.Name] = subbed
	}
	return &GameSnapshot{
		aGame.gameState.Table(),
		aGame.gameState.CurrentPlayer().Name,
		statuses,
		aGame.gameState.HasBeenWon(),
	}
}

// BroadcastPublicGameState sends a snapshot of the game down to all subscribed clients.
// Should be called when the game state is updated.
func (aGame *ActiveGame) BroadcastPublicGameState() {
	// get the snapshot and serialize it
	wrappedSnap := Envelope{
		GAME_SNAPSHOT,
		aGame.snapshot(),
	}

	for _, client := range aGame.connectedClients {
		client.Send(wrappedSnap)
	}
}

// gracefully close the game
func (aGame *ActiveGame) Close() {
	aGame.closer <- true
}

func (aGame *ActiveGame) connectPlayer(connection *websocket.Conn, player *rummikub.Player) {
	c := Client{
		player,
		aGame,
		connection,
		aGame.logSink.WithFields(logrus.Fields{
			"client_name": player.Name,
		}),
		make(chan []byte),
	}

	// tell the gameManager to subscribe the player to the game.
	aGame.subscribe <- &c

	// activate the client I/O pumps that handle the direct communication with the websocket.
	go c.readPump()
	go c.writePump()

}

// check if the game is ready to start (i.e. all provisioned human players are connected).
func (aGame *ActiveGame) ReadyToStart() bool {
	for _, p := range aGame.gameState.Players {
		if p.Human {
			if !aGame.IsPlayerSubscribed(p.Name) {
				return false
			}
		}
	}
	return true
}

// gameManager does several things:
// - Centrally manages writes to the clients map (subscriptions and unsubscribe actions)
// - Handles the graceful termination of the receiver ActiveGame by invoking the provided onClose() function.
func (aGame *ActiveGame) gameManager() {

	// call the cleanup function when the gameManager returns.
	defer aGame.onClose(aGame)

	// activate the heartbeat.
	heartbeat := time.NewTicker(gameHeartBeatPeriod)
	defer heartbeat.Stop()

	for {
		select {

		// centralize write access to the clients map
		case client := <-aGame.subscribe:
			aGame.connectedClients[client.player.Name] = client
			logger.Infof("Player %v subscribed to the game.", client.player.Name)

			// update the players on the game state now that the new player has joined
			aGame.BroadcastPublicGameState()

			// send the newly connected player the contents of his hand
			client.SyncHandStatus()

			// If all players have joined, start the game by running the AI move
			if aGame.ReadyToStart() {
				//TODO make RunAITurns non-recursive so state updates can be synced after every call
				logger.Info("All players connected. Game ready to start.")
				aGame.gameState.RunAITurns()
				aGame.BroadcastPublicGameState()
			}

		case client := <-aGame.unsubscribe:
			// unsubscribe a client from the game, if it is still subscribed.
			playerName := client.player.Name
			if _, ok := aGame.connectedClients[playerName]; ok {
				delete(aGame.connectedClients, playerName)
				close(client.send)
				logger.Infof("Unsubscribed %v", client.player.Name)
			}

			// if all players have unsubscribed, we return this function,
			// invoking the ActiveGame's graceful termination onClose() function.
			if len(aGame.connectedClients) == 0 {
				logger.Info("Finished unsubscribing. No clients are connected. Terminating gameManager routine.")
				return
			}

			// update the remaining players on the game state now that a player has left.
			aGame.BroadcastPublicGameState()

		//If no clients are subscribed, return this function.
		// If clients are subscribed, unsubscribe all clients.
		case <-aGame.closer:
			//fmt.Println("closer called")
			logger.Info("Close channel invoked. Unsubscribing remaining clients...")
			if len(aGame.connectedClients) == 0 {
				logger.Info("No clients are connected. Terminating gameManager routine.")
				return
			}
			// asynchronously tell the client to unsubscribe itself.
			// doing this synchronously would block this goroutine.
			for _, client := range aGame.connectedClients {
				go client.Unsubscribe()
			}

		// process heartbeat pings to initiate regular checks.
		case <-heartbeat.C:
			// kill the game if no clients are connected
			if len(aGame.connectedClients) == 0 {
				logger.Info("Heartbeat check showed no clients are connected. Terminating gameManager routine.")
				return
			}

		case candidateMove := <-aGame.moveCandidates:
			// Process the candidate move
			logger.Infof("Processing submitted candidate move by %v", candidateMove.client.player.Name)

			// check if the move counts as a forfeiture (encoded as empty []rummikub.BrickCombination)
			var move rummikub.Move
			if len(candidateMove.table) == 0 {
				move = rummikub.NewMove(candidateMove.client.player.Name, aGame.gameState.Table())
			} else {
				move = rummikub.NewMove(candidateMove.client.player.Name, candidateMove.table)
			}

			// check the legality of the move against the game state
			accepted, reason := aGame.gameState.ProcessMove(move)

			if accepted {
				logger.Infof("Move submitted by %v was accepted (why: %v). Synchronizing game state and running AI turns if applicable...", candidateMove.client.player.Name, reason)
				// synchronize the new game state to the clients
				//candidateMove.client.SyncHandStatus()
				//aGame.BroadcastPublicGameState()

				// run the AI turns if applicable.
				//TODO handle game cycling in a neater way. Each AI turn should trigger a broadcast.
				aGame.gameState.RunAITurns()
				aGame.BroadcastPublicGameState()
				candidateMove.client.SyncHandStatus()

			} else {
				// inform the client of the reason his move was rejected
				logger.Infof("Move submitted by %v was not accepted for reason: %v", candidateMove.client.player.Name, reason)
				candidateMove.client.Send(Envelope{MessageType: MOVE_REJECTION, Payload: reason})
			}

		}
	}
}

type Client struct {
	player     *rummikub.Player
	activeGame *ActiveGame
	conn       *websocket.Conn

	// client-specific logger object. Amends structured messages with client name and game ID.
	//TODO check if timestamps in this Entry struct are not fixed at creation of Entry.
	logSink *logrus.Entry

	// this channel feeds directly to the writePump
	// closing it will kill the Write pump, which kills the read pump as it closes the websocket.
	send chan []byte
}

type HandSnapshot struct {
	MessageType string           `json:"message_type"`
	Hand        []rummikub.Brick `json:"hand"`
	IsFirstMove bool             `json:"is_first_move"`
}

// NewHandSnapshot produces parametrised HandSnapshot structs.
func (c *Client) NewHandSnapshot() HandSnapshot {
	c.activeGame.Lock()
	defer c.activeGame.Unlock()
	return HandSnapshot{
		HAND_UPDATE,
		c.player.Hand(),
		c.activeGame.gameState.IsFirstMove(c.player.Name),
	}
}

// SyncHandStatus tells the player the contents of his hand
func (c *Client) SyncHandStatus() {
	update := Envelope{
		HAND_UPDATE,
		c.NewHandSnapshot(),
	}
	c.Send(update)
}

// Unsubscribe the client from the game, initiating its graceful termination.
func (c *Client) Unsubscribe() {
	c.activeGame.unsubscribe <- c
}

// Send is a convenience method to send arbitrary serialized data to the write pump through the send channel.
func (c *Client) Send(message Envelope) {
	data, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	c.send <- data
}

func (c *Client) SendError(errorMsg string) {
	data, err := json.Marshal(Envelope{
		ERROR_MESSAGE,
		errorMsg,
	})
	if err != nil {
		panic(err)
	}
	c.send <- data
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {

	// Note that this function can only return if the channel is closed.
	// ensure the connection is closed if this function returns.
	defer func() {
		c.conn.Close()
		c.logSink.Info("Read pump and websocket connection closed")
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// read messages coming from the websocket.
	for {
		_, message, err := c.conn.ReadMessage()

		// if the websocket is closed (e.g. when the writePump closes it), this goroutine is returned.
		if err != nil {
			// if the websocket is closed unexpectedly, log it.
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.logSink.WithField("error", err).Error("unexpected websocket close error.")
			}
			return
		}

		// process the incoming message //TODO concurrently?
		c.processUserMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {

	// the ticker for the ping messages.
	ticker := time.NewTicker(pingPeriod)

	// If the function returns, the websocket is closed and an unsubscribe action is requested.
	defer func() {
		c.logSink.Info("Writepump closed. Unsubscribing Client and closing websocket.")
		c.Unsubscribe()
		ticker.Stop()
		c.conn.Close()
	}()

	// block for messages.
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The channel was closed by the ActiveGame.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.logSink.Info("The 'send' channel was closed by the ActiveGame. Terminating Writepump...")
				return
			}

			// write queued messages to the websocket.
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.logSink.WithField("error", err).Error("An error occurred writing to the client's websocket. Considering it dead. Terminating client...")
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				c.logSink.WithField("error", err).Error("An error occurred closing the Writer to the client's websocket. Considering the socket dead. Terminating client...")
				return
			}

		// send a ping
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				c.logSink.WithField("error", err).Error("An error writing the heartbeat to the client's websocket. Considering the socket dead. Terminating client...")
				return
			}
		}
	}
}

// The move candidate to be processed by the game manager.
// Note that forfeitures are encoded as empty table slices.
type MoveProposal struct {
	table  []rummikub.BrickCombination
	client *Client
}

func (c *Client) processUserMessage(messageBytes []byte) {
	var msg json.RawMessage
	env := Envelope{
		Payload: &msg,
	}
	sublogger := c.logSink.WithFields(logrus.Fields{
		"payload": string(messageBytes), //TODO for debugging only. Remove soon.
	})

	if err := json.Unmarshal(messageBytes, &env); err != nil {
		sublogger.Error("error unmarshalling message sent by user")
		c.SendError(UNKNOWN_MESSAGE_TYPE)
		return
	}

	switch env.MessageType {
	case MOVE_PROPOSAL:
		sublogger.Info("Processing incoming move proposal.")

		// construct a move proposal struct from the message
		// TODO Nice inline struct you got there, would be a shame if something were to happen to it... (needed for unmarshalling null values i.e. empty []BrickCombination)
		var tableProposal struct{ Table []rummikub.BrickCombination }
		if err := json.Unmarshal(msg, &tableProposal); err != nil {
			sublogger.Error("Error unmarshalling payload of user message.")
			c.SendError(UNKNOWN_MESSAGE_TYPE)
			return
		}

		// send the move proposal to the gameManager for evaluation.
		prop := MoveProposal{table: tableProposal.Table, client: c}
		c.activeGame.moveCandidates <- prop
		return

	default:
		c.SendError(UNKNOWN_MESSAGE_TYPE)
		sublogger.Errorf("Unknown message type sent by user: %v", env.MessageType)
		return
	}

}
