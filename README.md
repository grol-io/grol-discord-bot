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
