# Roadmap

Where Drive goes after v0.2. It's ordered roughly by when it matters: first
what makes Drive trustworthy, then where it falls short of Dropbox, then what
only Drive can do because it runs on Urbit. [PROTOCOL.md](PROTOCOL.md) has the
design these build on.

**Next up**, after the first round of real-world testing:

1. Live-network hardening and installing from a publisher, because nothing
   else matters until other people can install it.
2. Previews in the web app, the cheapest big jump in how finished it feels.

**Done since v0.2:** share links (files, subfolders and whole folders, with
optional expiry and password, view and download counts, a Links page, and
"Download all" as a zip). See PROTOCOL.md, Share links.

## 1. Trustworthy

Before anyone relies on it.

- **The live network.** Real planets behind NAT, computers on other networks
  than their planet, hosted ships. Routing between a computer and a planet
  hosted at home is the biggest unknown so far: everything to date ran on fake
  ships on one machine.
- **Installing from a publisher.** `|install ~publisher %drive` stalled in
  testing: the publisher's clay failed to serve the desk's first revision
  (`%clay-sing-indirect-data`). This has to work before the desk can be listed
  in Landscape, and before the desktop app's "install Drive for me" step is
  useful.
- **Large folders.** First sync of tens of thousands of files and multi-GB
  files:
  - split big commits into batches;
  - read several chunks at once from the planet (today it reads them one wave
    at a time);
  - watch the planet's state and event-log growth.
- **Real Macs and Windows machines.** Tray behaviour, signing and
  notarization, the Windows installer, start at login. All of it is built, and
  none of it has been run on those platforms.
- **Bounded growth.** Prune the planet's record of commit results, and trim
  the event log after large uploads.
- **Crash safety.** Today a large wave is applied to disk inside one
  transaction. Interrupted downloads should resume where they stopped.

## 2. Parity with Dropbox

What a Dropbox user notices is missing.

- **Empty folders and symlinks made on a computer.** The planet can already
  create empty folders, but computers don't upload them yet.
- **Previews** in the web app: images, PDFs, text, audio and video.
- **File-type icons**, plus Share and Download on row hover.
- **Real moves.** A move is a delete plus an add today, so version history and
  Activity lose the thread. Record renames and moves as such.
- **Pause and resume**, bandwidth limits, and clear states for a full disk or
  a folder the computer can't write to.
- **Files on demand.** Placeholders that download when opened, using macOS
  File Provider and Windows Cloud Files (and FUSE on Linux). It's a big job,
  and it's what makes huge folders usable on small laptops.
- **Phones.** A good mobile web app first, then a real app; photo backup is
  the feature that sells it.
- **Search inside files**, not just names.
- **Notifications** for new shares, conflicts, and a computer that's gone
  quiet.

## 3. Only on Urbit

What a hosted sync service can't offer, because your ship is the server and
the namespace is the transport.

- **Share links, further.** Done: links to files and folders, with optional
  expiry and password. Next: links for people to upload into a folder (file
  requests), links made by guests with edit rights, download limits, and
  rate limiting on the ship.
- **Share with groups.** Share a folder with a Tlon group or a list of ships,
  and access follows group membership.
- **Folders other apps can read.** Let other Urbit apps (notes, a photo
  gallery, a blog) read your folders through the namespace, with your
  permission. Your files become a data layer, not a silo.
- **Public folders as publishing.** A folder anyone can read by remote scry.
  Relays cache it, so a popular folder doesn't load your planet. Think static
  sites, podcasts, datasets.
- **End-to-end encryption, per folder.** For people on hosted ships, folders
  whose contents the planet can't read. It costs previews and server-side
  search, which is why it's opt-in.
- **Local sync between your computers.** Computers on the same network pass
  chunks directly while the planet is far away or offline. The planet still
  decides the order of changes.
- **Backup to another ship.** A friend's planet, or a star you trust, keeps
  an encrypted copy of your chunks. That's backup without a cloud company, and
  an answer to "what if my planet dies?".
- **Signed history.** Every revision is immutable and signed in the
  namespace, so Drive could prove what a file looked like at a given time.

## 4. Polish and ecosystem

- **The look in Landscape.** Keep the Dropbox-straight look on purpose
  ([DESIGN.md](DESIGN.md)), or blend into Landscape. This is a product choice.
- **Clean up small inconsistencies** in the design system: a few paddings off
  the 4px grid, the tray's colours against the page palette, the client logo
  against the app icon, and token names that differ between the planet page
  and the client page.
- **Storage** quotas per folder, and a breakdown by folder and by history, for
  hosted ships with limited space.
- **Give back to libmesa** what other native Urbit apps could use: the moon
  pairing flow, `%shut` helpers, file-event tracking.
