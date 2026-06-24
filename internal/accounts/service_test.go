package accounts

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/sreeni/openbank-bian/pkg/consentcli"
	"github.com/sreeni/openbank-bian/pkg/httpx"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// fakeConsent is an in-memory ConsentClient. It maps a consent id to the View
// the consent service would return, so tests drive enforcement without HTTP.
type fakeConsent struct {
	views map[string]*consentcli.View
}

func newFakeConsent() *fakeConsent {
	return &fakeConsent{views: make(map[string]*consentcli.View)}
}

// add registers an authorised account-access consent with the given perms.
func (f *fakeConsent) add(id string, perms ...string) {
	f.views[id] = &consentcli.View{
		ConsentID:   id,
		Type:        consentcli.TypeAccountAccess,
		Status:      consentcli.StatusAuthorised,
		Permissions: perms,
	}
}

func (f *fakeConsent) addView(id string, v *consentcli.View) { f.views[id] = v }

func (f *fakeConsent) Get(_ context.Context, id string) (*consentcli.View, error) {
	v, ok := f.views[id]
	if !ok {
		return nil, consentcli.ErrNotFound
	}
	return v, nil
}

// allPerms is an authorised consent that grants every AIS permission.
func allPerms() []string {
	return []string{
		permReadAccountsBasic, permReadAccountsDetail, permReadBalances,
		permReadTransactionsBasic, permReadTransactionsCredits,
		permReadTransactionsDebits, permReadTransactionsDetail,
	}
}

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()
	var apiErr *httpx.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *httpx.APIError, got %v", err)
	}
	if apiErr.Status != status {
		t.Fatalf("status = %d, want %d (%s)", apiErr.Status, status, apiErr.Message)
	}
}

func TestConsentEnforcement(t *testing.T) {
	ctx := context.Background()
	fc := newFakeConsent()
	fc.add("good", allPerms()...)
	svc := NewService(NewMemRepository(), fc)

	t.Run("missing header is 401", func(t *testing.T) {
		_, err := svc.ListAccounts(ctx, "")
		wantStatus(t, err, http.StatusUnauthorized)
	})

	t.Run("unknown consent is 403", func(t *testing.T) {
		_, err := svc.ListAccounts(ctx, "nope")
		wantStatus(t, err, http.StatusForbidden)
	})

	t.Run("wrong consent type is 403", func(t *testing.T) {
		fc.addView("payment", &consentcli.View{
			ConsentID: "payment",
			Type:      consentcli.TypeDomesticPayment,
			Status:    consentcli.StatusAuthorised,
		})
		_, err := svc.ListAccounts(ctx, "payment")
		wantStatus(t, err, http.StatusForbidden)
	})

	t.Run("unauthorised status is 403", func(t *testing.T) {
		fc.addView("awaiting", &consentcli.View{
			ConsentID:   "awaiting",
			Type:        consentcli.TypeAccountAccess,
			Status:      consentcli.StatusAwaitingAuthorisation,
			Permissions: allPerms(),
		})
		_, err := svc.ListAccounts(ctx, "awaiting")
		wantStatus(t, err, http.StatusForbidden)
	})

	t.Run("missing permission is 403", func(t *testing.T) {
		// Grant only ReadBalances, then attempt to list accounts.
		fc.add("balances-only", permReadBalances)
		_, err := svc.ListAccounts(ctx, "balances-only")
		wantStatus(t, err, http.StatusForbidden)

		// And attempt to read transactions with the same balances-only consent.
		_, err = svc.ListTransactions(ctx, "balances-only", "22289")
		wantStatus(t, err, http.StatusForbidden)
	})
}

