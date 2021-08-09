from mattermostdriver import Websocket


class ExtendedWebsocket(Websocket):
    websocket = None

    async def _authenticate_websocket(self, websocket, event_handler):
        self.websocket = websocket
        await super()._authenticate_websocket(websocket, event_handler)


# noinspection PyProtectedMember
def patch_event_handler(event_handler):
    event_handler.start = lambda: event_handler.driver.init_websocket(event_handler._handle_event, ExtendedWebsocket)
