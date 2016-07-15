package coolmaze

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/pusher/pusher-http-go"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/dispatch", dispatch)
}

// Note that AppEngine doesn't support response streaming.
// The "dispatch" http response will be 1-shot.
func dispatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Only POST method is accepted")
		return
	}

	channelID := r.FormValue("chanID")
	event := "maze-cast"
	message := r.FormValue("message")

	if channelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Mandatory parameter: chanID")
		return
	}
	if message == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Mandatory parameter: message")
		return
	}
	if _, err := strconv.Atoi(channelID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "channelID must be an integer")
		return
	}

	c := appengine.NewContext(r)
	urlfetchClient := urlfetch.Client(c)
	log.Infof(c, "Sending to chan [%v] message [%v]", channelID, message)

	pusherClient := pusher.Client{
		AppId:      "197093",
		Key:        "e36002cfca53e4619c15",
		Secret:     secret,
		HttpClient: urlfetchClient,
	}

	sub, errSub := pubsubSubscribe(c, channelID)
	if errSub != nil {
		log.Warningf(c, "Problem with pubsub sub: %v", errSub)
	}

	data := map[string]string{"message": message}
	be, err := pusherClient.Trigger(channelID, event, data)
	if err != nil {
		log.Errorf(c, "%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Encountered error:", err)
		return
	}
	log.Infof(c, "Events = %v", be)

	if errSub == nil {
		err = waitForAck(c, sub, channelID)
		if err != nil {
			log.Errorf(c, "%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Encountered error:", err)
			return
		}
	}

	fmt.Fprintln(w, "Done :)")
}

var secret string
