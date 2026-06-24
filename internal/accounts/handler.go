package accounts

import (
	"net/http"

	"github.com/sreeni/openbank-bian/pkg/httpx"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// consentHeader carries the consent id a TPP presents on every public AIS read.
// In a full FAPI deployment the consent id is derived from the access token's
// scope; here it is passed directly for the demo.
const consentHeader = "x-consent-id"

// Handler exposes the accounts service over HTTP using OBIE response shapes.
// baseURL is used to build absolute Self links.
type Handler struct {
	svc     *Service
	baseURL string
}

// NewHandler constructs the HTTP handler.
func NewHandler(svc *Service, baseURL string) *Handler {
	return &Handler{svc: svc, baseURL: baseURL}
}

// Routes registers every accounts route on a ServeMux and returns it.
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Public OBIE AIS resources (consent-protected via x-consent-id).
	mux.HandleFunc("GET /accounts", h.listAccounts)
	mux.HandleFunc("GET /accounts/{accountId}", h.getAccount)
	mux.HandleFunc("GET /accounts/{accountId}/balances", h.accountBalances)
	mux.HandleFunc("GET /accounts/{accountId}/transactions", h.accountTransactions)
	mux.HandleFunc("GET /balances", h.allBalances)
	mux.HandleFunc("GET /transactions", h.allTransactions)

	// Internal API used by the CBPII (funds) service. No consent required.
	mux.HandleFunc("GET /internal/funds-confirmation", h.fundsConfirmation)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

func (h *Handler) self(r *http.Request) string { return h.baseURL + r.URL.Path }

// consentID extracts the consent id presented by the caller.
func consentID(r *http.Request) string { return r.Header.Get(consentHeader) }

// ---- accounts ----

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	accs, err := h.svc.ListAccounts(r.Context(), consentID(r))
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	data := accountListData{Account: accountsToDTO(accs)}
	httpx.WriteJSON(w, http.StatusOK, obie.NewResponse(h.self(r), data))
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.GetAccount(r.Context(), consentID(r), r.PathValue("accountId"))
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	// OBIE returns the single account inside a one-element array.
	data := accountListData{Account: []accountDTO{accountToDTO(*a)}}
	httpx.WriteJSON(w, http.StatusOK, obie.NewResponse(h.self(r), data))
}

func (h *Handler) accountBalances(w http.ResponseWriter, r *http.Request) {
	bs, err := h.svc.ListBalances(r.Context(), consentID(r), r.PathValue("accountId"))
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	data := balanceListData{Balance: balancesToDTO(bs)}
	httpx.WriteJSON(w, http.StatusOK, obie.NewResponse(h.self(r), data))
}

func (h *Handler) accountTransactions(w http.ResponseWriter, r *http.Request) {
	ts, err := h.svc.ListTransactions(r.Context(), consentID(r), r.PathValue("accountId"))
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	data := transactionListData{Transaction: transactionsToDTO(ts)}
	httpx.WriteJSON(w, http.StatusOK, obie.NewResponse(h.self(r), data))
}

func (h *Handler) allBalances(w http.ResponseWriter, r *http.Request) {
	bs, err := h.svc.ListAllBalances(r.Context(), consentID(r))
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	data := balanceListData{Balance: balancesToDTO(bs)}
	httpx.WriteJSON(w, http.StatusOK, obie.NewResponse(h.self(r), data))
}

func (h *Handler) allTransactions(w http.ResponseWriter, r *http.Request) {
	ts, err := h.svc.ListAllTransactions(r.Context(), consentID(r))
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	data := transactionListData{Transaction: transactionsToDTO(ts)}
	httpx.WriteJSON(w, http.StatusOK, obie.NewResponse(h.self(r), data))
}

// ---- internal ----

// fundsConfirmation answers the CBPII funds check. It reads the account by its
// OBIE Identification and compares the available balance to the requested
// amount. The query params and response shape match pkg/accountscli.
func (h *Handler) fundsConfirmation(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	identification := q.Get("identification")
	if identification == "" {
		httpx.RespondError(w, httpx.BadRequest("identification query parameter is required",
			httpx.Detail(obie.ErrFieldMissing, "missing identification", "")))
		return
	}
	amount, err := obie.NewAmount(q.Get("amount"), q.Get("currency"))
	if err != nil {
		httpx.RespondError(w, httpx.BadRequest("Invalid amount or currency",
			httpx.Detail(obie.ErrFieldInvalid, err.Error(), "")))
		return
	}

	available, err := h.svc.FundsAvailable(r.Context(), identification, amount)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, fundsConfirmationResp{FundsAvailable: available})
}
