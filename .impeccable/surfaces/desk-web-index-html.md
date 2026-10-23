---
version: 1
slug: "desk-web-index-html"
primary_target: "desk/web/index.html"
related_targets: ["client/internal/ui/web.go"]
---

# Surface: Drive on the planet (desk/web)

Scope: the %drive web page served at /drive/ inside a logged-in Urbit
session. Mode: Operate. Related: the desktop client's local pairing and
settings page and its tray menu (client/internal/ui), which follow the same
system.

Audience and job: anyone with an Urbit ID, newcomers included. They browse
their folders and files, download, upload, rename, move, delete, restore
old versions and deleted files, share a folder with another ship and answer
invitations, see what changed, and manage computers and history settings.
File-level status leads; device sync state stays quiet.

Constraints: static HTML/CSS/JS, no build step, no external hosts. Phone
width must work.

## Direction contract

THESIS: Dropbox's web app, played straight: the file browser is the page.
It refuses an invented world, a dashboard of device health, and protocol
vocabulary up front.

OWN-WORLD: white work area, a light grey left rail, near-black text, one
Dropbox-blue for primary actions, selection and links. A system sans, a
tight 1.125 type scale, 4px spacing grid, folder and file glyphs in a
single line-icon style, hairline-ruled rows with checkbox selection.

STORY: someone opens Drive, sees their folders, clicks into one, finds a
file and downloads it or restores an older version, shares a folder, and
checks Activity to see what changed.

FIRST VIEWPORT: left rail (Drive wordmark, Folders, Activity, More, then
storage use at the foot). A top bar with search across the work area. The
main area: breadcrumb title, a toolbar (Upload, New folder, Share), then
the table: Name, Modified, Size, Who can access. A details panel opens on
the right for the selected file: its versions, with Restore.

FORM: canon, the category standard (Dropbox web); seed:68abb47cf3d64602

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
