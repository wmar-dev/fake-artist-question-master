# Feature Specification: Fake Artist Question Master

**Feature Branch**: `001-fake-artist-game-master`

**Created**: 2026-11-01

**Status**: Draft

**Input**: User description: "Make a website for people to access A Fake Artist Goes to New York game master, so everyone can play together. qr code scanning to visit website, support multiple games."

## Clarifications

### Session 2026-11-01

- Q: Where do players draw? → A: On real paper in the same room. The site does not handle votes or scores.
- Q: Who picks the category and word? → A: The site always does, from a built-in word list.
- Q: What does "support multiple games" mean? → A: Separate groups can play at the same time, and each group can play repeated rounds.
- Q: How is the site deployed? → A: With Docker, both locally on a home or venue network and on a cloud platform.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start a game and let friends join by QR code (Priority: P1)

A host opens the website and creates a game. The site shows a QR code and a short game code. Friends scan the QR code with their phone camera, enter a display name, and appear in the game's player list on every screen.

**Why this priority**: Nobody can play until the group is gathered in one game, and scanning a code is the fastest way to get a room of people there.

**Independent Test**: Create a game on one device, join from several others by scanning the QR code and by typing the game code, and confirm that every device shows the same player list.

**Acceptance Scenarios**:

1. **Given** a visitor on the home page, **When** they create a game, **Then** they become the host and see a QR code, a game code, and the player list.
2. **Given** a host showing the QR code, **When** a friend scans it, **Then** the friend reaches that game's join page without typing an address.
3. **Given** a friend who cannot scan, **When** they enter the game code on the home page, **Then** they reach the same join page.
4. **Given** the join page, **When** a player enters a display name, **Then** they join the game and every other player sees them appear.

---

### User Story 2 - Receive a secret role privately (Priority: P1)

When the host deals a round, the site acts as the Question Master. It announces a category and the first player to draw to everyone, and privately shows each artist the secret word, except one randomly chosen player, who is told they are the fake artist. The group then draws on paper and votes aloud at the table.

**Why this priority**: Dealing hidden information is the core job of a Question Master, and it is what the group cannot do for itself when nobody wants to sit out.

**Independent Test**: Deal a round with at least the minimum number of players and confirm that exactly one player sees the fake artist message, all others see the same word, and all see the same category and first player.

**Acceptance Scenarios**:

1. **Given** a game with enough players, **When** the host deals a round, **Then** every player sees the category and the first player, and exactly one player is the fake artist.
2. **Given** a round in progress, **When** an artist views their role, **Then** they see the secret word and do not see who the fake artist is.
3. **Given** a round in progress, **When** the fake artist views their role, **Then** they see the category and do not see the secret word.
4. **Given** a player's phone showing their role, **When** they choose to hide it, **Then** people nearby cannot read it until they choose to show it again.
5. **Given** a round in progress, **When** a player looks at their screen, **Then** it reminds them that the group draws one mark each, clockwise, for two laps, and then votes.

---

### User Story 3 - Reveal the answer and deal again (Priority: P2)

When the group has voted at the table, the host reveals the round. Every screen shows who the fake artist was and what the word was, so the group can settle the result. The host then deals another round for the same group.

**Why this priority**: It settles arguments and lets a group keep playing, but the group can finish a single round at the table without it.

**Independent Test**: Reveal a round and confirm that every screen shows the same fake artist and word. Then deal again and confirm that a new word and a new random fake artist are chosen.

**Acceptance Scenarios**:

1. **Given** a round in progress, **When** the host reveals it, **Then** every player sees the fake artist and the word.
2. **Given** a revealed round, **When** the host deals again, **Then** the same players receive a new round.
3. **Given** a game with several rounds played, **When** a new round is dealt, **Then** its word has not been used earlier in the game.

---

### User Story 4 - Run many games at once (Priority: P3)

Separate groups use the site at the same time, and nothing from one game is visible in another.

**Why this priority**: It makes the site usable at parties and events, but a single group can play without it.

**Independent Test**: Run two games at the same time on separate devices and confirm that players, rounds, and words stay separate.

**Acceptance Scenarios**:

1. **Given** two games in progress, **When** either one changes, **Then** the other is unaffected.
2. **Given** a player in one game, **When** they use the site, **Then** they cannot see anything from another game.

---

### Edge Cases

- What happens when a player refreshes, locks their phone, or loses connection? They return to the same game and see the same role.
- What happens when a player leaves during a round? The host can remove them. If the group can no longer play that round, the host deals a new one.
- What happens when the host disconnects? The game continues, and any other player can take over as host if the host stays away for 1 minute.
- What happens when someone joins after a round has started? They wait and receive a role in the next round.
- What happens when a game is full or the code is wrong or expired? The player sees a plain message that says what to do next.
- What happens when two players choose the same name? The second is asked to choose a different name.
- What happens when all words in the word list have been used in a game? Words may repeat, starting with those used longest ago.
- What happens when the group wants to skip a word? The host can deal a new round at any time, which discards the current one.

