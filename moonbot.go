package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/gobuffalo/envy"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

var (
	dtoken  string
	trigger string
	host    string
	slug    string
	token   string

	client *http.Client
)

func main() {

	envy.Load()

	tr := &http.Transport{
		//TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Timeout: 30 * time.Second, Transport: tr}

	var err error

	dtoken, err = envy.MustGet("DISCORD_TOKEN")
	checkError(err)
	trigger = envy.Get("PREFIX", "!!moons")
	host, err = envy.MustGet("SEAT_HOST")
	checkError(err)
	slug, err = envy.MustGet("SEAT_SLUG")
	checkError(err)
	token, err = envy.MustGet("SEAT_TOKEN")
	checkError(err)

	dg, err := discordgo.New("Bot " + dtoken)
	if err != nil {
		log.Printf("error creating Discord session, %v", err.Error())
		return
	}

	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		log.Printf("error opening connection, %v", err.Error())
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func checkError(err error){
	if err != nil {
		log.Fatalln(err)
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content != trigger {
		return
	}
	// https://ocic1.crypta.tech/moonbot/public/5bc10e8e-ed4f-49a7-8dbb-393bf29c180b
	url := fmt.Sprintf("%s/moonbot/public/%s", host, slug)
	spew.Dump(url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error creating request: %s", err.Error()))
		return
	}
	req.Header.Set("Authorization", token)

	res, err := client.Do(req)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error getting request: %s", err.Error()))
		return
	}
	defer res.Body.Close()

	var resp MoonBotResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error decoding response: %s", err.Error()))
		return
	}
	sort.Sort(resp)

	embed := NewEmbed().
		SetTitle("Moon Report Running").
		SetColor(0x17A2B8)
	s.ChannelMessageSendEmbed(m.ChannelID, embed.MessageEmbed)

	activeEmbed := NewEmbed().
		SetTitle("Active Extractions").
		SetColor(0x28A745)
	isActive := false

	upcomingEmbed := NewEmbed().
		SetTitle("Upcoming Extractions").
		SetColor(0xFFC107)
	isUpcoming := false

	p := message.NewPrinter(language.English)

	for _, e := range resp {
		if e.isActive() {
			isActive = true
			if e.Structure.Info.StructureID > 0 {
				//	We have the structure info, act accordingly
				activeEmbed.AddField(e.Structure.Info.Name, p.Sprintf("%d m3", e.volume()))
			} else {
				activeEmbed.AddField(e.Moon.Name, p.Sprintf("%d m3", e.volume()))
			}
		} else {
			isUpcoming = true
			if e.Structure.Info.StructureID > 0 {
				//	We have the structure info, act accordingly
				upcomingEmbed.AddField(e.Structure.Info.Name, e.ChunkArrivalTimeParsed().Format("Jan _2 15:04"))
			} else {
				upcomingEmbed.AddField(e.Moon.Name, e.ChunkArrivalTimeParsed().Format("Jan _2 15:04"))
			}
		}
	}

	if isActive {
		s.ChannelMessageSendEmbed(m.ChannelID, activeEmbed.MessageEmbed)
	}

	if isUpcoming {
		s.ChannelMessageSendEmbed(m.ChannelID, upcomingEmbed.MessageEmbed)
	}

	embedFoot := NewEmbed().
		SetTitle("Moon Report Complete!").
		SetColor(0x17A2B8).
		SetFooter("MoonBot by Crypta Electrica")
	s.ChannelMessageSendEmbed(m.ChannelID, embedFoot.MessageEmbed)



}
