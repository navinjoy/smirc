# Minimalistic IRC client in Go.

## Principles:
  - everything in one file
  - no Javascript
  - no unnecessary features

Run it with: `go run ./minirc.go`

## Configuration
1. Change the `smirc.conf` JSON file: 
```json
{
  "server": "irc.freenode.net",
  "port": 6667,
  "channel": "#midnightcafe",
  "web-server-port-number": 8080
}
```
  - change `server` to your favorite [IRC server](https://www.mirc.com/servers.html)
  - change `channel` to your favorite channel

2. There are a few environment variables that need to be set:
  - `IRC_NICKNAME` - Your nickname is how other chat users will see you
  - `IRC_USERNAME` - What's your Username?
  - `IRC_REALNAME` - What's your Real Name?
  - `CONFIG_FILENAME` - point this to `smirc.conf`

## More
For IRC protocol details see: https://www.ietf.org/rfc/rfc1459.txt