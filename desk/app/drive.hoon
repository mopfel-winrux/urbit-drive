::  drive: file sync host
::
::    Devices are moons of ours, paired over eyre (%pair). Each folder is
::    a revision log, served to devices by remote scry under a %shut coop
::    (+coop:lib), every binding at revision 1:
::
::      /g/x/1/drive//1/f/<fid>/<epoch>/wave/<aeon>   -> wave
::      /g/x/1/drive//1/f/<fid>/<epoch>/rock/<floor>  -> (list [fpath entry])
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
::    host, serving their chunks at its own /up/<hex>.
::
::    Retention: waves older than [keep] are folded into the folder's
::    base, published as a rock at its floor; replaced and deleted files
::    stay restorable for [keep]; chunks nothing refers to are dropped.
::    Readers always ask for revision 1, so nothing is ever rebound:
::    instead a folder moves to a new coop epoch (+rekey), republishing
::    what's live and tombing everything under the old one. Removing a
::    member or a device does the same, so old keys read nothing.
::
::    pokes:   %drive-action (nouns from devices and ships), %json from
::             the browser
::    watch:   /v1/folders (our devices), /v1/share/<fid> (guests)
::    Share links: /drive-api/s/<token> serves a file or folder to anyone
::    with the link (and its password, if it has one); see +share-http.
::
::    scries:  /x/version /x/state /x/tree/<fid> /x/old/<fid> /x/links
::             /x/pair/<moon> (json), /x/file/<fid>/<hex> (mime)
::             /c/f/<fid>/<epoch>/<ship>, /c/up/<ship>: coop key checks
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
      invites=(map [ship fid] invite)
      upped=(set hash)                        ::  %tend'ed at /up, for hosts
      keep=@dr                                ::  how long history lasts
      links=(map @t link)                     ::  share links, by token
      unlocked=(map @uv [token=@t until=@da]) ::  password-unlocked visitors
  ==
+$  card  card:agent:gall
++  on-log  ((on aeon wave) lth)
++  max-chunk  0x40.0000                      ::  4 MiB
++  prune-every  ~h6
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
  =.  keep  ~d30
  =^  cards  state  (new-folder:hc 'Drive')
  :_  this
  :*  [%pass /germ/up %germ /up]
      [%pass /prune %arvo %b %wait (add now.bowl prune-every)]
      [%pass /eyre %arvo %e %connect [~ /drive-api] %drive]
      cards
  ==
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  =/  ver  !<(versioned-state old)
  ?-    -.ver
      %0
    :_  this(state ver)
    [%pass /eyre %arvo %e %connect [~ /drive-api] %drive]~
  ==
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  ?:  ?=(%handle-http-request mark)
    =^  cards  state  (serve-http:hc !<([@ta inbound-request:eyre] vase))
    [cards this]
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
      [%http-response @ ~]
    ::  eyre, for a request we answer; a share link's visitor is a guest
    ::
    `this
  ::  our devices: every folder, and the results of this device's
  ::  commits (a commit that changed nothing has no wave), in one fact
  ::
      [%v1 %folders ~]
    ?>  (allowed:hc src.bowl)
    :_  this
    :_  ~
    %-  fact:hc
    :+  %hello
      (turn ~(tap by folders) |=([=fid f=folder] (folder-info:hc fid f)))
    (results:hc |=(who=ship =(who src.bowl)))
  ::  a guest: the folder, and its devices' results
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
    ``json+!>(`json`(pairs:enjs:format ~[version+s+'0.4.0']))
  ::
      [%x %state ~]
    =/  bytes  (roll ~(val by blobs) |=([c=chunk n=@ud] (add n len.c)))
    :^  ~  ~  %json
    !>  ^-  json
    %-  pairs:enjs:format
    :~  ship+s+(scot %p our.bowl)
        devices+a+(turn ~(tap by devices) device:enjs:lib)
        folders+a+(turn ~(tap by folders) folder:enjs:lib)
        invites+a+(turn ~(tap by invites) invite:enjs:lib)
        keep+(numb:enjs:format (div keep ~d1))
        :-  %stored
        (pairs:enjs:format ~[chunks+(numb:enjs:format ~(wyt by blobs)) bytes+(numb:enjs:format bytes)])
    ==
  ::
      [%x %tree fid=@ ~]
    ?~  f=(~(get by folders) fid.pole)  [~ ~]
    ``json+!>((tree:enjs:lib tree.u.f))
  ::
  ::  the latest changes across folders, newest first
  ::
      [%x %activity ~]
    =/  all=(list [=fid name=@t f=folder =wave])
      %-  zing
      %+  turn  ~(tap by folders)
      |=  [=fid f=folder]
      (turn (tap:on-log log.f) |=([a=aeon w=wave] [fid name.f f w]))
    =/  top
      %+  scag  100
      (sort all |=([a=[@ @ * w=wave] b=[@ @ * w=wave]] (gth when.w.a when.w.b)))
    :^  ~  ~  %json
    !>  ^-  json
    :-  %a
    %+  turn  top
    |=  [=fid name=@t f=folder =wave]
    =/  by=@t
      ?~  d=(~(get by devices) author.wave)  (scot %p author.wave)
      name.u.d
    (activity:enjs:lib fid name by f wave)
  ::
      [%x %links ~]
    :^  ~  ~  %json
    !>  ^-  json
    :-  %a
    %+  turn  ~(tap by links)
    |=  [tok=@t l=link]
    =/  f  (~(get by folders) fid.l)
    =/  sc  ?~(f ~ (scope u.f path.l))
    (link:enjs:lib tok l ?~(f '' name.u.f) ?~(sc '' kind.u.sc))
  ::
      [%x %old fid=@ ~]
    ?~  f=(~(get by folders) fid.pole)  [~ ~]
    ``json+!>((old:enjs:lib tree.u.f old.u.f))
  ::  a file's contents, by its hash: current, or a version in history
  ::
      [%x %file fid=@ hx=@ ~]
    ?~  f=(~(get by folders) fid.pole)  [~ ~]
    ?~  e=(find-entry:hc u.f (rash hx.pole hex))  [~ ~]
    ?~  dat=(file-bytes:hc u.e 0 size.u.e)  [~ ~]
    ``mime+!>(`mime`[/application/octet-stream u.dat])
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
      [%pull @ @ @ @ ~]
    ?>  ?=([%ames %sage *] sign)
    =*  gag  q.sage.sign
    =^  cards  state
      (take-pull:hc i.t.wire i.t.t.wire [i.t.t.t i.t.t.t.t]:wire ?~(gag ~ `gag))
    [cards this]
  ::
      [%retry @ ~]
    =^  cards  state  (pull:hc i.t.wire)
    [cards this]
  ::
      [%prune ~]
    =^  cards  state  prune-all:hc
    :_  this
    [[%pass /prune %arvo %b %wait (add now.bowl prune-every)] cards]
  ::
      [%pair *]  `this
      [%eyre ~]  `this
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
++  tend
  |=  [coo=path =path =noun]
  ^-  card
  [%pass /tend %tend coo path %noun noun]
