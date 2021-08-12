import logging
import re
from collections import OrderedDict
from os import environ
from typing import Dict

import yaml
from mmpy_bot.driver import Driver

from chadbot.dctypes import MultiLingualString, Answer, Question
from chadbot.extended_plugin import ExtendedPlugin
from chadbot.state import g

log = logging.getLogger(__name__)


class FlowQ(ExtendedPlugin):
    team = None
    settings_channel = None
    phantom_user_id = None

    def on_load(self, driver: Driver):
        self.team = driver.teams.get_team_by_name(environ['BOT_TEAM'])['id']
        self.settings_channel = driver.channels.get_channel_by_name(self.team, environ['SETTINGS_CHANNEL'])['id']
        self.reload_questions()
        self.phantom_user_id = self.driver.users.get_user_by_username('phantom').get('id')

    def reload_questions(self):
        g.questions.clear()
        g.specials.clear()
        g.from_actions.clear()
        posts = self.driver.posts.get_posts_for_channel(self.settings_channel)
        self.load_questions(posts, g.questions, g.specials, g.from_actions)
        log.info("%d questions loaded", len(g.questions))

    @staticmethod
    def load_questions(posts, g_questions, g_specials, g_from_actions):
        yaml_extractor = re.compile(r'(```yaml)(.*)(```)', re.DOTALL).match
        for post_id in reversed(posts['order']):
            post = posts['posts'][post_id]
            if '```yaml' not in post['message']:
                continue
            FlowQ.load_question(
                yaml.safe_load(yaml_extractor(post['message']).group(2)), g_questions, g_specials, g_from_actions
            )
        for qk, qq in g_from_actions.items():
            if isinstance(qq, str):
                g_from_actions[qk] = g_questions[qq]

    @staticmethod
    def ml_from_index(y: Dict, idx: str) -> MultiLingualString:
        q = y[idx]
        if isinstance(q, list):
            q = '\n'.join(q)
        qm = y.get(idx + 'm', q)
        if isinstance(qm, list):
            qm = '\n'.join(qm)
        qf = y.get(idx + 'f', qm)
        if isinstance(qf, list):
            qf = '\n'.join(qf)
        return MultiLingualString(q, qm, qf)

    @staticmethod
    def load_question(y: Dict, g_questions, g_specials, g_from_actions):
        if 'i' not in y:
            # probably a link, no need to load a new question
            return
        i = y['i']
        q = g_questions.get(i, Question(i, MultiLingualString(), OrderedDict()))
        q.q = FlowQ.ml_from_index(y, 'q')
        g_questions[i] = q
        if 's' in y:
            g_specials[y['s']].append(q)
        if 'a' in y:
            for a_id, a in y['a'].items():
                q.a[a_id] = Answer(a_id, FlowQ.ml_from_index(a, 'r'))
                FlowQ.load_question(a, g_questions, g_specials, g_from_actions)
                answer_target = a.get('l', a.get('i'))
                g_from_actions[f'{i}_{a_id}'] = g_questions.get(answer_target, answer_target)

    @staticmethod
    def load_answer(answers, line):
        answer = line[2:]
        a_id = answer.split()[-1]
        assert a_id.startswith('#')
        answer = answer[:-len(a_id)].strip()
        a_id = a_id[1:]
        answers[a_id] = answer

    @staticmethod
    def get_text(text: MultiLingualString, t: Dict):
        ret = getattr(text, t.get('locale', 'en'), None)
        if ret is None:
            ret = text.en
        if ret:
            return ret.format(**t)
        return ret

    async def populate_target(self, channel_id):
        t = None
        for member in self.driver.channels.get_channel_members(channel_id):
            if member['scheme_admin']:
                continue
            user_id = member['user_id']
            if user_id in (self.driver.user_id, self.phantom_user_id):
                continue
            t = self.driver.users.get_user(user_id)
            t['user_icon'] = f"{environ['EXTERNAL_MM_URL']}{self.driver.default_options['basepath']}" \
                             f"{self.driver.users.endpoint}/{user_id}/image"
            t['job_title'] = 'Engineer'
            break
        return t
