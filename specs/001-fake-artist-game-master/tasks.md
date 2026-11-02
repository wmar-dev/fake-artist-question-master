---

description: "Task list for implementing the Fake Artist Question Master"
---

# Tasks: Fake Artist Question Master

**Input**: Design documents from `/specs/001-fake-artist-game-master/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/http-api.md](contracts/http-api.md), [quickstart.md](quickstart.md)

**Tests**: Tests are REQUIRED by the constitution (Principle I). Every user story includes test tasks, written before the implementation tasks they cover. Test tasks must be seen to fail before the implementation starts.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create `go.mod` with module `fakeartist`, `go 1.27`, and `rsc.io/qr` pinned at v0.2.0, and generate `go.sum`
- [X] T002 [P] Create `.gitignore` for build output and `.dockerignore` excluding `.git`, `specs`, `.specify`, and `.claude`
- [X] T003 [P] Create `Dockerfile` with a `test` stage that runs `go test ./...`, a build stage that compiles a static binary with `CGO_ENABLED=0 -trimpath -ldflags="-s -w"`, and a final `scratch` stage that runs as user 65534 on port 8080
- [X] T004 [P] Create `compose.yaml` that builds the image, publishes port 8080, sets `restart: unless-stopped`, and passes through `PUBLIC_URL` and `MAX_GAMES`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Define the `Clock` interface (`Now`, `AfterFunc`) and `Rand` interface (`IntN`, `Token`) with real implementations using `crypto/rand` in `internal/game/clock.go`
- [X] T006 Create a fake clock that fires timers when advanced, and a scripted `Rand`, in `internal/gametest/gametest.go`
- [X] T007 [P] Write tests that the word list has at least 200 words across at least 10 categories, with no duplicates or blank fields, in `internal/game/words_test.go`
- [X] T008 [P] Write the built-in word list as `category,word` lines of easy-to-draw English words in `internal/game/words.txt`
- [X] T009 Implement word list embedding and parsing in `internal/game/words.go`
- [X] T010 Write tests for name rules (1 to 20 characters after trimming, no control characters, unique ignoring case), the 10-player limit, and removing players in `internal/game/game_test.go`
- [X] T011 Implement `Game` and `Player` with their mutex and the name, join, and remove rules from `data-model.md` in `internal/game/game.go`
- [X] T012 Write tests for 5-character codes from the agreed alphabet, case-insensitive lookup, `MAX_GAMES` refusal, and removal after 2 hours without activity using the fake clock in `internal/game/store_test.go`
- [X] T013 Implement the game `Store` with per-game expiry timers reset on activity in `internal/game/store.go`
- [X] T014 Write tests for `GET /healthz`, the error body format, and the 1 KB request limit in `internal/server/server_test.go`
- [X] T015 Implement the `Server` type, routing, JSON error helper, and request size limit in `internal/server/server.go`
- [X] T016 [P] Embed the `web` directory in `web/embed.go` and serve pages and `/static/` files through `internal/server/server.go`
- [X] T017 Read `PORT`, `PUBLIC_URL`, and `MAX_GAMES`, validate them, and start the HTTP server in `cmd/server/main.go`

**Checkpoint**: Foundation ready. `go test ./...` passes and the server answers `/healthz`.

---

## Phase 3: User Story 1 - Start a game and let friends join by QR code (Priority: P1) 🎯 MVP

**Goal**: A host creates a game, and friends join by QR code or game code and appear on every screen.

**Independent Test**: Create a game on one device, join from others by QR code and by typing the code, and confirm every device shows the same player list.

### Tests for User Story 1 (REQUIRED) ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T018 [P] [US1] Write acceptance tests for creating a game, joining by code, the live player list on all event streams, removing a player, and the `invalid_name`, `name_taken`, `game_full`, `game_not_found`, `not_host`, and `cannot_remove_host` errors in `internal/server/acceptance_us1_test.go`
- [X] T019 [P] [US1] Write tests that the session cookie is `HttpOnly` and `SameSite=Lax`, is scoped to the game's path, and adds `Secure` only when `PUBLIC_URL` uses HTTPS in `internal/server/session_test.go`
- [X] T020 [P] [US1] Write tests for the event stream: a `state` event on connect and on change, one keep-alive every 25 seconds using the fake clock, a `closed` event when removed or expired, and `X-Accel-Buffering: no` and `Cache-Control: no-cache` headers in `internal/server/events_test.go`
- [X] T021 [P] [US1] Write tests that the join address uses `PUBLIC_URL` when set, otherwise the request host, ignores forwarded headers, sets `joinUrlReachable` to false for `localhost`, `127.0.0.0/8`, and `::1`, and that `qr.png` returns a PNG in `internal/server/qr_test.go`

### Implementation for User Story 1

- [X] T022 [US1] Build the player view for lobby state, including the fields in `contracts/http-api.md` other than `round`, in `internal/game/view.go`
- [X] T023 [US1] Add change subscriptions and connection counting to `Game` in `internal/game/game.go`
- [X] T024 [US1] Implement token creation and session cookies in `internal/server/session.go`
- [X] T025 [US1] Implement Server-Sent Events with per-player views, keep-alives through `Clock`, and `closed` events in `internal/server/events.go`
- [X] T026 [US1] Implement the join address and `GET /api/games/{code}/qr.png` in `internal/server/qr.go`
- [X] T027 [US1] Add create, look up, join, and remove handlers in `internal/server/server.go`
- [X] T028 [P] [US1] Build the home page with create and join-by-code forms in `web/index.html`
- [X] T029 [P] [US1] Build the join and lobby screens, with QR code, game code, player list, and unreachable-address message, in `web/game.html`
- [X] T030 [US1] Implement form handling, `EventSource` rendering with `textContent`, plain error messages, and stopping after a `closed` event in `web/app.js`
- [X] T031 [P] [US1] Add phone-first styling with large tap targets in `web/style.css`

**Checkpoint**: User Story 1 works on its own. Quickstart Scenario 1 passes.

---

## Phase 4: User Story 2 - Receive a secret role privately (Priority: P1) 🎯 MVP

**Goal**: The host deals a round, and every player privately sees the right role.

**Independent Test**: Deal a round with at least 4 players and confirm that exactly one is the fake artist, all others see the same word, and all see the same category and first player.

### Tests for User Story 2 (REQUIRED) ⚠️

- [X] T032 [P] [US2] Write tests that dealing picks one fake artist from the participants, picks the first player independently, gives each player an equal chance over many seeded deals, requires 4 to 10 players, steps through the deck without repeats, and wraps oldest-first in `internal/game/round_test.go`
- [X] T033 [P] [US2] Write table tests for every row of the Player View table in `data-model.md`, including that the marshalled JSON for the fake artist does not contain the word anywhere, in `internal/game/view_test.go`
- [X] T034 [P] [US2] Write acceptance tests for dealing, `wrong_player_count`, `not_host`, the same role after reconnecting with the same cookie, and a late joiner seeing `waiting` in `internal/server/acceptance_us2_test.go`

### Implementation for User Story 2

- [X] T035 [US2] Implement `Round`, the per-game shuffled deck, and dealing in `internal/game/round.go`
- [X] T036 [US2] Add the `round` fields and their filtering rules to the player view in `internal/game/view.go`
- [X] T037 [US2] Add the deal handler in `internal/server/server.go`
- [X] T038 [US2] Add the deal button, artist, fake artist, and waiting screens, drawing rules, and a hide and show control in `web/game.html`
- [X] T039 [US2] Render the round screens and keep the hide and show choice on the device in `web/app.js`

**Checkpoint**: User Stories 1 and 2 work together as the MVP. Quickstart Scenario 2 passes, and a group can play a full round.

---

## Phase 5: User Story 3 - Reveal the answer and deal again (Priority: P2)

**Goal**: The host reveals the fake artist and word on every screen, then deals again.

**Independent Test**: Reveal a round, confirm every screen shows the same fake artist and word, then deal again and confirm a new word and random fake artist.

### Tests for User Story 3 (REQUIRED) ⚠️

- [X] T040 [P] [US3] Write tests that revealing sets the status to `revealed`, and that the reveal still shows the fake artist and first player names after they are removed, in `internal/game/round_test.go`
- [X] T041 [P] [US3] Write acceptance tests for revealing to all players, `not_host`, `no_round`, dealing after a reveal, dealing during a round to discard it, and no repeated word across rounds in `internal/server/acceptance_us3_test.go`

### Implementation for User Story 3

- [X] T042 [US3] Implement reveal in `internal/game/round.go`
- [X] T043 [US3] Add `fakeName` and word to the view after reveal in `internal/game/view.go`
- [X] T044 [US3] Add the reveal handler in `internal/server/server.go`
- [X] T045 [US3] Add the reveal screen and the host's reveal and deal-again buttons in `web/game.html`
- [X] T046 [US3] Render the reveal screen and host controls in `web/app.js`

**Checkpoint**: Quickstart Scenario 3 passes.

---

## Phase 6: User Story 4 - Run many games at once (Priority: P3)

**Goal**: Separate groups play at the same time with nothing shared between them.

**Independent Test**: Run two games at once and confirm that players, rounds, and words stay separate.

### Tests for User Story 4 (REQUIRED) ⚠️

- [X] T047 [P] [US4] Write acceptance tests that two games stay separate, that a cookie from one game gets `not_a_player` in another, and that creating a game past `MAX_GAMES` returns `server_full` in `internal/server/acceptance_us4_test.go`
- [X] T048 [P] [US4] Write a load test with 50 games and 500 live connections that checks updates arrive within 2 seconds, heap use stays at or under 100 MB, and a 10-player update is at most 2 KB in `internal/server/load_test.go`
- [X] T049 [P] [US4] Write a concurrency test that runs joins, deals, reveals, and disconnects across games at once, for use with `go test -race`, in `internal/game/store_test.go`

### Implementation for User Story 4

- [X] T050 [US4] Fix any isolation, locking, or performance problems the tests find in `internal/game/store.go` and `internal/game/game.go`

**Checkpoint**: All user stories work on their own. Quickstart Scenario 4 passes.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

### Host handover (FR-021)

- [X] T051 Write tests that `canClaimHost` turns true for other players 1 minute after the host disconnects, using the fake clock, that a returning host cancels this, and that claiming early returns `host_available`, in `internal/game/game_test.go`
- [X] T052 Implement the disconnect timer and host claim in `internal/game/game.go`
- [X] T053 Add the claim handler in `internal/server/server.go`, and the take-over button in `web/game.html` and `web/app.js`

### Hardening and budgets

- [X] T054 [P] Write a test that total embedded page files are 100 KB or less in `web/embed_test.go`
- [X] T055 Add a test and implementation for security headers, including a Content Security Policy that allows only same-origin scripts and images, in `internal/server/server_test.go` and `internal/server/server.go`
- [X] T056 Handle `SIGTERM` by sending `closed` with reason `shutdown` and exiting cleanly in `cmd/server/main.go`

### Documentation and validation

- [X] T057 [P] Write install, run, and test commands, local hosting steps, the configuration table, and the cloud platform requirements and setup in `README.md`
- [X] T058 Run `go test ./...`, `go test -race ./...`, and `docker build --target test .`, and fix any failures, using `Dockerfile` as the Docker entry point
- [ ] T059 Run quickstart Scenarios 1 to 5 on real phones and record the results in `specs/001-fake-artist-game-master/quickstart.md`
- [X] T060 Measure image size and idle memory against the budgets in `specs/001-fake-artist-game-master/plan.md` and record them in `specs/001-fake-artist-game-master/quickstart.md`
- [ ] T061 Deploy to a cloud platform, run quickstart Scenario 6, and record the results in `specs/001-fake-artist-game-master/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies. Can start immediately.
- **Foundational (Phase 2)**: Depends on Setup. Blocks all user stories.
- **User Stories (Phases 3 to 6)**: Depend on Foundational, and on each other as shown below.
- **Polish (Phase 7)**: Depends on all user stories.

