package accounts

import (
	"time"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// This file holds the OBIE wire shapes (DTOs) for the AIS read resources and
// the mappers from the domain types. AIS resources carry no Risk block, so the
// plain obie.Response envelope is used throughout (built by the handler).

// accountIdentifierDTO is the OBIE account identifier block (OBAccount6.Account
// element) nested inside an account resource.
type accountIdentifierDTO struct {
	SchemeName     string `json:"SchemeName"`
	Identification string `json:"Identification"`
	Name           string `json:"Name,omitempty"`
}

// accountDTO is the OBIE OBAccount6 resource on the wire.
type accountDTO struct {
	AccountID      string                 `json:"AccountId"`
	Status         string                 `json:"Status"`
	Currency       string                 `json:"Currency"`
	AccountType    string                 `json:"AccountType"`
	AccountSubType string                 `json:"AccountSubType"`
	Nickname       string                 `json:"Nickname,omitempty"`
	Account        []accountIdentifierDTO `json:"Account"`
}

func accountToDTO(a Account) accountDTO {
	return accountDTO{
		AccountID:      a.AccountID,
		Status:         a.Status,
		Currency:       a.Currency,
		AccountType:    a.AccountType,
		AccountSubType: a.AccountSubType,
		Nickname:       a.Nickname,
		Account: []accountIdentifierDTO{{
			SchemeName:     a.SchemeName,
			Identification: a.Identification,
			Name:           a.Name,
		}},
	}
}

func accountsToDTO(accs []Account) []accountDTO {
	out := make([]accountDTO, 0, len(accs))
	for _, a := range accs {
		out = append(out, accountToDTO(a))
	}
	return out
}

// balanceDTO is the OBIE OBReadBalance1 cash balance on the wire.
type balanceDTO struct {
	AccountID            string      `json:"AccountId"`
	Type                 string      `json:"Type"`
	CreditDebitIndicator string      `json:"CreditDebitIndicator"`
	Amount               obie.Amount `json:"Amount"`
	DateTime             string      `json:"DateTime"`
}

func balanceToDTO(b Balance) balanceDTO {
	return balanceDTO{
		AccountID:            b.AccountID,
		Type:                 b.Type,
		CreditDebitIndicator: b.CreditDebitIndicator,
		Amount:               b.Amount,
		DateTime:             rfc3339(b.DateTime),
	}
}

func balancesToDTO(bs []Balance) []balanceDTO {
	out := make([]balanceDTO, 0, len(bs))
	for _, b := range bs {
		out = append(out, balanceToDTO(b))
	}
	return out
}

// transactionDTO is the OBIE OBTransaction6 statement entry on the wire.
type transactionDTO struct {
	TransactionID          string      `json:"TransactionId"`
	AccountID              string      `json:"AccountId"`
	CreditDebitIndicator   string      `json:"CreditDebitIndicator"`
	Status                 string      `json:"Status"`
	Amount                 obie.Amount `json:"Amount"`
	BookingDateTime        string      `json:"BookingDateTime"`
	TransactionInformation string      `json:"TransactionInformation,omitempty"`
}

func transactionToDTO(t Transaction) transactionDTO {
	return transactionDTO{
		TransactionID:          t.TransactionID,
		AccountID:              t.AccountID,
		CreditDebitIndicator:   t.CreditDebitIndicator,
		Status:                 t.Status,
		Amount:                 t.Amount,
		BookingDateTime:        rfc3339(t.BookingDateTime),
		TransactionInformation: t.TransactionInformation,
	}
}

func transactionsToDTO(ts []Transaction) []transactionDTO {
	out := make([]transactionDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, transactionToDTO(t))
	}
	return out
}

// The OBIE Data payloads wrap each resource collection under its named key.

type accountListData struct {
	Account []accountDTO `json:"Account"`
}

type balanceListData struct {
	Balance []balanceDTO `json:"Balance"`
}

type transactionListData struct {
	Transaction []transactionDTO `json:"Transaction"`
}

// fundsConfirmationResp is the internal funds-confirmation endpoint body. It
// matches the shape pkg/accountscli decodes.
type fundsConfirmationResp struct {
	FundsAvailable bool `json:"FundsAvailable"`
}

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }
