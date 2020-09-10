# ref for access policy: https://docs.aws.amazon.com/cdk/latest/guide/permissions.html
# ref for ipaddress of access policy: https://translate.google.com/translate?hl=en&sl=ja&u=https://dev.classmethod.jp/articles/cdk-elasticsearch-deployment/&prev=search&pto=aue 
from aws_cdk import (
    core,
    aws_elasticsearch as aes,
    aws_iam as iam,
)
import os
import urllib.request
from helper.constants import constants

class ElasticStack(core.Stack):
    def __init__(
        self,
        scope: core.Construct,
        id: str,
        **kwargs,
    ) -> None:
        super().__init__(scope, id, **kwargs)
        # elastic policy
        elastic_policy = iam.PolicyStatement(
        effect=iam.Effect.ALLOW, actions=["es:*",], resources=["*"],conditions={"IpAddress":{'aws:SourceIp': constants["ES_CLIENT_IP"]}}
        #effect=iam.Effect.ALLOW, actions=["es:*",], resources=["*"],conditions={"IpAddress":{'aws:SourceIp':"127.0.0.1" }}
        )
        elastic_policy.add_any_principal()
        elastic_document = iam.PolicyDocument()
        elastic_document.add_statements(elastic_policy)
        # cluster config
        cluster_config = {
            "instanceCount": constants["ELASTIC_INSTANCE_COUNT"],
            "instanceType": constants["ELASTIC_INSTANCE_TYPE"],
            "zoneAwarenessEnabled": False,
            #"zoneAwarenessConfig": {"availabilityZoneCount": 1},
        }
        
        # create the elastic cluster
        self.elastic_domain = aes.CfnDomain(
            self,
            "elastic_domain",
            domain_name=constants["ELASTIC_NAME"],
            elasticsearch_cluster_config=cluster_config,
            elasticsearch_version=constants["ELASTIC_VERSION"],
            ebs_options={"ebsEnabled": True, "volumeSize": 10},
            access_policies=elastic_document,
            #log_publishing_options={"enabled": True},
            #cognito_options={"enabled": True},
        )
        #core.Tag.add(self.elastic_domain, "project", constants["PROJECT_TAG"])
        core.Tags.of(self.elastic_domain).add("project", constants["PROJECT_TAG"])
        core.CfnOutput(
            self,
            'DomainArn',
            export_name = "esarn",
            value = self.elastic_domain.attr_arn,
            description = "elasticsearch domain arn",
        )
