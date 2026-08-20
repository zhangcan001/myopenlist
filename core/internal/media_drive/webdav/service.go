package webdav

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	serverwebdav "github.com/OpenListTeam/OpenList/v4/server/webdav"
)

type Service struct {
	mu       sync.Mutex
	handler  http.Handler
	listener net.Listener
	server   *http.Server
	state    ServiceState
	address  string
}

func NewService() *Service {
	return newService(newOpenListWebDAVHandler())
}

func newService(handler http.Handler) *Service {
	return &Service{handler: handler, state: StateStopped}
}

func newOpenListWebDAVHandler() http.Handler {
	return &serverwebdav.Handler{
		LockSystem: serverwebdav.NewMemLS(),
	}
}

func (s *Service) Start(profile ManagedWebDAVProfile) error {
	profile = normalizeProfile(profile)
	if err := validateProfile(profile); err != nil {
		return s.fail(err)
	}
	if profile.PasswordHash == "" {
		return s.fail(ErrPasswordNotConfigured)
	}

	s.mu.Lock()
	if s.state == StateRunning || s.state == StateStarting || s.state == StateStopping {
		s.mu.Unlock()
		return ErrServiceRunning
	}
	s.state = StateStarting
	address := net.JoinHostPort(profile.BindAddress, formatPort(profile.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		s.state = StateFailed
		s.mu.Unlock()
		if isPortConflict(err) {
			return ErrPortConflict
		}
		return err
	}
	server := &http.Server{
		Handler: newManagedHandler(profile, s.handler, op.GetAdmin),
	}
	s.listener = listener
	s.server = server
	s.address = listener.Addr().String()
	s.state = StateRunning
	s.mu.Unlock()

	go s.serve(server, listener)
	return nil
}

func (s *Service) serve(server *http.Server, listener net.Listener) {
	err := server.Serve(listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server == server && s.state == StateRunning {
		s.state = StateFailed
		s.listener = nil
		s.server = nil
		s.address = ""
	}
}

func (s *Service) Stop() error {
	s.mu.Lock()
	if s.state == StateStopped {
		s.mu.Unlock()
		return nil
	}
	s.state = StateStopping
	server := s.server
	s.mu.Unlock()

	if server != nil {
		_ = server.Close()
	}

	s.mu.Lock()
	s.listener = nil
	s.server = nil
	s.address = ""
	s.state = StateStopped
	s.mu.Unlock()
	return nil
}

func (s *Service) Status() ServiceStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ServiceStatus{
		Running: s.state == StateRunning,
		Address: s.address,
		State:   s.state,
	}
}

func (s *Service) fail(err error) error {
	s.mu.Lock()
	s.state = StateFailed
	s.mu.Unlock()
	return err
}

type userProvider func() (*model.User, error)

type managedHandler struct {
	profile      ManagedWebDAVProfile
	inner        http.Handler
	userProvider userProvider
}

func newManagedHandler(profile ManagedWebDAVProfile, inner http.Handler, provider userProvider) http.Handler {
	return &managedHandler{profile: profile, inner: inner, userProvider: provider}
}

func (h *managedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.profile.AllowLocalhostOnly && !isLoopbackRemote(r.RemoteAddr) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	username, password, ok := r.BasicAuth()
	if !ok || username != h.profile.Username || !passwordMatches(password, h.profile.PasswordHash) {
		w.Header().Set("WWW-Authenticate", `Basic realm="managed-webdav"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	user, err := h.userProvider()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	ctx := context.WithValue(r.Context(), conf.UserKey, user)
	ctx = context.WithValue(ctx, conf.MetaPassKey, "")
	h.inner.ServeHTTP(w, r.WithContext(ctx))
}

func isLoopbackRemote(remoteAddr string) bool {
	host := remoteAddr
	if parsedHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func formatPort(port int) string {
	return strconv.Itoa(port)
}

func isPortConflict(err error) bool {
	message := strings.ToLower(err.Error())
	return errors.Is(err, syscall.EADDRINUSE) || strings.Contains(message, "address already in use") || strings.Contains(message, "only one usage")
}
