package mount

// MountBackend is the small boundary between lifecycle management and the
// platform filesystem implementation. Tests use a fake backend; Windows uses
// WinFsp through winfsp.go.
type MountBackend interface {
	Mount(MountProfile) (MountHandle, error)
}

type MountHandle interface {
	Wait() error
	Unmount() error
}
