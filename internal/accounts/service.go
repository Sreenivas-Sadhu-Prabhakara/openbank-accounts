package accounts

import (
	"context"
	"errors"

	"github.com/sreeni/openbank-bian/pkg/consentcli"
	"github.com/sreeni/openbank-bian/pkg/httpx"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// ConsentClient is the subset of the consent service client the accounts
// service depends on. *consentcli.Client satisfies it; tests inject a fake so
// consent enforcement can be exercised without a running consent service.
type ConsentClient interface {
	Get(ctx context.Context, id string) (*consentcli.View, error)
}

// OBIE AIS permissions, grouped by the endpoint family they unlock.
const (
	permReadAccountsBasic  = "ReadAccountsBasic"
	permReadAccountsDetail = "ReadAccountsDetail"
	permReadBalances       = "ReadBalances"

	permReadTransactionsBasic   = "ReadTransactionsBasic"
	permReadTransactionsCredits = "ReadTransactionsCredits"
	permReadTransactionsDebits  = "ReadTransactionsDebits"
	permReadTransactionsDetail  = "ReadTransactionsDetail"
)

// Service holds the accounts business logic: it reads the account model from
// the repository and authorises every public read against the consent service.
//
// SIMPLIFICATION: an authorised account-access consent grants read access to
// ALL seeded accounts. A real ASPSP records which of the PSU's accounts were
// selected at authorisation time and scopes access to those; we do not model
// PSU↔account selection, so any valid account-access consent can read any
// account in this demo estate.
type Service struct {
	repo    Repository
	consent ConsentClient
}

// NewService wires a Service to its repository and consent client.
func NewService(repo Repository, consent ConsentClient) *Service {
	return &Service{repo: repo, consent: consent}
}

// authorise validates the consent id taken from the x-consent-id header and
// checks the consent grants at least one of the required permissions. It
// returns the consent View on success, or an APIError mapping to 401/403.
//
// The wantPerms slice is an OR set: the consent needs any one of them. An empty
// set means the endpoint only requires a valid account-access consent (no
// specific permission), which no public endpoint currently uses.
func (s *Service) authorise(ctx context.Context, consentID string, wantPerms ...string) (*consentcli.View, error) {
	if consentID == "" {
		return nil, httpx.Unauthorized("Missing x-consent-id header",
			httpx.Detail(obie.ErrHeaderMissing, "x-consent-id header is required", ""))
	}

	view, err := s.consent.Get(ctx, consentID)
	if err != nil {
		if errors.Is(err, consentcli.ErrNotFound) {
			return nil, httpx.Forbidden("Consent not found",
				httpx.Detail(obie.ErrResourceInvalid, "unknown consent", ""))
		}
		return nil, httpx.Internal("could not validate consent")
	}

	if view.Type != consentcli.TypeAccountAccess {
		return nil, httpx.Forbidden("Consent is not an account-access consent",
			httpx.Detail(obie.ErrResourceInvalid, "consent type "+view.Type+" cannot read accounts", ""))
	}
	if view.Status != consentcli.StatusAuthorised {
		return nil, httpx.Forbidden("Consent is not authorised",
			httpx.Detail(obie.ErrResourceInvalid, "consent status is "+view.Status, ""))
	}

	if len(wantPerms) > 0 && !hasAnyPermission(view, wantPerms) {
		return nil, httpx.Forbidden("Consent does not grant the required permission",
			httpx.Detail(obie.ErrResourceInvalid, "missing required permission", ""))
	}
	return view, nil
}

// hasAnyPermission reports whether the consent grants at least one of perms.
func hasAnyPermission(view *consentcli.View, perms []string) bool {
	for _, p := range perms {
		if view.HasPermission(p) {
			return true
		}
	}
	return false
}

// ListAccounts authorises with ReadAccountsBasic|ReadAccountsDetail and returns
// every account in the estate.
func (s *Service) ListAccounts(ctx context.Context, consentID string) ([]Account, error) {
	if _, err := s.authorise(ctx, consentID, permReadAccountsBasic, permReadAccountsDetail); err != nil {
		return nil, err
	}
	accs, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, httpx.Internal("could not load accounts")
	}
	return accs, nil
}

