package nineonesniffer

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type obsProxyItem struct {
	proxy string
	inUse bool
	token string
}

type obscurer struct {
	proxies []*obsProxyItem
	sniffer *NineOneSniffer
}

func (obs *obscurer) yield() (string, string, error) {
	fetcher := obs.sniffer.fetcher

	for _, item := range obs.proxies {
		if !item.inUse {
			item.inUse = true
			cookies, err := fetcher.getCookies(item.proxy)
			if err != nil {
				continue
			}

			if _, ok := cookies["covid"]; ok {
				return item.proxy, cookies["covid"], nil
			} else {
				continue
			}
		}
	}

	return "", "", fmt.Errorf("Not more proxy available")
}

func (obs *obscurer) release(proxy string) {
	for _, item := range obs.proxies {
		if item.proxy == proxy {
			item.inUse = false
		}
	}
}

func (obs *obscurer) count() int {
	return len(obs.proxies)
}

func (obs *obscurer) queryhideme() (proxy []string) {
	fetcher := obs.sniffer.fetcher
	parser := obs.sniffer.parser

	data, err := fetcher.get("https://hidemy.name/en/proxy-list/?type=5", nil, "")
	if err != nil {
		log.Fatal(err.Error())
	}

	// fmt.Println(string(data))

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		log.Fatal(err.Error())
	}

	var f func(n *html.Node)

	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tbody" {
			for q := n.FirstChild; q != nil; q = q.NextSibling {
				if q.Type != html.ElementNode {
					continue
				}

				if q.Data == "tr" {
					r, err := parser.findFirstChildOfElementNode(q, "td")
					if err != nil {
						fmt.Println(err)
						continue
					}

					// fmt.Println(r.Data)

					ip, err := parser.getInnerHTMLOfElementNode(r)
					if err != nil {
						fmt.Println(err)
						continue
					}

					s, err := parser.findSiblingOfElementNode(r, "td")
					port, err := parser.getInnerHTMLOfElementNode(s)
					if err != nil {
						fmt.Println(err)
						continue
					}

					// fmt.Printf("%s:%s\n", ip, port)
					proxy = append(proxy, fmt.Sprintf("%s:%s", ip, port))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return proxy
}

func (obs *obscurer) queryspys() (proxy []string) {
	fetcher := obs.sniffer.fetcher
	// data, err := fetcher.get("https://spys.one/en/socks-proxy-list/", "")
	formData := map[string]string{
		"xpp": "2",
		"xf1": "0",
		"xf2": "0",
		"xf4": "0",
		"xf5": "2",
	}

	data, err := fetcher.post("https://spys.one/en/socks-proxy-list/", formData, nil, "")
	if err != nil {
		log.Fatal(err.Error())
	}

	// fmt.Println(string(data))

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		log.Fatal(err.Error())
	}

	var f0, f1, f2 func(*html.Node)
	var tableRows []*html.Node
	var variableDeclarationScript string

	f0 = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			if n.NextSibling != nil && n.NextSibling.Data == "script" {
				variableDeclarationScript = n.NextSibling.FirstChild.Data
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f0(c)
		}
	}
	f0(doc)

	variables := strings.Split(variableDeclarationScript, ";")

	effectiveVariables := make(map[string]int)
	for _, v := range variables {
		if strings.Contains(v, "^") {
			pos := strings.Index(v, "=")
			name := v[0:pos]
			value := v[pos+1:]
			effectiveVariables[name], _ = strconv.Atoi(value[0:1])
		}
	}

	f1 = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "spy1x") {
					tableRows = append(tableRows, n)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f1(c)
		}
	}
	f1(doc)

	type proxyItem struct {
		proxyIP   string
		proxyPort string
	}

	var proxies_ []proxyItem

	re := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

	for _, colNode := range tableRows {
		f2 = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "td" {
				for _, attr := range n.Attr {
					if attr.Key == "colspan" && strings.Contains(attr.Val, "1") {
						firstChild := n.FirstChild
						if firstChild != nil && firstChild.Type == html.ElementNode && firstChild.Data == "font" {
							if firstChild.FirstChild != nil && firstChild.FirstChild.Type == html.TextNode {
								var ip, port string

								if re.MatchString(firstChild.FirstChild.Data) {
									ip = firstChild.FirstChild.Data
									textNode := firstChild.FirstChild
									jsNode := textNode.NextSibling
									if jsNode != nil && jsNode.Type == html.ElementNode && jsNode.Data == "script" {
										port = jsNode.FirstChild.Data
										proxies_ = append(proxies_, proxyItem{proxyIP: ip, proxyPort: port})
									}
								}

							}
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f2(c)
			}
		}
		f2(colNode)
	}

	re = regexp.MustCompile(`\([\d\w]+\^[\d\w]+\)`)
	for _, p := range proxies_ {
		expression := re.FindAllString(p.proxyPort, -1)
		num := ""
		for _, expr := range expression {
			expr = strings.Trim(expr, "()")
			for k, v := range effectiveVariables {
				if strings.Contains(expr, k) {
					num += strconv.Itoa(v)
					break
				}
			}
		}

		proxy = append(proxy, fmt.Sprintf("%s:%s", p.proxyIP, num))
	}

	return proxy
}

func (obs *obscurer) proxyInvalidate() {
	spys := obs.queryspys()
	hideme := obs.queryhideme()
	spys = append(spys, hideme...)

	fetcher := obs.sniffer.fetcher
	confmgr := obs.sniffer.confmgr

	tryURL := fmt.Sprintf(confmgr.config.listPageURLBase + "1")

	if err := os.Remove("proxies.txt"); err != nil {
		log.Printf(err.Error())
	}

	f, err := os.OpenFile("proxies.txt", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	fmt.Printf("Query proxy item count - %d\n", len(spys))

	for _, proxy := range spys {
		fmt.Printf("Tring %s", proxy)
		if _, err := fetcher.fetchGeneric(tryURL, "GET", nil, nil, proxy, 30*time.Second, nil, nil); err != nil {
			fmt.Printf(" fail\n")
		} else {
			fmt.Printf(" success\n")
			line := fmt.Sprintf("%s\n", proxy)
			f.Write([]byte(line))
		}
	}
}

func (obs *obscurer) proxySetup() error {
	file, err := os.Open("proxies.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		obs.proxies = append(obs.proxies, &obsProxyItem{proxy: scanner.Text(), inUse: false})
	}

	return nil
}
