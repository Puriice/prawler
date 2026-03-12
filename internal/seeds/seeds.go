package seeds

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
)

var (
	ErrEmptySeed = errors.New("Empty seed loaded")
)

func LoadSeeds() ([]string, error) {
	seedFile := flag.String("seed-file", "./seeds.json", "A path to a json file containing array of seeds for crawling")

	flag.Parse()

	content, err := os.ReadFile(*seedFile)

	if err != nil {
		log.Fatal(err)
	}

	var seeds []string

	err = json.Unmarshal(content, &seeds)

	if err != nil {
		log.Fatal(err)
	}

	if len(seeds) == 0 {
		return nil, ErrEmptySeed
	}

	return seeds, nil
}
