package monitoring

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"reflect"
)

func init() {
	runPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(runPath + "/" + "config.yml"); os.IsNotExist(err) {
		fmt.Println("Config.yml doesn't exists")
		fmt.Println("Creating config.yml with default values")
		f, err := os.Create("config.yml")
		if err != nil {
			fmt.Println(err)
			return
		}
		_, err = f.WriteString(`# api server port
port: 3000

#available modules
hostInfo: true
cpu: true
ram: true
disks: true
networkDevices: true
networkBandwidth: true
processes: true`)

		if err != nil {
			fmt.Println(err)
			f.Close()
			return
		}
		f.Close()
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(runPath)
	err = viper.ReadInConfig()
	if err != nil {
		panic("config file error")
	}
}

func available(module string) bool {
	config := viper.Get(module)
	if config != nil {
		if reflect.TypeOf(config).String() == "bool" {
			return config.(bool)
		}
		return false
	}
	return false
}
