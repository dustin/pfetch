package main

import (
	"fmt"
	"io"
	"io/ioutil"
	corelog "log"
	"log/syslog"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var log *corelog.Logger

func init() {
	http.DefaultTransport = &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
	}
	_, err := syslog.NewLogger(syslog.LOG_INFO, 0)
	if err != nil {
		corelog.Fatal("Can't initialize logger: %v", err)
	}
	log = corelog.New(os.Stdout, "pfetch: ", 0)
}

func changed(u url, res *http.Response) {
	var f io.Writer
	var tmpfile string
	var err error

	if u.Output == "" {
		f = ioutil.Discard
	} else {
		tmpfile = strings.Join([]string{u.Output, "tmp"}, ".")
		fd, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("Error opening %s: %v", tmpfile, err)
			// XXX:  A real error here.
			return
		}
		defer fd.Close()
		f = fd
	}

	if len(u.matchPatterns) > 0 {
		bytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("Error reading stream: %v", err)
			// XXX:  A real error here
			return
		}
		_, err = f.Write(bytes)
		if err != nil {
			log.Printf("Error writing stream:  %v", err)
			// XXX:  A real error here
			return
		}
		for i, p := range u.matchPatterns {
			if !p.Match(bytes) {
				handleErrors(u,
					fmt.Errorf("Failed to match pattern: %v",
						u.RSrc[i]))
				return
			}
		}
		for i, p := range u.negMatchPatterns {
			if p.Match(bytes) {
				handleErrors(u,
					fmt.Errorf("Matched negative pattern: %v",
						u.NRSrc[i]))
				return
			}
		}
	} else {
		_, cerr := io.Copy(f, res.Body)
		if cerr != nil {
			log.Printf("Error copying stream: %v", cerr)
			// XXX:  A real error here.
			return
		}
	}

	if u.Output == "" {
		log.Printf("Verified %s", u.HREF)
	} else {
		if err = os.Rename(tmpfile, u.Output); err != nil {
			log.Printf("Error moving tmp file (%s) into place (%s): %v",
				tmpfile, u.Output, err)
		}
		log.Printf("Updated %s from %s", u.Output, u.HREF)
	}
	if u.Command.Path != "" {
		env := append(os.Environ(),
			fmt.Sprintf("%s=%s", "PFETCH_URL", u.HREF))
		env = append(env, fmt.Sprintf("%s=%s",
			"PFETCH_FILE", u.Output))
		cmd := exec.Cmd{Path: u.Command.Path,
			Args: append([]string{u.Command.Path},
				u.Command.Arg...),
			Env: env,
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
	switch res.StatusCode {
	case 200:
		changed(u, res)
	case 304:
	default:
		handleErrors(u, fmt.Errorf("%v", res.Status))
		log.Printf("%d for %s", res.StatusCode, u.HREF)
	}
}

func loop(u url, req *http.Request) {
	freq := time.Duration(u.Freq) * time.Second
	for {
		client := &http.Client{}
		res, err := client.Do(req)
		log.Printf("Grabbing %v", u)
		if err != nil {
			handleErrors(u, err)
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
	if len(u.RSrc) > 0 {
		log.Printf("    Will look for %v (%v)", u.RSrc, u.matchPatterns)
	}
	if len(u.NRSrc) > 0 {
		log.Printf("    Will look for not %v (%v)", u.NRSrc, u.negMatchPatterns)
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
	loadConfig("urls.xml")

	if len(config.Url) == 0 {
		log.Fatalf("No URLs found.")
	}

	for _, u := range config.Url {
		schedule(u)
	}

	// goroutines are doing the work
	select {}
}
