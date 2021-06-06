package nineonesniffer

import (
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
)

type nineOneFetcher struct {
	sniffer   *NineOneSniffer
	cookies   []*http.Cookie
	userAgent string
}

func (fetcher *nineOneFetcher) parseCookies(filename string) ([]*http.Cookie, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		tokenPos := strings.Index(line, "=")
		if tokenPos == -1 {
			continue
		}
		name := line[:tokenPos]
		val := line[tokenPos+1:]

		fetcher.cookies = append(fetcher.cookies, &http.Cookie{Name: name, Value: val})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return fetcher.cookies, nil
}

func (fetcher *nineOneFetcher) wget(url string, outputFile string, useProxy bool) error {
	var resp *http.Response
	var reader io.ReadCloser

	client := fetcher.newHTTPSClient(useProxy)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", fetcher.userAgent)
	req.Header.Add("Accept-Encoding", "gzip")

	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		err = errors.New(url + "resp.StatusCode: " + strconv.Itoa(resp.StatusCode))
		return err
	}

	defer resp.Body.Close()

	lastModified := resp.Header.Get("Last-Modified")
	t, err := time.Parse(time.RFC1123, lastModified)
	keepFileTimestamp := true
	if err != nil {
		keepFileTimestamp = false
	}

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		defer reader.Close()
	default:
		reader = resp.Body
	}

	f, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, reader); err != nil {
		return err
	}

	f.Close()
	if keepFileTimestamp {
		os.Chtimes(outputFile, t, t)
	}

	return nil
}

func (fetcher *nineOneFetcher) newHTTPClient(useProxy bool) *http.Client {
	if useProxy {
		baseDialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		dialSocksProxy, _ := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, baseDialer)
		contextDialer, _ := dialSocksProxy.(proxy.ContextDialer)
		dialContext := contextDialer.DialContext

		httpClient := &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           dialContext,
				MaxIdleConns:          10,
				IdleConnTimeout:       60 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
			},
		}

		return httpClient
	} else {
		return &http.Client{}
	}
}

func (fetcher *nineOneFetcher) newHTTPSClient(useProxy bool) *http.Client {
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    60 * time.Second,
		DisableCompression: true,
		TLSClientConfig:    cfg,
	}

	if useProxy {
		baseDialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		dialSocksProxy, _ := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, baseDialer)
		contextDialer, _ := dialSocksProxy.(proxy.ContextDialer)
		dialContext := contextDialer.DialContext
		tr.Proxy = http.ProxyFromEnvironment
		tr.DialContext = dialContext
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   120 * time.Second,
	}

	return client
}

func (fetcher *nineOneFetcher) fetchPage(url string, useProxy bool) (body []byte, err error) {
	var resp *http.Response
	var reader io.ReadCloser

	req, err := http.NewRequest("GET", url, nil)

	if fetcher.cookies != nil {
		for _, c := range fetcher.cookies {
			cookie := c
			req.AddCookie(cookie)
		}
	}

	req.Header.Set("User-Agent", fetcher.userAgent)
	req.Header.Add("Accept-Encoding", "gzip")

	client := fetcher.newHTTPClient(useProxy)

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		err = errors.New("resp.StatusCode: " + strconv.Itoa(resp.StatusCode))
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		defer reader.Close()
	default:
		reader = resp.Body
	}

	if body, err = ioutil.ReadAll(reader); err != nil {
		return nil, err
	}

	return body, nil
}

