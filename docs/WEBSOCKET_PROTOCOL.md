# WEBSOCKET_PROTOCOL

## Endpoint
GET /ws/connect

Authorization: Bearer JWT

## Envelope

{
  "event":"battle.submit_move",
  "data":{}
}

## Client Events
matchmaking.join
matchmaking.cancel
room.ready
room.unready
battle.submit_move
battle.use_hint
battle.surrender
battle.ping

## Server Events
matchmaking.found
room.updated
room.countdown
battle.started
battle.move_result
battle.opponent_update
battle.final_countdown
battle.ended
battle.reconnect_state
battle.error

## Heartbeat

Client:
battle.ping

Server:
battle.pong

Interval: 15s
Timeout: 45s
