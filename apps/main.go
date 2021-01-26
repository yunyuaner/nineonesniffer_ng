package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	nineonesniffer "github.com/yunyuaner/nineonesniffer_ng"
)

var mode string
var url string
var count int
var dir string
var persist bool
var thumbnail bool

var sniffer *nineonesniffer.NineOneSniffer

func init() {
	flag.StringVar(&mode, "mode", "", "prefetch|fetch|parse|load|dl_desc|dl_video")
	flag.IntVar(&count, "count", 10, "Fetch newest video list count")
	flag.StringVar(&url, "url", "", "url of the detailed video page")
	flag.StringVar(&dir, "dir", "", "Target directory")
	flag.BoolVar(&persist, "persist", false, "Persit infomation into database")
	flag.BoolVar(&thumbnail, "thumbnail", false, "See how many new thumbnails we newly got")

	sniffer = new(nineonesniffer.NineOneSniffer)
	sniffer.Init()
}

func main() {

	flag.Parse()

	switch mode {
	case "prefetch":
		dirname, err := sniffer.Prefetch(count)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Prefetched pages stored in %s\n", dirname)

	case "parse":
		if len(dir) == 0 {
			flag.PrintDefaults()
			os.Exit(0)
		}
		sniffer.RefreshDataset(dir)

		if persist {
			sniffer.Persist()
		}

	case "fetch":
		sniffer.Fetch()

	case "load":
		sniffer.Load()
		if thumbnail {
			sniffer.FetchThumbnails()
		}

	case "dl_desc":
		if len(url) == 0 {
			flag.PrintDefaults()
			os.Exit(0)
		}
		sniffer.FetchVideoPartsDscriptor(url)

	case "dl_video":
		sniffer.FetchVideoPartsAndMerge()

	default:
		flag.PrintDefaults()
	}
}
