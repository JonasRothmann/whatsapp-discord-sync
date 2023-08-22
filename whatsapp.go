package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func initialize_whatsapp() (*whatsmeow.Client, *types.GroupInfo) {
	dbLog := waLog.Stdout("Database", "ERROR", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:sqlite.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				// fmt.Println("QR code:", evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	group := get_whatsapp_group(client)

	return client, group
}

func get_whatsapp_group(whatsapp *whatsmeow.Client) *types.GroupInfo {
	group_id := os.Getenv("WHATSAPP_GROUP_ID")
	if group_id == "" {

		groups, err := whatsapp.GetJoinedGroups()
		if err != nil {
			fmt.Println("Error getting groups: ", err)
			return get_whatsapp_group(whatsapp)
		}

		group_names := make([]string, len(groups))
		for i, group := range groups {
			group_names[i] = fmt.Sprintf("%s (%d)", group.Name, i)
		}

		var input_i string
		fmt.Printf("Enter the name of the chat to post in.\nOptions: %s: \n", strings.Join(group_names, ", "))
		fmt.Scanln(&input_i)
		input_i_int, err := strconv.Atoi(input_i)
		if err != nil {
			fmt.Println("Write the number of the group, not the name. Try again.")
			return get_whatsapp_group(whatsapp)
		}

		for i, group := range groups {
			if i == input_i_int {
				write_to_dotenv("WHATSAPP_GROUP_ID", group.JID.String())
				return group
			}
		}

	} else {
		groups, err := whatsapp.GetJoinedGroups()
		if err != nil {
			fmt.Println("Error getting groups: ", err)
			return get_whatsapp_group(whatsapp)
		}

		for _, group := range groups {
			if group.JID.String() == group_id {
				return group
			}
		}
	}

	fmt.Println("Group not found. Try again.")
	return get_whatsapp_group(whatsapp)
}