func (fetcher *nineOneFetcher) fetchVideoList(count int, useProxy bool) (string, error) {
	confmgr := fetcher.sniffer.confmgr

	if _, err := os.Stat(confmgr.config.cookieFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	_, err := fetcher.parseCookies(confmgr.config.cookieFile)
	if err != nil {
		return "", err
	}

	now := time.Now()
	dir := confmgr.config.videoListBaseDir + "/" + now.Format("2006-01-02")
	if _, err := os.Open(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0644)
	}

	var concurrentRtnCount int
	doneChannel := make(chan struct{})
	indexChannel := make(chan int)
	observerChannel := make(chan int)

	var failCount int
	var successCount int

	var failIndexList []int

	fetchRoutine := func(useProxy bool) {
		for {
			index, ok := <-indexChannel
			if !ok {
				doneChannel <- struct{}{}
				break
			}

			src := fmt.Sprintf(confmgr.config.listPageURLBase+"%d", index+1)

			if info, err := fetcher.fetchPage(src, useProxy); err != nil {
				failCount += 1
				failIndexList = append(failIndexList, index)
			} else {
				successCount += 1
				observerChannel <- index
				htmlFile := fmt.Sprintf(dir+"/%04d.html", index+1)
				err = ioutil.WriteFile(htmlFile, info, 0644)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	if count <= 200 {
		concurrentRtnCount = 1
	} else {
		concurrentRtnCount = 8
	}

	for i := 0; i < concurrentRtnCount; i++ {
		go fetchRoutine(useProxy)
	}

	obverserRoutine := func() {
		for {
			if successCount+failCount == count {
				break
			}
			index := <-observerChannel
			fmt.Printf("\rLatest Done Index - %4d, Total - %4d, Success - %4d, Fail - %4d", index, count, successCount, failCount)
		}
	}

	go obverserRoutine()

	for i := 0; i < count; i++ {
		indexChannel <- i
	}

	/* Retry failed items if any */
	if len(failIndexList) > 0 {
		for _, index := range failIndexList {
			indexChannel <- index
		}
	}

	close(indexChannel)

	for i := 0; i < concurrentRtnCount; i++ {
		<-doneChannel
	}

	fmt.Printf("\n")

	return dir, nil
}

func (fetcher *nineOneFetcher) fetchThumbnails(script bool) {
	confmgr := fetcher.sniffer.confmgr

	thumbnailDir := confmgr.config.thumbnailBaseDir
	f, err := os.Open(thumbnailDir)
	if err != nil {
		log.Fatal(err)
	}

	thumbnailsInfo, err := f.Readdir(0)
	if err != nil {
		log.Fatal(err)
	}

	thumbnailsMap := make(map[string]bool)

	for _, info := range thumbnailsInfo {
		if !info.IsDir() {
			//fmt.Println(info.Name())
			thumbnailsMap[info.Name()] = true
		}
	}

	var newThumbnailsCount int

	httpHeadersFile, err := os.Open(confmgr.config.workDir + "/configs/thumbnail_http_headers.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer httpHeadersFile.Close()
	headers := make(map[string]string)
	scanner := bufio.NewScanner(httpHeadersFile)
	for scanner.Scan() {
		line := scanner.Text()
		tuple := strings.Split(line, ":")
		headers[tuple[0]] = tuple[1]
	}

	var thumbnail_http_headers string
	for k, v := range headers {
		thumbnail_http_headers += ` --header='` + k + `: ` + v + `' `
	}

	//fmt.Println(thumbnail_http_headers)
	os.Remove("./thumbnails_dl.sh")

	thumbnailf, err := os.OpenFile("thumbnails_dl.sh", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}

	thumbnailf.WriteString("#!/bin/bash\n")

	vds := fetcher.sniffer.vds
	vds.iterate(func(item *VideoItem) bool {
		_, ok := thumbnailsMap[item.ThumbnailName]
		if !ok {
			thumbnailf.WriteString("wget -O data/images/new/" + item.ThumbnailName +
				" --timeout 120 " + thumbnail_http_headers + " " + item.ThumbnailURL + "\n")
			newThumbnailsCount++
		}
		return true
	})

	thumbnailf.WriteString("curr_date=`date +'%y-%m-%dT%H:%M:%S.%N'`\n")
	thumbnailf.WriteString("tar zcvf /var/www/html/data/images/archive/images.${curr_date}.tar.gz data/images/new\n")
	thumbnailf.WriteString("mv -f data/images/new/*.jpg data/images/base/\n")

	fmt.Printf("Existing thumbnails count - %d\n", len(thumbnailsMap))
	fmt.Printf("Newly got thumbnails count - %d\n", newThumbnailsCount)

	thumbnailf.Close()

	if !script {
		if newThumbnailsCount > 0 {
			cmd := exec.Command("./thumbnails_dl.sh")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println(err)
			}
		}
	}

	//os.Remove("./thumbnails_dl.sh")
}

func (fetcher *nineOneFetcher) fetchVideoPartsDescriptor(url string, saveToDb bool, useProxy bool) error {
	confmgr := fetcher.sniffer.confmgr

	if len(url) == 0 {
		return fmt.Errorf("url shouldn't be empty")
	}

	content, err := fetcher.fetchPage(url, useProxy)
	if err != nil {
		return err
	}

	if strings.Contains(string(content), "Sorry, the page can not found!") {
		return fmt.Errorf("This video may have been removed, now stop!")
	}

	if strings.Contains(string(content), "Sorry") {
		return fmt.Errorf("Up limit reached, now stop!")
	}

	sniffer := *fetcher.sniffer
	parser := sniffer.parser

	info, err := parser.extract(string(content))
	if err != nil {
		return err
	}

	name, src := decode(info)

	if saveToDb {
		persister := fetcher.sniffer.persister
		thumbnail_id, _ := strconv.Atoi(name[:(len(name) - len(".m3u8"))])
		persister.updateVideoDescriptorURL(src, thumbnail_id)
	}

	isExist := func(filename string) (bool, error) {
		_, err := os.Open(filename)
		return !os.IsNotExist(err), err
	}

	exist, err := isExist(confmgr.config.videoPartsDescTodoDir + "/" + name)
	if exist {
		fmt.Printf("video descriptor - %s has already been in the repository, skip now\n", name)
		return err
	}

	exist, err = isExist(confmgr.config.videoPartsDescDoneDir + "/" + name)
	if exist {
		fmt.Printf("video descriptor - %s has already been in the repository, skip now\n", name)
		return err
	}

	filename := confmgr.config.videoPartsDescTodoDir + "/" + name

	// fmt.Printf("src - %s, filename - %s, useProxy - %v\n", *src, filename, useProxy)

	if err = fetcher.wget(src, filename, useProxy); err != nil {
		fmt.Printf("Failed to fetch video parts descriptor: %v\n", err)
	}

	return err
}

func (fetcher *nineOneFetcher) fetchVideoPartsByNameWithWorkers(filename string,
	videoPartsBaseName string, useProxy bool) {

	sniffer := *fetcher.sniffer
	parser := sniffer.parser
	confmgr := sniffer.confmgr

	finalFileName, filePartsCountInteger := parser.parseVideoDescriptor(filename,
		videoPartsBaseName)

	//fmt.Printf("finalFileName - %s, filePartsCountInteger - %02d\n", finalFileName, filePartsCountInteger)

	var howmanyWorkers int
	if filePartsCountInteger < 10 {
		howmanyWorkers = filePartsCountInteger
	} else if filePartsCountInteger >= 10 && filePartsCountInteger < 50 {
		howmanyWorkers = 10
	} else {
		howmanyWorkers = 15
	}

	func(jobCount int, workerCount int) {
		taskURLChannel := make(chan string, jobCount)
		taskResultChannel := make(chan string, jobCount)

		for i := 0; i < workerCount; i++ {
			go func(workerID int) {
				for {
					videoPartURL, ok := <-taskURLChannel
					if !ok {
						break
					}

					videoPartName := videoPartURL[strings.LastIndex(videoPartURL, "/")+1:]

					dirName := confmgr.config.videoPartsDir + "/" + finalFileName
					_, err := os.Open(dirName)
					if os.IsNotExist(err) {
						os.Mkdir(dirName, 0755)
					}

					name := confmgr.config.videoPartsDir + "/" + finalFileName + "/" + videoPartName

					if err = fetcher.wget(videoPartURL, name, useProxy); err != nil {
						fmt.Println(err)
						taskResultChannel <- fmt.Sprintf("Worker #%02d failed to download video part - %s", workerID, videoPartName)
					} else {
						taskResultChannel <- fmt.Sprintf("Worker #%02d done downloading video part - %s", workerID, videoPartName)
					}
				}
			}(i)
		}

		for j := 0; j < jobCount; j++ {
			taskURLChannel <- fmt.Sprintf(confmgr.config.videoPartsURLBase+"/%s/%s%d.ts", finalFileName, finalFileName, j)
		}

		for n := 0; n < jobCount; n++ {
			<-taskResultChannel
			fmt.Printf("\r%02d of %02d Done", n+1, jobCount)
		}
		fmt.Printf("\n")
	}(filePartsCountInteger, howmanyWorkers)

	/* Merge all the downloaded video parts into one and do transcoding */
	os.Remove(confmgr.config.videoMergedDir + finalFileName + ".ts")
	mergedFile, _ := os.OpenFile(confmgr.config.videoMergedDir+"/"+finalFileName+".ts", os.O_CREATE|os.O_WRONLY, 0644)

	/* TODO: Should resolve the case when some of the video parts are missing */
	for i := 0; i < filePartsCountInteger; i++ {
		filePart := fmt.Sprintf("%s/%s/%s%d.ts", confmgr.config.videoPartsDir, finalFileName, finalFileName, i)
		f, err := os.Open(filePart)
		if err != nil {
			fmt.Println(err)
			return
		}

		buffer, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Println(err)
			f.Close()
			return
		}

		f.Close()

		if _, err = mergedFile.Write(buffer); err != nil {
			fmt.Println(err)
			return
		}
	}

	mergedFile.Close()

	os.Rename(confmgr.config.videoPartsDescTodoDir+"/"+finalFileName+".m3u8", confmgr.config.videoPartsDescDoneDir+"/"+finalFileName+".m3u8")

	if fetcher.sniffer.Transcode {
		var cmd *exec.Cmd
		cmd = exec.Command("ffmpeg", "-i", confmgr.config.videoMergedDir+"/"+finalFileName+".ts", "-c:v",
			"h264_qsv", "-c:a", "aac", "-strict", "-2", confmgr.config.videoMergedDir+"/"+finalFileName+".mp4")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}

		kill := exec.Command("taskkill", "/T", "/F", "/IM", "ffmpeg.exe")
		kill.Env = []string{"PATH=\"C:\\Program Files (x86)\\FormatFactory\""}
		kill.Run()

		finalFileNameWithPath := confmgr.config.videoMergedDir + "/" + finalFileName + ".ts"

		if err := os.Remove(confmgr.config.videoMergedDir + "/" + finalFileName + ".ts"); err != nil {
			fmt.Println(err)

			cmd = exec.Command("cmd.exe", "/C", "del", finalFileNameWithPath)
			cmd.Run()

		}

		if err := os.RemoveAll(confmgr.config.videoPartsDir + "/" + finalFileName); err != nil {
			fmt.Println(err)
		}
	}
}

func (fetcher *nineOneFetcher) fetchVideoPartsAndMerge(useProxy bool) error {
	confmgr := fetcher.sniffer.confmgr

	f, err := os.Open(confmgr.config.videoPartsDescTodoDir)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo, _ := f.Readdir(0)
	for _, info := range fileInfo {
		if !info.IsDir() {
			descriptorName := info.Name()
			if strings.Contains(descriptorName, ".mp4") {
				/* Legacy video files do not have descriptor file */
				os.Rename(confmgr.config.videoPartsDescTodoDir+"/"+info.Name(), confmgr.config.videoMergedDir+"/"+info.Name())
				continue
			}

			baseName := descriptorName[:len(descriptorName)-len(".m3u8")]
			fmt.Printf("downloading file - %s\n", info.Name())

			fetcher.fetchVideoPartsByNameWithWorkers(confmgr.config.videoPartsDescTodoDir+"/"+descriptorName, baseName, useProxy)
			os.Rename(confmgr.config.videoPartsDescTodoDir+"/"+info.Name(), confmgr.config.videoPartsDescDoneDir+"/"+info.Name())
		}
	}

	return nil
}

func (fetcher *nineOneFetcher) queryHttpResourceLength(url string) (int, string, error) {
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    60 * time.Second,
		DisableCompression: true,
		TLSClientConfig:    cfg,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   120 * time.Second,
	}

	resp, err := client.Head(url)
	if err != nil {
		return -1, "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		err = fmt.Errorf("http status code - %s\n", resp.StatusCode)
		return -1, "", err
	}

	length, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	lastModifiedTime := resp.Header.Get("Last-Modified")
	return length, lastModifiedTime, nil
}
