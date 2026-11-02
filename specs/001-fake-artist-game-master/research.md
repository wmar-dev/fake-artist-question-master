# Research: Fake Artist Game Master

All Technical Context items are resolved. Availability was checked on 2026-11-01: Go 1.27.3 is the current release, and `rsc.io/qr` v0.2.0 declares no dependencies.

## Language and runtime

- **Decision**: Go 1.27, standard library first.
- **Rationale**: The standard library covers HTTP routing, Server-Sent Events, cookies, secure random numbers, and file embedding. It compiles to one static binary, which suits a very small Docker image and low memory use.
- **Alternatives considered**: Node.js needs a larger runtime image and more packages for the same result. Python has the same image size problem, and handling hundreds of open connections takes more care.

## Live updates

- **Decision**: Server-Sent Events from server to browser, and ordinary HTTP POST and DELETE requests for actions.
- **Rationale**: All live traffic flows from server to phone. The browser's `EventSource` reconnects by itself after a phone wakes, which covers FR-020. The Go standard library needs no extra package for it.
- **Alternatives considered**: WebSockets need a dependency and offer two-way traffic the site does not use. Polling breaks the idle work rule in Principle II.

## QR codes

- **Decision**: Generate QR images on the server with `rsc.io/qr` v0.2.0, pinned in `go.mod`.
- **Justification under Principle III**: It provides QR encoding with error correction. The Go standard library has no QR support. It adds one small module with no transitive dependencies.
- **Alternatives considered**: Writing an encoder would take a few hundred lines of error-prone code and does not count as a small module. A third-party QR image service would break the rule against requiring hosted services and would not work on a closed local network. A browser library would add to page weight.

## Game storage

- **Decision**: Keep games in memory in one process, with up to 200 games by default, each removed after 2 hours without activity.
- **Rationale**: Games last minutes and hold nothing worth keeping. This needs no database, and it satisfies FR-022 and FR-023.
- **Consequences**: A restart ends games in progress, and the site cannot run on more than one instance at a time. Both are acceptable for this scale, since one small instance covers SC-005.
- **Alternatives considered**: An external store such as Redis adds a service to run and pay for, and solves a scaling need that the spec does not have.

## Timers instead of background sweeps

- **Decision**: Each game has one expiry timer, reset on activity, and a host handover timer that starts when the host disconnects. Both go through the injected `Clock`.
- **Rationale**: Nothing runs while the site is idle, which meets Principle II, and tests can advance time without sleeping.
- **Alternatives considered**: A periodic cleanup loop is simpler to write but does periodic work when idle.

## Player identity

- **Decision**: When a player joins, set a random 128-bit token in an `HttpOnly`, `SameSite=Lax` cookie, scoped to that game's path. Add `Secure` when `PUBLIC_URL` uses HTTPS.
- **Rationale**: This restores the same player on the same browser (FR-020) with no account. Scripts cannot read the cookie, and scoping it by game lets a browser hold more than one game.
- **Alternatives considered**: Tokens in `localStorage` are readable by scripts, and tokens in the URL would leak through shared links.

## Keeping secrets secret

- **Decision**: A single function builds each player's view from the round and the player. No other code sends game state to browsers.
- **Rationale**: FR-012 forbids sending the word to the fake artist's device, so hiding it in the browser would not be enough. One function can be tested directly against every role.

## Game codes

- **Decision**: 5 characters drawn with `crypto/rand` from an alphabet without look-alike characters (no `0`, `O`, `1`, `I`, or `L`).
- **Rationale**: Codes are quick to read aloud and type. With 31 characters there are about 28 million codes, and at most 200 are live at once, so guessing one is unlikely and it would reveal only a joinable lobby.

## Word order

- **Decision**: Shuffle the word list once per game and step through it, wrapping at the end.
- **Rationale**: This meets FR-017 and the spec's edge case directly: no repeats until the list is used up, and then the oldest words come back first.

## Join address and local hosting

- **Decision**: Use `PUBLIC_URL` when it is set. Otherwise build the join address from the request the host made. When there is no `PUBLIC_URL` and the host opened the site on `localhost` or a loopback address, show a message telling them to open it using the computer's network address (FR-026).
- **Rationale**: A process in a container cannot see the computer's network address, so automatic detection would give an address phones cannot reach. The host's own browser address is already correct on a local network.
- **Cloud note**: TLS usually ends at the platform's proxy, so the server cannot tell the original scheme from the connection. Cloud deployments must set `PUBLIC_URL`. The server does not trust forwarded headers, so a client cannot change the address in a QR code.

## Docker image

- **Decision**: A multi-stage `Dockerfile`. A `test` stage runs the test suite, a build stage compiles a static binary, and the final `scratch` image holds only that binary and runs as a non-root user.
- **Rationale**: The site makes no outbound calls, so it needs no shell, CA certificates, or OS packages. This keeps the image under budget and leaves little to attack. `docker build --target test .` also lets people run tests without installing Go.
- **Alternatives considered**: An Alpine runtime image is a few MB larger and adds a shell the site does not need.

## Cloud deployment

- **Decision**: Target any platform that runs a container, rather than one provider. The README lists what the platform must provide and how to set it on common ones.
- **Platform requirements**:
  - Listens on the port from the `PORT` variable, default 8080.
  - Exactly one instance, with scale-to-zero off, because games are in memory.
  - No buffering of streaming responses. The server sends `X-Accel-Buffering: no` and `Cache-Control: no-cache`.
  - Long-lived responses allowed. If the platform limits request length, `EventSource` reconnects and the session survives.
  - HTTPS provided by the platform.
- **Keep-alive**: A comment line every 25 seconds keeps proxies from closing idle connections. The constitution allows this as work a connection needs to stay alive.
- **Shutdown**: The server handles `SIGTERM` by closing event streams and exiting cleanly. Games end, as the spec assumes.

## Browser testing

- **Decision**: Automated tests stop at HTTP. `app.js` only renders the view it receives and sends actions, so game logic stays in tested Go code. `quickstart.md` covers real phones by hand.
- **Rationale**: A browser test tool would be a large development dependency for a small page, and QR scanning needs a real phone camera anyway.
