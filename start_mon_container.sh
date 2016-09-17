#!/bin/bash

docker run -it --env-file=env.asc --name mon -p 80:8080 hkjn/armv7l-golang
