/*
 * Copyright (c) Joseph Prichard 2023
 */

package game

import (
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
)

type GameTurn struct {
	CurrWord        string             `json:"currWord"`        // current word to guess in session
	CurrPlayerIndex int                `json:"currPlayerIndex"` // index of player drawing on canvas
	Canvas          []Circle           `json:"canvas"`          // canvas of circles, acts as a sparse matrix which can be used to construct a bitmap
	guessers        map[uuid.UUID]bool // map storing each player ID who has guessed correctly this game
	startTimeSecs   int64              // start time in milliseconds (unix epoch)
}

type Circle struct {
	Color     uint8  `json:"color"`
	Radius    uint8  `json:"radius"`
	X         uint16 `json:"x"`
	Y         uint16 `json:"y"`
	Connected bool   `json:"connected"`
}

type Drawing struct {
	Signature string
}

func NewGameTurn() GameTurn {
	return GameTurn{
		Canvas:          make([]Circle, 0),
		CurrPlayerIndex: -1,
		startTimeSecs:   time.Now().Unix(),
		guessers:        make(map[uuid.UUID]bool),
	}
}

func (turn *GameTurn) ClearGuessers() {
	for k := range turn.guessers {
		delete(turn.guessers, k)
	}
}

func (turn *GameTurn) ClearCanvas() {
	turn.Canvas = turn.Canvas[0:0]
}

func (turn *GameTurn) ResetStartTime() {
	turn.startTimeSecs = time.Now().Unix()
}

func (turn *GameTurn) CalcResetScore() int {
	return len(turn.guessers) * 50
}

func (turn *GameTurn) ContainsCurrWord(text string) bool {
	for _, word := range strings.Split(text, " ") {
		if word == turn.CurrWord {
			return true
		}
	}
	return false
}

func (turn *GameTurn) SetGuesser(player *Player) {
	turn.guessers[player.ID] = true
}

func (turn *GameTurn) Draw(stroke Circle) {
	turn.Canvas = append(turn.Canvas, stroke)
}

func (turn *GameTurn) CaptureDrawing() (Drawing,error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(turn.Canvas)
	if err != nil {
		return Drawing{}, errors.New("Failed to capture the drawing")
	}
	return Drawing{Signature: buf.String()}, nil
}
