package main

import (
	"os"
)

type Config struct {
	ImageDir     string
	Secure       string
	Password     string
	Port         string
	Title        string
	Icon         string
	Adderss      string
	Dynamic      string
	Linuxdo      string
	ClientId     string
	ClientSecret string
}

var config = Config{}

var categoryCache []Category

type Category struct {
	Name        string
	EncodedName string
	CoverImage  string
}

var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

func initEnv() {
	envDefaults := map[string]struct {
		target     *string
		envKey     string
		defaultVal string
	}{
		"ImageDir":     {&config.ImageDir, "SITE_DIR", "./images"},
		"Port":         {&config.Port, "SITE_PORT", "8008"},
		"Title":        {&config.Title, "SITE_TITLE", "在线图集"},
		"Icon":         {&config.Icon, "SITE_ICON", "https://i.obai.cc/favicon.ico"},
		"Dynamic":      {&config.Dynamic, "SITE_DYNAMIC", "false"},
		"Linuxdo":      {&config.Linuxdo, "SITE_LINUXDO", "false"},
		"Address":      {&config.Adderss, "SITE_Address", "http://localhost:8008"},
		"ClientId":     {&config.ClientId, "SITE_CLIENT_ID", ""},
		"ClientSecret": {&config.ClientSecret, "SITE_CLIENT_SECRET", ""},
		"Secure":       {&config.Secure, "SITE_SECURE", "false"},
		"Password":     {&config.Password, "SITE_PASSWORD", ""},
	}

	for _, cfg := range envDefaults {
		if val := os.Getenv(cfg.envKey); val != "" {
			*cfg.target = val
		} else {
			*cfg.target = cfg.defaultVal
		}
	}

}
