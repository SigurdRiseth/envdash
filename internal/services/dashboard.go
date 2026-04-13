package services

import (
	"context"
	"log"
	"sync"

	"envdash/internal/clients"
	"envdash/internal/firebase"
	"envdash/internal/models"
	"envdash/internal/webhook"
)

// DashboardService retrieves and assembles populated dashboard data.
type DashboardService interface {
	Get(ctx context.Context, id string) (*models.DashboardResponse, error)
}

type dashboardService struct {
	regs      firebase.RegistrationRepository
	notifs    firebase.NotificationRepository
	countries *clients.CountriesClient
	meteo     *clients.MeteoClient
	openaq    *clients.OpenAQClient
	currency  *clients.CurrencyClient
	dispatcher *webhook.Dispatcher
}

// NewDashboardService constructs a DashboardService.
func NewDashboardService(
	regs firebase.RegistrationRepository,
	notifs firebase.NotificationRepository,
	countries *clients.CountriesClient,
	meteo *clients.MeteoClient,
	openaq *clients.OpenAQClient,
	currency *clients.CurrencyClient,
	dispatcher *webhook.Dispatcher,
) DashboardService {
	return &dashboardService{
		regs:      regs,
		notifs:    notifs,
		countries: countries,
		meteo:     meteo,
		openaq:    openaq,
		currency:  currency,
		dispatcher: dispatcher,
	}
}

// Get assembles a populated dashboard for the given registration ID.
// External API calls that are independent are made concurrently.
func (s *dashboardService) Get(ctx context.Context, id string) (*models.DashboardResponse, error) {
	reg, err := s.regs.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	f := reg.Features
	resp := &models.DashboardResponse{
		Country:       reg.Country,
		ISOCode:       reg.ISOCode,
		LastRetrieval: timestamp(),
	}

	// Always fetch country data — it provides capital, coordinates, population,
	// area, and base currency in a single request.
	countryData, err := s.countries.GetByISO(ctx, reg.ISOCode)
	if err != nil {
		log.Printf("dashboard %s: countries API error: %v", id, err)
	}

	// Fill country-sourced fields
	if countryData != nil {
		if resp.Country == "" {
			resp.Country = countryData.Name
		}
		if f.Capital {
			resp.Features.Capital = &countryData.Capital
		}
		if f.Coordinates {
			resp.Features.Coordinates = &models.CoordinatesData{
				Latitude:  countryData.Latitude,
				Longitude: countryData.Longitude,
			}
		}
		if f.Population {
			resp.Features.Population = &countryData.Population
		}
		if f.Area {
			resp.Features.Area = &countryData.Area
		}
	}

	// Determine coordinates for weather/air quality (from country data or fallback).
	var lat, lon float64
	if countryData != nil {
		lat, lon = countryData.Latitude, countryData.Longitude
	}

	// Fan out independent API calls concurrently.
	var wg sync.WaitGroup
	var (
		meteoData    *clients.MeteoData
		openaqData   *clients.OpenAQData
		currencyRates map[string]float64
		meteoErr     error
		openaqErr    error
		currencyErr  error
	)

	needsMeteo    := f.Temperature || f.Precipitation
	needsOpenAQ   := f.AirQuality
	needsCurrency := len(f.TargetCurrencies) > 0 && countryData != nil && countryData.BaseCurrency != ""

	if needsMeteo && lat != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			meteoData, meteoErr = s.meteo.GetForecast(ctx, lat, lon)
		}()
	}
	if needsOpenAQ && lat != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			openaqData, openaqErr = s.openaq.GetAirQuality(ctx, lat, lon)
		}()
	}
	if needsCurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			currencyRates, currencyErr = s.currency.GetRates(ctx, countryData.BaseCurrency, f.TargetCurrencies)
		}()
	}

	wg.Wait()

	// Populate weather fields
	if needsMeteo {
		if meteoErr != nil {
			log.Printf("dashboard %s: meteo API error: %v", id, meteoErr)
		} else if meteoData != nil {
			if f.Temperature {
				resp.Features.Temperature = &meteoData.Temperature
			}
			if f.Precipitation {
				resp.Features.Precipitation = &meteoData.Precipitation
			}
		}
	}

	// Populate air quality
	if needsOpenAQ {
		if openaqErr != nil {
			log.Printf("dashboard %s: openaq API error: %v", id, openaqErr)
		} else if openaqData != nil {
			resp.Features.AirQuality = &models.AirQualityData{
				PM25:  openaqData.PM25,
				PM10:  openaqData.PM10,
				Level: openaqData.Level,
			}
		}
	}

	// Populate currency
	if needsCurrency {
		if currencyErr != nil {
			log.Printf("dashboard %s: currency API error: %v", id, currencyErr)
		} else {
			resp.Features.TargetCurrencies = currencyRates
		}
	}

	// Fire lifecycle webhooks (INVOKE) — non-blocking
	s.fireInvoke(ctx, reg.ISOCode)

	// Fire threshold webhooks — non-blocking
	s.fireThresholds(ctx, reg, resp)

	return resp, nil
}

