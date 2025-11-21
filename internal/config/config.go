package config

import (
	"flag"

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
}

func Configure() {

	flag.StringVar(&Server.A, "a", "localhost:8080", "server address and port")
	flag.StringVar(&Server.B, "b", "http://localhost:8080/", "server address and port")
	flag.StringVar(&Server.Storage, "f", "storage.db", "storage")
	flag.Parse()
}
