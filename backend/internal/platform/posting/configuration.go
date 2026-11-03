// Package posting provides transaction-scoped configuration gates, not business
// posting or cost recognition workflows.
package posting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // IANA validation must also work on standalone Windows installs.

	"buildestate/backend/internal/contracts"
)

var ErrNotReady = errors.New("posting configuration is incomplete or invalid")

type Configuration struct{}

var _ contracts.PostingConfiguration = Configuration{}

// RequireReady reads every gate inside the caller's write transaction. It never
// caches settings or substitutes deployment defaults for operator confirmation.
func (Configuration) RequireReady(ctx context.Context, tx contracts.Transaction) (contracts.PostingSettings, error) {
	var settings contracts.PostingSettings
	var code, notice, retention, confirmed sql.NullString
	var digits sql.NullInt64
	var timezoneConfirmed int
	err := tx.QueryRowContext(ctx, `SELECT currency_code,currency_minor_digits,timezone,timezone_confirmed,
		notice_version,retention_schedule_version,operational_policy_confirmed_at FROM company_settings WHERE id='company'`).Scan(
		&code, &digits, &settings.Timezone, &timezoneConfirmed, &notice, &retention, &confirmed)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.PostingSettings{}, ErrNotReady
	}
	if err != nil {
		return contracts.PostingSettings{}, fmt.Errorf("read posting settings: %w", err)
	}
	if !code.Valid || !digits.Valid || !validCurrency(code.String) || digits.Int64 < 0 || digits.Int64 > 4 ||
		timezoneConfirmed != 1 || settings.Timezone == "" || settings.Timezone == "Local" ||
		strings.TrimSpace(notice.String) == "" || strings.TrimSpace(retention.String) == "" || !confirmed.Valid {
		return contracts.PostingSettings{}, ErrNotReady
	}
	if _, err = time.LoadLocation(settings.Timezone); err != nil {
		return contracts.PostingSettings{}, ErrNotReady
	}
	if _, err = time.Parse(time.RFC3339Nano, confirmed.String); err != nil {
		return contracts.PostingSettings{}, ErrNotReady
	}
	rows, err := tx.QueryContext(ctx, `SELECT country_code FROM operating_countries WHERE settings_id='company'`)
	if err != nil {
		return contracts.PostingSettings{}, fmt.Errorf("read operating countries: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var country string
		if err = rows.Scan(&country); err != nil {
			return contracts.PostingSettings{}, err
		}
		if len(country) != 2 || country[0] < 'A' || country[0] > 'Z' || country[1] < 'A' || country[1] > 'Z' {
			return contracts.PostingSettings{}, ErrNotReady
		}
		count++
	}
	if err = rows.Err(); err != nil {
		return contracts.PostingSettings{}, err
	}
	if count == 0 {
		return contracts.PostingSettings{}, ErrNotReady
	}
	settings.CurrencyCode = code.String
	settings.MinorDigits = int(digits.Int64)
	return settings, nil
}

// LockCurrency must be called in the posting transaction. Repeated locking is
// harmless; rollback removes a first lock along with the failed business writes.
func (c Configuration) LockCurrency(ctx context.Context, tx contracts.Transaction) error {
	if _, err := c.RequireReady(ctx, tx); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE company_settings SET currency_locked=1 WHERE id='company' AND currency_locked=0`)
	if err != nil {
		return fmt.Errorf("lock posting currency: %w", err)
	}
	return nil
}

func validCurrency(code string) bool {
	// ISO 4217 monetary codes, including fund/settlement units. XXX (no
	// currency) and XTS (testing) cannot configure live financial posting.
	const codes = " AED AFN ALL AMD ANG AOA ARS AUD AWG AZN BAM BBD BDT BGN BHD BIF BMD BND BOB BOV BRL BSD BTN BWP BYN BZD CAD CDF CHE CHF CHW CLF CLP CNY COP COU CRC CUP CVE CZK DJF DKK DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL GHS GIP GMD GNF GTQ GYD HKD HNL HTG HUF IDR ILS INR IQD IRR ISK JMD JOD JPY KES KGS KHR KMF KPW KRW KWD KYD KZT LAK LBP LKR LRD LSL LYD MAD MDL MGA MKD MMK MNT MOP MRU MUR MVR MWK MXN MXV MYR MZN NAD NGN NIO NOK NPR NZD OMR PAB PEN PGK PHP PKR PLN PYG QAR RON RSD RUB RWF SAR SBD SCR SDG SEK SGD SHP SLE SOS SRD SSP STN SVC SYP SZL THB TJS TMT TND TOP TRY TTD TWD TZS UAH UGX USD USN UYI UYU UYW UZS VED VES VND VUV WST XAF XCD XCG XDR XOF XPF XSU XUA YER ZAR ZMW ZWG "
	return len(code) == 3 && strings.Contains(codes, " "+code+" ")
}
