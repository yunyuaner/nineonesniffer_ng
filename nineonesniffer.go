package nineonesniffer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type NineOneSniffer struct {
	fetcher   nineOneFetcher
	parser    nineOneParser
	vds       *VideoDataSet
	Transcode bool
	confmgr   *NineOneConfManager
	persister *nineonePersister
	obs       *obscurer
}

func (sniffer *NineOneSniffer) Init() {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(workDir)
	}

	/* configuration setup */
	sniffer.confmgr = new(NineOneConfManager)
	configFile := filepath.Join(workDir, "configs", "NineOneSniffer.conf")
	sniffer.confmgr.Start(configFile)
	// sniffer.confmgr.showConfig()

	/* database setup */
	sniffer.persister = new(nineonePersister)
	sniffer.persister.sniffer = sniffer
	sniffer.persister.init()

	sniffer.prerequisite()

	sniffer.obs = new(obscurer)
	sniffer.obs.sniffer = sniffer
	sniffer.obs.proxySetup()

	sniffer.fetcher.sniffer = sniffer
	sniffer.parser.sniffer = sniffer
	sniffer.fetcher.userAgent = sniffer.confmgr.config.userAgent
	sniffer.vds = &VideoDataSet{}
	sniffer.Transcode = false
}

func (sniffer *NineOneSniffer) prerequisite() {
	confmgr := sniffer.confmgr
	dirs := []string{confmgr.config.configBaseDir,
		confmgr.config.dataBaseDir,
		confmgr.config.tempDir,
		confmgr.config.thumbnailBaseDir,
		confmgr.config.thumbnailNewDir,
		confmgr.config.videoListBaseDir,
		confmgr.config.videoMergedDir,
		confmgr.config.videoPartsDescDoneDir,
		confmgr.config.videoPartsDescTodoDir,
		confmgr.config.videoPartsDir,
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
	}
}

func (sniffer *NineOneSniffer) ProxyQuery() {
	sniffer.obs.proxyInvalidate()
	// fmt.Println(sniffer.obs.queryhideme())
	// fmt.Println(sniffer.obs.queryspys())
}

func (sniffer *NineOneSniffer) Prefetch(count int, useProxy bool) (string, error) {
	return sniffer.fetcher.fetchVideoList(count, useProxy)
}

func (sniffer *NineOneSniffer) FetchThumbnails(script bool) {
	sniffer.fetcher.fetchThumbnails(script)
}

func (sniffer *NineOneSniffer) FetchVideoPartsDscriptor(url string, saveToDb bool, useProxy bool) {
	if err := sniffer.fetcher.fetchVideoPartsDescriptor(url, saveToDb, useProxy); err != nil {
		fmt.Println(err)
	}
}

func (sniffer *NineOneSniffer) FetchVideoPartsAndMerge(useProxy bool) {
	if err := sniffer.fetcher.fetchVideoPartsAndMerge(useProxy); err != nil {
		fmt.Println(err)
	}
}

func (sniffer *NineOneSniffer) RefreshDataset(dirname string, keep bool) {
	sniffer.parser.refreshDataset(dirname, keep)
	fmt.Printf("Got %d items\n", sniffer.vds.size())
}

func (sniffer *NineOneSniffer) IdentifyVideoUploadedDate() {
	sniffer.parser.identifyVideoUploadedDate()
}

func (sniffer *NineOneSniffer) Persist() {
	sniffer.vds.save(sniffer.persister)
}

func (sniffer *NineOneSniffer) Sync() {
	sniffer.vds.sync(sniffer.persister)
}

func (sniffer *NineOneSniffer) ParseVideoList(filename string) {
	sniffer.parser.parseVideoList(filename)
}
