package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/urfave/cli.v1" // imports as package "cli"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func reboot() {
	uptime, err := ioutil.ReadFile("/proc/uptime")
	check(err)

	uptimeSeconds, _ := strconv.ParseFloat(string(bytes.Split(uptime, []byte(" "))[0]), 64)

	if uptimeSeconds > 3600 {
		fmt.Println("Rebooting system...")
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	} else {
		fmt.Println("System has been up for less then 1 hour. Skipping reboot...")
	}

}

func hashCheck(logFile string) bool {
	buff := tail(logFile)
	if buff != nil {
		r, _ := regexp.Compile("ETH: G")
		rHash, _ := regexp.Compile("GPU[0-9]? ([0-9]*)")

		lines := strings.Split(string(buff), "\n")
		for i := len(lines) - 1; i > 0; i-- {
			if r.Match([]byte(lines[i])) {
				fmt.Println("Line Matched")
				fmt.Println(lines[i])
				gpuMatches := rHash.FindAllStringSubmatch(lines[i], -1)
				hashes := make([]int, len(gpuMatches))
				for i, matches := range gpuMatches {
					hashes[i], _ = strconv.Atoi(matches[1])
				}
				fmt.Println(hashes)
				for _, hash := range hashes {
					if hash == 0 {
						return true
					}
				}
			}
		}
	}
	return false
}

func timeCheck() bool {
	uptime, err := ioutil.ReadFile("/proc/uptime")
	check(err)

	uptimeSeconds, _ := strconv.ParseFloat(string(bytes.Split(uptime, []byte(" "))[0]), 64)

	fmt.Println(uptimeSeconds)

	if uptimeSeconds > 86400 {
		return true
	}
	return false
}

func apiCheck(serverHash string, rigHash string) bool {
	url := fmt.Sprintf("http://%v.ethosdistro.com/?json=yes", serverHash)
	resp, err := http.Get(url)
	defer resp.Body.Close()

	check(err)

	if err != nil {
		return false
	}

	fmt.Println("api success")

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return false
	}
	type RigResponse struct {
		ServerTime int64 `json:"server_time"`
	}

	type MinerResponse struct {
		Rigs map[string]RigResponse
	}

	res := MinerResponse{}
	json.Unmarshal(body, &res)

	rig, ok := res.Rigs[rigHash]

	if !ok {
		return false
	}

	fmt.Println("Unmarshal success")

	fmt.Println(rig.ServerTime)

	t := time.Now().Unix()

	fmt.Println(t)

	if t-rig.ServerTime > 3600 {
		fmt.Println("server hasnt reported in an hour fam")
		return true
	}
	fmt.Println("Server has been up fam")
	return false
}

func tail(logFile string) []byte {

	file, err := os.Open(logFile)
	defer file.Close()

	if err != nil {
		return nil
	}

	buff := make([]byte, 1024)

	fileInfo, err := os.Stat(logFile)
	file.ReadAt(buff, fileInfo.Size()-1024)

	return buff
}

func watch(c *cli.Context) error {
	logFile := c.String("log")
	rigHash := c.String("rig")
	serverHash := c.String("server")

	for {
		restartTime := timeCheck()
		restartHash := hashCheck(logFile)
		restartAPI := apiCheck(serverHash, rigHash)

		if restartHash || restartTime || restartAPI {
			reboot()
		}
		time.Sleep(time.Duration(5) * time.Minute)
	}
}

func install(c *cli.Context) error {
	fmt.Println("completed task: ", c.Args().First())

	//TODO configure binary location

	const Service = `
	description "Foreman log service"
	author      "Sam Bolgert"
	
	start on filesystem
	
	script
		exec /home/ethos/foreman watch
	end script
	`
	const ServicePath string = "/etc/init/foreman.conf"

	_, err := os.Stat(ServicePath)

	if os.IsNotExist(err) {
		file, err := os.Create(ServicePath)
		check(err)
		file.WriteString(Service)
		file.Close()

		cmd := exec.Command("service", "foreman", "start")
		cmd.Run()
		fmt.Println("Service started...")
	} else {
		fmt.Println("File already exists. Skipping install...")
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "Foreman"
	app.Usage = "Makes sure miners keep mining"
	app.Version = "0.0.2"
	app.Commands = []cli.Command{
		{
			Name:    "watch",
			Aliases: []string{"w"},
			Usage:   "add a task to the list",
			Action:  watch,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "log, l",
					Value: "/var/run/miner.output",
					Usage: "Log file to watch for mining",
				},
				cli.StringFlag{
					Name:  "server",
					Value: "a63b06",
					Usage: "Server hash to monitor for external validation",
				},
				cli.StringFlag{
					Name:  "rig",
					Value: "4d1d7f",
					Usage: "Rig hash to monitor for external validation",
				},
			},
			//TODO configure reboot threshold
			//TODO configure coin
			//TODO configure hash regex
		},
		{
			Name:    "install",
			Aliases: []string{"i"},
			Usage:   "Installs a systemd service to automatically run watch command on boot",
			Action:  install,
		},
	}
	app.Run(os.Args)
}