::
++  tomb
  |=  =spur
  ^-  card
  [%pass /tomb %tomb [%ud 1] spur]
::
++  folder-info
  |=  [=fid f=folder]
  ^-  info
  [fid name.f host.f epoch.f head.f role.f floor.f]
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
    =/  f  (~(got by folders) i.fids)
    =^  caz  state  (rekey i.fids ~ pub.f floor.f)
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
    =^  cards  state  (rekey fid.act ~ pub.f floor.f)
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
      :*  (fresh-folder-name name.inv host.act)  host.act  0  0  ~  ~  ~  ~
          role.inv  `[0 0 0 | ~ ~ ~]  0  ~  ~
      ==
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
  ::  an old version back in place, by an ordinary commit from us
  ::
      %restore
    ?>  =(our src):bowl
    =/  f  (~(got by folders) fid.act)
    =/  vs  (~(get ja old.f) path.act)
    ?~  hit=(skim vs |=(v=version =(hash.act hash.entry.v)))
      ~|(no-such-version+path.act !!)
    =/  cur  (bind (~(get by tree.f) path.act) |=(e=entry hash.e))
    =/  =commit
      [(end 6 eny.bowl) fid.act head.f [path.act cur `entry.i.hit]~]
    (take-commit our.bowl our.bowl 'restore' commit)
  ::
      %keep
    ?>  =(our src):bowl
    ?>  (gte keep.act ~m1)
    `state(keep keep.act)
  ::
      %prune
    ?>  =(our src):bowl
    prune-all
  ::
      %put
    ?>  =(our src):bowl
    =/  f  (~(got by folders) fid.act)
    ?>  (levy chunks.entry.act |=(h=hash (~(has by blobs) h)))
    =/  cur  (bind (~(get by tree.f) path.act) |=(e=entry hash.e))
    %^  take-commit  our.bowl  our.bowl
    :+  'web'  (end 6 eny.bowl)
    [fid.act head.f [path.act cur `entry.act]~]
  ::
      %del
    ?>  =(our src):bowl
    =/  f  (~(got by folders) fid.act)
    =/  ops=(list cop)
      %+  murn  ~(tap by tree.f)
      |=  [p=fpath e=entry]
      ?.  (under p path.act)  ~
      `[p `hash.e ~]
    ?~  ops  `state
    (take-commit our.bowl our.bowl 'web' [(end 6 eny.bowl) fid.act head.f ops])
  ::
      %mv
    ?>  =(our src):bowl
    ?<  =(from.act to.act)
    ?<  (under to.act from.act)                       ::  into itself
    =/  f  (~(got by folders) fid.act)
    =/  ops=(list cop)
      %-  zing
      %+  murn  ~(tap by tree.f)
      |=  [p=fpath e=entry]
      ^-  (unit (list cop))
      ?.  (under p from.act)  ~
      =/  q  (cat 3 to.act (rsh [3 (met 3 from.act)] p))
      ?<  (~(has by tree.f) q)
      `~[[p `hash.e ~] [q ~ `e]]
    ?~  ops  `state
    (take-commit our.bowl our.bowl 'web' [(end 6 eny.bowl) fid.act head.f ops])
  ::
      %link-set
    ?>  =(our src):bowl
    ?>  (good-token token.act)
    =/  f  (hosted fid.act)
    ?>  ?=(^ (scope f path.act))
    =/  old  (~(get by links) token.act)
    ?>  |(?=(~ old) &(=(fid.u.old fid.act) =(path.u.old path.act)))
    =/  pass=(unit [salt=@ hash=@])
      ?-  pass.act
        %keep  ?~(old ~ pass.u.old)
        %none  ~
        [%set *]  =/  salt  (end 7 eny.bowl)  `[salt (hash-pass salt p.pass.act)]
      ==
    =/  =link
      ?~  old  [fid.act path.act now.bowl expires.act pass 0 0]
      u.old(expires expires.act, pass pass)
    `state(links (~(put by links) token.act link))
  ::
      %link-drop
    ?>  =(our src):bowl
    `state(links (~(del by links) token.act))
  ::  a device wants a link: reuse a plain working one, or make this one
  ::
      %link-ask
    ?>  &(!=(our src):bowl (allowed src.bowl))
    =/  answer
      |=  res=(each @t @t)
      ^-  card
      (give fid.act [%link src.bowl token.act res])
    ?.  (good-token token.act)  [~[(answer |+'bad token')] state]
    ?~  f=(~(get by folders) fid.act)  [~[(answer |+'That folder isn’t on your ship.')] state]
    ?.  =(our.bowl host.u.f)
      [~[(answer |+'Only folders your ship hosts can have links. Ask the folder’s owner.')] state]
    ?~  (scope u.f path.act)  [~[(answer |+'That isn’t on your ship yet. Wait for it to sync, then try again.')] state]
    =/  have=(list @t)
      %+  murn  ~(tap by links)
      |=  [t=@t l=link]
      ?.  &(=(fid.l fid.act) =(path.l path.act) ?=(~ pass.l) ?=(~ expires.l))  ~
      `t
    ?^  have  [~[(answer &+i.have)] state]
    :-  ~[(answer &+token.act)]
    state(links (~(put by links) token.act [fid.act path.act now.bowl ~ ~ 0 0]))
  ::
      %mkdir
    ?>  =(our src):bowl
    =/  f  (~(got by folders) fid.act)
    =/  p  (cat 3 path.act '/')
    ?<  (~(has by tree.f) p)
    =/  =entry  [`@ux`0 0 now.bowl | ~]
    (take-commit our.bowl our.bowl 'web' [(end 6 eny.bowl) fid.act head.f [p ~ `entry]~])
  ==
