package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
)

type iCollection interface {
	poetryList() <-chan iPoetry
	name() string
}

type shijingCollection struct {
	content []*shijing
}

func newShijingCollection(baseDir string) (*shijingCollection, error) {
	col := new(shijingCollection)

	binData, err := ioutil.ReadFile(filepath.Join(baseDir, "shijing", "shijing.json"))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(binData, &col.content)
	if err != nil {
		return nil, err
	}

	return col, nil
}

func (c *shijingCollection) name() string {
	return "shijing"
}

func (c *shijingCollection) poetryList() <-chan iPoetry {
	ret := make(chan iPoetry)

	go func() {
		defer close(ret)
		for _, line := range c.content {
			ret <- line
		}
	}()
	return ret
}

type tangCollection struct {
	rankedCollection
}

func newTangCollection(baseDir string) (*tangCollection, error) {
	poetris, err := loadRankedCollection(baseDir, "tang")
	if err != nil {
		return nil, err
	}

	col := &tangCollection{rankedCollection{poetris}}
	return col, err
}

func (c *tangCollection) name() string {
	return "tang"
}

type songCollection struct {
	rankedCollection
}

func newSongCollection(baseDir string) (*tangCollection, error) {
	poetris, err := loadRankedCollection(baseDir, "song")
	if err != nil {
		return nil, err
	}

	col := &tangCollection{rankedCollection{poetris}}
	return col, err
}

func (c *songCollection) name() string {
	return "song"
}

type rankedCollection struct {
	content []*rankedPoetry
}

func (c *rankedCollection) poetryList() <-chan iPoetry {
	ret := make(chan iPoetry)

	go func() {
		defer close(ret)
		for _, line := range c.content {
			ret <- line
		}
	}()
	return ret
}

type rankFilePoetry struct {
	Author string `json:"author"`
	Title  string `json:"title"`

	Baidu  int64 `json:"baidu"`
	So360  int64 `json:"so360"`
	Bing   int64 `json:"bing"`
	BingEn int64 `json:"bing_en"`
	Google int64 `json:"google"`
}

func loadRankedCollection(baseDir string, collectionName string) ([]*rankedPoetry, error) {
	contentpath := filepath.Join(baseDir, "json")
	rankpath := filepath.Join(baseDir, "rank", "poet")

	contentFiles, err := ioutil.ReadDir(contentpath)
	if err != nil {
		return nil, err
	}
	rankFiles, err := ioutil.ReadDir(rankpath)
	if err != nil {
		return nil, err
	}

	ret := []*rankedPoetry{}
	titleAuthMap := make(map[string]*rankedPoetry)

	// load content
	for _, fi := range contentFiles {
		if fi.IsDir() {
			continue
		}

		nameExtract := extractFilename(fi.Name(), filetypeContent)
		if nameExtract == nil ||
			nameExtract.collectionName != collectionName ||
			nameExtract.filetype != filetypeContent {
			continue
		}

		log.Println("processing", fi.Name())
		contentFilepath := filepath.Join(contentpath, fi.Name())
		binData, err := ioutil.ReadFile(contentFilepath)
		if err != nil {
			log.Println("failed reading file", contentFilepath, err)
			return nil, err
		}

		var rpList []*rankedPoetry
		err = json.Unmarshal(binData, &rpList)
		if err != nil {
			return nil, err
		}

		ret = append(ret, rpList...)
		for _, rp := range rpList {
			key := makeRankedPoetryKey(*rp.Title, *rp.Author)
			titleAuthMap[key] = rp
		}
	}

	// load rank
	for _, fi := range rankFiles {
		if fi.IsDir() {
			continue
		}

		nameExtract := extractFilename(fi.Name(), filetypeRank)
		if nameExtract == nil ||
			nameExtract.collectionName != collectionName ||
			nameExtract.filetype != filetypeRank {
			continue
		}

		log.Println("processing", fi.Name())
		fullpath := filepath.Join(rankpath, fi.Name())
		binData, err := ioutil.ReadFile(fullpath)
		if err != nil {
			log.Println("failed reading file", fullpath, err)
			return nil, err
		}

		var rfpList []*rankFilePoetry
		err = json.Unmarshal(binData, &rfpList)
		if err != nil {
			return nil, err
		}

		for _, rfp := range rfpList {
			key := makeRankedPoetryKey(rfp.Title, rfp.Author)

			rp := titleAuthMap[key]
			if rp == nil {
				continue
			}

			rp.baiduInfluence = rfp.Baidu
			rp.bingInfluence = rfp.Bing
			rp.googleInfluence = rfp.Google
		}
	}

	calculateRank(ret)

	return ret, nil
}

func makeRankedPoetryKey(title, author string) string {
	return fmt.Sprintf("%s::%s", title, author)
}

func calculateRank(ret []*rankedPoetry) {
	size := len(ret)
	baidu := make([]*rankedPoetry, size)
	bing := make([]*rankedPoetry, size)
	google := make([]*rankedPoetry, size)
	copy(baidu, ret)
	copy(bing, ret)
	copy(google, ret)

	// TODO make code cleaner
	sort.Slice(baidu, func(i, j int) bool { return baidu[i].baiduInfluence < baidu[j].baiduInfluence })
	sort.Slice(bing, func(i, j int) bool { return bing[i].bingInfluence < bing[j].bingInfluence })
	sort.Slice(google, func(i, j int) bool { return google[i].googleInfluence < google[j].googleInfluence })
	for idx, rp := range baidu {
		rp.baiduRank = float64(idx) / float64(size)
	}
	for idx, rp := range bing {
		rp.bingRank = float64(idx) / float64(size)
	}
	for idx, rp := range google {
		rp.googleRank = float64(idx) / float64(size)
	}
}

const (
	filetypeContent = "content"
	filetypeRank    = "rank"
)

var (
	fileType2Regexp = map[string]*regexp.Regexp{
		filetypeContent: regexp.MustCompile(`poet\.(?P<collectionname>[a-z_]+)\.(?P<fileno>\d+)\.json`),
		filetypeRank:    regexp.MustCompile(`poet\.(?P<collectionname>[a-z_]+)\.rank\.(?P<fileno>\d+)\.json`),
	}
)

type filenameParseResult struct {
	filetype       string
	fileno         string
	collectionName string
}

func extractFilename(filename string, filetype string) *filenameParseResult {
	p := fileType2Regexp[filetype]
	if p == nil {
		return nil
	}

	ret := new(filenameParseResult)

	m := p.FindStringSubmatch(filename)
	if len(m) == 0 {
		return nil
	}

	ret.filetype = filetype
	for i, name := range p.SubexpNames() {
		if i == 0 {
			continue
		}

		if m[i] == "" {
			return nil
		}

		switch name {
		case "collectionname":
			ret.collectionName = m[i]
		case "fileno":
			ret.fileno = m[i]
		}
	}
	return ret
}