### User Story Dependencies

Players must be in a game before anything else can happen, so the stories are not fully independent.

- **User Story 1 (P1)**: Starts after Foundational. No story dependencies.
- **User Story 2 (P1)**: Depends on User Story 1, for joining and live updates.
- **User Story 3 (P2)**: Depends on User Story 2, because it reveals a dealt round.
- **User Story 4 (P3)**: Depends on User Story 1. It also tests dealing, so it is best done after User Story 2.

```text
Setup → Foundational → US1 → US2 → US3 ─┐
                               └─→ US4 ─┴→ Polish
```

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Game rules before views, views before handlers, handlers before browser files
- Story complete before moving to the next priority

### Parallel Opportunities

- T002, T003, and T004 can run together in Setup.
- T007 and T008 can run together, as can T016 with the Phase 2 tests.
- All test tasks within a story can run together.
- T028, T029, and T031 can run together.
- After User Story 2, Stories 3 and 4 can run in parallel.

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Acceptance tests in internal/server/acceptance_us1_test.go"
Task: "Session cookie tests in internal/server/session_test.go"
Task: "Event stream tests in internal/server/events_test.go"
Task: "Join address and QR tests in internal/server/qr_test.go"

# Launch the independent browser files together:
Task: "Home page in web/index.html"
Task: "Join and lobby screens in web/game.html"
Task: "Styling in web/style.css"
```

---

## Implementation Strategy

### MVP First (User Stories 1 and 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phases 3 and 4: User Stories 1 and 2
4. **STOP and VALIDATE**: Play a real round with quickstart Scenarios 1 and 2

Story 1 alone only gathers players, so the MVP needs both P1 stories to be playable.

### Incremental Delivery

1. Setup + Foundational → the server runs
2. Add User Story 1 → players can gather
3. Add User Story 2 → a group can play (MVP)
4. Add User Story 3 → groups can reveal and keep playing
5. Add User Story 4 → many groups can play at once
6. Polish → ready for hosting

---

## Notes

- [P] tasks use different files and have no dependencies
- The [Story] label maps each task to a user story for traceability
- Run `go test ./...` after each task
- Commit after each task or logical group
- Stop at any checkpoint to validate a story independently
- T059 needs real phones, and T061 needs a cloud account
