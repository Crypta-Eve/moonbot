package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/dustin/go-humanize"
	"github.com/gobuffalo/envy"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	dtoken  string
	trigger string
	host    string
	slug    string
	token   string

	client *http.Client

	//go:embed ore_variants.json
	oreVariantsRaw []byte

	oreVariantsMap map[int]int
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

	// Now parse in the ore variants file
	type variantsParsed []struct {
		Src  int    `json:"src"`
		Name string `json:"name"`
		Dst  int    `json:"dst"`
	}

	var variants variantsParsed
	err = json.Unmarshal(oreVariantsRaw, &variants)
	checkError(err)
	oreVariantsMap = make(map[int]int)
	for _, variant := range variants {
		oreVariantsMap[variant.Src] = variant.Dst
	}

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

func checkError(err error) {
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

	embed := NewEmbed().
		SetTitle("Moon Report Running").
		SetColor(0x17A2B8)
	smsg, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed.MessageEmbed)
	defer s.ChannelMessageDelete(m.ChannelID, smsg.ID)

	embed2 := NewEmbed().
		SetTitle("Updating API Data, Please Wait.").
		SetColor(0x17A2B8)
	smsg2, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed2.MessageEmbed)

	// Send the update request

	url := fmt.Sprintf("%s/moonbot/public/update/%s", host, slug)
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
	res.Body.Close()

	s.ChannelMessageDelete(m.ChannelID, smsg2.ID)

	// Request the data.

	url = fmt.Sprintf("%s/moonbot/public/%s", host, slug)
	spew.Dump(url)
	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error creating request: %s", err.Error()))
		return
	}
	req.Header.Set("Authorization", token)

	res, err = client.Do(req)
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

	upcomingEmbed := NewEmbed().
		SetTitle("Upcoming Extractions").
		SetColor(0xFFC107)
	isUpcoming := false

	p := message.NewPrinter(language.English)

	for _, e := range resp {
		if e.isActive() {

			name := ""
			if e.Structure.Info.StructureID > 0 {
				name = e.Structure.Info.Name
			} else {
				name = e.Moon.Name
			}

			moonEmbed := NewEmbed().
				SetTitle(name).
				SetColor(0x28A745)

			// First check if we have mining data available
			minedTotal := make(map[int]int)
			if e.Observer.ObserverID > 0 && len(e.Observer.Entries) > 0 {
				for _, mined := range e.Observer.Entries {
					entryTime, err := time.Parse(timeFormat, mined.LastUpdated)
					if err != nil {
						continue
					}
					d := 24 * time.Hour
					// make sure that this entry refers to this extraction
					if entryTime.Truncate(d).After(e.ChunkArrivalTimeParsed().Truncate(d).Add(-1 * time.Second)) {
						minedTotal[oreVariantsMap[mined.TypeID]] += mined.Quantity
					}
				}
			}

			log.Printf("%#v", minedTotal)

			for _, ore := range e.Moon.MoonReport.Content {
				pe, err := strconv.ParseFloat(ore.Pivot.Rate, 32)
				if err != nil {
					continue
				}
				pulledVol := float64(e.volume()) * pe
				remainVol := pulledVol - float64(minedTotal[ore.TypeID]*ore.Volume)

				volString := p.Sprintf("%s left (%.1f%%)", humanize.SIWithDigits(remainVol, 2, "m3"), remainVol/pulledVol*100)

				moonEmbed.AddField(ore.TypeName, volString)
			}

			_, err = s.ChannelMessageSendEmbed(m.ChannelID, moonEmbed.MessageEmbed)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to send active moon embed: %w", err))
			}

		} else {
			isUpcoming = true

			secondLine := e.ChunkArrivalTimeParsed().Format("Jan _2 15:04") +
				" -- " + humanize.SIWithDigits(float64(e.volume()), 2, "m3") + "\n"

			// Now sort the moon contents, luckily with groupIDs the higher is better for moons.
			contents := e.Moon.MoonReport.Content
			sort.Slice(contents, func(i, j int) bool {
				return contents[i].GroupID > contents[j].GroupID
			})

			for _, ore := range contents {
				secondLine += " " + ore.TypeName + ","
			}

			secondLine = strings.TrimSuffix(secondLine, ",")

			if e.Structure.Info.StructureID > 0 {
				//	We have the structure info, act accordingly
				upcomingEmbed.AddField(e.Structure.Info.Name, secondLine)
			} else {
				upcomingEmbed.AddField(e.Moon.Name, secondLine)
			}
		}
	}

	if isUpcoming {
		s.ChannelMessageSendEmbed(m.ChannelID, upcomingEmbed.MessageEmbed)
	}

	embedFoot := NewEmbed().
		SetTitle("Moon Report Complete!").
		SetColor(0x17A2B8).
		SetFooter("MoonBot by Crypta Electrica")
	_, err = s.ChannelMessageSendEmbed(m.ChannelID, embedFoot.MessageEmbed)

}
