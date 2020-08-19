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
	RequestBody string        `json: requestbody`
	ElapsedTime time.Duration `json: elapsedtime`
}

type RequestMetric struct {
	reqheader http.Header
	reqbody   []byte
}

func main() {
	//rm := &RequestMetric{}

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
	// responseContent := map[string]interface{}{}
	// fmt.Printf("%T", parseReqBody(*http.Request, &responseContent))

	reverseProxy := &httputil.ReverseProxy{Director: director}
	handler := handler{proxy: reverseProxy}
	http.Handle("/", handler)

	starting_msg := "Listening on port 8000"
	log.Printf(starting_msg)

	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
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
	responseContent := map[string]interface{}{}
	reqbody, jsonbody := parseReqBody(r, &responseContent)
	rm.reqbody = reqbody
	log.Printf("parseReqBody type: %T", reqbody)
	log.Printf("parse request body: %v \n ", string(reqbody))
	log.Printf("parse json body: %+v \n ", jsonbody)
	//log.Printf("request Body: %v, %v\n", rm.reqheader, rm.reqbody)
	starttime := time.Now()
	h.proxy.ServeHTTP(w, r)
	endtime := time.Now()
	elapsed := endtime.Sub(starttime)

	// sending to firehose
	streamName := "ddbhose"
	sess := session.Must(session.NewSession())
	firehoseService := firehose.New(sess, aws.NewConfig().WithRegion("ap-northeast-2"))
	recordsInput := &firehose.PutRecordInput{}
	recordsInput = recordsInput.SetDeliveryStreamName(streamName)
	metrics := &Metrics{
		CurrentTime: starttime,
		Method:      strings.Split(r.Header["X-Amz-Target"][0], ".")[1],
		TableName:   "Movies",
		RequestBody: string(reqbody),
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

func parseReqBody(req *http.Request, unmarshalStruct interface{}) ([]byte, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}
	req.Body.Close()
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	// &rm.reqheader = req.Header
	// return json.Unmarshal(body, unmarshalStruct)
	return body, json.Unmarshal(body, unmarshalStruct)
}
