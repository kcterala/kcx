package client

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func generateRandomSubdomain() string {
    adjectives := []string{"happy", "clever", "bright", "swift", "cool", "neat", "quick", "smart"}
    nouns := []string{"cat", "dog", "bird", "fish", "lion", "bear", "wolf", "fox"}
    
    adj := adjectives[randomInt(len(adjectives))]
    noun := nouns[randomInt(len(nouns))]
    num := randomInt(1000)
    
    return fmt.Sprintf("%s-%s-%d", adj, noun, num)
}

func randomInt(max int) int {
    n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
    return int(n.Int64())
}
