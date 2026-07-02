package logging

import (
	"encoding/json"
	"log"
	"time"
)

type Fields map[string]any

type Entry struct {
	Ts     string `json:"ts"`
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	Fields Fields `json:"fields,omitempty"`
}

func Info(msg string, fields Fields)  { write("info", msg, fields) }
func Warn(msg string, fields Fields)  { write("warn", msg, fields) }
func Error(msg string, fields Fields) { write("error", msg, fields) }

func write(level, msg string, fields Fields) {
	entry := Entry{Ts: time.Now().UTC().Format(time.RFC3339Nano), Level: level, Msg: msg, Fields: fields}
	b, err := json.Marshal(entry)
	if err != nil {
		log.Printf("level=%s msg=%q fields_marshal_error=%v", level, msg, err)
		return
	}
	log.Print(string(b))
}
