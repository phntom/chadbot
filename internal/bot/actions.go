package bot

import (
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"io"
	"net/http"
	"strings"
)

const ActionURLSuffix = "/action"
const LivelinessSuffix = "/_healthz"

func (b *Bot) StartActionsHTTPd(bindIP string, bindPort string) {
	http.HandleFunc(ActionURLSuffix, b.ActionHandler)
	http.HandleFunc(LivelinessSuffix, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello")
	})
	go http.ListenAndServe(fmt.Sprintf("%s:%s", bindIP, bindPort), nil)
}

func (b *Bot) ActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		mlog.Error("Failed parsing body", mlog.Err(err))
		http.Error(w, "Failed parsing body", 500)
		return
	}
	var action model.PostActionIntegrationRequest
	err = json.Unmarshal(body, &action)
	if err != nil {
		mlog.Error("Failed unmarshaling body", mlog.Err(err))
		http.Error(w, "Failed unmarshaling body", 500)
		return
	}
	qID, ok := action.Context["q"]
	if !ok {
		mlog.Error("Question id not in request",
			mlog.Any("action", action),
		)
		http.Error(w, "Question id not in request", 500)
		return
	}
	q, ok := b.state.Questions[qID.(string)]
	if !ok {
		mlog.Error("question not found",
			mlog.Any("action", action),
		)
		http.Error(w, "Question not found", 500)
		return
	}
	aID, ok := action.Context["a"]
	if !ok {
		mlog.Error("Answer id not in request",
			mlog.Any("action", action),
		)
		http.Error(w, "Answer id not in request", 500)
		return
	}
	a, ok := q.Answer[aID.(string)]
	if !ok {
		mlog.Error("answer not found",
			mlog.Any("action", action),
		)
		http.Error(w, "Answer not found", 500)
		return
	}
	c, ok := b.activeChats[action.ChannelId]
	if !ok {
		mlog.Error("channel not found",
			mlog.Any("action", action),
		)
		http.Error(w, "Channel not found", 500)
		return
	}
	lines := strings.Split(b.GetTextQuestion(q, c), "\n")
	lastLine := lines[0]
	if len(lines) > 1 {
		lastLine = lines[len(lines)-1]
	}
	update := model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: lastLine,
		},
	}
	if q.ID == "lang" {
		c.Props["locale"] = a.ID
		update.Update.AddProp("locale", a.ID)
	} else if q.ID == "job_title" {
		c.Props[q.ID] = lastLine
		update.Update.AddProp(q.ID, c.Props[q.ID])
	}
	name, icon := b.GetNameIconForChannel(c)
	update.Update.AddProp("attachments", []model.SlackAttachment{
		{
			AuthorName: name,
			AuthorIcon: icon,
			Text:       fmt.Sprintf("> %s", b.GetTextAnswer(a, c)),
		},
	})
	responseBody, err := json.Marshal(update)
	if err != nil {
		mlog.Error("Response marshal error",
			mlog.Any("action", action),
			mlog.Any("update", update),
			mlog.Err(err),
		)
		http.Error(w, "Response marshal error", 500)
		return
	}
	_, err = w.Write(responseBody)
	if err != nil {
		mlog.Error("Response write error",
			mlog.Any("action", action),
			mlog.Any("update", update),
			mlog.Err(err),
		)
		http.Error(w, "Response write error", 500)
		return
	}
	mlog.Info("Responded", mlog.Any("action", action), mlog.Any("update", update))
	disablePostID := action.Context["d"].(string)
	target, ok := b.state.FromActions[fmt.Sprintf("%s_%s", q.ID, a.ID)]
	if !ok {
		target = b.state.Specials["catchall"][0]
	}
	go b.PrintQuestion(c, target, disablePostID)
}
