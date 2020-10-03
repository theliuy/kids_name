package main

import (
	"fmt"
	"strings"

	"github.com/mozillazg/go-pinyin"
)

type iPoetry interface {
	headline() string
	lines() []string
	containsTonLines(string) ([]string, bool)
	filtered(float64, bool) bool
}

type poetry struct {
	Author     *string  `json:"author"`
	Chapter    *string  `json:"chapter"`
	Content    []string `json:"content"`
	Id         *string  `json:"id"`
	Paragraphs []string `json:"paragraphs"`
	Section    *string  `json:"section"`
	Title      *string  `json:"title"`
}

func (p *poetry) headline() string {
	comp := []string{}
	for _, field := range []*string{
		p.Title, p.Chapter, p.Section, p.Author,
	} {
		if field == nil {
			continue
		}

		comp = append(comp, *field)
	}
	return strings.Join(comp, " | ")
}

func (p *poetry) lines() []string {
	if len(p.Content) > 0 {
		return p.Content
	}
	return p.Paragraphs
}

func (p *poetry) fullContent() string {
	return strings.Join(p.lines(), "\n")
}

func (p *poetry) filtered(threshold float64, strictMode bool) bool {
	return false
}

var (
	pinyinArgs = pinyin.Args{
		Style:     pinyin.Tone3,
		Heteronym: true,
		Separator: pinyin.Separator,
		Fallback:  pinyin.Fallback,
	}
)

func (p *poetry) containsTonLines(ton3 string) ([]string, bool) {
	for _, line := range p.lines() {
		results := pinyin.Pinyin(line, pinyinArgs)
		for _, charPyList := range results {
			for _, charTon3 := range charPyList {
				if ton3 == charTon3 {
					return p.lines(), true
				}
			}
		}
	}
	return nil, false
}

type shijing struct {
	poetry
}

func (p *shijing) containsTonLines(ton3 string) ([]string, bool) {
	var ret []string
	for _, line := range p.lines() {
		results := pinyin.Pinyin(line, pinyinArgs)
		for _, charPyList := range results {
			for _, charTon3 := range charPyList {
				if ton3 == charTon3 {
					ret = append(ret, line)
				}
			}
		}
	}
	return ret, len(ret) > 0
}

type rankedPoetry struct {
	poetry

	baiduInfluence  int64
	bingInfluence   int64
	googleInfluence int64

	baiduRank  float64
	bingRank   float64
	googleRank float64
}

type tang struct {
	rankedPoetry
}

type song struct {
	rankedPoetry
}

func (p *rankedPoetry) containsTonLines(ton3 string) ([]string, bool) {
	start := 0
	hit := false

	hitLines := []string{}
	lines := p.lines()
	size := len(lines)

	for idx, line := range lines {
		results := pinyin.Pinyin(line, pinyinArgs)
		for _, charPyList := range results {
			for _, charTon3 := range charPyList {
				if ton3 == charTon3 {
					hit = true
				}
			}
		}

		if idx%2 == 1 || idx == size {
			if hit {
				for i := start; i <= idx; i += 1 {
					hitLines = append(hitLines, lines[i])
				}
				hitLines = append(hitLines, "...")
			}
			start = idx + 1
			hit = false
		}
	}
	return hitLines, len(hitLines) > 0
}

func (p *rankedPoetry) headline() string {
	comp := []string{}
	rankStr := fmt.Sprintf("bd:%.2f bi:%.2f gg:%.2f", p.baiduRank, p.bingRank, p.googleRank)
	for _, field := range []*string{
		p.Title, p.Chapter, p.Section, p.Author, &rankStr,
	} {
		if field == nil {
			continue
		}

		comp = append(comp, *field)
	}
	return strings.Join(comp, " | ")
}

func (p *rankedPoetry) filtered(threshold float64, strictMode bool) bool {
	if strictMode {
		return p.baiduRank < threshold || p.bingRank < threshold || p.googleRank < threshold
	}
	return p.baiduRank < threshold && p.bingRank < threshold && p.googleRank < threshold
}
