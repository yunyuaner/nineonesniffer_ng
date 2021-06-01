package nineonesniffer

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
	"golang.org/x/net/proxy"
)

const (
	defaultVideoItemCount = 5000
	createTableStmt       = `create table testTable if not exist (
								id integer primary key autoincrement,
								username text, 
								surname text,
								age Integer,
								university text)`
	dir                    = "data/view_video"
	baseurl                = "http://www.91porn.com/v.php?next=watch&page="
	mozillaUserAgentString = "Mozilla/5.0 (platform; rv:17.0) Gecko/20100101 SeaMonkey/2.7.1"
	start                  = 0
	cookieFile             = "./configs/cookies.txt"
	videoPartsDir          = "data/video/video_parts"
	videoMergedDir         = "data/video/video_merged"
	videoPartsDescTodoDir  = "data/video/m3u8/todo"
	videoPartsDescDoneDir  = "data/video/m3u8/done"
	videoPartsURLBase      = "https://cdn.91p07.com//m3u8"
	utilsDir               = "../utils"
)

type nineOneSnifferDaemonCfg struct {
	baseURL                  string `json:"base_url"`
	videoPartsURLBase        string `json:"video_parts_url_base"`
	userAgent                string `json:"user_agent"`
	configBaseDir            string `json:"config_base_dir"`
	cookieFile               string `json:"cookie_file"`
	thumbnailHttpHeadersFile string `json:"thumbnail_http_headers_file"`
	dataBaseDir              string `json:"data_base_dir"`
	videoPartsDir            string `json:"video_parts_dir"`
	videoMergedDir           string `json:"video_merged_dir"`
	videoPartsDescTodoDir    string `json:"video_parts_desc_todo_dir"`
	videoPartsDescDoneDir    string `json:"video_parts_desc_done_dir"`
	videoListBaseDir         string `json:"video_list_base_dir"`
	thumbnailBaseDir         string `json:"thumbnail_base_dir"`
	thumbnailNewDir          string `json:"thumbnail_new_dir"`
	utilsDir                 string `json:"utils_dir"`
	tempDir                  string `json:"temp_dir"`
}

type nineOneSnifferWebGUICfg struct {
}

type nineOneSnifferCliCfg struct {
}

type nineOneSnifferCfg struct {
	snifferDaemon nineOneSnifferDaemonCfg `json:"sniffer_daemon"`
	webgui        nineOneSnifferWebGUICfg `json:"webgui"`
	cli           nineOneSnifferCliCfg    `json:"cli"`
}

type ImageItem struct {
	ImgID     int
	ImgSource string
	ImgName   string
}

type VideoItem struct {
	Title                string
	Author               string
	VideoTime            time.Duration
	UploadTime           time.Time
	VideoDetailedPageURL string
	VideoSource          string
	ViewKey              string
	Thumbnail            ImageItem
}

type VideoDataSet map[string]*VideoItem

/*
 * To be obsolete
 */
func (ds *VideoDataSet) add(key string, item *VideoItem) *VideoDataSet {
	(*ds)[key] = item
	return ds
}

/*
 * To be obsolete
 */
func (ds *VideoDataSet) remove(item *VideoItem) *VideoDataSet {
	delete(*ds, item.ViewKey)
	return ds
}

/*
 * To be obsolete
 */
func (ds *VideoDataSet) has(viewkay string) bool {
	_, ok := (*ds)[viewkay]
	return ok
}

/*
 * To be obsolete
 */
func (ds *VideoDataSet) get(viewkey string) (*VideoItem, bool) {
	item, ok := (*ds)[viewkey]
	return item, ok
}

/*
 * To be obsolete
 */
func (ds *VideoDataSet) iterate(visitor func(item *VideoItem) bool) {
	for _, info := range *ds {
		if ret := visitor(info); !ret {
			return
		}
	}
}

//var vds VideoDataSet

type NineOneSniffer struct {
	fetcher   nineOneFetcher
	parser    nineOneParser
	ds        VideoDataSet
	Transcode bool
	cfg       nineOneSnifferCfg
}

type nineOneFetcher struct {
	sniffer   *NineOneSniffer
	cookies   []*http.Cookie
	userAgent string
}

type nineOneParser struct {
	sniffer *NineOneSniffer
}

func (sniffer *NineOneSniffer) Init() {
	f, err := os.Open("./configs/NineOneSniffer.json")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	info, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	cfg := &((*sniffer).cfg)

	err = json.Unmarshal(info, cfg)
	if err != nil {
		log.Fatal(err)
	}

	sniffer.fetcher.sniffer = sniffer
	sniffer.parser.sniffer = sniffer
	sniffer.fetcher.userAgent = mozillaUserAgentString
	sniffer.ds = make(map[string]*VideoItem)
	sniffer.Transcode = false
}

func (sniffer *NineOneSniffer) DumpCfg() {
	daemonCfg := &((*sniffer).cfg.snifferDaemon)
	fmt.Printf("baseURL - %s\n", daemonCfg.baseURL)
}

func (sniffer *NineOneSniffer) Prefetch(count int, useProxy bool) (string, error) {
	return sniffer.fetcher.fetchVideoList(count, useProxy)
}

func (sniffer *NineOneSniffer) Fetch() {
	sniffer.fetcher.fetchDetailedVideoPages()
}

func (sniffer *NineOneSniffer) FetchThumbnails(script bool) {
	sniffer.fetcher.fetchThumbnails(script)
}

func (sniffer *NineOneSniffer) FetchVideoPartsDscriptor(url string, saveToDb bool) {
	if err := sniffer.fetcher.fetchVideoPartsDescriptor(url, saveToDb); err != nil {
		fmt.Println(err)
	}
}

