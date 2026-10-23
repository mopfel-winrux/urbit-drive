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
::    Sharing: the host of a folder shares it with other ships, whose
::    %drive mirrors it. A mirror follows the host (/v1/share/<fid>, and
::    the same remote scries a device uses), republishes each wave to its
::    own devices under its own coop, and relays their commits to the
::    host, serving their chunks at its own /up/<hex>. Removing a member
::    or a device moves a folder to a new coop epoch: gall can't drop a
::    coop's key yet, so we publish everything again under a new one.
::
::    pokes:   %drive-action (nouns from devices and ships), %json from
::             the browser
::    watch:   /v1/folders (our devices), /v1/share/<fid> (guests)
::    scries:  /x/version /x/state /x/tree/<fid> /x/pair/<moon> (json)
::             /c/f/<fid>/<epoch>/<ship>, /c/up/<ship>: coop key checks
::
/-  *drive
/+  lib=drive, default-agent, dbug
|%
+$  versioned-state
  $%  state-0
      state-1
  ==
+$  state-1
  $:  %1
      devices=(map ship device)
      folders=(map fid folder)
      blobs=(map hash chunk)
      jobs=(map [ship @uv] job)
      want=(jug hash [ship @uv])              ::  chunks in flight, for jobs
      done=(map [ship @uv] (each aeon @t))    ::  results, for resent commits
      invites=(map [ship fid] invite)
      upped=(set hash)                        ::  %tend'ed at /up, for hosts
  ==
::  state-0: before sharing
::
+$  state-0
  $:  %0
      devices=(map ship device)
      folders=(map fid folder-0)
      blobs=(map hash chunk)
      jobs=(map [ship @uv] job-0)
      want=(jug hash [ship @uv])
      done=(map [ship @uv] (each aeon @t))
  ==
+$  folder-0
  $:  name=@t
      host=ship
      head=aeon
      epoch=@ud
      tree=(map fpath entry)
      log=((mop aeon wave) lth)
      pub=(set hash)
  ==
+$  job-0  [src=ship =commit need=(set hash)]
+$  card  card:agent:gall
++  on-log  ((on aeon wave) lth)
++  max-chunk  0x40.0000                      ::  4 MiB
--
::
=|  state-1
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
  [[[%pass /germ/up %germ /up] cards] this]
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  =/  ver  !<(versioned-state old)
  ?-    -.ver
      %1  [~ this(state ver)]
      %0
    =/  fol
      %-  ~(run by folders.ver)
      |=  f=folder-0
      ^-  folder
      [name host head epoch tree log pub ~ %rw ~]:f
    =/  jos
      %-  ~(run by jobs.ver)
      |=  j=job-0
      ^-  job
      [src.j (scot %p src.j) commit.j need.j]
    :-  [%pass /germ/up %germ /up]~
    this(state [%1 devices.ver fol blobs.ver jos want.ver done.ver ~ ~])
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
  |=  =(pole knot)
  ^-  (quip card _this)
  ?+    pole  (on-watch:def pole)
  ::  our devices: every folder's head, then the results of this
  ::  device's commits (a commit that changed nothing has no wave)
  ::
      [%v1 %folders ~]
    ?>  (allowed:hc src.bowl)
    :_  this
    :_  ~
    %-  fact:hc
    :+  %hello
      (turn ~(tap by folders) |=([=fid f=folder] (folder-info:hc fid f)))
    (results:hc |=(who=ship =(who src.bowl)))
  ::  a guest: the folder's head, and its devices' results
  ::
      [%v1 %share =fid ~]
    =/  f  (~(got by folders) fid.pole)
    ?>  &(=(our.bowl host.f) (~(has by members.f) src.bowl))
    :_  this
    :_  ~
    %-  fact:hc
    :+  %hello
      ~[(folder-info:hc fid.pole f)]
    (results:hc |=(who=ship =(src.bowl (sein:title our.bowl now.bowl who))))
  ==
