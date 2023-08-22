package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		println("No .env file found")
	}

	discord, channel_id, webhook := initialize_discord()

	whatsapp, group := initialize_whatsapp()

	discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.ChannelID != channel_id {
			return
		}

		if m.Author.Bot {
			return
		}

		on_discord_message_create(s, m, whatsapp, group)
	})

	fmt.Println("Bot is now listening to discord messages. Press CTRL-C to exit.")

	whatsapp.AddEventHandler(func(evt interface{}) {
		eventHandler(evt, discord, webhook, whatsapp)
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")

	whatsapp.Disconnect()
}

func eventHandler(evt interface{}, discord *discordgo.Session, webhook *discordgo.Webhook, whatsapp *whatsmeow.Client) {
	switch v := evt.(type) {
	case *events.Message:
		if strings.HasPrefix(*v.Message.Conversation, "(Discord)") {
			return
		}

		var profile_picture_url *string
		profile_picture, err := whatsapp.GetProfilePictureInfo(v.Info.Sender, &whatsmeow.GetProfilePictureParams{Preview: true, ExistingID: "", IsCommunity: false})
		if err != nil {
			log.Println(err)
		} else {
			profile_picture_url = &profile_picture.URL
		}

		send_message_with_webhook(*webhook, discord, v.Info.PushName, profile_picture_url, *v.Message.Conversation)
	}
}

func on_discord_message_create(s *discordgo.Session, m *discordgo.MessageCreate, whatsapp *whatsmeow.Client, group *types.GroupInfo) {
	name := m.Member.Nick
	if name == "" {
		if m.Author.GlobalName == "" {
			name = m.Author.Username
		} else {
			name = m.Author.GlobalName
		}
	}

	whatsapp.SendMessage(context.Background(), group.JID, &waProto.Message{
		Conversation: proto.String(fmt.Sprintf("(Discord) %s: %s", name, m.Content)),
	})

	fmt.Printf("Sent message \"%s\" to whatsapp\n", m.Content)
}

func send_message_with_webhook(webhook discordgo.Webhook, discord *discordgo.Session, username string, avatar_url *string, content string) (*discordgo.Message, error) {
	webhook_data := &discordgo.WebhookParams{
		Content:  content,
		Username: username,
	}

	if avatar_url != nil {
		webhook_data.AvatarURL = *avatar_url
	}

	return discord.WebhookExecute(webhook.ID, webhook.Token, true, webhook_data)
}
