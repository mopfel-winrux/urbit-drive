---
name: Drive
description: Your planet is the server. A Dropbox-style file browser for %drive, played straight.
colors:
  canvas: "#ffffff"
  warm-grey: "#f7f5f2"
  warm-grey-press: "#eeeae4"
  selection-blue: "#e9f1ff"
  hairline: "#e7e3dd"
  ink: "#1e1919"
  ink-secondary: "#6b645d"
  ink-tertiary: "#8f877f"
  dropbox-blue: "#0061fe"
  dropbox-blue-hover: "#0057e4"
  amber: "#a85d00"
  amber-soft: "#fff3e0"
  red: "#c4312b"
  red-soft: "#fdecea"
  green: "#1f7a4d"
  icon-blue-light: "#3d8bfd"
  icon-blue-deep: "#1f4fc4"
typography:
  headline:
    fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif"
    fontSize: "24px"
    fontWeight: 650
    lineHeight: 1.25
    letterSpacing: "-0.015em"
  title-lg:
    fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif"
    fontSize: "18px"
    fontWeight: 650
    lineHeight: 1.45
    letterSpacing: "-0.01em"
  title:
    fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif"
    fontSize: "16px"
    fontWeight: 650
    lineHeight: 1.45
  body:
    fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.45
  body-strong:
    fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif"
    fontSize: "14px"
    fontWeight: 550
    lineHeight: 1.45
  body-sm:
    fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.45
    fontFeature: "tnum"
  label:
    fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif"
    fontSize: "12px"
    fontWeight: 550
    lineHeight: 1.45
rounded:
  xs: "4px"
  sm: "6px"
  md: "8px"
  pill: "10px"
  lg: "12px"
spacing:
  "1": "4px"
  "2": "8px"
  "3": "12px"
  "4": "16px"
  "5": "20px"
  "6": "24px"
  "8": "32px"
  "16": "64px"
components:
  button-primary:
    backgroundColor: "{colors.dropbox-blue}"
    textColor: "{colors.canvas}"
    typography: "{typography.body-strong}"
    rounded: "{rounded.md}"
    padding: "0 14px"
    height: "36px"
  button-primary-hover:
    backgroundColor: "{colors.dropbox-blue-hover}"
    textColor: "{colors.canvas}"
  button-secondary:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.body-strong}"
    rounded: "{rounded.md}"
    padding: "0 14px"
    height: "36px"
  button-secondary-hover:
    backgroundColor: "{colors.warm-grey}"
  button-secondary-active:
    backgroundColor: "{colors.warm-grey-press}"
  button-danger:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.red}"
    rounded: "{rounded.md}"
    height: "36px"
  button-danger-solid:
    backgroundColor: "{colors.red}"
    textColor: "{colors.canvas}"
    rounded: "{rounded.md}"
    height: "36px"
  icon-button:
    textColor: "{colors.ink-secondary}"
    rounded: "{rounded.sm}"
    size: "32px"
  icon-button-hover:
    backgroundColor: "{colors.warm-grey-press}"
    textColor: "{colors.ink}"
  field:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "0 10px"
    height: "36px"
  search:
    backgroundColor: "{colors.warm-grey}"
    textColor: "{colors.ink-secondary}"
    rounded: "{rounded.md}"
    padding: "0 12px"
    height: "40px"
  rail:
    backgroundColor: "{colors.warm-grey}"
    padding: "16px 12px"
    width: "240px"
  nav-item:
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "0 12px"
    height: "40px"
  nav-item-active:
    backgroundColor: "{colors.warm-grey-press}"
    textColor: "{colors.ink}"
  file-row:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    padding: "0 8px"
    height: "48px"
  file-row-hover:
    backgroundColor: "{colors.warm-grey}"
  file-row-selected:
    backgroundColor: "{colors.selection-blue}"
  tag-conflict:
    backgroundColor: "{colors.amber-soft}"
    textColor: "{colors.amber}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "0 8px"
    height: "20px"
  tag-read-only:
    backgroundColor: "{colors.warm-grey}"
    textColor: "{colors.ink-secondary}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "0 8px"
    height: "20px"
  banner:
    backgroundColor: "{colors.selection-blue}"
    rounded: "{rounded.md}"
    padding: "12px 16px"
  details-panel:
    backgroundColor: "{colors.canvas}"
    padding: "20px"
    width: "320px"
  dialog:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    rounded: "{rounded.lg}"
    padding: "24px"
    width: "480px"
  menu:
    backgroundColor: "{colors.canvas}"
    rounded: "{rounded.md}"
    padding: "6px"
  toast:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.canvas}"
    rounded: "{rounded.md}"
    padding: "10px 16px"
