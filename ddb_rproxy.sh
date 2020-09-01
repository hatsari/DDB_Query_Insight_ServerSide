#!/bin/bash
## ddb_rproxy damon script
set -e

# Must be a valid filename
NAME=ddbrproxy
now=$(date +"%Y-%m-%d-%H%M%S")
if [ ! -d logs ]; then
    mkdir logs
fi
PIDFILE=logs/$NAME.pid
LOGFILE=logs/$NAME.$now.log
#This is the command to be run, give the full pathname
PROG=ddb_rproxy
DAEMON_OPTS="--port=8000 --debug=true"
### DAEMON_OPTS="--port=8000 --debug=true --region_name=ap-northeast-2 --stream_name=ddbhose --send_to_firehose=false"
RETVAL=0

export PATH="${PATH:+$PATH:}/usr/sbin:/sbin:./"

start() {
        echo -n "Starting daemon: "$NAME
	$PROG $DAEMON_OPTS 1> "$LOGFILE" 2>&1 &
	echo $! > "$PIDFILE"
	echo "$PROG started with options "$DAEMON_OPTS
        echo "."
}

stop() {
    echo -n "Stopping daemon: "$NAME
    kill -9 $(cat $PIDFILE)
    echo "."
}

case "$1" in
    start)
	start
	;;
  stop)
      stop
	;;
  restart)
        echo -n "Restarting daemon: "$NAME
	stop
	sleep 2
	start
	echo "."
	;;
  help)
      echo "arguments: --port --debug --region_name --stream_name"
      echo "--port: indicate listen  port number, default:8000"
      echo "--debug: show detailed logs, default: false"
      echo "--region_name: aws region name"
      echo "--send_to_firehose: sending dynamo metrics to kinesis firehose, default: true"
      echo "--stream_name: kinesis firehose stream name"
      echo "..."
      echo "DAEMON_OPTS example:"
      echo "1) --port=8000 --debug=true"
      echo "2) --port=8000 --debug=true --region_name=ap-northeast-2 --stream_name=ddbhose --send_to_firehose=false"
      ;;

  *)
	echo "Usage: "$1" {start|stop|restart|help}"
	exit 1
esac

exit $RETVAL
