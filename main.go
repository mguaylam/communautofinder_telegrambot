package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/mguaylam/communautofinder"
)

// Possible states in conversation with the bot
const (
	NotSearching = iota
	AskingType
	AskingMargin
	AskingPosition
	AskingDateStart
	AskingDateEnd
	Searching
	EndSearch
)

const (
	Flex = iota
	Station
)

type UserContext struct {
	chatId     int64
	state      int
	searchType int
	kmMargin   float64
	latitude   float64
	longitude  float64
	dateStart  time.Time
	dateEnd    time.Time
}

const cityId = 59 // see available cities -> https://restapifrontoffice.reservauto.net/ReservautoFrontOffice/index.html?urls.primaryName=Branch%20version%202%20(6.93.1)#/

var userContexts = make(map[int64]UserContext)
var resultChannel = make(map[int64]chan int)
var cancelSearchingMethod = make(map[int64]context.CancelFunc)

const layoutDate = "2006-01-02 15:04"

const dateExample = "2023-11-21 20:12"

var bot *tgbotapi.BotAPI

var mutex = sync.Mutex{}

func main() {

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	go http.ListenAndServe(":8444", nil)

	// Find TOKEN in .env file if exist
	godotenv.Load()
	var err error

	bot, err = tgbotapi.NewBotAPI(os.Getenv("TOKEN_COMMUNAUTOSEARCH_BOT"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on Telegram account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		message := update.Message

		mutex.Lock()

		userCtx, found := userContexts[int64(userID)]
		userCtx.chatId = update.Message.Chat.ID

		if !found {
			resultChannel[userCtx.chatId] = make(chan int, 1)
		}

		response := generateResponse(&userCtx, message)

		userContexts[int64(userID)] = userCtx

		mutex.Unlock()

		msg := tgbotapi.NewMessage(userCtx.chatId, response)
		bot.Send(msg)
	}
}

func generateResponse(userCtx *UserContext, message *tgbotapi.Message) string {

	messageText := message.Text

	if strings.ToLower(messageText) == "/aide" {
		return "Écrire:\n/chercher pour initier une nouvelle recherche.\n/recommencer pour redémarrer une recherche avec les mêmes paramètres que la recherche précédente."
	} else if strings.ToLower(messageText) == "/chercher" {

		if userCtx.state == Searching {
			log.Printf("Cancelling searching for user " + strconv.FormatInt(userCtx.chatId, 10))
			cancelSearchingMethod[userCtx.chatId]()
		}

		userCtx.state = AskingType
		log.Printf("Asking user " + strconv.FormatInt(userCtx.chatId, 10) + " vehicule type")
		return "Bonjour ! Tapez :\n- station pour rechercher une Communauto en station.\n- flex pour rechercher un véhicule Communauto Flex."
	} else if userCtx.state == AskingType {
		if strings.ToLower(messageText) == "station" {
			userCtx.searchType = Station
			userCtx.state = AskingMargin
			log.Printf("Asking user " + strconv.FormatInt(userCtx.chatId, 10) + " station radius search")
			return "Quelle est votre distance de recherche en kilomètres ?"

		} else if strings.ToLower(messageText) == "flex" {
			userCtx.searchType = Flex
			userCtx.state = AskingMargin
			log.Printf("Asking user " + strconv.FormatInt(userCtx.chatId, 10) + " flex radius search")
			return "Quelle est votre distance de recherche en kilomètres ?"
		}

	} else if userCtx.state == AskingMargin {
		margin, err := strconv.ParseFloat(messageText, 64)

		if err == nil {

			if margin > 0 {
				userCtx.kmMargin = margin
				userCtx.state = AskingPosition
				log.Printf("Asking user " + strconv.FormatInt(userCtx.chatId, 10) + " location")
				return "Veuillez partager votre position pour votre recherche."
			}
		}

		return "Veuillez entrer un rayon de recherche correct."

	} else if userCtx.state == AskingPosition {
		if message.Location != nil {
			userCtx.latitude = message.Location.Latitude
			userCtx.longitude = message.Location.Longitude

			if userCtx.searchType == Flex {
				userCtx.state = Searching
				go launchSearch(*userCtx)
				return generateMessageResearch(*userCtx)

			} else if userCtx.searchType == Station {
				userCtx.state = AskingDateStart
				log.Printf("Asking user " + strconv.FormatInt(userCtx.chatId, 10) + " start date and time for station")
				return fmt.Sprintf("Quelle est la date et l'heure de début de la location au format %s ?", dateExample)
			}
		}
	} else if userCtx.state == AskingDateStart {

		t, err := time.Parse(layoutDate, messageText)

		if err == nil {
			userCtx.dateStart = t
			userCtx.state = AskingDateEnd
			log.Printf("Asking user " + strconv.FormatInt(userCtx.chatId, 10) + " end date and time for station")
			return fmt.Sprintf("Quelle est la date et l'heure de fin de la location au format %s ?", dateExample)
		}

	} else if userCtx.state == AskingDateEnd {

		t, err := time.Parse(layoutDate, messageText)

		if err == nil {
			userCtx.dateEnd = t
			userCtx.state = Searching
			go launchSearch(*userCtx)
			return generateMessageResearch(*userCtx)
		}

	} else if strings.ToLower(messageText) == "/recommencer" {

		if userCtx.state == EndSearch {
			userCtx.state = Searching

			go launchSearch(*userCtx)
			return generateMessageResearch(*userCtx)
		} else {
			return "Veuillez initier une nouvelle recherche avant de la redémarrer."
		}

	}
	log.Printf("Invalid input from user " + strconv.FormatInt(userCtx.chatId, 10))
	return "Je n'ai pas bien compris. 😕"
}

func generateMessageResearch(userCtx UserContext) string {

	var typeSearch string

	if userCtx.searchType == Flex {
		typeSearch = "flex"
	} else if userCtx.searchType == Station {
		typeSearch = "station"
	}

	roundedKmMargin := int(userCtx.kmMargin)

	message := fmt.Sprintf("🔍 Recherche d'un véhicule %s dans un rayon de %dkm autour de la position que vous avez entrée. Vous recevrez un message lorsque l'un sera trouvé.", typeSearch, roundedKmMargin)

	if userCtx.searchType == Station {
		message += fmt.Sprintf(" de %s a %s", userCtx.dateStart.Format(layoutDate), userCtx.dateEnd.Format(layoutDate))
	}

	return message
}

func launchSearch(userCtx UserContext) {

	var currentCoordinate communautofinder.Coordinate = communautofinder.New(userCtx.latitude, userCtx.longitude)

	ctx, cancel := context.WithCancel(context.Background())

	cancelSearchingMethod[userCtx.chatId] = cancel

	if userCtx.searchType == Flex {
		go communautofinder.SearchFlexCarForGoRoutine(cityId, currentCoordinate, userCtx.kmMargin, resultChannel[userCtx.chatId], ctx, cancel)
		log.Printf("Searching a flex vehicule for user " + strconv.FormatInt(userCtx.chatId, 10))
	} else if userCtx.searchType == Station {
		go communautofinder.SearchStationCarForGoRoutine(cityId, currentCoordinate, userCtx.kmMargin, userCtx.dateStart, userCtx.dateEnd, resultChannel[userCtx.chatId], ctx, cancel)
		log.Printf("Searching a station vehicule for user " + strconv.FormatInt(userCtx.chatId, 10))
	}

	nbCarFound := <-resultChannel[userCtx.chatId]

	var msg tgbotapi.MessageConfig

	if nbCarFound != -1 {
		msg = tgbotapi.NewMessage(userCtx.chatId, fmt.Sprintf("💡 Trouvé ! %d véhicule(s) disponible(s) selon vos critères de recherche.", nbCarFound))
		log.Printf("Found vehicule(s) for user " + strconv.FormatInt(userCtx.chatId, 10))
	} else {
		msg = tgbotapi.NewMessage(userCtx.chatId, "😞 Une erreur est survenue dans vos critères de recherche. Veuillez lancer une nouvelle recherche.")
		log.Printf("Search failure for user " + strconv.FormatInt(userCtx.chatId, 10))
	}

	bot.Send(msg)

	mutex.Lock()

	newUserCtx := userContexts[userCtx.chatId]
	newUserCtx.state = EndSearch
	userContexts[newUserCtx.chatId] = newUserCtx

	mutex.Unlock()

	delete(cancelSearchingMethod, userCtx.chatId)
}
