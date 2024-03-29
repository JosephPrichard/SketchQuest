/*
 * Copyright (c) Joseph Prichard 2023
 */

package servers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
)

type ErrorResp struct {
	Code      int    `json:"code"`
	Status    int    `json:"status"`
	ErrorDesc string `json:"errorDesc"`
}

func WriteError(w http.ResponseWriter, status int, errorDesc string) {
	resp := ErrorResp{Status: status, ErrorDesc: errorDesc}
	buf, err := json.Marshal(resp)
	if err != nil {
		log.Println("Failed to serialize error for http response")
		return
	}
	w.WriteHeader(resp.Status)
	_, err = w.Write(buf)
	if err != nil {
		log.Println("Failed to write body to response")
		return
	}
}

func ReadJson[T any](r *http.Request, result *T) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.New("Failed to read data from request body")
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return errors.New("Invalid format for request body")
	}
	return nil
}

func WriteJson(w http.ResponseWriter, v any) {
	buf, err := json.Marshal(v)
	if err != nil {
		log.Println("Failed to marshal json response")
		return
	}
	_, err = w.Write(buf)
	if err != nil {
		log.Println("Failed to write body as response")
		return
	}
}

func EnableCors(w *http.ResponseWriter) {
	header := (*w).Header()
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Headers", "*")
}

func CreateUpgrade() websocket.Upgrader {
	upgrade := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	upgrade.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	return upgrade
}

func PostJson(url string, v any) ([]byte, error) {
	jsonBody, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal json %v", err)
	}
	resp, err := http.Post(url, "text/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("Error: %s", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading resp body: %s", err)
	}
	return body, nil
}
