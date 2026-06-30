package wire

//contains all the poasisble messages ws connection will send/receive
import "encoding/json"

const (
	TypeQueueJoin    = "queue.join"
	TypeQueueLeave   = "queue.leave"
	TypeMove         = "move"
	TypeResign       = "resign"
	TypeDrawOffer    = "draw.offer"
	TypeDrawRespond  = "draw.respond"
	TypePing         = "ping"
	TypeOnlineCount  = "online.count"
	TypeQueueWaiting = "queue.waiting"
	TypeGameStart    = "game.start"
	TypeGameState    = "game.state"
	TypeGameOver     = "game.over"
	TypeDrawOffered  = "draw.offered"
	TypePong         = "pong"
	TypeError        = "error"

	TypeChallengeCreateDirect = "challenge.create_direct"
	TypeChallengeAccept       = "challenge.accept"
	TypeChallengeDecline      = "challenge.decline"
	TypeChallengeCancel       = "challenge.cancel"
	TypeChallengeIncoming     = "challenge.incoming"
	TypeChallengeCreated      = "challenge.created"
	TypeChallengeDeclined     = "challenge.declined"
	TypeChallengeGone         = "challenge.gone"
	TypeOnlineList            = "online.list"

	TypeRematchOffer    = "rematch.offer"
	TypeRematchRespond  = "rematch.respond"
	TypeChatSend        = "chat.send"
	TypeSpectateJoin    = "spectate.join"
	TypeSpectateLeave   = "spectate.leave"
	TypeOpponentGone    = "opponent.disconnected"
	TypeOpponentBack    = "opponent.reconnected"
	TypeRematchOffered  = "rematch.offered"
	TypeRematchDeclined = "rematch.declined"
	TypeChatMsg         = "chat.msg"
	TypeGamesLive       = "games.live"

	ColorSpectator = "spectator"
)

const (
	CodeBadMessage    = "bad_message"
	CodeIllegalMove   = "illegal_move"
	CodeNotYourTurn   = "not_your_turn"
	CodeNotInGame     = "not_in_game"
	CodeGameNotActive = "game_not_active"
	CodeUnknownGame   = "unknown_game"

	CodeUnknownChallenge = "unknown_challenge"
	CodeChallengeSelf    = "challenge_self"
	CodeBusy             = "busy"
	CodeOpponentOffline  = "opponent_offline"

	CodeRematchUnavailable = "rematch_unavailable"
)

type Conn interface {
	UserID() string
	DisplayName() string
	Send(v any)
	Close()
}

type Envelope struct {
	Type string `json:"type"`
}

func DecodeType(data []byte) (string, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return "", err
	}
	return e.Type, nil
}

type QueueJoin struct {
	Type      string `json:"type"`
	Category  string `json:"category"`
	Base      int    `json:"base"`
	Increment int    `json:"increment"`
}

type QueueLeave struct {
	Type string `json:"type"`
}

type Move struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
	UCI    string `json:"uci"`
}

type Resign struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
}

type DrawOffer struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
}

type DrawRespond struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
	Accept bool   `json:"accept"`
}

type OnlineCount struct {
	Type string `json:"type"`
	N    int    `json:"n"`
}

func NewOnlineCount(n int) OnlineCount {
	return OnlineCount{Type: TypeOnlineCount, N: n}
}

type QueueWaiting struct {
	Type string `json:"type"`
}

func NewQueueWaiting() QueueWaiting { return QueueWaiting{Type: TypeQueueWaiting} }

type Clocks struct {
	WhiteMs int64 `json:"white_ms"`
	BlackMs int64 `json:"black_ms"`
}

type GameStart struct {
	Type     string `json:"type"`
	GameID   string `json:"game_id"`
	Color    string `json:"color"`
	Opponent string `json:"opponent"`
	Clocks   Clocks `json:"clocks"`
	FEN      string `json:"fen"`
	White    string `json:"white,omitempty"`
	Black    string `json:"black,omitempty"`
}

type GameState struct {
	Type     string `json:"type"`
	GameID   string `json:"game_id"`
	FEN      string `json:"fen"`
	LastMove string `json:"last_move"`
	WhiteMs  int64  `json:"white_ms"`
	BlackMs  int64  `json:"black_ms"`
	Turn     string `json:"turn"`
}

type GameOver struct {
	Type    string       `json:"type"`
	GameID  string       `json:"game_id"`
	Result  string       `json:"result"`
	Reason  string       `json:"reason"`
	Ratings *GameRatings `json:"ratings,omitempty"`
}

type GameRatings struct {
	White RatingChange `json:"white"`
	Black RatingChange `json:"black"`
}

