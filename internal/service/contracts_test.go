package service

import (
	"context"
	"encoding/json"
	"testing"
)

func TestOperationContractAndSecretPathRedaction(t *testing.T) {
	op := OperationFunc[string, string]{
		Operation: OperationForecastSeal,
		Run: func(_ context.Context, request string) (Result[string], error) {
			return Result[string]{
				Operation: OperationForecastSeal,
				Code:      "forecast.sealed",
				Message:   "Forecast was sealed",
				Data:      request,
				Effects: []SideEffect{{
					Kind: EffectKey, Action: EffectCreate, Status: EffectCompleted,
					Root: "keys", Path: "f-001.key", Owned: true, Rollback: RollbackRetainSecret,
				}},
				Recovery: Recovery{State: RecoveryComplete},
			}, nil
		},
	}

	if op.Name() != OperationForecastSeal {
		t.Fatalf("operation name = %q", op.Name())
	}
	result, err := op.Execute(context.Background(), "f-001")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(struct {
		Roots  Roots          `json:"roots"`
		Result Result[string] `json:"result"`
	}{
		Roots:  Roots{Secret: []Root{{Name: "keys", Class: RootSecret, Path: "/private/keys"}}},
		Result: result,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || contains(string(encoded), "/private/keys") {
		t.Fatalf("serialized contract exposed private root: %s", encoded)
	}
}

func TestUnconfiguredOperationFailsClosed(t *testing.T) {
	var operation OperationFunc[struct{}, struct{}]
	if _, err := operation.Execute(context.Background(), struct{}{}); err == nil {
		t.Fatal("unconfigured operation succeeded")
	}
}

func contains(value, substring string) bool {
	for index := 0; index+len(substring) <= len(value); index++ {
		if value[index:index+len(substring)] == substring {
			return true
		}
	}
	return false
}
