// Package accounts implements the BIAN "Current Account" / OBIE AISP service
// domain. It owns the account, balance and transaction read model exposed by
// the OBIE Account and Transaction API. It never stores consent itself: every
// public read is authorised by calling the central consent service, so this
// service trusts only a consent id presented in the x-consent-id header.
package accounts

import (
	"time"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// OBIE balance types. We model only InterimAvailable — the balance an AISP
// uses to decide whether a payment can be afforded — which is also what the
// internal funds-confirmation endpoint compares against.
const (
	BalanceTypeInterimAvailable = "InterimAvailable"
)

// Credit/Debit indicators per the OBIE OBCreditDebitCode enum.
const (
	IndicatorCredit = "Credit"
	IndicatorDebit  = "Debit"
)

// Account is the OBIE OBAccount6 resource: a single payment account the PSU
// holds. The SchemeName/Identification/Name triple is the OBIE account
// identifier block used to address the account from other resources and from
// the funds-confirmation flow.
type Account struct {
	AccountID      string
	Status         string // OBIE OBExternalAccountStatus1Code, e.g. "Enabled"
	Currency       string // ISO 4217 code
	AccountType    string // OBIE OBExternalAccountType1Code, e.g. "Personal"
	AccountSubType string // OBIE OBExternalAccountSubType1Code, e.g. "CurrentAccount"
	Nickname       string

	// OBIE account identifier block (OBAccount6.Account).
	SchemeName     string
	Identification string
	Name           string
}

// Balance is the OBIE OBReadBalance1 cash balance for an account. Only the
// InterimAvailable balance is modelled in this estate.
type Balance struct {
	AccountID            string
	Type                 string
	CreditDebitIndicator string
	Amount               obie.Amount
	DateTime             time.Time
}

// Transaction is the OBIE OBTransaction6 entry on an account statement. All
// seeded transactions are Booked (settled); pending transactions are not
// modelled.
type Transaction struct {
	TransactionID          string
	AccountID              string
	CreditDebitIndicator   string
	Status                 string // OBIE OBEntryStatus1Code, e.g. "Booked"
	Amount                 obie.Amount
	BookingDateTime        time.Time
	TransactionInformation string
}

// availableBalance reduces a slice of balances to the account's available
// funds: the amount of the single InterimAvailable balance. A Debit indicator
// means the available balance is negative (an overdrawn account), so the value
// is negated; a Credit indicator is taken as-is. The zero Amount is returned
// when the account has no InterimAvailable balance, so callers always get a
// usable value.
//
// It returns the InterimAvailable amount and whether one was found.
func availableBalance(balances []Balance) (obie.Amount, bool) {
	for _, b := range balances {
		if b.Type != BalanceTypeInterimAvailable {
			continue
		}
		if b.CreditDebitIndicator == IndicatorDebit {
			return obie.Amount{Value: b.Amount.Value.Neg(), Currency: b.Amount.Currency}, true
		}
		return b.Amount, true
	}
	return obie.Amount{}, false
}
