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
var transcode bool
var dumpCfg bool
var script bool
var proxy bool
var keep bool

var sniffer *nineonesniffer.NineOneSniffer

const (
	ffmpegDir = "C:\\Program Files (x86)\\FormatFactory"
)

func init() {
	pwd, _ := os.Getwd()
	os.Setenv("PATH", ffmpegDir+";"+pwd+"\\tools")

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
	flag.BoolVar(&keep, "keep", false, "Keep fetched web page data")
	flag.BoolVar(&proxy, "proxy", false, "Use SOCKS5 proxy")
	flag.BoolVar(&thumbnail, "thumbnail", false, "See how many new thumbnails we newly got")
	flag.BoolVar(&script, "script", false, "Only generate script for downloading thumbnails")
	flag.BoolVar(&transcode, "transcode", false, "Convert download video files from ts to mp4 format")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.BoolVar(&dumpCfg, "dump_cfg", false, "Dump configurations")
}

const (
	tab       = "    "
	doubleTab = "        "
)

func showHelp(name string) {
	fmt.Printf("Usage: %s -mode [prefetch|fetch|parse|dl_desc|dl_video|sync|load|identify_date] [url] [dir] [count] [persist] [thumbnail] [help]\n", name)

	fmt.Printf("%sGet the newest video list\n", tab)
	fmt.Printf("%s%s -mode prefetch -count num [-proxy]\n", doubleTab, name)

	fmt.Printf("%sParse the newest video list items and persit into datastore\n", tab)
	fmt.Printf("%s%s -mode parse -dir dirname -persist\n", doubleTab, name)

	fmt.Printf("%sDownload video descriptor\n", tab)
	fmt.Printf("%s%s -mode dl_desc -url video_page_url [-presist]\n", doubleTab, name)

	fmt.Printf("%sDownload video files using per-downloaded video descriptors\n", tab)
	fmt.Printf("%s%s -mode dl_video [-url video_page_url] [-transcode] [-persist]\n", doubleTab, name)

	fmt.Printf("%sSync the lastest video list ( prefetch + parse )\n", tab)
	fmt.Printf("%s%s -mode sync -count num [-proxy] [-keep]\n", doubleTab, name)

	fmt.Printf("%sDownload thumbnails\n", tab)
	fmt.Printf("%s%s -mode load -thumbnail [-script]\n", doubleTab, name)

	fmt.Printf("%sIdentify video uploaded date according to thumbnails\n", tab)
	fmt.Printf("%s%s -mode identify_date\n", doubleTab, name)
}

func main() {

	flag.Parse()

	if help {
		showHelp(os.Args[0])
		//flag.PrintDefaults()
		os.Exit(0)
	}

	if dumpCfg {
		sniffer.DumpCfg()
		os.Exit(0)
	}

	switch mode {
	case "prefetch":
		dirname, err := sniffer.Prefetch(count, proxy)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Prefetched pages stored in %s\n", dirname)

	case "sync":
		dirname, err := sniffer.Prefetch(count, proxy)
		if err != nil {
			log.Fatal(err)
		}
		sniffer.RefreshDataset(dirname, keep)
		sniffer.Persist()

	case "parse":
		if len(dir) == 0 {
			showHelp(os.Args[0])
			os.Exit(0)
		}
		sniffer.RefreshDataset(dir, true)

		if persist {
			sniffer.Persist()
		}

	case "fetch":
		sniffer.WhatIsNew(proxy)

	case "load":
		sniffer.Load()
		if thumbnail {
			sniffer.FetchThumbnails(script)
		}

	case "identify_date":
		sniffer.IdentifyVideoUploadedDate()

	case "dl_desc":
		if len(url) == 0 {
			showHelp(os.Args[0])
			os.Exit(0)
		}
		sniffer.FetchVideoPartsDscriptor(url, persist, proxy)

	case "dl_video":
		if len(url) > 0 {
			sniffer.FetchVideoPartsDscriptor(url, persist, proxy)
		}

		if transcode {
			sniffer.Transcode = true
		}
		sniffer.FetchVideoPartsAndMerge(proxy)

	default:
		showHelp(os.Args[0])
	}
}
