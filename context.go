package porm

import "context"

var transactionKey = &contextKey{Name: "transaction_key"}

type contextKey struct {
	Name string
}

func txORMFromContext(ctx context.Context) *orm {
	value := ctx.Value(transactionKey)
	if orm, ok := value.(*orm); ok {
		return orm
	}
	return nil
}

func WithTxContext(ctx context.Context, orm *orm) context.Context {
	return context.WithValue(ctx, transactionKey, orm)
}