// fireInvoke dispatches INVOKE webhook notifications for the given country.
// It looks up all webhooks registered for the INVOKE event that match isoCode
// (including wildcard registrations with an empty country) and POSTs to each
// URL asynchronously. Errors from the repository are silently ignored so that
// a failing Firestore read never prevents a dashboard response from being sent.
func (s *dashboardService) fireInvoke(ctx context.Context, isoCode string) {
	notifs, err := s.notifs.ListMatching(ctx, isoCode, models.EventInvoke)
	if err != nil {
		return
	}
	ts := timestamp()
	for _, n := range notifs {
		s.dispatcher.Dispatch(models.WebhookPayload{
			ID:      n.ID,
			Country: isoCode,
			Event:   models.EventInvoke,
			Time:    ts,
		}, n.URL)
	}
}

// fireThresholds evaluates all THRESHOLD webhooks for the given registration
// against the live values in resp. For each webhook whose condition is met
// (measuredValue <operator> threshold.Value), a payload is dispatched
// asynchronously. Fields that are disabled in the registration or that failed
// to populate (nil in resp) are skipped — the webhook will never fire for them.
func (s *dashboardService) fireThresholds(ctx context.Context, reg *models.Registration, resp *models.DashboardResponse) {
	notifs, err := s.notifs.ListMatching(ctx, reg.ISOCode, models.EventThreshold)
	if err != nil {
		return
	}
	if len(notifs) == 0 {
		return
	}

	// Build a map of field → measured value from the response
	measured := map[string]float64{}
	if resp.Features.Temperature != nil {
		measured["temperature"] = *resp.Features.Temperature
	}
	if resp.Features.Precipitation != nil {
		measured["precipitation"] = *resp.Features.Precipitation
	}
	if resp.Features.AirQuality != nil {
		measured["pm25"] = resp.Features.AirQuality.PM25
		measured["pm10"] = resp.Features.AirQuality.PM10
	}

	ts := timestamp()
	for _, n := range notifs {
		if n.Threshold == nil {
			continue
		}
		val, ok := measured[n.Threshold.Field]
		if !ok {
			continue // field not enabled or not populated
		}
		if thresholdCrossed(val, n.Threshold.Operator, n.Threshold.Value) {
			s.dispatcher.Dispatch(models.WebhookPayload{
				ID:      n.ID,
				Country: reg.ISOCode,
				Event:   models.EventThreshold,
				Time:    ts,
				Details: &models.ThresholdDetails{
					Field:         n.Threshold.Field,
					Operator:      n.Threshold.Operator,
					Threshold:     n.Threshold.Value,
					MeasuredValue: val,
				},
			}, n.URL)
		}
	}
}

// thresholdCrossed reports whether measured satisfies the comparison
// "measured <operator> threshold". Returns false for any unrecognised operator
// so that unknown operators are treated as non-matching rather than panicking.
func thresholdCrossed(measured float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return measured > threshold
	case "<":
		return measured < threshold
	case ">=":
		return measured >= threshold
	case "<=":
		return measured <= threshold
	default:
		return false
	}
}
