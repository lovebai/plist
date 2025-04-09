package main

import "os"

type Config struct {
	ImageDir string
	Secure   string
	Password string
	Port     string
	Title    string
	Icon     string
	Url      string
	Dynamic  string
	Linuxdo  string
}

var config = Config{
	ImageDir: "./images",
	Password: "",
	Port:     "8009",
	Title:    "在线图集",
	Icon:     "https://i.obai.cc/favicon.ico",
	Dynamic:  "false",
	Url:      "http://localhost:8009",
	Secure:   "false",
	Linuxdo:  "false",
}

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

func initConfig() {
	envVars := map[string]*string{
		"SITE_DIR":      &config.ImageDir,
		"SITE_SECURE":   &config.Secure,
		"SITE_PASSWORD": &config.Password,
		"SITE_PORT":     &config.Port,
		"SITE_TITLE":    &config.Title,
		"SITE_ICON":     &config.Icon,
		"SITE_DYNAMIC":  &config.Dynamic,
		"SITE_LINUXDO":  &config.Linuxdo,
		"SITE_URL":      &config.Url,
	}

	for env, conf := range envVars {
		if val := os.Getenv(env); val != "" {
			*conf = val
		}
	}
}
