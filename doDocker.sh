#!/bin/bash
clean=false
push=false
while getopts cp: flag
do
    case "${flag}" in
        c) clean=true;;
        p) push=${OPTARG};;
    esac
done
#Open Docker, only if is not running
if (! docker stats --no-stream ); then
  # On Mac OS this would be the terminal command to launch Docker
  open /Applications/Docker.app
 #Wait until Docker daemon is running and has completed initialisation
while (! docker stats --no-stream ); do
  # Docker takes a few seconds to initialize
  echo "Waiting for Docker to launch..."
  sleep 1
done
fi
if (! test "$push" = false); then
    git add .
    git commit -m "$push"
    git push
fi
if ("$clean" = true); then 
    docker stop "$(docker ps -q)"
    docker rm "$(docker ps -a -q)"
    docker rmi $(docker images -q)
    docker build -t rs . --no-cache --pull
else
    docker build -t rs .
fi

docker run -it --rm -p 80:8080 rs