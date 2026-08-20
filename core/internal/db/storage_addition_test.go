package db

import (
	"testing"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUpdateStorageAdditionDoesNotOverwriteStorageFields(t *testing.T) {
	connection, err := gorm.Open(sqlite.Open("file:storage_addition_test_"+time.Now().Format("20060102150405.000000000")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := connection.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	conf.Conf = conf.DefaultConfig("data")
	Init(connection)

	original := model.Storage{
		MountPath:       "/115",
		Driver:          "115 Open",
		Order:           9,
		CacheExpiration: 77,
		Remark:          "custom remark",
		Addition:        `{"root_folder_id":"0","access_token":"old"}`,
		EnableSign:      true,
		Proxy:           model.Proxy{WebdavPolicy: "use_proxy_url", DownProxyURL: "http://proxy.invalid"},
	}
	if err = CreateStorage(&original); err != nil {
		t.Fatal(err)
	}
	if err = UpdateStorageAddition(original.ID, `{"root_folder_id":"0","access_token":"new"}`); err != nil {
		t.Fatal(err)
	}
	updated, err := GetStorageById(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Addition != `{"root_folder_id":"0","access_token":"new"}` {
		t.Fatalf("addition = %s", updated.Addition)
	}
	if updated.MountPath != original.MountPath || updated.Driver != original.Driver || updated.Order != original.Order || updated.CacheExpiration != original.CacheExpiration || updated.Remark != original.Remark || !updated.EnableSign || updated.Proxy != original.Proxy {
		t.Fatalf("non-addition fields changed: before=%+v after=%+v", original, updated)
	}
}
