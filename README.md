# DDB_Query_Insight
- Date: 2020.09.01
- Yongki Kim(kyongki@)

![ddb_query_insight](images/ddb_query_insight.png)

## Objectives
DDB Query Insight is aimed to show Dynamodb user more insight providing with the chart of historical DDB query count, failed query and elapsed time. I developed http reverse proxy to gather http header and body, and push them to kinesis firehose, then firehose sends those data to ElasticSearch to store and visualize the metrics. It will help DDB user understand current and historical DDB query processing status.

### Providing Features
Using the DynamoDB Metric dashboard in AWS console, you can see various metrics, such as read/write requests, throttled read/write events, and GET/PUT/SCAN/Query latency. But sometimes DDB user would like to figure out more specific metrics. Below are the typical metrics which DDB Query Insight can show.
- historical success/failed count
- failed or slow DDB query
- accessed client and table name
- anything which you want to know with http head/body information

## Architecture
    ![ddb_query_insight_architecture](images/architecture.png)

## DDB Reverse Proxy(DDB rproxy)
DDB Reverse Proxy(DDB rproxy) gathers http request header and body information, and response header information and it sends them to firehose. Also you can parse the DDB rproxy log data and utilize it by yourself without sending to firehose. I tried to make variables be adjustable as parameters. Therefore, you can input your own variables, not compiling the source code. As well, I created simple script(ddb_rproxy.sh) to start and stop easily.
### How to use DDB rproxy
#### run DDB rproxy

``` shell
$ ddb_rproxy.sh start
$ ps aux | grep ddb_rproxy # check whether ddb_rproxy is listed
$ netstat -tnl             # check whether ddb_rproxy port is opened
```

If you want to change the parameter, edit *DAEMON_OPTS* variable in the *ddb_rproxy.sh*. Available parameters are show with *help* parameter.

``` shell
$ ./ddb_rproxy.sh help

arguments: --port --debug --region_name --stream_name
--port: indicate listen  port number, default:8000
--debug: show detailed logs, default: false
--region_name: aws region name
--send_to_firehose: sending dynamo metrics to kinesis firehose, default: true
--stream_name: kinesis firehose stream name
...
DAEMON_OPTS example:
1) ddbrproxy
2) ddbrproxy --port=8000 --debug=true --region_name=ap-northeast-2 --stream_name=ddbhose --send_to_firehose=false
```

All gathered information are stored log file in *logs* directory, so you can process it on your taste. below is the sample log data.

``` shell
$ cat ddbrproxy.2020-08-31-28.log
2020/08/31 15:27:28 DDB rproxy is listening on  8000
2020/08/31 15:28:59 raw metrics: {map[Accept-Encoding:[identity] Authorization:[AWS4-HMAC-SHA256 Credential=/ap-northeast-2/dynamodb/aws4_request, SignedHeaders=accept-encoding;content-length;content-type;host;x-amz-date;x-amz-security-token;x-amz-target, Signature=xxxxxx] Content-Length:[74] Content-Type:[application/x-amz-json-1.0] User-Agent:[aws-sdk-go/1.34.6 (go1.13.4; linux; amd64)] X-Amz-Date:[20200831T152859Z] X-Amz-Security-Token:[==] X-Amz-Target:[DynamoDB_20120810.GetItem]] map[Key:map[title:map[S:Samsara] year:map[N:2011]] TableName:Movies] {"Key":{"title":{"S":"Samsara"},"year":{"N":"2011"}},"TableName":"Movies"} map[Content-Length:[338] Content-Type:[application/x-amz-json-1.0] Date:[Mon, 31 Aug 2020 15:28:58 GMT] X-Amz-Crc32:[3554984406] X-Amzn-Requestid:[1BP4MHBVO2ALEAIEHURDNQ8C6BVV4KQNSO5AEMVJFAJG]] map[] 200 172.31.38.36:47240}
2020/08/31 15:28:59 Metrics::  {
	"Date": "2020-08-31T15:28:59.143491725Z",
	"Method": "GetItem",
	"TableName": "Movies",
	"Query": "{\"Key\":{\"title\":{\"S\":\"Samsara\"},\"year\":{\"N\":\"2011\"}},\"TableName\":\"Movies\"}",
	"ElapsedTime": 12,
	"Client": "172.31.38.36",
	"ResponseCode": 200
}
```

## Integrating with Kinesis Firehose and ElasticSearch
In order to monitor DDB transaction, analyzing log data is not enough. ElasticSearch and Kinesis firehose is good tool for ingesting and visualizing the data. With those two tools, you can make chars and dashboard easily on your purpose.
### Creating ElasticSearch
### Creating Firehose
### Making DDB rproxy connect to firehose
## Test
### uploading DDB sample data
## Performance
## Conclusion
my customer asked me how to find out which query failed frequently and how to figure it out, so I developed it, even though I am not professional programmer and not ElasticSearch expert. And this program is not evaluated in production environment, so please test it in your test environment before adopting it in production. Any feedback, question, and experience is welcome, I just hope that it is helpful to you.
## References
- golang proxy and metric: https://www.sidneyw.com/go-reverse-proxy/ (authentication failed)
- kinesis firehose go example: https://gist.github.com/coboshm/1c89bcc7bf2c9f9694e4984051474951
- reverse proxy: https://developer20.com/writing-proxy-in-go/?utm_source=reddit&utm_medium=link&utm_campaign=proxy-in-go
- args in golang: https://gobyexample.com/command-line-flags
- make daemon script: https://gist.github.com/alobato/1968852
- api gw in lambda: https://github.com/apex/gateway
- aws dynamodb golang example: https://github.com/awsdocs/aws-doc-sdk-examples/tree/master/go/example_code/dynamodb
- aws golang sdk: https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/#DynamoDB
- ecs replacing lambda: https://www.gravitywell.co.uk/insights/using-ecs-tasks-on-aws-fargate-to-replace-lambda-functions/
- envoy for dynamodb filter: https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/dynamodb_filter#config-http-filters-dynamo
- map string interface: https://bitfieldconsulting.com/golang/map-string-interface
