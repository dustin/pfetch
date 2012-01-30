package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/syslog"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type command struct {
	Path string   `xml:"path,attr"`
	Arg  []string `xml:"arg"`
}

type url struct {
	HREF    string  `xml:"href,attr"`
	Output  string  `xml:"output,attr"`
	Freq    int     `xml:"freq,attr"`
	Command command `xml:"command"`
}

type urls struct {
	Url []url `xml:"url"`
}

var log = syslog.NewLogger(syslog.LOG_INFO, 0)

func init() {
	http.DefaultTransport = &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
	}
}

func changed(u url, res *http.Response) {
	tmpfile := strings.Join([]string{u.Output, "tmp"}, ".")
	f, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Error opening %s: %v", tmpfile, err)
	}
	defer f.Close()
	_, cerr := io.Copy(f, res.Body)
	if cerr != nil {
		log.Printf("Error copying stream: %v", cerr)
	}
	if err = os.Rename(tmpfile, u.Output); err != nil {
		log.Printf("Error moving tmp file (%s) into place (%s): %v",
			tmpfile, u.Output, err)
	}
	log.Printf("Updated %s from %s", u.Output, u.HREF)
	if u.Command.Path != "" {
		env := append(os.Environ(), fmt.Sprintf("%s=%s", "PFETCH_URL", u.HREF))
		env = append(env, fmt.Sprintf("%s=%s", "PFETCH_FILE", u.Output))
		cmd := exec.Cmd{Path: u.Command.Path,
			Args: append([]string{u.Command.Path}, u.Command.Arg...),
			Env:  env,
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Printf("Error running %s: (%v): %v\n%s",
				u.Command.Path, u.Command.Arg, err,
				string(output))
		}
	}
}

func handleResponse(u url, req *http.Request, res *http.Response) {
	defer res.Body.Close()
	// Set up conditional request if we got an etag
	if etag := res.Header.Get("ETag"); etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if res.StatusCode == 200 {
		changed(u, res)
	} else {
		log.Printf("%d for %s", res.StatusCode, u.HREF)
	}

}

func loop(u url, req *http.Request) {
	freq := time.Duration(u.Freq) * time.Second
	for {
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			log.Printf("Error in response: %v", err)
		} else {
			handleResponse(u, req, res)
		}
		time.Sleep(freq)
	}
}

func schedule(u url) {
	freq := time.Duration(u.Freq) * time.Second
	start := time.Duration(rand.Int31()%int32(u.Freq)) * time.Second
	log.Printf("Scheduling %s -> %s every %s, starting in %s",
		u.HREF, u.Output, freq.String(), start.String())
	if u.Command.Path != "" {
		log.Printf("    Will run> %s %v", u.Command.Path, u.Command.Arg)
	}

	req, err := http.NewRequest("GET", u.HREF, strings.NewReader(""))
	if err != nil {
		log.Fatalf("Error creating request:  %v", err)
	}

	go func() {
		time.Sleep(start)
		loop(u, req)
	}()
}

func main() {
	f, e := os.Open("urls.xml")
	if e != nil {
		log.Fatalf("boo:  %v", e)
	}

	var result urls
	xml.NewDecoder(f).Decode(&result)
	f.Close()

	if len(result.Url) == 0 {
		log.Fatalf("No URLs found.")
	}

	for _, u := range result.Url {
		schedule(u)
	}

	// goroutines are doing the work
	select {}
}
