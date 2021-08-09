#!/usr/bin/env bash

set -ex
# bash build/deploy.sh
# build

# cd ..
VERSION=$(grep -oP "(?<=appVersion: )[0-9.]+\b" charts/chadbot/Chart.yaml)
#echo kubectl scale deployment -n chat chadbot --replicas 0 > build.bat
echo docker build . -f build/Dockerfile -t phntom/chadbot:$VERSION >> build.bat
echo docker push phntom/chadbot:$VERSION >> build.bat
echo helm upgrade -n chat chadbot charts/chadbot -f charts/chadbot/values.d/minthe.secret.yaml --atomic >> build.bat

#source build.bat
