package main

import (
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
	"time"
)

const (
	counterKeyAttribute   = "k"
	counterValueAttribute = "v"
)

type Counter struct {
	Key   string `dynamodbav:"k"`
	Value int64  `dynamodbav:"v"`
}

type BetaDistributionParams struct {
	Alpha float64 `json:"alpha"`
	Beta  float64 `json:"beta"`
}

type CTR struct {
	Key    string `json:"key"`
	Views  int64  `json:"views"`
	Clicks int64  `json:"clicks"`
}

func (ctr *CTR) Mean() float64 {
	views := ctr.Views
	clicks := ctr.Clicks
	if clicks > views {
		clicks = views
	}
	if clicks == 0 {
		return 0.
	}
	return float64(clicks) / float64(views)
}

func (ctr *CTR) PopulateBetaDistributionParams() *BetaDistributionParams {
	views := float64(ctr.Views)
	clicks := float64(ctr.Clicks)
	if clicks > views {
		clicks = views
	}
	return &BetaDistributionParams{
		Alpha: clicks + 1,
		Beta:  views - clicks + 1,
	}
}

func (ctr *CTR) Sample() float64 {
	betaDistributionParams := ctr.PopulateBetaDistributionParams()
	seed := time.Now().UnixNano()
	source := rand.NewSource(uint64(seed))
	return distuv.Beta{
		Alpha: betaDistributionParams.Alpha,
		Beta:  betaDistributionParams.Beta,
		Src:   source,
	}.Rand()
}
