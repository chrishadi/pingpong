package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Target address is not provided.\nUsage: ping <addr>.")
	}

	targetAddr := os.Args[1]

	url := fmt.Sprintf("http://%s/ping", targetAddr)
	resp, err := http.Post(url, "", nil)
	if err != nil {
		log.Fatalf("Error sending POST request to %s: %s", url, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server responded with status code: %d", resp.StatusCode)
		return
	}

	msgBytes, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		log.Fatalf("Error reading server response: %s", errRead)
		return
	}

	defer resp.Body.Close()
	log.Printf("Message from server: %s", string(msgBytes))
}
