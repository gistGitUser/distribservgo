package main

import (
	"fmt"
	"github.com/spf13/viper"
	"godistrserv/pkg/monitoring"
	"net"
	"net/http"
	"reflect"
	"strconv"
)

var availableConfigPort string

func portAvailable(configPort string) {
	fmt.Println("Setting port to " + configPort)
	ln, err := net.Listen("tcp", ":"+configPort)
	if err != nil {
		fmt.Println("Can't listen on port " + configPort)
		newConfigPort, errStrConv := strconv.Atoi(configPort)
		if errStrConv != nil {
			panic(errStrConv)
		}

		newConfigPort = newConfigPort + 1
		newConfigPortString := strconv.Itoa(newConfigPort)
		portAvailable(newConfigPortString)
	} else {
		_ = ln.Close()
		availableConfigPort = configPort
	}
}

func setPort() string {
	config := viper.Get("port")
	if config != nil {
		if reflect.TypeOf(config).String() == "int" {
			return strconv.Itoa(config.(int))
		}
		return "3000"
	}
	return "3000"
}

func main() {
	configPort := setPort() //Read port from config.yml

	portAvailable(configPort)

	fmt.Println("Starting Server at port " + availableConfigPort)

	monitoring.CreateEndPoint("mode", "")
	if err := http.ListenAndServe(":"+availableConfigPort, nil); err != nil {
		fmt.Println(err)
	}

}