::
::  +under: [p] is [pre], or inside the folder [pre]
::
++  under
  |=  [p=fpath pre=fpath]
  ^-  ?
  ?|  =(p pre)
      =((cat 3 pre '/') (end [3 +((met 3 pre))] p))
  ==
::  +serve-http: POST /drive-api/blob takes one upload chunk's raw bytes
::  and answers its hash
::
++  serve-http
  |=  [eid=@ta req=inbound-request:eyre]
  ^-  (quip card _state)
  =/  pax  /http-response/[eid]
  =/  reply
    |=  [code=@ud body=@t]
    ^-  (list card)
    :~  [%give %fact ~[pax] %http-response-header !>(`response-header:http`[code ['content-type' 'application/json']~])]
        [%give %fact ~[pax] %http-response-data !>(`(unit octs)``(as-octs:mimes:html body))]
        [%give %kick ~[pax] ~]
    ==
  =/  [base=@t qs=(list [p=@t q=@t])]  (split-url url.request.req)
  =/  pax=(unit path)  (rush base stap)
  ?:  ?=([~ %drive-api %s @ *] pax)
    (share-http eid req i.t.t.u.pax t.t.t.u.pax qs)
  ::  the file viewer, public: share pages load it too
  ::
  ?:  =('/drive-api/preview.js' base)
    =/  js  .^(@t %cx /(scot %p our.bowl)/[q.byk.bowl]/(scot %da now.bowl)/web/preview/js)
    :_  state
    :~  [%give %fact ~[/http-response/[eid]] %http-response-header !>(`response-header:http`[200 ~[['content-type' 'text/javascript; charset=utf-8'] ['cache-control' 'no-cache']]])]
        [%give %fact ~[/http-response/[eid]] %http-response-data !>(`(unit octs)``(as-octs:mimes:html js))]
        [%give %kick ~[/http-response/[eid]] ~]
    ==
  ?.  authenticated.req  [(reply 403 '{"error":"log in first"}') state]
  ?:  ?=([~ %drive-api %file @ @ ~] pax)
    (file-http eid req i.t.t.u.pax i.t.t.t.u.pax qs)
  ?.  &(=(%'POST' method.request.req) =('/drive-api/blob' base))
    [(reply 404 '{"error":"not found"}') state]
  ?~  body.request.req  [(reply 400 '{"error":"empty body"}') state]
  =/  c=chunk  [p q]:u.body.request.req
  ?.  (lte len.c max-chunk)  [(reply 413 '{"error":"chunk too large"}') state]
  =/  h  (hash-chunk:lib c)
  :-  (reply 200 (crip "\{\"hash\":\"{(trip (hext:lib h))}\"}"))
  state(blobs (~(put by blobs) h c))
::  +good-token: 32 lowercase hex digits (128 bits)
::
++  good-token
  |=  t=@t
  ^-  ?
  ?&  =(32 (met 3 t))
      %+  levy  (trip t)
      |=(c=@t |(&((gte c '0') (lte c '9')) &((gte c 'a') (lte c 'f'))))
  ==
::  +split-url: an eyre url into its path and decoded query
::
++  split-url
  |=  url=@t
  ^-  [@t (list [p=@t q=@t])]
  =/  t  (trip url)
  =/  i  (find "?" t)
  ?~  i  [url ~]
  :-  (crip (scag u.i t))
  (fall (rush (crip (slag u.i t)) yque:de-purl:html) ~)
::  +hash-pass: a share link's password, salted and stretched
::
++  hash-pass
  |=  [salt=@ pw=@t]
  ^-  @
  =/  h  (shax (jam [salt pw]))
  =/  n=@ud  0
  |-
  ?:  =(n 4.096)  h
  $(n +(n), h (shax (cat 3 h salt)))
