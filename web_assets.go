package main

const webHTML = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Twitch Multi Tool</title>
  <style>
    :root {
      --bg: #f7f7f8;
      --panel: rgba(255,255,255,.9);
      --panel-soft: rgba(248,248,250,.92);
      --line: rgba(207,210,218,.9);
      --text: #18181b;
      --muted: #6f7280;
      --accent: #9147ff;
      --accent-strong: #772ce8;
      --accent-soft: rgba(145,71,255,.16);
      --danger: #e54b4b;
      --button-bg: rgba(255,255,255,.84);
      --button-hover: #efeff1;
      --input-bg: rgba(255,255,255,.94);
      --player-bg: #efeff1;
      --composer-bg: #f3f3f5;
      --shadow: 0 20px 48px rgba(24,24,27,.08);
    }
    body.theme-dark {
      --bg: #0e0e10;
      --panel: rgba(31,31,35,.92);
      --panel-soft: rgba(24,24,27,.94);
      --line: rgba(83,83,95,.55);
      --text: #efeff1;
      --muted: #adadb8;
      --accent: #9147ff;
      --accent-strong: #772ce8;
      --accent-soft: rgba(145,71,255,.24);
      --danger: #ff7070;
      --button-bg: rgba(46,46,54,.92);
      --button-hover: #3a3a44;
      --input-bg: rgba(24,24,27,.96);
      --player-bg: #0a0a0d;
      --composer-bg: #111114;
      --shadow: 0 28px 60px rgba(0,0,0,.34);
    }
    * { box-sizing: border-box; }
    body { margin: 0; overflow: hidden; background: radial-gradient(circle at top, rgba(145,71,255,.16), transparent 26%), radial-gradient(circle at right, rgba(119,44,232,.14), transparent 18%), var(--bg); color: var(--text); font: 14px/1.45 "Avenir Next", "Segoe UI", "Helvetica Neue", sans-serif; }
    button, input, select { font: inherit; }
    button { border: 1px solid var(--line); border-radius: 8px; background: var(--button-bg); color: var(--text); padding: 8px 10px; cursor: pointer; transition: background .16s ease, border-color .16s ease, transform .16s ease; }
    button:hover { background: var(--button-hover); }
    button:hover, .page-tab:hover, .tab:hover, .account:hover { transform: translateY(-1px); }
    button.primary { border-color: var(--accent); background: var(--accent); color: #fff; font-weight: 650; box-shadow: 0 0 0 1px rgba(255,255,255,.04) inset; }
    button.primary:hover { background: var(--accent-strong); }
    input, select { width: 100%; border: 1px solid var(--line); border-radius: 8px; background: var(--input-bg); color: var(--text); padding: 9px 10px; outline: none; }
    input:focus, select:focus { border-color: var(--accent); box-shadow: 0 0 0 3px rgba(145,71,255,.18); }
    label { color: var(--muted); font-size: 12px; font-weight: 650; }
    .app { height: 100vh; display: grid; grid-template-rows: 58px 1fr; }
    .topbar { display: grid; grid-template-columns: 230px auto minmax(260px, 460px) 1fr auto; gap: 14px; align-items: center; padding: 0 16px; border-bottom: 1px solid var(--line); background: linear-gradient(180deg, rgba(145,71,255,.08), transparent 100%), var(--panel); box-shadow: var(--shadow); backdrop-filter: blur(14px); }
    .brand { display: flex; align-items: center; gap: 10px; font-weight: 750; }
    .mark { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 8px; background: linear-gradient(180deg, #a970ff, #772ce8); color: white; box-shadow: 0 12px 24px rgba(145,71,255,.34); }
    .page-tabs { display: flex; gap: 6px; }
    .page-tab.active { color: #fff; background: var(--accent); border-color: var(--accent); }
    .channel-form { display: grid; grid-template-columns: 1fr auto; gap: 8px; }
    .status { color: var(--muted); text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .status.error { color: var(--danger); }
    .layout { min-height: 0; display: grid; grid-template-columns: minmax(300px, 380px) minmax(520px, 1fr) minmax(330px, 360px); }
    aside, main, .right, .chat-panel { min-height: 0; }
    aside { overflow: hidden; border-left: 1px solid var(--line); background: var(--panel-soft); display: grid; grid-template-rows: auto minmax(120px, 1fr) auto auto; box-shadow: inset 0 1px 0 rgba(255,255,255,.03); }
    .chat-panel { border-right: 1px solid var(--line); background: var(--panel-soft); display: grid; grid-template-rows: auto 1fr; box-shadow: inset 0 1px 0 rgba(255,255,255,.03); }
    .pane-head { min-height: 48px; display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 12px; border-bottom: 1px solid var(--line); font-weight: 650; }
    .account-list { overflow: auto; padding: 10px; }
    .account { width: 100%; display: grid; gap: 4px; text-align: left; border: 1px solid transparent; background: transparent; padding: 10px; border-radius: 6px; }
    .account:hover { border-color: rgba(145,71,255,.42); background: rgba(145,71,255,.08); }
    .account.active { border-color: rgba(145,71,255,.72); background: var(--accent-soft); box-shadow: 0 0 0 1px rgba(145,71,255,.18) inset; }
    .account-row { display: flex; justify-content: space-between; gap: 8px; }
    .account-name { font-weight: 650; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .meta { color: var(--muted); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .tag { flex: 0 0 auto; padding: 2px 6px; border-radius: 999px; background: rgba(145,71,255,.14); color: var(--muted); font-size: 11px; font-weight: 650; }
    .tag.proxy { color: #fff; background: rgba(145,71,255,.58); }
    .editor { border-top: 1px solid var(--line); padding: 12px; display: grid; gap: 8px; background: var(--panel); }
    .account-add { border-top: 1px solid var(--line); background: var(--panel); }
    .account-add summary { list-style: none; padding: 12px; cursor: pointer; color: var(--text); font-weight: 650; }
    .account-add summary::-webkit-details-marker { display: none; }
    .account-add summary::after { content: "+"; float: right; color: var(--muted); }
    .account-add[open] summary::after { content: "-"; }
    .account-add form { border-top: 1px solid var(--line); }
    .field { display: grid; gap: 6px; }
    .button-row { display: flex; gap: 8px; }
    main { min-width: 0; display: grid; grid-template-rows: 1fr auto; background: var(--player-bg); }
    .player-wrap { min-height: 0; display: grid; place-items: center; padding: 14px; }
    .player-frame { width: 100%; max-width: 1280px; aspect-ratio: 16 / 9; background: #050608; border: 1px solid rgba(145,71,255,.24); border-radius: 12px; overflow: hidden; box-shadow: 0 30px 80px rgba(0,0,0,.42); }
    .player-frame iframe { width: 100%; height: 100%; border: 0; display: block; }
    .compose-wrap { border-top: 1px solid rgba(145,71,255,.2); background: linear-gradient(180deg, rgba(145,71,255,.06), transparent 40%), var(--composer-bg); }
    .quick-presets { display: none; gap: 8px; overflow-x: auto; padding: 10px 12px 0; }
    .quick-presets.active { display: flex; }
    .preset-chip { flex: 0 0 auto; max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; border-color: rgba(145,71,255,.28); background: rgba(145,71,255,.16); color: var(--text); }
    .preset-chip:hover { background: rgba(145,71,255,.24); }
    .reply-bar { display: none; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 12px 0; color: #c9d2e3; }
    .reply-bar.active { display: flex; }
    .reply-target { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .composer { padding: 12px; display: grid; grid-template-columns: minmax(130px, 160px) minmax(0, 1fr) 42px 118px; gap: 10px; align-items: center; position: relative; }
    .send-account { min-width: 0; height: 42px; font-weight: 650; }
    .emoji-button { width: 42px; padding: 8px 0; font-size: 18px; }
    .emoji-picker { display: none; position: absolute; right: 104px; bottom: 58px; z-index: 5; width: min(360px, calc(100vw - 32px)); max-height: 220px; overflow: auto; padding: 8px; border: 1px solid var(--line); border-radius: 6px; background: var(--panel); box-shadow: 0 14px 36px rgba(0,0,0,.24); grid-template-columns: repeat(8, 1fr); gap: 6px; }
    .emoji-picker.active { display: grid; }
    .emoji-choice { min-width: 0; padding: 6px 0; font-size: 20px; line-height: 1; }
    .right { min-width: 0; overflow: hidden; background: var(--panel-soft); display: grid; grid-template-rows: minmax(260px, 1fr) minmax(220px, 32vh); min-height: 0; box-shadow: inset 1px 0 0 rgba(255,255,255,.02); }
    .right.tools-hidden { grid-template-rows: minmax(0, 1fr) auto; }
    .tools { min-height: 0; display: grid; grid-template-rows: auto 1fr; border-top: 1px solid var(--line); }
    .tools.collapsed { grid-template-rows: auto; }
    .tools.collapsed .tool-body { display: none; }
    .chat { min-height: 0; overflow: auto; padding: 12px; display: flex; flex-direction: column; gap: 9px; background: linear-gradient(180deg, rgba(145,71,255,.04), transparent 140px), var(--panel); }
    .message { display: grid; grid-template-columns: 58px minmax(0, 1fr); gap: 8px; align-items: start; }
    .message.out { color: #a970ff; }
    .message.system { grid-template-columns: 1fr; color: var(--muted); font-size: 13px; border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); padding: 6px 0; }
    .time { color: var(--muted); font-size: 12px; font-variant-numeric: tabular-nums; }
    .sender { font-weight: 700; margin-right: 6px; }
    .text { overflow-wrap: anywhere; }
    .reply-button { margin-left: 8px; padding: 2px 6px; font-size: 12px; color: var(--muted); border-color: transparent; background: transparent; }
    .reply-button:hover { border-color: rgba(145,71,255,.36); background: rgba(145,71,255,.12); }
    .tabs { min-width: 0; display: flex; flex-wrap: wrap; gap: 6px; }
    .tab.active { color: white; background: var(--accent); border-color: var(--accent); }
    .tools-actions { flex: 0 0 auto; margin-left: auto; }
    .tool-toggle { min-width: 82px; }
    .tool-body { overflow: auto; padding: 10px; display: grid; gap: 8px; align-content: start; border-top: 1px solid var(--line); }
    .item { border: 1px solid var(--line); border-radius: 8px; background: linear-gradient(180deg, rgba(145,71,255,.05), transparent), var(--panel); padding: 9px; display: grid; gap: 3px; }
    .item-title { font-weight: 650; overflow-wrap: anywhere; }
    .item-actions { display: flex; gap: 6px; margin-top: 6px; }
    .preset-form { display: grid; gap: 8px; padding: 10px; border: 1px solid var(--line); border-radius: 6px; background: var(--panel); }
    .check-list { display: grid; gap: 6px; }
    .check-row { display: flex; align-items: center; gap: 8px; color: var(--text); }
    .check-row input { width: auto; }
    .range-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
    .range-presets { display: flex; flex-wrap: wrap; gap: 6px; }
    .auto-status { display: grid; gap: 4px; }
    .auto-page { display: none; min-height: 0; overflow: auto; background: radial-gradient(circle at top left, rgba(145,71,255,.18), transparent 22%), var(--bg); padding: 18px; }
    .app.page-auto .layout { display: none; }
    .app.page-auto .auto-page { display: block; }
    .auto-shell { max-width: 980px; margin: 0 auto; display: grid; gap: 12px; }
    .auto-title { margin: 0; font-size: 22px; line-height: 1.2; }
    .auto-body { display: grid; gap: 10px; }
    .error { color: var(--danger); }
    @media (max-width: 1100px) {
      .layout { grid-template-columns: 260px 1fr; }
      .right { grid-column: 1 / -1; grid-template-columns: 1fr 320px; grid-template-rows: 360px; border-top: 1px solid var(--line); }
    }
    @media (max-width: 760px) {
      .app { height: auto; min-height: 100vh; }
      body { overflow: auto; }
      .topbar { grid-template-columns: 1fr; height: auto; padding: 10px; }
      .page-tabs { overflow-x: auto; }
      .status { text-align: left; }
      .layout { grid-template-columns: 1fr; }
      aside { border-left: 0; border-bottom: 1px solid var(--line); }
      .account-list { max-height: 190px; }
      .right { grid-template-columns: 1fr; grid-template-rows: auto 320px; }
      .composer { grid-template-columns: 1fr auto auto; }
      .send-account { grid-column: 1 / -1; }
    }
  </style>
</head>
<body>
  <div class="app">
    <header class="topbar">
      <div class="brand"><div class="mark">T</div><div>Twitch Multi Tool</div></div>
      <nav class="page-tabs">
        <button class="page-tab active" id="chatPageTab" type="button">Чат</button>
        <button class="page-tab" id="autoPageTab" type="button">Auto-sender</button>
      </nav>
      <form class="channel-form" id="channelForm">
        <input id="channelInput" placeholder="channel">
        <button class="primary" type="submit">Подключить</button>
      </form>
      <div class="status" id="status">connecting</div>
      <button id="themeToggle" type="button">Тёмная</button>
    </header>
    <div class="layout">
      <section class="chat-panel">
        <div class="pane-head"><span id="chatTitle">Чат</span><button id="clearChat">Очистить</button></div>
        <div class="chat" id="chat"></div>
      </section>
      <main>
        <div class="player-wrap">
          <div class="player-frame"><iframe id="player" allow="autoplay; fullscreen; picture-in-picture" allowfullscreen></iframe></div>
        </div>
        <div class="compose-wrap">
          <div class="quick-presets" id="quickPresets"></div>
          <div class="reply-bar" id="replyBar">
            <div class="reply-target" id="replyTarget"></div>
            <button id="cancelReply" type="button">×</button>
          </div>
          <form class="composer" id="sendForm">
            <select class="send-account" id="sendAccountSelect" title="Аккаунт отправки"></select>
            <input id="messageInput" autocomplete="off" placeholder="Сообщение или emote code">
            <button class="emoji-button" id="emojiButton" type="button" title="Emoji">☺</button>
            <div class="emoji-picker" id="emojiPicker"></div>
            <button class="primary" type="submit">Отправить</button>
          </form>
        </div>
      </main>
      <section class="right">
        <aside>
          <div class="pane-head">
            <span>Аккаунты</span>
            <button id="refreshState" title="Обновить">↻</button>
          </div>
          <div class="account-list" id="accounts"></div>
          <div class="editor">
            <div class="field">
              <label for="proxyInput">SOCKS5 proxy</label>
              <input id="proxyInput" placeholder="socks5://user:pass@127.0.0.1:1080">
            </div>
            <div class="button-row">
              <button class="primary" id="saveProxy">Сохранить</button>
              <button id="clearProxy">Очистить</button>
            </div>
          </div>
          <details class="account-add">
            <summary>Добавить аккаунт</summary>
            <form class="editor" id="addAccountForm">
              <div class="field"><label for="newName">Название</label><input id="newName" placeholder="bot3"></div>
              <div class="field"><label for="newLogin">Login</label><input id="newLogin" placeholder="twitch_login"></div>
              <div class="field"><label for="newToken">OAuth token</label><input id="newToken" type="password" placeholder="oauth token"></div>
              <div class="field"><label for="newProxy">SOCKS5 proxy</label><input id="newProxy" placeholder="socks5://127.0.0.1:1080"></div>
              <button class="primary" type="submit">Добавить</button>
            </form>
          </details>
        </aside>
        <div class="tools">
          <div class="pane-head">
            <div class="tabs">
              <button class="tab active" data-tab="emotes">Эмоции</button>
              <button class="tab" data-tab="rewards">Награды</button>
              <button class="tab" data-tab="presets">Заготовки</button>
            </div>
            <div class="button-row tools-actions">
              <button id="reloadTools" title="Обновить">↻</button>
              <button class="tool-toggle" id="toggleTools" title="Скрыть">Скрыть</button>
            </div>
          </div>
          <div class="tool-body" id="toolBody"></div>
        </div>
      </section>
    </div>
    <section class="auto-page" id="autoPage">
      <div class="auto-shell">
        <div>
          <h1 class="auto-title">Auto-sender</h1>
          <div class="meta">Сообщения из auto-sender.txt уходят выбранными аккаунтами по кругу в текущий канал.</div>
        </div>
        <div class="auto-body" id="autoBody"></div>
      </div>
    </section>
  </div>
  <script>
    const state = { accounts: [], presets: [], autoSender: {}, current: "", selected: "", channel: "", tab: "emotes", page: "chat", replyTo: null, seen: new Set() };
    const chat = document.getElementById("chat");
    const accountsEl = document.getElementById("accounts");
    const statusEl = document.getElementById("status");
    const proxyInput = document.getElementById("proxyInput");
    const toolBody = document.getElementById("toolBody");
    const autoBody = document.getElementById("autoBody");
    const channelInput = document.getElementById("channelInput");
    const player = document.getElementById("player");
    const chatTitle = document.getElementById("chatTitle");
    const replyBar = document.getElementById("replyBar");
    const replyTarget = document.getElementById("replyTarget");
    const quickPresets = document.getElementById("quickPresets");
    const emojiPicker = document.getElementById("emojiPicker");
    const emojiButton = document.getElementById("emojiButton");
    const messageInput = document.getElementById("messageInput");
    const themeToggle = document.getElementById("themeToggle");
    const sendAccountSelect = document.getElementById("sendAccountSelect");
    const toolsPanel = document.querySelector(".tools");
    const rightPanel = document.querySelector(".right");
    const toggleTools = document.getElementById("toggleTools");
    const chatPageTab = document.getElementById("chatPageTab");
    const autoPageTab = document.getElementById("autoPageTab");
    const emojis = ["😀","😄","😂","🤣","😊","😍","😘","😎","🤔","😢","😭","😡","😱","🥳","🤯","😴","👍","👎","👏","🙌","🙏","💪","🔥","❤️","💜","💙","💚","💛","✨","⭐","⚡","💯","✅","❌","🎮","🏆","🎉","👀","💀","🤝"];

    function setStatus(text, isError) {
      statusEl.textContent = text;
      statusEl.className = isError ? "status error" : "status";
    }

    function setPage(page) {
      state.page = page;
      document.querySelector(".app").classList.toggle("page-auto", page === "auto");
      chatPageTab.classList.toggle("active", page === "chat");
      autoPageTab.classList.toggle("active", page === "auto");
      if (page === "auto") renderAutoSender(autoBody);
    }

    function applyTheme(theme) {
      document.body.classList.toggle("theme-dark", theme === "dark");
      themeToggle.textContent = theme === "dark" ? "Светлая" : "Тёмная";
      localStorage.setItem("tw-theme", theme);
    }

    async function api(path, options = {}) {
      const res = await fetch(path, { ...options, headers: { "Content-Type": "application/json", ...(options.headers || {}) } });
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    }

    function playerURL(channel) {
      const parent = encodeURIComponent(location.hostname || "127.0.0.1");
      return "https://player.twitch.tv/?channel=" + encodeURIComponent(channel || "") + "&parent=" + parent + "&muted=false&autoplay=true";
    }

    function applyState(next) {
      state.accounts = next.accounts || [];
      state.presets = next.presets || [];
      state.autoSender = next.auto_sender || state.autoSender || {};
      state.current = next.current || "";
      state.channel = next.channel || "";
      if (!state.accounts.some(acc => acc.name === state.selected)) state.selected = state.current;
      channelInput.value = state.channel;
      chatTitle.textContent = "Чат #" + state.channel;
      if (state.channel) player.src = playerURL(state.channel);
      renderAccounts();
      renderSendAccountSelect();
      renderQuickPresets();
    }

    function renderAccounts() {
      accountsEl.innerHTML = "";
      state.accounts.forEach(acc => {
        const btn = document.createElement("button");
        const isSelected = acc.name === (state.selected || state.current);
        btn.className = "account" + (isSelected ? " active" : "");
        btn.innerHTML = '<div class="account-row"><span class="account-name"></span><span class="tag"></span></div><div class="meta"></div>';
        btn.querySelector(".account-name").textContent = acc.name;
        const tag = btn.querySelector(".tag");
        tag.textContent = acc.has_proxy ? "SOCKS5" : "direct";
        if (acc.has_proxy) tag.classList.add("proxy");
        btn.querySelector(".meta").textContent = acc.login || "not validated";
        btn.onclick = async () => {
          state.selected = acc.name;
          proxyInput.value = acc.socks5_proxy || "";
          renderAccounts();
          try {
            setStatus("switching account");
            applyState(await api("/api/account", { method: "POST", body: JSON.stringify({ name: acc.name }) }));
            await loadTools();
            setStatus("ready");
          } catch (err) { setStatus(err.message.trim(), true); }
        };
        accountsEl.appendChild(btn);
      });
      const selected = state.accounts.find(acc => acc.name === state.selected) || state.accounts.find(acc => acc.active);
      proxyInput.value = selected ? (selected.socks5_proxy || "") : "";
    }

    function renderSendAccountSelect() {
      sendAccountSelect.innerHTML = "";
      state.accounts.forEach(acc => {
        const option = document.createElement("option");
        option.value = acc.name;
        option.textContent = acc.name;
        option.selected = acc.name === (state.selected || state.current);
        sendAccountSelect.appendChild(option);
      });
      sendAccountSelect.disabled = state.accounts.length === 0;
    }

    function addMessage(msg) {
      const key = msg.id || [msg.time, msg.login, msg.text].join("|");
      if (state.seen.has(key)) return;
      state.seen.add(key);
      const row = document.createElement("div");
      if (msg.id) row.dataset.id = msg.id;
      row.className = "message" + (msg.outgoing ? " out" : "") + (msg.system ? " system" : "");
      if (msg.system) {
        row.innerHTML = '<div class="text"></div>';
        row.querySelector(".text").textContent = msg.text;
      } else {
        row.innerHTML = '<div class="time"></div><div class="text"><span class="sender"></span><span class="body"></span><button class="reply-button" type="button">Ответить</button></div>';
        row.querySelector(".time").textContent = msg.time || "";
        row.querySelector(".sender").textContent = (msg.display_name || msg.login || "chat") + ":";
        row.querySelector(".body").textContent = " " + (msg.text || "");
        row.querySelector(".reply-button").onclick = () => setReplyTarget(msg);
      }
      chat.appendChild(row);
      while (chat.children.length > 300) chat.removeChild(chat.firstChild);
      chat.scrollTop = chat.scrollHeight;
    }

    function setReplyTarget(msg) {
      if (!msg.id) {
        setStatus("у сообщения нет id для ответа", true);
        return;
      }
      state.replyTo = { id: msg.id, login: msg.display_name || msg.login || "chat", text: msg.text || "" };
      replyTarget.textContent = "Ответ: " + state.replyTo.login + " · " + state.replyTo.text;
      replyBar.classList.add("active");
      document.getElementById("messageInput").focus();
    }

    function clearReplyTarget() {
      state.replyTo = null;
      replyTarget.textContent = "";
      replyBar.classList.remove("active");
    }

    function renderQuickPresets() {
      quickPresets.innerHTML = "";
      if (!state.presets.length) {
        quickPresets.classList.remove("active");
        return;
      }
      quickPresets.classList.add("active");
      state.presets.forEach(preset => {
        const btn = document.createElement("button");
        btn.className = "preset-chip";
        btn.type = "button";
        btn.title = preset.message;
        btn.textContent = preset.title || preset.message;
        btn.onclick = async () => {
          try {
            await sendMessage(preset.message);
            setStatus("preset sent");
          } catch (err) { setStatus(err.message.trim(), true); }
        };
        quickPresets.appendChild(btn);
      });
    }

    function addToolEmpty(text) {
      const el = document.createElement("div");
      el.className = "item meta";
      el.textContent = text;
      toolBody.appendChild(el);
    }

    async function loadTools() {
      toolBody.innerHTML = "";
      try {
        if (state.tab === "emotes") {
          const emotes = await api("/api/emotes?kind=channel");
          if (!emotes.length) addToolEmpty("Эмоций для этого канала не найдено.");
          emotes.forEach(emote => {
            const el = document.createElement("div");
            el.className = "item";
            el.innerHTML = '<div class="item-title"></div><div class="meta"></div>';
            el.querySelector(".item-title").textContent = emote.name;
            el.querySelector(".meta").textContent = [emote.emote_type, emote.id].filter(Boolean).join(" · ");
            el.onclick = () => {
              const input = document.getElementById("messageInput");
              input.value = input.value ? input.value + " " + emote.name : emote.name;
              input.focus();
            };
            toolBody.appendChild(el);
          });
        } else if (state.tab === "rewards") {
          const rewards = await api("/api/rewards");
          if (!rewards.length) addToolEmpty("Наград канала не найдено или они недоступны для этого аккаунта.");
          rewards.forEach(reward => {
            const stateText = !reward.is_enabled ? "disabled" : (reward.is_paused ? "paused" : "enabled");
            const el = document.createElement("div");
            el.className = "item";
            el.innerHTML = '<div class="item-title"></div><div class="meta"></div>';
            el.querySelector(".item-title").textContent = reward.title;
            el.querySelector(".meta").textContent = reward.cost + " points · " + stateText;
            toolBody.appendChild(el);
          });
        } else {
          renderPresets();
        }
      } catch (err) {
        const el = document.createElement("div");
        el.className = "item error";
        el.textContent = err.message.trim();
        toolBody.appendChild(el);
      }
    }

    function renderPresets() {
      const form = document.createElement("form");
      form.className = "preset-form";
      form.innerHTML = '<div class="field"><label>Название</label><input id="presetTitle" placeholder="Привет"></div><div class="field"><label>Текст</label><input id="presetMessage" placeholder="Hello chat!"></div><button class="primary" type="submit">Добавить заготовку</button>';
      form.onsubmit = async (event) => {
        event.preventDefault();
        const title = form.querySelector("#presetTitle").value.trim();
        const message = form.querySelector("#presetMessage").value.trim();
        if (!message) return;
        try {
          applyState(await api("/api/presets", { method: "POST", body: JSON.stringify({ title, message }) }));
          await loadTools();
          setStatus("preset saved");
        } catch (err) { setStatus(err.message.trim(), true); }
      };
      toolBody.appendChild(form);

      state.presets.forEach(preset => {
        const el = document.createElement("div");
        el.className = "item";
        el.innerHTML = '<div class="item-title"></div><div class="meta"></div><div class="item-actions"><button class="primary" type="button">Отправить</button><button type="button">Удалить</button></div>';
        el.querySelector(".item-title").textContent = preset.title || preset.message;
        el.querySelector(".meta").textContent = preset.message;
        const buttons = el.querySelectorAll("button");
        buttons[0].onclick = async () => {
          try {
            await sendMessage(preset.message);
            setStatus("preset sent");
          } catch (err) { setStatus(err.message.trim(), true); }
        };
        buttons[1].onclick = async () => {
          try {
            applyState(await api("/api/presets?id=" + encodeURIComponent(preset.id), { method: "DELETE" }));
            await loadTools();
            setStatus("preset deleted");
          } catch (err) { setStatus(err.message.trim(), true); }
        };
        toolBody.appendChild(el);
      });
    }

    async function renderAutoSender(target = autoBody) {
      target.innerHTML = "";
      try {
        state.autoSender = await api("/api/auto-sender");
      } catch (err) {
        state.autoSender = { error: err.message.trim() };
      }
      const auto = state.autoSender || {};
      const selected = new Set((auto.accounts && auto.accounts.length ? auto.accounts : [state.current]).filter(Boolean));
      const form = document.createElement("form");
      form.className = "preset-form";
      form.innerHTML = '<div class="auto-status"><div class="item-title"></div><div class="meta"></div><div class="meta last"></div><div class="error auto-error"></div></div><div class="field"><label>Аккаунты для отправки</label><div class="check-list"></div></div><div class="field"><label>Диапазон задержки</label><div class="range-presets"><button type="button" data-min="10" data-max="20">10-20 сек</button><button type="button" data-min="30" data-max="60">30-60 сек</button><button type="button" data-min="60" data-max="120">1-2 мин</button><button type="button" data-min="120" data-max="300">2-5 мин</button></div></div><div class="range-grid"><div class="field"><label>От, сек</label><input id="autoMin" type="number" min="1"></div><div class="field"><label>До, сек</label><input id="autoMax" type="number" min="1"></div></div><div class="button-row"><button class="primary auto-start" type="submit">Старт</button><button class="auto-stop" type="button">Стоп</button><button class="auto-refresh" type="button">Обновить файл</button></div>';

      form.querySelector(".item-title").textContent = auto.running ? "Auto-sender работает" : "Auto-sender остановлен";
      form.querySelector(".meta").textContent = (auto.file || "auto-sender.txt") + " · строк: " + (auto.file_lines || 0) + " · отправлено: " + (auto.sent || 0);
      const last = form.querySelector(".last");
      const next = auto.next_at ? new Date(auto.next_at).toLocaleTimeString() : "";
      last.textContent = [auto.last_account ? "последний: " + auto.last_account : "", auto.last_message || "", next ? "следующее: " + next : ""].filter(Boolean).join(" · ");
      form.querySelector(".auto-error").textContent = auto.error || "";

      const checkList = form.querySelector(".check-list");
      state.accounts.forEach(acc => {
        const label = document.createElement("label");
        label.className = "check-row";
        const checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        checkbox.name = "accounts";
        checkbox.value = acc.name;
        checkbox.checked = selected.has(acc.name);
        label.appendChild(checkbox);
        label.appendChild(document.createTextNode(acc.name + (acc.login ? " (" + acc.login + ")" : "")));
        checkList.appendChild(label);
      });
      if (!state.accounts.length) {
        const empty = document.createElement("div");
        empty.className = "meta";
        empty.textContent = "Сначала добавь хотя бы один аккаунт.";
        checkList.appendChild(empty);
      }

      const minInput = form.querySelector("#autoMin");
      const maxInput = form.querySelector("#autoMax");
      minInput.value = auto.min_seconds || 30;
      maxInput.value = auto.max_seconds || 60;
      form.querySelectorAll("[data-min]").forEach(btn => {
        btn.onclick = () => {
          minInput.value = btn.dataset.min;
          maxInput.value = btn.dataset.max;
        };
      });
      form.querySelector(".auto-stop").disabled = !auto.running;
      form.querySelector(".auto-stop").onclick = async () => {
        try {
          state.autoSender = await api("/api/auto-sender", { method: "DELETE" });
          setStatus("auto-sender stopped");
          await renderAutoSender(target);
        } catch (err) { setStatus(err.message.trim(), true); }
      };
      form.querySelector(".auto-refresh").onclick = async () => {
        await renderAutoSender(target);
        setStatus("auto-sender file refreshed");
      };
      form.onsubmit = async (event) => {
        event.preventDefault();
        const accounts = Array.from(form.querySelectorAll('input[name="accounts"]:checked')).map(input => input.value);
        try {
          state.autoSender = await api("/api/auto-sender", { method: "POST", body: JSON.stringify({ accounts, min_seconds: Number(minInput.value), max_seconds: Number(maxInput.value) }) });
          setStatus("auto-sender started");
          await renderAutoSender(target);
        } catch (err) { setStatus(err.message.trim(), true); }
      };
      target.appendChild(form);
    }

    document.getElementById("channelForm").onsubmit = async (event) => {
      event.preventDefault();
      const channel = channelInput.value.trim().replace(/^#/, "");
      if (!channel) return;
      try {
        setStatus("connecting channel");
        chat.innerHTML = "";
        state.seen.clear();
        clearReplyTarget();
        applyState(await api("/api/channel", { method: "POST", body: JSON.stringify({ channel }) }));
        await loadTools();
        setStatus("ready");
      } catch (err) { setStatus(err.message.trim(), true); }
    };

    document.getElementById("sendForm").onsubmit = async (event) => {
      event.preventDefault();
      const input = document.getElementById("messageInput");
      const message = input.value.trim();
      if (!message) return;
      try {
        await sendMessage(message);
        input.value = "";
        setStatus("sent");
      } catch (err) { setStatus(err.message.trim(), true); }
    };

    async function sendMessage(message) {
      await api("/api/send", { method: "POST", body: JSON.stringify({ message, reply_to: state.replyTo ? state.replyTo.id : "" }) });
      clearReplyTarget();
    }

    function insertAtCursor(value) {
      const start = messageInput.selectionStart ?? messageInput.value.length;
      const end = messageInput.selectionEnd ?? messageInput.value.length;
      const before = messageInput.value.slice(0, start);
      const after = messageInput.value.slice(end);
      const needsLeftSpace = before && !before.endsWith(" ");
      const needsRightSpace = after && !after.startsWith(" ");
      const insert = (needsLeftSpace ? " " : "") + value + (needsRightSpace ? " " : "");
      messageInput.value = before + insert + after;
      const pos = before.length + insert.length;
      messageInput.focus();
      messageInput.setSelectionRange(pos, pos);
    }

    function renderEmojiPicker() {
      emojiPicker.innerHTML = "";
      emojis.forEach(emoji => {
        const btn = document.createElement("button");
        btn.className = "emoji-choice";
        btn.type = "button";
        btn.textContent = emoji;
        btn.onclick = () => {
          insertAtCursor(emoji);
          emojiPicker.classList.remove("active");
        };
        emojiPicker.appendChild(btn);
      });
    }

    document.getElementById("addAccountForm").onsubmit = async (event) => {
      event.preventDefault();
      const payload = {
        name: document.getElementById("newName").value.trim(),
        login: document.getElementById("newLogin").value.trim(),
        token: document.getElementById("newToken").value.trim(),
        socks5_proxy: document.getElementById("newProxy").value.trim()
      };
      try {
        setStatus("adding account");
        applyState(await api("/api/accounts", { method: "POST", body: JSON.stringify(payload) }));
        event.target.reset();
        setStatus("account saved");
      } catch (err) { setStatus(err.message.trim(), true); }
    };

    document.getElementById("saveProxy").onclick = async () => {
      const selected = state.selected || state.current;
      try {
        setStatus("saving proxy");
        applyState(await api("/api/account/proxy", { method: "POST", body: JSON.stringify({ name: selected, proxy: proxyInput.value.trim() }) }));
        setStatus("proxy saved");
      } catch (err) { setStatus(err.message.trim(), true); }
    };
    document.getElementById("clearProxy").onclick = () => { proxyInput.value = ""; document.getElementById("saveProxy").click(); };
    document.getElementById("clearChat").onclick = () => { chat.innerHTML = ""; state.seen.clear(); };
    document.getElementById("cancelReply").onclick = clearReplyTarget;
    document.getElementById("refreshState").onclick = async () => { try { applyState(await api("/api/state")); setStatus("ready"); } catch (err) { setStatus(err.message.trim(), true); } };
    document.getElementById("reloadTools").onclick = loadTools;
    chatPageTab.onclick = () => setPage("chat");
    autoPageTab.onclick = () => setPage("auto");
    sendAccountSelect.onchange = async () => {
      if (!sendAccountSelect.value) return;
      try {
        setStatus("switching account");
        state.selected = sendAccountSelect.value;
        applyState(await api("/api/account", { method: "POST", body: JSON.stringify({ name: sendAccountSelect.value }) }));
        await loadTools();
        setStatus("ready");
      } catch (err) { setStatus(err.message.trim(), true); }
    };
    toggleTools.onclick = () => {
      const hidden = toolsPanel.classList.toggle("collapsed");
      rightPanel.classList.toggle("tools-hidden", hidden);
      toggleTools.textContent = hidden ? "Показать" : "Скрыть";
      toggleTools.title = hidden ? "Показать" : "Скрыть";
      localStorage.setItem("tw-tools-hidden", hidden ? "1" : "0");
    };
    if (localStorage.getItem("tw-tools-hidden") === "1") {
      toolsPanel.classList.add("collapsed");
      rightPanel.classList.add("tools-hidden");
      toggleTools.textContent = "Показать";
      toggleTools.title = "Показать";
    }
    emojiButton.onclick = () => emojiPicker.classList.toggle("active");
    document.addEventListener("click", (event) => {
      if (!emojiPicker.contains(event.target) && event.target !== emojiButton) {
        emojiPicker.classList.remove("active");
      }
    });
    renderEmojiPicker();
    themeToggle.onclick = () => {
      applyTheme(document.body.classList.contains("theme-dark") ? "light" : "dark");
    };
    applyTheme(localStorage.getItem("tw-theme") || "dark");

    document.addEventListener("visibilitychange", () => {
      if (!document.hidden && state.channel) {
        player.src = playerURL(state.channel) + "&resume=" + Date.now();
      }
    });

    document.querySelectorAll(".tab").forEach(tab => {
      tab.onclick = () => {
        document.querySelectorAll(".tab").forEach(item => item.classList.remove("active"));
        tab.classList.add("active");
        state.tab = tab.dataset.tab;
        loadTools();
      };
    });

    const events = new EventSource("/events");
    events.onmessage = (event) => {
      if (!event.data || event.data === "{}") return;
      const payload = JSON.parse(event.data);
      if (payload.type === "state") {
        applyState(payload.data);
        setStatus("ready");
        if (state.page === "auto") renderAutoSender(autoBody);
        else loadTools();
      }
      if (payload.type === "auto") {
        state.autoSender = payload.data || {};
        if (state.page === "auto") renderAutoSender(autoBody);
      }
      if (payload.type === "chat") addMessage(payload.data);
    };
    events.onerror = () => setStatus("event stream reconnecting", true);
  </script>
</body>
</html>`
