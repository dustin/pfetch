package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"log/syslog"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func init() {
	http.DefaultTransport = &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
	}
}

func changed(u *url, res *http.Response) bool {
	var f io.Writer
	var tmpfile string
	var err error

	if u.Output == "" {
		f = ioutil.Discard
	} else {
		tmpfile = u.Output + ".tmp"
		fd, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("Error opening %s: %v", tmpfile, err)
			return false
		}
		defer fd.Close()
		f = fd
	}

	if len(u.matchPatterns) > 0 {
		bytes, err := ioutil.ReadAll(io.TeeReader(res.Body, f))
		if err != nil {
			handleErrors(u,
				fmt.Errorf("error reading stream: %v", err))
			return false
		}
		for i, p := range u.matchPatterns {
			if !p.Match(bytes) {
				handleErrors(u,
					fmt.Errorf("failed to match pattern: %v",
						u.RSrc[i]))
				return false
			}
		}
		for i, p := range u.negMatchPatterns {
			if p.Match(bytes) {
				handleErrors(u,
					fmt.Errorf("matched negative pattern: %v",
						u.NRSrc[i]))
				return false
			}
		}
	} else {
		_, cerr := io.Copy(f, res.Body)
		if cerr != nil {
			handleErrors(u,
				fmt.Errorf("error copying stream: %v", cerr))
			return false
		}
	}

	if u.Output != "" {
		if err = os.Rename(tmpfile, u.Output); err != nil {
			handleErrors(u,
				fmt.Errorf("error moving tmp file (%s) into place (%s): %v",
					tmpfile, u.Output, err))
			return false
		}
	}
	if u.Command.Path != "" {
		env := append(os.Environ(),
			fmt.Sprintf("%s=%s", "PFETCH_URL", u.HREF),
			fmt.Sprintf("%s=%s", "PFETCH_FILE", u.Output))
		cmd := exec.Cmd{Path: u.Command.Path,
			Args: append([]string{u.Command.Path},
				u.Command.Arg...),
			Env: env,
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			handleErrors(u,
				fmt.Errorf("error running %s: (%v): %v\n%s",
					u.Command.Path, u.Command.Arg, err,
					string(output)))
			return false
		}
	}
	return true
}

func handleResponse(u *url, req *http.Request, res *http.Response) {
	defer res.Body.Close()
	// Set up conditional request if we got an etag
	if etag := res.Header.Get("ETag"); etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	switch res.StatusCode {
	case 200:
		if changed(u, res) {
			handleSuccess(u)
		}
	case 304:
		handleSuccess(u)
	default:
		handleErrors(u, fmt.Errorf("%v", res.Status))
		log.Printf("%d for %s", res.StatusCode, u.HREF)
	}
}

func (u *url) freq() time.Duration {
	freq, err := time.ParseDuration(u.Freq)
	if err != nil {
		log.Fatalf("Error parsing duration %v for %v: %v", u.Freq, u.HREF, err)
	}
	return freq
}

func loop(u *url, req *http.Request) {
	for _ = range time.Tick(u.freq()) {
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			handleErrors(u, err)
		} else {
			handleResponse(u, req, res)
		}
	}
}

func schedule(u *url) {
	freq := u.freq()
	start := time.Duration(rand.Int31()%int32(freq.Seconds())) * time.Second
	log.Printf("Scheduling %s -> %s every %v, starting in %v",
		u.HREF, u.Output, freq, start)
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

	time.AfterFunc(start, func() { go loop(u, req) })
}

func initLogger(slog bool) {
	if slog {
		sl, err := syslog.New(syslog.LOG_INFO, "pfetch")
		if err != nil {
			log.Fatalf("Can't initialize logger: %v", err)
		}
		log.SetOutput(sl)
		log.SetFlags(0)
	}
}

func main() {
	confPath := flag.String("config", "urls.xml", "Path to config")
	useSyslog := flag.Bool("syslog", false, "Log to syslog")

	flag.Parse()

	initLogger(*useSyslog)

	loadConfig(*confPath)

	if len(config.URL) == 0 {
		log.Fatalf("No URLs found.")
	}

	go notifyLoop()

	for _, u := range config.URL {
		schedule(u)
	}

	// goroutines are doing the work
	select {}
}
