// snippet-comment:[These are tags for the AWS doc team's sample catalog. Do not remove.]
// snippet-sourceauthor:[Doug-AWS]
// snippet-sourcedescription:[DynamoDBUpdateItem.go updates an item in an Amazon DynamoDB table.]
// snippet-keyword:[Amazon DynamoDB]
// snippet-keyword:[UpdateItem function]
// snippet-keyword:[Go]
// snippet-sourcesyntax:[go]
// snippet-service:[dynamodb]
// snippet-keyword:[Code Sample]
// snippet-sourcetype:[full-example]
// snippet-sourcedate:[2019-03-19]
/*
   Copyright 2010-2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
   This file is licensed under the Apache License, Version 2.0 (the "License").
   You may not use this file except in compliance with the License. A copy of
   the License is located at
    http://aws.amazon.com/apache2.0/
   This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
   CONDITIONS OF ANY KIND, either express or implied. See the License for the
   specific language governing permissions and limitations under the License.
*/
// snippet-start:[dynamodb.go.update_item]
package main

// snippet-start:[dynamodb.go.update_item.imports]
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/dynamodb"

    "fmt"
)
// snippet-end:[dynamodb.go.update_item.imports]


const DDB_ENDPOINT = "http://127.0.0.1:8000"

func main() {
    // Initialize a session in us-west-2 that the SDK will use to load
    // credentials from the shared credentials file ~/.aws/credentials.
    sess, err := session.NewSession(&aws.Config{
        Region: aws.String("ap-northeast-2")},
    )

    // Create DynamoDB client
    // svc := dynamodb.New(sess)
    svc := dynamodb.New(sess, aws.NewConfig().WithEndpoint(DDB_ENDPOINT))

    // snippet-start:[dynamodb.go.update_item.call]
    // Update item in table Movies
    tableName := "Movies2"
    movieName := "The Big New Movie"
    movieYear := "2015"
    movieRating := "0.5"

    input := &dynamodb.UpdateItemInput{
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":r": {
                N: aws.String(movieRating),
            },
        },
        TableName: aws.String(tableName),
        Key: map[string]*dynamodb.AttributeValue{
            "Year": {
                N: aws.String(movieYear),
            },
            "Title": {
                S: aws.String(movieName),
            },
        },
        ReturnValues:     aws.String("UPDATED_NEW"),
        UpdateExpression: aws.String("set Rating = :r"),
    }

    itemresult, err := svc.UpdateItem(input)
    if err != nil {
        fmt.Println(err.Error())
        return
    }

    fmt.Println(itemresult)

    fmt.Println("Successfully updated '" + movieName + "' (" + movieYear + ") rating to " + movieRating)
    // snippet-end:[dynamodb.go.update_item.call]
}
// snippet-end:[dynamodb.go.update_item]