::  +scope: what a link to [p] in [f] covers, as paths relative to it:
::  a file is just itself; a folder ('' for all) is everything in it.
::  ~ when nothing is there any more.
::
++  scope
  |=  [f=folder p=fpath]
  ^-  (unit [kind=?(%file %dir) es=(list [rel=@t e=entry])])
  ?:  &(!=('' p) (~(has by tree.f) p) !=('/' (rsh [3 (sub (met 3 p) 1)] p)))
    `[%file [(rear (rash p (more fas (cook crip (star ;~(less fas next)))))) (~(got by tree.f) p)]~]
  =/  pre=@t  ?:(=('' p) '' (cat 3 p '/'))
  =/  n  (met 3 pre)
  =/  es
    %+  murn  ~(tap by tree.f)
    |=  [q=fpath e=entry]
    ?.  =(pre (end [3 n] q))  ~
    =/  rel  (rsh [3 n] q)
    ?:(=('' rel) ~ `[rel e])
  ?:  &(?=(~ es) !=('' p))  ~
  `[%dir es]
::  +share-http: a share link's page and its JSON, unlock and file routes
::
++  share-http
  |=  [eid=@ta req=inbound-request:eyre tok=@t rest=path qs=(list [p=@t q=@t])]
  ^-  (quip card _state)
  =/  pax  /http-response/[eid]
  =/  respond
    |=  [code=@ud hed=(list [key=@t value=@t]) body=(unit octs)]
    ^-  (list card)
    =/  safe  ~[['x-content-type-options' 'nosniff'] ['referrer-policy' 'no-referrer'] ['cache-control' 'no-store']]
    :~  [%give %fact ~[pax] %http-response-header !>(`response-header:http`[code (weld hed safe)])]
        [%give %fact ~[pax] %http-response-data !>(body)]
        [%give %kick ~[pax] ~]
    ==
  =/  jsn
    |=  [code=@ud j=json]
    (respond code ['content-type' 'application/json']~ `(as-octs:mimes:html (en:json:html j)))
  =/  err
    |=  [code=@ud msg=@t]
    (jsn code (pairs:enjs:format ~[error+s+msg]))
  ::  the page itself: always served; it asks /info what the link is
  ::
  ?~  rest
    =/  page  .^(@t %cx /(scot %p our.bowl)/[q.byk.bowl]/(scot %da now.bowl)/web/share/html)
    :_  state
    (respond 200 ['content-type' 'text/html; charset=utf-8']~ `(as-octs:mimes:html page))
  =/  lin  (~(get by links) tok)
  =/  gone=?  |(?=(~ lin) &(?=(^ expires.u.lin) (gth now.bowl u.expires.u.lin)))
  ?:  gone  [(err 404 'It may have expired, or whoever shared it turned it off. Ask them for a new link.') state]
  ?>  ?=(^ lin)
  =/  l  u.lin
  =/  f  (~(get by folders) fid.l)
  =/  sc  ?~(f ~ (scope u.f path.l))
  ?~  sc  [(err 410 'The file or folder behind this link is no longer there.') state]
  ?>  ?=(^ f)
  =/  name=@t
    ?:  =('' path.l)  name.u.f
    (rear (rash path.l (more fas (cook crip (star ;~(less fas next))))))
  ::  a password-protected link needs this visitor's unlock cookie
  ::
  =/  open=?
    ?~  pass.l  &
    =/  ck  (get-cookie req 'drive-share')
    ?~  ck  |
    ?~  key=(slaw %uv u.ck)  |
    ?~  u=(~(get by unlocked) u.key)  |
    &(=(token.u.u tok) (gth until.u.u now.bowl))
  ?+    rest  [(err 404 'Not found.') state]
      [%info ~]
    =/  base=(list [@t json])
      :~  name+s+name
          kind+s+kind.u.sc
          owner+s+(scot %p our.bowl)
          locked+b+!open
          expires+?~(expires.l ~ (time:enjs:format u.expires.l))
      ==
    ?.  open  [(jsn 200 (pairs:enjs:format base)) state]
    :-  %+  jsn  200
        %-  pairs:enjs:format
        %+  snoc  base
        :-  %files
        :-  %a
        %+  turn  es.u.sc
        |=  [rel=@t e=entry]
        (pairs:enjs:format ~[path+s+rel size+(numb:enjs:format size.e) mtime+(time:enjs:format mtime.e)])
    state(links (~(put by links) tok l(views +(views.l))))
  ::
      [%unlock ~]
    ?.  =(%'POST' method.request.req)  [(err 405 'Use POST.') state]
    ?~  pass.l  [(err 400 'This link has no password.') state]
    =/  pw=@t  ?~(body.request.req '' q.u.body.request.req)
    ?.  =(hash.u.pass.l (hash-pass salt.u.pass.l pw))
      [(err 403 'That password isn’t right.') state]
    =/  key=@uv  (end 7 eny.bowl)
    =/  cookie=@t
      %-  crip
      ;:  weld
        "drive-share={(scow %uv key)}; Path=/drive-api/s/{(trip tok)}; HttpOnly; SameSite=Lax; Max-Age=86400"
        ?:(secure.req "; Secure" "")
      ==
    :-  (respond 200 ~[['content-type' 'application/json'] ['set-cookie' cookie]] `(as-octs:mimes:html '{"ok":true}'))
    state(unlocked (~(put by unlocked) key [tok (add now.bowl ~d1)]))
  ::
      [%file ~]
    ?.  open  [(err 403 'This link needs its password.') state]
    =/  want=@t  (fall (find-q qs 'path') '')
    =/  hit  (skim es.u.sc |=([rel=@t e=entry] =(rel want)))
    ?~  hit  [(err 404 'That file isn’t part of this link.') state]
    =/  fname  (rear (rash rel.i.hit (more fas (cook crip (star ;~(less fas next))))))
    =/  inline=?  =(`'1' (find-q qs 'inline'))
    =/  [code=@ud hed=(list [key=@t value=@t]) body=(unit octs)]
      (file-response req e.i.hit fname inline)
    :-  (respond code hed body)
    ::  a download counts once; previews and ranges of it don't
    ::
    ?:  |(inline !=(200 code))  state
    state(links (~(put by links) tok l(downloads +(downloads.l))))
  ==
::  +file-http: GET /drive-api/file/<fid>/<hash>?name=<name>[&inline=1],
::  a file or a version of it, for the owner's viewer and downloads.
::  Its hash names its contents, so it caches for good.
::
++  file-http
  |=  [eid=@ta req=inbound-request:eyre fid=@ta hx=@ta qs=(list [p=@t q=@t])]
  ^-  (quip card _state)
  =/  pax  /http-response/[eid]
  =/  send
    |=  [code=@ud hed=(list [key=@t value=@t]) body=(unit octs)]
    ^-  (list card)
    =?  hed  |(=(200 code) =(206 code))
      [['cache-control' 'private, max-age=31536000, immutable'] hed]
    :~  [%give %fact ~[pax] %http-response-header !>(`response-header:http`[code [['x-content-type-options' 'nosniff'] hed]])]
        [%give %fact ~[pax] %http-response-data !>(body)]
        [%give %kick ~[pax] ~]
    ==
  ?~  h=(rush hx hex)  [(send 404 ~ ~) state]
  ?~  f=(~(get by folders) fid)  [(send 404 ~ ~) state]
  ?~  e=(find-entry u.f u.h)  [(send 404 ~ ~) state]
  =/  name=@t  (fall (find-q qs 'name') 'file')
  =/  [code=@ud hed=(list [key=@t value=@t]) body=(unit octs)]
    (file-response req u.e name =(`'1' (find-q qs 'inline')))
  [(send code hed body) state]
