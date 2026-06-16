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
