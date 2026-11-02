"use strict";

// This file only renders what the server sends and reports what players do.
// Game rules and secrets stay on the server.

const $ = (id) => document.getElementById(id);

const show = (element, visible) => {
  element.hidden = !visible;
};

// api returns the response body, and throws an Error with a message that is
// safe to show the player.
async function api(method, path, body) {
  let response;
  try {
    response = await fetch(path, {
      method,
      headers: body ? { "Content-Type": "application/json" } : {},
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new Error("We couldn't reach the game. Check your connection and try again.");
  }
  if (response.status === 204) {
    return null;
  }
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    const error = new Error(data?.message ?? "Something went wrong. Please try again.");
    error.code = data?.error;
    throw error;
  }
  return data;
}

function startHomePage() {
  const showError = (message) => {
    $("error").textContent = message;
    show($("error"), true);
  };

  $("create-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const { code } = await api("POST", "/api/games", { name: $("create-name").value });
      location.href = `/g/${code}`;
    } catch (error) {
      showError(error.message);
    }
  });

  $("find-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const code = $("find-code").value.trim().toUpperCase();
      await api("GET", `/api/games/${encodeURIComponent(code)}`);
      location.href = `/g/${code}`;
    } catch (error) {
      showError(error.message);
    }
  });
}

function startGamePage() {
  let code = location.pathname.split("/").pop().toUpperCase();
  let source = null;
  let roleRound = 0; // The round number the role was last hidden for.
  let roleVisible = false;
  let lastStatus = null;

  const screens = ["loading", "message-screen", "join-form", "game-screen"];
  const showScreen = (name) => screens.forEach((id) => show($(id), id === name));

  const notify = (message) => {
    $("notice").textContent = message;
    show($("notice"), Boolean(message));
  };

  const showMessage = (title, text) => {
    $("message-title").textContent = title;
    $("message-text").textContent = text;
    notify("");
    showScreen("message-screen");
  };

  const act = async (method, path) => {
    try {
      notify("");
      await api(method, `/api/games/${code}${path}`);
    } catch (error) {
      notify(error.message);
    }
  };

  function renderPlayers(view) {
    const list = $("players");
    list.replaceChildren();
    for (const player of view.players) {
      const item = document.createElement("li");
      const name = document.createElement("span");
      name.textContent = player.name;
      item.append(name);
      if (player.isHost) {
        const tag = document.createElement("span");
        tag.className = "tag";
        tag.textContent = "Host";
        item.append(tag);
      }
      if (!player.connected) {
        const tag = document.createElement("span");
        tag.className = "tag offline";
        tag.textContent = "Offline";
        item.append(tag);
      }
      if (view.you.isHost && !player.isHost) {
        const remove = document.createElement("button");
        remove.type = "button";
        remove.className = "link";
        remove.textContent = "Remove";
        remove.setAttribute("aria-label", `Remove ${player.name}`);
        remove.addEventListener("click", () => {
          if (confirm(`Remove ${player.name} from the game?`)) {
            act("DELETE", `/players/${player.id}`);
          }
        });
        item.append(remove);
      }
      list.append(item);
    }
    $("players-title").textContent = `Players (${view.players.length} of ${view.maxPlayers})`;
  }

  function renderRole(round) {
    if (round.number !== roleRound) {
      roleRound = round.number;
      roleVisible = false;
    }
    if (round.isFake) {
      $("role-label").textContent = "Your role";
      $("role-text").textContent = "You are the fake artist";
      $("role-hint").textContent = "You don't know the word. Blend in, and work it out from the drawing.";
    } else {
      $("role-label").textContent = "The word is";
      $("role-text").textContent = round.word;
      $("role-hint").textContent = "One player doesn't know it. Don't make it too obvious.";
    }
    show($("role"), roleVisible);
    $("role-toggle").textContent = roleVisible ? "Hide my role" : "Show my role";
  }

  function renderRound(view) {
    const round = view.round;
    const dealt = view.status === "dealt";
    show($("round-panel"), dealt);
    show($("reveal-panel"), view.status === "revealed");
    if (dealt) {
      show($("waiting-panel"), Boolean(round.waiting));
      show($("dealt-panel"), !round.waiting);
      if (!round.waiting) {
        $("category").textContent = round.category;
        $("drawing-rules").textContent = `${round.firstPlayer} draws first. Take turns clockwise, one mark each, for two laps. Then vote out loud.`;
        renderRole(round);
      }
    }
    if (view.status === "revealed") {
      $("reveal-fake").textContent = round.fakeName;
      $("reveal-word").textContent = round.word;
    }
  }

  function renderHost(view) {
    const missing = view.minPlayers - view.players.length;
    const dealt = view.status === "dealt";
    show($("host-panel"), view.you.isHost);
    show($("reveal-button"), dealt);
    $("deal-button").textContent = { lobby: "Deal first round", dealt: "Skip this word", revealed: "Deal next round" }[view.status];
    $("deal-button").className = dealt ? "secondary" : "";
    $("deal-button").disabled = missing > 0;
    $("deal-hint").textContent = missing > 0 ? `Add ${missing} more to deal a round.` : "";
    show($("deal-hint"), missing > 0);
    show($("claim-button"), view.canClaimHost);
  }

  function render(view) {
    showScreen("game-screen");
    notify("");
    $("game-code").textContent = view.code;
    $("invite-code").textContent = view.code;
    $("join-url").textContent = view.joinUrl;
    if (!$("qr").getAttribute("src")) {
      $("qr").src = `/api/games/${view.code}/qr.png`;
    }
    show($("address-warning"), view.you.isHost && !view.joinUrlReachable);
    if (view.status !== lastStatus) {
      $("invite").open = view.status === "lobby";
      lastStatus = view.status;
    }

    const missing = view.minPlayers - view.players.length;
    show($("lobby-panel"), view.status === "lobby");
    $("lobby-text").textContent = missing > 0
      ? `You need at least ${view.minPlayers} players to start. ${missing} more to go.`
      : view.you.isHost ? "Everyone in? Deal the first round." : "Waiting for the host to deal the first round.";

    renderRound(view);
    renderHost(view);
    renderPlayers(view);
  }

  const closedMessages = {
    removed: ["You've left this game", "The host removed you from the game."],
    expired: ["This game has ended", "It was closed after 2 hours without activity. Start a new game to keep playing."],
    shutdown: ["This game has ended", "The site restarted. Start a new game to keep playing."],
  };

  function connect() {
    source = new EventSource(`/api/games/${code}/events`);
    source.addEventListener("state", (event) => render(JSON.parse(event.data)));
    source.addEventListener("closed", (event) => {
      source.close();
      const [title, text] = closedMessages[JSON.parse(event.data).reason] ?? closedMessages.expired;
      showMessage(title, text);
    });
    source.onerror = () => {
      if (source.readyState === EventSource.CLOSED) {
        load(); // The server refused the stream, so find out why.
      } else {
        notify("Reconnecting…");
      }
    };
  }

  async function load() {
    try {
      const game = await api("GET", `/api/games/${code}`);
      code = game.code;
      $("game-code").textContent = code;
      if (game.joined) {
        showScreen("game-screen");
        connect();
      } else if (game.joinable) {
        showScreen("join-form");
      } else {
        showMessage("This game is full", "It already has 10 players. Ask the host to remove someone, or start a new game.");
      }
    } catch (error) {
      if (error.code === "game_not_found") {
        showMessage("Game not found", error.message);
      } else {
        showScreen("loading");
        notify(error.message);
      }
    }
  }

  $("join-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await api("POST", `/api/games/${code}/players`, { name: $("join-name").value });
      notify("");
      showScreen("game-screen");
      connect();
    } catch (error) {
      notify(error.message);
    }
  });

  $("role-toggle").addEventListener("click", () => {
    roleVisible = !roleVisible;
    show($("role"), roleVisible);
    $("role-toggle").textContent = roleVisible ? "Hide my role" : "Show my role";
  });
  $("deal-button").addEventListener("click", () => act("POST", "/rounds"));
  $("reveal-button").addEventListener("click", () => act("POST", "/reveal"));
  $("claim-button").addEventListener("click", () => act("POST", "/host"));

  load();
}

if ($("create-form")) {
  startHomePage();
} else {
  startGamePage();
}
