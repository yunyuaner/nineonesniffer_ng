package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	f, err := os.Open("source.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	info, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	infoStr := string(info)

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

	//fmt.Println(b.String())
	unescaped := b.String()
	start = strings.Index(unescaped, "src='") + len("src='")
	end = strings.Index(unescaped[start:], "'")
	srcWithParams := unescaped[start : start+end]
	fmt.Println(srcWithParams)
	questionMarkPos := strings.Index(srcWithParams, "?")
	httpGetSrc := srcWithParams[:questionMarkPos]

	caCert, err := ioutil.ReadFile("rootCA.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	req, err := http.NewRequest("GET", httpGetSrc, nil)
	if err != nil {
		log.Fatal(err)
	}

	q := req.URL.Query()
	q.Add()
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(&req)
}
