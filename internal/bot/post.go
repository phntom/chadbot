package bot

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"strings"
	"time"
)

func (b *Bot) CreatePost(c *model.Channel, line string, props model.StringInterface) {
	post := model.Post{
		ChannelId: c.Id,
		Message:   line,
	}
	if props != nil {
		post.SetProps(props)
	}
	_, r, err := b.client.CreatePost(context.Background(), &post)
	if err != nil {
		mlog.Error("failed to post",
			mlog.Any("line", line),
			mlog.Any("channel", c.Name),
			mlog.Any("response", r),
			mlog.Err(err),
		)
	}
}

func (b *Bot) UserTyping(c *model.Channel) {
	b.webSocketClient.UserTyping(c.Id, "")
	time.Sleep(1 * time.Second)
}

func (b *Bot) PrintQuestion(c *model.Channel, q *ScriptQuestion, disablePostID string) {
	t := b.GetTextQuestion(q, c)
	mlog.Info("Printing question",
		mlog.Any("q", q),
		mlog.Any("text", t),
		mlog.Any("c", c.Name),
	)
	lines := strings.Split(t, "\n")
	for i := 0; i < len(lines)-1; i++ {
		b.UserTyping(c)
		b.CreatePost(c, lines[i], nil)
	}
	b.UserTyping(c)
	var actions []*model.PostAction
	for _, aID := range q.AnswerOrder {
		answer := q.Answer[aID]
		actions = append(actions, &model.PostAction{
			Id:   aID,
			Name: b.GetTextAnswer(answer, c),
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("%s%s", b.CallbackURL, ActionURLSuffix),
				Context: map[string]any{
					"q": q.ID,
					"a": aID,
					"d": disablePostID,
				},
			},
		})
	}
	if len(actions) == 0 {
		b.CreatePost(c, lines[len(lines)-1], nil)
	} else {
		name, icon := b.GetNameIconForChannel(c)
		props := model.StringInterface{
			"attachments": []model.SlackAttachment{
				{
					Actions:    actions,
					Footer:     name,
					FooterIcon: icon,
					Title:      lines[len(lines)-1],
				},
			},
		}
		b.CreatePost(c, "", props)
	}
}

func (b *Bot) GetNameIconForChannel(c *model.Channel) (string, string) {
	if c.Props == nil {
		return "", ""
	}
	icon, ok := c.Props["icon"]
	if !ok {
		icon = ""
	}
	target, ok := c.Props["target"]
	if !ok {
		return "", ""
	}
	u, ok := b.users[target.(string)]
	if !ok {
		return "", ""
	}
	name := fmt.Sprintf("%s %s", u.FirstName, u.LastName)
	return name, icon.(string)
}