## Requirements *(mandatory)*

### Functional Requirements

#### Joining

- **FR-001**: Users MUST be able to create a game from the home page without an account.
- **FR-002**: The system MUST show a scannable QR code and a short game code for each game before its first round, and on request afterwards.
- **FR-003**: Scanning the QR code with a standard phone camera MUST open that game's join page.
- **FR-004**: Users MUST be able to join by entering the game code.
- **FR-005**: The system MUST require a display name of 1 to 20 characters that is unique within the game.
- **FR-006**: The system MUST show the live player list to everyone in the game.
- **FR-007**: The system MUST allow a round to be dealt only with 4 to 10 players, and MUST refuse new players once a game has 10.
- **FR-008**: The host MUST be able to remove a player.

#### Dealing a round

- **FR-009**: The system MUST choose each round's category and word from a built-in list of at least 200 easy-to-draw words across at least 10 categories.
- **FR-010**: The system MUST choose the fake artist and the first player at random for each round, independently of each other and with every player equally likely.
- **FR-011**: The system MUST show the category and first player to everyone in the round.
- **FR-012**: The system MUST show the secret word to artists only, and MUST NOT send the word to the fake artist's device or reveal the fake artist's identity to any player before the host reveals the round.
- **FR-013**: Players MUST be able to hide and show their role on their own screen.
- **FR-014**: The screen MUST show the drawing rules during a round: one mark per turn, clockwise from the first player, for two laps, followed by a vote at the table.

#### Revealing and repeating

- **FR-015**: The host MUST be able to reveal a round, which MUST show the fake artist and the word to every player in it.
- **FR-016**: The host MUST be able to deal a new round at any time.
- **FR-017**: The system MUST avoid repeating a word within a game until the word list is used up.
- **FR-018**: The system MUST NOT collect votes, track scores, or provide a drawing surface.

#### Multiple games

- **FR-019**: The system MUST run multiple games at the same time, with players, rounds, and words kept separate.

#### Reliability and data

- **FR-020**: A player who reconnects from the same device and browser MUST return to the same game and role.
- **FR-021**: If the host has been disconnected for 1 minute, any other player MUST be able to become host.
- **FR-022**: The system MUST remove a game and its data after 2 hours without activity.
- **FR-023**: The system MUST NOT collect personal information beyond the display name.
- **FR-024**: Error messages MUST use plain language and state what the player can do next.
- **FR-025**: The same Docker image MUST run both locally and on a cloud platform, with no code changes between them.
- **FR-026**: If a game's join address cannot be reached from a phone, the host MUST see a plain message explaining how to fix it.

### Key Entities

- **Game**: One group's session. Has a game code, a host, players, the words already used, and the current round.
- **Player**: A participant in one game. Has a display name and a connection status.
- **Round**: One deal of the game. Has a category, a secret word, a fake artist, a first player, and whether it has been revealed.
- **Word Entry**: A category and word pair that a round can use.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new player can go from scanning the QR code to appearing in the game in under 30 seconds.
- **SC-002**: A group of 10 can go from creating a game to seeing their roles in under 2 minutes.
- **SC-003**: After any action, every player in the game sees the update within 2 seconds.
- **SC-004**: At least 90% of first-time players see their role and know what to do next without outside help.
- **SC-005**: The site supports 50 games of 10 players at the same time without noticeable slowdown.
- **SC-006**: In 100% of test rounds, no player can find another player's role or, as the fake artist, the word before the reveal.

## Assumptions

- Players are in the same room, share paper and pens, and use a phone each. Remote play is out of scope.
- The group follows the published rules at the table for voting, the fake artist's guess, and scoring, and keeps any score itself.
- The site always takes the Question Master's role, so everyone draws. The published game is for 5 to 10 players including the Question Master, who does not draw, so a game here supports 4 to 10.
- Because the site does not follow play at the table, it shows the first player and the drawing direction instead of tracking each turn live.
- Each player uses a current phone browser, and at least one other device can scan the code from the host's screen.
- The interface and word list are English only for this feature.
- The word list is written for this project rather than copied from the published game, and the published game's artwork is not used.
- The site is deployed with Docker, either locally or in the cloud, at an address that players' phones can reach. QR codes must use that address.
- Games exist only while the site is running, so a restart ends any games in progress.
- Accounts, saved history, and public game listings are out of scope.
