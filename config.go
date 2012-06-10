package main

import (
	"encoding/xml"
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
	HREF    string         `xml:"href,attr"`
	Output  string         `xml:"output,attr"`
	RSrc    []string       `xml:"mustmatch"`
	Freq    int            `xml:"freq,attr"`
	Command command        `xml:"command"`
	OnError []ErrorHandler `xml:"onerror"`

	matchPatterns []*regexp.Regexp
}

type Notifier struct {
	Name string   `xml:"name,attr"`
	Type string   `xml:"type,attr"`
	Arg  []string `xml:"arg"`
}

type pfetchConf struct {
	Notifiers []Notifier `xml:"notifiers>notifier"`
	Url       []url      `xml:"url"`
}

var config pfetchConf

func getNamedNotifier(name string) *Notifier {
	for _, notifier := range config.Notifiers {
		if notifier.Name == name {
			return &notifier
		}
	}
	return nil
}

func loadConfig(path string) {
	f, e := os.Open("urls.xml")
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
			log.Printf("Compiling %v", r)
			config.Url[i].matchPatterns = append(config.Url[i].matchPatterns,
				regexp.MustCompile(r))
		}

		for _, eh := range u.OnError {
			if getNamedNotifier(eh.Notify) == nil {
				log.Fatalf("Undefined notifier %#v for url %#v",
					eh.Notify, u.HREF)
			}
		}
	}
}
