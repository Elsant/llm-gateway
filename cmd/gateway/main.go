package main

import (
	"errors"
	"fmt"
	"log"
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
}
