package client

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
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
    n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
    if err != nil {
        // Fall back to timestamp-based randomness
        return int(time.Now().UnixNano() % int64(max))
    }
    return int(n.Int64())
}
