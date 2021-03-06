package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	UrlRoot string `json:"url_root"`
	UrlPara string `json:"url_para"`

	RequestPerRoutine      int32 `json:"request_per_routine"`
	LoadAgentWorkerNumber  int32 `json:"load_agent_worker_number"`
	CacheAgentWorkerNumber int32 `json:"cache_agent_worker_number"`
	CacheAgentMaxAttempt   int32 `json:"cache_agent_max_attempt"`

	MinFileSizeTenMegaByte int64 `json:"min_file_size_10_mega_byte"`
	MaxFileSizeTenMegaByte int64 `json:"max_file_size_10_mega_byte"`

	MinTestBlockSizeKiloByte int64 `json:"min_test_block_size_kilo_byte"`
	MaxTestBlockSizeKiloByte int64 `json:"max_test_block_size_kilo_byte"`
}

func (c *Config) LoadConfig(fileName string) error {
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
