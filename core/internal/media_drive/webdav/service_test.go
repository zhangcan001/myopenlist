package webdav

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	corenet "github.com/OpenListTeam/OpenList/v4/internal/net"
	"github.com/OpenListTeam/OpenList/v4/pkg/http_range"
)

type memoryProfileStore struct {
	mu      sync.Mutex
	profile ManagedWebDAVProfile
	value   string
}

func (s *memoryProfileStore) Load() (ManagedWebDAVProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profile.ID == "" {
		return DefaultProfile(), nil
	}
	return s.profile, nil
}

func (s *memoryProfileStore) Save(profile ManagedWebDAVProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profile = profile
	value, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	s.value = string(value)
	return nil
}

func TestLocalhostOnlyRejectRemote(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	profile := DefaultProfile()
	profile.PasswordHash = hash
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := newManagedHandler(profile, inner, func() (*model.User, error) {
		return &model.User{Username: "admin"}, nil
	})

	remote := httptest.NewRecorder()
	remoteRequest := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	remoteRequest.RemoteAddr = "192.168.1.100:1234"
	remoteRequest.SetBasicAuth("media", "secret")
	handler.ServeHTTP(remote, remoteRequest)
	if remote.Code != http.StatusForbidden {
		t.Fatalf("remote request status = %d, want %d", remote.Code, http.StatusForbidden)
	}

	local := httptest.NewRecorder()
	localRequest := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	localRequest.RemoteAddr = "127.0.0.1:1234"
	localRequest.SetBasicAuth("media", "secret")
	handler.ServeHTTP(local, localRequest)
	if local.Code != http.StatusOK {
		t.Fatalf("loopback request status = %d, want %d", local.Code, http.StatusOK)
	}
}

func TestWebDAVPasswordNotStoredPlaintext(t *testing.T) {
	store := &memoryProfileStore{}
	manager := NewManager(store)
	view, err := manager.UpdateProfile(ProfileUpdate{Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if !view.PasswordConfigured {
		t.Fatal("password should be configured")
	}
	if strings.Contains(store.value, "password123") {
		t.Fatalf("profile persistence contains plaintext password: %s", store.value)
	}
	if store.profile.PasswordHash == "" || store.profile.PasswordHash == "password123" {
		t.Fatalf("invalid stored password hash: %q", store.profile.PasswordHash)
	}
}

func TestWebDAVServiceLifecycle(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	profile := DefaultProfile()
	profile.Enabled = true
	profile.Port = 0
	profile.PasswordHash = hash

	service := newService(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := service.Start(profile); err != nil {
		t.Fatal(err)
	}
	status := service.Status()
	if status.State != StateRunning || !status.Running || !strings.HasPrefix(status.Address, "127.0.0.1:") {
		t.Fatalf("running status = %+v", status)
	}
	if err := service.Stop(); err != nil {
		t.Fatal(err)
	}
	status = service.Status()
	if status.State != StateStopped || status.Running || status.Address != "" {
		t.Fatalf("stopped status = %+v", status)
	}
}

func TestWebDAVPortConflict(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()

	port := occupied.Addr().(*net.TCPAddr).Port
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	profile := DefaultProfile()
	profile.Enabled = true
	profile.Port = port
	profile.PasswordHash = hash

	service := newService(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := service.Start(profile); !errors.Is(err, ErrPortConflict) {
		t.Fatalf("Start error = %v, want %v", err, ErrPortConflict)
	}
	if status := service.Status(); status.State != StateFailed {
		t.Fatalf("failed status = %+v", status)
	}
}

func TestWebDAVRangeRequest(t *testing.T) {
	const size int64 = 50 * 1024 * 1024 * 1024
	reader := &recordingRangeReader{}
	request := httptest.NewRequest(http.MethodGet, "http://localhost/movie.mkv", nil)
	request.Header.Set("Range", "bytes=0-1023")
	recorder := httptest.NewRecorder()

	err := corenet.ServeHTTP(recorder, request, "movie.mkv", time.Time{}, size, &model.RangeReadCloser{RangeReader: reader})
	if err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusPartialContent)
	}
	if recorder.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("Accept-Ranges = %q", recorder.Header().Get("Accept-Ranges"))
	}
	if recorder.Body.Len() != 1024 {
		t.Fatalf("body length = %d, want 1024", recorder.Body.Len())
	}
	if len(reader.ranges) != 1 || reader.ranges[0].Start != 0 || reader.ranges[0].Length != 1024 {
		t.Fatalf("requested ranges = %+v", reader.ranges)
	}
	if reader.maxRequestedSize >= size {
		t.Fatalf("reader was asked for the complete large file: %d", reader.maxRequestedSize)
	}
}

func TestWebDAVStreaming(t *testing.T) {
	const size int64 = 50 * 1024 * 1024 * 1024
	for _, start := range []int64{0, 4 * 1024 * 1024 * 1024, 49 * 1024 * 1024 * 1024} {
		t.Run("offset-"+strconv.FormatInt(start/(1024*1024*1024), 10)+"-gb", func(t *testing.T) {
			reader := &recordingRangeReader{}
			request := httptest.NewRequest(http.MethodGet, "http://localhost/movie.mkv", nil)
			request.Header.Set("Range", formatRange(start, start+1023))
			recorder := httptest.NewRecorder()

			if err := corenet.ServeHTTP(recorder, request, "movie.mkv", time.Time{}, size, &model.RangeReadCloser{RangeReader: reader}); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusPartialContent || recorder.Body.Len() != 1024 {
				t.Fatalf("status/body = %d/%d", recorder.Code, recorder.Body.Len())
			}
			if len(reader.ranges) != 1 || reader.ranges[0].Start != start || reader.ranges[0].Length != 1024 {
				t.Fatalf("requested ranges = %+v, want start %d length 1024", reader.ranges, start)
			}
		})
	}
}

func formatRange(start, end int64) string {
	return "bytes=" + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10)
}

type recordingRangeReader struct {
	mu               sync.Mutex
	ranges           []http_range.Range
	maxRequestedSize int64
}

func (r *recordingRangeReader) RangeRead(_ context.Context, requested http_range.Range) (io.ReadCloser, error) {
	r.mu.Lock()
	r.ranges = append(r.ranges, requested)
	if requested.Length > r.maxRequestedSize {
		r.maxRequestedSize = requested.Length
	}
	r.mu.Unlock()
	return io.NopCloser(io.LimitReader(zeroReader{}, requested.Length)), nil
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 0
	}
	return len(buffer), nil
}
