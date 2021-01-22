package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
)

var filename string
var auto bool

func init() {
	flag.StringVar(&filename, "file", "", "m3u8 file name to parse")
	flag.BoolVar(&auto, "auto", false, "download video parts, merge and transcode automatically")
}

func main() {
	flag.Parse()
	file, err := os.Open(filename)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	defer file.Close()

	fileContent, _ := ioutil.ReadAll(file)
	fileContentStr := string(fileContent)
	//fmt.Println(fileContentStr)

	r := regexp.MustCompile(`[0-9]*\.ts`)
	videoParts := r.FindAllString(fileContentStr, -1)
	var videoPartsWithoutSuffix []int
	for _, part := range videoParts {
		val, _ := strconv.Atoi(part[:len(part)-3])
		videoPartsWithoutSuffix = append(videoPartsWithoutSuffix, val)
	}

	sort.Ints(videoPartsWithoutSuffix)
	//fmt.Println(videoPartsWithoutSuffix)
	finalFileName := strconv.Itoa(videoPartsWithoutSuffix[0] / 10)
	filePartsCount := strconv.Itoa(videoPartsWithoutSuffix[len(videoPartsWithoutSuffix)-1] % 100)

	fmt.Printf("final file name - %s, file parts count - %s\n", finalFileName, filePartsCount)
	if auto {
		cmd := exec.Command("./get.sh", finalFileName, filePartsCount)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		cmd = exec.Command("./cat.sh", finalFileName, filePartsCount)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}
