package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gopkg.in/urfave/cli.v1" // imports as package "cli"
	"io/ioutil"
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

	uptimeSeconds := binary.BigEndian.Uint64(bytes.Split(uptime, []byte(" "))[0])

	if uptimeSeconds > 3600 {
		fmt.Println("Rebooting system...")
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	} else {
		fmt.Println("System has been up for less then 1 hour. Skipping reboot...")
	}

}

func watch(c *cli.Context) error {
	logFile := c.String("log")

	for {
		fileInfo, err := os.Stat(logFile)
		check(err)

		file, err := os.Open(logFile)
		check(err)
		defer file.Close()

		buff := make([]byte, 1024)

		file.ReadAt(buff, fileInfo.Size()-1024)

		r, _ := regexp.Compile("ETH:")
		rHash, _ := regexp.Compile("GPU[0-9]? ([0-9]*)")

		lines := strings.Split(string(buff), "\n")
		restart := false
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
						restart = true
						break
					}
				}
			}
		}
		if restart {
			reboot()
		}
		time.Sleep(time.Duration(5) * time.Minute)
	}
	return nil
}

func install(c *cli.Context) error {
	fmt.Println("completed task: ", c.Args().First())

	//TODO configure binary location

	const Service = `
	description "Foreman log service"
	author      "Sam Bolgert"
	
	start on filesystem
	stop on shutdown
	
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
	app.Version = "0.0.1a"
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
			},
			//TODO configure reboot threshold
			//TODO configure coin
			//TODO configure hash regex
		},
		{
			Name:    "install",
			Aliases: []string{"i"},
			Usage:   "Installs a systemd service to automatically run watch command on boot",
			Action: install,
		},
	}
	app.Run(os.Args)
}
