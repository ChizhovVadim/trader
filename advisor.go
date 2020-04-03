package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const jsonDateTimeLayout = "2006-01-02T15:04:05"

type JsonTime struct {
	time.Time
}

func (t JsonTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(jsonDateTimeLayout)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, jsonDateTimeLayout)
	b = append(b, '"')
	return b, nil
}

func (m *JsonTime) UnmarshalJSON(p []byte) error {
	var s = strings.Replace(string(p), "\"", "", -1)
	t, err := time.ParseInLocation(jsonDateTimeLayout, s, Moscow)
	m.Time = t
	return err
}

type AdviceModel struct {
	SecurityCode string
	DateTime     JsonTime
	Price        float64
	Position     float64
}

type CandleModel struct {
	SecurityCode string
	DateTime     JsonTime
	ClosePrice   float64
	Volume       float64
}

type Advisor struct {
	logger     *log.Logger
	url        string
	httpClient *http.Client
}

func (adv *Advisor) GetSecurities() ([]string, error) {
	var url = adv.url + "/api/advisors"
	resp, err := adv.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %v", resp.Status)
	}
	var result []string
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (adv *Advisor) GetAdvices(ctx context.Context, security string, advices chan<- Advice) error {
	const Timeout = 90 // must < http.client.Timeout
	var since time.Time
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var advice, err = adv.GetAdvice(ctx, security, since, Timeout)
		if err != nil {
			adv.logger.Println("GetAdvices error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Minute):
				continue
			}
		}
		if advice.DateTime.After(since) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case advices <- advice:
				since = advice.DateTime
			}
		}
	}
}

func (adv *Advisor) PublishCandles(ctx context.Context, candles <-chan Candle) error {
	var candlesToSend []Candle
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case candle := <-candles:
			candlesToSend = append(candlesToSend, candle)
			if time.Since(candle.DateTime) < 9*time.Minute {
				var err = adv.PostCandles(candlesToSend)
				if err != nil {
					adv.logger.Println("PublishCandles error", err)
					continue
				}
				candlesToSend = candlesToSend[:0]
			}
		}
	}
}

func (adv *Advisor) GetAdvice(ctx context.Context,
	securityCode string, since time.Time, timeout int) (Advice, error) {
	baseUrl, err := url.Parse(adv.url + "/api/advisors/" + securityCode)
	if err != nil {
		return Advice{}, err
	}

	var values = make(url.Values)
	values.Set("since", since.Format(jsonDateTimeLayout))
	values.Set("timeout", strconv.Itoa(timeout))

	baseUrl.RawQuery = values.Encode()

	url := baseUrl.String()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Advice{}, err
	}
	req = req.WithContext(ctx)
	resp, err := adv.httpClient.Do(req)
	if err != nil {
		return Advice{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Advice{}, fmt.Errorf("http status %v", resp.Status)
	}
	var result AdviceModel
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return Advice{}, err
	}
	return Advice{
		SecurityCode: result.SecurityCode,
		DateTime:     result.DateTime.Time,
		Price:        result.Price,
		Position:     result.Position,
	}, nil
}

func (adv *Advisor) PostCandles(candles []Candle) error {
	var jsonCandles = make([]CandleModel, len(candles))
	for i := range candles {
		jsonCandles[i] = convertToJsonCandle(candles[i])
	}
	bb, err := json.Marshal(jsonCandles)
	if err != nil {
		return err
	}
	var url = adv.url + "/api/candles"
	resp, err := adv.httpClient.Post(url, "application/json", bytes.NewBuffer(bb))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %v", resp.Status)
	}
	return nil
}

func convertToJsonCandle(candle Candle) CandleModel {
	return CandleModel{
		SecurityCode: candle.SecurityCode,
		DateTime:     JsonTime{candle.DateTime},
		ClosePrice:   candle.ClosePrice,
		Volume:       candle.Volume,
	}
}
