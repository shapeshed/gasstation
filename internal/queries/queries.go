package queries

import (
	"context"

	"github.com/cenkalti/backoff/v4"
	ssbackoff "github.com/shapeshed/servicestation/internal/backoff"

	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func GetBalance(ctx context.Context, client bank.QueryClient, user, denom string) (*bank.QueryBalanceResponse, error) {
	req := bank.QueryBalanceRequest{
		Address: user,
		Denom:   denom,
	}

	var userBalance *bank.QueryBalanceResponse
	var err error

	exponentialBackoff := ssbackoff.NewBackoff(ctx)

	retryableRequest := func() error {
		userBalance, err = client.Balance(ctx, &req)
		return err
	}

	if err := backoff.Retry(retryableRequest, exponentialBackoff); err != nil {
		return nil, err
	}

	return userBalance, nil
}
