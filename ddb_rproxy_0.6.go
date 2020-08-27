// https://developer20.com/writing-proxy-in-go/?utm_source=reddit&utm_medium=link&utm_campaign=proxy-in-go
// receiving response body
// Change Logs:
// - date: 2020.08.21
// - refactoring variables
// - retrieve response body
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
    // "sort"
	"time"
)

const DDB_ENDPOINT = "http://dynamodb.ap-northeast-2.amazonaws.com:80"
var streamName string = "ddbhose"
var regionName string = "ap-northeast-2"
var including_response_body bool = false
var debug_mode bool = true

var tm = &TotalMetric{}
var metrics = &Metrics{}
var elapsedTime = map[string]time.Time{}
var metricMap = map[string]interface{}{}

type Metrics struct {
	Date        time.Time              `json: date`
	Method      string                 `json: method`
	TableName   string                 `json: tablename`
	//Query       map[string]interface{} `json: query`
	Query       string `json: query`
	ElapsedTime int64   `json: elapsedtime`
    Client      string                 `json: client`
    ResponseCode int                   `json: code`
}

type TotalMetric struct {
	reqheader http.Header
	reqbody   map[string]interface{}
	resheader http.Header
	resbody   map[string]interface{}
	ressc     int
    reqclient string
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
	}
	reverseProxy := &httputil.ReverseProxy{Director: director}
	reverseProxy.ModifyResponse = func(res *http.Response) error {
		err := tm.setResTotalMetric(res)
		//elapsedTime["endTime"] = time.Now()
		return err
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
	tm.setReqTotalMetric(r)
	elapsedTime["startTime"] = time.Now()
	h.proxy.ServeHTTP(w, r)

	log.Printf("total metric: %v", *tm)
    metrics.setMetrics(tm)
	// log.Printf("metrics: %v", *metrics)

	err := sendToFirehose(metrics)
	if err != nil {
		log.Printf("firehose err: %v", err)
	}
}

func (metrics *Metrics) setMetrics(tm *TotalMetric) error{
	elapsedTime["endTime"] = time.Now()
    // metrics.Date = tm.resheader["Date"].(time.Time)
    metrics.Date = elapsedTime["startTime"]
    metrics.Method = strings.Split(tm.reqheader["X-Amz-Target"][0], ".")[1]
    // nmetrics.TableName = tm.reqbody["TableName"].(string)
    metrics.TableName = getTableName(metrics)
    // log.Printf("tablename: %v", metrics.TableName)
    metrics.Query = parseMapToString("", tm.reqbody)
    log.Printf("query: %v", metrics.Query)
    // time.Duration: default nanoseconds, converting to milliseconds
    metrics.ElapsedTime = elapsedTime["endTime"].Sub(elapsedTime["startTime"]).Milliseconds()  //unit: millisecond
    // metrics.Client = strings.Split(tm.reqclient,":")[0]
    metrics.Client = tm.reqclient
    metrics.ResponseCode = tm.ressc
    return nil
}
func getTableName(m *Metrics) string {
    var tableName string
    if tm.reqbody["TableName"] ==  nil {
        tableName = ""
    } else {
        tableName = tm.reqbody["TableName"].(string)
    }
    /*
    tableName := ""
    variantMethodList := []string{"BatchGetItem"}
    sort.Strings(variantMethodList)
    i := sort.SearchStrings(variantMethodList, m.Method)
    log.Printf("i: %v", i)
    if i < len(variantMethodList) && variantMethodList[i] == m.Method {
        tableName = ""
    } else {
        tableName = tm.reqbody["TableName"].(string)
    }
    */
    return tableName
}
func (tm *TotalMetric) setReqTotalMetric(req *http.Request) error {
	tm.reqheader = req.Header
    tm.reqclient = req.RemoteAddr //req.Host //req.RequestURI 
	requestBody := map[string]interface{}{}
	err := getReqBody(req, &requestBody)
	tm.reqbody = requestBody
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

func getReqBody(req *http.Request, unmarshalStruct interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("getReqBody error: %v\n", err)
		return err
	}
	req.Body.Close()
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return json.Unmarshal(body, unmarshalStruct)
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
	firehoseService := firehose.New(sess, aws.NewConfig().WithRegion(regionName))
	recordsInput := &firehose.PutRecordInput{}
	recordsInput = recordsInput.SetDeliveryStreamName(streamName)

	// metricMap for firehose to S3, not implemented
	mMap := tm.parseMetricMap(metricMap)
	// log.Printf("metricMap: %v\n", mMap)

	//tn := requestBody["TableName"].(string)
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

func parseMapToString(prefix string, m map[string]interface{}) string {
    if len(prefix) > 0 { // only add the . if this is not the first call.
        prefix = prefix + "."
    }
    // builder stores the results string, appended to it
    var builder string
    for mKey, mVal := range m {
        //builder += mKey + ":" + mVal + " "

        // update a local prefix for this map key / value combination
        pp := prefix + mKey

        switch typedVal := mVal.(type) {
        case string:
            builder += fmt.Sprintf("%s:%s, ", pp, typedVal)
        case float64:
            builder += fmt.Sprintf("%s.%-1.0f, ", pp, typedVal)
        case map[string]interface{}:
            // add all the values to the builder, you already know they are correct.
            builder += parseMapToString(pp, typedVal)
        }
    }

    // return the string that this call has built
    return builder
}

func showDebugLog(debug_mode bool, topic string){
    if debug_mode{
        log.Printf("%s:  %v", topic, content)
    }
}
