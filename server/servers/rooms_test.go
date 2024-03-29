/*
 * Copyright (c) Joseph Prichard 2023
 */

package servers

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"guessthesketch/game"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// stub implementation of a brokerage that only stores a single broker
type StubBrokerage struct {
	code string
	room game.Broker
}

func (stub *StubBrokerage) Get(code string) game.Broker {
	if stub.code == code {
		return stub.room
	}
	return nil
}

func (stub *StubBrokerage) Set(code string, room game.Broker) {
	stub.code = code
	stub.room = room
}

func (stub *StubBrokerage) Codes(_ int, _ int) []string {
	return []string{stub.code}
}

// stub implementation of an authenticator where we can provide the authenticated test player
type StubAuthenticator struct {
	testPlayer game.Player
}

func (stub StubAuthenticator) GetSession(_ string) (*JwtSession, error) { return nil, nil }

func (stub StubAuthenticator) GetPlayer(_ string) game.Player { return stub.testPlayer }

// no-op implementation of handler - we don't care about testing this
type FakeHandler struct{}

func (fake FakeHandler) DoShutdown(_ []game.GameResult) {}

func (fake FakeHandler) DoCapture(_ game.Snapshot) {}

func (fake FakeHandler) OnTermination() {}

// e2e tests for the websocket server
func TestRoomsServer_CreateRoom(t *testing.T) {
	roomsServer := NewRoomsServer(&StubBrokerage{}, &StubAuthenticator{}, &FakeHandler{}, []string{})

	testSettings := game.RoomSettings{}

	buf, err := json.Marshal(testSettings)
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := strings.NewReader(string(buf))

	r := httptest.NewRequest("", "/", body)
	w := httptest.NewRecorder()

	roomsServer.CreateRoom(w, r)

	resp := w.Result()
	t.Logf("Testing create room finished with %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		t.Fail()
	}
}

func beforeTestJoinRoom(t *testing.T, initialState game.GameState) (*httptest.Server, *websocket.Conn, game.Player) {
	testRoom := game.NewRoom(initialState, true, &FakeHandler{})
	mockRooms := StubBrokerage{}
	go testRoom.Start()
	mockRooms.Set(initialState.Code(), testRoom)

	player := GuestUser()
	roomsServer := NewRoomsServer(&mockRooms, &StubAuthenticator{testPlayer: player}, &FakeHandler{}, []string{})

	s := httptest.NewServer(http.HandlerFunc(roomsServer.JoinRoom))

	u := "ws" + strings.TrimPrefix(s.URL, "http") + "?code=" + initialState.Code()
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// check the first two messages end when joining first
	_, _, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("%v", err)
	}
	_, _, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("%v", err)
	}

	return s, ws, player
}

// runs a test for a message with a particular input and expected output against the websocket connection
func runTestMessage[I any, O any](t *testing.T, ws *websocket.Conn,
	input game.InputPayload[I], expected game.OutputPayload[O]) {
	bufIn, _ := json.Marshal(input)
	if err := ws.WriteMessage(websocket.TextMessage, bufIn); err != nil {
		t.Fatalf("%v", err)
	}

	_, bufOut, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("%v", err)
	}

	var payload game.OutputPayload[O]
	err = json.Unmarshal(bufOut, &payload)
	if err != nil {
		t.Fatalf("Failed to unmarhsall output payload, didn't receieve the expected type")
	}

	if !reflect.DeepEqual(payload, expected) {
		t.Fatalf("Output %+v didn't match expected value %+v", payload, expected)
	}
}

func MockSettings(word string) game.RoomSettings {
	var settings game.RoomSettings
	game.SettingsWithDefaults(&settings)
	settings.SharedWordBank = []string{word}
	return settings
}

func TestRoomsServer_ChatMessage(t *testing.T) {
	initialState := game.NewGameState("123abc", MockSettings("Word"))
	s, ws, player := beforeTestJoinRoom(t, initialState)
	defer s.Close()
	defer ws.Close()

	input := game.InputPayload[game.TextMsg]{
		Code: game.TextCode,
		Msg:  game.TextMsg{Text: "Hello 123"},
	}
	expOutput := game.OutputPayload[game.Chat]{
		Code: game.ChatCode,
		Msg:  game.Chat{Player: player, Text: "Hello 123"},
	}

	runTestMessage(t, ws, input, expOutput)
}

func TestRoomServer_StartMessage(t *testing.T) {
	word := "Word"
	initialState := game.NewGameState("123abc", MockSettings(word))

	s, ws, _ := beforeTestJoinRoom(t, initialState)
	defer s.Close()
	defer ws.Close()

	input := game.InputPayload[struct{}]{
		Code: game.StartCode,
	}
	expOutput := game.OutputPayload[game.BeginMsg]{
		Code: game.BeginCode,
		Msg: game.BeginMsg{
			NextWord:        word,
			NextPlayerIndex: 0,
		},
	}

	runTestMessage(t, ws, input, expOutput)
}

func TestRoomsServer_DrawMessage(t *testing.T) {
	initialState := game.NewGameState("123abc", MockSettings("Word"))
	initialState.StartGame()

	s, ws, _ := beforeTestJoinRoom(t, initialState)
	defer s.Close()
	defer ws.Close()

	input := game.InputPayload[game.DrawMsg]{
		Code: game.DrawCode,
		Msg:  game.DrawMsg{X: 34, Y: 47},
	}
	expOutput := game.OutputPayload[game.DrawMsg]{
		Code: game.DrawCode,
		Msg:  game.DrawMsg{X: 34, Y: 47},
	}

	runTestMessage(t, ws, input, expOutput)
}
