package main

import (
	"github.com/phntom/chadbot/internal/bot"
	"os"
)

func main() {
	b := bot.Bot{
		ChatDomain:    os.Getenv("CHAT_DOMAIN"),
		CallbackURL:   os.Getenv("CALLBACK_URL"),
		ChannelPrefix: os.Getenv("CHANNEL_PREFIX"),
		ExternalURL:   os.Getenv("EXTERNAL_MM_URL"),
	}
	b.Register()
	b.Connect(os.Getenv("AUTH_TOKEN"))
	b.FindBotChannel(os.Getenv("CONFIG_CHANNEL"), os.Getenv("ADMIN_USERNAME"))
	b.LoadScript()
	b.StartActionsHTTPd(os.Getenv("BIND_IP"), os.Getenv("BIND_PORT"))
	b.AwaitMessages()
}
