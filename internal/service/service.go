package service

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"shortener/internal/config"
	"shortener/internal/storage"
	"strings"
	"sync"
)

type FileType struct {
	UUID        string `json:"UUID"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var mu sync.RWMutex

func Save(ft FileType) {

	if config.Server.DSN != "" {
		SaveToDB(ft)
		return
	}

	if config.Server.Storage != "" {
		SaveToFile(ft)
		SaveToMemory(ft)
		return
	}

	SaveToMemory(ft)

}

func SaveToMemory(ft FileType) {
	storage.Save(ft.ShortURL, ft.OriginalURL)

}

func SaveToDB(ft FileType) {

}
func ReadFromDB(ft FileType) {

}

func SaveToFile(ft FileType) {

	_, err := os.Stat(config.Server.Storage)
	if err != nil {
		f, err := os.Create(config.Server.Storage)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		defer f.Close()

		f.Write([]byte("[\n]"))
	}

	f, err := os.OpenFile(config.Server.Storage, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	defer f.Close()

	_, err = f.Seek(-1, io.SeekEnd)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	var bs [1]byte
	f.Read(bs[:])

	stat, err := os.Stat(config.Server.Storage)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	if string(bs[0]) == "]" {
		f.Truncate(stat.Size() - 1)
	}

	mu.Lock()
	//fmt.Fprintf(f, "    ")
	//err = json.NewEncoder(f).Encode(ft)
	//fmt.Printf("err: %v\n", err)
	fmt.Fprintf(f, `    {"uuid":"%s","short_url":"%s","original_url":"%s"},`+"\n]", ft.UUID, ft.ShortURL, ft.OriginalURL)
	mu.Unlock()

}

func ReadFromFile() {

	f, err := os.OpenFile(config.Server.Storage, os.O_RDONLY, os.ModePerm)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}

	bs, err := io.ReadAll(f)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	bs = bs[1 : len(bs)-1]

	ss := strings.Split(strings.TrimSpace(string(bs)), "\n")

	for i := range ss {

		s := strings.TrimSpace(strings.ReplaceAll(ss[i], "},", "}"))
		var ft FileType

		err := json.Unmarshal([]byte(s), &ft)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}

		storage.Save(ft.ShortURL, ft.OriginalURL)

	}

}
