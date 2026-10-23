::  drive: types
::
::    A folder is a revision log of waves (sets of file ops), numbered by
::    aeon from 1. File contents are content-addressed chunks, shared by
::    every folder that references them. File paths are cords ("a/b.txt"),
::    not hoon paths, since real file names aren't knots.
::
|%
::  $hash: sha-256 of a chunk, as an atom whose @ux is the usual hex
::
+$  hash   @ux
+$  fid    @ta                         ::  folder id, 16 hex digits
+$  aeon   @ud
+$  fpath  @t                          ::  "dir/sub/name.ext"
+$  chunk  [len=@ud dat=@]
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
+$  folder
  $:  name=@t
      host=ship
      head=aeon
      epoch=@ud
      tree=(map fpath entry)
      log=((mop aeon wave) lth)
      pub=(set hash)                   ::  blobs %tend'ed under this epoch
  ==
::  $device: a moon we paired, kept forever (moons are never reused)
::
+$  device  [name=@t pass=@ux since=@da revoked=(unit @da)]
::  $job: a commit waiting on chunks from its writer
::
+$  job    [src=ship =commit need=(set hash)]
::
::  $action: pokes in. %pair, %unpair, %new-folder come from our own
::  ship (the client over eyre, or the web UI); %commit from devices.
::
+$  action
  $%  [%pair name=@t moon=ship pass=@ux]
      [%unpair moon=ship]
      [%new-folder name=@t]
      [%commit =commit]
  ==
::  $update: facts on /v1/folders
::
+$  update
  $%  [%folder =fid name=@t host=ship epoch=@ud head=aeon]
      [%head =fid =aeon]
      [%result who=ship cid=@uv res=(each aeon @t)]
  ==
--
