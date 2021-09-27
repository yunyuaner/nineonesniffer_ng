package nineonesniffer

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type nineOneParser struct {
	sniffer *NineOneSniffer
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
		var videoDetailedPageURL, thumbnailURL string
		var title string

		attrVal, err := parser.findAttrValueOfElementNode(n, "href")
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

			divElem, err := parser.findFirstChildOfElementNode(n, "div")
			if err != nil {
				return
			}

			// Parse viewkey
			pos := strings.Index(videoDetailedPageURL, "viewkey=")
			viewkey := videoDetailedPageURL[pos+len("viewkey="):]

			// Parse img source url
			imgElem, err := parser.findFirstChildOfElementNode(divElem, "img")

			// 'img' tag has attribute class="img-responsive"
			match, _ := parser.isElementNodeHasAttrs(imgElem, []*html.Attribute{
				{
					Key: "class",
					Val: "img-responsive",
				},
			}, nil)

			if match {
				srcAttrVal, err := parser.findAttrValueOfElementNode(imgElem, "src")
				if err != nil {
					return
				}
				thumbnailURL = srcAttrVal
			}

			// Extract img name and img id
			pos = strings.LastIndex(thumbnailURL, "/") + 1
			thumbnailName := thumbnailURL[pos:]
			pos = strings.Index(thumbnailName, ".")
			thumbnailId, _ := strconv.Atoi(thumbnailName[:pos])

			// Parse video duration
			durationElem, _ := parser.findSiblingOfElementNode(imgElem, "span")
			durationText, _ := parser.getInnerHTMLOfElementNode(durationElem)
			durationText = strings.Replace(durationText, ":", "m", 1)
			durationText = fmt.Sprintf("%ss", durationText)
			duration, _ := time.ParseDuration(durationText)

			// Parse video title
			spanElem, err := parser.findSiblingOfElementNode(divElem, "span")
			if err != nil {
				return
			}

			attrCompare := func(lhs string, rhs string) bool {
				return strings.Contains(lhs, rhs)
			}

			match, err = parser.isElementNodeHasAttrs(spanElem, []*html.Attribute{
				{
					Key: "class",
					Val: "video-title",
				},
			}, &attrCompare)

			attrVal, err = parser.findAttrValueOfElementNode(spanElem, "class")
			if err != nil {
				return
			}

			if match {
				title, _ = parser.getInnerHTMLOfElementNode(spanElem)
			}

			// Parse video author
			spanSibling, _ := parser.findSiblingOfElementNode(n, "span")
			spanAuthor, _ := parser.findSiblingOfElementNode(spanSibling, "span")
			author := spanAuthor.NextSibling.Data
			author = strings.TrimSpace(author)
			author = strings.Trim(author, "\n\r\t")

			item := &VideoItem{
				Title:                title,
				Author:               author,
				VideoDetailedPageURL: videoDetailedPageURL,
				ViewKey:              viewkey,
				Duration:             duration,
				ThumbnailURL:         thumbnailURL,
				ThumbnailName:        thumbnailName,
				ThumbnailId:          thumbnailId,
			}

			*items = append(*items, item)
		}

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

type partial struct {
	thumbnail_id  int
	thumbnail     string
	uploaded_date time.Time
	valid         bool
}

