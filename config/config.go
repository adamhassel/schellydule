package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/BurntSushi/toml"
)

// An MID MUST be 18 digits
const midLength = 18

type confdata struct {
	Token     string `toml:"token"`
	MID       string `toml:"mid"`
	DarkHours int    `toml:"darkhours"`
	Hours     int    `toml:"hours"`
	Port      int    `toml:"port"`
	ShellyIP  string `toml:"shelly_ip"`
}

type Config struct {
	token     string
	mid       string
	darkHours int
	hours     int
	port      int
	shellyIP  net.IP
}

var conf Config

func init() {

}

func LoadConfig(filename string) (Config, error) {
	err := conf.Load(filename)
	return conf, err
}

func GetConf() Config {
	return conf
}

func (c Config) Token() string {
	return c.token
}

func (c Config) MID() string {
	return c.mid
}

func (c Config) IP() net.IP {
	return c.shellyIP
}

func (c Config) Port() int {
	return c.port
}

func (c Config) Hours() int {
	return c.hours
}

func (c Config) DarkHours() int {
	return c.darkHours
}

func (c *Config) Load(filename string) error {
	tomlData, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", filename, err)
	}
	var d confdata
	_, err = toml.Decode(string(tomlData), &d)
	if err != nil {
		return err
	}
	c.mid = d.MID
	c.token = d.Token
	if len(c.mid) != midLength {
		return fmt.Errorf("MID is not %d digits", midLength)
	}
	if c.token == "" {
		return errors.New("empty token")
	}
	c.shellyIP = net.ParseIP(d.ShellyIP)
	c.darkHours = defaultValue(d.DarkHours, 3)
	c.hours = defaultValue(d.Hours, 12)
	c.port = d.Port
	return nil
}

func defaultValue(i, d int) int {
	if i == 0 {
		return d
	}
	return i
}