func (sniffer *NineOneSniffer) FetchVideoPartsAndMerge() {
	if err := sniffer.fetcher.fetchVideoPartsAndMerge(); err != nil {
		fmt.Println(err)
	}
}

func (sniffer *NineOneSniffer) RefreshDataset(dirname string) {
	sniffer.parser.refreshDataset(dirname)
	fmt.Printf("Got %d items\n", sniffer.datasetSize())
}

func (sniffer *NineOneSniffer) IdentifyVideoUploadedDate() {
	//sniffer.parser.identifyVideoUploadedDate()
	sniffer.parser.identifyVideoUploadedDate2()
}

func (sniffer *NineOneSniffer) Persist() {
	sniffer.parser.datasetPersist()
}

func (sniffer *NineOneSniffer) Sync() {
	sniffer.parser.datasetSync()
}

func (sniffer *NineOneSniffer) Load() {
	sniffer.parser.datasetLoad()
}

/* Fetch the most recent 100 videos */
func (sniffer *NineOneSniffer) WhatIsNew() {
	db, _ := sql.Open("sqlite3", "nineone.db")

	defer db.Close()

	rows, err := db.Query("SELECT url, thumbnail_id FROM VideoListTable ORDER by thumbnail_id DESC LIMIT 100")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var detailedVideoPageURL string
		var thumbnail_id int

		err = rows.Scan(&detailedVideoPageURL, &thumbnail_id)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%d - %s\n", thumbnail_id, detailedVideoPageURL)

		if _, err = os.Open("./data/video/m3u8/todo" + strconv.Itoa(thumbnail_id) + ".m3u8"); !os.IsNotExist(err) {
			fmt.Printf("skip\n")
			continue
		}

		if _, err = os.Open("./data/video/m3u8/done" + strconv.Itoa(thumbnail_id) + ".m3u8"); !os.IsNotExist(err) {
			fmt.Printf("skip\n")
			continue
		}

		err = sniffer.fetcher.fetchVideoPartsDescriptor(detailedVideoPageURL, false)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
}

func (sniffer *NineOneSniffer) ParseVideoList(filename string) {
	sniffer.parser.parseVideoList(filename)
}

func (sniffer *NineOneSniffer) datasetAppend(key string, item *VideoItem) *VideoItem {
	dataset := sniffer.ds
	dataset[key] = item
	return item
}

func (sniffer *NineOneSniffer) datasetRemove(item *VideoItem) *VideoItem {
	dataset := sniffer.ds
	delete(dataset, item.ViewKey)
	return item
}

func (sniffer *NineOneSniffer) datasetHas(key string) bool {
	dataset := sniffer.ds
	_, ok := dataset[key]
	return ok
}

func (sniffer *NineOneSniffer) datasetGet(key string) (*VideoItem, bool) {
	dataset := sniffer.ds
	item, ok := dataset[key]
	return item, ok
}

func (sniffer *NineOneSniffer) datasetIterate(visitor func(item *VideoItem) bool) {
	dataset := sniffer.ds
	for _, info := range dataset {
		if ret := visitor(info); !ret {
			return
		}
	}
}

func (sniffer *NineOneSniffer) datasetSize() int {
	return len(sniffer.ds)
}

func findFirstChildOfElementNode(node *html.Node, tagName string) (*html.Node, error) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tagName {
			return c, nil
		}
	}

	return nil, fmt.Errorf("element - %s not found", tagName)
}

func findSiblingOfElementNode(node *html.Node, tagName string) (*html.Node, error) {
	for c := node.NextSibling; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tagName {
			return c, nil
		}
	}

	return nil, fmt.Errorf("element - %s not found", tagName)
}

func findAttrValueOfElementNode(node *html.Node, attrName string) (string, error) {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val, nil
		}
	}

	return "", fmt.Errorf("attribute - %s not found", attrName)
}

func getInnerHTMLOfElementNode(node *html.Node) (string, error) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			return c.Data, nil
		}
	}

	return "", fmt.Errorf("inner HTML not found")
}

func isElementNodeHasAttrs(node *html.Node, attrsExpected []*html.Attribute,
	attrCompare *func(lhs string, rhs string) bool) (bool, error) {
	results := make(map[string]bool)
	var defaultAttrCompare func(lhs string, rhs string) bool

	if attrCompare == nil {
		defaultAttrCompare = func(lhs string, rhs string) bool {
			return lhs == rhs
		}
		attrCompare = &defaultAttrCompare
	}

	for _, attr := range node.Attr {
		for _, expected := range attrsExpected {
			if attr.Key == expected.Key && (*attrCompare)(attr.Val, expected.Val) {
				results[attr.Key] = true
			}
		}

	}

	for _, val := range results {
		if !val {
			return false, fmt.Errorf("can't find the expected attribute")
		}
	}

	return true, nil
}

func nodeTypeToString(t html.NodeType) string {
	nodeTypeStr := "Invalid Node"

	switch t {
	case html.ErrorNode:
		nodeTypeStr = "Error Node"
	case html.TextNode:
		nodeTypeStr = "Text Node"
	case html.DocumentNode:
		nodeTypeStr = "Documemt Node"
	case html.ElementNode:
		nodeTypeStr = "Element Node"
	case html.CommentNode:
		nodeTypeStr = "Comment Node"
	case html.DoctypeNode:
		nodeTypeStr = "Doctype Node"
	case html.RawNode:
		nodeTypeStr = "Raw Node"
	}

	return nodeTypeStr
}

func (parser *nineOneParser) htmlDOMTraverse(node *html.Node,
	visitor func(node *html.Node, data interface{}), data interface{}) {
	visitor(node, data)
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		parser.htmlDOMTraverse(c, visitor, data)
	}
}

