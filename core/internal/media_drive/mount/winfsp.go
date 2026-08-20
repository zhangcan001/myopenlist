//go:build windows

package mount

import (
	"io"
	"os"

	"github.com/OpenListTeam/OpenList/v4/pkg/gowebdav"
	"github.com/winfsp/cgofuse/fuse"
)

type winFSPBackend struct{}

func newWinFSPBackend() MountBackend {
	return &winFSPBackend{}
}

func (b *winFSPBackend) Mount(profile MountProfile) (MountHandle, error) {
	filesystem := &webDAVFileSystem{
		client: gowebdav.NewClient(profile.WebDAVURL, profile.Username, profile.Password),
		ready:  make(chan struct{}),
	}
	host := fuse.NewFileSystemHost(filesystem)
	host.SetCapCaseInsensitive(true)
	host.SetCapReaddirPlus(true)
	mount := &winFSPMount{
		host:  host,
		ready: filesystem.ready,
		done:  make(chan error, 1),
	}
	go mount.run(profile.DriveLetter)

	select {
	case <-mount.ready:
		return mount, nil
	case err := <-mount.done:
		if err == nil {
			err = ErrMountFailed
		}
		return nil, err
	}
}

type winFSPMount struct {
	host  *fuse.FileSystemHost
	ready <-chan struct{}
	done  chan error
}

func (m *winFSPMount) run(driveLetter string) {
	var err error
	defer func() {
		if recover() != nil {
			err = ErrWinFSPUnavailable
		}
		m.done <- err
	}()
	if !m.host.Mount(driveLetter, nil) {
		err = ErrMountFailed
	}
}

func (m *winFSPMount) Wait() error {
	return <-m.done
}

func (m *winFSPMount) Unmount() error {
	if !m.host.Unmount() {
		return ErrUnmountFailed
	}
	return nil
}

type webDAVFileSystem struct {
	fuse.FileSystemBase
	client *gowebdav.Client
	ready  chan struct{}
}

func (f *webDAVFileSystem) Init() {
	close(f.ready)
}

func (f *webDAVFileSystem) Statfs(_ string, stat *fuse.Statfs_t) int {
	stat.Bsize = 4096
	stat.Frsize = 4096
	stat.Blocks = 1 << 60
	stat.Bfree = 0
	stat.Bavail = 0
	stat.Files = 1 << 20
	stat.Ffree = 0
	stat.Favail = 0
	stat.Namemax = 255
	return 0
}

func (f *webDAVFileSystem) Getattr(name string, stat *fuse.Stat_t, _ uint64) int {
	info, err := f.client.Stat(name)
	if err != nil {
		return webDAVError(err)
	}
	*stat = fileStat(info)
	return 0
}

func (f *webDAVFileSystem) Open(name string, flags int) (int, uint64) {
	if flags&fuse.O_ACCMODE != fuse.O_RDONLY {
		return -fuse.EROFS, 0
	}
	info, err := f.client.Stat(name)
	if err != nil {
		return webDAVError(err), 0
	}
	if info.IsDir() {
		return -fuse.EISDIR, 0
	}
	return 0, 0
}

func (f *webDAVFileSystem) Read(name string, buffer []byte, offset int64, _ uint64) int {
	if offset < 0 {
		return -fuse.EINVAL
	}
	if len(buffer) == 0 {
		return 0
	}
	reader, err := f.client.ReadStreamRange(name, offset, int64(len(buffer)))
	if err != nil {
		return webDAVError(err)
	}
	defer reader.Close()
	n, err := io.ReadFull(reader, buffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return -fuse.EIO
	}
	return n
}

func (f *webDAVFileSystem) Opendir(name string) (int, uint64) {
	info, err := f.client.Stat(name)
	if err != nil {
		return webDAVError(err), 0
	}
	if !info.IsDir() {
		return -fuse.ENOTDIR, 0
	}
	return 0, 0
}

func (f *webDAVFileSystem) Readdir(name string, fill func(string, *fuse.Stat_t, int64) bool, _ int64, _ uint64) int {
	files, err := f.client.ReadDir(name)
	if err != nil {
		return webDAVError(err)
	}
	for _, info := range files {
		stat := fileStat(info)
		if !fill(info.Name(), &stat, 0) {
			break
		}
	}
	return 0
}

func (f *webDAVFileSystem) Access(name string, mask uint32) int {
	if mask&(fuse.W_OK|fuse.DELETE_OK) != 0 {
		return -fuse.EROFS
	}
	_, err := f.client.Stat(name)
	if err != nil {
		return webDAVError(err)
	}
	return 0
}

func (*webDAVFileSystem) Create(string, int, uint32) (int, uint64) {
	return -fuse.EROFS, 0
}

func (*webDAVFileSystem) Write(string, []byte, int64, uint64) int {
	return -fuse.EROFS
}

func (*webDAVFileSystem) Truncate(string, int64, uint64) int {
	return -fuse.EROFS
}

func (*webDAVFileSystem) Mkdir(string, uint32) int {
	return -fuse.EROFS
}

func (*webDAVFileSystem) Unlink(string) int {
	return -fuse.EROFS
}

func (*webDAVFileSystem) Rmdir(string) int {
	return -fuse.EROFS
}

func (*webDAVFileSystem) Rename(string, string) int {
	return -fuse.EROFS
}

func (*webDAVFileSystem) Release(string, uint64) int {
	return 0
}

func (*webDAVFileSystem) Releasedir(string, uint64) int {
	return 0
}

func fileStat(info os.FileInfo) fuse.Stat_t {
	mode := uint32(fuse.S_IFREG | 0444)
	if info.IsDir() {
		mode = fuse.S_IFDIR | 0555
	}
	modified := fuse.NewTimespec(info.ModTime())
	return fuse.Stat_t{
		Mode:     mode,
		Nlink:    1,
		Size:     info.Size(),
		Atim:     modified,
		Mtim:     modified,
		Ctim:     modified,
		Birthtim: modified,
	}
}

func webDAVError(err error) int {
	if gowebdav.IsErrNotFound(err) {
		return -fuse.ENOENT
	}
	return -fuse.EIO
}

var _ fuse.FileSystemInterface = (*webDAVFileSystem)(nil)
