::  drive: file sync host
::
::    Devices are moons of ours, paired over eyre (%pair). Each folder is
::    a revision log, served to devices by remote scry under a %shut coop
::    (+coop:lib), every binding at revision 1:
::
::      /g/x/1/drive//1/f/<fid>/<epoch>/wave/<aeon>   -> wave
::      /g/x/1/drive//1/f/<fid>/<epoch>/blob/<hex>    -> chunk
::
::    A device commits with a %commit poke naming chunks by hash. We read
::    the ones we lack from its namespace (/up/<hex>, %shut) and apply the
::    commit, settling conflicts per file (see +apply).
::
::    pokes:   %drive-action (nouns from devices), %json from the browser
::    watch:   /v1/folders (%drive-update facts)
::    scries:  /x/version /x/state /x/tree/<fid> /x/pair/<moon> (json)
::             /c/f/<fid>/<epoch>/<ship>: may this ship have the coop key
::
/-  *drive
/+  lib=drive, default-agent, dbug
|%
+$  versioned-state
  $%  state-0
  ==
+$  state-0
  $:  %0
      devices=(map ship device)
      folders=(map fid folder)
      blobs=(map hash chunk)
      jobs=(map [ship @uv] job)
      want=(jug hash [ship @uv])              ::  chunks in flight, for jobs
      done=(map [ship @uv] (each aeon @t))    ::  results, for resent commits
  ==
+$  card  card:agent:gall
++  on-log  ((on aeon wave) lth)
++  max-chunk  0x40.0000                      ::  4 MiB
--
::
=|  state-0
=*  state  -
%-  agent:dbug
^-  agent:gall
=<
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %.n) bowl)
    hc    ~(. +> bowl)
::
++  on-init
  ^-  (quip card _this)
  =^  cards  state  (new-folder:hc 'Drive')
  [cards this]
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  =/  ver  !<(versioned-state old)
  ?-  -.ver
    %0  [~ this(state ver)]
  ==
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  =/  act=action
    ?+  mark  ~|(bad-mark+mark !!)
      %drive-action  !<(action vase)
      %json          (action:dejs:lib !<(json vase))
    ==
  =^  cards  state  (poke:hc act)
  [cards this]
::
++  on-watch
  |=  =path
  ^-  (quip card _this)
  ?+  path  (on-watch:def path)
      [%v1 %folders ~]
    ?>  (allowed:hc src.bowl)
    :_  this
    %+  turn  ~(tap by folders)
    |=  [=fid f=folder]
    ^-  card
    [%give %fact ~ %drive-update !>(`update`[%folder fid name.f host.f epoch.f head.f])]
  ==
::
++  on-peek
  |=  =(pole knot)
  ^-  (unit (unit cage))
  ?+    pole  (on-peek:def pole)
      [%x %version ~]
    ``json+!>(`json`(pairs:enjs:format ~[version+s+'0.1.0']))
  ::
      [%x %state ~]
    :^  ~  ~  %json
    !>  ^-  json
    %-  pairs:enjs:format
    :~  ship+s+(scot %p our.bowl)
        devices+a+(turn ~(tap by devices) device:enjs:lib)
        folders+a+(turn ~(tap by folders) folder:enjs:lib)
    ==
  ::
      [%x %tree fid=@ ~]
    ?~  f=(~(get by folders) fid.pole)  [~ ~]
    ``json+!>((tree:enjs:lib tree.u.f))
  ::
      [%x %pair moon=@ ~]
    =/  moon  (slav %p moon.pole)
    ``json+!>(`json`(pairs:enjs:format ~[paired+b+(allowed:hc moon)]))
  ::
      [%c %f fid=@ ep=@ who=@ ~]
    =/  ok=?
      ?~  f=(~(get by folders) fid.pole)  |
      ?&  =((dec:lib epoch.u.f) ep.pole)
          (allowed:hc (slav %p who.pole))
      ==
    ``noun+!>(ok)
  ==
