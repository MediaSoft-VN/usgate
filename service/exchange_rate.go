package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

type vietcombankExchangeResponse struct {
	Date        string `json:"Date"`
	UpdatedDate string `json:"UpdatedDate"`
	Data        []struct {
		CurrencyCode string `json:"currencyCode"`
		Cash         string `json:"cash"`
		Transfer     string `json:"transfer"`
		Sell         string `json:"sell"`
	} `json:"Data"`
}

func VietcombankExchangeRates(ctx context.Context, date time.Time) error {
	endpoint := "https://www.vietcombank.com.vn/api/exchangerates?date=" + date.Format("2006-01-02")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("exchange rate provider returned status %d", resp.StatusCode)
	}
	var payload vietcombankExchangeResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return err
	}
	for _, item := range payload.Data {
		rate, err := exchangeRateValue(item.Cash, item.Transfer, item.Sell, operation_setting.GetGeneralSetting().ExchangeRateType)
		if err != nil || rate == "" {
			continue
		}
		if _, err := decimal.NewFromString(rate); err != nil {
			continue
		}
		if err := model.SaveExchangeRate(&model.ExchangeRate{
			BaseCurrency: "VND", QuoteCurrency: strings.ToUpper(item.CurrencyCode), Rate: rate,
			Source: model.ExchangeRateSourceVietcombank, RateType: operation_setting.GetGeneralSetting().ExchangeRateType,
			EffectiveDate: date.Format("2006-01-02"), FetchedAt: time.Now().Unix(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func exchangeRateValue(cash string, transfer string, sell string, rateType string) (string, error) {
	switch strings.ToLower(rateType) {
	case model.ExchangeRateTypeCash:
		return cash, nil
	case model.ExchangeRateTypeTransfer:
		return transfer, nil
	case model.ExchangeRateTypeSell:
		return sell, nil
	default:
		return "", fmt.Errorf("unsupported exchange rate type %q", rateType)
	}
}

func CurrentExchangeRate(quoteCurrency string) (decimal.Decimal, error) {
	quoteCurrency = strings.ToUpper(strings.TrimSpace(quoteCurrency))
	if quoteCurrency == "VND" {
		return decimal.NewFromInt(1), nil
	}
	settings := operation_setting.GetGeneralSetting()
	if settings.ExchangeRateSource == model.ExchangeRateSourceManual {
		return manualExchangeRate(settings.ManualExchangeRates, quoteCurrency)
	}
	rate := model.GetLatestExchangeRate("VND", quoteCurrency, settings.ExchangeRateSource, settings.ExchangeRateType)
	if rate == nil {
		return decimal.Zero, fmt.Errorf("exchange rate unavailable for %s", quoteCurrency)
	}
	value, err := decimal.NewFromString(rate.Rate)
	if err != nil || value.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("invalid exchange rate for %s", quoteCurrency)
	}
	return value, nil
}

func ConvertToVND(amount decimal.Decimal, currency string) (decimal.Decimal, decimal.Decimal, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("amount must be positive")
	}
	rate, err := CurrentExchangeRate(currency)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return amount.Mul(rate), rate, nil
}

// ProviderFundingQuota converts a provider deposit in its currency into the
// quota units used by users. Rates are expressed through VND so the configured
// system currency and provider currency can differ without float arithmetic.
func ProviderFundingQuota(amountMinor int64, currency string) (int, string, error) {
	if amountMinor <= 0 {
		return 0, "", fmt.Errorf("amount must be positive")
	}
	systemCurrency := operation_setting.GetFinancialCurrencyCode()
	providerRate, err := CurrentExchangeRate(currency)
	if err != nil {
		return 0, "", err
	}
	systemRate, err := CurrentExchangeRate(systemCurrency)
	if err != nil {
		return 0, "", err
	}
	amount := decimal.NewFromInt(amountMinor).Div(decimal.NewFromInt(100)).Mul(providerRate).Div(systemRate)
	quota, err := common.WalletQuotaFromDecimalStrict(amount.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Round(0))
	if err != nil || quota <= 0 {
		return 0, providerRate.String(), fmt.Errorf("converted quota exceeds supported range")
	}
	return quota, providerRate.String(), nil
}

type DisplayExchangeRate struct {
	QuoteCurrency string `json:"quote_currency"`
	Rate          string `json:"rate"`
	EffectiveDate string `json:"effective_date"`
	Source        string `json:"source"`
	RateType      string `json:"rate_type"`
}

// CurrentRatesForDisplay returns what the admin UI table shows: stored bank
// snapshots for bank sources, or the parsed manual map for manual mode.
func CurrentRatesForDisplay() []DisplayExchangeRate {
	settings := operation_setting.GetGeneralSetting()
	if settings.ExchangeRateSource == model.ExchangeRateSourceManual {
		var rates map[string]string
		if err := common.UnmarshalJsonStr(settings.ManualExchangeRates, &rates); err != nil {
			return nil
		}
		out := make([]DisplayExchangeRate, 0, len(rates))
		for quote, rate := range rates {
			if _, err := decimal.NewFromString(rate); err != nil {
				continue
			}
			out = append(out, DisplayExchangeRate{
				QuoteCurrency: strings.ToUpper(strings.TrimSpace(quote)),
				Rate:          rate,
				Source:        model.ExchangeRateSourceManual,
				RateType:      settings.ExchangeRateType,
			})
		}
		return out
	}
	rows := model.ListLatestExchangeRates("VND", settings.ExchangeRateSource, settings.ExchangeRateType)
	out := make([]DisplayExchangeRate, 0, len(rows))
	for _, row := range rows {
		out = append(out, DisplayExchangeRate{
			QuoteCurrency: row.QuoteCurrency,
			Rate:          row.Rate,
			EffectiveDate: row.EffectiveDate,
			Source:        row.Source,
			RateType:      row.RateType,
		})
	}
	return out
}

func manualExchangeRate(raw string, quoteCurrency string) (decimal.Decimal, error) {
	var rates map[string]string
	if err := common.UnmarshalJsonStr(raw, &rates); err != nil {
		return decimal.Zero, err
	}
	value, ok := rates[quoteCurrency]
	if !ok {
		return decimal.Zero, fmt.Errorf("manual exchange rate unavailable for %s", quoteCurrency)
	}
	rate, err := decimal.NewFromString(value)
	if err != nil || rate.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("invalid manual exchange rate for %s", quoteCurrency)
	}
	return rate, nil
}
