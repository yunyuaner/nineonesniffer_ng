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
var persist bool
var help bool
var transcode bool
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
	flag.BoolVar(&persist, "persist", false, "Persit infomation into database")
	flag.BoolVar(&keep, "keep", false, "Keep fetched web page data")
	flag.BoolVar(&proxy, "proxy", false, "Use SOCKS5 proxy")
	flag.BoolVar(&transcode, "transcode", false, "Convert download video files from ts to mp4 format")
	flag.BoolVar(&help, "help", false, "Show help")
}

const (
	tab       = "    "
	doubleTab = "        "
)

func showHelp(name string) {
	fmt.Printf("Usage: %s -mode [dl_desc|dl_video|sync|identify_date] [url] [count] [persist] [help]\n", name)

	fmt.Printf("%sDownload video descriptor\n", tab)
	fmt.Printf("%s%s -mode dl_desc -url video_page_url [-presist]\n", doubleTab, name)

	fmt.Printf("%sDownload video files using per-downloaded video descriptors\n", tab)
	fmt.Printf("%s%s -mode dl_video [-url video_page_url] [-transcode] [-persist]\n", doubleTab, name)

	fmt.Printf("%sSync the lastest video list ( prefetch + parse )\n", tab)
	fmt.Printf("%s%s -mode sync -count num [-proxy] [-keep]\n", doubleTab, name)

	fmt.Printf("%sIdentify video uploaded date\n", tab)
	fmt.Printf("%s%s -mode identify_date\n", doubleTab, name)
}

func main() {

	flag.Parse()

	if help {
		showHelp(os.Args[0])
		os.Exit(0)
	}

	switch mode {
	case "sync":
		dirname, err := sniffer.Prefetch(count, proxy)
		if err != nil {
			log.Fatal(err)
		}
		sniffer.RefreshDataset(dirname, keep)
		sniffer.Persist()

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
