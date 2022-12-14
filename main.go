// Copyright 2022 Paul D. Shaw International Barley Hub/The James Hutton Institute

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//==========================================================================
// Hydrogen Backup Tools
// This is really just a quick wrapper on the ArangoDB arangodump and arangorestore tools
// to try and make it a bit easier to run these if you are working with a Hydrogen development
// server. Developed on Windows so not sure how well this will work on other platforms - may
// require a bit of tinkering.
//
// To use this tool just run the exe from the command line:
// >hydrogenbackup backup OR >hydrogenbackup restore then select the backup you want to
// restore. Backups should be in the same directory as this tool at the moment but that can
// change in the future to make it more flexible. This is just going to pull everything from the
// ArangoDB endpoint in the .env file.
//
// To use this you will need a file called config.env in the same directory as the .exe in YAML
// format:
// SERVER_ENDPOINT: "tcp://127.0.0.1:8529"
// SERVER_USERNAME: "username"
// SERVER_PASSWORD: "password"
//
// Obvioulsy stick your own details in there!
//=========================================================================

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/spf13/viper"
)

// This defines the directories in which backups will be created.
// Its used later on to work out which directories are backups or not.
const DIRECTORY_SIGNATURE = "hydrogenbackup"

func main() {
	// Config passed in YAML config.env file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	// Make sure we are either backing up or restoring otherwise flake out.
	if os.Args[1] == "backup" || os.Args[1] == "restore" {
		if os.Args[1] == "backup" {
			backup()
		} else if os.Args[1] == "restore" {
			restore()
		} else {
			os.Exit(1)
		}
	} else {
		fmt.Println("You need to either choose the backup or restore command line arguments")
		os.Exit(1)
	}
}

func backup() {
	// Get a timestamp (Unix seconds) to append to the backup directory name so
	// we have one directory per backup. We can pick apart the Unix seconds later
	// to get back to a timestamp when the backup was created if required.

	hbd := fmt.Sprintf(DIRECTORY_SIGNATURE+"_%d", getTimestamp())

	// Run arangodump. Now this foxed me for a bit byou you need to include the
	// --include-system-collections true if you want to backup the actual graphs and
	// not just the collections.

	cmd := exec.Command("arangodump",
		"--output-directory", hbd,
		"--server.endpoint", viper.GetString("SERVER_ENDPOINT"),
		"--server.username", viper.GetString("SERVER_USERNAME"),
		"--server.password", viper.GetString("SERVER_PASSWORD"),
		"--all-databases", "true",
		"--include-system-collections", "true")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
}

// TODO: Tidy this up as we dont really need this function
func restore() {
	listBackupDirectories()
}

// Runs the restore of data from backup directories.

func runRestore(mapName string) {
	cmd := exec.Command("arangorestore",
		"--input-directory", mapName,
		"--server.endpoint", viper.GetString("SERVER_ENDPOINT"),
		"--server.username", viper.GetString("SERVER_USERNAME"),
		"--server.password", viper.GetString("SERVER_PASSWORD"),
		"--all-databases", "true",
		"--create-database", "true",
		"--include-system-collections", "true")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
}

// Looks in the directory where the main application is running and pulls
// back any directories that match the DIRECTORY_SIGNATURE patter.
// Prompts user to enter the number of the backup they want to restore
// then runs arangorestore.

func listBackupDirectories() {
	filesMap := make(map[int]string)
	filesCounter := 1

	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	r, err := regexp.Compile(DIRECTORY_SIGNATURE)
	if err != nil {
		fmt.Printf("There is a problem with the regex.\n")
		return
	}

	for _, file := range files {
		if file.IsDir() {
			file := file.Name()
			if r.MatchString(file) == true {
				filesMap[filesCounter] = file
				filesCounter++
			}
		}
	}

	for k, v := range filesMap {
		details := strings.Split(v, "_")
		i, err := strconv.ParseInt(details[1], 10, 64)
		if err != nil {
			panic(err)
		}
		fmt.Printf("[%s] %s : %s\n", color.FgYellow.Render(k), color.FgRed.Render(v), color.FgGreen.Render(time.Unix(i, 0)))
	}
	fmt.Printf("Please enter the %s of the backup you want to restore : ", color.FgYellow.Render("[number]"))
	var first string
	fmt.Scanln(&first)

	number, err := strconv.Atoi(first)
	if err != nil {
		panic(err)
	}

	if _, ok := filesMap[number]; ok {
		// then do the backup
		fmt.Printf("%s %s %s", color.FgBlue.Render("Attempting to restore"), color.FgRed.Render(filesMap[number]),
			color.FgBlue.Render("from backup..."))
		runRestore(filesMap[number])
	}
}

// TODO : create another function that allows us to get the timestamp back from the backup directory name
func getTimestamp() int64 {
	now := time.Now()
	sec := now.Unix()
	return sec
}
