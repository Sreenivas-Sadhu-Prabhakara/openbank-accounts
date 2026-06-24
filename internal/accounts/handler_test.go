package accounts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// newTestHandler wires a handler over the seeded MemRepository and a fake
// consent client granting every AIS permission under the id "good".
func newTestHandler() http.Handler {
	fc := newFakeConsent()
	fc.add("good", allPerms()...)
	fc.add("balances-only", permReadBalances)
	svc := NewService(NewMemRepository(), fc)
	return NewHandler(svc, "http://accounts.test").Routes()
}

// do issues a request to the handler with an optional consent id and returns
// the recorder.
func do(t *testing.T, h http.Handler, method, path, consentID string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	if consentID != "" {
		r.Header.Set(consentHeader, consentID)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func mustDecode(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
}

func TestListAccountsHandler(t *testing.T) {
	h := newTestHandler()
	w := do(t, h, http.MethodGet, "/accounts", "good")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body)
	}
	var resp struct {
		Data  accountListData `json:"Data"`
		Links obie.Links      `json:"Links"`
		Meta  obie.Meta       `json:"Meta"`
	}
	mustDecode(t, w, &resp)
	if len(resp.Data.Account) != 2 {
		t.Fatalf("got %d accounts, want 2", len(resp.Data.Account))
	}
	if resp.Links.Self != "http://accounts.test/accounts" {
		t.Fatalf("Self = %s", resp.Links.Self)
	}
	if resp.Meta.TotalPages != 1 {
		t.Fatalf("TotalPages = %d", resp.Meta.TotalPages)
	}
}

func TestGetAccountHandlerSingleElementArray(t *testing.T) {
	h := newTestHandler()
	w := do(t, h, http.MethodGet, "/accounts/22289", "good")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body)
	}
	var resp struct {
		Data accountListData `json:"Data"`
	}
	mustDecode(t, w, &resp)
	if len(resp.Data.Account) != 1 {
		t.Fatalf("expected single-element array, got %d", len(resp.Data.Account))
	}
	if resp.Data.Account[0].Account[0].Identification != "80200110203345" {
		t.Fatalf("identification = %s", resp.Data.Account[0].Account[0].Identification)
	}
}

func TestBalancesAndTransactionsHandlers(t *testing.T) {
	h := newTestHandler()

	wb := do(t, h, http.MethodGet, "/accounts/22289/balances", "good")
	if wb.Code != http.StatusOK {
		t.Fatalf("balances status = %d", wb.Code)
	}
	var balResp struct {
		Data balanceListData `json:"Data"`
	}
	mustDecode(t, wb, &balResp)
	if len(balResp.Data.Balance) != 1 || balResp.Data.Balance[0].Amount.String() != "1230" {
		t.Fatalf("unexpected balances %+v", balResp.Data.Balance)
	}

	wt := do(t, h, http.MethodGet, "/accounts/22289/transactions", "good")
	if wt.Code != http.StatusOK {
		t.Fatalf("transactions status = %d", wt.Code)
	}
	var txnResp struct {
		Data transactionListData `json:"Data"`
	}
	mustDecode(t, wt, &txnResp)
	if len(txnResp.Data.Transaction) != 2 {
		t.Fatalf("got %d transactions, want 2", len(txnResp.Data.Transaction))
	}

	// Bulk endpoints.
	wab := do(t, h, http.MethodGet, "/balances", "good")
	var allBal struct {
		Data balanceListData `json:"Data"`
	}
	mustDecode(t, wab, &allBal)
	if len(allBal.Data.Balance) != 2 {
		t.Fatalf("all balances = %d, want 2", len(allBal.Data.Balance))
	}

	wat := do(t, h, http.MethodGet, "/transactions", "good")
	var allTxn struct {
		Data transactionListData `json:"Data"`
	}
	mustDecode(t, wat, &allTxn)
	if len(allTxn.Data.Transaction) != 3 {
		t.Fatalf("all transactions = %d, want 3", len(allTxn.Data.Transaction))
	}
}

func TestMissingConsentHeaderIs401(t *testing.T) {
	h := newTestHandler()
	w := do(t, h, http.MethodGet, "/accounts", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body)
	}
	var errBody obie.ErrorResponse
	mustDecode(t, w, &errBody)
	if len(errBody.Errors) == 0 || errBody.Errors[0].ErrorCode != obie.ErrHeaderMissing {
		t.Fatalf("unexpected error body %+v", errBody)
	}
}

func TestInsufficientPermissionIs403(t *testing.T) {
	h := newTestHandler()
	// balances-only consent must not be able to list accounts.
	w := do(t, h, http.MethodGet, "/accounts", "balances-only")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body)
	}
	var errBody obie.ErrorResponse
	mustDecode(t, w, &errBody)
	if errBody.Code != http.StatusText(http.StatusForbidden) {
		t.Fatalf("code = %s", errBody.Code)
	}
}

func TestFundsConfirmationHandler(t *testing.T) {
	h := newTestHandler()

	t.Run("funds available, no consent required", func(t *testing.T) {
		w := do(t, h, http.MethodGet,
			"/internal/funds-confirmation?identification=80200110203345&amount=1000.00&currency=GBP", "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", w.Code, w.Body)
		}
		var resp fundsConfirmationResp
		mustDecode(t, w, &resp)
		if !resp.FundsAvailable {
			t.Fatal("expected FundsAvailable true")
		}
	})

	t.Run("funds not available", func(t *testing.T) {
		w := do(t, h, http.MethodGet,
			"/internal/funds-confirmation?identification=80200110203345&amount=99999.00&currency=GBP", "")
		var resp fundsConfirmationResp
		mustDecode(t, w, &resp)
		if resp.FundsAvailable {
			t.Fatal("expected FundsAvailable false")
		}
	})

	t.Run("unknown identification is 404", func(t *testing.T) {
		w := do(t, h, http.MethodGet,
			"/internal/funds-confirmation?identification=00000000000000&amount=1.00&currency=GBP", "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestHealth(t *testing.T) {
	h := newTestHandler()
	w := do(t, h, http.MethodGet, "/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}
