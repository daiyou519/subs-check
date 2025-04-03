package model

type Config struct {
	Server struct {
		Port int    `json:"port"`
		Host string `json:"host"`
	} `json:"server"`
	Database struct {
		Path string `json:"path"`
	} `json:"database"`
	JWT struct {
		Secret    string `json:"secret"`
		ExpiresIn int    `json:"expires_in"`
	} `json:"jwt"`
}
