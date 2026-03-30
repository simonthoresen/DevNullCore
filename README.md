# null-space

null-space is a Windows server for hosting real-time multiplayer terminal games over SSH. You run it on your machine, share an invite command, and anyone can join instantly with a plain `ssh` command — no client software required. Games and plugins are single JavaScript files that you drop in a folder (or load directly from a URL), so creating and sharing new games is as simple as writing a `.js` file and pasting a GitHub link. The server handles everything else: player connections, a shared chat channel, synchronized rendering at 60 fps, and automatic tunnel setup via Pinggy so you can host from anywhere without touching your router.

## Install

Paste this into a PowerShell window:

```powershell
iwr -useb https://raw.githubusercontent.com/simonthoresen/null-space/main/install.ps1 | iex
```

This creates a `NullSpace` folder in your current directory containing everything you need. No other dependencies required.

## Start the server

```powershell
cd NullSpace
.\start.ps1 --password yourpassword
```

The server prints an invite command that others can paste into any terminal to join.

## Load a game

Once the server is running, type into the server console:

```
/game load example
/game load https://github.com/someone/repo/blob/main/mygame.js
```

## Write your own game

See [API-REFERENCE.md](API-REFERENCE.md) for the full JavaScript API.