---

# Design System: Drive

## Overview

**Creative North Star: "The Category Standard, Played Straight"**

Drive looks and behaves like Dropbox's web app, on purpose. The file browser is the page: a warm grey left rail with three destinations (Folders, Activity, More), a top bar carrying search, and a white work area where a breadcrumb title, a short toolbar and a hairline-ruled table of folders and files do almost all of the work. A details panel opens on the right for the selected file, with its versions and Restore. There is no invented visual world, no dashboard of device health, and no Urbit vocabulary on the surface; the craft bar is the category leader's, executed without decoration.

Density is a working file manager's: 48px rows, a 14px base, one system sans, numbers in tabular figures. Colour is almost entirely neutral and slightly warm, so the single Dropbox blue reads at once as "act here" or "this is selected". Amber, red and green only ever carry state: conflict, danger and failure, and done. The same tokens run in the desktop client's local settings page, so the planet page and the computer feel like one product.

Light and dark schemes are both first-class. Dark mode keeps the warm cast (a warm near-black canvas, warm greys) and lifts the blue to stay legible.

**Key Characteristics:**
- White work area, warm grey rail, near-black warm ink.
- One blue for primary actions, selection, links, folder glyphs and focus.
- Hairline-ruled rows with checkbox selection; no cards around content.
- One line-icon set on a 24-unit grid at a 1.75 stroke.
- Flat at rest; shadows only on things that float.
- Phone width: the rail slides over, the details panel takes the screen, secondary columns drop into a meta line.

## Colors

A warm-neutral field with one saturated blue, plus three state colours that never decorate.

### Primary
- **Dropbox Blue** (`{colors.dropbox-blue}`): the primary button (Upload, Download, Accept), links, the active nav icon, the active tab underline, checkbox accent, the storage meter and upload progress fill, the drop-zone dash, and the 2px focus ring. Folder glyphs are stroked in it and filled with Selection Blue. Hover darkens to **Pressed Blue** (`{colors.dropbox-blue-hover}`). In dark mode it lifts to a lighter blue, and text on it flips to a dark ink.
- **Selection Blue** (`{colors.selection-blue}`): the background of selected rows and picker choices, the invitation banner, text selection, and the folder glyph fill.

### Tertiary (state only)
- **Conflict Amber** (`{colors.amber}`) on **Amber Wash** (`{colors.amber-soft}`): conflicted copies. It tints the file glyph and fills the "conflicted copy" tag.
- **Danger Red** (`{colors.red}`) on **Red Wash** (`{colors.red-soft}`): destructive actions (outline and solid danger buttons, danger menu items), failed uploads, errors, and the "deleted" verb in Activity.
- **Done Green** (`{colors.green}`): completed uploads, "added" in Activity, the tray's synced check.