func (parser *nineOneParser) parseVideoList(fileName string) (items []*VideoItem, err error) {
	info, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(info)))
	if err != nil {
		return nil, err
	}

	/**
	 * should use pointer to slice *[]*VideoItem
	 * https://stackoverflow.com/questions/39993688/are-golang-slices-passed-by-value
	 */
	parser.htmlDOMTraverse(doc, parser.videoListVisitor, &items)
	return items, nil
}

func (parser *nineOneParser) videoListVisitor(n *html.Node, data interface{}) {
	items := data.(*[]*VideoItem)

	/**
			<div class="well well-sm videos-text-align">

	          <a href="http://www.91porn.com/view_video.php?viewkey=b7af4fb81bf14a65ef4e&page=1&viewtype=basic&category=mr">
	            <div class="thumb-overlay"   id="playvthumb_433614" >
	              <img class="img-responsive" src="https://i.p04.space/thumb/433614.jpg" />
	               <div class="hd-text-icon">HD</div>
	              <span class="duration">03:05</span>
	            </div>
	            <span class="video-title title-truncate m-t-5">酒店跟女同事xx</span>
	          </a>

	 		  <link rel="stylesheet" href="/css/voting.css" />



	          <span class="info">添加时间:</span>  36 分钟  前 <br />
	          <span class="info">作者:</span> lyzwwwwww<br/>
	          <span class="info">查看:</span> 346&nbsp;
	          <span class="info">收藏:</span> 3
			  <span class="info">留言:</span> 0&nbsp;<br>
	          <span class="info">积分:</span> 0&nbsp; &nbsp; &nbsp; &nbsp;<img src=images/like.png height=10>0&nbsp; <img src=images/dislike.png height=10> 0

	        </div>
	*/
	if n.Type == html.ElementNode && n.Data == "a" {
		var videoDetailedPageURL, imgSource string
		var title string

		attrVal, err := findAttrValueOfElementNode(n, "href")
		if err != nil {
			return
		}

		if strings.Contains(attrVal, "viewkey") && strings.Contains(attrVal, "viewtype") {
			// Parse video source url
			if where := strings.Index(attrVal, "page"); where >= 0 {
				videoDetailedPageURL = attrVal[:where-1]
			} else {
				videoDetailedPageURL = attrVal
			}

			divElem, err := findFirstChildOfElementNode(n, "div")
			if err != nil {
				return
			}

			// Parse viewkey
			pos := strings.Index(videoDetailedPageURL, "viewkey=")
			viewkey := videoDetailedPageURL[pos+len("viewkey="):]

			// Parse img source url
			imgElem, err := findFirstChildOfElementNode(divElem, "img")

			// 'img' tag has attribute class="img-responsive"
			match, _ := isElementNodeHasAttrs(imgElem, []*html.Attribute{
				&html.Attribute{
					Key: "class",
					Val: "img-responsive",
				},
			}, nil)

			if match {
				srcAttrVal, err := findAttrValueOfElementNode(imgElem, "src")
				if err != nil {
					return
				}
				imgSource = srcAttrVal
			}

			// Extract img name and img id
			pos = strings.LastIndex(imgSource, "/") + 1
			imgName := imgSource[pos:]
			pos = strings.Index(imgName, ".")
			imgIDStr := imgName[:pos]
			imgID, _ := strconv.Atoi(imgIDStr)

			// Parse video duration
			durationElem, _ := findSiblingOfElementNode(imgElem, "span")
			durationText, _ := getInnerHTMLOfElementNode(durationElem)
			durationText = strings.Replace(durationText, ":", "m", 1)
			durationText = fmt.Sprintf("%ss", durationText)
			duration, _ := time.ParseDuration(durationText)

			// Parse video title
			spanElem, err := findSiblingOfElementNode(divElem, "span")
			if err != nil {
				return
			}

			attrCompare := func(lhs string, rhs string) bool {
				return strings.Contains(lhs, rhs)
			}

			match, err = isElementNodeHasAttrs(spanElem, []*html.Attribute{
				&html.Attribute{
					Key: "class",
					Val: "video-title"},
			}, &attrCompare)

			attrVal, err = findAttrValueOfElementNode(spanElem, "class")
			if err != nil {
				return
			}

			if match {
				title, _ = getInnerHTMLOfElementNode(spanElem)
			}

			// Parse video author
			spanSibling, _ := findSiblingOfElementNode(n, "span")
			spanAuthor, _ := findSiblingOfElementNode(spanSibling, "span")
			author := spanAuthor.NextSibling.Data
			author = strings.TrimSpace(author)
			author = strings.Trim(author, "\n\r\t")

			item := &VideoItem{
				Title:                title,
				Author:               author,
				VideoDetailedPageURL: videoDetailedPageURL,
				ViewKey:              viewkey,
				VideoTime:            duration,
				Thumbnail:            ImageItem{ImgSource: imgSource, ImgName: imgName, ImgID: imgID},
			}

			//fmt.Println(title)
			//fmt.Println(author)
			//fmt.Println(imgSource)
			//fmt.Println(imgName)
			//fmt.Println(imgID)
			//fmt.Println(viewkey)
			//fmt.Println(duration)
			//fmt.Printf("\n")

			*items = append(*items, item)
			//fmt.Printf("visitor, items - %d\n", len(items))
		}

	}
}

