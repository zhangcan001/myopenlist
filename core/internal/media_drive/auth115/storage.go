package auth115

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	_115_open "github.com/OpenListTeam/OpenList/v4/drivers/115_open"
	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	managedMountPath       = "/115"
	managedRemark          = "managed-by:openlist-115-media-drive"
	managedCacheExpiration = 30
)

type CoreStorageProvisioner struct {
	mu        sync.Mutex
	mountPath string
	latest    TokenPair
	latestSet bool
}

func NewCoreStorageProvisioner() *CoreStorageProvisioner {
	return &CoreStorageProvisioner{mountPath: managedMountPath}
}

func (p *CoreStorageProvisioner) Provision(ctx context.Context, pair TokenPair) (StorageResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.latest = pair
	p.latestSet = true
	return p.provisionLocked(ctx, pair)
}

func (p *CoreStorageProvisioner) provisionLocked(ctx context.Context, pair TokenPair) (StorageResult, error) {
	storage, err := db.GetStorageByMountPath(p.mountPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		addition, marshalErr := newManagedAddition(pair)
		if marshalErr != nil {
			return StorageResult{}, authError(CodeStorageInitFailed, marshalErr)
		}
		id, createErr := op.CreateStorage(ctx, model.Storage{
			MountPath:       p.mountPath,
			Driver:          _115_open.ConfigName(),
			CacheExpiration: managedCacheExpiration,
			Addition:        addition,
			Remark:          managedRemark,
		})
		if createErr != nil {
			// CreateStorage may have inserted the row before driver Init failed.
			// Never retry creation here: a second row is worse than a clear error.
			return StorageResult{StorageID: id, MountPath: p.mountPath}, authError(CodeStorageInitFailed, createErr)
		}
		if current, currentErr := op.GetStorageByMountPath(p.mountPath); currentErr != nil || current.GetStorage().Status != op.WORK {
			if currentErr == nil {
				currentErr = fmt.Errorf("managed storage is not in WORK state")
			}
			return StorageResult{StorageID: id, MountPath: p.mountPath}, authError(CodeStorageInitFailed, currentErr)
		}
		return StorageResult{StorageID: id, MountPath: p.mountPath, Connected: true, State: StateReady}, nil
	}
	if err != nil {
		return StorageResult{}, authError(CodeStorageInitFailed, err)
	}
	if storage.Driver != _115_open.ConfigName() {
		return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath}, authError(CodeStorageConflict, fmt.Errorf("managed mount is owned by another driver"))
	}

	addition, err := mergeTokenPair(storage.Addition, pair)
	if err != nil {
		return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath}, authError(CodeStorageInitFailed, err)
	}
	if err = db.UpdateStorageAddition(storage.ID, addition); err != nil {
		return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath}, authError(CodeStorageInitFailed, err)
	}
	storage.Addition = addition

	if current, currentErr := op.GetStorageByMountPath(p.mountPath); currentErr == nil {
		if open, ok := current.(*_115_open.Open115); ok {
			open.SetTokenPair(pair.AccessToken, pair.RefreshToken)
			open.GetStorage().Addition = addition
			if initErr := open.Init(ctx); initErr != nil {
				open.SetStatus(initErr.Error())
				return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath}, authError(CodeStorageInitFailed, initErr)
			}
			open.SetStatus(op.WORK)
			return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath, Connected: true, State: StateReady}, nil
		}
	}
	if err = op.LoadStorage(ctx, *storage); err != nil {
		return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath}, authError(CodeStorageInitFailed, err)
	}
	current, currentErr := op.GetStorageByMountPath(p.mountPath)
	if currentErr != nil || current.GetStorage().Status != op.WORK {
		if currentErr == nil {
			currentErr = fmt.Errorf("managed storage is not in WORK state")
		}
		return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath}, authError(CodeStorageInitFailed, currentErr)
	}
	return StorageResult{StorageID: storage.ID, MountPath: storage.MountPath, Connected: true, State: StateReady}, nil
}

func (p *CoreStorageProvisioner) Retry(ctx context.Context) (StorageResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if current, err := op.GetStorageByMountPath(p.mountPath); err == nil {
		if open, ok := current.(*_115_open.Open115); ok {
			if persistErr := open.RetryTokenPersistence(); persistErr != nil {
				return StorageResult{StorageID: open.GetStorage().ID, MountPath: p.mountPath}, authError(CodePersistenceFailed, persistErr)
			}
			return StorageResult{StorageID: open.GetStorage().ID, MountPath: p.mountPath, Connected: open.GetStorage().Status == op.WORK, State: StateReady}, nil
		}
	}
	if !p.latestSet {
		return StorageResult{MountPath: p.mountPath}, authError(CodeStorageNotFound, nil)
	}
	return p.provisionLocked(ctx, p.latest)
}

func (p *CoreStorageProvisioner) Status() StorageStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	storage, err := db.GetStorageByMountPath(p.mountPath)
	if err != nil {
		return StorageStatus{}
	}
	if err = p.ensureManagedDefaults(storage); err != nil {
		log.Warnf("failed to migrate managed 115 storage defaults: %v", err)
	}
	status := StorageStatus{
		StorageID:   storage.ID,
		MountPath:   storage.MountPath,
		Connected:   !storage.Disabled && storage.Status == op.WORK,
		Persistence: TokenPersistenceStatus{State: string(_115_open.TokenPersistenceClean)},
	}
	if current, currentErr := op.GetStorageByMountPath(p.mountPath); currentErr == nil {
		if open, ok := current.(*_115_open.Open115); ok {
			status.Connected = !open.GetStorage().Disabled && open.GetStorage().Status == op.WORK
			persistence := open.TokenPersistenceStatus()
			status.Persistence = TokenPersistenceStatus{
				State:     string(persistence.State),
				Attempts:  persistence.Attempts,
				LastError: persistence.LastError,
			}
		}
	}
	return status
}

func (p *CoreStorageProvisioner) ensureManagedDefaults(storage *model.Storage) error {
	if storage.Driver != _115_open.ConfigName() || storage.Remark != managedRemark || storage.CacheExpiration > 0 {
		return nil
	}
	if err := db.UpdateStorageCacheExpiration(storage.ID, managedCacheExpiration); err != nil {
		return err
	}
	storage.CacheExpiration = managedCacheExpiration
	if current, err := op.GetStorageByMountPath(p.mountPath); err == nil {
		current.GetStorage().CacheExpiration = managedCacheExpiration
	}
	return nil
}

func newManagedAddition(pair TokenPair) (string, error) {
	addition, err := json.Marshal(_115_open.Addition{
		RootID:       driver.RootID{RootFolderID: "0"},
		LimitRate:    1,
		PageSize:     200,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
	return string(addition), err
}

func mergeTokenPair(addition string, pair TokenPair) (string, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(addition), &values); err != nil {
		return "", err
	}
	if values == nil {
		return "", fmt.Errorf("storage addition must be a JSON object")
	}
	access, err := json.Marshal(pair.AccessToken)
	if err != nil {
		return "", err
	}
	refresh, err := json.Marshal(pair.RefreshToken)
	if err != nil {
		return "", err
	}
	values["access_token"] = access
	values["refresh_token"] = refresh
	merged, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(merged), nil
}
