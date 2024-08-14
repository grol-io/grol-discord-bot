# grol-discord-bot
Discord bot offering grol

![screenshot](screenshot.png)

## Invite link
Invite link (add grol bot to your server):

https://discord.com/oauth2/authorize?client_id=1262453631706202193

(doesn't seem to work without adding it to a server)
## Install

```
CGO_ENABLED=0 go install -trimpath -ldflags="-w -s" grol.io/grol-discord-bot@latest
```

And see [grol.service](grol.service) for systemd setup.

Though really... We're running it

Current default setup allows `save()` and thus `load()` to a single `.gr` file in the CWD.
Can be disabled setting env var `GROL_DISABLE_AUTOSAVE=1`.

### Dev bot version

Note for self

Portal: https://discord.com/developers/applications

Add: bot, Send Messages, Send Messages in Thread to Installation
and MESSAGE CONTENT INTENT in "bot"

Now grol-dev vs grol

Grol dev

https://discord.com/oauth2/authorize?client_id=1273320816066560041
