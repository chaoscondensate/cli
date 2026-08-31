package service

import (
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestOutcomeClassifiersCoverDryRunUnchangedAndSafeFailure(t *testing.T) {
	tests := []struct {
		name      string
		operation OperationName
		input     OutcomeInput
		code      string
		state     OutcomeState
		failure   app.ErrorCode
	}{
		{"publication dry run", OperationPublicationBuild, OutcomeInput{DryRun: true, Data: PublicationBuildResult{}}, "publication.build.planned", OutcomePlanned, ""},
		{"ledger unchanged", OperationLedgerUpdate, OutcomeInput{Data: RootMetadataFileResult{}}, "ledger.unchanged", OutcomeUnchanged, ""},
		{"platform unchanged", OperationPlatformUpdate, OutcomeInput{Data: PlatformFileResult{}}, "platform.unchanged", OutcomeUnchanged, ""},
		{"question unchanged", OperationQuestionUpdate, OutcomeInput{Data: QuestionFileResult{}}, "question.unchanged", OutcomeUnchanged, ""},
		{"reveal unchanged", OperationForecastReveal, OutcomeInput{Data: ForecastFileResult{}}, "forecast.reveal.unchanged", OutcomeUnchanged, ""},
		{"key hint unchanged", OperationForecastKeyHintUpdate, OutcomeInput{Data: ForecastFileResult{}}, "forecast.key_hint.unchanged", OutcomeUnchanged, ""},
		{"timestamp pass", OperationTimestampVerify, OutcomeInput{Data: TimestampVerifyResult{Verification: VerificationLayer{State: LayerPass}}}, "timestamp.verified", OutcomeSuccess, ""},
		{"target safe failure", OperationTargetCheck, OutcomeInput{Data: TargetOperationResult{FailureCode: app.CodeVerification}}, "target.failed", OutcomePartialFailure, app.CodeVerification},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := definitionByName(t, test.operation)
			outcome := definition.ClassifyOutcome(test.input)
			if outcome.Code != test.code || outcome.State != test.state || outcome.FailureCode != test.failure || !outcome.HasData {
				t.Fatalf("outcome = %+v", outcome)
			}
		})
	}
}

func definitionByName(t *testing.T, name OperationName) OperationDefinition {
	t.Helper()
	for _, definition := range OperationDefinitions() {
		if definition.Name == name {
			return definition
		}
	}
	t.Fatalf("operation %s is not registered", name)
	return OperationDefinition{}
}
