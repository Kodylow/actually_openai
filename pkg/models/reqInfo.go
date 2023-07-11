package request

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bolt-observer/go-runes/runes"
	"github.com/kodylow/actually_openai/pkg/utils"
)

type RequestInfo struct {
	AuthHeader string
	Method  string
	Path    string
	Body    []byte
}

// GetRequestHash returns the SHA256 hash of the request's relevant fields
func (self *RequestInfo) GetReqHash() string {
	// Create a new SHA256 hash
	h := sha256.New()

	// Write the relevant parts of the request to the hash
	h.Write([]byte(self.Method))
	h.Write([]byte(self.Path))
	h.Write(self.Body)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func destructureL402AuthHeader(authHeader string) (string, string, error) {
	// Split the authHeader string by " "
	parts := strings.Split(authHeader, " ")
	// Check the parts length, it should be 2 ("L402" and "token:invoice")
	if len(parts) != 2 {
		log.Println("Invalid authorization header format destructuring L402")
		return "", "", errors.New("invalid authorization header format")
	}

	// Split the second part by ":" to get token and invoice
	tokenPreimage := strings.Split(parts[1], ":")
	log.Println("tokenPreimage:", tokenPreimage)
	// Check the tokenPreimage length, it should be 2
	if len(tokenPreimage) != 2 {
		log.Println("Invalid token:preimage format destructuring L402")
		return "", "", errors.New("invalid token:preimage format")
	}

	return tokenPreimage[0], tokenPreimage[1], nil
}

// L402IsValid checks if the given rune is valid
func checkTokenRestrictions(runeB64 string, preimage string, reqHash string) bool {
	// hash the preimage to get the paymentHash
	hash := utils.Sha256Hash(preimage)
	log.Println("Payment Hash Calculated from Preimage:", hash)
	// get the master rune from the secret

	// Read secret from environment variable
	envSecret := os.Getenv("RUNE_SECRET")

	// Convert hex encoded string secret to byte array
	var err error
	secret, err := hex.DecodeString(envSecret)
	
	master := runes.MustMakeMasterRune(secret)
	log.Println("Master Rune:", master.Rune.ToBase64())
	// decode the given rune from base64
	restrictedRune := runes.MustGetFromBase64(runeB64)
	log.Println("Restricted Rune:", restrictedRune)
	// create map with the values to evaluate
	values := map[string]any{
		"paymentHash": hash,
		"requestHash": reqHash,
	}

	// evaluate the rune to check if the given hashes match the restrictions
	err = master.Check(&restrictedRune, values)
	if err != nil {
		log.Println("Error checking rune:", err)
		return false
	}
	return true
}

func (self *RequestInfo) L402IsValid() error {
	// destructure off the token and preimage
	token, preimage, err := destructureL402AuthHeader(self.AuthHeader)
	if err != nil {
		log.Println("Error destructuring L402:", err)
		return err
	}

	// Check the token and preimage against the restrictions
	res := checkTokenRestrictions(token, preimage, self.GetReqHash())
	if !res {
		log.Println("Token doesn't match restrictions")
		return errors.New("invalid token, doesn't match restrictions")
	}

	return nil
}