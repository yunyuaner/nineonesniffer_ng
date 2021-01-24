package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var url string

func init() {
	flag.StringVar(&url, "url", "", "url of the detailed video page")
}

func Decode(infoStr string) (*string, *string) {
	start := strings.Index(infoStr, "\"") + 1
	end := strings.LastIndex(infoStr, "\"")

	escapedSrc := infoStr[start:end]

	var b bytes.Buffer

	for where := 0; where < len(escapedSrc); where += 3 {
		n := strings.Index(escapedSrc[where:], "%")
		val := escapedSrc[where+n+1 : where+n+3]
		integerCh, _ := strconv.ParseInt(val, 16, 32)
		b.WriteByte(byte(integerCh))
	}

	unescaped := b.String()
	start = strings.Index(unescaped, "src='") + len("src='")
	end = strings.Index(unescaped[start:], "'")
	srcWithParams := unescaped[start : start+end]
	//fmt.Println(srcWithParams)
	questionMarkPos := strings.Index(srcWithParams, "?")
	httpGetSrc := srcWithParams[:questionMarkPos]
	slash := strings.LastIndex(httpGetSrc, "/")
	name := httpGetSrc[slash+1:]

	return &name, &srcWithParams
}

func Extract(fileContent string) (*string, error) {
	//f, err := os.Open(filename)
	//if os.IsNotExist(err) {
	//	return nil, err
	//}

	//defer f.Close()

	//fileContent, err := ioutil.ReadAll(f)
	//if err != nil {
	//	return nil, err
	//}

	r := regexp.MustCompile(`document.write\(strencode2\(.*\)\);`)
	info := r.FindString(string(fileContent))
	//fmt.Println(info)
	//info = info[len("document.write(strencode2(\"") : len(info)-len("\"));")]
	//fmt.Println(info)
	return &info, nil
}

func GetDetailedVideoPage(url string) (*string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	buffer := bytes.NewBuffer(body)
	content := buffer.String()
	return &content, nil
}

func main() {
	flag.Parse()

	if len(url) == 0 {
		fmt.Println("url shouldn't be empty")
		os.Exit(0)
	}

	content, err := GetDetailedVideoPage(url)
	if err != nil {
		log.Fatal(err)
	}

	info, err := Extract(*content)
	if err != nil {
		log.Fatal(err)
	}

	name, src := Decode(*info)
	cmd := exec.Command("wget", "-O", "./m3u8/todo/"+*name, *src)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
