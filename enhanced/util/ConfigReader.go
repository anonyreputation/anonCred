package util
import (
	"os"
	"log"
	"bufio"
	"strings"
)


var config map[string]string

func ReadConfig() map[string]string{
	config = make(map[string]string)
	readConnProperties()
	readLocalProperties()
	return config
}

func GetParameter(name string) string {
	if config == nil {
		config = make(map[string]string)
		readLocalProperties()
		readConnProperties()
	}
	return config[name]
}

func readLocalProperties() {
	readConfig("config/local.properties")
}

func readConnProperties() {
	readConfig("config/conn.properties")
}


func readConfig(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		s := strings.Split(line,"=")
		config[s[0]] = s[1]
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}