func (parser *nineOneParser) detailedVideoItemVisitor(n *html.Node, data interface{}) {
	item := data.(*VideoItem)
	if n.Type == html.ElementNode {

		if n.Data == "source" {
			// Parse video source
			videoSrc, err := findAttrValueOfElementNode(n, "src")
			if err != nil {
				return
			}

			item.VideoSource = videoSrc
		} else if n.Data == "div" {
			match, _ := isElementNodeHasAttrs(n, []*html.Attribute{
				&html.Attribute{
					Key: "id",
					Val: "useraction",
				},
			}, nil)

			if match {
				// Parse video duration
				spanNode, err := findFirstChildOfElementNode(n, "span")
				if err != nil {
					return
				}

				spanNode, err = findFirstChildOfElementNode(spanNode, "span")
				match, err := isElementNodeHasAttrs(spanNode, []*html.Attribute{
					&html.Attribute{
						Key: "class",
						Val: "video-info-span",
					},
				}, nil)

				if match {
					videoDuration, _ := getInnerHTMLOfElementNode(spanNode)
					strings.Replace(videoDuration, ":", "m", 1)
					videoDuration += "s"
					item.VideoTime, _ = time.ParseDuration(videoDuration)
				}
			} else {
				match, _ := isElementNodeHasAttrs(n, []*html.Attribute{
					&html.Attribute{
						Key: "class",
						Val: "videodetails-content",
					},
				}, nil)

				if match {
					// Parse video upload time
					firstChildNode, _ := findFirstChildOfElementNode(n, "div")
					firstSiblingNode, _ := findSiblingOfElementNode(firstChildNode, "div")
					spanNode, _ := findFirstChildOfElementNode(firstSiblingNode, "span")
					spanNode, _ = findSiblingOfElementNode(spanNode, "span")
					uploadTime, _ := getInnerHTMLOfElementNode(spanNode)
					const layout = "2016-12-12"
					item.UploadTime, _ = time.Parse(layout, uploadTime)

					// Parse video author
					secondSiblingNode, _ := findSiblingOfElementNode(firstSiblingNode, "div")
					spanNode, _ = findFirstChildOfElementNode(secondSiblingNode, "span")
					spanNode, _ = findSiblingOfElementNode(spanNode, "span")

					spanNode, _ = findFirstChildOfElementNode(spanNode, "span")
					match, _ = isElementNodeHasAttrs(spanNode, []*html.Attribute{
						&html.Attribute{
							Key: "class",
							Val: "title",
						},
					}, nil)

					if match {
						videoAuthor, _ := getInnerHTMLOfElementNode(spanNode)
						item.Author = videoAuthor
					}
				}
			}
		}

		// Video Source, Duration, Upload Time, Author
		fmt.Println(item.VideoSource)
		fmt.Println(item.VideoTime)
		fmt.Println(item.UploadTime)
		fmt.Println(item.Author)
		fmt.Printf("\n")
	}
}

func (parser *nineOneParser) parseDetailedVideoItem(fileName string, viewkey string) {
	info, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := html.Parse(strings.NewReader(string(info)))
	if err != nil {
		log.Fatal(err)
	}

	item, ok := parser.sniffer.datasetGet(viewkey)
	if ok {
		parser.htmlDOMTraverse(doc, parser.detailedVideoItemVisitor, item)
	}
}

func (parser *nineOneParser) parseVideoDescriptor(filename string, videoPartsBaseName string) (string, int) {

	file, err := os.Open(filename)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	defer file.Close()

	fileContent, _ := ioutil.ReadAll(file)
	//fileContentStr := string(fileContent)

	r := regexp.MustCompile(`[0-9]*\.ts`)
	videoParts := r.FindAllString(string(fileContent), -1)
	//videoPartsLengthMap := make(map[int]int)

	var videoPartsWithoutSuffix []int
	for _, part := range videoParts {
		val, _ := strconv.Atoi(part[:len(part)-3])
		videoPartsWithoutSuffix = append(videoPartsWithoutSuffix, val)
	}
	sort.Ints(videoPartsWithoutSuffix)

	finalFileName := videoPartsBaseName
	lastVideoPartName := strconv.Itoa(videoPartsWithoutSuffix[len(videoPartsWithoutSuffix)-1])
	n := strings.Index(lastVideoPartName, finalFileName)
	filePartsCount := lastVideoPartName[n+len(finalFileName):]

	filePartsCountInteger, _ := strconv.Atoi(filePartsCount)

	/* video file parts begin with suffix 0, so the total count should be the last video parts suffix plus 1 */
	return finalFileName, (filePartsCountInteger + 1)
}

/* Query video uploaded date using http header 'Last-Modified' */
func (parser *nineOneParser) identifyVideoUploadedDate2() {
	db, err := sql.Open("sqlite3", "nineone.db")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	/* Query from database and find the video items that has no upload_date yet */
	rows, err := db.Query("select thumbnail_id, thumbnail from VideoListTable where upload_date is null")
	if err != nil {
		log.Fatal(err)
	}

	type partial struct {
		thumbnail_id  int
		thumbnail     string
		uploaded_date time.Time
		valid         bool
	}

	var partialist []partial

	for rows.Next() {
		var thumbnailID int
		var thumbnail string

		err = rows.Scan(&thumbnailID, &thumbnail)
		if err != nil {
			log.Print(err)
			continue
		}
		partialist = append(partialist, partial{thumbnail_id: thumbnailID, thumbnail: thumbnail})
	}

	fmt.Printf("\rGot %d items \n", len(partialist))

	fetcher := parser.sniffer.fetcher
	failcount := 0

	for _, item := range partialist {
		/* Query upload_date, use multi-task approach later */
		_, lastModified, err := fetcher.queryHttpResourceLength(item.thumbnail)
		if err != nil {
			fmt.Printf("Failed to query timestamp from video item %d\n", item.thumbnail_id)
			failcount = failcount + 1
			if failcount >= 10 {
				fmt.Printf("Something went wrong, will quit now, you may try it later\n")
				break
			}
			continue
		}
		t, err := time.Parse(time.RFC1123, lastModified)
		item.uploaded_date = t
		item.valid = true
		fmt.Printf("video item - %d, uploaded_date - %v\n", item.thumbnail_id, item.uploaded_date)

		/* Persist upload_date into database */
		tx, _ := db.Begin()
		stmt, _ := tx.Prepare("update VideoListTable set upload_date=?  where thumbnail_id=?")
		_, err = stmt.Exec(item.uploaded_date.Format("2006-01-02 15:04:05"), item.thumbnail_id)
		if err != nil {
			fmt.Println(err)
			tx.Rollback()
			continue
		}

		tx.Commit()

		fmt.Printf("persist video item %d done\n", item.thumbnail_id)
	}
}

