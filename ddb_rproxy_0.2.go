// https://developer20.com/writing-proxy-in-go/?utm_source=reddit&utm_medium=link&utm_campaign=proxy-in-go

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const DDB_ENDPOINT = "http://dynamodb.ap-northeast-2.amazonaws.com:80"

type Metrics struct {
	CurrentTime time.Time     `json: currenttime`
	Method      string        `json: method`
	TableName   string        `json: tablename`
	RequestBody map[string]interface{}        `json: requestbody`
	ElapsedTime time.Duration `json: elapsedtime`
}

type RequestMetric struct {
	reqheader http.Header
	reqbody   map[string]interface{}
        resheader http.Header
}

func main() {
	url, err := url.Parse(DDB_ENDPOINT)
	if err != nil {
		panic(err)
	}

	port := flag.Int("p", 8000, "port")
	flag.Parse()

	director := func(req *http.Request) {
		req.URL.Scheme = url.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.URL.Host = url.Host
		// req.Host = req.URL.Host
	}
	reverseProxy := &httputil.ReverseProxy{Director: director}
	reverseProxy.ModifyResponse = func(res *http.Response) error {
	        responseContent := map[string]interface{}{}
	        resST, resHeader, err := parseResponse(res, &responseContent)
	        if err != nil {
		        return err
		}
		log.Printf("resHeader: %v \n", resHeader)
		log.Printf("resST: %v \n", resST)
		return captureMetrics(responseContent)
	}
	handler := handler{proxy: reverseProxy}
	http.Handle("/", handler)

	starting_msg := "Listening on port 8000"
	log.Printf(starting_msg)

	err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *port), nil)
	if err != nil {
		panic(err)
	}
}

type handler struct {
	proxy *httputil.ReverseProxy
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// h.proxy.ServeHTTP(w, r)
	rm := &RequestMetric{}
	rm.reqheader = r.Header
	// log.Printf("request Header: %v, \n, \n, %v, %v", r, r.Header, r.RequestURI)
	log.Printf("##########################################")
	requestContent := map[string]interface{}{}
	parseReqBody(r, &requestContent)
	rm.reqbody = requestContent
	//log.Printf("parseReqBody type: %T", reqbody)
	//log.Printf("parse request body: %v \n ", string(reqbody))
	log.Printf("parse json body: %v \n ", requestContent)
	log.Printf("parse table name: %v \n ", requestContent["TableName"])
	//log.Printf("parse json body2: %v \n ", jsonbody)
	log.Printf("request Header & Body: %v \n %v\n", rm.reqheader, rm.reqbody)
	starttime := time.Now()
	h.proxy.ServeHTTP(w, r)
	endtime := time.Now()
	elapsed := endtime.Sub(starttime)
    //resheader := func(res *http.Response) http.Header{
    //    log.Printf("resheader: %v \n", res.Header)
    //    return res.Header
    //    }
    //log.Printf("rrrr: %v \n", resheader)

	// sending to firehose
	streamName := "ddbhose"
	sess := session.Must(session.NewSession())
	firehoseService := firehose.New(sess, aws.NewConfig().WithRegion("ap-northeast-2"))
	recordsInput := &firehose.PutRecordInput{}
	recordsInput = recordsInput.SetDeliveryStreamName(streamName)
    tn := requestContent["TableName"].(string)
	metrics := &Metrics{
		CurrentTime: starttime,
		Method:      strings.Split(r.Header["X-Amz-Target"][0], ".")[1],
		TableName:   tn,
		RequestBody: requestContent,
		ElapsedTime: elapsed,
	}
	b, _ := json.Marshal(metrics)
	record := &firehose.Record{Data: b}
	recordsInput = recordsInput.SetRecord(record)
	resp, err := firehoseService.PutRecord(recordsInput)
	if err != nil {
		log.Printf("PutRecordBatch err: %v\n", err)
	} else {
		log.Printf("PutRecordBatch: %v\n", resp)
	}
	log.Printf("metrics: %v", *metrics)

}

func parseReqBody(req *http.Request, unmarshalStruct interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	req.Body.Close()
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return json.Unmarshal(body, unmarshalStruct)
}


func parseResponse(res *http.Response, unmarshalStruct interface{}) (int, http.Header, error) {
        resST := res.StatusCode
      	header := res.Header
        body, err := ioutil.ReadAll(res.Body)
	if err != nil {
	       log.Printf("%v", err)
	       return 500, nil, err
	}
	res.Body.Close()
	res.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return resST, header, json.Unmarshal(body, unmarshalStruct)
}

func captureMetrics(m map[string]interface{}) error {
	// Add your metrics capture code here
	log.Printf("resBody = %+v\n", m)
	return nil
}