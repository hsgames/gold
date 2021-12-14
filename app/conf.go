package app

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
)

func LoadConf(dir string, out interface{}) error {
	var unmarshal func([]byte, interface{}) error
	if strings.HasSuffix(dir, ".yaml") {
		unmarshal = yaml.Unmarshal
	} else if strings.HasSuffix(dir, ".json") {
		unmarshal = json.Unmarshal
	}
	if unmarshal == nil {
		return errors.Errorf("app: load config %s no unmarshal func", dir)
	}
	data, err := os.ReadFile(dir)
	if err != nil {
		return errors.Wrapf(err, "app: load config %s read file", dir)
	}
	err = unmarshal(data, out)
	if err != nil {
		return errors.Wrapf(err, "app: load config %s unmarshal", dir)
	}
	return nil
}
