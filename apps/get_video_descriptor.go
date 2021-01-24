package main

import (
	"flag"

	nineonesniffer "github.com/yunyuaner/nineonesniffer_ng"
)

var sniffer *nineonesniffer.NineOneSniffer
var url string

func init() {
	flag.StringVar(&url, "url", "", "url of the detailed video page")
	sniffer = new(nineonesniffer.NineOneSniffer)
	sniffer.Init()
}

func main() {
	flag.Parse()
	sniffer.FetchVideoPartsDscriptor(url)
}
