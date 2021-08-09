import asyncio
import json
from typing import Optional

from mattermostdriver.exceptions import NotEnoughPermissions
from mmpy_bot import Plugin, Message
from mmpy_bot.driver import Driver
from mmpy_bot.settings import Settings


class ExtendedPlugin(Plugin):
    seq = 1
    direct_channels = {}

    def initialize(self, driver: Driver, settings: Optional[Settings] = None):
        super().initialize(driver, settings)
        self.on_load(driver)
        return self

    def on_load(self, driver: Driver):
        pass

    async def user_typing(self, channel_id: str, parent_post_id: Optional[str] = None):
        if not parent_post_id:
            parent_post_id = ''
        self.seq += 1
        json_data = json.dumps({
            "seq": self.seq,
            "action": "user_typing",
            "data": {
                "channel_id": channel_id,
                "parent_id": parent_post_id,
            }
        }).encode('utf8')
        await self.driver.websocket.websocket.send(json_data)
        await asyncio.sleep(1)

    async def direct_reply(self, message: Message, response: str, human=True, *nargs, **kwargs):
        if not message.is_direct_message:
            try:
                return self.driver.reply_to(message, response, ephemeral=True, *nargs, **kwargs)
            except NotEnoughPermissions:
                pass
        direct_channel_id = self.direct_channels.get(message.user_id)
        if not direct_channel_id:
            result = self.driver.channels.create_direct_message_channel([message.user_id, self.driver.user_id])
            direct_channel_id = result['id']
            self.direct_channels[message.user_id] = direct_channel_id
        if message.is_direct_message and human:
            await self.user_typing(direct_channel_id)
        return self.driver.create_post(direct_channel_id, response, *nargs, **kwargs)
