// https://developer20.com/writing-proxy-in-go/?utm_source=reddit&utm_medium=link&utm_campaign=proxy-in-go
// receiving response body
// Change Logs:
// - date: 2020.08.31
//   - dealing variables with flag to use them as parameters
// - date: 2020.08.21
//   - refactoring variables
//   - retrieve response body, experimental
// Requried:
// - go 1.8

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

//var DDB_ENDPOINT = "http://dynamodb.ap-northeast-2.amazonaws.com:80"

// handling Arguments
var port = flag.Int("port", 8000, "port")
var debug_mode = flag.Bool("debug", false, "enabling debug mode")
var region_name = flag.String("region_name","ap-northeast-2","AWS region name")
var send_to_firehose = flag.Bool("send_to_firehose", true, "enabling to send to firehose")
var stream_name = flag.String("stream_name", "ddbhose", "aws firehose stream name")
// if you set "including_response_body" be 'true', unexpected termination could be happen!!
var including_response_body = false

// Don't need to modify from below
var DDB_ENDPOINT = "http://dynamodb." + *region_name + ".amazonaws.com:80"
var tm = &TotalMetric{}
var metrics = &Metrics{}
var elapsedTime = map[string]time.Time{}
var metricMap = map[string]interface{}{}

type Metrics struct {
	Date        time.Time              `json: date`
	Method      string                 `json: method`
	TableName   string                 `json: tablename`
	Query       string `json: query`
	ElapsedTime int64   `json: elapsedtime`
    Client      string                 `json: client`
    ResponseCode int                   `json: code`
}

type TotalMetric struct {
	reqheader http.Header
	reqbody   map[string]interface{}
    query     string
	resheader http.Header
	resbody   map[string]interface{}
	ressc     int
    reqclient string
}

func main() {
	flag.Parse()
	url, err := url.Parse(DDB_ENDPOINT)
	if err != nil {
		panic(err)
	}


	director := func(req *http.Request) {
		req.URL.Scheme = url.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.URL.Host = url.Host
	}
	reverseProxy := &httputil.ReverseProxy{Director: director}
	reverseProxy.ModifyResponse = func(res *http.Response) error {
		err := tm.setResTotalMetric(res)
		return err
	}
	handler := handler{proxy: reverseProxy}
	http.Handle("/", handler)

	starting_msg := "DDB rproxy is listening on "
	log.Printf("%s %v", starting_msg, *port)

	err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *port), nil)
	if err != nil {
		panic(err)
	}
}

type handler struct {
	proxy *httputil.ReverseProxy
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tm.setReqTotalMetric(r)
	elapsedTime["startTime"] = time.Now()
	h.proxy.ServeHTTP(w, r)

	log.Printf("raw metrics: %v", *tm)
    // tmVals := prettyPrint(*tm)
    //showDebugLog(*debug_mode, "Raw Metrics:", tmVals)
    metrics.setMetrics(tm)

    if *send_to_firehose {
        err := sendToFirehose(metrics)
        if err != nil {
            log.Printf("firehose err: %v", err)
        }
    }
}

func (metrics *Metrics) setMetrics(tm *TotalMetric) error{
    elapsedTime["endTime"] = time.Now()
    metrics.Date = elapsedTime["startTime"]
    metrics.Method = strings.Split(tm.reqheader["X-Amz-Target"][0], ".")[1]
    metrics.TableName = getTableName(metrics)
    metrics.Query = tm.query
    metrics.ElapsedTime = elapsedTime["endTime"].Sub(elapsedTime["startTime"]).Milliseconds()  //unit: millisecond
    metrics.Client = strings.Split(tm.reqclient,":")[0]
    metrics.ResponseCode = tm.ressc
    showDebugLog(*debug_mode, "Metrics:", prettyPrint(metrics))
    return nil
}
func prettyPrint(i interface{}) string {
    s, _:= json.MarshalIndent(i, "", "\t")
    return string(s)
}
func getTableName(m *Metrics) string {
    var tableName string
    if tm.reqbody["TableName"] ==  nil {
        tableName = "none"
    } else {
        tableName = tm.reqbody["TableName"].(string)
    }
    return tableName
}
func (tm *TotalMetric) setReqTotalMetric(req *http.Request) error {
	tm.reqheader = req.Header
        tm.reqclient = req.RemoteAddr //req.Host //req.RequestURI 
	requestBody := map[string]interface{}{}
	query, err := getReqBody(req, &requestBody)
    tm.query = query
	tm.reqbody = requestBody
	// showDebugLog(*debug_mode, "reqbody", requestBody)
	// log.Printf("reqbody: %v", requestBody)
	return err
}

func (tm *TotalMetric) setResTotalMetric(res *http.Response) error {
	if including_response_body {
		responseBody := map[string]interface{}{}
		ressc, resHeader, err := getResponse(res, &responseBody)
		tm.resheader = resHeader
		tm.resbody = responseBody
		tm.ressc = ressc
		return err
	} else {
		ressc, resHeader := getResponse_wo_reqbody(res)
		tm.ressc = ressc
		tm.resheader = resHeader
		return nil
	}
}

func getReqBody(req *http.Request, unmarshalStruct interface{}) (string, error){
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("getReqBody error: %v\n", err)
		return "", err
	}
	req.Body.Close()
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return string(body), json.Unmarshal(body, unmarshalStruct)
}

func getResponse(res *http.Response, unmarshalStruct interface{}) (int, http.Header, error) {
	ressc := res.StatusCode
	header := res.Header
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("getResponse error: %v", err)
		return 500, nil, err
	}
	res.Body.Close()
	res.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return ressc, header, json.Unmarshal(body, unmarshalStruct)
}
func getResponse_wo_reqbody(res *http.Response) (int, http.Header) {
	ressc := res.StatusCode
	header := res.Header
	return ressc, header
}

func sendToFirehose(metrics *Metrics) error {
	// sending to firehose
	sess := session.Must(session.NewSession())
	firehoseService := firehose.New(sess, aws.NewConfig().WithRegion(*region_name))
	recordsInput := &firehose.PutRecordInput{}
	recordsInput = recordsInput.SetDeliveryStreamName(*stream_name)

	// metricMap for firehose to S3, not implemented
	// mMap := tm.parseMetricMap(metricMap)
	// log.Printf("metricMap: %v\n", mMap)

	b, _ := json.Marshal(metrics)
	record := &firehose.Record{Data: b}
	recordsInput = recordsInput.SetRecord(record)
	_, err := firehoseService.PutRecord(recordsInput)
	if err != nil {
		log.Printf("PutRecordBatch err: %v\n", err)
	} else {
		// log.Printf("PutRecordBatch: %v\n", resp)
		log.Printf("ingesting to firehose is completed\n")
	}
	return err
}

func (tm *TotalMetric) parseMetricMap(m map[string]interface{}) map[string]interface{} {
	m["res"] = map[string]map[string]interface{}{"h": {"m": tm.ressc, "Date": tm.resheader["Date"]}}
	m["req"] = map[string]interface{}{"p": tm.reqbody, "h": tm.reqheader}
	return m
}

func showDebugLog(debug_mode bool, topic string, content interface{}){
    if debug_mode {
        log.Printf("%s:  %v", topic, content)
    }
}