::
++  on-arvo
  |=  [=wire sign=sign-arvo]
  ^-  (quip card _this)
  ?+    wire  (on-arvo:def wire sign)
      [%up @ @ ~]
    ?>  ?=([%ames %sage *] sign)
    =*  gag  q.sage.sign
    =^  cards  state
      %^  take-chunk:hc  (slav %p i.t.wire)  (rash i.t.t.wire hex)
      ?~(gag ~ `gag)
    [cards this]
  ::
      [%pair *]  `this
  ==
::
++  on-leave  on-leave:def
++  on-agent  on-agent:def
++  on-fail   on-fail:def
--
::
|_  =bowl:gall
::  +allowed: our own ship and its paired (not revoked) devices
::
++  allowed
  |=  who=ship
  ^-  ?
  ?:  =(our.bowl who)  &
  ?~  d=(~(get by devices) who)  |
  =(~ revoked.u.d)
::
++  give
  |=  =update
  ^-  card
  [%give %fact ~[/v1/folders] %drive-update !>(update)]
::
++  poke
  |=  act=action
  ^-  (quip card _state)
  ?-    -.act
      %pair
    ?>  =(our src):bowl
    ?>  =(%earl (clan:title moon.act))
    ?>  =(our.bowl (sein:title our.bowl now.bowl moon.act))
    ?<  (~(has by devices) moon.act)
    ::  fake ships derive everyone's keys from their names (and jael
    ::  has a rift for every ship), so skip the check and use those
    ::
    =/  fak  .^(? %j /(scot %p our.bowl)/fake/(scot %da now.bowl))
    ?>  ?|  fak
            =(~ .^((unit rift) %j /(scot %p our.bowl)/ryft/(scot %da now.bowl)/(scot %p moon.act)))
        ==
    =/  pas=pass
      ?.  fak  pass.act
      pub:ex:(pit:nu:cric:crypto 512 moon.act %b ~)
    :_  state(devices (~(put by devices) moon.act [name.act pas now.bowl ~]))
    :_  ~
    :*  %pass  /pair/(scot %p moon.act)  %arvo  %j
        %moon  moon.act  *id:block:jael  %keys  [1 1 pas]  %.n
    ==
  ::
      %unpair
    ?>  =(our src):bowl
    ?~  d=(~(get by devices) moon.act)  `state
    `state(devices (~(put by devices) moon.act u.d(revoked `now.bowl)))
  ::
      %new-folder
    ?>  (allowed src.bowl)
    (new-folder name.act)
  ::
      %commit
    ?>  &(!=(our src):bowl (allowed src.bowl))
    (take-commit commit.act)
  ==
::
++  new-folder
  |=  name=@t
  ^-  (quip card _state)
  =/  =fid  (crip ((x-co:co 16) (end 6 eny.bowl)))
  =/  =folder  [name our.bowl 0 0 ~ ~ ~]
  :_  state(folders (~(put by folders) fid folder))
  :~  [%pass /germ/[fid] %germ (coop:lib fid 0)]
      (give [%folder fid name our.bowl 0 0])
  ==
::  +take-commit: apply now, or fetch the chunks we lack first
::
++  take-commit
  |=  =commit
  ^-  (quip card _state)
  =/  key  [src.bowl cid.commit]
  ?^  res=(~(get by done) key)
    [~[(give [%result src.bowl cid.commit u.res])] state]
  ?:  (~(has by jobs) key)  `state
  ?.  (~(has by folders) fid.commit)
    (finish key |+'no such folder')
  =|  named=(list hash)
  =.  named
    |-  ^-  (list hash)
    ?~  ops.commit  named
    =/  new  new.i.ops.commit
    $(ops.commit t.ops.commit, named ?~(new named (weld chunks.u.new named)))
  =/  need=(set hash)
    (silt (skip named |=(h=hash (~(has by blobs) h))))
  ?:  =(~ need)
    (apply key commit)
  =/  fresh=(list hash)
    (skip ~(tap in need) |=(h=hash (~(has by want) h)))
  :_  %_  state
        jobs  (~(put by jobs) key [src.bowl commit need])
        want  (~(gas ju want) (turn ~(tap in need) |=(h=hash [h key])))
      ==
  %+  turn  fresh
  |=  h=hash
  ^-  card
  =/  hx  (hext:lib h)
  [%pass /up/(scot %p src.bowl)/[hx] %keen & src.bowl ~[%g %x '1' %drive %$ '1' %up hx]]
::  +take-chunk: a chunk we asked a device for, or its absence
::
++  take-chunk
  |=  [who=ship h=hash gag=(unit page)]
  ^-  (quip card _state)
  =/  keys=(list [ship @uv])  ~(tap in (~(get ju want) h))
  =.  want  (~(del by want) h)
  =/  got=(unit chunk)
    ?~  gag  ~
    ?~  c=((soft chunk) q.u.gag)  ~
    ?.  ?&  (lte len.u.c max-chunk)
            (lte (met 3 dat.u.c) len.u.c)
            =(h (hash-chunk:lib u.c))
        ==
      ~
    c
  =?  blobs  ?=(^ got)  (~(put by blobs) h u.got)
  =|  cards=(list card)
  |-  ^-  (quip card _state)
  ?~  keys  [cards state]
  ?~  job=(~(get by jobs) i.keys)  $(keys t.keys)
  ?~  got
    =^  caz  state  (finish i.keys |+(crip "chunk {(trip (hext:lib h))} unavailable"))
    $(keys t.keys, cards (weld cards caz))
  =.  need.u.job  (~(del in need.u.job) h)
  ?.  =(~ need.u.job)
    $(keys t.keys, jobs (~(put by jobs) i.keys u.job))
  =^  caz  state  (apply i.keys commit.u.job)
  $(keys t.keys, cards (weld cards caz))
::
++  finish
  |=  [key=[who=ship cid=@uv] res=(each aeon @t)]
  ^-  (quip card _state)
  :-  ~[(give [%result who.key cid.key res])]
  state(jobs (~(del by jobs) key), done (~(put by done) key res))
::  +apply: settle a commit against the head, per file
::
::    An op whose base is the current hash applies. Otherwise the file
::    changed since the writer saw it: an edit beats a delete (either way
::    round), the same contents are no change, and different contents are
::    kept as a conflicted copy beside the current file.
::
++  apply
  |=  [key=[who=ship cid=@uv] =commit]
  ^-  (quip card _state)
  =/  fol=folder  (~(got by folders) fid.commit)
  =/  writer=@t
    ?~  d=(~(get by devices) who.key)  (scot %p who.key)
    name.u.d
  =/  tre  tree.fol
  =/  cops  ops.commit
  =|  out=(list op)
  |-
  ?^  cops
    =*  c  i.cops
    =/  cur  (~(get by tre) path.c)
    =/  had=(unit hash)  (bind cur |=(e=entry hash.e))
    ?:  =(had base.c)
      ?~  new.c
        ?~  cur  $(cops t.cops)
        $(cops t.cops, tre (~(del by tre) path.c), out [[path.c ~] out])
      ?:  =(u.new.c (fall cur *entry))  $(cops t.cops)
      $(cops t.cops, tre (~(put by tre) path.c u.new.c), out [[path.c new.c] out])
    ?~  new.c  $(cops t.cops)
    ?~  cur
      $(cops t.cops, tre (~(put by tre) path.c u.new.c), out [[path.c new.c] out])
    ?:  =(hash.u.cur hash.u.new.c)  $(cops t.cops)
    =/  alt  (fresh-name tre (conflicted:lib path.c writer now.bowl))
    $(cops t.cops, tre (~(put by tre) alt u.new.c), out [[alt new.c] out])
  ?~  out
    (finish key &+head.fol)
  =/  =aeon  +(head.fol)
  =/  =wave  [aeon who.key now.bowl (flop out)]
  =|  fresh=(set hash)
  =.  fresh
    =/  os=(list op)  out
    |-  ^-  (set hash)
    ?~  os  fresh
    =/  new  new.i.os
    ?~  new  $(os t.os)
    $(os t.os, fresh (~(gas in fresh) chunks.u.new))
  =/  new-blobs=(list hash)
    (skip ~(tap in fresh) |=(h=hash (~(has in pub.fol) h)))
  =/  coo  (coop:lib fid.commit epoch.fol)
  =.  fol
    %=  fol
      head  aeon
      tree  tre
      log   (put:on-log log.fol aeon wave)
      pub   (~(gas in pub.fol) new-blobs)
    ==
  =^  caz  state  (finish key &+aeon)
  :_  state(folders (~(put by folders) fid.commit fol))
  ;:  weld
    %+  turn  new-blobs
    |=  h=hash
    ^-  card
    [%pass /tend %tend coo /blob/(hext:lib h) %noun (~(got by blobs) h)]
  ::
    [%pass /tend %tend coo /wave/(dec:lib aeon) %noun wave]~
    [(give [%head fid.commit aeon]) caz]
  ==
::
++  fresh-name
  |=  [tre=(map fpath entry) p=fpath]
  ^-  fpath
  =/  n=@ud  1
  =/  q  p
  |-
  ?.  (~(has by tre) q)  q
  $(n +(n), q (crip "{(trip p)} {(a-co:co +(n))}"))
--
