# ref for access policy: https://docs.aws.amazon.com/cdk/latest/guide/permissions.html
# ref for ipaddress of access policy: https://translate.google.com/translate?hl=en&sl=ja&u=https://dev.classmethod.jp/articles/cdk-elasticsearch-deployment/&prev=search&pto=aue 
# ref for cdk sdk for python: https://docs.aws.amazon.com/cdk/api/latest/python/aws_cdk.aws_elasticsearch/CfnDomain.html
from aws_cdk import (
    core,
    aws_kinesisfirehose as afh, 
    aws_iam as iam,
    aws_s3 as s3,
)
import os
import urllib.request
from helper.constants import constants

class FirehoseStack(core.Stack):
    def __init__(self, scope: core.Construct, id:str, **kwargs) -> None:
        super().__init__(scope, id, **kwargs)

        ### Code for FirehoseStack

        # get role arn value from ddbqiIam stack
        print("++++++++++++++++++++++++++++++++++++")
        #rolearn = core.Token.toString(core.Fn.import_value("rolearn"))
        rolearn = core.Fn.import_value("rolearn")
        esarn = core.Fn.import_value("esarn")
        print("++++++++++++++++++++++++++++++++++++")

        # creating s3 bucket for failed logs
        log_s3 = s3.Bucket(self, constants["S3_BUCKET_NAME"])
        s3_config = {
            "bucketArn" : log_s3.bucket_arn,
            #"roleArn": firehose_role.role_arn
            "roleArn": rolearn
        }
        es_dest_config = {
            "domainArn" : esarn,
            "indexName" : constants["DDBES_INDEX_NAME"], 
            "roleArn": rolearn,
            "s3Configuration" : s3_config,
            "bufferingHints": {"intervalInSeconds": 60, "sizeInMBs":1},
        }
        self.firehose_deliverySystem = afh.CfnDeliveryStream(
            self,
            "ddbqiStream",
            delivery_stream_name = constants["FH_DELIVERY_STREAM_NAME"],
            delivery_stream_type = "DirectPut",
            elasticsearch_destination_configuration = es_dest_config
        )
        core.Tags.of(self.firehose_deliverySystem).add("project", constants["PROJECT_TAG"])

        core.CfnOutput(
            self,
            'StreamName',
            export_name = "streamName",
            value = constants["FH_DELIVERY_STREAM_NAME"],
            description = "firehose stream name",
        )
