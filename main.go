package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"golang.org/x/oauth2"
)

const (
	twitterAPIBaseURL = "https://api.x.com/2"
)

var (
	websiteURL   = "http://localhost:81"
	clientID     = os.Getenv("TWITTER_CONSUMER_KEY")
	clientSecret = os.Getenv("TWITTER_CONSUMER_SECRET")
	redirectURL  = websiteURL + "/callback"
	oauthState   = "random_state_string" // Protect against CSRF
)

var oauthConfig = &oauth2.Config{
	ClientID:     clientID,
	ClientSecret: clientSecret,
	RedirectURL:  redirectURL,
	Scopes:       []string{"tweet.read", "tweet.write", "users.read"}, // Adjust scopes as necessary
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://twitter.com/i/oauth2/authorize",
		TokenURL: twitterAPIBaseURL + "/oauth2/token",
	},
}

// Tweet represents the structure of a tweet
type Tweet struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// UserTimelineResponse represents the response from Twitter API
type UserTimelineResponse struct {
	Data []Tweet `json:"data"`
}

// getBearerToken retrieves the Twitter Bearer Token from environment variables
func getBearerToken() string {
	// Replace this with your actual Bearer Token
	return os.Getenv("TWITTER_BEARER_TOKEN")
}

// listTweets fetches the recent tweets for the specified user ID
func listTweets(userID string) ([]Tweet, error) {
	// Construct the request URL
	url := fmt.Sprintf("%s/users/%s/tweets", twitterAPIBaseURL, userID)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", getBearerToken()))

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch tweets, status code: %d", resp.StatusCode)
	}

	// Decode the response
	var timeline UserTimelineResponse
	err = json.NewDecoder(resp.Body).Decode(&timeline)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return timeline.Data, nil
}

// deleteTweet deletes the specified tweet using its tweet ID
func deleteTweet(tweetID string) error {
	// Construct the request URL
	url := fmt.Sprintf("%s/tweets/%s", twitterAPIBaseURL, tweetID)

	// Create a new HTTP request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", getBearerToken()))

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete tweet, status code: %d", resp.StatusCode)
	}

	return nil
}

type UserResponse struct {
	Data struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"data"`
}

// homePageHandler handles the display of tweets and deletion form
func homePageHandler(w http.ResponseWriter, r *http.Request) {
	// Replace with your user ID

	username := "marlonbochi" // Replace with your Twitter username
	userID, err := getUserID(username)
	if err != nil {
		log.Fatalf("Error fetching user ID: %v", err)
	}

	tweets, err := listTweets(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch tweets: %v", err), http.StatusInternalServerError)
		return
	}

	// Load and parse the template
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load template: %v", err), http.StatusInternalServerError)
		return
	}

	// Render the template with the tweets data
	err = tmpl.Execute(w, struct {
		Tweets []Tweet
	}{
		Tweets: tweets,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
	}
}

// deleteTweetHandler handles the deletion of a tweet via the form
func deleteTweetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	tweetID := r.FormValue("tweetID")
	if tweetID == "" {
		http.Error(w, "Tweet ID is required", http.StatusBadRequest)
		return
	}

	err := deleteTweet(tweetID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete tweet: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getUserID(username string) (string, error) {
	url := fmt.Sprintf("%s/users/by/username/%s", twitterAPIBaseURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set the authorization header with your Bearer Token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", getBearerToken()))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch user ID, status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	var userResp UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return userResp.Data.ID, nil
}

// Handle the OAuth login route
func handleLogin(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL(oauthState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Handle the OAuth callback from Twitter
func handleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify the state string
	if r.FormValue("state") != oauthState {
		log.Println("Invalid OAuth state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Exchange the OAuth code for an access token
	code := r.FormValue("code")
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Failed to exchange code for token: %s", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Use the access token to make requests to the Twitter API
	fmt.Fprintf(w, "Access Token: %s\n", token.AccessToken)
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Route for the homepage
	http.HandleFunc("/", homePageHandler)

	// Route for deleting a tweet
	http.HandleFunc("/delete", deleteTweetHandler)

	// Route for deleting a tweet
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)

	// Start the web server on port 81
	log.Println("Server started at " + websiteURL)
	log.Fatal(http.ListenAndServe(":81", nil))
}