func TestPerEndpointPermissions(t *testing.T) {
	ctx := context.Background()
	fc := newFakeConsent()
	fc.add("accounts", permReadAccountsBasic)
	fc.add("balances", permReadBalances)
	fc.add("txns", permReadTransactionsDetail)
	svc := NewService(NewMemRepository(), fc)

	if _, err := svc.ListAccounts(ctx, "accounts"); err != nil {
		t.Fatalf("ReadAccountsBasic should allow /accounts: %v", err)
	}
	if _, err := svc.ListBalances(ctx, "balances", "22289"); err != nil {
		t.Fatalf("ReadBalances should allow balances: %v", err)
	}
	if _, err := svc.ListTransactions(ctx, "txns", "22289"); err != nil {
		t.Fatalf("ReadTransactionsDetail should allow transactions: %v", err)
	}

	// Cross-checks: accounts consent must not unlock balances or transactions.
	if _, err := svc.ListBalances(ctx, "accounts", "22289"); err == nil {
		t.Fatal("accounts consent should not grant balances")
	}
	if _, err := svc.ListTransactions(ctx, "accounts", "22289"); err == nil {
		t.Fatal("accounts consent should not grant transactions")
	}
}

func TestHappyPathsReturnData(t *testing.T) {
	ctx := context.Background()
	fc := newFakeConsent()
	fc.add("good", allPerms()...)
	svc := NewService(NewMemRepository(), fc)

	accs, err := svc.ListAccounts(ctx, "good")
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accs) != 2 {
		t.Fatalf("got %d accounts, want 2", len(accs))
	}

	a, err := svc.GetAccount(ctx, "good", "22289")
	if err != nil || a.Identification != "80200110203345" {
		t.Fatalf("get account: err=%v account=%+v", err, a)
	}

	if _, err := svc.GetAccount(ctx, "good", "missing"); err == nil {
		t.Fatal("expected not found for missing account")
	} else {
		wantStatus(t, err, http.StatusNotFound)
	}

	bs, err := svc.ListBalances(ctx, "good", "22289")
	if err != nil || len(bs) != 1 || bs[0].Type != BalanceTypeInterimAvailable {
		t.Fatalf("balances: err=%v balances=%+v", err, bs)
	}

	ts, err := svc.ListTransactions(ctx, "good", "22289")
	if err != nil || len(ts) != 2 {
		t.Fatalf("transactions: err=%v count=%d", err, len(ts))
	}

	allBal, err := svc.ListAllBalances(ctx, "good")
	if err != nil || len(allBal) != 2 {
		t.Fatalf("all balances: err=%v count=%d", err, len(allBal))
	}

	allTxn, err := svc.ListAllTransactions(ctx, "good")
	if err != nil || len(allTxn) != 3 {
		t.Fatalf("all transactions: err=%v count=%d", err, len(allTxn))
	}
}

func TestFundsAvailable(t *testing.T) {
	ctx := context.Background()
	// Funds confirmation requires no consent, so a nil ConsentClient is fine.
	svc := NewService(NewMemRepository(), newFakeConsent())

	t.Run("sufficient funds returns true", func(t *testing.T) {
		ok, err := svc.FundsAvailable(ctx, "80200110203345", obie.MustAmount("1000.00", "GBP"))
		if err != nil || !ok {
			t.Fatalf("expected funds available: ok=%v err=%v", ok, err)
		}
	})

	t.Run("exact balance returns true", func(t *testing.T) {
		ok, err := svc.FundsAvailable(ctx, "80200110203345", obie.MustAmount("1230.00", "GBP"))
		if err != nil || !ok {
			t.Fatalf("expected funds available at exact balance: ok=%v err=%v", ok, err)
		}
	})

	t.Run("insufficient funds returns false", func(t *testing.T) {
		ok, err := svc.FundsAvailable(ctx, "80200110203345", obie.MustAmount("9999.99", "GBP"))
		if err != nil || ok {
			t.Fatalf("expected funds unavailable: ok=%v err=%v", ok, err)
		}
	})

	t.Run("unknown identification is 404", func(t *testing.T) {
		_, err := svc.FundsAvailable(ctx, "00000000000000", obie.MustAmount("1.00", "GBP"))
		wantStatus(t, err, http.StatusNotFound)
	})

	t.Run("currency mismatch is not available", func(t *testing.T) {
		ok, err := svc.FundsAvailable(ctx, "80200110203345", obie.MustAmount("1.00", "EUR"))
		if err != nil || ok {
			t.Fatalf("expected unavailable on currency mismatch: ok=%v err=%v", ok, err)
		}
	})
}
