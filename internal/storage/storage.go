package storage

var storage map[string]string = map[string]string{}

func Save(link, url string) error {

	storage[link] = url

	return nil
}

func Read(link string) (string, error) {

	if l, ok := storage[link]; ok {

		return l, nil
	}

	return "", nil

}
