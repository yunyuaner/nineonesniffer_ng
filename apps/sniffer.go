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
var help bool

var sniffer *nineonesniffer.NineOneSniffer

func init() {
	initParameters()
	sniffer = new(nineonesniffer.NineOneSniffer)
	sniffer.Init()
}

func initParameters() {
	flag.StringVar(&mode, "mode", "", "prefetch|fetch|parse|load|dl_desc|dl_video")
	flag.IntVar(&count, "count", 10, "Fetch newest video list count")
	flag.StringVar(&url, "url", "", "url of the detailed video page")
	flag.StringVar(&dir, "dir", "", "Target directory")
	flag.BoolVar(&persist, "persist", false, "Persit infomation into database")
	flag.BoolVar(&thumbnail, "thumbnail", false, "See how many new thumbnails we newly got")
	flag.BoolVar(&help, "help", false, "Show help")
}

const (
	tab       = "    "
	doubleTab = "        "
)

func showHelp(name string) {
	fmt.Printf("Usage: %s -mode [prefetch|fetch|parse|dl_desc|dl_video|sync|load] [url] [dir] [count] [persist] [thumbnail] [help]\n", name)
	fmt.Printf("%sGet the newest video list\n", tab)
	fmt.Printf("%s%s -mode prefetch -count num\n", doubleTab, name)
	fmt.Printf("%sParse the newest video list items and persit into datastore\n", tab)
	fmt.Printf("%s%s -mode parse -dir dirname -persist\n", doubleTab, name)
	fmt.Printf("%sDownload video descriptor\n", tab)
	fmt.Printf("%s%s -mode dl_desc -url video_page_url\n", doubleTab, name)
	fmt.Printf("%sDownload video files using per-downloaded video descriptors\n", tab)
	fmt.Printf("%s%s -mode dl_video\n", doubleTab, name)
	fmt.Printf("%sSync video date set with more detail items\n", tab)
	fmt.Printf("%s%s -mode sync\n", doubleTab, name)
	fmt.Printf("%sGenerate thumbnails fetching script\n", tab)
	fmt.Printf("%s%s -mode load -thumbnail\n", doubleTab, name)
}

func main() {

	flag.Parse()

	if help {
		showHelp(os.Args[0])
		//flag.PrintDefaults()
		os.Exit(0)
	}

	switch mode {
	case "prefetch":
		dirname, err := sniffer.Prefetch(count)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Prefetched pages stored in %s\n", dirname)

	case "sync":
		if len(dir) == 0 {
			showHelp(os.Args[0])
			os.Exit(0)
		}
		sniffer.RefreshDataset(dir)
		sniffer.Sync()

	case "parse":
		if len(dir) == 0 {
			showHelp(os.Args[0])
			os.Exit(0)
		}
		sniffer.RefreshDataset(dir)

		if persist {
			sniffer.Persist()
		}

	case "fetch":
		sniffer.WhatIsNew()

	case "load":
		sniffer.Load()
		if thumbnail {
			sniffer.FetchThumbnails()
		}

	case "dl_desc":
		if len(url) == 0 {
			showHelp(os.Args[0])
			os.Exit(0)
		}
		sniffer.FetchVideoPartsDscriptor(url)

	case "dl_video":
		sniffer.FetchVideoPartsAndMerge()

	default:
		showHelp(os.Args[0])
	}
}
