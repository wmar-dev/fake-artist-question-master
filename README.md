# Fake Artist Question Master

A phone-friendly Question Master for [A Fake Artist Goes to New York](https://oinkgames.com/en/games/analog/a-fake-artist-goes-to-new-york). One person creates a game, everyone else scans a QR code, and the site privately deals the category, the secret word, and the fake artist.

Because the site is the Question Master, everyone gets to play, and nobody has to sit out.

Everyone draws on real paper and votes out loud. The site doesn't handle drawing, votes, or scores.

## Commands

You need Docker with Compose. Go 1.27 is only needed to run the tests without Docker.

| Task | Command |
|------|---------|
| Install | `docker compose build` |
| Run | `docker compose up --build` |
| Test | `docker build --target test .` |
| Test with Go | `go test ./...` |

## Host it locally

Use this to play at home or at a venue, with every phone on the same network.

1. Run `docker compose up --build`.
2. Find your computer's network address. On macOS, run `ipconfig getifaddr en0`.
3. Open `http://<network address>:8080` on your computer, not `localhost`. Phones can't open `localhost`, and the site warns you if you use it.
4. Create a game and have everyone scan the QR code.

If your computer's firewall asks, allow incoming connections on port 8080.

## Deploy to the cloud

Build the image with `docker build -t fake-artist .`, then push it to any platform that runs containers. The same image runs locally and in the cloud. The image is about 9 MB.

Your platform must:

| Requirement | Why |
|-------------|-----|
| Run exactly one instance, and not scale to zero | Games are held in memory. A second instance wouldn't see them, and stopping the instance ends them. |
| Set `PUBLIC_URL` to the site's HTTPS address | QR codes and secure cookies depend on it. |
| Send traffic to the port in `PORT` | The default is 8080. |
| Not buffer streaming responses | Phones get live updates over a long-lived response. |
| Provide HTTPS | The server only speaks plain HTTP. |

If your platform limits how long a request can last, phones reconnect by themselves and keep their place in the game.

For example, on Google Cloud Run, deploy with `--min-instances=1 --max-instances=1 --no-cpu-throttling` and set `PUBLIC_URL` to the service's address. On Fly.io, run one machine with `auto_stop_machines` off. Check your platform's current documentation for the exact settings.

Use `GET /healthz` for health checks. Restarting or redeploying ends any games in progress.

## Configuration

| Variable | Default | Meaning |
|----------|---------|---------|
| `PORT` | `8080` | Port to listen on |
| `PUBLIC_URL` | Empty | Base address for join links and QR codes. Required in the cloud |
| `MAX_GAMES` | `200` | Maximum games at once |

## Project layout

| Path | Contents |
|------|----------|
| `cmd/server` | Starts the server |
| `internal/game` | Game rules, with no HTTP |
| `internal/server` | HTTP routes, live updates, and QR codes |
| `web` | Pages, embedded in the binary |

The design is in [specs/001-fake-artist-game-master](specs/001-fake-artist-game-master).

This is an unofficial fan project, and isn't affiliated with Oink Games.