// GetAccount authorises with ReadAccountsBasic|ReadAccountsDetail and returns a
// single account.
func (s *Service) GetAccount(ctx context.Context, consentID, accountID string) (*Account, error) {
	if _, err := s.authorise(ctx, consentID, permReadAccountsBasic, permReadAccountsDetail); err != nil {
		return nil, err
	}
	a, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return nil, s.mapNotFound(err)
	}
	return a, nil
}

// ListBalances authorises with ReadBalances and returns one account's balances.
func (s *Service) ListBalances(ctx context.Context, consentID, accountID string) ([]Balance, error) {
	if _, err := s.authorise(ctx, consentID, permReadBalances); err != nil {
		return nil, err
	}
	bs, err := s.repo.ListBalances(ctx, accountID)
	if err != nil {
		return nil, s.mapNotFound(err)
	}
	return bs, nil
}

// ListTransactions authorises with any of the ReadTransactions* permissions and
// returns one account's transactions.
func (s *Service) ListTransactions(ctx context.Context, consentID, accountID string) ([]Transaction, error) {
	if _, err := s.authorise(ctx, consentID, transactionPerms()...); err != nil {
		return nil, err
	}
	ts, err := s.repo.ListTransactions(ctx, accountID)
	if err != nil {
		return nil, s.mapNotFound(err)
	}
	return ts, nil
}

// ListAllBalances authorises with ReadBalances and returns balances across all
// accounts.
func (s *Service) ListAllBalances(ctx context.Context, consentID string) ([]Balance, error) {
	if _, err := s.authorise(ctx, consentID, permReadBalances); err != nil {
		return nil, err
	}
	bs, err := s.repo.ListAllBalances(ctx)
	if err != nil {
		return nil, httpx.Internal("could not load balances")
	}
	return bs, nil
}

// ListAllTransactions authorises with any ReadTransactions* permission and
// returns transactions across all accounts.
func (s *Service) ListAllTransactions(ctx context.Context, consentID string) ([]Transaction, error) {
	if _, err := s.authorise(ctx, consentID, transactionPerms()...); err != nil {
		return nil, err
	}
	ts, err := s.repo.ListAllTransactions(ctx)
	if err != nil {
		return nil, httpx.Internal("could not load transactions")
	}
	return ts, nil
}

// FundsAvailable answers the internal funds-confirmation question: does the
// account identified by identification hold at least amount of available
// funds? No consent is required — this is an internal estate call used by the
// CBPII (funds) service, which enforces its own consent.
func (s *Service) FundsAvailable(ctx context.Context, identification string, amount obie.Amount) (bool, error) {
	a, err := s.repo.FindByIdentification(ctx, identification)
	if err != nil {
		return false, s.mapNotFound(err)
	}
	available, err := s.repo.AvailableBalance(ctx, a.AccountID)
	if err != nil {
		return false, httpx.Internal("could not load balance")
	}
	ok, err := available.GreaterThanOrEqual(amount)
	if err != nil {
		// Currency mismatch between the request and the account: by definition
		// the requested funds are not available in that currency.
		return false, nil
	}
	return ok, nil
}

// transactionPerms is the OR set of permissions any of which unlocks the
// transactions endpoints.
func transactionPerms() []string {
	return []string{
		permReadTransactionsBasic,
		permReadTransactionsCredits,
		permReadTransactionsDebits,
		permReadTransactionsDetail,
	}
}

func (s *Service) mapNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound("Account not found",
			httpx.Detail(obie.ErrResourceNotFound, "no such account", ""))
	}
	return httpx.Internal("could not load account")
}
