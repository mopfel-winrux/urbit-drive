# %drive — Dropbox-style sync over mesa

Status: draft 0.4, 2026-10-23. Milestones 1–3 work on fake `~zod` + `~bus`,
and milestone 4 is done for Linux (see Milestones).

A gall desk, `%drive`, turns a planet into a file-sync server. Native clients
(Go, on [libmesa](https://github.com/mopfel-winrux/libmesa)) run on
Linux/macOS/Windows as **moons of that planet** and keep a local folder in
sync with it over mesa: pokes and subscriptions for control, remote scry for
all bulk data. Planets share folders with each other using the same protocol
the moons use with their planet.

Assumptions: vere64 and large-atom storage, so loom size and atom size are
not design constraints.

## Decisions

| topic | decision |
| --- | --- |
| scope | one user, many devices **and** sharing with other ships |
| storage on the planet | content-addressed chunk store + manifests + revision log in `%drive`'s state (not clay) |
| client | Go, the libmesa cgo binding, fsnotify, system tray UI |
| encryption | planet sees plaintext; transport is mesa (`%shut` group keys) |
| share routing | the owner's planet hosts a folder and orders its commits; a guest planet keeps a mirror; devices only ever talk to their own planet |
| permissions | per-member `%r` / `%rw` on each shared folder |
| OS model | a plain folder on disk with full local copies (selective sync later); no files-on-demand |
| history | 30 days of revisions + trash, then unreferenced chunks are deleted (configurable) |
| desk publisher | configurable in the client; placeholder for now |
| desk scaffold | [build-urbit-app](https://github.com/mopfel-winrux/build-urbit-app) template |

## Roles

```
   device (moon, Go client)          planet (%drive)               other planet (%drive)
   ┌──────────────────────┐   mesa   ┌────────────────────┐   mesa   ┌────────────────────┐
   │ watcher + local db   │◀────────▶│ host of own folders│◀────────▶│ host of folder F   │
   │ serves its staged    │          │ mirror of shared F │          │ (owner)            │
   │ chunks (%shut)       │          │ serves everything  │          │                    │
   └──────────────────────┘          │ to its moons(%shut)│          └────────────────────┘
                                     └────────────────────┘
```

There is one relationship, used twice: a **follower** syncs a folder with
its **host**. A moon follows its own planet for every folder the planet has.
A guest planet follows the owner's planet for each folder shared with it, and
re-hosts the mirror to its own moons. The wire protocol is identical; only
authorization differs.

## Data model

```hoon
+$  hash     @ux                       ::  sha-256 over (len, bytes), via +shay
+$  chunk    [len=@ud dat=@]           ::  ≤ 4 MiB; content-defined boundaries
+$  entry                              ::  a file at one revision
  $:  =hash                            ::  hash of the whole file contents
      size=@ud
      mtime=@da                        ::  the writer's mtime, informational
      exec=?                           ::  unix x bit
      chunks=(list hash)               ::  in order
  ==
+$  op       [=path new=(unit entry)]  ::  ~ = delete
+$  aeon     @ud                       ::  per-folder revision, dense from 1
+$  wave                               ::  one committed revision, immutable
  $:  =aeon
      author=ship                      ::  device or guest planet that wrote it
      when=@da
      ops=(list op)
  ==
+$  folder-id  @uv                     ::  random; stable across renames
+$  folder
  $:  name=@t
      host=ship                        ::  owner planet (us, or a share's owner)
      head=aeon
      tree=(map path entry)            ::  state at head
      log=((mop aeon wave) lth)        ::  pruned after retention
      members=(map ship ?(%r %rw))     ::  only when we host and share
      epoch=@ud                        ::  coop key epoch (bumped on revoke)
  ==
```

Chunks are content-defined (FastCDC, min 256 KiB / avg 1 MiB / max 4 MiB) so
an edit in the middle of a large file re-sends only the chunks around it,
and identical chunks across files and folders are stored once. Hashes use
`+shay` with an explicit length, because `+shax` on an atom drops trailing
zero bytes.

## Namespace (what the host serves)

Everything a follower reads by remote scry has a path that is bound once and
never rebound, so a relay can cache it and any holder can re-serve it. Each
binding is a gall `%tend` at revision 1 under a coop:

```
coop /f/<folder-id>/<epoch>
  /f/<folder-id>/<epoch>/wave/<aeon>     -> wave
  /f/<folder-id>/<epoch>/rock/<aeon>     -> (map path entry), every 256 aeons and on demand
  /f/<folder-id>/<epoch>/blob/<hash>     -> chunk
```

Read as `/g/x/1/drive//1/f/<id>/<epoch>/...`. Gall asks `%drive`'s `on-peek`
at `/c/f/<id>/<epoch>/<ship>` whether a ship may have the coop key; `%drive`
says yes to its own moons that are registered devices and to the folder's
members. Blobs live under each folder's coop, not in one global pool, so
a share's key never reveals another folder's chunks. A chunk that appears in
two folders is stored once and published under both.

**Revocation.** Gall cannot drop a coop key yet (`%snip` is a TODO in
gall.hoon), so removing a member bumps `epoch`. The host publishes the latest
rock and all live blobs under the new coop (the nouns are shared, so this
doesn't copy data), stops publishing under the old one, and tells the
remaining followers the new epoch. The revoked member keeps what it already
had and gets nothing new.

**Frontier** (which aeon is the latest) isn't in the namespace, since it
changes. Followers learn it from a gall subscription:

```
watch /v1/folders  (devices)  /v1/share/<fid>  (guests)   ;  %drive-update
  [%hello folders=(list info) results=(list result)]  the only fact a watch starts with
  [%folder fid name host epoch head role]             name, epoch (rekey) or role changed
  [%head fid aeon]                                    each commit
  [%result who cid (each aeon @t)]                    outcome of a commit
  [%gone fid]                                         left, or unshared
```

A watch is answered with **one** `%hello`. Ames kicks a subscriber once
more than five facts on its flow are unacked (`msg.cong`). An earlier
version sent a fact per folder plus one per replayed result, and a device
with a few failed commits got stuck kicking and resubscribing forever.

A follower watches afresh every time it starts. It leaves its old
subscription, gets the `%folder` snapshot again, and reads the waves it is
missing (or the nearest rock, if they've been pruned). A fresh watch also
replays the `%result`s of that ship's commits, because a commit that changed
nothing produces no wave. Starting over matters in practice: after a device
has been offline, facts on its old flow sit in the planet's retransmit
backoff. In testing they arrived about 90 s after the device came back,
while a fresh watch answers in under a second. The subscription only makes
updates arrive faster. Correctness comes from the immutable log.

## Writing: commits are pull-based

A follower never pushes file contents. It publishes them and asks the host to
read them:

1. The follower (a moon) chunks the changed files, stores those chunks in its
   staging area, and serves them under its own `%shut` prefix
   `/g/x/1/drive//1/up/<hash>`. Its `ShutKey` hook gives the key only to its
   planet (or, for a guest planet, to the folder's owner).
2. It pokes `%drive-commit`:
   `[cid=@uv folder=folder-id base=aeon ops=(list [path base=(unit hash) new=(unit entry)])]`.
   `base` is the hash of the file the follower last saw at that path, used for
   compare-and-swap.
3. The host acks the poke once it has recorded it (`cid` makes resends
   idempotent). It then reads every chunk it doesn't already have from the
   follower with `%keen` (secret), checks each hash, and applies the commit.
4. **Conflicts are settled per file on the host**, which puts all commits in
   one order:
   - if `base` equals the current hash, the op applies;
   - if they differ and both sides changed contents, the incoming file is
     written as `name (conflicted copy, ~device, 2026-10-23).ext` and the
     current one is kept;
   - an edit beats a concurrent delete: the edited file comes back. As in
     clay-sync, we lean toward keeping data.
5. The host publishes the new wave and gives `[%head id aeon]` and
   `[%result cid %& aeon]`. The follower then pulls the wave like any other,
   which includes any conflicted copies.

A guest planet with `%rw` relays its moons' commits upstream: it pokes the
owner with the same commit, serves the chunks from its own store, and
forwards the `%result`. A `%r` member's commits are nacked.

## Pairing a device

Pairing is the only step that uses HTTP. Everything after it runs over mesa.

1. The user gives the client the planet's URL and `+code`. The client logs
   in to eyre (`POST /~/login`).
2. `GET /~/scry/drive/version.json`. On 404 the client installs the desk
   (`docket-install` poke with `~publisher/drive`, falling back to kiln) and
   polls until the agent answers.
3. The client generates its moon keys **locally** (a random `ring`) and pokes
   `%drive` with `{"pair": {"name": "<device>", "pass": "<@ux public key>"}}`.
   The private key never leaves the device.
4. `%drive` chooses an unused moon, passes jael
   `[%moon who [[rift=0 life=1] %keys [1 1 pass] boot=|]]`, records the
   device, and returns the moon's name through the poke's scry result
   (`/~/scry/drive/pair/<nonce>.json`).
5. The client writes its keyfile, opens a libmesa node with `Breached: true`
   (a fresh identity), and subscribes to `/v1/folders`. The planet learned the
   moon's keys from jael, so it first speaks classic ames to the moon.
   libmesa's `%ahoy` moves that conversation to mesa.
6. The eyre cookie is thrown away.

**Re-pairing** (store lost, reinstall) creates a new moon. Unpairing (from
the client, or from the web UI) removes the device, and `%drive` stops
handing the moon coop keys. Keys the moon already has stay valid until the
epoch is bumped, so unpairing a lost device also bumps every folder's
epoch.

**Fake ships** (development): keys are derived from ship names, so the agent
registers fake keys for the moon and the client uses `FakeIdentity(moon)`.

## Sharing (milestone 2, implemented)

- The owner pokes its own `%drive` `[%share id ~guest %rw]`. `%drive` adds the
  member and pokes the guest's `%drive` with an invite
  `[%invite id name ~owner]`.
- The guest accepts in the web UI or the client. Its `%drive` then follows
  `~owner` exactly as a moon would: it subscribes to `/v1/folders` on the
  owner, filtered to what the guest can see, and reads waves and blobs with
  `%keen` secret. It hosts the mirror to its own moons under its own coop
  `/f/<id>/<local-epoch>`, re-publishing the same blobs it already holds.
- For a mirrored folder, `head` and the aeon numbers are the owner's.
  The guest only re-publishes them, so a guest's moons and the owner's moons
  see the same history.
- A guest device's commit goes to its own planet. The planet reads the
  chunks, publishes them at its own `/up/<hex>` (coop `/up`, whose key goes
  to the hosts of folders it mirrors), and pokes the owner
  `[%relay author device-name commit]`. The owner accepts it only from a
  `%rw` member, for itself or one of its moons. It reads the chunks from the
  guest's `/up` and applies the commit with the **device** as the wave's
  author, so the device recognizes its own wave. The guest forwards the
  `%result`.
- `%r` members' devices learn their role from `%folder` and don't upload.
  The planet refuses their commits anyway.
- A mirror's local name gets ` (~owner)` appended if the guest already has a
  folder with that name, because devices use the name as the directory.
- Unsharing kicks the guest, pokes it `%uninvite`, and rekeys. Unpairing a
  device rekeys every folder, mirrors included.

## Client

```
client/
  cmd/drive/          entry point: tray app, or `drive daemon|pair|status`
  internal/pair/      eyre login, desk install, moon key generation
  internal/sync/      the follower: subscriptions, wave pulls, commit pushes
  internal/scan/      fsnotify watcher, debounced like clay-sync's (100 ms quiet,
                      1 s cap, 300 ms delete grace), plus a periodic full sweep
  internal/chunk/     FastCDC + sha-256
  internal/store/     SQLite tables beside libmesa's (one database)
  internal/ui/        system tray
```

**Local state** lives in the same SQLite database as the node, so applying
a wave and acking the fact that announced it happen in one transaction:

- `files(folder, path, hash, size, mtime_ns, inode, aeon)`: the last version
  we synced, the base for compare-and-swap, and how a local edit is detected
  (a cheap mtime/size/inode check, then a hash).
- `staged(hash, data)`: chunks of commits not yet applied, so they stay
  servable even if the file changes before the planet reads them. They're
  deleted when the `%result` arrives.
- `pending(cid, folder, body)`: commits sent and waiting for their result.

**Applying a wave to disk:** write to a temporary file in the same
directory, check its hash, then `rename`. If the local file changed since
the last sync, this is clay-sync's don't-clobber rule:
- someone else changed a file we edited and haven't committed: move our
  edit aside as a conflicted copy (the next scan uploads it), then write
  theirs;
- our edit is already in a commit in flight: write theirs, because the
  planet will turn ours into the conflicted copy;
- our own wave arriving after a newer local edit: keep the edit, and
  commit it next on top;
- a delete arriving for a file we edited: keep the file (an edit beats a
  delete).

A device scans only once it has heard each folder's head in the current
run, so it doesn't commit on a stale base after a restart.

**Platforms:** the Go binding uses only libmesa's core (Go does the UDP,
timers and HTTP), so a Windows build needs only the core's C (urcrypt subset +
SQLite) built with `zig cc`. Sync works only through files on disk, so it's
the same on every OS. The tray uses `fyne.io/systray`.

## Retention (milestone 3)

- Each folder keeps waves `floor+1..head` in its log, and its tree at
  `floor` as `base`. Every 6 hours, `prune-all` folds waves older than
  `keep` (default 30 days; settable from the web page) into `base`, and
  publishes `base` as `/rock/<floor>`. A follower behind the floor (a new
  device, a device back after a long time, or a mirror) reads the rock and
  then the waves after it. A device treats the rock as a wave from the
  planet that deletes whatever it has that the rock doesn't.
- Every replaced or deleted entry goes to `old` (newest first) for `keep`.
  The web page lists them as **Trash** (paths no longer in the tree) and
  **Earlier versions**. Any of them can be downloaded
  (`/~/scry/drive/file/<fid>/<hash>.mime`) or restored. A restore is an
  ordinary commit from the planet, so it works on mirrors too, by relay.
- A chunk is live while any tree, base, live wave, version, commit in
  flight, or pending mirror read refers to it. Pruning drops the others.
- **Compaction through epochs.** Readers always ask for revision 1, so a
  tombed binding can never come back at the same path. Instead, every folder
  whose readable set changed moves to a new coop epoch (`+rekey`): what's
  live is published under the new coop, and everything under the old one
  is `%tomb`ed (its value replaced by a hash, freeing it). Revocation uses
  the same move, so an old key reads nothing at all afterwards. A device
  that sees a new epoch cancels its reads in flight and starts over with
  the new key.

## Selective sync

A device can exclude a path (a file, or a folder and everything in it) from
its copy of a folder: `drive exclude FOLDER PATH` or the settings page.
Unedited local copies are removed. Excluded paths are neither downloaded nor
scanned. Including a path again sets a **replay**: the device rereads the
folder from its rock or its first wave, applying only ops under the included
path, until it is back to where it was. The replay is persisted, so a crash
mid-replay doesn't re-apply other paths' history.

## Milestones

1. **Vertical slice:** done.
2. **Sharing:** done (invites, mirrors, roles, relays, revocation).
3. **Retention:** done (pruning, rocks, history, trash, restore, download,
   chunk GC, storage display).
4. **Packaging:**
   - Linux amd64/arm64: static binaries, tarball with `install.sh` (desktop
     entry, autostart, systemd user unit) and `.deb`. Built and tested here.
   - macOS: libmesa cross-builds, but the tray needs the macOS SDK, so
     macOS builds run on macOS CI runners. `Drive.app` and the dmg script
     haven't been run yet.
   - Windows: libmesa's core and its Go binding build for Windows (libmesa
     10b79c6). `drive.exe` cross-compiles from Linux with `zig cc`, and the
     release workflow ships the zip. The Inno Setup installer needs a Windows
     runner (`WINDOWS_BUILDS=true`). Not yet run on Windows.
   - `.github/workflows/release.yml` publishes tagged builds to
     `mopfel-winrux/urbit-drive` releases, and signs them when the signing
     secrets are set. It has never run on GitHub.
   - Start-at-login toggle in the tray and the settings page; the new icon in
     the tray, the installers and the planet tile.

## Known issues

- Installing from a publisher: `|install ~zod %drive` on `~bus` stalled.
  `~zod`'s clay logged `%clay-sing-indirect-data` for `care=%y case=[%ud 1]`
  on the `|new-desk`-made desk. It's unclear whether this is a fake-ship or
  clay quirk. The test installed `~bus` from a local copy instead. A real
  publisher desk needs checking before release.
- `done` (commit results) isn't pruned yet. It's small, but it grows by one
  entry per commit.
- Empty directories and symlinks aren't synced.

## Open questions

- Large commits (an initial upload of 10k files): split into commits of at
  most N ops / M bytes so one commit never blocks the flow for long.
- How many chunks to read in parallel. mesa's congestion control is per
  peer, so start with 8 outstanding `%keen`s and tune from there.
- Does a moon's sponsor-side routing hold when the planet is behind NAT and
  the moon is on another network? Test on the live network early in
  milestone 2.

## Running it

See the README's Development section: `client/make` fetches libmesa (a submodule) and zig 0.15.2, and `tools/boot-fakezod.sh` fetches vere.