### Neutral
- **Canvas White** (`{colors.canvas}`): the work area, details panel, dialogs, menus, and secondary button faces.
- **Warm Grey** (`{colors.warm-grey}`): the left rail, the resting search field, row hover, secondary button hover, the read-only tag, skeleton bars. It is the browser theme colour in light mode.
- **Warm Grey Pressed** (`{colors.warm-grey-press}`): the active nav item, icon-button hover, button press.
- **Hairline** (`{colors.hairline}`): every 1px rule (table rows, top bar, details divider, tabs, settings rows), button and field strokes, meter tracks, scrollbar thumbs.
- **Ink** (`{colors.ink}`): primary text and the toast surface.
- **Ink Secondary** (`{colors.ink-secondary}`): column headers, metadata columns, breadcrumbs, descriptions, icon buttons, generic file glyphs.
- **Ink Tertiary** (`{colors.ink-tertiary}`): breadcrumb chevrons, empty-state glyphs, the tray's deleted glyph. Non-text only.
- **App-icon gradient** (`{colors.icon-blue-light}` to `{colors.icon-blue-deep}`): the app tile, tray icon and the client page's logo square. It belongs to the icon, not to interface chrome.

### Named Rules
**The One Blue Rule.** Blue means action, selection, link or focus. Nothing else is blue except folder glyphs, which the category also draws in blue.

**The State-Only Rule.** Amber, red and green appear only when they say something about a file or an action (conflict, danger or failure, done). Never as accents.

**The Warm Neutral Rule.** Greys carry a slight warm cast (`{colors.warm-grey}`, `{colors.hairline}`, `{colors.ink-secondary}`), in both schemes. Don't introduce cool slate greys.

## Typography

**Display Font:** none (no separate display face)
**Body Font:** system-ui (with -apple-system, Segoe UI, Roboto, Ubuntu, sans-serif)

**Character:** One native sans at every size, so the page sits inside the user's OS like a tool rather than a brand piece. Hierarchy comes from weight (400, 500, 550, 650) and a tight size ramp, not from a second family.

### Hierarchy
- **Headline** (650, 24px, 1.25, -0.015em): the folder or page title under the breadcrumb.
- **Title Large** (650, 18px): the rail wordmark and dialog titles.
- **Title** (650, 16px): section heads (h2), the details panel file name, empty-state heading.
- **Body** (400, 14px, 1.45): everything by default. File names in the table are 500. Buttons, tabs and form labels are 550 (Body Strong). Sub-heads (h3) are 650 at 14px.
- **Body Small** (400, 13px, tabular figures): details-panel facts, version rows, Activity times and sub-lists, the ship name in the top bar, error text, day headers (650).
- **Label** (550, 12px): table column headers, tags, storage usage, the phone meta line under a file name.

### Named Rules
**The Tabular Figures Rule.** Sizes, times, counts and version stamps use tabular numerals so columns line up.

**The Weight-Not-Size Rule.** Prefer a weight step (500, 550, 650) to a new size. The ramp is 12, 13, 14, 16, 18, 24 and nothing else on the planet page.

## Layout

A two-column app shell on a CSS grid: a 240px rail and the main column. The main column stacks a 64px top bar (search field up to 560px wide, ship name at the far right) over a body that splits into the scrolling content area (24px top, 32px sides, 64px bottom padding) and an optional 320px details panel behind a hairline.

Spacing sits on a 4px grid: 4, 8, 12, 16, 20, 24, 32 and 64px. Rows are 48px (`--row`); column headers 36px; buttons and fields 36px; nav items, tabs and search 40px; icon buttons 32px.

The file table is fixed-layout: a 44px checkbox column, a flexible Name column, Who can access (22%), Modified (16%), Size (11%, right-aligned) and a 52px row-menu column. Settings rows are capped at 720px. The desktop client's page is a single centred column, 640px max, 32px/24px padding.

At 760px and below: rows grow to 52px, the rail becomes a 272px off-canvas drawer opened from a menu button, the ship name hides, content padding drops to 16px, the details panel goes full-screen, the access, modified and size columns hide and reappear as a meta line under the name, secondary toolbar buttons collapse to icons, and row menus stay visible. The upload tray spans the width.

## Elevation & Depth

