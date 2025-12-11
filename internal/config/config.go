package config

import (
	"flag"
	"net/url"
	"os"

	"go.uber.org/zap"
)

var Sugar zap.SugaredLogger

var Server struct {
	A       string
	B       string
	Path    string
	Host    string
	Scheme  string
	Storage string
	DSN     string
}

func Configure() {
	//var defaultDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", "localhost", "postgres", `postgres`, "postgres")

	flag.StringVar(&Server.A, "a", "localhost:8080", "server address and port")
	flag.StringVar(&Server.B, "b", "http://localhost:8080/", "server address and port")
	flag.StringVar(&Server.Storage, "f", "", "storage")
	flag.StringVar(&Server.DSN, "d", "", "database service name")
	flag.Parse()

	if a := os.Getenv("SERVER_ADDRESS"); a != "" {
		Server.A = a
	}
	if b := os.Getenv("BASE_URL"); b != "" {
		Server.B = b
	}
	if f := os.Getenv("FILE_STORAGE_PATH"); f != "" {
		Server.Storage = f
	}
	if d := os.Getenv("DATABASE_DSN"); d != "" {
		Server.DSN = d
	}

	p, _ := url.Parse(Server.B)
	Server.Path = p.Path

	if Server.Path == "" {
		Server.Path = "/"
	}

	Server.Host = p.Host
	Server.Scheme = p.Scheme
}
