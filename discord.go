package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func initialize_discord() (*discordgo.Session, string, *discordgo.Webhook) {
	discord_bot_token := os.Getenv("DISCORD_BOT_TOKEN")
	if discord_bot_token == "" {
		fmt.Println("Enter your discord bot token (https://discord.com/developers/applications/): ")
		fmt.Scanln(&discord_bot_token)
		write_to_dotenv("DISCORD_BOT_TOKEN", discord_bot_token)
	}

	discord, err := discordgo.New("Bot " + discord_bot_token)
	if err != nil {
		log.Fatal(err)
	}

	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentMessageContent | discordgo.IntentsGuildMembers | discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions)

	wait_for_bot(discord)

	discord_guild_id := os.Getenv("DISCORD_GUILD_ID")
	if discord_guild_id == "" {
		discord_guild_id = authorize_bot(discord)
		write_to_dotenv("DISCORD_GUILD_ID", discord_guild_id)
	}

	println("Bot authorized for guild: " + discord_guild_id)

	channel_id := get_channel_id(discord, discord_guild_id)

	webhook := get_webhook(discord, channel_id)

	return discord, channel_id, webhook
}

func get_webhook(discord *discordgo.Session, channel_id string) *discordgo.Webhook {
	webhook_id := os.Getenv("WEBHOOK_ID")
	if webhook_id == "" {
		webhook, err := discord.WebhookCreate(channel_id, "whatsapp-dsc-sync", "")
		if err != nil {
			log.Fatal(err)
		}
		write_to_dotenv("WEBHOOK_ID", webhook.ID)

		return webhook
	} else {
		webhook, err := discord.Webhook(webhook_id)
		if err != nil {
			remove_from_env("WEBHOOK_ID")
			fmt.Println("Webhook not found. Removing from .env. Try again.")
			return get_webhook(discord, channel_id)
		}

		return webhook
	}
}

func get_channel_id(discord *discordgo.Session, guild_id string) string {
	channel_name := os.Getenv("CHANNEL_NAME")
	if channel_name == "" {
		fmt.Println("Enter the name of the channel to post in: ")
		fmt.Scanln(&channel_name)

		channels, err := discord.GuildChannels(guild_id)
		if err != nil {
			log.Fatal(err)
		}

		for _, channel := range channels {
			if channel.Name == channel_name {
				write_to_dotenv("CHANNEL_NAME", channel_name)
				return channel.ID
			}
		}

		fmt.Println("Channel not found. Try again.")
		return get_channel_id(discord, guild_id)
	} else {
		channels, err := discord.GuildChannels(guild_id)
		if err != nil {
			log.Fatal(err)
		}

		for _, channel := range channels {
			if channel.Name == channel_name {
				return channel.ID
			}
		}

		fmt.Println("Channel not found. Try again.")
		return get_channel_id(discord, guild_id)
	}
}

func wait_for_bot(discord *discordgo.Session) {
	wait_chan := make(chan bool)

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot is ready")
		wait_chan <- true
	})

	err := discord.Open()
	if err != nil {
		log.Fatal(err)
	}

	println("Intents: ", discord.Identify.Intents)

	for {
		select {
		case <-time.After(10 * time.Second):
			println("Waiting for bot to be ready...")
		case <-wait_chan:
			return
		}
	}
}

func authorize_bot(discord *discordgo.Session) string {
	fmt.Printf("Invite the bot using the following link: https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot&permissions=1099511627775\n", discord.State.User.ID)

	guildChan := make(chan string)

	discord.AddHandlerOnce(func(s *discordgo.Session, r *discordgo.GuildCreate) {
		guildChan <- r.ID
	})

	for {
		select {
		case guildID := <-guildChan:
			return guildID
		case <-time.After(10 * time.Minute):
			log.Fatal("Timed out waiting for bot authorization")
		}
	}
}

func write_to_dotenv(name string, value string) {
	f, err := os.OpenFile(".env", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s=%s\n", name, value))
	if err != nil {
		log.Fatal(err)
	}
}

func remove_from_env(name string) error {
	envs, err := godotenv.Read(".env")
	if err != nil {
		return err
	}

	delete(envs, name)

	file, err := os.OpenFile(".env", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for key, value := range envs {
		_, err := writer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		if err != nil {
			return err
		}
	}
	writer.Flush()

	return nil
}
