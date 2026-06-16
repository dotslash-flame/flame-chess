package hub

//matchmakng engine
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	mrand "math/rand/v2"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/game"
	"github.com/dotslash-flame/flame-chess/internal/recorder"
	"github.com/dotslash-flame/flame-chess/internal/store"
	"github.com/dotslash-flame/flame-chess/internal/wire"
)

type gameActor = game.Actor

type GameAction struct {
	GameID string
	Kind   string // move or resign or draw_offer or draw_respond
	UCI    string
	Accept bool
}

type poolKey struct {
	base      int
	increment int
}

type challenge struct {
	token       string
	creator     string
	creatorName string
	opponent    string
	base        int
	increment   int
}

type Options struct {
	Rng    func(n int) int  // picks colors
	NextID func() string    // game id generator
	Now    func() time.Time // clock start time
	Store  *store.Store     // persistence; nil → games are not recorded/rated
}

type Hub struct {
	cmds chan command

	online      map[string]wire.Conn
	pools       map[poolKey][]string
	games       map[string]*game.Actor
	userGame    map[string]string
	gamePlayers map[string][]string
	challenges  map[string]challenge

	rng    func(n int) int
	nextID func() string
	now    func() time.Time
	run    func(*game.Actor)
	store  *store.Store
}

func New(opts Options) *Hub {
	h := &Hub{
		cmds:        make(chan command, 64),
		online:      map[string]wire.Conn{},
		pools:       map[poolKey][]string{},
		games:       map[string]*game.Actor{},
		userGame:    map[string]string{},
		gamePlayers: map[string][]string{},
		challenges:  map[string]challenge{},
		rng:         opts.Rng,
		nextID:      opts.NextID,
		now:         opts.Now,
		run:         func(a *game.Actor) { go a.Run() },
		store:       opts.Store,
	}
	if h.rng == nil {
		h.rng = func(n int) int { return mrand.IntN(n) }
	}
	if h.nextID == nil {
		h.nextID = randomID
	}
	if h.now == nil {
		h.now = time.Now
	}
	return h
}

func randomID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (h *Hub) Run(done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case c := <-h.cmds:
			h.handle(c)
		}
	}
}

func (h *Hub) Register(c wire.Conn)   { h.enqueue(registerCmd{conn: c}) }
func (h *Hub) Unregister(c wire.Conn) { h.enqueue(unregisterCmd{conn: c}) }
func (h *Hub) QueueJoin(userID, category string, base, increment int) {
	h.enqueue(queueJoinCmd{userID: userID, category: category, base: base, increment: increment})
}
func (h *Hub) QueueLeave(userID string)          { h.enqueue(queueLeaveCmd{userID: userID}) }
func (h *Hub) Route(userID string, a GameAction) { h.enqueue(routeCmd{userID: userID, action: a}) }

func (h *Hub) CreateChallenge(creator, creatorName, opponent string, base, increment int) string {
	reply := make(chan string, 1)
	h.enqueue(createChallengeCmd{
		creator: creator, creatorName: creatorName, opponent: opponent,
		base: base, increment: increment, reply: reply,
	})
	return <-reply
}

func (h *Hub) CreateDirectChallenge(userID, opponentID string, base, increment int) {
	h.enqueue(createChallengeCmd{creator: userID, opponent: opponentID, base: base, increment: increment})
}

func (h *Hub) AcceptChallenge(userID, token string) {
	h.enqueue(acceptChallengeCmd{userID: userID, token: token})
}
func (h *Hub) DeclineChallenge(userID, token string) {
	h.enqueue(declineChallengeCmd{userID: userID, token: token})
}
func (h *Hub) CancelChallenge(userID, token string) {
	h.enqueue(cancelChallengeCmd{userID: userID, token: token})
}

func (h *Hub) enqueue(c command) { h.cmds <- c }

type command interface{ isCommand() }

