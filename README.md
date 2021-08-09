# chadbot

### Purpose

Chad bot will handle initial conversations with sourcing specialists to quickly filter out irrelevant propositions.

The avatar picture is of Chad Kroeger - lead singer of Nickelback, husband of Avril Lavigne.

The bot allows for quick flowchart style conversations with buttons for responses (similar to telegram bots).

Conversations are multilingual and can be edited from a dedicated channel.

Future versions will also employ forms for req descriptions.

### To deploy a new version:

```shell
VERSION=$(grep -oP "(?<=appVersion: )[0-9.]+\b" charts/chadbot/Chart.yaml)
docker build . -f build/Dockerfile -t phntom/chadbot:$VERSION
docker push phntom/chadbot:$VERSION
helm upgrade -n chat chadbot charts/chadbot -f charts/chadbot/values.d/minthe.secret.yaml --atomic

```