Flat by default. Depth on the page comes from tone (warm grey rail beside a white work area) and 1px hairlines, never from shadows on content. One shadow token exists, a soft two-layer ambient, and it is used only for surfaces that float above the page: dialogs, context menus, the upload progress tray, the drop-zone card, the toast, and the phone rail drawer when open. Dialogs dim the page with a warm, 40% black backdrop.

### Shadow Vocabulary
- **Float** (`box-shadow: 0 2px 6px rgb(30 25 25 / 0.08), 0 12px 32px rgb(30 25 25 / 0.12)`; dark: `0 2px 6px rgb(0 0 0 / 0.3), 0 12px 32px rgb(0 0 0 / 0.45)`): anything that floats over the page.

### Named Rules
**The Float-Only Rule.** A shadow means "this is above the page and will go away". Rows, panels, banners and the rail never cast one.

## Shapes

Gently rounded, small radii. The working radius is 8px (`{rounded.md}`) for buttons, fields, search, nav items, banners, menus, pickers and the toast. Smaller things go smaller: 6px (`{rounded.sm}`) for icon buttons, menu items and the inline rename field; 4px (`{rounded.xs}`) for the focus ring, breadcrumb links and inline code. Floating containers go larger: 12px (`{rounded.lg}`) for dialogs, the upload tray and the drop card. Tags are pills (`{rounded.pill}` on a 20px height). Meters and progress bars are 3 to 4px tall with 2px ends.

Icons are line drawings on a 24-unit grid with round caps and joins: a 1.75 stroke at 20px in chrome, 1.5 at 24px in rows, 1.25 at 40 to 48px. The folder silhouette (a tab at the upper left, 1.5-unit corners) recurs in the folder, shared folder, new folder and move icons and in the app tile. The app tile is a 256-unit rounded square (56-unit corners) with a diagonal blue gradient, a white folder and two circular sync arrows.

## Components

### Buttons
Plain, confident, category-standard.
- **Shape:** gently rounded (8px), 36px tall, 14px side padding, an 18px icon before the label with an 8px gap.
- **Primary:** Dropbox Blue face and border, white label at 550. One per toolbar (Upload, Download, Accept).
- **Secondary (default):** white face, hairline border, ink label. Hover fills Warm Grey; press fills Warm Grey Pressed.
- **Ghost:** no border or face until hover.
- **Danger:** red label on the secondary shape; **Danger Solid** (red face, white label) only to confirm a destructive dialog.
- **Disabled:** 50% opacity.
- **Icon button:** 32px square, 6px radius, Ink Secondary glyph; hover fills Warm Grey Pressed and darkens to Ink.
- **Focus:** a 2px Dropbox Blue outline, offset 2px, 4px radius, on every focusable element.

### Chips
- **Tags:** 20px pills, 12px at 550, beside a file name. Conflict: Amber Wash with Conflict Amber text. Read-only: Warm Grey with Ink Secondary text.

### Cards / Containers
Content is not carded. The table, Activity list, versions list, settings rows and members list are all ruled lists separated by Hairline. The containers that do exist:
- **Details panel:** 320px, white, a hairline on its left, 20px padding; a 40px file glyph and 16px title, a two-column facts list in 13px, actions, then Version history rows with Restore and Download.
- **Banner (invitations):** Selection Blue, 8px radius, 12px by 16px padding, blue icon, message, then actions on the right.
- **Dialog:** white, 12px radius, Float shadow, 480px max (16px from the screen edge on phones), 24px padding with 16px between parts, actions right-aligned in the foot.

### Inputs / Fields
- **Field:** 36px, white, hairline stroke, 8px radius, 10px side padding. Focus swaps the stroke to Dropbox Blue with no glow. Number fields are 72px, right-aligned, tabular, without spinners.
- **Search:** 40px, sits on Warm Grey with a transparent stroke and a leading search icon; on focus it turns white with a blue stroke.
- **Checkbox:** native, 16px, blue accent.
- **Errors:** 13px Danger Red text under the field (planet page); a Red Wash box with red text (client page).

