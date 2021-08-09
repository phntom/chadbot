import logging
from collections import OrderedDict
from os import environ
from typing import Dict

from mmpy_bot.driver import Driver

from chadbot.dctypes import MultiLingualString, Answer, Question
from chadbot.extended_plugin import ExtendedPlugin
from chadbot.state import g

log = logging.getLogger(__name__)


class FlowQ(ExtendedPlugin):
    team = None
    settings_channel = None

    def on_load(self, driver: Driver):
        self.team = driver.teams.get_team_by_name(environ['BOT_TEAM'])['id']
        self.settings_channel = driver.channels.get_channel_by_name(self.team, environ['SETTINGS_CHANNEL'])['id']
        self.reload_questions()

    def reload_questions(self):
        g.questions.clear()
        g.specials.clear()
        g.from_actions.clear()
        posts = self.driver.posts.get_posts_for_channel(self.settings_channel)
        self.load_questions(posts, g.questions, g.specials, g.from_actions)
        log.info("%d questions loaded", len(g.questions))

    @staticmethod
    def load_questions(posts, g_questions, g_specials, g_from_actions):
        for post_id in posts['order']:
            post = posts['posts'][post_id]
            pre = []
            specials = set()
            from_actions = set()
            answers = OrderedDict()
            question_text = None
            for line in post['message'].split('\n'):
                if line.startswith('#'):
                    q_id, language = FlowQ.load_hashtags(line, specials, from_actions)
                    FlowQ.commit_question(answers, language, pre, q_id, question_text, specials, from_actions,
                                          g_questions, g_specials, g_from_actions)
                elif line.startswith('* '):
                    FlowQ.load_answer(answers, line)
                    if question_text is None:
                        question_text = pre.pop(-1)
                else:
                    pre.append(line)

    @staticmethod
    def load_answer(answers, line):
        answer = line[2:]
        a_id = answer.split()[-1]
        assert a_id.startswith('#')
        answer = answer[:-len(a_id)].strip()
        a_id = a_id[1:]
        answers[a_id] = answer

    @staticmethod
    def load_hashtags(line, specials, from_actions):
        q_id = None
        language = None
        for hashtag in line.split():
            if hashtag.startswith('#l_'):
                language = hashtag[3:]
            elif hashtag.startswith('#q_'):
                q_id = hashtag[3:]
            elif hashtag.startswith('#s_'):
                specials.add(hashtag[3:])
            elif hashtag.startswith('#f_'):
                from_actions.add(hashtag[3:])
        return q_id, language

    @staticmethod
    def commit_question(answers, language, pre, q_id, question_text, specials, from_actions, g_questions, g_specials,
                        g_from_actions):
        assert q_id
        q = g_questions.get(q_id, Question(id=q_id, q=MultiLingualString(None, None), a=OrderedDict(), pre=[]))
        g_questions[q_id] = q
        for special in specials:
            g_specials[special].append(q)
        setattr(q.q, language, question_text)
        for a_id, a_text in answers.items():
            a_item = q.a.get(a_id, Answer(a_id, MultiLingualString(None, None)))
            setattr(a_item.text, language, a_text)
            q.a[a_id] = a_item
        if q.pre:
            for q_pre, pre_text in zip(q.pre, pre):
                setattr(q_pre, language, pre_text)
        else:
            q.pre = [MultiLingualString(**{language: pre_text}) for pre_text in pre]
        for fa in from_actions:
            g_from_actions[fa] = q

    @staticmethod
    def get_text(text: MultiLingualString, t: Dict):
        ret = getattr(text, t['locale'], None)
        if ret is None:
            ret = text.en
        if ret:
            return ret.format(**t)
        return ret

    async def populate_target(self, channel_id):
        t = {}
        for member in self.driver.channels.get_channel_members(channel_id):
            if member['scheme_admin']:
                continue
            user_id = member['user_id']
            if user_id == self.driver.user_id:
                continue
            t = self.driver.users.get_user(user_id)
            t['user_icon'] = f"{environ['EXTERNAL_MM_URL']}{self.driver.default_options['basepath']}" \
                             f"{self.driver.users.endpoint}/{user_id}/image"
            break
        return t
