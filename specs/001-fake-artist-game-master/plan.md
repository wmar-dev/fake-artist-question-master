# Implementation Plan: Fake Artist Question Master

**Branch**: `001-fake-artist-game-master` | **Date**: 2026-11-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-fake-artist-game-master/spec.md`

## Summary

Build a phone-first website that replaces the Question Master in A Fake Artist Goes to New York. A host creates a game, friends join by QR code or game code, and the site privately deals each round. It does not handle drawing, votes, or scores.

The site is one Go binary that embeds plain HTML, CSS, and JavaScript, keeps games in memory, and pushes each player a filtered view of the game over Server-Sent Events. The same Docker image runs locally and in the cloud, and only environment variables differ.

## Technical Context

**Language/Version**: Go 1.27 for the server; plain JavaScript, HTML, and CSS with no build step for the browser

**Primary Dependencies**: Go standard library and `rsc.io/qr` v0.2.0 for QR encoding (see [research.md](research.md))

**Storage**: In memory only; games are lost on restart, as the spec's assumptions allow

**Testing**: `go test` with `net/http/httptest`; an injected random source and clock keep tests deterministic

**Target Platform**: Linux container on amd64 and arm64; current phone browsers

**Project Type**: Web application served by a single binary

**Performance Goals**: Updates reach all players within 2 seconds (SC-003); 50 games of 10 players at once (SC-005)

**Constraints**: Resource budgets, each verified as shown:

| Resource | Budget | Verified by |
|----------|--------|-------------|
| Docker image size | 15 MB or less | `quickstart.md` measurement recorded in the change |
| Memory, idle after start | 30 MB or less | `quickstart.md` measurement recorded in the change |
| Memory, 50 games with 500 live connections | 100 MB or less | Automated load test |
| Total page download, uncompressed | 100 KB or less, excluding the QR image | Automated test on embedded files |
| Game update size | 2 KB or less | Automated test at 10 players |
| Network traffic while idle | One keep-alive every 25 seconds per open connection | Automated test with the fake clock |
| Retained games | 200 by default (`MAX_GAMES`); each expires after 2 hours without activity | Automated tests |

**Scale/Scope**: One server process; up to 200 games and 2,000 players; 2 pages; about 10 HTTP endpoints

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Result: PASS before research and after design, with one recorded deviation.

- [x] **I. Well Tested**: Game rules are unit tested in `internal/game`, which has no network or browser use. Each user story has an acceptance test file in `internal/server/`, driven over HTTP, which is the highest level the project automates. Randomness and time are injected.
- [x] **II. Low Resource Usage**: Budgets are stated above, each with a verification method. Timers replace polling, so idle states do no periodic work apart from keep-alives on open connections. Retained games are capped and expire.
- [x] **III. Low Dependency**: The one runtime dependency, `rsc.io/qr`, is justified in research.md and pinned in `go.mod`. Playing does not need a third-party hosted service, and the cloud option is any platform that runs a container.
- [x] **IV. Easy to Use**: SC-001 is the start-or-join criterion. No account is needed. Install, run, and test each take one documented command. The whose-turn rule is met differently, as recorded under Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/001-fake-artist-game-master/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── http-api.md      # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks command)
```

### Source Code (repository root)

```text
cmd/server/
└── main.go              # Reads configuration, starts the server, shuts down cleanly

internal/game/           # Game rules: no HTTP, no real clock, no real randomness
├── game.go              # Game, players, host handover
├── round.go             # Dealing and revealing
├── view.go              # Filtered per-player view
├── store.go             # All games, game codes, capacity, expiry
├── clock.go             # Clock and random source interfaces
├── words.go             # Word list loading
├── words.txt            # Built-in word list
└── *_test.go

internal/gametest/       # Fake clock and scripted random source shared by tests
└── gametest.go

internal/server/         # HTTP layer
├── server.go            # Routes and error responses
├── session.go           # Player cookies
├── events.go            # Server-Sent Events
├── qr.go                # QR image and join address
├── acceptance_us*_test.go  # One file per user story, driven over HTTP
├── load_test.go         # Memory and update-size budgets
└── *_test.go

web/
├── embed.go             # Embeds the files below in the binary
├── index.html           # Home page: create or join
├── game.html            # Join, lobby, role, and reveal screens
├── app.js
└── style.css

Dockerfile               # Test stage, build stage, and minimal runtime image
compose.yaml             # Local hosting
README.md                # Install, run, test, and deployment guide
```

**Structure Decision**: A single Go module. Game rules in `internal/game` stay separate from HTTP in `internal/server`, so the rules can be tested without a network, as Principle I requires. Tests sit beside the code, which is the Go convention, rather than in the template's `tests/` directory.

## Hosting

| Setting | Local | Cloud |
|---------|-------|-------|
| Start | `docker compose up --build` | Deploy the same image to a container platform |
| `PUBLIC_URL` | Optional; otherwise the address the host opened is used | Required, for example `https://game.example.com` |
| Address players use | `http://<computer's network address>:8080` | The platform's HTTPS address |
| Instances | One | Exactly one, always running |

The cloud platform must run one instance without scaling to zero, because games are held in memory, and must not buffer streaming responses. See [research.md](research.md).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle IV: the screen shows the first player and drawing direction, not whose turn it is live | Drawing happens on paper, so the site cannot see turns | Requiring players to tap after each mark adds work at the table and gives nothing back |