### Navigation
- **Rail:** Warm Grey, 16px by 12px padding. Wordmark (28px tile and "Drive" at 18px/650) on top; nav items are 40px rows with a 20px icon, 12px gap, ink at 500. Hover tints toward Warm Grey Pressed; the current page sits on Warm Grey Pressed at 650 with its icon in blue. The foot holds storage use (12px with a 4px blue meter) and "Get the desktop app".
- **Breadcrumb:** 14px Ink Secondary links separated by 14px chevrons, above the Headline.
- **Tabs (More):** 40px, Ink Secondary at 550 with an icon; the current tab turns Ink with a 2px blue underline sitting on the hairline.
- **Phone:** the rail becomes a drawer that slides in over the page (200ms ease-out) with the Float shadow.

### File table (signature)
The page itself. A 12px/550 header row in Ink Secondary with sortable column buttons and a rotating chevron. 48px rows ruled by Hairline; hover fills Warm Grey; checked rows fill Selection Blue, and a selection bar ("n selected" plus actions) replaces the toolbar. The name cell leads with a 24px glyph: folders blue on a Selection Blue fill, files Ink Secondary, conflicted copies amber on Amber Wash. The row menu (three dots) appears on hover, selection or focus. Rename happens inline in a blue-stroked 32px field. Loading shows pulsing Warm Grey skeleton bars; empty folders show a centred 48px glyph, a Title, a 44ch line and one action.

### Activity list
Grouped under 13px/650 day headers. Each event is a three-column grid: a 24px glyph, the sentence (names in 600), and the time in 13px tabular Ink Secondary, with an optional sub-list of files whose verbs are coloured by state (added green, deleted red).

### Floating feedback
- **Menu:** white, hairline, 8px radius, Float shadow, 6px padding, 200px minimum; 36px items with an 18px Ink Secondary icon; danger items in red; hairline separators.
- **Upload tray:** bottom-left beside the rail, 360px, 12px radius, Float shadow; a header and rows each with a 3px progress bar in blue that turns green when done.
- **Drop zone:** an 8% blue veil with a 2px dashed blue border and a centred white card.
- **Toast:** bottom-centre, Ink surface with Canvas text, 8px radius; red when something failed.

### Tray (desktop client)
The native menu lists recent files, each with a state glyph drawn from the same line-icon geometry and coloured by the state palette: up and down arrows in Dropbox Blue, a check in Done Green, a warning triangle in Conflict Amber, a cross in Danger Red for failures and in Ink Tertiary for deletes. The tray icon is the app tile rasterized per state.

## Do's and Don'ts

### Do:
- **Do** keep the file table as the page: breadcrumb, Headline, toolbar, then the table, with the details panel opening on the right.
- **Do** use `{colors.dropbox-blue}` for exactly one primary button per toolbar or dialog, plus links, selection and focus.
- **Do** separate rows and sections with 1px `{colors.hairline}` rules instead of boxes.
- **Do** draw every icon from the one line-icon set (24-unit grid, round caps and joins, 1.75 stroke at 20px).
- **Do** set sizes, times and counts in tabular figures.
- **Do** define every colour as a token with a dark-scheme value that keeps the warm cast.
- **Do** give every focusable element the 2px blue focus outline.

### Don't:
- **Don't** add a second accent colour, or use amber, red or green for anything but state.
- **Don't** put shadows on rows, panels, the rail or banners; the Float shadow is for things that float.
- **Don't** wrap content in cards or tinted panels; the table and lists sit directly on the canvas.
- **Don't** introduce cool slate greys or pure-neutral greys next to the warm neutrals.
- **Don't** add a display typeface or a size outside the 12/13/14/16/18/24 ramp.
- **Don't** lead with device sync state or Urbit vocabulary (moon, desk, galaxy) in the interface chrome.
