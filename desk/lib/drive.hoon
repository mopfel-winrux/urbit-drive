::  drive: helpers and json conversions
::
::    NOTE the import is `/-  drive` (with a face), NOT `/-  *drive`.
::
/-  drive
|%
::  +hash-chunk: sha-256 of a chunk, big-endian as an atom
::
++  hash-chunk
  |=  c=chunk:drive
  ^-  hash:drive
  (rev 3 32 (shay len.c dat.c))
::  +hext: 64 hex digits, no dots (the form used in scry paths)
::
++  hext
  |=  h=hash:drive
  ^-  @ta
  (crip ((x-co:co 64) h))
::  +dec: decimal without dots (+scot %ud groups by thousands)
::
++  dec
  |=  a=@ud
  ^-  @ta
  (crip (a-co:co a))
::  +coop: the %shut coop a folder's bindings live under
::
++  coop
  |=  [=fid:drive epoch=@ud]
  ^-  path
  /f/[fid]/(dec epoch)
::  +conflicted: "dir/name (conflicted copy, who, 2026-10-23).ext"
::
++  conflicted
  |=  [p=fpath:drive who=@t when=@da]
  ^-  fpath:drive
  =/  t=tape  (trip p)
  =/  slas  (fand "/" t)
  =/  start=@ud  ?~(slas 0 +((rear slas)))
  ::  a dot that starts the name (.bashrc) isn't an extension
  ::
  =/  dots  (skim (fand "." t) |=(i=@ud (gth i start)))
  =/  dat  (yore when)
  =/  pad  |=(n=@ud ?:((lth n 10) "0{(a-co:co n)}" (a-co:co n)))
  =/  tag=tape
    " (conflicted copy, {(trip who)}, {(a-co:co y.dat)}-{(pad m.dat)}-{(pad d.t.dat)})"
  ?~  dots  (crip (weld t tag))
  =/  at  (rear dots)
  (crip :(weld (scag at t) tag (slag at t)))
::
++  enjs
  =,  enjs:format
  |%
  ++  folder
    |=  [=fid:drive f=folder:drive]
    ^-  json
    %-  pairs
    :~  fid+s+fid
        name+s+name.f
        host+s+(scot %p host.f)
        head+(numb head.f)
        epoch+(numb epoch.f)
        floor+(numb floor.f)
        trash+(numb (lent (skip ~(tap in ~(key by old.f)) |=(p=@t (~(has by tree.f) p)))))
        role+s+role.f
        files+(numb ~(wyt by tree.f))
        bytes+(numb (roll ~(val by tree.f) |=([e=entry:drive n=@ud] (add n size.e))))
      ::
        :-  %members
        :-  %a
        %+  turn  ~(tap by members.f)
        |=  [who=@p r=role:drive]
        (pairs ~[ship+s+(scot %p who) role+s+r])
      ::
        :-  %behind
        ?~  up.f  (numb 0)
        (numb (sub (max goal.u.up.f head.f) head.f))
    ==
  ::
  ::  +old: per path, its versions in history, newest first; a path not
  ::  in the tree is in the trash
  ::
  ++  old
    |=  [t=(map fpath:drive entry:drive) o=(jar fpath:drive version:drive)]
    ^-  json
    :-  %a
    %+  turn  (sort ~(tap by o) |=([a=[p=@t *] b=[p=@t *]] (aor p.a p.b)))
    |=  [p=fpath:drive vs=(list version:drive)]
    %-  pairs
    :~  path+s+p
        deleted+b+!(~(has by t) p)
      ::
        :-  %versions
        :-  %a
        %+  turn  vs
        |=  v=version:drive
        %-  pairs
        :~  aeon+(numb aeon.v)
            when+(time when.v)
            who+s+(scot %p who.v)
            hash+s+(hext hash.entry.v)
            size+(numb size.entry.v)
            mtime+(time mtime.entry.v)
        ==
    ==
  ::
  ++  invite
    |=  [[host=@p =fid:drive] i=invite:drive]
    ^-  json
    (pairs ~[host+s+(scot %p host) fid+s+fid name+s+name.i role+s+role.i])
  ::
  ++  tree
    |=  t=(map fpath:drive entry:drive)
    ^-  json
    :-  %a
    %+  turn  (sort ~(tap by t) |=([a=[p=@t *] b=[p=@t *]] (aor p.a p.b)))
    |=  [p=fpath:drive e=entry:drive]
    %-  pairs
    :~  path+s+p
        hash+s+(hext hash.e)
        size+(numb size.e)
        mtime+(time mtime.e)
    ==
  ::
  ++  device
    |=  [who=@p d=device:drive]
    ^-  json
    %-  pairs
    :~  moon+s+(scot %p who)
        name+s+name.d
        since+(time since.d)
        revoked+?~(revoked.d ~ (time u.revoked.d))
    ==
  ::
  ++  update
    |=  u=update:drive
    ^-  json
    %-  pairs
    ?-  -.u
      %hello   ~[hello+(numb (lent folders.u))]
      %folder  ~[folder+(pairs ~[fid+s+fid.u name+s+name.u head+(numb head.u)])]
      %head    ~[head+(pairs ~[fid+s+fid.u aeon+(numb aeon.u)])]
      %gone    ~[gone+s+fid.u]
    ::
        %result
      =/  ok=?  ?=(%& -.res.u)
      =/  val=json
        ?-  -.res.u
          %&  (numb p.res.u)
          %|  s+p.res.u
        ==
      :~  :-  %result
          (pairs ~[who+s+(scot %p who.u) cid+s+(scot %uv cid.u) ok+b+ok value+val])
      ==
    ==
  --
::
++  dejs
  =,  dejs:format
  |%
  ::  NOTE: switch on the key by hand. `of:dejs` produces a type that does
  ::  not nest into a $% union, so a cast to $-(json action) fails.
  ::
  ++  action
    |=  jon=json
    ^-  action:drive
    ?>  ?=([%o [@ *] ~ ~] jon)
    =*  key  p.n.p.jon
    =*  val  q.n.p.jon
    ?+  key  ~|(bad-key+key !!)
        %pair
      =/  [name=@t moon=@t pass=@t]  ((ot name+so moon+so pass+so ~) val)
      [%pair name (slav %p moon) (rash pass hex)]
    ::
      %unpair      [%unpair (slav %p (so val))]
      %new-folder  [%new-folder (so val)]
      %leave       [%leave (so val)]
      %keep        [%keep (mul ~d1 (ni val))]
      %prune       [%prune ~]
    ::
        %restore
      =/  [=fid:drive p=@t h=@t]  ((ot fid+so path+so hash+so ~) val)
      [%restore fid p (rash h hex)]
    ::
        %share
      =/  [=fid:drive who=@t r=@t]  ((ot fid+so ship+so role+so ~) val)
      [%share fid (slav %p who) ;;(role:drive r)]
    ::
        %unshare
      =/  [=fid:drive who=@t]  ((ot fid+so ship+so ~) val)
      [%unshare fid (slav %p who)]
    ::
        %accept
      =/  [host=@t =fid:drive]  ((ot host+so fid+so ~) val)
      [%accept (slav %p host) fid]
    ::
        %decline
      =/  [host=@t =fid:drive]  ((ot host+so fid+so ~) val)
      [%decline (slav %p host) fid]
    ==
  --
--
