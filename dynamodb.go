package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"strconv"
)

type DynamoDBClientWrapper struct {
	ctx               context.Context
	client            *dynamodb.Client
	countersTableName string
}

func NewDynamoDBClientWrapper(ctx context.Context, countersTableName string) (*DynamoDBClientWrapper, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &DynamoDBClientWrapper{
		ctx:               ctx,
		client:            dynamodb.NewFromConfig(cfg),
		countersTableName: countersTableName,
	}, nil
}

func (c *DynamoDBClientWrapper) SetValue(key string, value int64) (*Counter, error) {
	counter := &Counter{
		Key:   key,
		Value: value,
	}
	item, err := attributevalue.MarshalMap(counter)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal counter: %v", err)
	}
	_, err = c.client.PutItem(c.ctx, &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(c.countersTableName),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot put initial counter value for key %q: %v", key, err)
	}
	return counter, nil
}

func (c *DynamoDBClientWrapper) GetValue(key string) (*Counter, error) {
	res, err := c.client.GetItem(c.ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			counterKeyAttribute: &types.AttributeValueMemberS{
				Value: key,
			},
		},
		TableName: aws.String(c.countersTableName),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get counter value for key %q: %v", key, err)
	}
	if res.Item == nil {
		counter, err := c.SetValue(key, 0)
		if err != nil {
			return nil, fmt.Errorf("cannot init counter for key %q: %v", key, err)
		}
		return counter, nil
	}
	counter := &Counter{}
	if err := attributevalue.UnmarshalMap(res.Item, counter); err != nil {
		return nil, fmt.Errorf("cannot unmarshal counter data for key %q: %v", key, err)
	}
	return counter, nil
}

func (c *DynamoDBClientWrapper) IncrementAndGetNewValue(key string) (*Counter, error) {
	res, err := c.client.UpdateItem(c.ctx, &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			counterKeyAttribute: &types.AttributeValueMemberS{
				Value: key,
			},
		},
		TableName: aws.String(c.countersTableName),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":incr": &types.AttributeValueMemberN{
				Value: "1",
			},
		},
		ReturnValues:     types.ReturnValueUpdatedNew,
		UpdateExpression: aws.String(fmt.Sprintf("SET %s = %s + :incr", counterValueAttribute, counterValueAttribute)),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot update counter value for key %q: %v", key, err)
	}
	value, ok := res.Attributes[counterValueAttribute]
	if !ok {
		return nil, fmt.Errorf("cannot found the %s attribute in response", counterValueAttribute)
	}
	stringValue := value.(*types.AttributeValueMemberN).Value
	intValue, err := strconv.ParseInt(stringValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %s attribute %q as integer: %v", counterValueAttribute, stringValue, err)
	}
	return &Counter{
		Key:   key,
		Value: intValue,
	}, nil
}
