package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var s *discordgo.Session

func init() {
	godotenv.Load()
	log.Print("Getting bot token from .env file")
	var BotToken = os.Getenv("TOKEN")
	var err error
	s, err = discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v | Check the .env", err)
	}
	JSONCheck()
	fetchWarningToJson()
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "help",
		},
		{
			Name:        "botstatus",
			Description: "botstatus",
		},
		{
			Name:        "addchannel",
			Description: "adds channels to the database",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "torchannel",
					Description: "tornado channel",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "svrstormchannel",
					Description: "severe storm channel",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "winterchannel",
					Description: "winter channel",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "swschannel",
					Description: "Special Weather Statement channel",
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"botstatus": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "Bot Uptime",
							Description: fmt.Sprintf("Last Start: \nUptime:"),
							Color:       0x57F287,
						},
					},
				},
			})
		},
		"addchannel": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			torchannel := i.ApplicationCommandData().Options[0].StringValue()
			svrstormchannel := i.ApplicationCommandData().Options[1].StringValue()
			winterchannel := i.ApplicationCommandData().Options[2].StringValue()
			swschannel := i.ApplicationCommandData().Options[2].StringValue()
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title: "Channel Added",
							Fields: []*discordgo.MessageEmbedField{
								{
									Value:  "Tornado Channel:" + torchannel + "\n" + "Severe Thunderstorm Channel: " + svrstormchannel + "\n" + "Winter Channel: " + winterchannel + "\n" + "Special Weather Statement: " + swschannel,
									Inline: false,
								},
							},
							Color: 0x57F287,
						},
					},
				},
			})
			addChannel(torchannel, svrstormchannel, winterchannel, swschannel)
		},
		"help": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title: "List of Commands",
							Color: 0xFF0090,
							Fields: []*discordgo.MessageEmbedField{
								{
									Name:   "/botstatus",
									Value:  "Shows bot uptime",
									Inline: false,
								},
							},
						},
					},
				},
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	GuildID := os.Getenv("GuildID")

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		_ = s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{
				{
					Name: UTCTime(),
					Type: discordgo.ActivityTypeWatching,
				},
			},
			Status: "online",
		})

		for range ticker.C {
			err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
				Activities: []*discordgo.Activity{
					{
						Name: UTCTime(),
						Type: discordgo.ActivityTypeWatching,
					},
				},
				Status: "online",
			})
			if err != nil {
				log.Println("Failed to update status:", err)
			}
		}
	}()

	existing, err := s.ApplicationCommands(s.State.User.ID, GuildID)
	if err != nil {
		log.Fatalf("Failed to list existing commands: %v", err)
	}

	for _, cmd := range existing {
		err := s.ApplicationCommandDelete(s.State.User.ID, GuildID, cmd.ID)
		if err != nil {
			log.Printf("Failed to delete old command '%v': %v", cmd.Name, err)
		}
	}

	log.Println("Adding commands...")
	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
	}

	log.Println("Refreshing commands...")
	_, err = s.ApplicationCommandBulkOverwrite(s.State.User.ID, GuildID, commands)
	if err != nil {
		log.Fatalf("Cannot refresh commands: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop
}