::  +find-entry: a file in [f] by its hash: current, or a version in history
::
++  find-entry
  |=  [f=folder h=hash]
  ^-  (unit entry)
  =/  vs=(list version)  (zing ~(val by old.f))
  =/  es=(list entry)  (weld ~(val by tree.f) (turn vs |=(v=version entry.v)))
  =/  hit  (skim es |=(e=entry =(h hash.e)))
  ?~(hit ~ `i.hit)
::  +file-response: [e], called [name], whole or the range asked for.
::  Inline, it's typed by its name so the browser can show it; nothing in
::  it can run (html goes as text, and the sandbox stops svg's scripts).
::  Otherwise it's a download.
::
++  file-response
  |=  [req=inbound-request:eyre e=entry name=@t inline=?]
  ^-  [@ud (list [key=@t value=@t]) (unit octs)]
  =/  rng  (get-range req size.e)
  ?:  ?=([~ ~] rng)
    [416 ~[['content-range' (crip "bytes */{(a-co:co size.e)}")]] ~]
  =/  [a=@ud b=@ud]  ?~(rng [0 size.e] (need u.rng))
  ?~  dat=(file-bytes e a b)
    [503 ~[['content-type' 'text/plain; charset=utf-8']] `(as-octs:mimes:html 'This file isn’t fully on the ship yet.')]
  =/  typ=@t  ?:(inline (file-type name) 'application/octet-stream')
  =/  hed=(list [key=@t value=@t])
    :~  ['content-type' typ]
        ['content-disposition' (crip "{?:(inline "inline" "attachment")}; filename*=UTF-8''{(en-urlt:html (trip name))}")]
        ['accept-ranges' 'bytes']
    ==
  ::  (a sandboxed pdf won't open in Chrome's viewer)
  =?  hed  !=('application/pdf' typ)  [['content-security-policy' 'sandbox'] hed]
  ?~  rng  [200 hed dat]
  [206 [['content-range' (crip "bytes {(a-co:co a)}-{(a-co:co (sub b 1))}/{(a-co:co size.e)}")] hed] dat]
