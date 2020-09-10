#!/usr/bin/env python3
### ref: https://github.com/aws-samples/aws-cdk-managed-elkk/blob/master/elastic/elastic_stack.py
import os
from aws_cdk import core
#import cdk classes
from es.esstack import ElasticStack
from iam.iamstack import IamStack
from firehose.firehosestack import FirehoseStack

app = core.App()

# Elastic stack
elastic_stack = ElasticStack(
    app,
    "ddbqi-es",
    env=core.Environment(
        account=os.environ["CDK_DEFAULT_ACCOUNT"],
        region=os.environ["CDK_DEFAULT_REGION"],
    ),
)

iam_stack = IamStack(
    app,
    "ddbqi-iam",
    env=core.Environment(
        account=os.environ["CDK_DEFAULT_ACCOUNT"],
        region=os.environ["CDK_DEFAULT_REGION"],
    ),
)

datastream_stack = FirehoseStack(
    app,
    "ddbqi-firehose",
    #core.Fn.import_value("rolearn"),
    env=core.Environment(
        account=os.environ["CDK_DEFAULT_ACCOUNT"],
        region=os.environ["CDK_DEFAULT_REGION"],
    ),
)

datastream_stack.add_dependency(elastic_stack)
datastream_stack.add_dependency(iam_stack)
app.synth()
