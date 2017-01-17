package main

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
)

func InitAlarming(ws *restful.WebService) *Alarming {
	alr := Alarming{
		alarmRequests: make(chan ReqResp),
	}
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

type Alarming struct {
	alarmList     []AlarmEntry
	alarmRequests chan ReqResp
}

type AlarmEntry struct {
	Text string
	Kind string
}

type ReqResp struct {
	r *restful.Request
	w *restful.Response
	d *chan bool
	f func(alr *Alarming, r *restful.Request, w *restful.Response)
}

func (alr *Alarming) Run() {
	go func() {
		for {
			rr := <-alr.alarmRequests
			rr.f(alr, rr.r, rr.w)
			(*rr.d) <- true
		}
	}()

}

func getAlarmListUnsafe(alr *Alarming, r *restful.Request, w *restful.Response) {
	w.WriteAsJson(alr.alarmList)
	alr.alarmList = []AlarmEntry{}
}

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
