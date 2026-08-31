package service

import (
	"reflect"
	"sort"
	"strings"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/document"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
)

type PlatformMutation struct {
	Ledger  *ledger.Ledger
	Patches []document.PatchOperation
}

type PlatformListItem struct {
	ID             ledger.Slug         `json:"id"`
	Name           string              `json:"name"`
	Kind           ledger.PlatformKind `json:"kind"`
	ReferenceCount int                 `json:"reference_count"`
}

type PlatformShowResult struct {
	ID                     ledger.Slug     `json:"id"`
	Platform               ledger.Platform `json:"platform"`
	ReferencingQuestionIDs []ledger.Slug   `json:"referencing_question_ids"`
}

func BuildPlatformAdd(model *ledger.Ledger, id ledger.Slug, input PlatformCreateInput) (PlatformMutation, error) {
	var result PlatformMutation
	if model == nil {
		return result, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	if err := ValidatePlatformID(id); err != nil {
		return result, err
	}
	platform := ledger.Platform{Name: input.Name, Kind: input.Kind, URL: cloneString(input.URL)}
	if input.Account != nil {
		platform.Account = &ledger.PlatformAccount{Username: cloneString(input.Account.Username), UserID: cloneString(input.Account.UserID), ProfileURL: cloneString(input.Account.ProfileURL)}
	}
	if err := ValidatePlatform(platform); err != nil {
		return result, err
	}
	if _, exists := model.Platforms[id]; exists {
		return result, app.WithDetails(app.NewError(app.CodeConflict, "platform ID already exists", nil), map[string]any{"platform_id": id})
	}
	prospective, err := cloneLedger(model)
	if err != nil {
		return result, err
	}
	prospective.Platforms[id] = clonePlatform(platform)
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return result, err
	}
	value, err := jsonPatchValue(platform)
	if err != nil {
		return result, err
	}
	result.Ledger = prospective
	result.Patches = []document.PatchOperation{{Kind: document.PatchAdd, Pointer: "/platforms/" + string(id), Value: value}}
	return result, nil
}

func BuildPlatformUpdate(model *ledger.Ledger, id ledger.Slug, input PlatformPatchInput) (PlatformMutation, error) {
	var result PlatformMutation
	if model == nil {
		return result, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	if err := ValidatePlatformID(id); err != nil {
		return result, err
	}
	original, exists := model.Platforms[id]
	if !exists {
		return result, app.WithDetails(app.NewError(app.CodeNotFound, "platform was not found", nil), map[string]any{"platform_id": id})
	}
	updated := clonePlatform(original)
	if input.Name.Set {
		if input.Name.Null {
			return result, invalidField("name", "platform name cannot be removed")
		}
		updated.Name = input.Name.Value
	}
	if input.Kind.Set {
		if input.Kind.Null {
			return result, invalidField("kind", "platform kind cannot be removed")
		}
		updated.Kind = input.Kind.Value
	}
	if input.URL.Set {
		if input.URL.Null {
			updated.URL = nil
		} else {
			updated.URL = cloneString(&input.URL.Value)
		}
	}
	if input.Account.Set {
		if input.Account.Null {
			updated.Account = nil
		} else {
			account := ledger.PlatformAccount{}
			if original.Account != nil {
				account = *original.Account
				account.Username = cloneString(original.Account.Username)
				account.UserID = cloneString(original.Account.UserID)
				account.ProfileURL = cloneString(original.Account.ProfileURL)
			}
			applyOptionalString(&account.Username, input.Account.Value.Username)
			applyOptionalString(&account.UserID, input.Account.Value.UserID)
			applyOptionalString(&account.ProfileURL, input.Account.Value.ProfileURL)
			updated.Account = &account
		}
	}
	if err := ValidatePlatform(updated); err != nil {
		return result, err
	}
	prospective, err := cloneLedger(model)
	if err != nil {
		return result, err
	}
	prospective.Platforms[id] = clonePlatform(updated)
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return result, err
	}
	result.Ledger = prospective
	if reflect.DeepEqual(original, updated) {
		return result, nil
	}
	value, err := jsonPatchValue(updated)
	if err != nil {
		return result, err
	}
	result.Patches = []document.PatchOperation{{Kind: document.PatchReplace, Pointer: "/platforms/" + string(id), Value: value}}
	return result, nil
}

func BuildPlatformRemove(model *ledger.Ledger, id ledger.Slug) (PlatformMutation, error) {
	var result PlatformMutation
	if model == nil {
		return result, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	if err := ValidatePlatformID(id); err != nil {
		return result, err
	}
	if _, exists := model.Platforms[id]; !exists {
		return result, app.WithDetails(app.NewError(app.CodeNotFound, "platform was not found", nil), map[string]any{"platform_id": id})
	}
	index, err := ledger.BuildIndex(model)
	if err != nil {
		return result, app.NewError(app.CodeInvalidData, "ledger indexes are invalid", err)
	}
	references := append([]ledger.Slug(nil), index.PlatformQuestionIDs[id]...)
	if len(references) > 0 {
		sort.Slice(references, func(i, j int) bool { return references[i] < references[j] })
		return result, app.WithDetails(app.NewError(app.CodeConflict, "platform is still referenced by questions", nil), map[string]any{"platform_id": id, "question_ids": references})
	}
	prospective, err := cloneLedger(model)
	if err != nil {
		return result, err
	}
	delete(prospective.Platforms, id)
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return result, err
	}
	result.Ledger = prospective
	result.Patches = []document.PatchOperation{{Kind: document.PatchRemove, Pointer: "/platforms/" + string(id)}}
	return result, nil
}

func ListPlatforms(model *ledger.Ledger) ([]PlatformListItem, error) {
	if model == nil {
		return nil, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	index, err := ledger.BuildIndex(model)
	if err != nil {
		return nil, app.NewError(app.CodeInvalidData, "ledger indexes are invalid", err)
	}
	ids := make([]ledger.Slug, 0, len(model.Platforms))
	for id := range model.Platforms {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]PlatformListItem, len(ids))
	for position, id := range ids {
		platform := model.Platforms[id]
		result[position] = PlatformListItem{ID: id, Name: platform.Name, Kind: platform.Kind, ReferenceCount: len(index.PlatformQuestionIDs[id])}
	}
	return result, nil
}

func ShowPlatform(model *ledger.Ledger, id ledger.Slug) (PlatformShowResult, error) {
	if model == nil {
		return PlatformShowResult{}, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	platform, exists := model.Platforms[id]
	if !exists {
		return PlatformShowResult{}, app.WithDetails(app.NewError(app.CodeNotFound, "platform was not found", nil), map[string]any{"platform_id": id})
	}
	index, err := ledger.BuildIndex(model)
	if err != nil {
		return PlatformShowResult{}, app.NewError(app.CodeInvalidData, "ledger indexes are invalid", err)
	}
	references := append([]ledger.Slug(nil), index.PlatformQuestionIDs[id]...)
	sort.Slice(references, func(i, j int) bool { return references[i] < references[j] })
	return PlatformShowResult{ID: id, Platform: clonePlatform(platform), ReferencingQuestionIDs: references}, nil
}

func applyOptionalString(target **string, input Optional[string]) {
	if !input.Set {
		return
	}
	if input.Null {
		*target = nil
		return
	}
	value := input.Value
	*target = &value
}

func ValidatePlatformID(value ledger.Slug) error {
	if strings.TrimSpace(string(value)) == "" {
		return invalidField("platform", "platform ID is required")
	}
	return ValidateSlug(value, "platform")
}
