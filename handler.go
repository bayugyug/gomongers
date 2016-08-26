package main

import (
	"bufio"
	//"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

func b(s string) *bufio.Reader { return bufio.NewReader(strings.NewReader(s)) }

//Store data holder for ios data
type Store struct {
	Preview  string `json:"preview,omitempty"`
	StoreID  string `json:"store_id,omitempty"`
	Category string `json:"category,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
}

//WebRes holder for http request data
type WebRes struct {
	Status int
	Body   string
}

var urlList = []string{
	"https://itunes.apple.com/us/genre/ios-games-adventure/id7002?mt=8",
	/*
	   "https://itunes.apple.com/us/genre/ios-games-action/id7001?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-arcade/id7003?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-board/id7004?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-card/id7005?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-casino/id7006?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-dice/id7007?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-educational/id7008?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-family/id7009?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-music/id7011?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-puzzle/id7012?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-racing/id7013?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-role-playing/id7014?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-simulation/id7015?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-sports/id7016?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-strategy/id7017?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-trivia/id7018?mt=8",
	   "https://itunes.apple.com/us/genre/ios-games-word/id7019?mt=8",
	*/
}

var urlPage chan string
var storeData chan *Store
var webData chan *WebRes

func doIt() {

	//set
	log.Println(".doIt(): Start!")
	memDmp()

	urlPage = make(chan string)
	storeData = make(chan *Store)
	webData = make(chan *WebRes)

	/*
		//get task
		dFlag := make(chan bool)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		go parseIt(dFlag, wg)
	*/

	//get task
	zFlag := make(chan bool)
	zwg := new(sync.WaitGroup)
	zwg.Add(1)
	go storeIt(zFlag, zwg)

	tFlag := make(chan bool)
	twg := new(sync.WaitGroup)

	idx := 0
	//get task
	for _, url := range urlList {
		//var w io.Writer
		log.Println("try: ", url)
		//var pg = 0
		for i := 65; i <= 91; i++ {
			j := i
			if i == 91 {
				j = 42
			}
			//urlPage <- fmt.Sprintf("%s&letter=%c&page=", url, j)
			idx++
			twg.Add(1)
			go processIt(tFlag, twg, fmt.Sprintf("%s&letter=%c&page=", url, j), idx)
			//sig-check
			if !pStillRunning {
				log.Println("Signal detected ...")
				break
			}
			//pause it by batch
			if idx > 1 && idx%3 == 0 {
				log.Println("Batch Max reached ... waiting ...")
				twg.Wait()
			}
		}
	}
	close(urlPage)
	//dont leave your friend behind :-)

	/**
	log.Println("doIt: waiting ... parseIt")
	wg.Wait()
	close(dFlag)
	**/

	log.Println("doIt: waiting ... processIt")
	twg.Wait()
	close(tFlag)

	close(storeData)

	log.Println("doIt: waiting ... storeIt")
	zwg.Wait()
	close(zFlag)

	//stats
	memDmp()
	statsDmp()
	log.Println("doIt: done    ... ")
}

func processIt(doneFlg chan bool, wg *sync.WaitGroup, bfr string, idx int) {

	go func() {
		for {
			select {
			//wait till doneFlag has value ;-)
			case <-doneFlg:
				//done already ;-)
				wg.Done()
				return
			}
		}
	}()

	//tunnel
	surl := strings.Split(bfr, "/")
	category := ""
	if len(surl) > 5 {
		category = strings.ToUpper(strings.Replace(strings.Replace(strings.TrimSpace(surl[5]), "ios-", "", -1), "-", "_", -1))
	}
	page := 0
	for {
		//sig-check
		if !pStillRunning {
			log.Println("Signal detected ...")
			break
		}
		page++
		nurl := fmt.Sprintf("%s%d", bfr, page)
		status, body := getResult(nurl)
		if status != 200 || body == "" {
			log.Println("ERROR: invalid http status", status)
			break
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
		if err != nil {
			log.Println("ERROR: ", err)
			break
		}
		//SUB-TITLE
		found := 0
		doc.Find("#selectedcontent").Find("a").Each(func(i int, n *goquery.Selection) {
			for _, v := range n.Nodes[0].Attr {
				if v.Key == "href" {
					storeid := ""
					xtores := strings.Split(v.Val, "/")
					if len(xtores) > 6 {
						ztores := strings.Split(xtores[6], "?")
						if ztores[0] != "" {
							storeid = strings.TrimSpace(ztores[0])
						}
						found++
						storeData <- &Store{URL: nurl, Preview: strings.TrimSpace(v.Val), Category: category, Name: strings.TrimSpace(n.Text()), StoreID: strings.Replace(storeid, "id", "", -1)}
					}
				}
			}
		})
		if found < 3 {
			break
		}
	} //page loop

	//dont leave your friend behind :-)
	log.Println("processIt: waiting!", idx, " - ", bfr)
	//send signal -> DONE
	doneFlg <- true
}

func getResult(url string) (int, string) {
	//client
	c := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 300 * time.Second,
			}).Dial,
		},
	}
	res, err := c.Get(url)
	if err != nil {
		log.Println("ERROR: getResult:", err)
		return 0, ""
	}
	//get response
	robots, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		log.Println("ERROR: getResult:", err)
		return 0, ""
	}
	//give
	return res.StatusCode, string(robots)
}

func storeIt(doneFlg chan bool, wg *sync.WaitGroup) {

	go func() {
		for {
			select {
			//wait till doneFlag has value ;-)
			case <-doneFlg:
				//done already ;-)
				wg.Done()
				return
			}
		}
	}()

	for {
		row, more := <-storeData
		if !more {
			break
		}
		//jdata, _ := json.Marshal(row)
		pStats.setStats(row.Category)
		//log.Println("Data: ", string(jdata))
		//sig-check
		if !pStillRunning {
			log.Println("Signal detected ...")
			break
		}

	}
	//send signal -> DONE
	doneFlg <- true
}

func parseIt(doneFlg chan bool, wg *sync.WaitGroup) {

	go func() {
		for {
			select {
			//wait till doneFlag has value ;-)
			case <-doneFlg:
				//done already ;-)
				log.Println("parseIt: done   !")
				wg.Done()
				return
			}
		}
	}()

	dTot := 0

	//tunnel
	for bfr := range urlPage {
		dTot++
		surl := strings.Split(bfr, "/")
		category := ""
		if len(surl) > 5 {
			category = strings.ToUpper(strings.Replace(strings.Replace(strings.TrimSpace(surl[5]), "ios-", "", -1), "-", "_", -1))
		}
		//sig-check
		if !pStillRunning {
			log.Println("Signal detected ...")
			break
		}
		page := 0
		for {
			//sig-check
			if !pStillRunning {
				log.Println("Signal detected ...")
				break
			}
			page++
			nurl := fmt.Sprintf("%s%d", bfr, page)
			status, body := getResult(nurl)
			if status != 200 || body == "" {
				log.Println("ERROR: invalid http status", status)
				break
			}
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
			if err != nil {
				log.Println("ERROR: ", err)
				break
			}
			//SUB-TITLE
			found := 0
			doc.Find("#selectedcontent").Find("a").Each(func(i int, n *goquery.Selection) {
				for _, v := range n.Nodes[0].Attr {
					if v.Key == "href" {
						storeid := ""
						xtores := strings.Split(v.Val, "/")
						if len(xtores) > 6 {
							ztores := strings.Split(xtores[6], "?")
							if ztores[0] != "" {
								storeid = strings.TrimSpace(ztores[0])
							}
							found++
							storeData <- &Store{URL: nurl, Preview: strings.TrimSpace(v.Val), Category: category, Name: strings.TrimSpace(n.Text()), StoreID: strings.Replace(storeid, "id", "", -1)}
						}
					}
				}
			})
			if found < 3 || page > 2 {
				break
			}
		} //page loop
	} //url loop

	//dont leave your friend behind :-)
	log.Println("parseIt: waiting!")
	close(storeData)
	log.Println("parseIt: closing channel!")
	//send signal -> DONE
	doneFlg <- true
}
