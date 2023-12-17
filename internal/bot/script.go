package bot

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"strings"
)

type ScriptYaml struct {
	ID                string                `yaml:"i"`
	Special           string                `yaml:"s"`
	QuestionEng       interface{}           `yaml:"q"`
	QuestionHebMale   interface{}           `yaml:"qm"`
	QuestionHebFemale interface{}           `yaml:"qf"`
	ResponseEng       string                `yaml:"r"`
	ResponseHebMale   string                `yaml:"rm"`
	ResponseHebFemale string                `yaml:"rf"`
	Link              string                `yaml:"l"`
	Answer            map[string]ScriptYaml `yaml:"a"`
}

type ScriptQuestion struct {
	ID                string
	Special           string
	QuestionEng       string
	QuestionHebMale   string
	QuestionHebFemale string
	Link              string
	AnswerOrder       []string
	Answer            map[string]*ScriptAnswer
}

type ScriptAnswer struct {
	ID                string
	ResponseEng       string
	ResponseHebMale   string
	ResponseHebFemale string
}

type ScriptState struct {
	Questions   map[string]*ScriptQuestion
	Specials    map[string][]*ScriptQuestion
	FromActions map[string]*ScriptQuestion
}

func (b *Bot) LoadScript() {
	ctx := context.Background()
	postsForChannel, response, err := b.client.GetPostsForChannel(
		ctx,
		b.configChannel.Id,
		0,
		100,
		"",
		true,
		false,
	)
	if err != nil {
		mlog.Error("Failed loading posts from configuration channel",
			mlog.Any("configChannel", b.configChannel),
			mlog.Any("response", response),
			mlog.Err(err),
		)
		os.Exit(3)
	}
	state := ScriptState{
		Questions:   make(map[string]*ScriptQuestion),
		Specials:    make(map[string][]*ScriptQuestion),
		FromActions: make(map[string]*ScriptQuestion),
	}
	for _, post := range postsForChannel.Posts {
		if !strings.HasPrefix(post.Message, "```yaml") {
			continue
		}
		y := strings.Replace(strings.Replace(post.Message, "```yaml", "", 1), "```", "", 1)
		var loadedScript ScriptYaml
		err := yaml.Unmarshal([]byte(y), &loadedScript)
		if err != nil {
			mlog.Error("Failed to unmarshal message", mlog.Any("y", y), mlog.Err(err))
			continue
		}
		ParseQuestion(loadedScript, state)
	}
	b.state = state
}

func ParseQuestion(script ScriptYaml, state ScriptState) {
	if script.ID == "" {
		// probably a link, no need to load a new question
		return
	}
	q, ok := state.Questions[script.ID]
	if !ok {
		q = &ScriptQuestion{}
		state.Questions[script.ID] = q
	}
	q.ID = script.ID
	q.Special = script.Special
	q.QuestionEng = ProcessString(script.QuestionEng)
	q.QuestionHebMale = ProcessString(script.QuestionHebMale)
	q.QuestionHebFemale = ProcessString(script.QuestionHebFemale)
	q.Link = script.Link
	q.Answer = make(map[string]*ScriptAnswer)

	if q.Special != "" {
		state.Specials[q.Special] = append(state.Specials[q.Special], q)
	}
	if len(script.Answer) > 0 {
		for aID, aScript := range script.Answer {
			a := ScriptAnswer{
				ID:                aID,
				ResponseEng:       ProcessString(aScript.ResponseEng),
				ResponseHebMale:   ProcessString(aScript.ResponseHebMale),
				ResponseHebFemale: ProcessString(aScript.ResponseHebFemale),
			}
			q.Answer[aID] = &a
			q.AnswerOrder = append(q.AnswerOrder, aID)
			target := aScript.Link
			if target == "" {
				target = aScript.ID
			}
			qa, ok := state.Questions[target]
			if !ok {
				qa = &ScriptQuestion{}
				state.Questions[target] = qa
			}
			state.FromActions[fmt.Sprintf("%s_%s", q.ID, aID)] = qa
			if aScript.QuestionEng != nil {
				//aScript.ID = aID
				ParseQuestion(aScript, state)
			}
		}
	}
}

func ProcessString(in interface{}) string {
	if in == nil {
		return ""
	}
	if reflect.TypeOf(in).Kind() == reflect.String {
		return in.(string)
	} else {
		var result []string
		for _, s := range in.([]interface{}) {
			result = append(result, s.(string))
		}
		return strings.Join(result, "\n")
	}
}