/* Query video uploaded date using http header 'Last-Modified' */
func (parser *nineOneParser) identifyVideoUploadedDate(useProxy bool) {
	/* Query from database and find the video items that has no upload_date yet */
	persister := parser.sniffer.persister
	fetcher := parser.sniffer.fetcher
	obs := parser.sniffer.obs

	partialist, _ := persister.queryVideoItemsWithoutUploadDate()

	fmt.Printf("Got %d items \n", len(partialist))
	if len(partialist) == 0 {
		return
	}

	var maxConcurrentRtn int
	doneChannel := make(chan struct{})
	taskChannel := make(chan *partial)
	observerChannel := make(chan string)

	var failedItems []*partial

	if useProxy {
		maxConcurrentRtn = obs.count()
	} else {
		maxConcurrentRtn = 1
	}

	/* Step 1: launch observer routine */
	go func() {
		for {
			message, ok := <-observerChannel
			if !ok {
				break
			} else {
				log.Println(message)
			}
		}
	}()

	/* Step 2: launch worker routines */
	for i := 0; i < maxConcurrentRtn; i += 1 {
		go func() {
			var proxy_ string

			if useProxy {
				proxy_, _ = obs.yield()
			}

			for {
				item, ok := <-taskChannel
				if !ok {
					break
				}

				lastModified, err := fetcher.queryHttpResourceDate((*item).thumbnail, proxy_)
				if err != nil {
					observerChannel <- fmt.Sprintf("proxy - %s failed to query timestamp from video item %d: %v",
						proxy_, (*item).thumbnail_id, err)
					failedItems = append(failedItems, item)
				} else {
					t, err := time.Parse(time.RFC1123, lastModified)
					if err != nil {
						observerChannel <- fmt.Sprintf("%v", err)
						doneChannel <- struct{}{}
						continue
					}

					(*item).uploaded_date = t
					(*item).valid = true
					observerChannel <- fmt.Sprintf("video item - %d, uploaded_date - %v", (*item).thumbnail_id, (*item).uploaded_date)

					if err := persister.updateVideoUploadDate((*item).uploaded_date, (*item).thumbnail_id); err != nil {
						observerChannel <- fmt.Sprintf("persist video item %d fail: %v", (*item).thumbnail_id, err)
					} else {
						observerChannel <- fmt.Sprintf("persist video item %d done", (*item).thumbnail_id)
					}
				}
				doneChannel <- struct{}{}
			}
		}()
	}

	/* Step 3: dispatch task to worker routines */
	go func() {
		log.Printf("dispatch task to worker routines, task count - %d\n", len(partialist))

		for i := 0; i < len(partialist); i++ {
			taskChannel <- &partialist[i]
		}
	}()

	/* Step 4: wait till all the tasks have been proceed, no matter succeed or not */
	for i := 0; i < len(partialist); i += 1 {
		<-doneChannel
	}

	/* Step 5: retry the failed tasks*/
	var originalFailedItems []*partial
	originalFailedItems = append(originalFailedItems, failedItems...)

	if len(originalFailedItems) > 0 {
		log.Printf("retry the failed tasks, count - %d\n", len(originalFailedItems))

		for _, item := range originalFailedItems {
			taskChannel <- item
		}

		for i := 0; i < len(originalFailedItems); i++ {
			<-doneChannel
		}
	}

	log.Println("close task channel")
	close(taskChannel)
	log.Println("close observer channel")
	close(observerChannel)
}

func (parser *nineOneParser) refreshDataset(dirname string, keep bool) (int, error) {
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
			fullpath := filepath.Join(dirname, file.Name())
			allFiles = append(allFiles, fullpath)
		}
	}

	sort.Strings(allFiles)
	vds := parser.sniffer.vds

	for _, file := range allFiles {
		items, err := parser.parseVideoList(file)
		if err != nil {
			return 0, err
		}
		for _, item := range items {
			if !vds.has(item.ViewKey) {
				vds.append(item.ViewKey, item)
			}
		}
	}

	if !keep {
		os.RemoveAll(dirname)
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

	vds := parser.sniffer.vds
	vds.iterate(func(item *VideoItem) bool {
		line := fmt.Sprintf("wget --tries=10 -O ./tmp/data/images/%s %s\n",
			item.ThumbnailName, item.ThumbnailURL)

		if _, err := script.Write([]byte(line)); err != nil {
			f.Close()
			return false
		}

		return true
	})

	return len(files), nil
}

func (parser *nineOneParser) extract(fileContent string) (string, error) {
	r := regexp.MustCompile(`document.write\(strencode2\(.*\)\);`)
	info := r.FindString(string(fileContent))
	return info, nil
}

func (parser *nineOneParser) findFirstChildOfElementNode(node *html.Node, tagName string) (*html.Node, error) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tagName {
			return c, nil
		}
	}

	return nil, fmt.Errorf("element - %s not found", tagName)
}

func (parser *nineOneParser) findSiblingOfElementNode(node *html.Node, tagName string) (*html.Node, error) {
	for c := node.NextSibling; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tagName {
			return c, nil
		}
	}

	return nil, fmt.Errorf("element - %s not found", tagName)
}

func (parser *nineOneParser) findAttrValueOfElementNode(node *html.Node, attrName string) (string, error) {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val, nil
		}
	}

	return "", fmt.Errorf("attribute - %s not found", attrName)
}

func (parser *nineOneParser) getInnerHTMLOfElementNode(node *html.Node) (string, error) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			return c.Data, nil
		}
	}

	return "", fmt.Errorf("inner HTML not found")
}

func (parser *nineOneParser) isElementNodeHasAttrs(node *html.Node, attrsExpected []*html.Attribute,
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

func (parser *nineOneParser) nodeTypeToString(t html.NodeType) string {
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
