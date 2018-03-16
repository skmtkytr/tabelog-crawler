package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/gocrawl"
	"github.com/PuerkitoBio/goquery"
)

// リクエスト
type Request struct {
	url   string
	depth int
}

// 結果
type Result struct {
	err error
	url string
}

// チャンネル
type Channels struct {
	req  chan Request
	res  chan Result
	quit chan int
}

// チャンネルを取得。
func NewChannels() *Channels {
	return &Channels{
		req:  make(chan Request, 10),
		res:  make(chan Result, 10),
		quit: make(chan int, 10),
	}
}

// 指定された URL の Web ページを取得し、ページに含まれる URL の一覧を取得。
func Fetch(u string) (urls []string, err error) {
	baseUrl, err := url.Parse(u)
	if err != nil {
		return
	}

	resp, err := http.Get(baseUrl.String())
	if err != nil {
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	urls = make([]string, 0)
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			reqUrl, err := baseUrl.Parse(href)
			if err == nil {
				urls = append(urls, reqUrl.String())
			}
		}
	})

	return
}

// Restaurant Data
type Restaurant struct {
	name     string
	score    float64
	genre    string
	tel      string
	address  string
	holiday  string
	opentime string
	url      string
}

type Ext struct {
	*gocrawl.DefaultExtender
}

func fetchRestaurantData(doc *goquery.Document) Restaurant {
	rest := Restaurant{}
	rest.name = strings.TrimSpace(doc.Find("h2 > a > span").Text())
	rest.score = getScore(doc)
	rest.genre = strings.TrimSpace(doc.Find("table.c-table > tbody > tr > td").Eq(2).Text())
	rest.tel = strings.TrimSpace(doc.Find("table.c-table > tbody > tr > td").Eq(3).Text())
	rest.address = strings.TrimSpace(doc.Find("table.c-table > tbody > tr > td").Eq(5).Text())
	rest.opentime = strings.TrimSpace(doc.Find("table.c-table > tbody > tr > td").Eq(7).Text())
	rest.holiday = strings.TrimSpace(doc.Find("table.c-table > tbody > tr > td").Eq(8).Text())
	return rest
}

func getScore(doc *goquery.Document) float64 {
	txt := doc.Find("span.rdheader-rating__score-val-dtl").Text()
	if txt == "-" {
		return 0
	}
	value, err := strconv.ParseFloat(txt, 64)
	if err != nil {
	}
	return value
}

func (e *Ext) Visit(ctx *gocrawl.URLContext, res *http.Response, doc *goquery.Document) (interface{}, bool) {
	fmt.Printf("Visit: %s\n", ctx.URL())
	fetchRestaurantData(doc)
	/*
		fmt.Printf("Name: %v\n", rest.name)
		fmt.Printf("Score: %v\n", rest.score)
		fmt.Printf("Genre: %v\n", rest.genre)
		fmt.Printf("Tel: %v\n", rest.tel)
		fmt.Printf("Address: %v\n", rest.address)
		fmt.Printf("OpenTime: %v\n", rest.opentime)
		fmt.Printf("Holiday: %v\n", rest.holiday)
	*/
	return nil, true
}

var targetTOP = regexp.MustCompile(`http://tabelog\.com/tokyo$`)
var targetPAGING = regexp.MustCompile(`http://tabelog.com/tokyo/rstLst/[0-9]+$`)
var targetRESTAURANT = regexp.MustCompile(`http://tabelog.com/tokyo/A[0-9]{4}/A[0-9]{6}/[0-9]{8}$`)

func (e *Ext) Filter(ctx *gocrawl.URLContext, isVisited bool) bool {
	currentURL := ctx.NormalizedURL().String()
	if isVisited {
		return false
	}

	if targetPAGING.MatchString(currentURL) || targetRESTAURANT.MatchString(currentURL) {
		return true
	}

	if targetTOP.MatchString(currentURL) {
		return true
	}

	return false
}

func main() {
	ext := &Ext{&gocrawl.DefaultExtender{}}
	// Set custom options
	opts := gocrawl.NewOptions(ext)
	opts.CrawlDelay = 1 * time.Second
	opts.LogFlags = gocrawl.LogError
	opts.SameHostOnly = false
	opts.MaxVisits = 140000

	c := gocrawl.NewCrawlerWithOptions(opts)
	c.Run("https://tabelog.com/tokyo/")
}
