package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dustin/nma.go"
)

func alertNMA(u url, err error, key, app, pri string) error {
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

func notifyNamed(u url, err error, name string) {
	notifier := getNamedNotifier(name)
	if notifier != nil {
		if notifier.alertAfter.After(time.Now()) {
			log.Printf("Too soon to alert, next up at %v",
				notifier.alertAfter)
			return
		}
		if notifier.Type == "nma" && len(notifier.Arg) == 3 {
			err := alertNMA(u, err,
				notifier.Arg[0],
				notifier.Arg[1],
				notifier.Arg[2])
			if err != nil {
				log.Printf("Error sending NMA message: ", err)
			}
		}
		notifier.alertAfter = time.Now().Add(time.Hour)
		log.Printf("Next eligible send: %v", notifier.alertAfter)
	} else {
		log.Printf("Couldn't find notifier named %v", name)
	}
}

func handleErrors(u url, err error) {
	log.Printf("Error in response: %v", err)
	for _, eh := range u.OnError {
		log.Printf("Sending to %v", eh)
		notifyNamed(u, err, eh.Notify)
	}
}