type registerCmd struct{ conn wire.Conn }
type unregisterCmd struct{ conn wire.Conn }
type queueJoinCmd struct {
	userID    string
	category  string
	base      int
	increment int
}
type queueLeaveCmd struct{ userID string }
type routeCmd struct {
	userID string
	action GameAction
}
type gameEndedCmd struct{ gameID string }
type createChallengeCmd struct {
	creator     string
	creatorName string
	opponent    string
	base        int
	increment   int
	reply       chan string // non-nil for synchronous (REST link) creates
}
type acceptChallengeCmd struct {
	userID string
	token  string
}
type declineChallengeCmd struct {
	userID string
	token  string
}
type cancelChallengeCmd struct {
	userID string
	token  string
}

func (registerCmd) isCommand()         {}
func (unregisterCmd) isCommand()       {}
func (queueJoinCmd) isCommand()        {}
func (queueLeaveCmd) isCommand()       {}
func (routeCmd) isCommand()            {}
func (gameEndedCmd) isCommand()        {}
func (createChallengeCmd) isCommand()  {}
func (acceptChallengeCmd) isCommand()  {}
func (declineChallengeCmd) isCommand() {}
func (cancelChallengeCmd) isCommand()  {}

func (h *Hub) handle(c command) {
	switch cmd := c.(type) {
	case registerCmd:
		h.handleRegister(cmd)
	case unregisterCmd:
		h.handleUnregister(cmd)
	case queueJoinCmd:
		h.handleQueueJoin(cmd)
	case queueLeaveCmd:
		h.handleQueueLeave(cmd)
	case routeCmd:
		h.handleRoute(cmd)
	case gameEndedCmd:
		h.handleGameEnded(cmd)
	case createChallengeCmd:
		h.handleCreateChallenge(cmd)
	case acceptChallengeCmd:
		h.handleAcceptChallenge(cmd)
	case declineChallengeCmd:
		h.handleDeclineChallenge(cmd)
	case cancelChallengeCmd:
		h.handleCancelChallenge(cmd)
	}
}

func (h *Hub) handleRegister(c registerCmd) {
	uid := c.conn.UserID()
	if old, ok := h.online[uid]; ok && old != c.conn {
		old.Close()
	}
	h.online[uid] = c.conn
	h.broadcastOnlineCount()
	h.broadcastOnlineList()
}

func (h *Hub) handleUnregister(c unregisterCmd) {
	uid := c.conn.UserID()
	if cur, ok := h.online[uid]; !ok || cur != c.conn {
		return
	}
	delete(h.online, uid)
	h.removeFromPools(uid)
	h.dropChallengesBy(uid)
	if gid, ok := h.userGame[uid]; ok {
		if a, ok := h.games[gid]; ok {
			a.PlayerGone(uid)
		}
	}
	h.broadcastOnlineCount()
	h.broadcastOnlineList()
}

func (h *Hub) handleQueueJoin(c queueJoinCmd) {
	conn, online := h.online[c.userID]
	if !online {
		return
	}
	if _, inGame := h.userGame[c.userID]; inGame {
		return
	}
	key := poolKey{base: c.base, increment: c.increment}
	h.removeFromPools(c.userID)

	for len(h.pools[key]) > 0 {
		oppID := h.pools[key][0]
		h.pools[key] = h.pools[key][1:]
		oppConn, ok := h.online[oppID]
		if !ok {
			continue
		}
		if _, inGame := h.userGame[oppID]; inGame {
			continue
		}
		h.startGame(key, oppConn, conn)
		return
	}
	h.pools[key] = append(h.pools[key], c.userID)
	conn.Send(wire.NewQueueWaiting())
}

