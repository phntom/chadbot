package bot

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"os"
	"os/signal"
	"strings"
)

type Bot struct {
	ChatDomain      string
	CallbackURL     string
	ChannelPrefix   string
	ExternalURL     string
	IsOnline        bool
	client          *model.Client4
	webSocketClient *model.WebSocketClient
	serverVersion   string
	username        string
	userId          string
	activeChats     map[string]*model.Channel
	configChannel   *model.Channel
	openChannels    map[string]*model.Channel
	adminUser       *model.User
	meUser          *model.User
	state           ScriptState
	users           map[string]*model.User
}

func (b *Bot) Register() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			mlog.Info("Exiting")
			b.Disconnect()
			os.Exit(0)
		}
	}()
}

func (b *Bot) Connect(authToken string) {
	wsc, err := model.NewWebSocketClient4(
		strings.Replace(b.ChatDomain, "http", "ws", 1),
		authToken,
	)
	if err != nil {
		mlog.Error("Failed connecting to a websocket", mlog.Err(err))
		os.Exit(1)
	}
	b.webSocketClient = wsc
	b.client = model.NewAPIv4Client(b.ChatDomain)
	b.MakeSureServerIsRunning()
	b.LoginAsTheBotUser(authToken)
	b.IsOnline = true
	mlog.Info("Connected")
}

func (b *Bot) Disconnect() {
	if b.webSocketClient != nil {
		b.webSocketClient.Close()
	}
	b.IsOnline = false
	mlog.Info("Disconnected")
}

func (b *Bot) MakeSureServerIsRunning() {
	props, _, err := b.client.GetOldClientConfig(context.Background(), "")
	if err != nil {
		mlog.Error("There was a problem pinging the Mattermost server.  Are you sure it's running?", mlog.Err(err))
		os.Exit(1)
	}
	b.serverVersion = props["Version"]
	mlog.Info("Server detected and is running", mlog.Any("serverVersion", b.serverVersion))
}

func (b *Bot) LoginAsTheBotUser(authToken string) {
	b.client.SetToken(authToken)
	user, _, err := b.client.GetMe(context.Background(), "")
	if err != nil {
		mlog.Error("There was a problem logging into the Mattermost server", mlog.Err(err))
		os.Exit(1)
	}
	b.username = user.Username
	b.userId = user.Id
	mlog.Info("Logged in", mlog.Any("username", b.username), mlog.Any("userId", b.userId))
}

func (b *Bot) FindBotChannel(configChannelName string, adminUsername string) {
	b.activeChats = make(map[string]*model.Channel)
	b.openChannels = make(map[string]*model.Channel)
	b.users = make(map[string]*model.User)
	teams, _, err := b.client.GetTeamsForUser(context.Background(), b.userId, "")
	if err != nil {
		mlog.Error("There was a problem fetching active teams for bot", mlog.Err(err))
		os.Exit(1)
	}
	for _, team := range teams {
		if team == nil {
			continue
		}
		channels, _, err := b.client.GetChannelsForTeamForUser(context.Background(), team.Id, b.userId, false, "")
		if err != nil {
			mlog.Error("There was a problem fetching openChannels for team", mlog.Any("teamId", team.Id), mlog.Err(err))
			os.Exit(1)
		}
		for _, channel := range channels {
			if channel == nil {
				continue
			}
			channel.AddProp("teamName", team.Name)
			if channel.Name == configChannelName {
				b.configChannel = channel
			} else if channel.Type == model.ChannelTypePrivate {
				if strings.Contains(channel.Name, "-") {
					target := strings.Split(channel.Name, "-")[1]
					u, r, err := b.client.GetUserByUsername(context.Background(), target, "")
					if err != nil {
						mlog.Error("target user not found",
							mlog.Any("target", target),
							mlog.Any("response", r),
							mlog.Err(err),
						)
					} else {
						b.SetChannelPropsForTargetUser(u, channel)
						b.activeChats[channel.Id] = channel
					}
				}
			} else if channel.IsOpen() {
				b.openChannels[channel.Id] = channel
			}
		}
	}
	if b.openChannels == nil || len(b.openChannels) == 0 {
		mlog.Fatal("Bot is not a monitoring any openChannels, invite him and try again")
		os.Exit(2)
	}
	if b.configChannel == nil {
		mlog.Fatal("Bot is not a member of chad-config config channel, invite him and try again",
			mlog.Any("configChannelName", configChannelName),
		)
		os.Exit(2)
	}
	var channelNames []string
	for _, channel := range b.openChannels {
		channelNames = append(channelNames, fmt.Sprintf("%s/%s", channel.Props["teamName"], channel.Name))
	}
	mlog.Info("Monitoring openChannels", mlog.Any("openChannels", channelNames))
	mlog.Info("Active chats loaded", mlog.Any("activeChats", len(b.activeChats)))
	mlog.Info("Config channel", mlog.Any("configChannel",
		fmt.Sprintf("%s/%s", b.configChannel.Props["teamName"], b.configChannel.Name),
	))
	b.adminUser, _, err = b.client.GetUserByUsername(context.Background(), adminUsername, "")
	if err != nil {
		mlog.Error("There was a problem finding admin user",
			mlog.Any("adminUsername", adminUsername),
			mlog.Err(err),
		)
		os.Exit(1)
	}
	b.meUser, _, err = b.client.GetMe(context.Background(), "")
	if err != nil {
		mlog.Error("There was a problem finding own user", mlog.Err(err))
		os.Exit(1)
	}
}
