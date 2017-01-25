/*
why is the *channel* and *goroutine* approach used?

We used ReqResp type channel to pass data from one go routine to another
so that we can avoid mutex to handle data across go routines. This way the overhead is very less.

what would be an alternative / more traditional way?

Alternative way would be using `GLOBAL` variable to keep track of all the alarm entries.
Passing the global handle to the required functions to perform read and write operation
but this is not thread-safe way, overhead is higher when we deal with multiple go routines
sharing the same GLOBAL handle.

*/

package main

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
)

// InitAlarming initialising alarm routes
func InitAlarming(ws *restful.WebService) *Alarming {
	alr := Alarming{
		alarmRequests: make(chan ReqResp),
	}

	// this is the route that responses to `HTTP POST` method to persist the incoming alarm entry
	// Which is using anonymous function to send data via channel `alarmRequests`
	// And we have a boolean channel to signal method end.
	ws.Route(
		ws.POST("/alarm").Doc("post alarm").
			To(func(r *restful.Request, w *restful.Response) {
				done := make(chan bool)
				alr.alarmRequests <- ReqResp{r, w, &done, postAlarmUnsafe}
				<-done
			}).
			Operation("postalarm").
			Param(ws.BodyParameter("alarm", "alarm content as json").DataType("main.AlarmEntry")).
			Writes(AlarmEntry{}))

	// this route responses to `GET HTTP` method to list all the available alarm entry
	// We pass `ReqResp` data through `alarmRequests` channel
	// And we have a boolean channel defined to signal the method end at other end.
	ws.Route(
		ws.GET("/alarmlist").Doc("get alarmlist").
			To(func(r *restful.Request, w *restful.Response) {
				done := make(chan bool)
				alr.alarmRequests <- ReqResp{r, w, &done, getAlarmListUnsafe}
				<-done
			}).
			Operation("getalarmlist").
			Writes([]string{}))

	return &alr
}

// Alarming is a struct represents list of alarm entry and an unbuffered channel
type Alarming struct {
	alarmList     []AlarmEntry
	alarmRequests chan ReqResp
}

// AlarmEntry is a basic model to represent a single alarm entity.
type AlarmEntry struct {
	Text string
	Kind string
}

// ReqResp is type for channel `alarmRequests`
type ReqResp struct {
	r *restful.Request
	w *restful.Response
	d *chan bool
	f func(alr *Alarming, r *restful.Request, w *restful.Response)
}

// Run perform the specific operation depending upon the routes we select
// and receives its data from channel
func (alr *Alarming) Run() {
	go func() {
		for {
			rr := <-alr.alarmRequests
			rr.f(alr, rr.r, rr.w)
			(*rr.d) <- true
		}
	}()

}

// getAlarmListUnsafe write the list of alarm to response writer and clears the list.
func getAlarmListUnsafe(alr *Alarming, r *restful.Request, w *restful.Response) {
	w.WriteAsJson(alr.alarmList)
	alr.alarmList = []AlarmEntry{}
}

// postAlarmUnsafe write the single alarm entry to response writer
// then appends the data to `alarmList` struct to keep track of all created alarm entry
func postAlarmUnsafe(alr *Alarming, r *restful.Request, w *restful.Response) {
	entry := AlarmEntry{}
	err := r.ReadEntity(&entry)
	if err != nil {
		w.WriteError(http.StatusNotAcceptable, fmt.Errorf(" cannot read alarm entry, err= %v", err))
		return
	}
	w.WriteAsJson(entry)
	alr.alarmList = append(alr.alarmList, entry)
}
