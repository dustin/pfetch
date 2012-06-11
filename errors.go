package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dustin/nma.go"
)

type notification struct {
	u   *url
	err error
}

var notificationChan = make(chan notification)

func alertNMA(u *url, msgText, key, app, pri string) error {
	i, e := strconv.Atoi(pri)
	if e != nil {
		return e
	}

	n := nma.New(key)

	msg := nma.Notification{
		Application: app,
		Description: msgText,
		Event:       "pfetch",
		Priority:    i,
	}

	return n.Notify(&msg)
}

func notifyNamed(u *url, msgText, name string) {
	notifier := getNamedNotifier(name)
	if notifier != nil {
		if notifier.Type == "nma" && len(notifier.Arg) == 3 {
			err := alertNMA(u, msgText,
				notifier.Arg[0],
				notifier.Arg[1],
				notifier.Arg[2])
			if err != nil {
				log.Printf("Error sending NMA message: ", err)
			}
		}
	} else {
		log.Printf("Couldn't find notifier named %v", name)
	}
}

func notifyFailure(current map[*url]time.Time, n notification) {
	now := time.Now()
	outstanding, found := current[n.u]
	if outstanding.Before(now) {
		if found {
			delete(current, n.u)
		}

		for _, eh := range n.u.OnError {
			log.Printf("Sending to %v", eh)
			notifyNamed(n.u,
				fmt.Sprintf("Problem with %v: %v",
					n.u.HREF, n.err),
				eh.Notify)
		}

		current[n.u] = time.Now().Add(time.Hour)
	} else {
		log.Printf("Too soon to alert, next up at %v",
			outstanding)
	}
}

func notifySuccess(current map[*url]time.Time, n notification) {
	_, found := current[n.u]
	if found {
		delete(current, n.u)
		log.Printf("Was broken. Bringing it back.")
		for _, eh := range n.u.OnRecover {
			log.Printf("Sending to %v", eh)
			notifyNamed(n.u,
				fmt.Sprintf("Recovery from %v", n.u.HREF),
				eh.Notify)
		}
	}
}

func notifier() {
	current := map[*url]time.Time{}

	for n := range notificationChan {
		if n.err == nil {
			notifySuccess(current, n)
		} else {
			notifyFailure(current, n)
		}
	}
}

func handleErrors(u *url, err error) {
	log.Printf("Error in response: %v", err)
	notificationChan <- notification{u, err}
}

func handleSuccess(u *url) {
	notificationChan <- notification{u, nil}
}
