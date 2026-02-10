package dbhelper

import "server/pkg/dbs"

func GetAllOssUrl() ([]string, error) {
	var ossUrls []string
	err := dbs.DBAdmin.Table("global_oss_url").Cols("url").Find(&ossUrls)
	if err != nil {
		return nil, err
	}
	return ossUrls, nil
}

func GetAllGostIPs() ([]string, error) {
	var hosts []string
	err := dbs.DBAdmin.Table("servers").Cols("host").Where("server_type = 2 and status = 1").Find(&hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}