::  +get-range: the bytes a request's Range header asks for, as [a b)
::  and at most 4 MiB of them: ~ for the whole file (no header, or one we
::  don't read), [~ ~] when it's past the end
::
++  get-range
  |=  [req=inbound-request:eyre size=@ud]
  ^-  (unit (unit [@ud @ud]))
  ?:  =(0 size)  ~
  =/  hs  (skim header-list.request.req |=([k=@t v=@t] =('range' (crip (cass (trip k))))))
  ?~  hs  ~
  =/  r  (rush value.i.hs ;~(pfix (jest 'bytes=') ;~(plug (punt dem) ;~(pfix hep (punt dem)))))
  ?~  r  ~
  =/  [lo=(unit @ud) hi=(unit @ud)]  u.r
  =/  ab=(unit [a=@ud b=@ud])
    ?~  lo
      ?~  hi  ~
      `[(sub size (min size u.hi)) size]
    `[u.lo ?~(hi size (min size +(u.hi)))]
  ?~  ab  ~
  ?:  (gte a.u.ab b.u.ab)  [~ ~]
  ``[a.u.ab (min b.u.ab (add a.u.ab max-chunk))]
::  +file-bytes: bytes [a b) of [e], cut from just the chunks they're in;
::  ~ if one of those isn't here
::
++  file-bytes
  |=  [e=entry a=@ud b=@ud]
  ^-  (unit octs)
  =|  out=(list [@ud @])
  =/  o=@ud  0
  =/  cs=(list hash)  chunks.e
  |-  ^-  (unit octs)
  ?~  cs  `[(sub b a) (can 3 (flop out))]
  ?:  (gte o b)  `[(sub b a) (can 3 (flop out))]
  ?~  c=(~(get by blobs) i.cs)  ~
  =/  top=@ud  (add o len.u.c)
  ?:  (lte top a)  $(cs t.cs, o top)
  =/  s=@ud  (max a o)
  =/  n=@ud  (sub (min b top) s)
  $(cs t.cs, o top, out [[n (cut 3 [(sub s o) n] dat.u.c)] out])
::
++  find-q
  |=  [qs=(list [p=@t q=@t]) k=@t]
  ^-  (unit @t)
  ?~  qs  ~
  ?:  =(k p.i.qs)  `q.i.qs
  $(qs t.qs)
::
++  get-cookie
  |=  [req=inbound-request:eyre name=@t]
  ^-  (unit @t)
  =/  pre=tape  (weld (trip name) "=")
  =/  hs  header-list.request.req
  =|  parts=(list tape)
  =.  parts
    |-  ^-  (list tape)
    ?~  hs  parts
    ?.  =('cookie' key.i.hs)  $(hs t.hs)
    =/  more=(list tape)  (rash value.i.hs (more mic (star ;~(less mic next))))
    $(hs t.hs, parts (weld parts more))
  |-  ^-  (unit @t)
  ?~  parts  ~
  =/  kv=tape  i.parts
  =?  kv  &(?=(^ kv) =(' ' i.kv))  t.kv
  ?:  =(pre (scag (lent pre) kv))  `(crip (slag (lent pre) kv))
  $(parts t.parts)
::
::  +file-type: a file's media type, by its name, for showing it inline;
::  anything we don't know is shown as text
::
++  file-type
  |=  name=@t
  ^-  @t
  =/  t  (cass (trip name))
  =/  ext=tape  ?~(d=(find "." (flop t)) "" (slag (sub (lent t) u.d) t))
  =/  types=(map tape @t)
    %-  malt
    ^-  (list [tape @t])
    :~  ["png" 'image/png']  ["jpg" 'image/jpeg']  ["jpeg" 'image/jpeg']
        ["gif" 'image/gif']  ["webp" 'image/webp']  ["avif" 'image/avif']
        ["bmp" 'image/bmp']  ["ico" 'image/x-icon']  ["svg" 'image/svg+xml']
        ["pdf" 'application/pdf']
        ["mp3" 'audio/mpeg']  ["m4a" 'audio/mp4']  ["aac" 'audio/aac']
        ["wav" 'audio/wav']  ["ogg" 'audio/ogg']  ["oga" 'audio/ogg']
        ["opus" 'audio/ogg']  ["flac" 'audio/flac']
        ["mp4" 'video/mp4']  ["m4v" 'video/mp4']  ["webm" 'video/webm']
        ["mov" 'video/quicktime']  ["ogv" 'video/ogg']
    ==
  (fall (~(get by types) ext) 'text/plain; charset=utf-8')
::
++  new-folder
  |=  name=@t
  ^-  (quip card _state)
  =/  =fid  (crip ((x-co:co 16) (end 6 eny.bowl)))
  =/  =folder
    [(fresh-folder-name name our.bowl) our.bowl 0 0 ~ ~ ~ ~ %rw ~ 0 ~ ~]
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
::  +rekey: move a folder to a new coop epoch. Publish what's live under
::  it (waves after the floor, the rock at it, [pub]); tomb what was
::  under the old one: waves [gone] and the log's, the rock at
::  [was-floor], and chunks [was-pub].
::
++  rekey
  |=  [=fid gone=(list aeon) was-pub=(set hash) was-floor=aeon]
  ^-  (quip card _state)
  =/  f  (~(got by folders) fid)
  =/  was  (coop:lib fid epoch.f)
  =.  epoch.f  +(epoch.f)
  =/  coo  (coop:lib fid epoch.f)
  =/  ws  (tap:on-log log.f)
  :_  state(folders (~(put by folders) fid f))
  ;:  weld
    [%pass /germ/[fid] %germ coo]~
    (turn ws |=([a=aeon w=wave] (tend coo /wave/(dec:lib a) w)))
  ::
    ?:  =(0 floor.f)  ~
    [(tend coo /rock/(dec:lib floor.f) ~(tap by base.f))]~
  ::
    (turn ~(tap in pub.f) |=(h=hash (tend coo /blob/(hext:lib h) (~(got by blobs) h))))
    (turn (weld gone (turn ws head)) |=(a=aeon (tomb (weld was /wave/(dec:lib a)))))
  ::
    ?:  =(0 was-floor)  ~
    [(tomb (weld was /rock/(dec:lib was-floor)))]~
  ::
    (turn ~(tap in was-pub) |=(h=hash (tomb (weld was /blob/(hext:lib h)))))
    [(give fid (folder-update fid f))]~
  ==
::  +entry-chunks: every chunk of some entries
::
++  entry-chunks
  |=  es=(list entry)
  ^-  (set hash)
  =|  out=(set hash)
  |-
  ?~  es  out
  $(es t.es, out (~(gas in out) chunks.i.es))
::
++  wave-entries
  |=  ws=(list wave)
  ^-  (list entry)
  =|  out=(list entry)
  |-
  ?~  ws  out
  =/  os=(list op)  ops.i.ws
  |-
  ?~  os  ^$(ws t.ws)
  ?~  new.i.os  $(os t.os)
  $(os t.os, out [u.new.i.os out])
::  +readable: the chunks a folder's readers may need: its base's and
::  its waves'
::
++  readable
  |=  f=folder
  ^-  (set hash)
  %-  entry-chunks
  (weld ~(val by base.f) (wave-entries (turn (tap:on-log log.f) tail)))
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
::  +prune-all: fold waves older than [keep] into their folder's base,
::  forget history as old, drop chunks nothing refers to, and move each
::  folder whose readable set changed to a new epoch
::
++  prune-all
  ^-  (quip card _state)
  =/  cutoff  (sub now.bowl keep)
  =/  fids  ~(tap in ~(key by folders))
  =|  moved=(list [=fid gone=(list aeon) pub=(set hash) floor=aeon])
  ::  waves and history
  ::
  =^  moved  folders
    |-  ^-  [_moved _folders]
    ?~  fids  [moved folders]
    =/  f  (~(got by folders) i.fids)
    =.  old.f
      %-  malt
      %+  murn  ~(tap by old.f)
      |=  [p=fpath vs=(list version)]
      =/  k  (skim vs |=(v=version (gte when.v cutoff)))
      ?~(k ~ `[p k])
    =/  ws  (tap:on-log log.f)
    =|  gone=(list aeon)
    =/  was  [pub.f floor.f]
    |-
    =/  stop=?  ?~(ws & (gte when.val.i.ws cutoff))
    ?:  |(stop ?=(~ ws))
      =.  folders  (~(put by folders) i.fids f)
      ?~  gone  ^$(fids t.fids)
      ^$(fids t.fids, moved [[i.fids gone was] moved])
    =*  w  val.i.ws
    =.  base.f
      =/  os=(list op)  ops.w
      |-  ^+  base.f
      ?~  os  base.f
      ?~  new.i.os  $(os t.os, base.f (~(del by base.f) path.i.os))
      $(os t.os, base.f (~(put by base.f) path.i.os u.new.i.os))
    $(ws t.ws, gone [aeon.w gone], floor.f aeon.w, log.f +:(del:on-log log.f aeon.w))
  ::  chunks: live if a tree, base, wave, version, commit or mirror
  ::  still needs it
  ::
  =/  fs  ~(val by folders)
  =/  live=(set hash)
    =|  out=(list entry)
    |-  ^-  (set hash)
    ?~  fs  (entry-chunks out)
    =*  f  i.fs
    =/  vs=(list version)  (zing ~(val by old.f))
    =/  pend=(list wave)  ?~(up.f ~ ?~(wave.u.up.f ~ [u.wave.u.up.f ~]))
    =/  rok=(list entry)  ?~(up.f ~ ?~(rock.u.up.f ~ ~(val by u.rock.u.up.f)))
    =/  ws=(list wave)  (turn (tap:on-log log.f) tail)
    %=  $
      fs   t.fs
      out  ;:  weld
             out
             ~(val by tree.f)
             ~(val by base.f)
             (wave-entries ws)
             (turn vs |=(v=version entry.v))
             (wave-entries pend)
             rok
           ==
    ==
  =.  live
    %-  ~(gas in live)
    (zing (turn ~(val by jobs) |=(j=job (named commit.j))))
  =/  dead=(list hash)  (skip ~(tap in ~(key by blobs)) |=(h=hash (~(has in live) h)))
  =/  dead-up  (skim dead |=(h=hash (~(has in upped) h)))
  =.  blobs  (~(dif by blobs) (malt (turn dead |=(h=hash [h *chunk]))))
  =.  upped  (~(dif in upped) (silt dead-up))
  ::  every folder whose readable set changed moves to a new epoch
  ::
  =/  mov  (malt moved)
  =.  fids  ~(tap in ~(key by folders))
  =/  cards=(list card)  (turn dead-up |=(h=hash (tomb /up/(hext:lib h))))
  |-  ^-  (quip card _state)
  ?~  fids  [cards state]
  =/  f  (~(got by folders) i.fids)
  =/  now-pub  (readable f)
  =/  m  (~(get by mov) i.fids)
  ?:  &(?=(~ m) =(now-pub pub.f))  $(fids t.fids)
  =/  [gone=(list aeon) was-pub=(set hash) was-floor=aeon]
    ?~(m [~ pub.f floor.f] u.m)
  =.  folders  (~(put by folders) i.fids f(pub now-pub))
  =^  caz  state  (rekey i.fids gone was-pub was-floor)
  $(fids t.fids, cards (weld cards caz))
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
    (turn new |=(h=hash (tend /up /(hext:lib h) (~(got by blobs) h))))
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
::  +publish: make [wave] the folder's head, with tree [tre]; what it
::  replaces or deletes goes into history
::
++  publish
  |=  [=fid tre=(map fpath entry) =wave]
  ^-  (quip card _state)
  =/  fol  (~(got by folders) fid)
  =/  prev  tree.fol
  =/  os=(list op)  ops.wave
  =|  fresh=(set hash)
  |-
  ?^  os
    =/  pre  (~(get by prev) path.i.os)
    =?  old.fol  ?=(^ pre)
      (~(add ja old.fol) path.i.os [aeon.wave when.wave author.wave u.pre])
    ?~  new.i.os  $(os t.os)
    $(os t.os, fresh (~(gas in fresh) chunks.u.new.i.os))
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
    (turn new-blobs |=(h=hash (tend coo /blob/(hext:lib h) (~(got by blobs) h))))
    [(tend coo /wave/(dec:lib aeon.wave) wave)]~
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
        (host-head fid epoch.i.i head.i.i floor.i.i)
      =/  new=(list result)
        %+  skim  results.upd
        |=  r=result
        &((allowed who.r) !=(`res.r (~(get by done) [who cid]:r)))
      :_  state(done (~(gas by done) (turn new |=(r=result [[who cid]:r res.r]))))
      (weld cards (turn new |=(r=result (give fid [%result r]))))
    ::
      %folder  (host-head fid epoch.upd head.upd floor.upd)
      %head    (host-head fid epoch.u.up.u.f aeon.upd floor.u.up.u.f)
    ::  our devices' commits, settled by the host
    ::
        %result
      ?.  (allowed who.upd)  `state
      :-  ~[(give fid upd)]
      state(done (~(put by done) [who cid]:upd res.upd))
    ::
      %gone  (drop fid)
      %link  `state                  ::  the host's own devices' business
    ==
  ==