type RatingChange struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Delta  int `json:"delta"`
}

type DrawOffered struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
	From   string `json:"from"`
}

type Pong struct {
	Type string `json:"type"`
}

func NewPong() Pong { return Pong{Type: TypePong} }

type SessionReplaced struct {
	Type string `json:"type"`
}

func NewSessionReplaced() SessionReplaced { return SessionReplaced{Type: "session.replaced"} }

type Error struct {
	Type string `json:"type"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func NewError(code, msg string) Error {
	return Error{Type: TypeError, Code: code, Msg: msg}
}

// Client → server challenge messages.

type ChallengeCreateDirect struct {
	Type       string `json:"type"`
	OpponentID string `json:"opponent_id"`
	Base       int    `json:"base"`
	Increment  int    `json:"increment"`
}

type ChallengeAccept struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type ChallengeDecline struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type ChallengeCancel struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Server → client challenge messages.

type ChallengeIncoming struct {
	Type      string `json:"type"`
	Token     string `json:"token"`
	From      string `json:"from"`
	FromName  string `json:"from_name"`
	Base      int    `json:"base"`
	Increment int    `json:"increment"`
	Category  string `json:"category"`
}

func NewChallengeIncoming(token, from, fromName string, base, increment int, category string) ChallengeIncoming {
	return ChallengeIncoming{
		Type: TypeChallengeIncoming, Token: token, From: from, FromName: fromName,
		Base: base, Increment: increment, Category: category,
	}
}

type ChallengeCreated struct {
	Type  string `json:"type"`
	Token string `json:"token"`
	URL   string `json:"url"`
}

func NewChallengeCreated(token, url string) ChallengeCreated {
	return ChallengeCreated{Type: TypeChallengeCreated, Token: token, URL: url}
}

type ChallengeDeclined struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func NewChallengeDeclined(token string) ChallengeDeclined {
	return ChallengeDeclined{Type: TypeChallengeDeclined, Token: token}
}

type ChallengeGone struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func NewChallengeGone(token string) ChallengeGone {
	return ChallengeGone{Type: TypeChallengeGone, Token: token}
}

type OnlineUser struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type OnlineList struct {
	Type  string       `json:"type"`
	Users []OnlineUser `json:"users"`
}

func NewOnlineList(users []OnlineUser) OnlineList {
	return OnlineList{Type: TypeOnlineList, Users: users}
}

type RematchOffer struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
}

type RematchRespond struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
	Accept bool   `json:"accept"`
}

type ChatSend struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
	Text   string `json:"text"`
}

type SpectateJoin struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
}

type SpectateLeave struct {
	Type string `json:"type"`
}

type OpponentDisconnected struct {
	Type         string `json:"type"`
	Color        string `json:"color"`
	GraceSeconds int    `json:"grace_seconds"`
}

func NewOpponentDisconnected(color string, graceSeconds int) OpponentDisconnected {
	return OpponentDisconnected{Type: TypeOpponentGone, Color: color, GraceSeconds: graceSeconds}
}

type OpponentReconnected struct {
	Type  string `json:"type"`
	Color string `json:"color"`
}

func NewOpponentReconnected(color string) OpponentReconnected {
	return OpponentReconnected{Type: TypeOpponentBack, Color: color}
}

type RematchOffered struct {
	Type     string `json:"type"`
	GameID   string `json:"game_id"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
}

func NewRematchOffered(gameID, from, fromName string) RematchOffered {
	return RematchOffered{Type: TypeRematchOffered, GameID: gameID, From: from, FromName: fromName}
}

type RematchDeclined struct {
	Type   string `json:"type"`
	GameID string `json:"game_id"`
}

func NewRematchDeclined(gameID string) RematchDeclined {
	return RematchDeclined{Type: TypeRematchDeclined, GameID: gameID}
}

type ChatMsg struct {
	Type     string `json:"type"`
	GameID   string `json:"game_id"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	Text     string `json:"text"`
	TS       int64  `json:"ts"`
}

func NewChatMsg(gameID, from, fromName, text string, ts int64) ChatMsg {
	return ChatMsg{Type: TypeChatMsg, GameID: gameID, From: from, FromName: fromName, Text: text, TS: ts}
}

type LiveGame struct {
	GameID    string `json:"game_id"`
	White     string `json:"white"`
	Black     string `json:"black"`
	Category  string `json:"category"`
	Base      int    `json:"base"`
	Increment int    `json:"increment"`
}

type GamesLive struct {
	Type  string     `json:"type"`
	Games []LiveGame `json:"games"`
}

func NewGamesLive(games []LiveGame) GamesLive {
	return GamesLive{Type: TypeGamesLive, Games: games}
}
