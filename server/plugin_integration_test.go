package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/stretchr/testify/assert"
)

type mockAPI struct {
	plugin.API
	createPostCalled bool
	createdPost      *model.Post
}

func (m *mockAPI) CreatePost(post *model.Post) (*model.Post, *model.AppError) {
	m.createPostCalled = true
	m.createdPost = post
	return post, nil
}

func (m *mockAPI) GetUserByUsername(name string) (*model.User, *model.AppError) {
	return &model.User{Id: "test-user-id"}, nil
}

func (m *mockAPI) GetBundlePath() (string, error) {
	return "", nil
}

func (m *mockAPI) GetConfig() *model.Config {
	siteURL := "http://localhost:8065"
	return &model.Config{
		ServiceSettings: model.ServiceSettings{
			SiteURL: &siteURL,
		},
	}
}

func (m *mockAPI) LogInfo(msg string, args ...interface{}) {}

type testPlugin struct {
	*Plugin
	api *mockAPI
}

func setupTestPlugin() *testPlugin {
	api := &mockAPI{}
	p := &Plugin{}
	p.SetAPI(api)
	return &testPlugin{
		Plugin: p,
		api:    api,
	}
}

func TestServeHTTP_DefaultRoute(t *testing.T) {
	p := setupTestPlugin()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Welcome to Blackjack!")
}

func TestServeHTTP_HitEndpoint_NoGame(t *testing.T) {
	p := setupTestPlugin()
	initPlayingDeck()

	req := httptest.NewRequest(http.MethodPost, "/hit", strings.NewReader("channel_id=test-channel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, p.api.createPostCalled)
}

func TestServeHTTP_HitEndpoint_WithJSONBody(t *testing.T) {
	p := setupTestPlugin()
	initPlayingDeck()

	body := map[string]interface{}{
		"channel_id": "test-channel-id",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/hit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, p.api.createPostCalled)
}

func TestServeHTTP_StayEndpoint(t *testing.T) {
	p := setupTestPlugin()
	initPlayingDeck()

	req := httptest.NewRequest(http.MethodPost, "/stay", strings.NewReader("channel_id=test-channel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, p.api.createPostCalled)
}

func TestServeHTTP_CardsEndpoint_InvalidCard(t *testing.T) {
	p := setupTestPlugin()

	req := httptest.NewRequest(http.MethodGet, "/cards?card=invalid_card_name", nil)
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServeHTTP_CombinedEndpoint_NoCards(t *testing.T) {
	p := setupTestPlugin()

	req := httptest.NewRequest(http.MethodGet, "/combined", nil)
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServeHTTP_CombinedEndpoint_WithCards(t *testing.T) {
	p := setupTestPlugin()

	req := httptest.NewRequest(http.MethodGet, "/combined?cards=ace_of_hearts,king_of_hearts", nil)
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
}

func TestServeHTTP_HitEndpoint_AfterGameOver(t *testing.T) {
	p := setupTestPlugin()
	gameOver = true

	req := httptest.NewRequest(http.MethodPost, "/hit", strings.NewReader("channel_id=test-channel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, p.api.createPostCalled)
	assert.Contains(t, p.api.createdPost.Message, "To start a new game")
}

func TestServeHTTP_StayEndpoint_AfterGameOver(t *testing.T) {
	p := setupTestPlugin()
	gameOver = true

	req := httptest.NewRequest(http.MethodPost, "/stay", strings.NewReader("channel_id=test-channel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	p.ServeHTTP(&plugin.Context{}, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, p.api.createPostCalled)
	assert.Contains(t, p.api.createdPost.Message, "To start a new game")
}

func TestCalculateHandScore(t *testing.T) {
	tests := []struct {
		name     string
		cards    []string
		expected int
	}{
		{"empty hand", []string{}, 0},
		{"two aces", []string{"ace_of_hearts", "ace_of_spades"}, 12},
		{"blackjack", []string{"ace_of_hearts", "king_of_hearts"}, 21},
		{"face cards", []string{"queen_of_hearts", "king_of_diamonds"}, 20},
		{"multiple aces with bust", []string{"ace_of_hearts", "ace_of_spades", "queen_of_hearts", "king_of_diamonds"}, 22},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateHandScore(tt.cards)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineWinner(t *testing.T) {
	tests := []struct {
		name            string
		playerCards     []string
		dealerCards     []string
		expectedContain string
	}{
		{"player busts", []string{"king_of_hearts", "queen_of_spades", "5_of_hearts"}, []string{"5_of_diamonds", "6_of_clubs"}, "Bust!"},
		{"dealer busts", []string{"king_of_hearts", "9_of_spades"}, []string{"10_of_diamonds", "8_of_clubs", "5_of_hearts"}, "Dealer bust!"},
		{"player wins", []string{"king_of_hearts", "9_of_spades"}, []string{"queen_of_diamonds", "7_of_clubs"}, "You win"},
		{"dealer wins", []string{"queen_of_hearts", "8_of_spades"}, []string{"king_of_diamonds", "9_of_clubs"}, "Dealer wins"},
		{"push", []string{"king_of_hearts", "9_of_spades"}, []string{"king_of_diamonds", "9_of_clubs"}, "Push"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dealtCards = tt.playerCards
			dealerCards = tt.dealerCards
			score = calculateHandScore(dealtCards)
			dealerScore = calculateHandScore(dealerCards)

			result := determineWinner()
			assert.Contains(t, result, tt.expectedContain)
		})
	}
}

func TestInitPlayingDeck(t *testing.T) {
	initPlayingDeck()

	assert.Equal(t, 52, len(playingCards))
	assert.False(t, gameOver)
	assert.Empty(t, dealtCards)
	assert.Empty(t, dealerCards)
	assert.Equal(t, 0, score)
	assert.Equal(t, 0, dealerScore)
}

func TestGetPluginURL(t *testing.T) {
	siteURL := "http://localhost:8065"
	result := getPluginURL(siteURL)
	expected := "http://localhost:8065/plugins/com.girishm.mattermost-blackjack"
	assert.Equal(t, expected, result)
}

func TestGetPluginURL_EmptySiteURL(t *testing.T) {
	result := getPluginURL("")
	expected := "/plugins/com.girishm.mattermost-blackjack"
	assert.Equal(t, expected, result)
}