func (h *Hub) startGame(key poolKey, head, joiner wire.Conn) {
	white, black := head, joiner
	if h.rng(2) == 1 {
		white, black = joiner, head
	}
	gid := h.nextID()
	g := game.NewGame(key.base, key.increment, h.now())

	var rec game.Recorder
	if h.store != nil {
		dbID, err := h.store.InsertActiveGame(context.Background(), white.UserID(), black.UserID(), string(g.Category()), key.base, key.increment)
		if err != nil {
			log.Printf("hub: insert active game: %v", err)
		} else {
			rec = recorder.New(h.store, dbID, white.UserID(), black.UserID())
		}
	}

	actor := game.NewActor(gid, g, white, black, func(id string) { h.enqueue(gameEndedCmd{gameID: id}) }, rec)

	h.games[gid] = actor
	h.userGame[white.UserID()] = gid
	h.userGame[black.UserID()] = gid
	h.gamePlayers[gid] = []string{white.UserID(), black.UserID()}
	h.run(actor)

	clocks := wire.Clocks{
		WhiteMs: int64(key.base) * 1000,
		BlackMs: int64(key.base) * 1000,
	}
	fen := g.FEN()
	white.Send(wire.GameStart{
		Type: wire.TypeGameStart, GameID: gid, Color: "white",
		Opponent: black.DisplayName(), Clocks: clocks, FEN: fen,
	})
	black.Send(wire.GameStart{
		Type: wire.TypeGameStart, GameID: gid, Color: "black",
		Opponent: white.DisplayName(), Clocks: clocks, FEN: fen,
	})
}

func (h *Hub) handleQueueLeave(c queueLeaveCmd) {
	h.removeFromPools(c.userID)
}

func (h *Hub) handleRoute(c routeCmd) {
	conn := h.online[c.userID]
	gid, inGame := h.userGame[c.userID]
	if !inGame {
		sendErr(conn, wire.CodeNotInGame, "not in a game")
		return
	}
	if c.action.GameID != "" && c.action.GameID != gid {
		sendErr(conn, wire.CodeUnknownGame, "unknown game")
		return
	}
	a, ok := h.games[gid]
	if !ok {
		sendErr(conn, wire.CodeNotInGame, "not in a game")
		return
	}
	switch c.action.Kind {
	case "move":
		a.Move(c.userID, c.action.UCI)
	case "resign":
		a.Resign(c.userID)
	case "draw_offer":
		a.OfferDraw(c.userID)
	case "draw_respond":
		a.RespondDraw(c.userID, c.action.Accept)
	}
}

func (h *Hub) handleGameEnded(c gameEndedCmd) {
	for _, uid := range h.gamePlayers[c.gameID] {
		delete(h.userGame, uid)
	}
	delete(h.gamePlayers, c.gameID)
	delete(h.games, c.gameID)
}

func (h *Hub) removeFromPools(userID string) {
	for key, q := range h.pools {
		filtered := q[:0:0]
		for _, uid := range q {
			if uid != userID {
				filtered = append(filtered, uid)
			}
		}
		if len(filtered) == 0 {
			delete(h.pools, key)
		} else {
			h.pools[key] = filtered
		}
	}
}

func (h *Hub) broadcastOnlineCount() {
	msg := wire.NewOnlineCount(len(h.online))
	for _, c := range h.online {
		c.Send(msg)
	}
}

func (h *Hub) broadcastOnlineList() {
	users := make([]wire.OnlineUser, 0, len(h.online))
	for uid, c := range h.online {
		users = append(users, wire.OnlineUser{UID: uid, Name: c.DisplayName()})
	}
	msg := wire.NewOnlineList(users)
	for _, c := range h.online {
		c.Send(msg)
	}
}

