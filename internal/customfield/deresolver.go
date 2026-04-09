package customfield

import (
	"context"
	"errors"
	"strings"
	"time"

	commons "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	"github.com/coscene-io/cocli/api"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/structpb"
)

type Deresolver struct {
	userCli api.UserInterface
}

func NewDeresolver(userCli api.UserInterface) *Deresolver {
	return &Deresolver{userCli: userCli}
}

func (d *Deresolver) Deresolve(ctx context.Context, customFieldValues []*commons.CustomFieldValue) []*structpb.Value {
	return lo.Map(customFieldValues, func(customFieldValue *commons.CustomFieldValue, _ int) *structpb.Value {
		customFieldStruct, _ := d.getCustomFieldStruct(ctx, customFieldValue)
		return structpb.NewStructValue(customFieldStruct)
	})
}

func (d *Deresolver) getCustomFieldStruct(ctx context.Context, customFieldValue *commons.CustomFieldValue) (*structpb.Struct, error) {
	switch customFieldValue.Property.GetType().(type) {
	case *commons.Property_Text:
		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    customFieldValue.GetText().Value,
		})
	case *commons.Property_Number:
		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    customFieldValue.GetNumber().Value,
		})
	case *commons.Property_Enums:
		if customFieldValue.Property.GetEnums().Multiple {
			enumNames := lo.Map(customFieldValue.GetEnums().Ids, func(id string, _ int) string {
				if v, ok := customFieldValue.Property.GetEnums().Values[id]; ok {
					return v
				}
				// Fallback to id if display name not found
				return id
			})
			return structpb.NewStruct(map[string]interface{}{
				"property": customFieldValue.Property.Name,
				"value":    strings.Join(enumNames, ", "),
			})
		} else {
			var enumName string
			if v, ok := customFieldValue.Property.GetEnums().Values[customFieldValue.GetEnums().Id]; ok {
				enumName = v
			} else {
				// Fallback to id if display name not found
				enumName = customFieldValue.GetEnums().Id
			}
			return structpb.NewStruct(map[string]interface{}{
				"property": customFieldValue.Property.Name,
				"value":    enumName,
			})
		}
	case *commons.Property_Time:
		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    customFieldValue.GetTime().Value.AsTime().Format(time.RFC3339),
		})
	case *commons.Property_User:
		userNames := make([]string, 0, len(customFieldValue.GetUser().Ids))
		for _, id := range customFieldValue.GetUser().Ids {
			user, err := d.userCli.GetUser(ctx, name.User{UserID: id}.String())
			if err != nil {
				logrus.Errorf("failed to get user: %v", err)
				return nil, err
			}
			userNames = append(userNames, *user.Nickname)
		}

		return structpb.NewStruct(map[string]interface{}{
			"property": customFieldValue.Property.Name,
			"value":    strings.Join(userNames, ", "),
		})
	default:
		return nil, errors.New("unknown custom field type")
	}
}
