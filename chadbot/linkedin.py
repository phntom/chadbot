import logging
from collections import OrderedDict
from os import environ
from re import IGNORECASE
from typing import Dict

from mmpy_bot import listen_to, Message, listen_webhook, WebHookEvent, ActionEvent
from mmpy_bot.driver import Driver

from chadbot.dctypes import Answer, MultiLingualString, Question
from chadbot.extended_plugin import ExtendedPlugin
from chadbot.state import g

log = logging.getLogger(__name__)


class LinkedIn(ExtendedPlugin):
    team = None
    settings_channel = None
    channel_targets = {}

    def on_load(self, driver: Driver):
        self.team = driver.teams.get_team_by_name(environ['BOT_TEAM'])['id']
        self.settings_channel = driver.channels.get_channel_by_name(self.team, environ['SETTINGS_CHANNEL'])['id']
        posts = driver.posts.get_posts_for_channel(self.settings_channel)
        self.load_questions(posts)
        log.info("%d questions loaded", len(g.questions))
        assert g.questions

    @staticmethod
    def load_questions(posts):
        for post_id in posts['order']:
            post = posts['posts'][post_id]
            pre = []
            specials = set()
            from_actions = set()
            answers = OrderedDict()
            question_text = None
            for line in post['message'].split('\n'):
                if line.startswith('#'):
                    q_id, language = LinkedIn.load_hashtags(line, specials, from_actions)
                    LinkedIn.commit_question(answers, language, pre, q_id, question_text, specials, from_actions)
                elif line.startswith('* '):
                    LinkedIn.load_answer(answers, line)
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
    def commit_question(answers, language, pre, q_id, question_text, specials, from_actions):
        assert q_id
        q = g.questions.get(q_id, Question(id=q_id, q=MultiLingualString(None, None), a=OrderedDict(), pre=[]))
        g.questions[q_id] = q
        for special in specials:
            g.specials[special].append(q)
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
            g.from_actions[fa] = q

    async def print_question(self, channel_id, question_id, disable_post_id=''):
        q = g.questions[question_id]
        t = self.channel_targets[channel_id]
        if q.pre:
            for pre_q in q.pre:
                self.driver.create_post(channel_id, self.get_text(pre_q, t))
                await self.user_typing(channel_id)
        props = {
            "attachments": [
                {
                    "actions": [
                        {
                            "id": a.id,
                            "name": self.get_text(a.text, t),
                            "integration": {
                                "url": f'{environ["WEBHOOK_HOST"]}/hooks/answer',
                                "context": {
                                    "q": q.id,
                                    "a": a.id,
                                    "d": disable_post_id,
                                }
                            }
                        } for a in q.a.values()
                    ],
                    "footer": f"{t['first_name']} {t['last_name']}",
                    "footer_icon": t['user_icon'],
                    "title": self.get_text(q.q, t),
                }
            ]
        }
        self.driver.create_post(channel_id, "", props=props)

    @listen_to("start", IGNORECASE)
    async def start(self, message: Message):
        channel_id = message.channel_id
        t = {}
        for member in self.driver.channels.get_channel_members(channel_id):
            if member['scheme_admin']:
                continue
            user_id = member['user_id']
            if user_id == self.driver.user_id:
                continue
            t = self.driver.users.get_user(user_id)
            t['user_icon'] = f"{environ['EXTERNAL_MM_URL']}{self.driver.default_options['basepath']}{self.driver.users.endpoint}/{user_id}/image"
            break
        self.channel_targets[channel_id] = t
        await self.print_question(channel_id, g.specials['init'][0].id)

    @staticmethod
    def get_text(text: MultiLingualString, t: Dict):
        ret = getattr(text, t['locale'], None)
        if ret is None:
            ret = text.en
        if ret:
            return ret.format(**t)
        return ret

    @listen_webhook("answer")
    async def answer(self, event: WebHookEvent):
        if not isinstance(event, ActionEvent):
            return
        t = self.channel_targets[event.channel_id]
        q = g.questions[event.context['q']]
        a_id = event.context['a']
        a_text = self.get_text(q.a[a_id].text, t)

        if q.id == 'lang':
            t['locale'] = a_id
        elif q.id == 'job_title':
            t[q.id] = a_text

        self.driver.respond_to_web(
            event,
            {
                "update": {"message": self.get_text(q.q, t), "props": {
                    "attachments": [
                        {
                            "author_name": f"{t['first_name']} {t['last_name']}",
                            "author_icon": t['user_icon'],
                            "text": "> " + a_text,
                            # "actions": [
                            #     {
                            #         "id": "edit",
                            #         "name": "Edit",
                            #         "integration": {
                            #             "url": f'{environ["WEBHOOK_HOST"]}/hooks/edit',
                            #             "context": {
                            #                 "q": q.id,
                            #                 "a": a_id,
                            #             }
                            #         }
                            #     }
                            # ],
                        }
                    ]
                }},
            },
        )
        await self.user_typing(event.channel_id)

        q = g.from_actions.get(f'{q.id}_{a_id}')
        if q is None:
            q = g.specials['catchall'][0]

        # disable_post_id = event.context.get('d')
        # if disable_post_id:
        #     post = self.driver.posts.get_post(disable_post_id)
        #     # log.info(json.dumps(post, indent=4, sort_keys=True))
        #     post['props']['attachments'][0].pop('action', None)
        #     self.driver.posts.update_post(disable_post_id, options={
        #         'id': post['id'],
        #         'message': post['message'],
        #         'is_pinned': post['is_pinned'],
        #         'props': post['props'],
        #     })

        await self.print_question(event.channel_id, q.id, event.post_id)

    @listen_webhook("health")
    async def health(self, event: WebHookEvent):
        pass


LINKEDIN = LinkedIn()
