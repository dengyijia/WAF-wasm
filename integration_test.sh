#!/bin/sh
cd integration_test
docker-compose up --build --abort-on-container-exit
cd ..