/* Retrive video uploaded date using timestamp from corresponding thumbnail file */
func (parser *nineOneParser) identifyVideoUploadedDate() {
	var fileMap map[int]time.Time

	f, err := os.Open("data/images/new")
	if err != nil {
		log.Fatal(err)
	}

	info, err := f.Readdir(0)
	if err != nil {
		log.Fatal(err)
	}

	fileMap = make(map[int]time.Time)

	for _, fileInfo := range info {
		//fmt.Printf("file - %s, date - %s\n", fileInfo.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"))

		videoID, _ := strconv.Atoi(fileInfo.Name()[:len(fileInfo.Name())-4])

		fileMap[videoID] = fileInfo.ModTime()
		os.Rename("./data/images/new/"+fileInfo.Name(), "./data/images/base/"+fileInfo.Name())
	}

	f.Close()

	db, _ := sql.Open("sqlite3", "nineone.db")
	defer db.Close()

	fileMapSize := len(fileMap)
	var counter int

	for k, v := range fileMap {
		//fmt.Printf("videoID - %d, date - %s\n", k, v.Format("2006-01-02 15:04:05"))
		tx, _ := db.Begin()
		stmt, _ := tx.Prepare("update VideoListTable set upload_date=?  where thumbnail_id=?")
		_, err := stmt.Exec(v.Format("2006-01-02 15:04:05"), strconv.Itoa(k))
		if err != nil {
			tx.Rollback()
			continue
		}

		counter++
		fmt.Printf("\r[%6d of %d] updated", counter, fileMapSize)

		tx.Commit()
	}

	fmt.Printf("\nDone\n")
}

func (parser *nineOneParser) refreshDataset(dirname string) (int, error) {
	//const dirname = "data/list/base"
	//sniffer := *parser.sniffer
	//dataset := &sniffer.ds

	f, err := os.Open(dirname)
	if err != nil {
		return 0, err
	}

	defer f.Close()

	files, err := f.Readdir(0)
	if err != nil {
		return 0, err
	}

	var allFiles []string

	for _, file := range files {
		if !file.IsDir() {
			fullpath := dirname + "/" + file.Name()
			allFiles = append(allFiles, fullpath)
		}
	}

	sort.Strings(allFiles)

	for _, file := range allFiles {
		//fmt.Println("process file - ", file)
		items, err := parser.parseVideoList(file)
		//fmt.Printf("items - %d\n", len(items))
		if err != nil {
			return 0, err
		}
		for _, item := range items {
			//fmt.Println(item.Title)
			if !parser.sniffer.datasetHas(item.ViewKey) {
				parser.sniffer.datasetAppend(item.ViewKey, item)
			}
		}
	}

	return len(allFiles), nil
}

func (parser *nineOneParser) scriptGenerate() (int, error) {
	const dirname = "tmp/data/list"

	f, err := os.Open(dirname)
	if err != nil {
		return 0, err
	}

	defer f.Close()

	files, err := f.Readdir(0)
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		if !file.IsDir() {
			fullpath := dirname + "/" + file.Name()
			fmt.Println("process file - ", fullpath)
			parser.parseVideoList(fullpath)
		}
	}

	script, err := os.OpenFile("./fetch_script.sh", os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}

	defer script.Close()

	visitor := func(item *VideoItem) bool {
		line := fmt.Sprintf("wget --tries=10 -O ./tmp/data/images/%s %s\n",
			item.Thumbnail.ImgName, item.Thumbnail.ImgSource)

		if _, err := script.Write([]byte(line)); err != nil {
			f.Close()
			return false
		}

		return true
	}

	//sniffer := *parser.sniffer
	//sniffer.ds.iterate(visitor)
	parser.sniffer.datasetIterate(visitor)

	return len(files), nil
}

func (parser *nineOneParser) datasetPersist() {
	db, err := sql.Open("sqlite3", "nineone.db")
	if err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	var newlyAdded int

	//sniffer := *parser.sniffer
	parser.sniffer.datasetIterate(func(item *VideoItem) bool {
		//fmt.Printf("title - %s, author - %s, duration - %s\n", item.Title, item.Author, item.VideoTime.String())
		err = videoListTableInsert(db, item.ViewKey, item.VideoDetailedPageURL,
			item.Title, item.Thumbnail.ImgSource, item.Thumbnail.ImgID,
			item.Author, item.VideoTime.String())
		if err == nil {
			fmt.Printf("title - %s, author - %s\n", item.Title, item.Author)
			newlyAdded++
		}
		return true
	})

	tx.Commit()

	fmt.Printf("%d new items added\n", newlyAdded)
}

func (parser *nineOneParser) datasetSync() {
	db, err := sql.Open("sqlite3", "nineone.db")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	var synced int

	syncFunc := func(item *VideoItem) bool {
		tx, _ := db.Begin()
		stmt, _ := tx.Prepare("update VideoListTable set author=?, duration=? where thumbnail_id=?")
		_, err := stmt.Exec(item.Author, item.VideoTime.String(), strconv.Itoa(item.Thumbnail.ImgID))
		if err != nil {
			tx.Rollback()
		}
		tx.Commit()
		synced++
		fmt.Printf("\r%6d items synced", synced)
		return true
	}

	parser.sniffer.datasetIterate(syncFunc)

	tx.Commit()
}

