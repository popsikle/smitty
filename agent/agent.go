package agent

import (
	"fmt"
	"os/exec"
	"strings"
	"github.com/garyburd/redigo/redis"
)

type OutlandConfig struct {
	Redis        bool     `yaml:"redis,omitempty"`
}

var outlandConfig map[string]OutlandConfig

func UpdateMaster(master_name string, ip string, port string) bool {
	Debug(fmt.Sprintf("Updating master %s to %s:%s.", master_name, ip, port))
	ol_config := outlandConfig[master_name]
	old_address    := ol_config['host']
	old_port       := ol_config['port']

	if port != old_port && ip != old_address {
		outlandConfig[master_name]['host'] = ip
		outlandConfig[master_name]['port'] = port
		if master_name == 'main_write' {
			outlandConfig['main_read']['host'] = ip
			outlandConfig['main_read']['port'] = port
		}
		return true
	}

	return false
}

func LoadOutlandConfig() {
	Debug("Loading Outland config.")
	ReadYaml(Settings.AppConfig, &outlandConfig)
}

func SaveOutlandConfig() {
	Debug("Saving Outland config.")
	WriteYaml(Settings.AppConfig, &outlandConfig)
}

func RestartOutland() error {
	Debug("Restarting Outland.")
	out, err := exec.Command(Settings.RestartCommand).Output()

	if err != nil {
		Debug(fmt.Sprintf("Cannot restart outland. output: %s. error: %s", out, err))
	}

	return err
}

func GetSentinel() (sentinel string) {
	address := ComposeRedisAddress(Settings.SentinelIp, Settings.SentinelPort)
	return address
}

func SwitchMaster(master_name string, ip string, port string) error {
	Debug("Received switch-master.")
	if UpdateMaster(master_name, ip, port) {
		SaveOutlandConfig()
		err := RestartTwemproxy()
		return err
	} else {
		return nil
	}
}

func ValidateCurrentMaster() error {
	c, err := redis.Dial("tcp", GetSentinel())
	if err != nil {
		return err
	}

	reply, err := redis.Values(c.Do("SENTINEL", "masters"))

	if err != nil {
		return err
	}

	var sentinel_info []string

	reply, err = redis.Scan(reply, &sentinel_info)
	if err != nil {
		return err
	}
	master_name := sentinel_info[1]
	ip          := sentinel_info[3]
	port        := sentinel_info[5]

	err = SwitchMaster(master_name, ip, port)

	return err
}

func SubscribeToSentinel() {
	sentinel := GetSentinel()
	c, err := redis.Dial("tcp", sentinel)
	if err != nil {
		Fatal("Cannot connect to redis sentinel:", sentinel)
	}

	err = ValidateCurrentMaster()
	if err != nil {
		Fatal("Cannot switch to current master")
	}
	psc := redis.PubSubConn{c}
	Debug("Subscribing to sentinel (+switch-master).")
	psc.Subscribe("+switch-master")
	psc.Subscribe("+slave-reconf-done")
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			Debug(fmt.Sprintf("%s: message: %s", v.Channel, v.Data))
			data := strings.Split(string(v.Data), string(' '))
			switch ch := v.Channel
			case "+switch-master":
				SwitchMaster(data[0], data[3], data[4])
			case "+slave-reconf-done"
				SwitchSlave(data[1], data[2], data[3])
			}
		case redis.Subscription:
			Debug(fmt.Sprintf("%s: %s %d", v.Channel, v.Kind, v.Count))
		case error:
			Fatal("Error with redis connection:", psc)
		}
	}
}

func Run() {
	LoadOutlandConfig()
	SubscribeToSentinel()
}
