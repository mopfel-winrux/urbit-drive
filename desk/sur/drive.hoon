::  drive: types
::
::    A folder is a revision log of waves (sets of file ops), numbered by
::    aeon from 1. File contents are content-addressed chunks, shared by
::    every folder that references them. File paths are cords ("a/b.txt"),
::    not hoon paths, since real file names aren't knots.
::
::    A folder is hosted by one ship, which orders its commits. A folder
::    shared with us is a mirror: we follow its host's log, publish it to
::    our own devices, and relay their commits to the host.
::
|%
::  $hash: sha-256 of a chunk, as an atom whose @ux is the usual hex
::
+$  hash   @ux
+$  fid    @ta                         ::  folder id, 16 hex digits
+$  aeon   @ud
+$  fpath  @t                          ::  "dir/sub/name.ext"
+$  chunk  [len=@ud dat=@]
+$  role   ?(%r %rw)
::  $entry: a file at one revision
::
+$  entry
  $:  =hash                            ::  sha-256 of the whole file
      size=@ud
      mtime=@da
      exec=?
      chunks=(list hash)
  ==
+$  op     [path=fpath new=(unit entry)]  ::  new=~ deletes
+$  wave   [=aeon author=ship when=@da ops=(list op)]
::  $cop: an op in a commit, with the hash the writer last saw there
::
+$  cop    [path=fpath base=(unit hash) new=(unit entry)]
+$  commit  [cid=@uv =fid base=aeon ops=(list cop)]
::
::  $folder: the log holds waves floor+1..head; [base] is the tree at
::  [floor], published as a rock once floor > 0. Older waves are folded
::  into base when they leave the retention window.
::
+$  folder
  $:  name=@t                          ::  ours to choose, for mirrors too
      host=ship
      head=aeon
      epoch=@ud                        ::  our coop's key epoch
      tree=(map fpath entry)
      log=((mop aeon wave) lth)
      pub=(set hash)                   ::  blobs %tend'ed under our coop
      members=(map ship role)          ::  who we share it with (as host)
      =role                            ::  ours (%rw if we host it)
      up=(unit mirror)                 ::  set when someone else hosts it
      floor=aeon
      base=(map fpath entry)
      old=(jar fpath version)          ::  replaced and deleted, newest first
  ==
::  $version: a file as it was until [aeon] replaced or deleted it
::
+$  version  [=aeon when=@da who=ship =entry]
::  $mirror: how far we've followed a host
::
+$  mirror
  $:  epoch=@ud                        ::  the host's coop epoch
      goal=aeon                        ::  the host's head
      floor=aeon                       ::  the host's floor
      busy=?                           ::  reading a wave or rock
      wave=(unit wave)                 ::  read, waiting on chunks
      rock=(unit (map fpath entry))    ::  read, waiting on chunks
      need=(set hash)
  ==
+$  invite  [name=@t =role]
::  $link: a share link, /drive-api/s/<token>, readable by anyone who has
::  it (and its password, if set). It always shows the current files.
::
+$  link
  $:  =fid
      path=fpath                       ::  a file, a folder, or '' for all
      made=@da
      expires=(unit @da)
      pass=(unit [salt=@ hash=@])
      views=@ud
      downloads=@ud
  ==
::  $device: a moon we paired, kept forever (moons are never reused)
::
+$  device  [name=@t pass=@ux since=@da revoked=(unit @da)]
::  $job: a commit waiting on chunks from [src], who sent it for
::  [by] (a device name, or the author's @p)
::
+$  job    [src=ship by=@t =commit need=(set hash)]
::
::  $action: pokes in.
::
::    from our ship (the client over eyre, or the web UI):
::      %pair %unpair %new-folder %share %unshare %accept %decline %leave
::      %restore %keep %prune %put %del %mv %mkdir %link-set %link-drop
::    from our devices:  %commit %link-ask
::    from a host:       %invite %uninvite
::    from a guest:      %relay (a guest's device's commit)
::
+$  action
  $%  [%pair name=@t moon=ship pass=@ux]
      [%unpair moon=ship]
      [%new-folder name=@t]
      [%commit =commit]
      [%share =fid who=ship =role]
      [%unshare =fid who=ship]
      [%invite =fid name=@t =role]
      [%uninvite =fid]
      [%accept host=ship =fid]
      [%decline host=ship =fid]
      [%leave =fid]
      [%relay author=ship by=@t =commit]
      [%restore =fid path=fpath =hash]  ::  a version of a file, as current
      [%keep keep=@dr]                  ::  how long history lasts
      [%prune ~]                        ::  prune now
    ::  from the web page: files, as commits from our ship. Chunks of
    ::  an upload arrive first by POST to /drive-api/blob.
    ::
      [%put =fid path=fpath =entry]
      [%del =fid path=fpath]            ::  a file, or a folder and all in it
      [%mv =fid from=fpath to=fpath]    ::  rename or move, likewise
      [%mkdir =fid path=fpath]          ::  an empty folder: a marker "path/"
    ::  share links; the page picks the token (128 random bits, hex)
    ::
      $:  %link-set  token=@t  =fid  path=fpath  expires=(unit @da)
          pass=?(%keep %none [%set p=@t])
      ==
      [%link-drop token=@t]
    ::  from our devices: a link to this file or folder. The answer is a
    ::  %link fact: the token to use (an existing plain link's, or this
    ::  one), or why not.
    ::
      [%link-ask token=@t =fid path=fpath]
  ==
::  $update: facts on /v1/folders (our devices) and /v1/share/<fid>
::  (a folder's guests)
::
::
::    A watch is answered with one %hello: ames kicks a subscriber with
::    more than five facts unacked, so a fact per folder and per result
::    would clog a new subscription.
::
+$  info    [=fid name=@t host=ship epoch=@ud head=aeon =role floor=aeon]
+$  result  [who=ship cid=@uv res=(each aeon @t)]
+$  update
  $%  [%hello folders=(list info) results=(list result)]
      [%folder info]
      [%head =fid =aeon]
      [%result result]
      [%gone =fid]
      [%link who=ship asked=@t res=(each @t @t)]  ::  token, or why not
  ==
--
