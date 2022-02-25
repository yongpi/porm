package porm

import "context"

type HookType int

const (
	BeforeInsert HookType = iota + 1
	AfterInsert
	BeforeUpdate
	AfterUpdate
	BeforeSelect
	AfterSelect
	BeforeDelete
	AfterDelete
)

type Hook func(ctx context.Context, orm *orm)

var Hooks map[HookType][]Hook

func InjectHook(hook Hook, hookType HookType) {
	list := Hooks[hookType]
	list = append(list, hook)
	Hooks[hookType] = list
}

func init() {
	Hooks = make(map[HookType][]Hook)
}

func Fishing(ctx context.Context, hookType HookType, o *orm) {
	hooks, ok := Hooks[hookType]
	if !ok {
		return
	}

	for _, hook := range hooks {
		hook(ctx, o)
		if o.err != nil {
			return
		}
	}

	return
}

type TxHook struct {
}

func (t *TxHook) BeginTxHook(ctx context.Context, orm *orm) {
	no, err := orm.BeginTx(ctx)
	if err != nil {
		orm.err = err
		return
	}

	orm.Copy(no)
	return
}

func (t *TxHook) EndTxHook(ctx context.Context, orm *orm) {
	if orm.err == nil {
		orm.err = orm.Commit()
		return
	}

	orm.err = orm.RollBack()
	return
}

type SelectHook struct {
}

func (t *SelectHook) BeforeHook(ctx context.Context, orm *orm) {
	no := TxORMFromContext(ctx)
	if no == nil {
		return
	}

	orm.Copy(no)
	return
}

func init() {
	txHook := &TxHook{}
	selectHook := &SelectHook{}

	InjectHook(selectHook.BeforeHook, BeforeSelect)

	InjectHook(txHook.BeginTxHook, BeforeInsert)
	InjectHook(txHook.EndTxHook, AfterInsert)

	InjectHook(txHook.BeginTxHook, BeforeDelete)
	InjectHook(txHook.EndTxHook, AfterDelete)

	InjectHook(txHook.BeginTxHook, BeforeUpdate)
	InjectHook(txHook.EndTxHook, AfterUpdate)
}
