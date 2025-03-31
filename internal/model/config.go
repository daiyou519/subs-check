package model

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	JWT struct {
		Secret    string `yaml:"secret"`
		ExpiresIn int    `yaml:"expires_in"`
	} `yaml:"jwt"`
}
