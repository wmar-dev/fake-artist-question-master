# HTTP Contract: Fake Artist Question Master

JSON bodies use `Content-Type: application/json`. Requests that act as a player are identified by the session cookie set when they join.

## Pages and health

| Method and path | Result |
|-----------------|--------|
| `GET /` | Home page: create a game or enter a code |
| `GET /g/{code}` | Game page: join, lobby, role, and reveal screens |
| `GET /static/{file}` | Embedded JavaScript and CSS |
| `GET /healthz` | `200 ok`, for platform health checks |

## Actions

| Method and path | Body | Who | Success |
|-----------------|------|-----|---------|
| `POST /api/games` | `{"name":"Sam"}` | Anyone | `201 {"code":"K7MQP"}` and session cookie; caller is host |
| `GET /api/games/{code}` | None | Anyone | `200 {"code":"K7MQP","joinable":true,"joined":false}` |
| `POST /api/games/{code}/players` | `{"name":"Alex"}` | Anyone | `201 {"playerId":"..."}` and session cookie |
| `DELETE /api/games/{code}/players/{id}` | None | Host | `204` |
| `POST /api/games/{code}/rounds` | None | Host | `201`, deals a round and discards any current one |
| `POST /api/games/{code}/reveal` | None | Host | `204` |
| `POST /api/games/{code}/host` | None | Player, after the handover wait | `204`, caller becomes host |
| `GET /api/games/{code}/qr.png` | None | Anyone | `200` PNG encoding the join address |
| `GET /api/games/{code}/events` | None | Player | `200` event stream |

## Event stream

`GET /api/games/{code}/events` returns `text/event-stream`. It sends a `state` event on connect and after every change, and a comment line every 25 seconds as a keep-alive.

```text
event: state
data: {"code":"K7MQP", ...}

event: closed
data: {"reason":"removed"}
```

`closed` has a reason of `removed`, `expired`, or `shutdown`, and the server then ends the stream. Clients stop reconnecting after `closed`.

### State payload

```json
{
  "code": "K7MQP",
  "status": "dealt",
  "you": { "id": "p1", "name": "Sam", "isHost": true },
  "players": [
    { "id": "p1", "name": "Sam", "connected": true, "isHost": true }
  ],
  "minPlayers": 4,
  "maxPlayers": 10,
  "canClaimHost": false,
  "joinUrl": "http://192.168.1.5:8080/g/K7MQP",
  "joinUrlReachable": true,
  "round": {
    "number": 1,
    "category": "Food",
    "firstPlayer": "Alex",
    "word": "Tomato"
  }
}
```

| Field | Rule |
|-------|------|
| `status` | `lobby`, `dealt`, or `revealed` |
| `round` | Absent in `lobby` |
| `round.number` | Counts up with each deal, so a browser can tell a new round from an update |
| `round.word` | Present for artists, and for everyone once revealed. Never present for the fake artist before reveal |
| `round.isFake` | Present and true only on the fake artist's own view |
| `round.fakeName` | Present only when `status` is `revealed` |
| `round.waiting` | Present and true for players who joined after the deal. Before reveal, their `round` holds only this field |
| `joinUrlReachable` | False when no `PUBLIC_URL` is set and the host opened the site on a loopback address (FR-026) |

## Errors

Errors return the status below and a body with a stable code and a message to show the player.

```json
{ "error": "game_full", "message": "This game already has 10 players. Ask the host to remove someone, or start a new game." }
```

| Status | `error` | When |
|--------|---------|------|
| 400 | `invalid_name` | Name is empty, too long, or has control characters |
| 400 | `invalid_request` | Body is malformed or over 1 KB |
| 401 | `not_a_player` | No valid session for this game |
| 403 | `not_host` | Action needs the host |
| 404 | `game_not_found` | Code is wrong or expired |
| 404 | `player_not_found` | Player is not in the game |
| 409 | `name_taken` | Name is already used in this game |
| 409 | `game_full` | Game has 10 players |
| 409 | `wrong_player_count` | Dealing with fewer than 4 players |
| 409 | `no_round` | Reveal with no round in progress |
| 409 | `host_available` | Host handover before the wait is over |
| 409 | `cannot_remove_host` | Host tried to remove themselves |
| 503 | `server_full` | `MAX_GAMES` has been reached |

## Configuration

| Variable | Default | Meaning |
|----------|---------|---------|
| `PORT` | `8080` | Port to listen on |
| `PUBLIC_URL` | Empty | Base address for join links and QR codes. Required in the cloud |
| `MAX_GAMES` | `200` | Maximum live games |
