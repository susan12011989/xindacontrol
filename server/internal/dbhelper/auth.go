package dbhelper

import (
	"errors"
	"server/pkg/dbs"
	"server/pkg/entity"
)

func GetSysUserByUsername(username string) (*entity.AdminUsers, error) {
	var user entity.AdminUsers
	exist, err := dbs.DBAdmin.Where("username = ?", username).Get(&user)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New("user not found")
	}
	return &user, nil
}
