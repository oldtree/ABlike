package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var remoteUrl = flag.String("URL", "www.baidu.com", "target url")
var times = flag.Int("times", 10, "request times ")
var ParamJsonPath = flag.String("JsonPath", "./params.json", "request params")

func ParseParamFromJson(path string) map[string]interface{} {
	defer func() {
		if x := recover(); x != nil {
			fmt.Println("error happened")
		}
	}()
	//path = "D:/workspace/src/PostData/params.json"
	body, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(body, &m)
	if err != nil {
		panic(err)
	}

	return m
}

type RespContent struct {
	Errorcode   string `json:"errorcode"`
	Description string `json:"description"`
}

type Analysis struct {
	URL        string
	ReqNumber  int
	SuccNumber int
	TimeCost   int64
	REQPS      int
	params     map[string]interface{}
	header     map[string]string
	RespChan   chan *RespContent
	Done       chan bool
	group      sync.WaitGroup
}

func (a *Analysis) Init() {
	a.params = ParseParamFromJson(*ParamJsonPath)
	if a.params == nil {
		fmt.Println("an empty json params ")
	}
	a.ReqNumber = *times
	if a.ReqNumber == 0 {
		fmt.Println("request number can not be 0,at lest 1")
		return
	}
	a.URL = *remoteUrl
	if a.URL == "" {
		fmt.Println("url path shoud not be empty ")
		return
	}
	a.header = make(map[string]string)
	a.header["Connection"] = "close"
	a.header["Accept-Charset"] = "utf-8"
	a.header["Cache-Control"] = "max-age=0"
	a.header["Content-Type"] = "application/x-www-form-urlencoded; param=value;charset=UTF-8"

	a.RespChan = make(chan *RespContent, 100)

	a.Done = make(chan bool, 1)
}
func (a *Analysis) ResponseToStruct(resp *http.Response, v interface{}) error {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	err = json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	return nil
}

func (a *Analysis) NewAction(method string) (resp *http.Response, err error) {
	client := &http.Client{}
	values := url.Values{}
	for key, value := range a.params {
		values.Set(key, value.(string))
	}
	valuesReader := bytes.NewReader([]byte(values.Encode()))
	req, err := http.NewRequest(method, a.URL, valuesReader)

	for key, value := range a.header {
		req.Header.Set(key, value)
	}
	if err != nil {
		return nil, err
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *Analysis) TaskWorker() {

	Resp, err := a.NewAction("POST")
	if err != nil {
		fmt.Println(err)
	}
	var content = new(RespContent)
	a.ResponseToStruct(Resp, content)
	a.RespChan <- content
}

func (a *Analysis) TaskLoop() {
	var isclosed bool
	isclosed = false
	tB := time.Now().Unix()

	for !isclosed {
		select {
		case resp, c := <-a.RespChan:
			if resp.Errorcode == "0" {
				a.SuccNumber++
			}
			if c == false {
				isclosed = true
			}

		default:
		}
	}
	tA := time.Now().Unix()
	a.TimeCost = tA - tB
	a.Done <- true
}

func (a *Analysis) OutPutFormat() {
	fmt.Println(*a)
}
