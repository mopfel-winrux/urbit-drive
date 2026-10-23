package sync

import "sort"

// FolderStatus is one folder as the UI shows it.
type FolderStatus struct {
	Fid, Name    string
	Synced, Head uint64
	Committing   bool // a commit of ours is in flight
	Busy         bool // reading waves or scanning
}

// UpToDate: nothing to read and nothing to send.
func (s FolderStatus) UpToDate() bool { return s.Synced >= s.Head && !s.Committing && !s.Busy }

// Snapshot is every folder's state, by name; call it from any goroutine
// but the node's.
func (f *Follower) Snapshot() ([]FolderStatus, error) {
	var out []FolderStatus
	err := f.n.Do(func() error {
		for _, fo := range f.folders {
			out = append(out, FolderStatus{
				Fid: fo.fid, Name: fo.name, Synced: fo.synced, Head: fo.head,
				Committing: fo.pending != nil, Busy: fo.pulling,
			})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, err
}
