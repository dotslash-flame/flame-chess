package wire

import (
	"encoding/json"
	"testing"
)

func TestDecodeChallengeCreateDirect(t *testing.T) {
	raw := []byte(`{"type":"challenge.create_direct","opponent_id":"u-2","base":300,"increment":2}`)
	var m ChallengeCreateDirect
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Type != TypeChallengeCreateDirect || m.OpponentID != "u-2" || m.Base != 300 || m.Increment != 2 {
		t.Errorf("decoded = %+v, want u-2/300/2", m)
	}
}

func TestDecodeChallengeAcceptDeclineCancel(t *testing.T) {
	var acc ChallengeAccept
	if err := json.Unmarshal([]byte(`{"type":"challenge.accept","token":"t1"}`), &acc); err != nil || acc.Token != "t1" {
		t.Errorf("accept decode = %+v err=%v", acc, err)
	}
	var dec ChallengeDecline
	if err := json.Unmarshal([]byte(`{"type":"challenge.decline","token":"t2"}`), &dec); err != nil || dec.Token != "t2" {
		t.Errorf("decline decode = %+v err=%v", dec, err)
	}
	var can ChallengeCancel
	if err := json.Unmarshal([]byte(`{"type":"challenge.cancel","token":"t3"}`), &can); err != nil || can.Token != "t3" {
		t.Errorf("cancel decode = %+v err=%v", can, err)
	}
}

func TestEncodeServerChallengeMessages(t *testing.T) {
	inc := NewChallengeIncoming("tok", "u-1", "Alice", 600, 0, "rapid")
	b, _ := json.Marshal(inc)
	var back ChallengeIncoming
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("incoming round-trip: %v", err)
	}
	if back.Type != TypeChallengeIncoming || back.Token != "tok" || back.FromName != "Alice" || back.Category != "rapid" {
		t.Errorf("incoming round-trip = %+v", back)
	}

	list := NewOnlineList([]OnlineUser{{UID: "u-1", Name: "Alice"}})
	b, _ = json.Marshal(list)
	var listBack OnlineList
	if err := json.Unmarshal(b, &listBack); err != nil {
		t.Fatalf("online.list round-trip: %v", err)
	}
	if listBack.Type != TypeOnlineList || len(listBack.Users) != 1 || listBack.Users[0].UID != "u-1" {
		t.Errorf("online.list round-trip = %+v", listBack)
	}
}

func TestDecodeExtrasClientMessages(t *testing.T) {
	var ro RematchOffer
	if err := json.Unmarshal([]byte(`{"type":"rematch.offer","game_id":"g1"}`), &ro); err != nil || ro.GameID != "g1" {
		t.Errorf("rematch.offer decode = %+v err=%v", ro, err)
	}
	var rr RematchRespond
	if err := json.Unmarshal([]byte(`{"type":"rematch.respond","game_id":"g1","accept":true}`), &rr); err != nil || !rr.Accept {
		t.Errorf("rematch.respond decode = %+v err=%v", rr, err)
	}
	var cs ChatSend
	if err := json.Unmarshal([]byte(`{"type":"chat.send","game_id":"g1","text":"hi"}`), &cs); err != nil || cs.Text != "hi" {
		t.Errorf("chat.send decode = %+v err=%v", cs, err)
	}
	var sj SpectateJoin
	if err := json.Unmarshal([]byte(`{"type":"spectate.join","game_id":"g1"}`), &sj); err != nil || sj.GameID != "g1" {
		t.Errorf("spectate.join decode = %+v err=%v", sj, err)
	}
	var sl SpectateLeave
	if err := json.Unmarshal([]byte(`{"type":"spectate.leave"}`), &sl); err != nil || sl.Type != "spectate.leave" {
		t.Errorf("spectate.leave decode = %+v err=%v", sl, err)
	}
}

func TestEncodeExtrasServerMessages(t *testing.T) {
	cases := []struct {
		name string
		v    any
	}{
		{"opponent.disconnected", NewOpponentDisconnected("white", 30)},
		{"opponent.reconnected", NewOpponentReconnected("black")},
		{"rematch.offered", NewRematchOffered("g1", "u-1", "Alice")},
		{"rematch.declined", NewRematchDeclined("g1")},
		{"chat.msg", NewChatMsg("g1", "u-1", "Alice", "hi", 12345)},
		{"games.live", NewGamesLive([]LiveGame{{GameID: "g1", White: "Alice", Black: "Bob", Category: "blitz", Base: 300}})},
	}
	for _, tc := range cases {
		b, err := json.Marshal(tc.v)
		if err != nil {
			t.Errorf("%s marshal: %v", tc.name, err)
			continue
		}
		typ, err := DecodeType(b)
		if err != nil || typ != tc.name {
			t.Errorf("%s decoded type = %q err=%v", tc.name, typ, err)
		}
	}
}

func TestChatMsgRoundTrip(t *testing.T) {
	b, _ := json.Marshal(NewChatMsg("g1", "u-1", "Alice", "hello", 999))
	var back ChatMsg
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("chat.msg round-trip: %v", err)
	}
	if back.Text != "hello" || back.From != "u-1" || back.FromName != "Alice" || back.TS != 999 {
		t.Errorf("chat.msg round-trip = %+v", back)
	}
}

func TestSpectatorGameStartCarriesNames(t *testing.T) {
	b, _ := json.Marshal(GameStart{Type: TypeGameStart, GameID: "g1", Color: ColorSpectator, White: "Alice", Black: "Bob"})
	var back GameStart
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("spectator game.start round-trip: %v", err)
	}
	if back.Color != ColorSpectator || back.White != "Alice" || back.Black != "Bob" {
		t.Errorf("spectator game.start = %+v", back)
	}
}
