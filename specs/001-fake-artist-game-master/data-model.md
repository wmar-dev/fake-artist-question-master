# Data Model: Fake Artist Game Master

All data is held in memory and removed when a game expires.

## Game

| Field | Type | Rules |
|-------|------|-------|
| Code | string | 5 characters, unique among live games, case-insensitive on entry |
| HostID | player ID | Always a current player |
| Players | ordered list of Player | At most 10 (FR-007) |
| Deck | shuffled list of word indexes | Fixed when the game is created |
| NextWord | integer | Position in Deck; wraps at the end (FR-017) |
| Round | Round or none | The current round only; earlier rounds are not kept |
| LastActivity | time | Reset by joins, actions, and connections; expiry is 2 hours later (FR-022) |

**Status** comes from Round, and is not stored:

```text
Lobby ──deal──▶ Dealt ──reveal──▶ Revealed
                  ▲ │                 │
                  └─┘                 │
                  deal                │
                  ▲───────deal────────┘
```

| Status | Meaning | Host can |
|--------|---------|----------|
| Lobby | No round yet | Deal, remove players |
| Dealt | Round in progress | Reveal, deal (discards the round), remove players |
| Revealed | Answer shown | Deal, remove players |

Dealing requires 4 to 10 players. Removing players does not change the status.

## Player

| Field | Type | Rules |
|-------|------|-------|
| ID | string | Random; public within the game |
| Token | string | Random 128 bits; only ever stored in that player's cookie |
| Name | string | 1 to 20 characters after trimming, no control characters, unique within the game ignoring case (FR-005) |
| Connections | integer | Open event streams |
| DisconnectedAt | time or none | Set when Connections reaches 0 |

A player counts as connected when Connections is above 0. When the host has been disconnected for 1 minute, any other player can become host (FR-021).

## Round

| Field | Type | Rules |
|-------|------|-------|
| Category | string | From the Word Entry |
| Word | string | From the Word Entry |
| FakeID | player ID | Chosen uniformly from the participants (FR-010) |
| FakeName | string | Kept so that the reveal still works if the fake artist is removed |
| FirstName | string | Chosen uniformly and separately from FakeID (FR-010); kept for the same reason |
| Number | integer | Counts up with each deal |
| Participants | set of player IDs | Players present at the deal; later joiners wait (spec edge case) |
| Revealed | boolean | Set by the host (FR-015) |

## Word Entry

Loaded once at startup from `internal/game/words.txt`, one `category,word` per line. The list has at least 200 words across at least 10 categories (FR-009). A test enforces these counts and rejects duplicates.

## Player View

This is the only game data sent to a browser, and it is built per player. Its shape is defined in [contracts/http-api.md](contracts/http-api.md).

| Player's position | Sees category and first player | Sees word | Sees fake artist |
|-------------------|--------------------------------|-----------|------------------|
| Artist, before reveal | Yes | Yes | No |
| Fake artist, before reveal | Yes | No | Self only, as `isFake` |
| Waiting (joined after the deal), before reveal | No | No | No |
| Anyone in the game, after reveal | Yes | Yes | Yes |
| Not in the game | No | No | No |

The view leaves out `word`, `fakeName`, and `fakeId` entirely where the table says No, rather than sending them empty.

## Limits

| Limit | Value | Source |
|-------|-------|--------|
| Games | 200 by default, set by `MAX_GAMES` | Principle II |
| Players per game | 10 | FR-007 |
| Game expiry | 2 hours without activity | FR-022 |
| Host handover wait | 1 minute | FR-021 |
| Request body | 1 KB | Principle II |
