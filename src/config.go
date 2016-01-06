package main

import (
	"encoding/json"
	"io/ioutil"
)

type config struct {
	UrlRoot string `json:"url_root"`
	UrlPara string `json:"url_para"`

	DataDir string `json:"data_dir"`

	RequestPerRoutine int32 `json:"request_per_routine"`
	RoutineNumber     int32 `json:"routine_number"`

	MinFileSizeTenMegaByte int64 `json:"min_file_size_10_mega_byte"`
	MaxFileSizeTenMegaByte int64 `json:"max_file_size_10_mega_byte"`

	MinTestBlockSizeMegaByte int64 `json:"min_test_block_size_mega_byte"`
	MaxTestBlockSizeMegaByte int64 `json:"max_test_block_size_mega_byte"`
}

func (c *config) loadConfig(fileName string) error {
	// Read config file
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	// Parse config file
	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	return nil
}
