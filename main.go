package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

// HomePage renders the HTML form from the template file
func HomePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// DeleteTweet handles the tweet deletion
func DeleteTweet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	tweetIDStr := r.FormValue("tweetID")
	if tweetIDStr == "" {
		http.Error(w, "Tweet ID is required", http.StatusBadRequest)
		return
	}

	// Convert tweet ID to int64
	tweetID, err := strconv.ParseInt(tweetIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Tweet ID", http.StatusBadRequest)
		return
	}

	// Twitter API credentials (use environment variables or secure storage in production)
	consumerKey := os.Getenv("TWITTER_CONSUMER_KEY")
	consumerSecret := os.Getenv("TWITTER_CONSUMER_SECRET")
	accessToken := os.Getenv("TWITTER_ACCESS_TOKEN")
	accessSecret := os.Getenv("TWITTER_ACCESS_SECRET")

	// Set up OAuth1 configuration
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	// Create Twitter client
	client := twitter.NewClient(httpClient)

	// Attempt to delete the tweet
	_, _, err = client.Statuses.Destroy(tweetID, &twitter.StatusDestroyParams{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete tweet: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Successfully deleted tweet with ID: %d", tweetID)
}

func main() {

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}
	// Route for the homepage
	http.HandleFunc("/", HomePage)

	// Route for deleting a tweet
	http.HandleFunc("/delete", DeleteTweet)

	// Start the web server on port 8080
	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
