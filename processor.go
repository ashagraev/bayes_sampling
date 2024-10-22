package main

import (
	"context"
	"fmt"
	"os"
)

const (
	countersTableNameEnvVariableName = "AWS_COUNTERS_TABLE"

	viewsKeySuffix  = "_views"
	clicksKeySuffix = "_clicks"
)

type CountersProcessor struct {
	dynamoDBWrapper *DynamoDBClientWrapper
}

func NewCountersProcessor(ctx context.Context) (*CountersProcessor, error) {
	if os.Getenv(countersTableNameEnvVariableName) == "" {
		return nil, fmt.Errorf("env variable %q is mandatory", countersTableNameEnvVariableName)
	}
	dynamoDBWrapper, err := NewDynamoDBClientWrapper(ctx, os.Getenv(countersTableNameEnvVariableName))
	if err != nil {
		return nil, err
	}
	return &CountersProcessor{
		dynamoDBWrapper: dynamoDBWrapper,
	}, nil
}

func toViewsKey(key string) string {
	return key + viewsKeySuffix
}

func toClicksKey(key string) string {
	return key + clicksKeySuffix
}

func (p *CountersProcessor) AddView(key string) (*Counter, error) {
	_, _ = p.dynamoDBWrapper.GetValue(toViewsKey(key))
	return p.dynamoDBWrapper.IncrementAndGetNewValue(toViewsKey(key))
}

func (p *CountersProcessor) AddClick(key string) (*Counter, error) {
	_, _ = p.dynamoDBWrapper.GetValue(toClicksKey(key))
	return p.dynamoDBWrapper.IncrementAndGetNewValue(toClicksKey(key))
}

func (p *CountersProcessor) SetViews(key string, views int64) (*Counter, error) {
	return p.dynamoDBWrapper.SetValue(toViewsKey(key), views)
}

func (p *CountersProcessor) SetClicks(key string, clicks int64) (*Counter, error) {
	return p.dynamoDBWrapper.SetValue(toClicksKey(key), clicks)
}

func (p *CountersProcessor) GetCTR(key string) (*CTR, error) {
	views, err := p.dynamoDBWrapper.GetValue(toViewsKey(key))
	if err != nil {
		return nil, err
	}
	clicks, err := p.dynamoDBWrapper.GetValue(toClicksKey(key))
	if err != nil {
		return nil, err
	}
	return &CTR{
		Key:    key,
		Views:  views.Value,
		Clicks: clicks.Value,
	}, nil
}

func (p *CountersProcessor) Sample(key string) (float64, error) {
	ctr, err := p.GetCTR(key)
	if err != nil {
		return 0, err
	}
	return ctr.Sample(), nil
}

func (p *CountersProcessor) PopulateBetaDistributionParams(key string) (*BetaDistributionParams, error) {
	ctr, err := p.GetCTR(key)
	if err != nil {
		return nil, err
	}
	return ctr.PopulateBetaDistributionParams(), nil
}
