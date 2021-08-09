#!/usr/bin/env python3
import asyncio

from aiohttp import web
from mmpy_bot import Bot, Settings

from chadbot.extended_websocket import patch_event_handler
from chadbot.linkedin import LINKEDIN

bot = Bot(
    settings=Settings(SSL_VERIFY=True, WEBHOOK_HOST_ENABLED=True, WEBHOOK_HOST_URL="http://0.0.0.0"),
    plugins=[LINKEDIN],
)

patch_event_handler(bot.event_handler)

bot.webhook_server.app.router.add_view('/_healthz', lambda _: web.Response(text="Hello"))

loop = asyncio.get_event_loop()
tasks = [
    loop.create_task(bot.run()),
]
loop.run_until_complete(asyncio.wait(tasks))
