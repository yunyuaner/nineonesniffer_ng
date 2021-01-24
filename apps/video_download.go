package main

import (
	nineonesniffer "github.com/yunyuaner/nineonesniffer_ng"
)

var sniffer *nineonesniffer.NineOneSniffer

func init() {
	sniffer = new(nineonesniffer.NineOneSniffer)
	sniffer.Init()
}

func main() {
	sniffer.FetchVideoPartsAndMerge()
}
