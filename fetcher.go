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
	"net/url"
	"os"
	"os/exec"
	"regexp"
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

func (fetcher *nineOneFetcher) get(url string, socks5Proxy string) (body []byte, err error) {
	return fetcher.fetchGeneric(url, "GET", nil, socks5Proxy, 30*time.Second, nil, nil)
}

func (fetcher *nineOneFetcher) post(url string, formData map[string]string, socks5Proxy string) (body []byte, err error) {
	return fetcher.fetchGeneric(url, "POST", nil, socks5Proxy, 30*time.Second, nil, nil)
}

func (fetcher *nineOneFetcher) fetchGeneric(
	url_ string,
	method string,
	formData map[string]string,
	socks5Proxy string,
	timeout time.Duration,
	callback func(resp *http.Response, body []byte, data interface{}) error,
	data interface{},
) (body []byte, err error) {

	var resp *http.Response
	var req *http.Request
	var reader io.ReadCloser
	var client *http.Client
	var useHttps, useProxy bool
	var contextDialer proxy.ContextDialer

	if strings.ToLower(method) == "post" && formData != nil && len(formData) > 0 {
		form := url.Values{}
		for k, v := range formData {
			form.Add(k, v)
		}

		req, _ = http.NewRequest(method, url_, strings.NewReader(form.Encode()))
	} else {
		req, _ = http.NewRequest(method, url_, nil)
	}

	if fetcher.cookies != nil {
		for _, c := range fetcher.cookies {
			cookie := c
			req.AddCookie(cookie)
		}
	}

	req.Header.Set("User-Agent", fetcher.userAgent)
	req.Header.Add("Accept-Encoding", "gzip")

	re := regexp.MustCompile(`^https`)
	url_ = strings.ToLower(strings.TrimSpace(url_))
	useHttps = re.MatchString(url_)

	if len(socks5Proxy) > 0 {
		re = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}:[0-9]+$`)
		socks5Proxy = strings.TrimSpace(socks5Proxy)
		useProxy = re.MatchString(socks5Proxy)
	}

	if useHttps {
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

		client = &http.Client{
			Transport: tr,
			Timeout:   120 * time.Second,
		}
	} else {
		tr := &http.Transport{
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		}

		if useProxy {
			baseDialer := &net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
			}

			dialSocksProxy, _ := proxy.SOCKS5("tcp", socks5Proxy, nil, baseDialer)
			contextDialer, _ = dialSocksProxy.(proxy.ContextDialer)
		}

		// tr.Proxy = http.ProxyFromEnvironment
		if contextDialer != nil {
			tr.DialContext = contextDialer.DialContext
		}

		client = &http.Client{
			Transport: tr,
			Timeout:   timeout,
		}
	}

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

	if callback != nil {
		callback(resp, body, data)
	}

	return body, nil
}

func (fetcher *nineOneFetcher) wget(url_ string, outputFile string, useProxy bool) error {
	var proxy_ string
	if useProxy {
		obs := fetcher.sniffer.obs
		proxy_, _ = obs.yield()
	}

	_, err := fetcher.fetchGeneric(
		url_,
		"GET",
		nil,
		proxy_,
		30*time.Second,
		func(resp *http.Response, body []byte, data interface{}) error {
			outputFile_ := data.(string)
			lastModified := resp.Header.Get("Last-Modified")
			t, err := time.Parse(time.RFC1123, lastModified)
			keepFileTimestamp := true
			if err != nil {
				keepFileTimestamp = false
			}

			f, err := os.OpenFile(outputFile_, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return err
			}

			defer f.Close()

			if _, err = f.Write(body); err != nil {
				return nil
			}

			if keepFileTimestamp {
				os.Chtimes(outputFile, t, t)
			}

			return nil
		}, outputFile)

	return err
}

func (fetcher *nineOneFetcher) fetchVideoList(count int, useProxy bool) (string, error) {
	confmgr := fetcher.sniffer.confmgr
	obs := fetcher.sniffer.obs

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

	var concurrentRtnCount, maxConcurrentRtn int
	doneChannel := make(chan struct{})
	indexChannel := make(chan int)
	observerChannel := make(chan int)

	var failCount int
	var successCount int

	var failIndexList []int

	if useProxy {
		maxConcurrentRtn = obs.count()
		concurrentRtnCount = maxConcurrentRtn
	} else {
		/* FIXME: when using multi-threaded method to fetch video list, it's very likely to get banned */
		maxConcurrentRtn = 1
		concurrentRtnCount = maxConcurrentRtn
	}

	/* Step 1: launch observer routine */
	go func() {
		for {
			index, ok := <-observerChannel
			if !ok {
				break
			} else {
				log.Printf("Threads - %2d, Latest Done Index - %4d, Total - %4d, Success - %4d, Fail - %4d\n",
					concurrentRtnCount, index, count, successCount, failCount)
			}
		}
	}()

	/* Step 2: launch worker routines */
	for i := 0; i < concurrentRtnCount; i += 1 {
		go func() {
			proxy := ""
			if useProxy {
				proxy, _ = obs.yield()
				log.Printf("yield proxy - %s\n", proxy)
			}

			for {
				index, ok := <-indexChannel
				if !ok {
					doneChannel <- struct{}{}
					obs.release(proxy)
					break
				}

				src := fmt.Sprintf(confmgr.config.listPageURLBase+"%d", index+1)
				log.Printf("proxy - %s, src - %s\n", proxy, src)

				if info, err := fetcher.get(src, proxy); err != nil {
					failCount += 1
					failIndexList = append(failIndexList, index)

					/* if failed, exit */
					concurrentRtnCount -= 1
					obs.release(proxy)

					log.Printf("proxy - %s fail, exit\n", proxy)
					doneChannel <- struct{}{}
					break
				} else {
					/* TODO: return result may be banned */
					successCount += 1
					observerChannel <- index

					htmlFile := fmt.Sprintf(dir+"/%04d.html", index+1)
					err = ioutil.WriteFile(htmlFile, info, 0644)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}()
	}

	go func(count_ int) {
		/* Step 3: dispatch task to worker routines */
		log.Printf("dispatch task to worker routines, taskk count - %d\n", count_)

		for i := 0; i < count_; i += 1 {
			indexChannel <- i
		}

		/* Step 4 (Optional): retry the failed tasks*/
		log.Println("retry the failed tasks")

		if len(failIndexList) > 0 {
			for _, index := range failIndexList {
				indexChannel <- index
			}
		}

		log.Println("close index channel")
		close(indexChannel)
	}(count)

	/* Step 5: wait till all the worker routines have been finished */
	for i := 0; i < maxConcurrentRtn; i += 1 {
		<-doneChannel
		log.Printf("done channels - [ %2d of %2d ]\n", i+1, maxConcurrentRtn)
	}

	log.Println("close observer channel")
	close(observerChannel)
	// log.Printf("\n")

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

	obs := fetcher.sniffer.obs
	proxy_, _ := obs.yield()

	content, err := fetcher.get(url, proxy_)
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
		err = fmt.Errorf("http status code - %v\n", resp.StatusCode)
		return -1, "", err
	}

	length, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	lastModifiedTime := resp.Header.Get("Last-Modified")
	return length, lastModifiedTime, nil
}
