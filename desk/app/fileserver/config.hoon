::  configuration for the %drive-fileserver agent (see app/drive-fileserver.hoon)
::
|%
::  serve /web from this desk under /drive
::
++  web-root  ^-  (list @t)  /drive
::  extensionless urls fall back to the index file,
::  so both /drive and /drive/ serve /web/index.html
::
++  extension  %fall
::  local login required (set to | to make the files public)
::
++  auth  ^-  ?  &
--
