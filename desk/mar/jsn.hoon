::  jsn: json kept as raw bytes
::
::    for large static data files under /web: served as application/json
::    without ever being parsed into a noun on the ship.
::
|_  dat=octs
++  grow
  |%
  ++  mime  [/application/json dat]
  --
++  grab
  |%
  ++  mime  |=([p=mite q=octs] q)
  ++  noun  octs
  --
++  grad  %mime
--
