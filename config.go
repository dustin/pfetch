package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"regexp"
)

type command struct {
	Path string   `xml:"path,attr"`
	Arg  []string `xml:"arg"`
}

type ErrorHandler struct {
	Notify string `xml:"notify,attr"`
}

type url struct {
	HREF      string         `xml:"href,attr"`
	Output    string         `xml:"output,attr"`
	RSrc      []string       `xml:"mustmatch"`
	NRSrc     []string       `xml:"mustnotmatch"`
	Freq      int            `xml:"freq,attr"`
	Command   command        `xml:"command"`
	OnError   []ErrorHandler `xml:"onerror"`
	OnRecover []ErrorHandler `xml:"onrecover"`

	matchPatterns    []*regexp.Regexp
	negMatchPatterns []*regexp.Regexp
}

type Notifier struct {
	Name string   `xml:"name,attr"`
	Type string   `xml:"type,attr"`
	Arg  []string `xml:"arg"`
}

type pfetchConf struct {
	Notifiers []Notifier `xml:"notifiers>notifier"`
	Url       []*url     `xml:"url"`
}

var config pfetchConf

func (u *url) String() string {
	return fmt.Sprintf("{%v -> %#v}", u.HREF, u.Output)
}

func getNamedNotifier(name string) *Notifier {
	for i, notifier := range config.Notifiers {
		if notifier.Name == name {
			return &config.Notifiers[i]
		}
	}
	return nil
}

func loadConfig(path string) {
	f, e := os.Open(path)
	if e != nil {
		log.Fatalf("Error opening config:  %v", e)
	}
	defer f.Close()

	e = xml.NewDecoder(f).Decode(&config)
	if e != nil {
		log.Fatalf("Error parsing xml: %v", e)
	}

	for i, u := range config.Url {
		u.matchPatterns = make([]*regexp.Regexp, 0, len(u.RSrc))
		for _, r := range u.RSrc {
			config.Url[i].matchPatterns = append(config.Url[i].matchPatterns,
				regexp.MustCompile(r))
		}

		u.negMatchPatterns = make([]*regexp.Regexp, 0, len(u.NRSrc))
		for _, r := range u.NRSrc {
			config.Url[i].negMatchPatterns = append(config.Url[i].negMatchPatterns,
				regexp.MustCompile(r))
		}

		for _, eh := range u.OnError {
			if getNamedNotifier(eh.Notify) == nil {
				log.Fatalf("Undefined notifier %#v for url %#v",
					eh.Notify, u.HREF)
			}
		}
		for _, eh := range u.OnRecover {
			if getNamedNotifier(eh.Notify) == nil {
				log.Fatalf("Undefined notifier %#v for url %#v",
					eh.Notify, u.HREF)
			}
		}
	}
}