::  +host-head: the host's epoch, head and floor for a mirror. A new
::  epoch tombs what we were reading: start over.
::
++  host-head
  |=  [=fid epoch=@ud head=aeon floor=aeon]
  ^-  (quip card _state)
  =/  f  (~(got by folders) fid)
  ?~  up.f  `state
  =/  m  u.up.f
  =?  m  !=(epoch epoch.m)  m(busy |, wave ~, rock ~, need ~)
  =.  m  m(epoch epoch, goal (max goal.m head), floor floor)
  =.  folders  (~(put by folders) fid f(up `m))
  (pull fid)
::  +pull: read the host's next wave, or its rock if we're behind its
::  floor
::
++  pull
  |=  =fid
  ^-  (quip card _state)
  ?~  f=(~(get by folders) fid)  `state
  ?~  up.u.f  `state
  =/  m  u.up.u.f
  ?:  |(busy.m (gte head.u.f goal.m))  `state
  =/  ep  (dec:lib epoch.m)
  =/  [kind=@ta n=@ta]
    ?:  (lth head.u.f floor.m)
      [%rock (dec:lib floor.m)]
    [%wave (dec:lib +(head.u.f))]
  :_  state(folders (~(put by folders) fid u.f(up `m(busy &, wave ~, rock ~, need ~))))
  :_  ~
  :*  %pass  /pull/[fid]/[ep]/[kind]/[n]  %keen  &  host.u.f
      ~[%g %x '1' %drive %$ '1' %f fid ep kind n]
  ==
