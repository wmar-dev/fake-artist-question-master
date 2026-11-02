# Quickstart: Validate the Fake Artist Question Master

These steps prove the feature works once it is implemented. See [http-api.md](contracts/http-api.md) for the contract and [plan.md](plan.md) for the budgets.

## Prerequisites

- Docker with Compose
- A computer and at least 4 phones or browser profiles on the same network. Use a separate private window for each player on one computer.
- Go 1.27, only to run tests without Docker

## Commands

| Task | Command |
|------|---------|
| Install | `docker compose build` |
| Run locally | `docker compose up --build` |
| Test with Docker | `docker build --target test .` |
| Test with Go | `go test ./...` |

## Scenario 1: Local hosting

1. Run the site, and find the computer's network address. On macOS, use `ipconfig getifaddr en0`.
2. Open `http://localhost:8080` and create a game. **Expected**: a message explains that phones cannot reach this address.
3. Open `http://<network address>:8080` and create a game. **Expected**: a QR code and a game code, and no warning.
4. Scan the code with a phone camera. **Expected**: the join page opens with no typing.
5. On another device, type the game code instead. **Expected**: the same join page opens.
6. Join with a name on each device. **Expected**: names appear on every screen within 2 seconds.

## Scenario 2: Dealing

1. With 3 players, try to deal. **Expected**: the site says that more players are needed.
2. With at least 4 players, deal. **Expected**: everyone sees the same category and first player. Exactly one player is the fake artist, and the rest see the same word.
3. Hide and show a role. **Expected**: the word is not visible while hidden.
4. On the fake artist's device, inspect the network traffic. **Expected**: the word appears nowhere.
5. Lock a phone for 30 seconds and unlock it. **Expected**: the same role is back.

## Scenario 3: Reveal and repeat

1. Reveal. **Expected**: every screen shows the fake artist and the word.
2. Deal again several times. **Expected**: the word changes each time and the players stay.
3. Join a new player during a round. **Expected**: they wait, and receive a role in the next round.

## Scenario 4: Multiple games

1. Run two games in separate browser profiles. **Expected**: players, rounds, and words stay separate.

## Scenario 5: Host handover

1. Close the host's browser and wait for 1 minute. **Expected**: another player is offered the host role and can take it.

## Scenario 6: Cloud deployment

1. Deploy the same image to a container platform with exactly one always-on instance and `PUBLIC_URL` set to its HTTPS address.
2. Open `/healthz`. **Expected**: `ok`.
3. Repeat Scenarios 1 to 3 over mobile data. **Expected**: same results, and the QR code uses `PUBLIC_URL`.
4. Leave a game open for 2 minutes. **Expected**: the stream stays connected, or reconnects without the player noticing.

## Budget measurements

Record these in the change that completes the feature:

| Budget | How to measure | Limit |
|--------|----------------|-------|
| Image size | `docker image ls` for the built image | 15 MB |
| Idle memory | `docker stats --no-stream` after start | 30 MB |

The other budgets are covered by `go test ./...`.

## Recorded results

Run on 2026-11-02 against the Docker image on macOS (arm64).

| Budget | Result | Limit |
|--------|--------|-------|
| Image size | 9.14 MB | 15 MB |
| Idle memory | 6.5 MiB | 30 MB |
| Page files | 18 KB | 100 KB |
| Memory, 50 games with 500 connections | 24 MB of heap, including the test's own clients | 100 MB |
| Time to update 500 players | About 7 ms on one machine, with no real network | 2 seconds |
| Largest update | 1,141 bytes | 2 KB |

| Scenario | Result |
|----------|--------|
| 1, 2, 3, and 5 | Passed in headless Chrome at a phone-sized viewport, with one isolated browser profile per player, over the computer's network address. Host handover ran on the real 1-minute clock. A page reload stood in for locking a phone |
| 4, and joining during a round | Covered by the automated HTTP tests, not run in a browser |
| QR code | Decoded with macOS Core Image to the exact join address, but not yet scanned with a phone camera |
| 1 to 5 on real phones | Not yet run |
| 6, cloud deployment | Not yet run |
| Shutdown | `docker compose stop` with a live stream: the stream received `closed` with reason `shutdown`, and the container exited with code 0 in 0.36 seconds |
