import logging
from os import environ
from re import IGNORECASE

from mmpy_bot import listen_to, Message, listen_webhook, WebHookEvent, ActionEvent

from chadbot.flowq import FlowQ
from chadbot.state import g

log = logging.getLogger(__name__)


class LinkedIn(FlowQ):
    channel_targets = {}

    @listen_to("start", IGNORECASE)
    async def start(self, message: Message):
        channel_id = message.channel_id
        self.channel_targets[channel_id] = await self.populate_target(channel_id)
        await self.print_question(channel_id, g.specials['init'][0].id)

    @listen_to("reload", IGNORECASE, allowed_users=['phantom'])
    async def reload(self, message: Message):
        self.reload_questions()
        self.driver.reply_to(message, f"reloaded {len(g.questions)} questions.")

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
