::  %drive-update: facts on /updates. `json` grow lets Eyre channels
::  deliver them to the browser as JSON.
::
/-  drive
/+  lib=drive
|_  upd=update:drive
++  grab
  |%
  ++  noun  update:drive
  --
++  grow
  |%
  ++  noun  upd
  ++  json  (update:enjs:lib upd)
  --
++  grad  %noun
--
