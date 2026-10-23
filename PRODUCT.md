# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Anyone with an Urbit ID, including people who have never opened a dojo.
They want their files on every computer they own, and want to share
folders with friends' ships, without a company in the middle. Most know
Dropbox-style apps. Many don't know Urbit vocabulary: moon, `+code`,
desk, galaxy.

Two places they meet Drive:

- **On their planet** (the `%drive` web page, opened from Landscape):
  pair and remove devices, see folders and their files, share a folder
  with another ship and accept invitations, restore deleted files and
  earlier versions, set how long history is kept.
- **On each computer** (the desktop client: a tray icon plus a small
  local settings page): pair the computer with their planet once, then
  forget about it. They check it to see whether things are synced, open
  the folder, exclude a subfolder, or start Drive at login.

## Product Purpose

Drive keeps a folder in sync across a person's computers, and lets them
share folders with other ships, the way Dropbox does, but with their own
planet as the server. Success: they pair a computer in a minute, files
appear everywhere without them thinking about it, and nothing is lost:
conflicts become side-by-side copies, and deleted files can be brought
back.

## Positioning

Your planet is the server. Each computer becomes a moon of your planet,
with its keys made on the computer itself. Files never pass through a
third party, and sharing is planet to planet. A hosted sync service
can't truthfully claim either.

## Operating Context

- The planet page runs inside a logged-in Urbit session (Landscape tile,
  `/drive/`), on desktop or phone, hosted or self-hosted ships.
- The desktop client runs on macOS, Windows and Linux. The tray menu is
  native. Settings and pairing open as a local page in the browser
  (`127.0.0.1`, token-guarded).
- Pairing needs the planet's web address and its `+code`, which a
  newcomer may not know how to find.
- The client is downloaded from
  https://github.com/mopfel-winrux/urbit-drive/releases/latest.

## Capabilities and Constraints

- **Devices:** pairing creates a new moon per computer, and removing a
  device revokes it. Every device stays on record with its name.
- **Folders:** each folder has files with size and time, conflicted
  copies ("name (conflicted copy, device, date)"), a trash and earlier
  versions kept for a set number of days (default 30), download of any
  version, and restore.
- **Sharing:** with another ship as can-edit or read-only. Invitations
  are accepted or declined on the receiving planet. Unsharing stops
  future access but can't erase what was already shared.
- **Selective sync:** per computer, exclude a path in a folder.
- **States the UI must show:** connecting, syncing (n of m), uploading,
  up to date, read-only folder, offline, removed device, folder no longer
  shared.
- **File-level status leads; device status doesn't.** On the planet page,
  which computers are synced is secondary: devices are managed there (pair,
  name, remove) but their sync state isn't a headline. The tray shows
  files: which ones are synced, uploading, downloading, waiting or in
  conflict, with the latest activity first.
- **Technical:** the planet page is static HTML/CSS/JS served by the
  desk (no build step; `ship.js` talks to the agent over eyre). The
  client's page is a Go `html/template` with inline CSS. The tray is
  fyne systray: a menu, an icon, text items. Neither page may load
  assets from other hosts.

## Brand Commitments

- The name is **Drive**.
- **Standing preference: the category standard, Dropbox.** The planet
  page works as closely as possible to Dropbox's web app. Folders is the
  main view, with Activity and More beside it (no Home). Dropbox's craft
  is the bar, and it's executed straight, without an invented visual world.
- Nothing else is binding. The current icon, colours and layout are
  evidence of a first pass, and can be replaced if that makes a better
  product.

## Evidence on Hand

- A working product on fake ships (`~zod`, `~bus`), with real folders,
  files, devices, shares, history and conflicts to show in screenshots.
- No users, testimonials, usage numbers or press yet. Don't invent any.

## Product Principles

1. **Set and forget.** The best session is the one where nobody opens
   Drive. When they do, the first thing they see answers "is everything
   synced?".
2. **Never lose a file, and say so.** Conflicts, deletes and revocations
   are explained plainly, along with how to get things back.
3. **Urbit without the jargon.** Use people's words (computer, folder,
   share, friend's ship) first, and Urbit terms only where they need to
   act on them (a `+code`, a ship name), with a hint.
4. **The planet is yours.** Make it felt, not lectured: no third party
   appears anywhere in the flow.

## Accessibility & Inclusion

No product-specific requirement established beyond normal good practice:
keyboard operable, readable contrast in light and dark, works at phone
width.
