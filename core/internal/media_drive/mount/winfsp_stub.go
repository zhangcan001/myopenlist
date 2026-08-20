//go:build !windows

package mount

type unsupportedBackend struct{}

func newWinFSPBackend() MountBackend {
	return unsupportedBackend{}
}

func (unsupportedBackend) Mount(MountProfile) (MountHandle, error) {
	return nil, ErrWinFSPUnavailable
}