func (parser *nineOneParser) datasetLoad() {
	db, err := sql.Open("sqlite3", "nineone.db")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	rows, err := db.Query("select title, viewkey, url, thumbnail, thumbnail_id from VideoListTable")
	if err != nil {
		log.Fatal(err)
	}

	count := 0

	for rows.Next() {
		var item VideoItem
		err = rows.Scan(&item.Title, &item.ViewKey, &item.VideoDetailedPageURL, &item.Thumbnail.ImgSource, &item.Thumbnail.ImgID)
		if err != nil {
			log.Print(err)
			continue
		}
		item.Thumbnail.ImgName = fmt.Sprintf("%d.jpg", item.Thumbnail.ImgID)
		parser.sniffer.datasetAppend(item.ViewKey, &item)

		count++
		fmt.Printf("\r%6d item added", count)
	}

	fmt.Printf("\rGot %d items \n", parser.sniffer.datasetSize())
}

func (parser *nineOneParser) decode(infoStr string) (*string, *string) {
	start := strings.Index(infoStr, "\"") + 1
	end := strings.LastIndex(infoStr, "\"")

	escapedSrc := infoStr[start:end]

	var b bytes.Buffer

	for where := 0; where < len(escapedSrc); where += 3 {
		n := strings.Index(escapedSrc[where:], "%")
		val := escapedSrc[where+n+1 : where+n+3]
		integerCh, _ := strconv.ParseInt(val, 16, 32)
		b.WriteByte(byte(integerCh))
	}

	/**
	 * unescaped may looks like:
	 * - Case 1)
	 * <source src='https://ccn.91p52.com//m3u8/459666/459666.m3u8?st=TM6j903f8X4G4lu2lkxyMQ&e=1619197640' type='application/x-mpegURL'>
	 * - Case 2)
	 * <source src='https://fdc.91p49.com/m3u8/459666/459666.m3u8' type='application/x-mpegURL'>
	 * notice that the former url doesn't have http get parameters!!
	 */
	unescaped := b.String()

	fmt.Println(unescaped)

	start = strings.Index(unescaped, "src='") + len("src='")
	end = strings.Index(unescaped[start:], "'")
	srcWithParams := unescaped[start : start+end]
	questionMarkPos := strings.Index(srcWithParams, "?")
	var name string

	fmt.Println(srcWithParams)

	if questionMarkPos == -1 {
		/* Case 2), in case of no http get parameters */
		slash := strings.LastIndex(srcWithParams, "/")
		name = srcWithParams[slash+1:]
	} else {
		/* Case 1) */
		httpGetSrc := srcWithParams[:questionMarkPos]
		slash := strings.LastIndex(httpGetSrc, "/")
		name = httpGetSrc[slash+1:]
	}

	return &name, &srcWithParams
}

