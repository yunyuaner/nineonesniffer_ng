package nineonesniffer

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
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
	videoListDatabase      = "video_list_by_viewkey.txt"
	cookieFile             = "cookies.txt"
)

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
	fetcher nineOneFetcher
	parser  nineOneParser
	ds      VideoDataSet
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
	sniffer.fetcher.sniffer = sniffer
	sniffer.parser.sniffer = sniffer
	sniffer.fetcher.userAgent = mozillaUserAgentString
	sniffer.ds = make(map[string]*VideoItem)
}

func (sniffer *NineOneSniffer) Prefetch() {
	sniffer.fetcher.fetchVideoList()
}

func (sniffer *NineOneSniffer) Fetch() {
	sniffer.fetcher.fetchDetailedVideoPages()
}

func (sniffer *NineOneSniffer) RefreshDataset() {
	sniffer.parser.refreshDataset()
	fmt.Printf("Got %d items\n", len(sniffer.ds))
	sniffer.parser.persistDataset()
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

func findFirstChildOfElementNode(node *html.Node, tagName string) (*html.Node, error) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tagName {
			return c, nil
		}
	}

	return nil, fmt.Errorf("element - %s not found", tagName)
}

func findSiblingOfElementNode(node *html.Node, tagName string) (*html.Node, error) {
	for c := node; c != nil; c = c.NextSibling {
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

	parser.htmlDOMTraverse(doc, parser.videoListVisitor, items)
	return items, nil
}

func (parser *nineOneParser) videoListVisitor(n *html.Node, data interface{}) {
	items := data.([]*VideoItem)

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

			item := &VideoItem{
				Title:                title,
				VideoDetailedPageURL: videoDetailedPageURL,
				ViewKey:              viewkey,
				Thumbnail:            ImageItem{ImgSource: imgSource, ImgName: imgName, ImgID: imgID},
			}

			//fmt.Println(title)
			//fmt.Println(videoSource)
			//fmt.Println(imgSource)
			//fmt.Println(imgName)
			//fmt.Println(imgID)
			//fmt.Println(viewkey)
			//fmt.Printf("\n")

			items = append(items, item)
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

func (parser *nineOneParser) refreshDataset() (int, error) {
	const dirname = "tmp/data/list"
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

	for _, file := range files {
		if !file.IsDir() {
			fullpath := dirname + "/" + file.Name()
			//fmt.Println("process file - ", fullpath)
			items, err := parser.parseVideoList(fullpath)
			if err != nil {
				return 0, err
			}
			for _, item := range items {
				fmt.Println(item.Title)
				parser.sniffer.datasetAppend(item.ViewKey, item)
			}
		}
	}

	return len(files), nil
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

func (parser *nineOneParser) persistDataset() {
	db, _ := sql.Open("sqlite3", "nineone.db")
	defer db.Close()

	//sniffer := *parser.sniffer
	parser.sniffer.datasetIterate(func(item *VideoItem) bool {
		fmt.Printf("title - %s, viewkey - %s\n", item.Title, item.ViewKey)
		videoListTableInsert(db, item.ViewKey, item.VideoDetailedPageURL)
		return true
	})
}

func (parser *nineOneParser) refreshDataset2() {
	const dirname = "tmp/data/detail"
	//files, err := ioutil.ReadDir(dir)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//for _, f := range files {
	//fileName := dir + "/" + f.Name()
	//fmt.Printf("parse - %s\n", fileName)
	//parser.parseDetailedVideoItem(fileName)
	//}

	//fmt.Println(len(dataset))

	//fmt.Println("<head><title>Results</title></head>")
	//fmt.Println("<body>")

	//for _, dataItem := range dataset {
	//	fmt.Println("<p>")
	//	fmt.Printf("<a href='%s' target='blank'>%s</a>\n", dataItem.src, dataItem.name)
	//	fmt.Println("</p>")
	//}

	//fmt.Println("</body>")
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

func (fetcher *nineOneFetcher) fetchPage(url string) (body []byte, err error) {
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

		client := new(http.Client)
		resp, err = client.Do(req)

		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			err = errors.New(url + "resp.StatusCode: " + strconv.Itoa(resp.StatusCode))
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

	body, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (fetcher *nineOneFetcher) fetchVideoList() error {
	var url string

	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		log.Fatal(err)
	}

	_, err := fetcher.parseCookies(cookieFile)
	if err != nil {
		return err
	}

	for i := start; i < 10; i++ {
		url = fmt.Sprintf(baseurl+"%d", i+1)
		fmt.Printf("fetch - %s\n", url)

		info, err := fetcher.fetchPage(url)

		fileName := fmt.Sprintf("tmp/data/list/%04d.html", i+1)

		err = ioutil.WriteFile(fileName, info, 0644)
		if err != nil {
			return err
		}
	}

	return nil
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

func videoListTableInsert(db *sql.DB, viewkey string, url string) {

	defer db.Close()

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("insert into VideoListTable (viewkey, url) values (?,?)")
	_, err := stmt.Exec(viewkey, url)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}