::
++  on-peek
  |=  =(pole knot)
  ^-  (unit (unit cage))
  ?+    pole  (on-peek:def pole)
      [%x %version ~]
    ``json+!>(`json`(pairs:enjs:format ~[version+s+'0.2.0']))
  ::
      [%x %state ~]
    :^  ~  ~  %json
    !>  ^-  json
    %-  pairs:enjs:format
    :~  ship+s+(scot %p our.bowl)
        devices+a+(turn ~(tap by devices) device:enjs:lib)
        folders+a+(turn ~(tap by folders) folder:enjs:lib)
        invites+a+(turn ~(tap by invites) invite:enjs:lib)
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
          (can-read:hc (slav %p who.pole) fid.pole)
      ==
    ``noun+!>(ok)
  ::  /up holds our devices' chunks for the hosts of our mirrors
  ::
      [%c %up who=@ ~]
    =/  who  (slav %p who.pole)
    ``noun+!>((~(any by folders) |=(f=folder &(?=(^ up.f) =(who host.f)))))
  ==
::
++  on-agent
  |=  [=wire =sign:agent:gall]
  ^-  (quip card _this)
  ?+    wire  (on-agent:def wire sign)
      [%share @ @ ~]
    =^  cards  state
      (take-share:hc (slav %p i.t.wire) i.t.t.wire sign)
    [cards this]
  ::
      [%relay @ @ ~]
    ?.  ?=(%poke-ack -.sign)  `this
    ?~  p.sign  `this
    =/  key  [(slav %p i.t.wire) (slav %uv i.t.t.wire)]
    =^  cards  state  (finish:hc key |+'the host refused the commit')
    [cards this]
  ::
      [%invite *]  `this
      [%uninvite *]  `this
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
      [%pull @ @ @ ~]
    ?>  ?=([%ames %sage *] sign)
    =*  gag  q.sage.sign
    =^  cards  state
      (take-pull:hc i.t.wire i.t.t.wire i.t.t.t.wire ?~(gag ~ `gag))
    [cards this]
  ::
      [%retry @ ~]
    =^  cards  state  (pull:hc i.t.wire)
    [cards this]
  ::
      [%pair *]  `this
  ==
::
++  on-leave  on-leave:def
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
::  +can-read: may [who] have a folder's coop key
::
++  can-read
  |=  [who=ship =fid]
  ^-  ?
  ?:  (allowed who)  &
  ?~  f=(~(get by folders) fid)  |
  &(=(our.bowl host.u.f) (~(has by members.u.f) who))
::
++  fact
  |=  =update
  ^-  card
  [%give %fact ~ %drive-update !>(update)]
::  +give: to our devices, and to a folder's guests
::
++  give
  |=  [=fid =update]
  ^-  card
  [%give %fact ~[/v1/folders /v1/share/[fid]] %drive-update !>(update)]
::
++  folder-info
  |=  [=fid f=folder]
  ^-  info
  [fid name.f host.f epoch.f head.f role.f]
::
++  folder-update
  |=  [=fid f=folder]
  ^-  update
  [%folder (folder-info fid f)]
::
++  results
  |=  ok=$-(ship ?)
  ^-  (list result)
  %+  murn  ~(tap by done)
  |=  [[who=ship cid=@uv] res=(each aeon @t)]
  ?.  (ok who)  ~
  `[who cid res]
::
++  hosted
  |=  =fid
  ^-  folder
  =/  f  (~(got by folders) fid)
  ?>  =(our.bowl host.f)
  f
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
  ::  a revoked device keeps the keys it has: move every folder on
  ::
      %unpair
    ?>  =(our src):bowl
    ?~  d=(~(get by devices) moon.act)  `state
    =.  devices  (~(put by devices) moon.act u.d(revoked `now.bowl))
    =/  fids  ~(tap in ~(key by folders))
    =|  cards=(list card)
    |-  ^-  (quip card _state)
    ?~  fids  [cards state]
    =^  caz  state  (rekey i.fids)
    $(fids t.fids, cards (weld cards caz))
  ::
      %new-folder
    ?>  (allowed src.bowl)
    (new-folder name.act)
  ::
      %commit
    ?>  &(!=(our src):bowl (allowed src.bowl))
    =/  writer=@t
      ?~  d=(~(get by devices) src.bowl)  (scot %p src.bowl)
      name.u.d
    (take-commit src.bowl src.bowl writer commit.act)
  ::
      %share
    ?>  =(our src):bowl
    ?<  =(our.bowl who.act)
    =/  f  (hosted fid.act)
    :_  state(folders (~(put by folders) fid.act f(members (~(put by members.f) who.act role.act))))
    :_  ~
    :*  %pass  /invite/[fid.act]/(scot %p who.act)
        %agent  [who.act %drive]  %poke
        %drive-action  !>(`action`[%invite fid.act name.f role.act])
    ==
  ::
      %unshare
    ?>  =(our src):bowl
    =/  f  (hosted fid.act)
    ?.  (~(has by members.f) who.act)  `state
    =.  folders  (~(put by folders) fid.act f(members (~(del by members.f) who.act)))
    =^  cards  state  (rekey fid.act)
    :_  state
    :+  [%give %kick ~[/v1/share/[fid.act]] `who.act]
      :*  %pass  /uninvite/[fid.act]/(scot %p who.act)
          %agent  [who.act %drive]  %poke
          %drive-action  !>(`action`[%uninvite fid.act])
      ==
    cards
  ::  an invite, or a new role in a folder we already mirror
  ::
      %invite
    ?<  =(our src):bowl
    ?~  f=(~(get by folders) fid.act)
      `state(invites (~(put by invites) [src.bowl fid.act] [name.act role.act]))
    ?>  =(src.bowl host.u.f)
    =.  role.u.f  role.act
    :-  ~[(give fid.act (folder-update fid.act u.f))]
    state(folders (~(put by folders) fid.act u.f))
  ::
      %uninvite
    =.  invites  (~(del by invites) [src.bowl fid.act])
    ?~  f=(~(get by folders) fid.act)  `state
    ?.  &(?=(^ up.u.f) =(src.bowl host.u.f))  `state
    (drop fid.act)
  ::
      %accept
    ?>  =(our src):bowl
    =/  inv  (~(got by invites) [host.act fid.act])
    ?<  (~(has by folders) fid.act)
    =/  =folder
      [(fresh-folder-name name.inv host.act) host.act 0 0 ~ ~ ~ ~ role.inv `[0 0 | ~ ~]]
    :_  %_  state
          folders  (~(put by folders) fid.act folder)
          invites  (~(del by invites) [host.act fid.act])
        ==
    :~  [%pass /germ/[fid.act] %germ (coop:lib fid.act 0)]
        [%pass /share/(scot %p host.act)/[fid.act] %agent [host.act %drive] %watch /v1/share/[fid.act]]
        (give fid.act (folder-update fid.act folder))
    ==
  ::
      %decline
    ?>  =(our src):bowl
    `state(invites (~(del by invites) [host.act fid.act]))
  ::
      %leave
    ?>  =(our src):bowl
    =/  f  (~(got by folders) fid.act)
    ?>  ?=(^ up.f)
    =^  cards  state  (drop fid.act)
    :_  state
    [[%pass /share/(scot %p host.f)/[fid.act] %agent [host.f %drive] %leave ~] cards]
  ::  a guest relays its device's commit
  ::
      %relay
    =/  f  (hosted fid.commit.act)
    ?>  =(%rw (~(gut by members.f) src.bowl %r))
    ?>  |(=(author.act src.bowl) =(src.bowl (sein:title our.bowl now.bowl author.act)))
    (take-commit author.act src.bowl by.act commit.act)
  ==
::
++  new-folder
  |=  name=@t
  ^-  (quip card _state)
  =/  =fid  (crip ((x-co:co 16) (end 6 eny.bowl)))
  =/  =folder  [(fresh-folder-name name our.bowl) our.bowl 0 0 ~ ~ ~ ~ %rw ~]
  :_  state(folders (~(put by folders) fid folder))
  :~  [%pass /germ/[fid] %germ (coop:lib fid 0)]
      (give fid (folder-update fid folder))
  ==
::  +fresh-folder-name: unique among our folders, since devices use the
::  name as the directory
::
++  fresh-folder-name
  |=  [name=@t host=ship]
  ^-  @t
  =/  taken  (silt (turn ~(val by folders) |=(f=folder name.f)))
  =?  name  &((~(has in taken) name) !=(host our.bowl))
    (crip "{(trip name)} ({(scow %p host)})")
  =/  n=@ud  2
  =/  try  name
  |-
  ?.  (~(has in taken) try)  try
  $(n +(n), try (crip "{(trip name)} {(a-co:co n)}"))
::  +drop: forget a mirror; devices keep their copies
::
++  drop
  |=  =fid
  ^-  (quip card _state)
  :-  ~[(give fid [%gone fid])]
  state(folders (~(del by folders) fid))
::  +rekey: publish the folder under a new coop epoch
::
++  rekey
  |=  =fid
  ^-  (quip card _state)
  =/  f  (~(got by folders) fid)
  =.  epoch.f  +(epoch.f)
  =/  coo  (coop:lib fid epoch.f)
  :_  state(folders (~(put by folders) fid f))
  ;:  weld
    [%pass /germ/[fid] %germ coo]~
  ::
    %+  turn  (tap:on-log log.f)
    |=  [a=aeon w=wave]
    ^-  card
    [%pass /tend %tend coo /wave/(dec:lib a) %noun w]
  ::
    %+  turn  ~(tap in pub.f)
    |=  h=hash
    ^-  card
    [%pass /tend %tend coo /blob/(hext:lib h) %noun (~(got by blobs) h)]
  ::
    [(give fid (folder-update fid f))]~
  ==
::  +named: the chunks a commit names
::
++  named
  |=  =commit
  ^-  (list hash)
  =|  out=(list hash)
  |-
  ?~  ops.commit  out
  =/  new  new.i.ops.commit
  $(ops.commit t.ops.commit, out ?~(new out (weld chunks.u.new out)))
::  +take-commit: [author]'s commit, sent by [src] (the author, or the
::  guest relaying for it); settle now, or read the chunks we lack first
::
++  take-commit
  |=  [author=ship src=ship writer=@t =commit]
  ^-  (quip card _state)
  =/  key  [author cid.commit]
  ?^  res=(~(get by done) key)
    [~[(give fid.commit [%result author cid.commit u.res])] state]
  ?:  (~(has by jobs) key)  `state
  ?~  f=(~(get by folders) fid.commit)
    (finish key |+'no such folder')
  ?.  ?=(%rw role.u.f)
    (finish key |+'read-only folder')
  =/  need=(set hash)
    (silt (skip (named commit) |=(h=hash (~(has by blobs) h))))
  ?:  =(~ need)
    (settle key writer commit)
  =/  fresh=(list hash)
    (skip ~(tap in need) |=(h=hash (~(has by want) h)))
  :_  %_  state
        jobs  (~(put by jobs) key [src writer commit need])
        want  (~(gas ju want) (turn ~(tap in need) |=(h=hash [h key])))
      ==
  %+  turn  fresh
  |=  h=hash
  ^-  card
  =/  hx  (hext:lib h)
  [%pass /up/(scot %p src)/[hx] %keen & src ~[%g %x '1' %drive %$ '1' %up hx]]
::
::  +check-chunk: a chunk read from someone, if it is what we asked for
::
++  check-chunk
  |=  [h=hash gag=(unit page)]
  ^-  (unit chunk)
  ?~  gag  ~
  ?~  c=((soft chunk) q.u.gag)  ~
  ?.  ?&  (lte len.u.c max-chunk)
          (lte (met 3 dat.u.c) len.u.c)
          =(h (hash-chunk:lib u.c))
      ==
    ~
  c
::  +take-chunk: a chunk we read for a commit, or its absence
::
++  take-chunk
  |=  [who=ship h=hash gag=(unit page)]
  ^-  (quip card _state)
  =/  keys=(list [ship @uv])  ~(tap in (~(get ju want) h))
  =.  want  (~(del by want) h)
  =/  got  (check-chunk h gag)
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
  =^  caz  state  (settle i.keys [by commit]:u.job)
  $(keys t.keys, cards (weld cards caz))
::
++  finish
  |=  [key=[who=ship cid=@uv] res=(each aeon @t)]
  ^-  (quip card _state)
  =/  =fid  ?~(j=(~(get by jobs) key) %$ fid.commit.u.j)
  :-  ~[(give fid [%result who.key cid.key res])]
  state(jobs (~(del by jobs) key), done (~(put by done) key res))
::  +settle: apply a commit we host; relay one to a mirror's host, with
::  its chunks at our /up for the host to read
::
++  settle
  |=  [key=[who=ship cid=@uv] writer=@t =commit]
  ^-  (quip card _state)
  =/  f  (~(got by folders) fid.commit)
  ?~  up.f  (apply key writer commit)
  =/  new=(list hash)
    ~(tap in (silt (skip (named commit) |=(h=hash (~(has in upped) h)))))
  :_  %_  state
        jobs   (~(del by jobs) key)
        upped  (~(gas in upped) new)
      ==
  %+  snoc
    %+  turn  new
    |=  h=hash
    ^-  card
    [%pass /tend %tend /up /(hext:lib h) %noun (~(got by blobs) h)]
  :*  %pass  /relay/(scot %p who.key)/(scot %uv cid.key)
      %agent  [host.f %drive]  %poke
      %drive-action  !>(`action`[%relay who.key writer commit])
  ==
::  +apply: settle a commit against the head, per file
::
::    An op whose base is the current hash applies. Otherwise the file
::    changed since the writer saw it: an edit beats a delete (either way
::    round), the same contents are no change, and different contents are
::    kept as a conflicted copy beside the current file.
::
++  apply
  |=  [key=[who=ship cid=@uv] writer=@t =commit]
  ^-  (quip card _state)
  =/  fol=folder  (~(got by folders) fid.commit)
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
  =/  =wave  [+(head.fol) who.key now.bowl (flop out)]
  =^  cards  state  (publish fid.commit tre wave)
  =^  caz  state  (finish key &+aeon.wave)
  [(weld cards caz) state]
::  +publish: make [wave] the folder's head, with tree [tre]
::
++  publish
  |=  [=fid tre=(map fpath entry) =wave]
  ^-  (quip card _state)
  =/  fol  (~(got by folders) fid)
  =|  fresh=(set hash)
  =.  fresh
    =/  os=(list op)  ops.wave
    |-  ^-  (set hash)
    ?~  os  fresh
    =/  new  new.i.os
    ?~  new  $(os t.os)
    $(os t.os, fresh (~(gas in fresh) chunks.u.new))
  =/  new-blobs=(list hash)
    (skip ~(tap in fresh) |=(h=hash (~(has in pub.fol) h)))
  =/  coo  (coop:lib fid epoch.fol)
  =.  fol
    %=  fol
      head  aeon.wave
      tree  tre
      log   (put:on-log log.fol aeon.wave wave)
      pub   (~(gas in pub.fol) new-blobs)
    ==
  :_  state(folders (~(put by folders) fid fol))
  ;:  weld
    %+  turn  new-blobs
    |=  h=hash
    ^-  card
    [%pass /tend %tend coo /blob/(hext:lib h) %noun (~(got by blobs) h)]
  ::
    [%pass /tend %tend coo /wave/(dec:lib aeon.wave) %noun wave]~
    [(give fid [%head fid aeon.wave])]~
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
::
::  mirrors: following a folder's host
::
::  +take-share: news from the host of a folder we mirror
::
++  take-share
  |=  [host=ship =fid =sign:agent:gall]
  ^-  (quip card _state)
  ?~  f=(~(get by folders) fid)  `state
  ?.  &(?=(^ up.u.f) =(host host.u.f))  `state
  ?+    -.sign  `state
      %kick
    :_  state
    [%pass /share/(scot %p host)/[fid] %agent [host %drive] %watch /v1/share/[fid]]~
  ::
      %watch-ack
    ?~  p.sign  `state
    ((slog leaf+"drive: lost access to {(trip name.u.f)} on {(scow %p host)}" ~) (drop fid))
  ::
      %fact
    =/  upd  !<(update q.cage.sign)
    ?-    -.upd
        %hello
      =/  i  (skim folders.upd |=(i=info =(fid fid.i)))
      =^  cards  state
        ?~  i  `state
        (host-head fid epoch.i.i head.i.i)
      =/  new=(list result)
        %+  skim  results.upd
        |=  r=result
        &((allowed who.r) !=(`res.r (~(get by done) [who cid]:r)))
      :_  state(done (~(gas by done) (turn new |=(r=result [[who cid]:r res.r]))))
      (weld cards (turn new |=(r=result (give fid [%result r]))))
    ::
      %folder  (host-head fid epoch.upd head.upd)
      %head    (host-head fid epoch.u.up.u.f aeon.upd)
    ::  our devices' commits, settled by the host
    ::
        %result
      ?.  (allowed who.upd)  `state
      :-  ~[(give fid upd)]
      state(done (~(put by done) [who cid]:upd res.upd))
    ::
      %gone  (drop fid)
    ==
  ==
::  +host-head: the host's epoch and head for a mirror
::
++  host-head
  |=  [=fid epoch=@ud head=aeon]
  ^-  (quip card _state)
  =/  f  (~(got by folders) fid)
  ?~  up.f  `state
  =.  folders
    (~(put by folders) fid f(up `u.up.f(epoch epoch, goal (max goal.u.up.f head))))
  (pull fid)
::  +pull: read the host's next wave
::
++  pull
  |=  =fid
  ^-  (quip card _state)
  ?~  f=(~(get by folders) fid)  `state
  ?~  up.u.f  `state
  =/  m  u.up.u.f
  ?:  |(busy.m (gte head.u.f goal.m))  `state
  =/  n  (dec:lib +(head.u.f))
  :_  state(folders (~(put by folders) fid u.f(up `m(busy &, wave ~, need ~))))
  :_  ~
  :*  %pass  /pull/[fid]/wave/[n]  %keen  &  host.u.f
      ~[%g %x '1' %drive %$ '1' %f fid (dec:lib epoch.m) %wave n]
  ==
::  +take-pull: a wave or chunk we read from a host
::
++  take-pull
  |=  [=fid kind=@ta arg=@ta gag=(unit page)]
  ^-  (quip card _state)
  ?~  f=(~(get by folders) fid)  `state
  ?~  up.u.f  `state
  =/  m  u.up.u.f
  ::  on any failure, read the wave again in a while
  ::
  =/  retry
    :-  [%pass /retry/[fid] %arvo %b %wait (add now.bowl ~s5)]~
    state(folders (~(put by folders) fid u.f(up `m(busy |, wave ~, need ~))))
  ?+    kind  `state
      %wave
    ?~  gag  retry
    ?~  w=((soft wave) q.u.gag)  retry
    ?.  =(aeon.u.w +(head.u.f))  retry
    =/  need=(set hash)
      %-  silt
      %+  skip
        =|  out=(list hash)
        =/  os=(list op)  ops.u.w
        |-  ^-  (list hash)
        ?~  os  out
        $(os t.os, out ?~(new.i.os out (weld chunks.u.new.i.os out)))
      |=(h=hash (~(has by blobs) h))
    ?:  =(~ need)
      (take-wave fid u.w)
    :_  state(folders (~(put by folders) fid u.f(up `m(wave w, need need))))
    %+  turn  ~(tap in need)
    |=  h=hash
    ^-  card
    =/  hx  (hext:lib h)
    :*  %pass  /pull/[fid]/blob/[hx]  %keen  &  host.u.f
        ~[%g %x '1' %drive %$ '1' %f fid (dec:lib epoch.m) %blob hx]
    ==
  ::
      %blob
    ?~  wave.m  `state
    =/  h  (rash arg hex)
    ?.  (~(has in need.m) h)  `state
    ?~  c=(check-chunk h gag)  retry
    =.  blobs  (~(put by blobs) h u.c)
    =.  need.m  (~(del in need.m) h)
    ?.  =(~ need.m)
      `state(folders (~(put by folders) fid u.f(up `m)))
    (take-wave fid u.wave.m)
  ==
::  +take-wave: a host's wave, with all its chunks: publish it as ours
::
++  take-wave
  |=  [=fid =wave]
  ^-  (quip card _state)
  =/  f  (~(got by folders) fid)
  =/  tre
    =/  os=(list op)  ops.wave
    =/  t  tree.f
    |-  ^-  (map fpath entry)
    ?~  os  t
    ?~  new.i.os  $(os t.os, t (~(del by t) path.i.os))
    $(os t.os, t (~(put by t) path.i.os u.new.i.os))
  =^  cards  state  (publish fid tre wave)
  =/  f  (~(got by folders) fid)
  ?>  ?=(^ up.f)
  =.  folders  (~(put by folders) fid f(up `u.up.f(busy |, wave ~, need ~)))
  =^  caz  state  (pull fid)
  [(weld cards caz) state]
--
