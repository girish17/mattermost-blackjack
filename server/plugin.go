package main

import (
	"fmt"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v5/plugin"
)

const bjCommand = "blackjack"
const bjBot = "blackjack-bot"
var cards = map[string]int {"ace_of_hearts": 11, "2_of_hearts": 2, "3_of_hearts": 3, "4_of_hearts": 4, "5_of_hearts": 5,
	"6_of_hearts": 6, "7_of_hearts": 7, "8_of_hearts": 8, "9_of_hearts": 9, "10_of_hearts": 10,
	"jack_of_hearts": 10, "queen_of_hearts": 10, "king_of_hearts": 10,
	"ace_of_spades": 11, "2_of_spades": 2, "3_of_spades": 3, "4_of_spades": 4, "5_of_spades": 5,
	"6_of_spades": 6, "7_of_spades": 7, "8_of_spades": 8, "9_of_spades": 9, "10_of_spades": 10,
	"jack_of_spades": 10, "queen_of_spades": 10, "king_of_spades": 10,
	"ace_of_diamonds": 11, "2_of_diamonds": 2, "3_of_diamonds": 3, "4_of_diamonds": 4, "5_of_diamonds": 5,
	"6_of_diamonds": 6, "7_of_diamonds": 7, "8_of_diamonds": 8, "9_of_diamonds": 9, "10_of_diamonds": 10,
	"jack_of_diamonds": 10, "queen_of_diamonds": 10, "king_of_diamonds": 10,
	"ace_of_clubs": 11, "2_of_clubs": 2, "3_of_clubs": 3, "4_of_clubs": 4, "5_of_clubs": 5, "6_of_clubs": 6,
	"7_of_clubs": 7, "8_of_clubs": 8, "9_of_clubs": 9, "10_of_clubs": 10, "jack_of_clubs": 10, "queen_of_clubs": 10,
	"king_of_clubs": 10}

var playingCards = []string {"ace_of_hearts", "2_of_hearts", "3_of_hearts",
	"4_of_hearts", "5_of_hearts", "6_of_hearts",
	"7_of_hearts", "8_of_hearts", "9_of_hearts",
	"10_of_hearts", "jack_of_hearts", "queen_of_hearts",
	"king_of_hearts",
	"ace_of_spades", "2_of_spades", "3_of_spades",
	"4_of_spades", "5_of_spades", "6_of_spades",
	"7_of_spades", "8_of_spades", "9_of_spades",
	"10_of_spades", "jack_of_spades", "queen_of_spades",
	"king_of_spades",
	"ace_of_diamonds", "2_of_diamonds", "3_of_diamonds",
	"4_of_diamonds", "5_of_diamonds", "6_of_diamonds",
	"7_of_diamonds", "8_of_diamonds", "9_of_diamonds",
	"10_of_diamonds", "jack_of_diamonds", "queen_of_diamonds",
	"king_of_diamonds",
	"ace_of_clubs", "2_of_clubs", "3_of_clubs",
	"4_of_clubs", "5_of_clubs", "6_of_clubs",
	"7_of_clubs", "8_of_clubs", "9_of_clubs",
	"10_of_clubs", "jack_of_clubs", "queen_of_clubs",
	"king_of_clubs"}

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
func (p *Plugin) OnActivate() error {
	if p.API.GetConfig().ServiceSettings.SiteURL == nil {
		p.API.LogError("SiteURL must be set. Some features will operate incorrectly if the SiteURL is not set. See documentation for details: http://about.mattermost.com/default-site-url")
	}

	if err := p.API.RegisterCommand(createBJCommand(p.GetSiteURL())); err != nil {
		return errors.Wrapf(err, "failed to register %s command", bjCommand)
	}

	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError){
	rand.Seed(time.Now().UnixNano())
	var result = ""
	var card1 = playingCards[rand.Intn(len(playingCards))]
	var card2 = playingCards[rand.Intn(len(playingCards))]
	var score = cards[card1] + cards[card2]
	if score == 22 {
		score = 12
	}
	if score == 21 {
		result = "\n**BlackJack. Congratulations, You won!**"
	}
	var cmdResp *model.CommandResponse
	cmdResp = &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Text: "!["+ card1 + "](" + getPluginURL(p.GetSiteURL()) + "/public/jpg-cards/" + card1 + ".jpg)!["+ card2 +"](" + getPluginURL(p.GetSiteURL()) + "/public/jpg-cards/" + card2 + ".jpg)" + result,
		Username: bjBot,
		Props: nil,
	}
	return cmdResp, nil
}

func getAttachmentJSON(pluginURL string) string {
	return `{
		"attachments": [
           {
			 "text": "Hit?",
			 "integration": {
				"url": "` + pluginURL + `"/hit",
				"context": {
					"action": "hit"
                 }
              }
		   },
           {
             "text": "Stay!",
             "integration": {
             "url": "` + pluginURL + `"/stay",
             "context": {
                 "action": "stay"
             }
           }
		]
	}`
}

func (p *Plugin) GetSiteURL() string {
	siteURL := ""
	ptr := p.API.GetConfig().ServiceSettings.SiteURL
	if ptr != nil {
		siteURL = *ptr
	}
	return siteURL
}

func getPluginURL(siteURL string) string {
	return siteURL + "/plugins/com.girishm.mattermost-blackjack"
}

func createBJCommand(siteURL string) *model.Command {
	return &model.Command{
		Trigger:              bjCommand,
		Method:               "POST",
		Username:             bjBot,
		AutoComplete:         true,
		AutoCompleteDesc:     "Play Blackjack",
		AutoCompleteHint:     "",
		DisplayName:          bjBot,
		Description:          "Blackjack game for Mattermost",
		URL:                  siteURL,
	}
}
