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

func alertNMA(u *url, err error, key, app, pri string) error {
	i, e := strconv.Atoi(pri)
	if e != nil {
		return e
	}

	n := nma.New(key)

	msgText := fmt.Sprintf("Problem with %s: %v", u.HREF, err)

	msg := nma.Notification{
		Application: app,
		Description: msgText,
		Event:       "pfetch",
		Priority:    i,
	}

	return n.Notify(&msg)
}

func notifyNamed(u *url, err error, name string) {
	notifier := getNamedNotifier(name)
	if notifier != nil {
		if notifier.Type == "nma" && len(notifier.Arg) == 3 {
			err := alertNMA(u, err,
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

func notifier() {
	current := map[*url]time.Time{}

	for notification := range notificationChan {

		now := time.Now()
		outstanding, found := current[notification.u]
		if outstanding.Before(now) {
			if found {
				delete(current, notification.u)
			}

			for _, eh := range notification.u.OnError {
				log.Printf("Sending to %v", eh)
				notifyNamed(notification.u, notification.err,
					eh.Notify)
			}

			current[notification.u] = time.Now().Add(time.Hour)
		} else {
			log.Printf("Too soon to alert, next up at %v",
				outstanding)
		}
	}
}

func handleErrors(u *url, err error) {
	log.Printf("Error in response: %v", err)
	notificationChan <- notification{u, err}
}
