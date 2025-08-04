package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

func init() {
	configFileName := "config.yaml"
	if _, err := os.Stat(configFileName); os.IsNotExist(err) {
		defauleConfig := Config{
			ImageDir:            "./images",
			Port:                "8008",
			Title:               "在线图集",
			Icon:                "https://i.051214.xyz/favicon.ico",
			Dynamic:             "true",
			LinuxdoEnable:       "false",
			WebAdderss:          "http://localhost:8008",
			LinuxdoClientId:     "",
			LinuxdoClientSecret: "",
			Secure:              "false",
			Password:            "123456",
		}
		config = defauleConfig

		content, err := yaml.Marshal(defauleConfig)
		if err != nil {
			log.Fatalf("无法序列化默认配置: %v", err)
		}
		if err := os.WriteFile(configFileName, content, 0644); err != nil {
			log.Fatalf("无法创建默认配置文件 %s: %v", configFileName, err)
		}

		log.Println("配置文件不存在，已为您创建config.yaml，请根据需要修改配置。并重启服务。")
	} else {
		// 这里可以添加读取配置文件的逻辑
		log.Println("加载配置文件:", configFileName)
		file, err := os.Open(configFileName)
		if err != nil {
			log.Fatalf("无法读取配置文件 %s: %v", configFileName, err)
		}
		defer file.Close()
		deyaml := yaml.NewDecoder(file)
		if err := deyaml.Decode(&config); err != nil {
			log.Fatalf("无法解析配置文件 %s: %v", configFileName, err)
		}
	}
}