::  +take-pull: a wave, rock or chunk we read from a host
::
++  take-pull
  |=  [=fid ep=@ta [kind=@ta arg=@ta] gag=(unit page)]
  ^-  (quip card _state)
  ?~  f=(~(get by folders) fid)  `state
  ?~  up.u.f  `state
  =/  m  u.up.u.f
  ?.  =(ep (dec:lib epoch.m))  `state                ::  from before a rekey
  ::  on any failure, read it again in a while
  ::
  =/  retry
    :-  [%pass /retry/[fid] %arvo %b %wait (add now.bowl ~s5)]~
    state(folders (~(put by folders) fid u.f(up `m(busy |, wave ~, rock ~, need ~))))
  =/  fetch
    |=  need=(set hash)
    ^-  (list card)
    %+  turn  ~(tap in need)
    |=  h=hash
    ^-  card
    =/  hx  (hext:lib h)
    [%pass /pull/[fid]/[ep]/blob/[hx] %keen & host.u.f ~[%g %x '1' %drive %$ '1' %f fid ep %blob hx]]
  ?+    kind  `state
      %wave
    ?~  gag  retry
    ?~  w=((soft wave) q.u.gag)  retry
    ?.  =(aeon.u.w +(head.u.f))  retry
    =/  need=(set hash)
      (silt (skip ~(tap in (entry-chunks (wave-entries ~[u.w]))) |=(h=hash (~(has by blobs) h))))
    ?:  =(~ need)
      (take-wave fid u.w)
    :-  (fetch need)
    state(folders (~(put by folders) fid u.f(up `m(wave w, need need))))
  ::
      %rock
    ?~  gag  retry
    ?~  r=((soft (list (pair fpath entry))) q.u.gag)  retry
    =/  rok  (malt u.r)
    =/  need=(set hash)
      (silt (skip ~(tap in (entry-chunks ~(val by rok))) |=(h=hash (~(has by blobs) h))))
    ?:  =(~ need)
      (take-rock fid rok)
    :-  (fetch need)
    state(folders (~(put by folders) fid u.f(up `m(rock `rok, need need))))
  ::
      %blob
    =/  h  (rash arg hex)
    ?.  (~(has in need.m) h)  `state
    ?~  c=(check-chunk h gag)  retry
    =.  blobs  (~(put by blobs) h u.c)
    =.  need.m  (~(del in need.m) h)
    ?.  =(~ need.m)
      `state(folders (~(put by folders) fid u.f(up `m)))
    ?^  wave.m  (take-wave fid u.wave.m)
    ?^  rock.m  (take-rock fid u.rock.m)
    `state
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
  =.  folders  (~(put by folders) fid f(up `u.up.f(busy |, wave ~, rock ~, need ~)))
  =^  caz  state  (pull fid)
  [(weld cards caz) state]
::  +take-rock: the host's tree at its floor, with all its chunks: we
::  jump there, and our devices read it as our rock
::
++  take-rock
  |=  [=fid rok=(map fpath entry)]
  ^-  (quip card _state)
  =/  f  (~(got by folders) fid)
  ?>  ?=(^ up.f)
  =/  m  u.up.f
  =/  gone  (turn (tap:on-log log.f) head)
  =/  [was-pub=(set hash) was-floor=aeon]  [pub.f floor.f]
  =.  f
    %=  f
      head   floor.m
      floor  floor.m
      tree   rok
      base   rok
      log    ~
      up     `m(busy |, wave ~, rock ~, need ~)
    ==
  =.  folders  (~(put by folders) fid f(pub (readable f)))
  =^  cards  state  (rekey fid gone was-pub was-floor)
  =^  caz  state  (pull fid)
  [(weld cards caz) state]
--
