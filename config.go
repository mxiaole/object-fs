package main

import (
	toml "github.com/pelletier/go-toml"
)

type mysqlConfig struct {
	Host     string
	Port     string
	Db       string
	UserName string
	Password string
}

type Config struct {
	Mysql *mysqlConfig
}

func GetDBConfig() *mysqlConfig {
	return globalConfig.Mysql
}

var globalConfig *Config

// init 配置文件初始化
func init() {
	defaultConfigPath := "config.toml"
	config, err := toml.LoadFile(defaultConfigPath)
	if err != nil {
		panic(err)
	}

	host := config.Get("mysql.host").(string)
	port := config.Get("mysql.port").(string)
	userName := config.Get("mysql.user_name").(string)
	password := config.Get("mysql.password").(string)
	db := config.Get("mysql.db").(string)

	globalConfig = &Config{
		Mysql: &mysqlConfig{
			Host:     host,
			Port:     port,
			UserName: userName,
			Password: password,
			Db:       db,
		},
	}
}
