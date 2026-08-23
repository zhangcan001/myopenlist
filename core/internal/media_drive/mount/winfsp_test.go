//go:build windows

package mount

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/OpenListTeam/OpenList/v4/pkg/gowebdav"
	"github.com/winfsp/cgofuse/fuse"
)

func TestGetattrCachesWebDAVStat(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = fmt.Fprintf(w, `<?xml version="1.0"?><d:multistatus xmlns:d="DAV:"><d:response><d:href>%s</d:href><d:propstat><d:prop><d:displayname>video.mkv</d:displayname><d:resourcetype/><d:getcontentlength>1024</d:getcontentlength><d:getlastmodified>Sun, 23 Aug 2026 00:00:00 GMT</d:getlastmodified></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`, r.URL.Path)
	}))
	t.Cleanup(server.Close)
	filesystem := newWebDAVFileSystem(gowebdav.NewClient(server.URL, "", ""))

	var stat fuse.Stat_t
	if result := filesystem.Getattr("/video.mkv", &stat, 0); result != 0 {
		t.Fatalf("first getattr = %d", result)
	}
	if result := filesystem.Getattr("/video.mkv", &stat, 0); result != 0 {
		t.Fatalf("second getattr = %d", result)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("PROPFIND requests = %d, want 1", got)
	}
}
