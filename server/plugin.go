package main

import (
	"encoding/json"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v5/plugin"
)

const bjCommand = "blackjack"
const bjBot = "blackjack-bot"
var bot = model.Bot{
	Username:       bjBot,
	DisplayName:    bjBot,
	Description:    "Blackjack bot for Mattermost",
}

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

var playingCards []string

var dealtCards []string
var cardTxt = ""
var score = 0

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
	var post *model.Post
	body, _ := ioutil.ReadAll(r.Body)
	var jsonBody map[string]interface{}
	_ = json.Unmarshal(body, &jsonBody)
	user, _ := p.API.GetUserByUsername(bjBot)
	p.API.LogInfo("Bot UserId for posting messages: ", user.Id)

	post = &model.Post{
		UserId: user.Id,
		ChannelId: jsonBody["channel_id"].(string),
	}
	switch path := r.URL.Path; path {
	case "/hit":
		var cardIndex = rand.Intn(len(playingCards))
		dealtCards = append(dealtCards, playingCards[cardIndex])
		cardTxt += "!["+ playingCards[cardIndex] + "](" + getImgURL(p.GetSiteURL()) + playingCards[cardIndex] + ".jpg)"
		//remove card from deck
		playingCards = append(playingCards[:cardIndex], playingCards[cardIndex+1:]...)
		p.API.LogInfo("Dealt cards: ", dealtCards)
		score = calculateScore(dealtCards)

		if score > 21 {
			post.Message = cardTxt + "\n**" + strconv.Itoa(score) + ". Bust! :disappointed: Game Over. Try again - `/blackjack`**"
			score = 0
			p.API.CreatePost(post)
		} else {
			if score < 21 {
				post.Message = cardTxt
				var attachmentMap map[string]interface{}
				var result = "**Your score is " + strconv.Itoa(score) + ".**"
				json.Unmarshal([]byte(getAttachmentJSON(getPluginURL(p.GetSiteURL()), result)), &attachmentMap)
				post.SetProps(attachmentMap)
				p.API.CreatePost(post)
			} else {
				post.Message = cardTxt + "\n**Blackjack! Congratulations, you win :moneybag: Thanks for playing!**"
				p.API.CreatePost(post)
			}
		}
		break
	case "/stay":
		post.Message = "**Your final score is " + strconv.Itoa(score) + ". Thanks for playing!**"
		p.API.CreatePost(post)
		break
	}
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
func (p *Plugin) OnActivate() error {
	if p.API.GetConfig().ServiceSettings.SiteURL == nil {
		p.API.LogError("SiteURL must be set. Some features will operate incorrectly if the SiteURL is not set. See documentation for details: http://about.mattermost.com/default-site-url")
	}

	if err := p.API.RegisterCommand(createBJCommand(p.GetSiteURL())); err != nil {
		return errors.Wrapf(err, "failed to register %s command", bjCommand)
	}

	if user, _ := p.API.GetUserByUsername(bjBot); user == nil {
		if _, err := p.API.CreateBot(&bot); err != nil {
			return errors.Wrapf(err, "failed to register %s bot", bjBot)
		}
		p.setBotIcon()
	}

	return nil
}

func calculateScore(dealtCards []string) int {
	score = 0
	aces := 0
	sort.Strings(dealtCards)
	for i := 0; i < len(dealtCards); i++ {
		if strings.Contains(dealtCards[i], "ace") {
			aces++
		} else {
			score += cards[dealtCards[i]]
		}
	}
	if aces > 0 {
		//adding ace value
		for j := 0; j < aces; j++ {
			score += 11
			if score > 21 {
				score -= 10
			}
		}
	}
	return score
}

func initPlayingDeck() {
	dealtCards = nil
	cardTxt = ""
	score = 0
	playingCards = []string {"ace_of_hearts", "2_of_hearts", "3_of_hearts",
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
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError){
	initPlayingDeck()
	rand.Seed(time.Now().UnixNano())
	var result = ""
	var siteURL = p.GetSiteURL()
	var attachmentMap map[string]interface{} = nil
	var card1Index = rand.Intn(len(playingCards))
	dealtCards = append(dealtCards, playingCards[card1Index])
	//removing dealt cards from the playing deck
	playingCards = append(playingCards[:card1Index], playingCards[card1Index+1:]...)

	var card2Index = rand.Intn(len(playingCards))
	dealtCards = append(dealtCards, playingCards[card2Index])
	//removing dealt cards from the playing deck
	playingCards = append(playingCards[:card2Index], playingCards[card2Index+1:]...)

	p.API.LogInfo("Dealt cards: ", dealtCards)
	score = calculateScore(dealtCards)

	var pluginURL = getPluginURL(siteURL)
	var imgURL = getImgURL(siteURL)
	cardTxt = "!["+ dealtCards[0] + "](" + imgURL + dealtCards[0] + ".jpg)!["+ dealtCards[1] +"](" + imgURL + dealtCards[1] + ".jpg)"

	if score < 21 {
		result = "**Your score is " + strconv.Itoa(score) + ".**"
		json.Unmarshal([]byte(getAttachmentJSON(pluginURL, result)), &attachmentMap)
	}
	if score == 21 {
		result = "\n**BlackJack! Congratulations, You win :moneybag: Thanks for playing!**"
		cardTxt += result
	}

	var cmdResp *model.CommandResponse
	cmdResp = &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:  cardTxt,
		Username: bjBot,
		Props: attachmentMap,
	}
	return cmdResp, nil
}

func getAttachmentJSON(pluginURL string, result string) string {
	return `{
		"attachments": [
           {
			 "text": "` + result + `",
             "actions": [
               {
                  "name": "Hit",
			      "integration": {
				    "url": "` + pluginURL + `/hit",
				    "context": {
					  "action": "hit"
                    }
                  }
              },
              {
                 "name": "Stay",
                 "integration": {
                     "url": "` + pluginURL + `/stay",
                     "context": {
                        "action": "stay"
                     }
                 }
              }
            ]
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

func getImgURL(siteURL string) string {
	return siteURL + "/plugins/com.girishm.mattermost-blackjack/public/jpg-cards/"
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

func (p *Plugin) setBotIcon() {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		p.API.LogError("failed to get bundle path", err)
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "starter-template-icon.svg"))
	if err != nil {
		p.API.LogError("failed to read profile image", err)
	}

	user, err := p.API.GetBot(bjBot, false)
	if err != nil {
		p.API.LogError("failed to fetch bot user", err)
	}

	if appErr := p.API.SetBotIconImage(user.UserId, profileImage); appErr != nil {
		p.API.LogError("failed to set profile image", appErr)
	}
}
