package main

import (
	"encoding/xml"
	"os"
	"path"
)

type StrategySettings struct {
	Clients    []Client `xml:"Clients>Client"`
	LogPath    string
	AdvisorUrl string
}

type Client struct {
	Key            string  `xml:",attr"`
	Firm           string  `xml:",attr"`
	Portfolio      string  `xml:",attr"`
	PublishCandles bool    `xml:",attr"`
	Amount         float64 `xml:",attr"`
	MaxAmount      float64 `xml:",attr"`
	Weight         float64 `xml:",attr"`
	Port           int     `xml:",attr"`
}

func loadConfig() (StrategySettings, error) {
	var config = StrategySettings{}
	var fn = executablePath("StrategySettings.config")
	var err = decodeXmlFile(fn, &config)
	if err != nil {
		return StrategySettings{}, err
	}
	return config, nil
}

func executablePath(filePath string) string {
	var exePath, err = os.Executable()
	if err == nil {
		filePath = path.Join(path.Dir(exePath), filePath)
	}
	return filePath
}

func decodeXmlFile(filePath string, v interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return xml.NewDecoder(file).Decode(v)
}
