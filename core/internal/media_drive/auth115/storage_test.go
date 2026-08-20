package auth115

import (
	"context"
	"testing"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestStorageConflict(t *testing.T) {
	connection, err := gorm.Open(sqlite.Open("file:auth115_storage_conflict_"+time.Now().Format("20060102150405.000000000")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := connection.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	conf.Conf = conf.DefaultConfig("data")
	db.Init(connection)
	existing := model.Storage{MountPath: "/115", Driver: "Local", Addition: `{"root_folder_path":"."}`, Remark: "keep me"}
	if err = db.CreateStorage(&existing); err != nil {
		t.Fatal(err)
	}

	provisioner := NewCoreStorageProvisioner()
	_, err = provisioner.Provision(context.Background(), TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"})
	assertAuthCode(t, err, CodeStorageConflict)

	unchanged, err := db.GetStorageById(existing.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Driver != "Local" || unchanged.Addition != existing.Addition || unchanged.Remark != existing.Remark {
		t.Fatalf("conflicting storage was modified: before=%+v after=%+v", existing, unchanged)
	}
}
