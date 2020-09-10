# ref for access policy: https://docs.aws.amazon.com/cdk/latest/guide/permissions.html
# ref for ipaddress of access policy: https://translate.google.com/translate?hl=en&sl=ja&u=https://dev.classmethod.jp/articles/cdk-elasticsearch-deployment/&prev=search&pto=aue 
# ref for cdk sdk for python: https://docs.aws.amazon.com/cdk/api/latest/python/aws_cdk.aws_elasticsearch/CfnDomain.html
from aws_cdk import (
    core,
    aws_iam as iam,
)
import os
from helper.constants import constants


class IamStack(core.Stack):
    def __init__(self, scope: core.Construct, id:str, **kwargs) -> None:
        super().__init__(scope, id, **kwargs)

        # IAM policy for firehose
        es_policy_statement = iam.PolicyStatement(
            effect=iam.Effect.ALLOW,
            actions=[
                "es:DescribeElasticsearchDomain",
                "es:DescribeElasticsearchDomains",
                "es:DescribeElasticsearchDomainConfig",
                "es:ESHttpPost",
                "es:ESHttpPut",
            ],
            resources=[
                "*",
            ],
        )
        es_policy_doc = iam.PolicyDocument(
            statements = [es_policy_statement] 
        )

        s3_policy_statement = iam.PolicyStatement(
            effect=iam.Effect.ALLOW,
            actions=["s3:*"],
            resources=["*"]
        )
        s3_policy_doc = iam.PolicyDocument(
            statements = [s3_policy_statement]
        )
        # creating Role
        self.firehose_role = iam.Role(
            self,
            id = "ddbqiRole",
            role_name = constants["FIREHOSE_ROLE"],
            assumed_by = iam.ServicePrincipal("firehose.amazonaws.com"),
        )
        # IAM policy
        firehose_policy = iam.Policy (
            self,
            "firehose_policy",
            policy_name = "ddbqiFhPolicy",
            statements = [es_policy_statement, s3_policy_statement],
            roles = [self.firehose_role]
        )

        # attach policy to role
        #firehose_role.add_to_policy(es_policy_statement)
        #firehose_role.add_to_policy(s3_policy_statement)
        #firehose_policy.attach_to_role(firehose_role)
        core.CfnOutput(
            self,
            'RoleArn',
            export_name = "rolearn",
            value = self.firehose_role.role_arn,
            description = "firehose role arn"
        )

        core.Tags.of(self.firehose_role).add("project", constants["PROJECT_TAG"])


