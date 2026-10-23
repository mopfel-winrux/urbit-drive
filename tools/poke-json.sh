#!/bin/bash
# Poke an agent with JSON over eyre and print the ack.
# usage: tools/poke-json.sh <port> <cookie-file> <ship-without-~> <app> '<json>'
PORT=$1; C=$(cat "$2"); SHIP=$3; APP=$4; JSON=$5; CH="poke-$(date +%s%N)"
curl -s -H "Cookie: $C" -X PUT "localhost:$PORT/~/channel/$CH" \
  -d "[{\"id\":1,\"action\":\"poke\",\"ship\":\"$SHIP\",\"app\":\"$APP\",\"mark\":\"json\",\"json\":$JSON}]"
timeout 5 curl -s -N -H "Cookie: $C" "localhost:$PORT/~/channel/$CH" | grep -m1 -o '"ok":"[^"]*"\|"err":"[^"]*"'
curl -s -o /dev/null -H "Cookie: $C" -X DELETE "localhost:$PORT/~/channel/$CH"
