package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
	"sync"
)

type Handler struct {
	ctx       context.Context
	processor *CountersProcessor
}

func NewHandler(ctx context.Context) (*Handler, error) {
	processor, err := NewCountersProcessor(ctx)
	if err != nil {
		return nil, err
	}
	return &Handler{
		ctx:       ctx,
		processor: processor,
	}, nil
}

func (h *Handler) ReportBadRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(fmt.Sprintf("%v", err)))
	log.Warn().Err(err).Msgf("bad_request")
}

func (h *Handler) ReportServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(fmt.Sprintf("%v", err)))
	log.Error().Err(err).Msgf("server_error")
}

func (h *Handler) ReportSuccess(w http.ResponseWriter, bytes []byte) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}

func (h *Handler) ReportData(w http.ResponseWriter, obj interface{}) {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		h.ReportServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	h.ReportSuccess(w, []byte("OK"))
}

func (h *Handler) HandleAddClick(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	if key == "" {
		h.ReportBadRequest(w, fmt.Errorf("key parameter is mandatory"))
		return
	}
	updatedClicks, err := h.processor.AddClick(key)
	if err != nil {
		h.ReportServerError(w, err)
		return
	}
	h.ReportData(w, updatedClicks)
}

func (h *Handler) HandleAddView(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	if key == "" {
		h.ReportBadRequest(w, fmt.Errorf("key parameter is mandatory"))
		return
	}
	updatedViews, err := h.processor.AddView(key)
	if err != nil {
		h.ReportServerError(w, err)
		return
	}
	h.ReportData(w, updatedViews)
}

func (h *Handler) HandleSetClicks(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	if key == "" {
		h.ReportBadRequest(w, fmt.Errorf("key parameter is mandatory"))
		return
	}
	clicksStr := req.URL.Query().Get("clicks")
	if key == "" {
		h.ReportBadRequest(w, fmt.Errorf("clicks parameter is mandatory"))
		return
	}
	clicks, err := strconv.ParseInt(clicksStr, 10, 64)
	if err != nil {
		h.ReportBadRequest(w, err)
		return
	}
	updatedClicks, err := h.processor.SetViews(key, clicks)
	if err != nil {
		h.ReportServerError(w, err)
		return
	}
	h.ReportData(w, updatedClicks)
}

func (h *Handler) HandleSetViews(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	if key == "" {
		h.ReportBadRequest(w, fmt.Errorf("key parameter is mandatory"))
		return
	}
	viewsStr := req.URL.Query().Get("views")
	if key == "" {
		h.ReportBadRequest(w, fmt.Errorf("clicks parameter is mandatory"))
		return
	}
	views, err := strconv.ParseInt(viewsStr, 10, 64)
	if err != nil {
		h.ReportBadRequest(w, err)
		return
	}
	updatedViews, err := h.processor.SetViews(key, views)
	if err != nil {
		h.ReportServerError(w, err)
		return
	}
	h.ReportData(w, updatedViews)
}

func (h *Handler) HandleGetCTR(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	if key == "" {
		h.ReportBadRequest(w, fmt.Errorf("key parameter is mandatory"))
		return
	}
	ctr, err := h.processor.GetCTR(key)
	if err != nil {
		h.ReportServerError(w, err)
		return
	}
	h.ReportData(w, ctr)
}

type SampleResults struct {
	SampledKay    string             `json:"sampled_key"`
	SampledScore  float64            `json:"sampled_score"`
	SampledScores map[string]float64 `json:"sampled_values"`
}

func (h *Handler) HandleSample(w http.ResponseWriter, req *http.Request) {
	keys := req.URL.Query()["key"]
	sampledScores := make([]float64, len(keys))
	errors := make([]error, len(keys))
	wg := sync.WaitGroup{}
	for idx := range keys {
		idx := idx
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctr, err := h.processor.GetCTR(keys[idx])
			if err != nil {
				errors[idx] = err
				return
			}
			sampledScores[idx] = ctr.Sample()
		}()
	}
	wg.Wait()
	for _, err := range errors {
		if err != nil {
			h.ReportServerError(w, err)
			return
		}
	}
	sampleResults := &SampleResults{
		SampledScore:  0.,
		SampledScores: map[string]float64{},
	}
	for idx := range keys {
		sampleResults.SampledScores[keys[idx]] = sampledScores[idx]
		if sampledScores[idx] > sampleResults.SampledScore {
			sampleResults.SampledScore = sampledScores[idx]
			sampleResults.SampledKay = keys[idx]
		}
	}
	h.ReportData(w, sampleResults)
}

func (h *Handler) HandleReportBetaDistributionParams(w http.ResponseWriter, req *http.Request) {
	keys := req.URL.Query()["key"]
	distributionParams := make([]*BetaDistributionParams, len(keys))
	errors := make([]error, len(keys))
	wg := sync.WaitGroup{}
	for idx := range keys {
		idx := idx
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctr, err := h.processor.GetCTR(keys[idx])
			if err != nil {
				errors[idx] = err
				return
			}
			distributionParams[idx] = ctr.PopulateBetaDistributionParams()
		}()
	}
	wg.Wait()
	for _, err := range errors {
		if err != nil {
			h.ReportServerError(w, err)
			return
		}
	}
	h.ReportData(w, distributionParams)
}