func (h *Hub) handleCreateChallenge(c createChallengeCmd) {
	creatorConn := h.online[c.creator]
	creatorName := c.creatorName
	if creatorName == "" && creatorConn != nil {
		creatorName = creatorConn.DisplayName()
	}

	if c.opponent != "" {
		if c.opponent == c.creator {
			h.replyChallenge(c.reply, "")
			sendErr(creatorConn, wire.CodeChallengeSelf, "you can't challenge yourself")
			return
		}
		oppConn, online := h.online[c.opponent]
		if !online {
			h.replyChallenge(c.reply, "")
			sendErr(creatorConn, wire.CodeOpponentOffline, "they're offline")
			return
		}
		if _, busy := h.userGame[c.opponent]; busy {
			h.replyChallenge(c.reply, "")
			sendErr(creatorConn, wire.CodeBusy, "one of you is already in a game")
			return
		}
		if _, busy := h.userGame[c.creator]; busy {
			h.replyChallenge(c.reply, "")
			sendErr(creatorConn, wire.CodeBusy, "one of you is already in a game")
			return
		}

		token := h.nextID()
		h.challenges[token] = challenge{
			token: token, creator: c.creator, creatorName: creatorName,
			opponent: c.opponent, base: c.base, increment: c.increment,
		}
		h.replyChallenge(c.reply, token)
		category := string(game.CategoryForBaseSeconds(c.base))
		oppConn.Send(wire.NewChallengeIncoming(token, c.creator, creatorName, c.base, c.increment, category))
		sendIfOnline(creatorConn, wire.NewChallengeCreated(token, ""))
		return
	}

	token := h.nextID()
	h.challenges[token] = challenge{
		token: token, creator: c.creator, creatorName: creatorName,
		opponent: "", base: c.base, increment: c.increment,
	}
	h.replyChallenge(c.reply, token)
}

func (h *Hub) handleAcceptChallenge(c acceptChallengeCmd) {
	accepterConn := h.online[c.userID]
	ch, ok := h.challenges[c.token]
	if !ok {
		sendErr(accepterConn, wire.CodeUnknownChallenge, "that challenge is no longer available")
		return
	}
	if ch.opponent != "" && ch.opponent != c.userID {
		sendErr(accepterConn, wire.CodeUnknownChallenge, "that challenge is no longer available")
		return
	}
	if c.userID == ch.creator {
		sendErr(accepterConn, wire.CodeChallengeSelf, "you can't challenge yourself")
		return
	}

	delete(h.challenges, c.token)

	creatorConn, online := h.online[ch.creator]
	if !online {
		sendErr(accepterConn, wire.CodeOpponentOffline, "they're offline")
		return
	}
	if _, busy := h.userGame[ch.creator]; busy {
		sendErr(accepterConn, wire.CodeBusy, "one of you is already in a game")
		return
	}
	if _, busy := h.userGame[c.userID]; busy {
		sendErr(accepterConn, wire.CodeBusy, "one of you is already in a game")
		return
	}

	h.removeFromPools(ch.creator)
	h.removeFromPools(c.userID)
	h.startGame(poolKey{base: ch.base, increment: ch.increment}, creatorConn, accepterConn)
}

func (h *Hub) handleDeclineChallenge(c declineChallengeCmd) {
	ch, ok := h.challenges[c.token]
	if !ok || ch.opponent != c.userID {
		return
	}
	delete(h.challenges, c.token)
	sendIfOnline(h.online[ch.creator], wire.NewChallengeDeclined(c.token))
}

func (h *Hub) handleCancelChallenge(c cancelChallengeCmd) {
	ch, ok := h.challenges[c.token]
	if !ok || ch.creator != c.userID {
		return
	}
	delete(h.challenges, c.token)
	if ch.opponent != "" {
		sendIfOnline(h.online[ch.opponent], wire.NewChallengeGone(c.token))
	}
}

func (h *Hub) dropChallengesBy(uid string) {
	for token, ch := range h.challenges {
		switch {
		case ch.creator == uid:
			delete(h.challenges, token)
			if ch.opponent != "" {
				sendIfOnline(h.online[ch.opponent], wire.NewChallengeGone(token))
			}
		case ch.opponent == uid:
			delete(h.challenges, token)
			sendIfOnline(h.online[ch.creator], wire.NewChallengeDeclined(token))
		}
	}
}

func (h *Hub) replyChallenge(reply chan string, token string) {
	if reply != nil {
		reply <- token
	}
}

func sendErr(c wire.Conn, code, msg string) {
	if c != nil {
		c.Send(wire.NewError(code, msg))
	}
}

func sendIfOnline(c wire.Conn, v any) {
	if c != nil {
		c.Send(v)
	}
}
