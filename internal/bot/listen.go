package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"strings"
)

func (b *Bot) AwaitMessages() {
	b.webSocketClient.Listen()
	defer b.webSocketClient.Close()
	for event := range b.webSocketClient.EventChannel {
		if event == nil {
			continue
		}
		t := event.EventType()
		if t == model.WebsocketEventPosted {
			data := event.GetData()
			var post model.Post
			err := json.Unmarshal([]byte(data["post"].(string)), &post)
			if err != nil {
				mlog.Error("failed to unmarshel post", mlog.Err(err))
				continue
			}
			mlog.Info("posted", mlog.Any("event", event), mlog.Any("post", &post))
		} else if t == model.WebsocketEventUserRemoved {
			//data
			//0 = team_id -> uej6phayiid47ymypfrpr9rb3y
			//1 = user_id -> 5cmngiuzpbdhfe6h1pywjc7pyy
			//broadcast
			//ChannelId = {string} "tnghbbimstnkxfptxj7cw6t8ec"
		} else if t == model.WebsocketEventUserAdded {
			_, ok := b.openChannels[event.GetBroadcast().ChannelId]
			if ok {
				data := event.GetData()
				userID := data["user_id"].(string)
				u, r, err := b.client.GetUser(context.Background(), userID, "")
				if err != nil {
					mlog.Error("could not load user who join channel", mlog.Err(err), mlog.Any("data", data), mlog.Any("r", r))
					continue
				}
				b.HandleJoinedUser(u)
			}
		}
	}
	if b.webSocketClient.ListenError != nil {
		mlog.Error("websocket terminated", mlog.Err(b.webSocketClient.ListenError))
	}
}

func (b *Bot) HandleJoinedUser(u *model.User) {
	targetChannelName := fmt.Sprintf("%santon-%s", b.ChannelPrefix, u.Username)
	c, response, err := b.client.GetChannelByName(context.Background(), targetChannelName, b.configChannel.TeamId, "")
	if response.StatusCode == 404 {
		newChan := model.Channel{
			TeamId:      b.configChannel.TeamId,
			Type:        model.ChannelTypePrivate,
			DisplayName: fmt.Sprintf("Anton & %s", u.FirstName),
			Name:        targetChannelName,
			CreatorId:   b.meUser.Id,
		}
		b.SetChannelPropsForTargetUser(u, &newChan)
		c, response, err = b.client.CreateChannel(context.Background(), &newChan)
		if err != nil {
			mlog.Error("failed creating new channel", mlog.Err(err), mlog.Any("channel", newChan), mlog.Any("response", response))
			return
		}
		_, _, err = b.client.AddChannelMember(context.Background(), c.Id, b.adminUser.Id)
		if err != nil {
			mlog.Error("failed add admin user to channel", mlog.Err(err), mlog.Any("channel", newChan), mlog.Any("user", u))
			return
		}
		_, _, err = b.client.AddChannelMember(context.Background(), c.Id, u.Id)
		if err != nil {
			mlog.Error("failed add target user to channel", mlog.Err(err), mlog.Any("channel", newChan), mlog.Any("user", u))
			return
		}
		b.activeChats[c.Id] = c
	} else if err != nil {
		mlog.Error("failed getting channel", mlog.Err(err), mlog.Any("channel", targetChannelName), mlog.Any("response", response))
		return
	}
	b.SetChannelPropsForTargetUser(u, c)
	b.StartConversation(c)
}

func (b *Bot) SetChannelPropsForTargetUser(u *model.User, c *model.Channel) {
	_, ok := c.Props["target"]
	if !ok {
		c.Props = map[string]any{
			"target": u.Id,
			"icon":   fmt.Sprintf("%s/api/v4/users/%s/image", b.ExternalURL, u.Id),
			"locale": u.Locale,
		}
		postsForChannel, r, err := b.client.GetPostsForChannel(context.Background(), c.Id, 0, 100, "", false, false)
		if err != nil {
			mlog.Error("Failed fetching posts for channel",
				mlog.Any("channel", c.Name),
				mlog.Any("target", u.Username),
				mlog.Any("response", r),
				mlog.Err(err),
			)
			return
		}
		for _, post := range postsForChannel.Posts {
			if post.GetProps() != nil {
				for propKey, propVal := range post.GetProps() {
					c.Props[propKey] = propVal
				}
			}
		}
	}
	b.users[u.Id] = u
}

func (b *Bot) StartConversation(c *model.Channel) {
	go b.PrintQuestion(c, b.state.Specials["init"][0], "")
}

func (b *Bot) GetTextQuestion(q *ScriptQuestion, c *model.Channel) string {
	locale := c.Props["locale"]
	var result string
	if locale == "hefemale" {
		result = q.QuestionHebFemale
	}
	if (locale == "hefemale" && result == "") || locale == "hemale" {
		result = q.QuestionHebMale
	}
	if result == "" {
		result = q.QuestionEng
	}
	target, ok := c.Props["target"]
	if ok {
		u, ok := b.users[target.(string)]
		if ok && u != nil {
			result = strings.Replace(result, "{first_name}", u.FirstName, -1)
		}
	}
	jobTitle, ok := c.Props["job_title"]
	if ok {
		result = strings.Replace(result, "{job_title}", jobTitle.(string), -1)
	}
	return result
}

func (b *Bot) GetTextAnswer(q *ScriptAnswer, c *model.Channel) string {
	locale := c.Props["locale"]
	result := ""
	if locale == "hefemale" {
		result = q.ResponseHebFemale
	}
	if (result == "" && locale == "hefemale") || locale == "hemale" {
		result = q.ResponseHebMale
	}
	if result == "" {
		result = q.ResponseEng
	}
	return result
}
