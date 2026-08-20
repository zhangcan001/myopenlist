package _115_open

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/driver"
)

func TestTokenPersistenceRetriesTransientFailure(t *testing.T) {
	attempts := 0
	sleeps := make([]time.Duration, 0, 2)
	err := retryPersistence(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	}, func(delay time.Duration) { sleeps = append(sleeps, delay) })
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(sleeps) != 2 || sleeps[0] != 100*time.Millisecond || sleeps[1] != 500*time.Millisecond {
		t.Fatalf("sleep schedule = %v", sleeps)
	}
}

func TestRefreshCallbackPersistsAtomicPair(t *testing.T) {
	driverStorage := &Open115{Addition: Addition{AccessToken: "old-access", RefreshToken: "old-refresh"}}
	saved := make([]Addition, 0, 1)
	if err := persistTestPair(driverStorage, "new-access", "new-refresh", func(addition driver.Additional) error {
		saved = append(saved, *addition.(*Addition))
		return nil
	}, func(time.Duration) {}); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0].AccessToken != "new-access" || saved[0].RefreshToken != "new-refresh" {
		t.Fatalf("saved pair = %+v", saved)
	}
}

func TestTokenPersistenceFailureKeepsRuntimePair(t *testing.T) {
	driverStorage := &Open115{Addition: Addition{AccessToken: "new-access", RefreshToken: "new-refresh"}}
	if err := persistTestPair(driverStorage, "new-access", "new-refresh", func(driver.Additional) error {
		return errors.New("storage unavailable")
	}, func(time.Duration) {}); err == nil {
		t.Fatal("expected persistence failure")
	}
	status := driverStorage.TokenPersistenceStatus()
	if status.State != TokenPersistenceFailed || status.Attempts != 3 {
		t.Fatalf("unexpected persistence status: %+v", status)
	}
	if driverStorage.AccessToken != "new-access" || driverStorage.RefreshToken != "new-refresh" {
		t.Fatalf("runtime pair was rolled back: %+v", driverStorage.Addition)
	}
}

func TestPersistenceRetryNeverWritesOlderTokenPair(t *testing.T) {
	driverStorage := &Open115{Addition: Addition{AccessToken: "access-b", RefreshToken: "refresh-b"}}
	saved := make([]string, 0, 6)
	failedSave := func(addition driver.Additional) error {
		pair := addition.(*Addition)
		saved = append(saved, pair.AccessToken+":"+pair.RefreshToken)
		return errors.New("transient")
	}
	noSleep := func(time.Duration) {}

	if err := persistTestPair(driverStorage, "access-a", "refresh-a", failedSave, noSleep); err == nil {
		t.Fatal("old pair persistence unexpectedly succeeded")
	}
	driverStorage.SetTokenPair("access-b", "refresh-b")
	if err := persistTestPair(driverStorage, "access-b", "refresh-b", func(addition driver.Additional) error {
		pair := addition.(*Addition)
		saved = append(saved, pair.AccessToken+":"+pair.RefreshToken)
		return nil
	}, noSleep); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 4 || saved[len(saved)-1] != "access-b:refresh-b" {
		t.Fatalf("saved pairs = %v", saved)
	}
	if driverStorage.AccessToken != "access-b" || driverStorage.RefreshToken != "refresh-b" {
		t.Fatalf("older pair replaced latest in-memory pair: %+v", driverStorage.Addition)
	}
}

func TestPersistenceGenerationLatestWins(t *testing.T) {
	driverStorage := &Open115{}
	driverStorage.SetTokenPair("access-a", "refresh-a")
	driverStorage.tokenPersistenceMu.Lock()
	generationA := driverStorage.tokenPersistenceGeneration
	driverStorage.tokenPersistenceMu.Unlock()

	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	saved := make([]string, 0, 2)
	save := func(addition driver.Additional) error {
		pair := addition.(*Addition)
		mu.Lock()
		saved = append(saved, pair.AccessToken+":"+pair.RefreshToken)
		mu.Unlock()
		if pair.AccessToken == "access-a" {
			close(started)
			<-release
			return errors.New("old pair failed")
		}
		return nil
	}

	oldDone := make(chan error, 1)
	go func() {
		oldDone <- driverStorage.persistTokenPair("access-a", "refresh-a", generationA, save, func(time.Duration) {})
	}()
	<-started
	driverStorage.SetTokenPair("access-b", "refresh-b")
	close(release)
	if err := <-oldDone; err != nil {
		t.Fatal(err)
	}

	driverStorage.tokenPersistenceMu.Lock()
	generationB := driverStorage.tokenPersistenceGeneration
	driverStorage.tokenPersistenceMu.Unlock()
	if err := driverStorage.persistTokenPair("access-b", "refresh-b", generationB, save, func(time.Duration) {}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(saved) != 2 || saved[0] != "access-a:refresh-a" || saved[1] != "access-b:refresh-b" {
		t.Fatalf("saved pairs = %v", saved)
	}
}

func persistTestPair(d *Open115, accessToken, refreshToken string, save func(driver.Additional) error, sleep func(time.Duration)) error {
	d.tokenPersistenceMu.Lock()
	d.tokenPersistenceGeneration++
	d.Addition.AccessToken = accessToken
	d.Addition.RefreshToken = refreshToken
	generation := d.tokenPersistenceGeneration
	d.tokenPersistenceMu.Unlock()
	return d.persistTokenPair(accessToken, refreshToken, generation, save, sleep)
}
