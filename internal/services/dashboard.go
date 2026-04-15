package services

import (
	"context"
	"log/slog"
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
	regs       firebase.RegistrationRepository
	notifs     firebase.NotificationRepository
	countries  *clients.CountriesClient
	meteo      *clients.MeteoClient
	openaq     *clients.OpenAQClient
	currency   *clients.CurrencyClient
	dispatcher *webhook.Dispatcher
	logger     *slog.Logger
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
	logger *slog.Logger,
) DashboardService {
	return &dashboardService{
		regs:       regs,
		notifs:     notifs,
		countries:  countries,
		meteo:      meteo,
		openaq:     openaq,
		currency:   currency,
		dispatcher: dispatcher,
		logger:     logger,
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
		s.logger.Warn("countries API error", "dashboard", id, "err", err)
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
			s.logger.Warn("meteo API error", "dashboard", id, "err", meteoErr)
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
			s.logger.Warn("openaq API error", "dashboard", id, "err", openaqErr)
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
			s.logger.Warn("currency API error", "dashboard", id, "err", currencyErr)
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
// against the live values in resp. A webhook fires only when every condition in
// its Thresholds list is satisfied simultaneously. Fields that are disabled in
// the registration or that failed to populate (nil in resp) are treated as
// absent — any condition referencing an absent field prevents the webhook from
// firing.
func (s *dashboardService) fireThresholds(ctx context.Context, reg *models.Registration, resp *models.DashboardResponse) {
	notifs, err := s.notifs.ListMatching(ctx, reg.ISOCode, models.EventThreshold)
	if err != nil {
		return
	}
	if len(notifs) == 0 {
		return
	}

	// Build a map of field name → live measured value from the dashboard response.
	// Only fields that are enabled and successfully populated are included.
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
	for _, notif := range notifs {
		if len(notif.Thresholds) == 0 {
			continue
		}

		// Evaluate every condition in the list. All must be satisfied for the
		// webhook to fire. Build the detail slice as we go; if any condition
		// fails we break early and skip the dispatch.
		conditionDetails, allMet := evaluateThresholds(notif.Thresholds, measured)
		if !allMet {
			continue
		}

		s.dispatcher.Dispatch(models.WebhookPayload{
			ID:      notif.ID,
			Country: reg.ISOCode,
			Event:   models.EventThreshold,
			Time:    ts,
			Details: &models.ThresholdDetails{
				Conditions: conditionDetails,
			},
		}, notif.URL)
	}
}

// evaluateThresholds checks every condition in the list against the live measured
// values. It returns the per-condition detail slice and a boolean indicating
// whether all conditions were satisfied. If any condition references a field that
// is absent from measured, or the comparison is not satisfied, allMet is false.
func evaluateThresholds(conditions []models.Threshold, measured map[string]float64) (details []models.ThresholdConditionDetail, allMet bool) {
	details = make([]models.ThresholdConditionDetail, 0, len(conditions))
	for _, condition := range conditions {
		measuredValue, fieldPresent := measured[condition.Field]
		if !fieldPresent || !thresholdCrossed(measuredValue, condition.Operator, condition.Value) {
			return nil, false
		}
		details = append(details, models.ThresholdConditionDetail{
			Field:         condition.Field,
			Operator:      condition.Operator,
			Threshold:     condition.Value,
			MeasuredValue: measuredValue,
		})
	}
	return details, true
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
