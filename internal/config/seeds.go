package config

import (
	"errors"
	"flag"

	"github.com/purrice/prawler/internal/file"
)

var (
	ErrEmptySeed = errors.New("Empty seed loaded")
)

func LoadSeeds() ([]string, error) {
	seedFile := flag.String("seed-file", "./seeds.json", "A path to a json file containing array of seeds for crawling")
	flag.Parse()

	var seeds []string

	file.LoadJson(*seedFile, &seeds)

	if len(seeds) == 0 {
		return nil, ErrEmptySeed
	}

	return seeds, nil
}
