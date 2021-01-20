package main

import (
	"flag"

	nineonesniffer "github.com/yunyuaner/nineonesniffer_ng"
)

var sniffer *nineonesniffer.NineOneSniffer

func init() {
	sniffer = new(nineonesniffer.NineOneSniffer)
	sniffer.Init()
}

func main() {
	prefetch := flag.Bool("prefetch", false, "Fetch newest video list")
	fetch := flag.Bool("fetch", false, "Fetch newest detailed video items")
	refresh := flag.Bool("refresh", false, "Refresh dataset")
	load := flag.Bool("load", false, "Load data from disk")

	flag.Parse()

	if *prefetch {
		sniffer.Prefetch()
	} else if *fetch {
		sniffer.Fetch()
	} else if *refresh {
		sniffer.RefreshDataset()
	} else if *load {
		sniffer.Load()
	} else {
		flag.PrintDefaults()
	}
}
