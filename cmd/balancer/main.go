package main

import (
	"context"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

func getEnv() map[string]string {
	result := map[string]string{}
	for _, env := range os.Environ() {
		pair := strings.Split(env, "=")
		key, value := pair[0], pair[1]
		result[key] = value
	}
	return result
}

func main() {
	env := getEnv()

	port, err := strconv.Atoi(env["PORT"])
	if err != nil || port >= math.MaxUint16 || port <= 1000 {
		log.Fatal("wrong port env var")
	}

	addrs := strings.Split(env["ADDRS"], ",")
	if len(addrs) == 0 {
		log.Fatal("no addresses")
	}

	var writer io.Writer = os.Stdout
	if logfile := env["LOGFILE"]; logfile != "" {
		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		writer = zerolog.MultiLevelWriter(writer, file)
	}
	logger := zerolog.New(writer).With().Timestamp().Logger()

	srv := NewServer(logger, addrs...)
	srv.Run(context.Background(), uint16(port))
}
