# twitch-multi-tool

Small CLI built on `github.com/adeithe/go-twitch` for managing several Twitch OAuth accounts.

## Config

Create `accounts.json` from `accounts.example.json`.

```json
{
  "accounts": [
    {
      "name": "bot1",
      "login": "bot_login",
      "token": "oauth_token_without_oauth_prefix",
      "socks5_proxy": "socks5://127.0.0.1:1080"
    }
  ]
}
```

`login`, `client_id`, `user_id`, and `socks5_proxy` are optional. The app fills Twitch IDs from OAuth validation when it runs. SOCKS5 may be `host:port`, `socks5://host:port`, or `socks5://user:pass@host:port`.

## Commands

Validate tokens:

```sh
go run . accounts
```

Send a chat message or emote code:

```sh
go run . send --account bot1 --channel somechannel --message "Kappa"
```

Send from all configured accounts with a delay:

```sh
go run . send --all --channel somechannel --message "hello" --delay 2s
```

Browser UI:

```sh
go run . web --channel somechannel
```

The web UI shows the Twitch stream in the center, chat on the right, and accounts on the left. Change the connected channel from the top channel input; the player, chat reader, emotes, rewards, and active account writer reconnect to that channel.
The Presets tab lets you save message templates and send them to chat with one click.

Live chat mode:

```sh
go run . live --account bot1 --channel somechannel
```

In live mode:

```text
/accounts          list configured accounts
/account bot2      switch active account
/whoami            show active account
/emotes channel    list first 20 channel emotes
/rewards           list official custom rewards for this channel
/quit              exit
```

List emotes:

```sh
go run . emotes --account bot1 --kind global
go run . emotes --account bot1 --kind channel --channel somechannel
go run . emotes --account bot1 --kind user --channel somechannel
```

List official broadcaster custom rewards:

```sh
go run . rewards --account broadcaster
```

## Twitch scopes

For IRC chat sending, use a user OAuth token with chat write permissions. Older IRC flows use `chat:edit`; modern Helix chat uses `user:write:chat`.

For broadcaster custom reward listing, Twitch requires the broadcaster token with `channel:read:redemptions` or `channel:manage:redemptions`.

## Channel Points limitation

Twitch Helix does not expose an official viewer API to read a viewer's Channel Points balance or redeem a reward as a viewer. This tool does not use private/undocumented endpoints for that.