func (parser *nineOneParser) extract(fileContent string) (*string, error) {
	r := regexp.MustCompile(`document.write\(strencode2\(.*\)\);`)
	info := r.FindString(string(fileContent))
	return &info, nil
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

func (fetcher *nineOneFetcher) wget(url string, outputFile string) error {
	var resp *http.Response
	var reader io.ReadCloser

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

	// contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
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

func (fetcher *nineOneFetcher) fetchPage(url string, useProxy bool) (body []byte, err error) {
	var resp *http.Response
	var reader io.ReadCloser

	if fetcher.cookies == nil {
		resp, err = http.Get(url)
		if err != nil {
			log.Printf("Fetching %s failed - %v\n", url, err)
			return nil, err
		}
	} else {
		req, err := http.NewRequest("GET", url, nil)

		for _, c := range fetcher.cookies {
			cookie := c
			req.AddCookie(cookie)
		}

		req.Header.Set("User-Agent", fetcher.userAgent)
		req.Header.Add("Accept-Encoding", "gzip")

		newHTTPClient := func() (*http.Client, error) {
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

			return httpClient, nil
		}

		var client *http.Client

		if useProxy {
			client, err = newHTTPClient()
			if err != nil {
				return nil, err
			}
		} else {
			client = &http.Client{}
		}

		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			err = errors.New("resp.StatusCode: " + strconv.Itoa(resp.StatusCode))
			return nil, err
		}
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
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	_, err := fetcher.parseCookies(cookieFile)
	if err != nil {
		return "", err
	}

	now := time.Now()
	dir := "data/list/" + now.Format("2006-01-02")
	if _, err := os.Open(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0644)
	}

	var failCount int
	var successCount int
	for i := start; i < count; i++ {
		url := fmt.Sprintf(baseurl+"%d", i+1)
		info, err := fetcher.fetchPage(url, useProxy)
		if err != nil {
			failCount += 1
			fmt.Printf("\rTotal - %4d, Success - %4d, Fail - %4d", count, successCount, failCount)
		} else {
			successCount += 1
			fmt.Printf("\rTotal - %4d, Success - %4d, Fail - %4d", count, successCount, failCount)

			htmlFile := fmt.Sprintf(dir+"/%04d.html", i+1)
			err = ioutil.WriteFile(htmlFile, info, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	fmt.Printf("\n")

	return dir, nil
}

func (fetcher *nineOneFetcher) fetchThumbnails(script bool) {
	thumbnailDir := "data/images/base"
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

	httpHeadersFile, err := os.Open("./configs/thumbnail_http_headers.txt")
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

	fetcher.sniffer.datasetIterate(func(item *VideoItem) bool {
		_, ok := thumbnailsMap[item.Thumbnail.ImgName]
		if !ok {
			thumbnailf.WriteString("wget -O data/images/new/" + item.Thumbnail.ImgName +
				" --timeout 120 " + thumbnail_http_headers + " " + item.Thumbnail.ImgSource + "\n")
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

func (fetcher *nineOneFetcher) fetchDetailedVideoPages() {
	f, err := os.Open("video_list_by_viewkey.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	var count int

	for scanner.Scan() {
		line := scanner.Text()
		urlBegin := strings.Index(line, "http")
		urlEnd := strings.LastIndex(line, ",")
		url := line[urlBegin:urlEnd]
		pos := strings.Index(url, "viewkey")
		pos += len("viewkey") + 1
		viewk := url[pos:]

		fmt.Printf("[%03d]---> fetch - %s\n", count, url)

		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		if strings.Contains(string(body), "Sorry") {
			fmt.Printf("Up limit reached, now stop\n")
			break
		}

		fileName := fmt.Sprintf("data/detail/%s.html", viewk)

		fmt.Printf("[%03d]<--- file - %s\n", count, fileName)

		err = ioutil.WriteFile(fileName, body, 0644)
		if err != nil {
			log.Fatal(err)
		}

		if count++; count >= 100 {
			break
		}
	}
}

func (fetcher *nineOneFetcher) fetchVideoPartsDescriptor(url string, saveToDb bool) error {
	if len(url) == 0 {
		return fmt.Errorf("url shouldn't be empty")
	}

	content, err := fetcher.fetchPage(url, false)
	if err != nil {
		return err
	}

	if strings.Contains(string(content), "Sorry") {
		return fmt.Errorf("Up limit reached, now stop")
	}

	sniffer := *fetcher.sniffer
	parser := sniffer.parser

	info, err := parser.extract(string(content))
	if err != nil {
		return err
	}

	persist := func(name, url string) {
		thumbnail_id, _ := strconv.Atoi(name[:(len(name) - len(".m3u8"))])

		db, err := sql.Open("sqlite3", "nineone.db")
		if err != nil {
			log.Fatal(err)
		}

		defer db.Close()

		tx, _ := db.Begin()
		stmt, _ := tx.Prepare("update VideoListTable set descriptor_url=?  where thumbnail_id=?")
		_, err = stmt.Exec(url, thumbnail_id)
		if err != nil {
			fmt.Println(err)
			tx.Rollback()
		}

		tx.Commit()
	}

	name, src := sniffer.parser.decode(*info)

	if saveToDb {
		persist(*name, *src)
	}

	isExist := func(filename string) (bool, error) {
		_, err := os.Open(filename)
		return !os.IsNotExist(err), err
	}

	exist, err := isExist(videoPartsDescTodoDir + "/" + *name)
	if exist {
		fmt.Printf("video descriptor - %s has already been in the repository, skip now\n", *name)
		return err
	}

	exist, err = isExist(videoPartsDescDoneDir + "/" + *name)
	if exist {
		fmt.Printf("video descriptor - %s has already been in the repository, skip now\n", *name)
		return err
	}

	filename := videoPartsDescTodoDir + "/" + *name

	if err = fetcher.wget(*src, filename); err != nil {
		fmt.Printf("Failed to fetch video parts descriptor: %v\n", err)
	}

	return err
}

func (fetcher *nineOneFetcher) fetchVideoPartsByNameWithWorkers(filename string,
	videoPartsBaseName string) {

	sniffer := *fetcher.sniffer
	parser := sniffer.parser

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

					dirName := "./data/video/video_parts/" + finalFileName
					_, err := os.Open(dirName)
					if os.IsNotExist(err) {
						os.Mkdir(dirName, 0755)
					}

					name := "./data/video/video_parts/" + finalFileName + "/" + videoPartName

					if err = fetcher.wget(videoPartURL, name); err != nil {
						fmt.Println(err)
						taskResultChannel <- fmt.Sprintf("Worker #%02d failed to download video part - %s", workerID, videoPartName)
					} else {
						taskResultChannel <- fmt.Sprintf("Worker #%02d done downloading video part - %s", workerID, videoPartName)
					}
				}
			}(i)
		}

		for j := 0; j < jobCount; j++ {
			taskURLChannel <- fmt.Sprintf("https://cdn.91p07.com//m3u8/%s/%s%d.ts", finalFileName, finalFileName, j)
		}

		for n := 0; n < jobCount; n++ {
			<-taskResultChannel
			fmt.Printf("\r%02d of %02d Done", n+1, jobCount)
		}
		fmt.Printf("\n")
	}(filePartsCountInteger, howmanyWorkers)

	/* Merge all the downloaded video parts into one and do transcoding */
	os.Remove("./data/video/video_merged/" + finalFileName + ".ts")
	mergedFile, _ := os.OpenFile("./data/video/video_merged/"+finalFileName+".ts", os.O_CREATE|os.O_WRONLY, 0644)

	/* TODO: Should resolve the case when some of the video parts are missing */
	for i := 0; i < filePartsCountInteger; i++ {
		filePart := fmt.Sprintf("./data/video/video_parts/%s/%s%d.ts", finalFileName, finalFileName, i)
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

	os.Rename("./data/video/m3u8/todo/"+finalFileName+".m3u8", "./data/video/m3u8/done/"+finalFileName+".m3u8")

	if fetcher.sniffer.Transcode {
		var cmd *exec.Cmd
		cmd = exec.Command("ffmpeg", "-i", "./data/video/video_merged/"+finalFileName+".ts", "-c:v",
			"h264_qsv", "-c:a", "aac", "-strict", "-2", "./data/video/video_merged/"+finalFileName+".mp4")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}

		kill := exec.Command("taskkill", "/T", "/F", "/IM", "ffmpeg.exe")
		kill.Env = []string{"PATH=\"C:\\Program Files (x86)\\FormatFactory\""}
		kill.Run()

		finalFileNameWithPath := "./data/video/video_merged/" + finalFileName + ".ts"

		if err := os.Remove("./data/video/video_merged/" + finalFileName + ".ts"); err != nil {
			fmt.Println(err)

			cmd = exec.Command("cmd.exe", "/C", "del", finalFileNameWithPath)
			cmd.Run()

		}

		if err := os.RemoveAll("./data/video/video_parts/" + finalFileName); err != nil {
			fmt.Println(err)
		}
	}
}

/**
 * Obsolete, will delete later!
 */
func (fetcher *nineOneFetcher) fetchVideoPartsByName(filename string, videoPartsBaseName string, reliable bool) error {
	utilsGetScript := utilsDir + "/get.sh"
	utilsCatScript := utilsDir + "/cat.sh"

	file, err := os.Open(filename)
	if os.IsNotExist(err) {
		log.Fatal(err)
	}

	defer file.Close()

	fileContent, _ := ioutil.ReadAll(file)
	fileContentStr := string(fileContent)

	r := regexp.MustCompile(`[0-9]*\.ts`)
	videoParts := r.FindAllString(fileContentStr, -1)
	videoPartsLengthMap := make(map[int]int)

	var videoPartsWithoutSuffix []int
	for _, part := range videoParts {
		val, _ := strconv.Atoi(part[:len(part)-3])
		videoPartsWithoutSuffix = append(videoPartsWithoutSuffix, val)
	}

	sort.Ints(videoPartsWithoutSuffix)
	finalFileName := videoPartsBaseName
	lastVideoPartName := strconv.Itoa(videoPartsWithoutSuffix[len(videoPartsWithoutSuffix)-1])
	n := strings.Index(lastVideoPartName, finalFileName)
	filePartsCount := lastVideoPartName[n+len(finalFileName):]

	filePartsCountInteger, _ := strconv.Atoi(filePartsCount)

	/* @finalFileName and @filePartsCountInteger are required in the next stage */

	/* First, query each video parts file length from server */
	if reliable {
		for i := 0; i < filePartsCountInteger; i++ {
			videoPartsNameWithExt := fmt.Sprintf("%s%d.ts", finalFileName, i)
			urlResource := fmt.Sprintf("%s/%s/%s", videoPartsURLBase, finalFileName, videoPartsNameWithExt)
			len, _, err := fetcher.queryHttpResourceLength(urlResource)
			if err != nil {
				return err
			}
			key, _ := strconv.Atoi(fmt.Sprintf("%s%d", finalFileName, i))
			videoPartsLengthMap[key] = len
		}
	}

	/* Retrive the video parts one by one */
	cmd := exec.Command(utilsGetScript, finalFileName, filePartsCount)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	/* TODO: Verify that all the video file parts are downloaded completely */
	if reliable {
		dirOfVideoParts := "data/video/video_parts/"
		for k, v := range videoPartsLengthMap {
			videoPartName := fmt.Sprintf("%d.ts", k)
			file := fmt.Sprintf("%s/%s", dirOfVideoParts, videoPartName)
			fmt.Printf("check existence of video part - %s\n", videoPartName)

			f, err := os.Open(file)
			if os.IsNotExist(err) {
				/* TODO: Should call download method again */
				fmt.Println("Should call download method again")
			} else if info, _ := f.Stat(); int(info.Size()) < v {
				/* TODO: Should call download method again */
				fmt.Println("Should call download method again")
			}

			f.Close()
		}
	}

	/* Merge all the downloaded video parts into one and do transcoding */
	cmd = exec.Command(utilsCatScript, finalFileName, filePartsCount)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (fetcher *nineOneFetcher) fetchVideoPartsAndMerge() error {
	f, err := os.Open(videoPartsDescTodoDir)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo, _ := f.Readdir(0)
	for _, info := range fileInfo {
		if !info.IsDir() {
			descriptorName := info.Name()
			if strings.Contains(descriptorName, ".mp4") {
				/* Legacy video files do not have descriptor file */
				os.Rename(videoPartsDescTodoDir+"/"+info.Name(), videoMergedDir+"/"+info.Name())
				continue
			}

			baseName := descriptorName[:len(descriptorName)-len(".m3u8")]
			fmt.Printf("analyze and download file - %s\n", info.Name())

			//fetcher.fetchVideoPartsByName(videoPartsDescTodoDir+"/"+descriptorName, baseName, false)
			fetcher.fetchVideoPartsByNameWithWorkers(videoPartsDescTodoDir+"/"+descriptorName, baseName)
			os.Rename(videoPartsDescTodoDir+"/"+info.Name(), videoPartsDescDoneDir+"/"+info.Name())

			//cmd := exec.Command("mv", "-f", videoPartsDescTodoDir+"/"+info.Name(), videoPartsDescDoneDir+"/"+info.Name())
			//if err = cmd.Run(); err != nil {
			//	fmt.Println(err)
			//}
		}
	}

	return nil
}

func videoListTableInsert(db *sql.DB, viewkey string, url string, title string, thumbnail string, thumbnailID int, author string, videoTime string) error {
	/* for sql statement, check https://stackoverflow.com/questions/40157049/sqlite-case-statement-insert-if-not-exists */
	//sql := `insert into VideoListTable(viewkey, url)
	//			select viewkey, url
	//			from (select ? as vk, ? as url) t
	//			where not exists (select 1 from VideoListTable where VideoListTable.viewkey = t.vk)`
	//fmt.Println(sql)
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("insert into VideoListTable (title, viewkey, url, thumbnail, thumbnail_id, date, author, duration) values (?,?,?,?,?,?,?,?)")
	//stmt, _ := tx.Prepare(sql)
	_, err := stmt.Exec(title, viewkey, url, thumbnail, thumbnailID, time.Now().Format("2006-01-02 15:04:05"), author, videoTime)
	if err != nil {
		//log.Print(err)
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
