package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"gopkg.in/yaml.v2"
)

const version string = "0.0.1"

// webservice definition, swagger registration and start:
func main() {
	ws := new(restful.WebService)
	ws.Path("/goass").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/version").Doc("get version number").
		To(func(r *restful.Request, w *restful.Response) { io.WriteString(w, version) }).
		Operation("version").
		Produces(restful.MIME_OCTET))

	ws.Route(
		ws.GET("/plant/{name}").Doc("get plant data").
			To(getplant).
			Operation("getplant").
			Param(ws.PathParameter("name", "plant name")).
			Writes(PlantDef{}))

	ws.Route(
		ws.GET("/plant/{name}/totalpower").Doc("get plant total power").
			To(getplantpower).
			Operation("getplantpower").
			Param(ws.PathParameter("name", "plant name")).
			Writes(struct{ Power float32 }{}))

	// simple alarming mock implemented in file alarming.go
	alarming := InitAlarming(ws)
	alarming.Run()

	restful.Add(ws)
	swagger.RegisterSwaggerService(
		swagger.Config{
			WebServices:     restful.DefaultContainer.RegisteredWebServices(), // you control what services are visible
			WebServicesUrl:  "/",
			ApiPath:         "/apidocs.json",
			SwaggerPath:     "/apidocs/",
			SwaggerFilePath: "./swaggerui"},
		restful.DefaultContainer)
	err := http.ListenAndServe(":8123", nil)
	if err != nil {
		log.Fatal(err)
	}
}

// definition of the plantlist JSON structure:
type DataDef struct {
	Plants []PlantDef `yaml:"plants" json:"plants"`
}

// definition of the plant definition JSON structure:
type PlantDef struct {
	Name   string `yaml:"name" json:"name"`
	Place  string `yaml:"place" json:"place"`
	Blocks []struct {
		Name    string `yaml:"name" json:"name"`
		Loggers []struct {
			Name      string `yaml:"name" json:"name"`
			Inverters []struct {
				Addr  int     `yaml:"addr" json:"addr"`
				Power float32 `yaml:"power" json:"power"`
			} `yaml:"inverters" json:"inverters"`
		} `yaml:"loggers" json:"loggers"`
	} `yaml:"blocks" json:"blocks"`
}

// webservice to read plant definition from test.yaml and return as JSON:
func getplant(r *restful.Request, w *restful.Response) {
	plant := r.PathParameter("name")
	if plant == "" {
		w.WriteError(http.StatusNotAcceptable, fmt.Errorf("plant name required"))
		return
	}

	thefile, err := ioutil.ReadFile("./test.yaml")
	if err != nil {
		w.WriteError(http.StatusNotAcceptable, fmt.Errorf(" cannot read definition, err= %v", err))
		return
	}

	data := DataDef{}
	err = yaml.Unmarshal(thefile, &data)
	if err != nil {
		w.WriteError(http.StatusNotAcceptable, fmt.Errorf(" cannot unmarshal definition, err= %v", err))
		return
	}

	for _, p := range data.Plants {
		if p.Name == plant {
			w.WriteAsJson(p)
			return
		}
	}
	w.WriteError(http.StatusNotAcceptable, fmt.Errorf("sorry, plant %s not vailable", plant))

}

// webservice to sum up all inverter powers in plant and return as json
func getplantpower(r *restful.Request, w *restful.Response) {
	plant := r.PathParameter("name")
	if plant == "" {
		w.WriteError(http.StatusNotAcceptable, fmt.Errorf("plant name required"))
		return
	}

	file, err := ioutil.ReadFile("./test.yaml")
	if err != nil {
		w.WriteError(http.StatusNotAcceptable, fmt.Errorf(" cannot read definition, err= %v", err))
		return
	}

	data := DataDef{}
	if err = yaml.Unmarshal(file, &data); err != nil {
		w.WriteError(http.StatusNotAcceptable, fmt.Errorf(" cannot unmarshal definition, err= %v", err))
		return
	}

	var totalPower float32
	for _, p := range data.Plants {
		if p.Name == plant {
			for _, b := range p.Blocks {
				for _, l := range b.Loggers {
					for _, i := range l.Inverters {
						totalPower += i.Power
					}
				}
			}
			break
		}
	}
	w.WriteAsJson(struct{ Power float32 }{totalPower})
}
