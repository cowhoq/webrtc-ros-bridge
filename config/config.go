package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
)

type TopicConfig struct {
	NameIn  string `json:"name_in"`
	NameOut string `json:"name_out"`
	Type    string `json:"type"` // only "sensor_msgs/msg/Image" is supported
}

type Config struct {
	Mode   string        `json:"mode"` // either "sender" or "receiver"
	Addr   string        `json:"addr"` // http service address
	Topics []TopicConfig `json:"topics"`
}

func LoadCfg() *Config {
	if _, err := os.Stat("wrb.json"); errors.Is(err, os.ErrNotExist) {
		slog.Info("wrb.json not found, using default config")
		return &Config{
			Mode: "receiver",
			Addr: "localhost:8080",
			Topics: []TopicConfig{
				{
					NameIn:  "image_raw",
					NameOut: "image",
					Type:    "sensor_msgs/msg/Image",
				},
			},
		}
	}
	f, err := os.Open("wrb.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		panic(err)
	}
	bf := make([]byte, stat.Size())
	_, err = bufio.NewReader(f).Read(bf)
	if err != nil && err != io.EOF {
		panic(err)
	}
	c := &Config{}
	err = json.Unmarshal(bf, c)
	if err != nil {
		panic(err)
	}
	// TODO: validate config
	return c
}
