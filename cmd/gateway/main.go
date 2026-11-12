package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	body, err := buildRequest("Hello, Claude")
	if err != nil {
		log.Fatalf("build request: %v", err)
	}
	fmt.Println(string(body))

	// Reuse err with "=" instead of declaring a new one inside the if:
	// `if _, err := ...` would shadow the outer err.
	_, err = buildRequest("   ")
	if errors.Is(err, errEmptyPrompt) {
		fmt.Println("empty prompt rejected:", err)
	} else if err != nil {
		log.Fatalf("build request: %v", err)
	} else {
		log.Fatal("expected an error for an empty prompt, got nil")
	}

	client := NewClient(os.Getenv("ANTHROPIC_API_KEY"))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Send(ctx, newMessageRequest("Hello, Claude"))
	if errors.Is(err, errNotImplemented) {
		fmt.Println("send:", err)
	} else if err != nil {
		log.Fatalf("send: %v", err)
	} else {
		fmt.Printf("response: %+v\n", resp)
	}
}
