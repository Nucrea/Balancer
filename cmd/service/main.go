package main

import (
	"context"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
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

	srv := NewServer()
	srv.Run(context.Background(), uint16(port))
}
