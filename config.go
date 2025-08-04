package main

type Config struct {
	ImageDir            string `yaml:"image_dir"`
	Secure              string `yaml:"secure"`
	Password            string `yaml:"password"`
	Port                string `yaml:"port"`
	Title               string `yaml:"title"`
	Icon                string `yaml:"icon"`
	Dynamic             string `yaml:"dynamic"`
	WebAdderss          string `yaml:"web_adderss"`
	LinuxdoEnable       string `yaml:"linuxdo_enable"`
	LinuxdoClientId     string `yaml:"linuxdo_client_id"`
	LinuxdoClientSecret string `yaml:"linuxdo_client_secret"`
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
