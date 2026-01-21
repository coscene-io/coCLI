// Copyright 2024 coScene
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	openv1alpha1connect "buf.build/gen/go/coscene-io/coscene-openapi/connectrpc/go/coscene/openapi/dataplatform/v1alpha1/services/servicesconnect"
	"buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/commons"
	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	openv1alpha1service "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/services"
	"connectrpc.com/connect"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"google.golang.org/genproto/protobuf/field_mask"
)

type RecordInterface interface {
	// Get gets a record by name.
	Get(ctx context.Context, recordName *name.Record) (*openv1alpha1resource.Record, error)

	// Create creates a record.
	Create(ctx context.Context, parent *name.Project, title string, deviceNameStr string, description string, labelDisplayNames []*openv1alpha1resource.Label) (*openv1alpha1resource.Record, error)

	// Copy copies a record to target project.
	Copy(ctx context.Context, recordName *name.Record, targetProjectName *name.Project) (*openv1alpha1resource.Record, error)

	// Move moves a record to target project.
	Move(ctx context.Context, recordName *name.Record, targetProjectName *name.Project) (*openv1alpha1resource.Record, error)

	// CopyFiles copies files from src record to dst record.
	CopyFiles(ctx context.Context, srcRecordName *name.Record, dstRecordName *name.Record, files []*openv1alpha1resource.File) error

	// MoveFiles moves files from src record to dst record.
	MoveFiles(ctx context.Context, srcRecordName *name.Record, dstRecordName *name.Record, files []*openv1alpha1resource.File) error

	// ListAllFiles lists all files in a record.
	ListAllFiles(ctx context.Context, recordName *name.Record) ([]*openv1alpha1resource.File, error)

	// ListAllFilesWithFilter lists all files in a record with additional filter.
	ListAllFilesWithFilter(ctx context.Context, recordName *name.Record, additionalFilter string) ([]*openv1alpha1resource.File, error)

	// ListFilesWithPagination lists files in a record with pagination support.
	ListFilesWithPagination(ctx context.Context, recordName *name.Record, pageSize int, skip int) ([]*openv1alpha1resource.File, error)

	// ListFilesWithPaginationAndFilter lists files in a record with pagination support and additional filter.
	ListFilesWithPaginationAndFilter(ctx context.Context, recordName *name.Record, pageSize int, skip int, additionalFilter string) ([]*openv1alpha1resource.File, error)

	// Delete deletes a record by name.
	Delete(ctx context.Context, recordName *name.Record) error

	// DeleteFile deletes a file by name.
	DeleteFile(ctx context.Context, recordName *name.Record, fileName string) error

	// Update updates a record.
	Update(ctx context.Context, recordName *name.Record, title string, description string, labels []*openv1alpha1resource.Label, fieldMask []string) error

	// ListAllEvents lists all events in a record.
	ListAllEvents(ctx context.Context, recordName *name.Record) ([]*openv1alpha1resource.Event, error)

	// ListAllMoments lists all moments in a record.
	ListAllMoments(ctx context.Context, recordName *name.Record) ([]*Moment, error)

	// SearchAll searches all records in a project using the new SearchRecords API.
	SearchAll(ctx context.Context, options *SearchRecordsOptions) ([]*openv1alpha1resource.Record, error)

	// SearchWithPageToken searches records with page token support (new API).
	SearchWithPageToken(ctx context.Context, options *SearchRecordsOptions) (*SearchRecordsResult, error)

	// GenerateRecordThumbnailUploadUrl generates a pre-signed URL for uploading a record thumbnail.
	GenerateRecordThumbnailUploadUrl(ctx context.Context, recordName *name.Record) (string, error)

	// RecordId2Name converts a record id or name to a record name.
	RecordId2Name(ctx context.Context, recordIdOrName string, projectNameStr *name.Project) (*name.Record, error)
}

type SearchRecordsOptions struct {
	Project        *name.Project
	Titles         []string
	Labels         []string
	IncludeArchive bool
	PageSize       int32
	PageToken      string
	OrderBy        string
}

type SearchRecordsResult struct {
	Records       []*openv1alpha1resource.Record
	NextPageToken string
	TotalSize     int64
}

type recordClient struct {
	recordServiceClient openv1alpha1connect.RecordServiceClient
	fileServiceClient   openv1alpha1connect.FileServiceClient
	userServiceClient   openv1alpha1connect.UserServiceClient
	labelServiceClient  openv1alpha1connect.LabelServiceClient
}

type Moment struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	TriggerTime       string            `json:"triggerTime"`
	Duration          string            `json:"duration"`
	Attribute         map[string]string `json:"attribute"`
	CustomFieldValues []map[string]any  `json:"customFieldValues"`
}

func NewRecordClient(recordServiceClient openv1alpha1connect.RecordServiceClient, fileServiceClient openv1alpha1connect.FileServiceClient, userServiceClient openv1alpha1connect.UserServiceClient, labelServiceClient openv1alpha1connect.LabelServiceClient) RecordInterface {
	return &recordClient{
		recordServiceClient: recordServiceClient,
		fileServiceClient:   fileServiceClient,
		userServiceClient:   userServiceClient,
		labelServiceClient:  labelServiceClient,
	}
}

func (c *recordClient) Get(ctx context.Context, recordName *name.Record) (*openv1alpha1resource.Record, error) {
	getRecordReq := connect.NewRequest(&openv1alpha1service.GetRecordRequest{
		Name: recordName.String(),
	})
	getRecordRes, err := c.recordServiceClient.GetRecord(ctx, getRecordReq)
	if err != nil {
		return nil, err
	}
	return getRecordRes.Msg, nil
}

func (c *recordClient) Create(ctx context.Context, parent *name.Project, title string, deviceNameStr string, description string, labels []*openv1alpha1resource.Label) (*openv1alpha1resource.Record, error) {
	var (
		device *openv1alpha1resource.Device = nil
	)
	if len(deviceNameStr) > 0 {
		device = &openv1alpha1resource.Device{Name: deviceNameStr}
	}

	req := connect.NewRequest(&openv1alpha1service.CreateRecordRequest{
		Parent: parent.String(),
		Record: &openv1alpha1resource.Record{
			Title:       title,
			Description: description,
			Device:      device,
			Labels:      labels,
		},
	})
	resp, err := c.recordServiceClient.CreateRecord(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Msg, err
}

func (c *recordClient) Copy(ctx context.Context, recordName *name.Record, targetProjectName *name.Project) (*openv1alpha1resource.Record, error) {
	req := connect.NewRequest(&openv1alpha1service.CopyRecordsRequest{
		Parent:      recordName.Project().String(),
		Destination: targetProjectName.String(),
		Records:     []string{recordName.String()},
	})
	resp, err := c.recordServiceClient.CopyRecords(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Msg.Records) != 1 {
		return nil, errors.Errorf("unexpected number of records in response: %d", len(resp.Msg.Records))
	}
	return resp.Msg.Records[0], nil
}

func (c *recordClient) CopyFiles(ctx context.Context, srcRecordName *name.Record, dstRecordName *name.Record, files []*openv1alpha1resource.File) error {
	copyPairs := lo.Map(files, func(file *openv1alpha1resource.File, _ int) *openv1alpha1service.CopyFilesRequest_CopyPair {
		return &openv1alpha1service.CopyFilesRequest_CopyPair{
			SrcFile: file.GetFilename(),
			DstFile: file.GetFilename(),
		}
	})

	req := connect.NewRequest(&openv1alpha1service.CopyFilesRequest{
		Parent:      srcRecordName.String(),
		Destination: dstRecordName.String(),
		CopyPairs:   copyPairs,
	})

	res, err := c.fileServiceClient.CopyFiles(ctx, req)
	if err != nil {
		return err
	}
	// TODO: The matrix server did not handle the copied files in the response correctly.
	// 	 We will be able to check the Files field after the server is updated.
	if res.Msg != nil {
		if res.Msg.Files != nil && len(res.Msg.Files) == len(files) {
			// Server returned copied files in response (ideal case)
			return nil
		}
		// Server did not return copied files in response (current behavior)
		return nil
	}
	return nil
}

func (c *recordClient) Move(ctx context.Context, recordName *name.Record, targetProjectName *name.Project) (*openv1alpha1resource.Record, error) {
	req := connect.NewRequest(&openv1alpha1service.MoveRecordsRequest{
		Parent:      recordName.Project().String(),
		Destination: targetProjectName.String(),
		Records:     []string{recordName.String()},
	})
	resp, err := c.recordServiceClient.MoveRecords(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Msg.Records) != 1 {
		return nil, errors.Errorf("unexpected number of records in response: %d", len(resp.Msg.Records))
	}
	return resp.Msg.Records[0], nil
}

func (c *recordClient) MoveFiles(ctx context.Context, srcRecordName *name.Record, dstRecordName *name.Record, files []*openv1alpha1resource.File) error {
	movePairs := lo.Map(files, func(file *openv1alpha1resource.File, _ int) *openv1alpha1service.MoveFilesRequest_MovePair {
		return &openv1alpha1service.MoveFilesRequest_MovePair{
			SrcFile: file.GetFilename(),
			DstFile: file.GetFilename(),
		}
	})

	req := connect.NewRequest(&openv1alpha1service.MoveFilesRequest{
		Parent:      srcRecordName.String(),
		Destination: dstRecordName.String(),
		MovePairs:   movePairs,
	})

	res, err := c.fileServiceClient.MoveFiles(ctx, req)
	if err != nil {
		return err
	}
	// TODO: The matrix server did not handle the moved files in the response correctly.
	// 	 We will be able to check the Files field after the server is updated.
	if res.Msg != nil {
		if res.Msg.Files != nil && len(res.Msg.Files) == len(files) {
			// Server returned moved files in response (ideal case)
			return nil
		}
		// Server did not return moved files in response (current behavior)
		return nil
	}
	return nil
}

func (c *recordClient) ListAllFiles(ctx context.Context, recordName *name.Record) ([]*openv1alpha1resource.File, error) {
	return c.listAllFilesWithFilter(ctx, recordName, "")
}

func (c *recordClient) ListAllFilesWithFilter(ctx context.Context, recordName *name.Record, additionalFilter string) ([]*openv1alpha1resource.File, error) {
	return c.listAllFilesWithFilter(ctx, recordName, additionalFilter)
}

func (c *recordClient) listAllFilesWithFilter(ctx context.Context, recordName *name.Record, additionalFilter string) ([]*openv1alpha1resource.File, error) {
	return c.listFilesCore(ctx, recordName, 0, 0, additionalFilter, true)
}

func (c *recordClient) ListFilesWithPagination(ctx context.Context, recordName *name.Record, pageSize int, skip int) ([]*openv1alpha1resource.File, error) {
	return c.listFilesWithPaginationAndFilter(ctx, recordName, pageSize, skip, "")
}

func (c *recordClient) ListFilesWithPaginationAndFilter(ctx context.Context, recordName *name.Record, pageSize int, skip int, additionalFilter string) ([]*openv1alpha1resource.File, error) {
	return c.listFilesWithPaginationAndFilter(ctx, recordName, pageSize, skip, additionalFilter)
}

func (c *recordClient) listFilesWithPaginationAndFilter(ctx context.Context, recordName *name.Record, pageSize int, skip int, additionalFilter string) ([]*openv1alpha1resource.File, error) {
	return c.listFilesCore(ctx, recordName, pageSize, skip, additionalFilter, false)
}

// listFilesCore is an internal helper that lists files either across all pages
// (all=true) or a single page (all=false). It returns raw results including
// directories. Callers can filter directories as needed.
func (c *recordClient) listFilesCore(ctx context.Context, recordName *name.Record, pageSize int, skip int, additionalFilter string, all bool) ([]*openv1alpha1resource.File, error) {
	filter := additionalFilter

	if all {
		var (
			ret  []*openv1alpha1resource.File
			offs = skip
		)
		for {
			req := connect.NewRequest(&openv1alpha1service.ListFilesRequest{
				Parent:   recordName.String(),
				PageSize: constants.MaxPageSize,
				Skip:     int32(offs),
				Filter:   filter,
			})
			res, err := c.fileServiceClient.ListFiles(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("failed to list files at skip %d: %w", offs, err)
			}

			ret = append(ret, res.Msg.Files...)
			offs += constants.MaxPageSize
			if offs >= int(res.Msg.TotalSize) {
				break
			}
		}
		return ret, nil
	}

	req := connect.NewRequest(&openv1alpha1service.ListFilesRequest{
		Parent:   recordName.String(),
		PageSize: int32(pageSize),
		Skip:     int32(skip),
		Filter:   filter,
	})
	res, err := c.fileServiceClient.ListFiles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return res.Msg.Files, nil
}

func (c *recordClient) Delete(ctx context.Context, recordName *name.Record) error {
	deleteRecordReq := connect.NewRequest(&openv1alpha1service.DeleteRecordRequest{
		Name: recordName.String(),
	})
	_, err := c.recordServiceClient.DeleteRecord(ctx, deleteRecordReq)
	return err
}

func (c *recordClient) DeleteFile(ctx context.Context, recordName *name.Record, fileName string) error {
	deleteFileReq := connect.NewRequest(&openv1alpha1service.DeleteFileRequest{
		Name: name.File{
			ProjectID: recordName.ProjectID,
			RecordID:  recordName.RecordID,
			Filename:  fileName,
		}.String(),
	})
	_, err := c.fileServiceClient.DeleteFile(ctx, deleteFileReq)
	return err
}

func (c *recordClient) Update(ctx context.Context, recordName *name.Record, title string, description string, labels []*openv1alpha1resource.Label, fieldMask []string) error {
	req := connect.NewRequest(&openv1alpha1service.UpdateRecordRequest{
		Record: &openv1alpha1resource.Record{
			Name:        recordName.String(),
			Title:       title,
			Description: description,
			Labels:      labels,
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: fieldMask,
		},
	})
	_, err := c.recordServiceClient.UpdateRecord(ctx, req)
	return err
}

func (c *recordClient) ListAllEvents(ctx context.Context, recordName *name.Record) ([]*openv1alpha1resource.Event, error) {
	var (
		skip = 0
		ret  []*openv1alpha1resource.Event
	)

	for {
		req := connect.NewRequest(&openv1alpha1service.ListRecordEventsRequest{
			Parent:   recordName.String(),
			PageSize: constants.MaxPageSize,
			Skip:     int32(skip),
			Filter:   "",
		})
		res, err := c.recordServiceClient.ListRecordEvents(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to list events at skip %d: %w", skip, err)
		}
		if len(res.Msg.Events) == 0 {
			break
		}
		ret = append(ret, res.Msg.Events...)
		skip += constants.MaxPageSize
	}

	return ret, nil
}

func (c *recordClient) ListAllMoments(ctx context.Context, recordName *name.Record) ([]*Moment, error) {
	events, err := c.ListAllEvents(ctx, recordName)
	if err != nil {
		return nil, err
	}

	users := []string{}
	var name2User map[string]*openv1alpha1resource.User
	lo.ForEach(events, func(event *openv1alpha1resource.Event, _ int) {
		lo.ForEach(event.CustomFieldValues, func(value *commons.CustomFieldValue, _ int) {
			if _, ok := value.GetProperty().GetType().(*commons.Property_User); ok {
				users = append(users, lo.Map(value.GetUser().GetIds(), func(id string, _ int) string { return name.User{UserID: id}.String() })...)
			}
		})
	})
	if len(users) > 0 {
		res, err := c.userServiceClient.BatchGetUsers(ctx, connect.NewRequest(&openv1alpha1service.BatchGetUsersRequest{Names: users}))
		if err != nil {
			return nil, err
		}
		name2User = lo.SliceToMap(res.Msg.Users, func(user *openv1alpha1resource.User) (string, *openv1alpha1resource.User) {
			return user.Name, user
		})
	}

	return lo.Map(events, func(event *openv1alpha1resource.Event, _ int) *Moment {
		customFields := make([]map[string]any, 0, len(event.CustomFieldValues))
		lo.ForEach(event.CustomFieldValues, func(value *commons.CustomFieldValue, _ int) {
			switch value.GetProperty().GetType().(type) {
			case *commons.Property_Text:
				customFields = append(customFields, map[string]any{
					value.GetProperty().GetName(): value.GetText().GetValue(),
				})
			case *commons.Property_Number:
				customFields = append(customFields, map[string]any{
					value.GetProperty().GetName(): value.GetNumber().GetValue(),
				})
			case *commons.Property_Enums:
				customFields = append(customFields, map[string]any{
					value.GetProperty().GetName(): value.GetProperty().GetType().(*commons.Property_Enums).Enums.GetValues()[value.GetEnums().GetId()],
				})
			case *commons.Property_Time:
				customFields = append(customFields, map[string]any{
					value.GetProperty().GetName(): value.GetTime().GetValue().AsTime().In(time.Local).Format(time.RFC3339Nano),
				})
			case *commons.Property_User:
				customFields = append(customFields, map[string]any{
					value.GetProperty().GetName(): lo.Map(value.GetUser().GetIds(), func(id string, _ int) string { return *name2User[name.User{UserID: id}.String()].Nickname }),
				})
			}
		})
		return &Moment{
			Name:              event.DisplayName,
			Description:       event.Description,
			TriggerTime:       event.TriggerTime.AsTime().In(time.Local).Format(time.RFC3339Nano),
			Duration:          fmt.Sprintf("%.9fs", event.Duration.AsDuration().Seconds()),
			Attribute:         lo.If(event.CustomizedFields != nil, event.CustomizedFields).Else(map[string]string{}),
			CustomFieldValues: customFields,
		}
	}), nil
}

func (c *recordClient) SearchAll(ctx context.Context, options *SearchRecordsOptions) ([]*openv1alpha1resource.Record, error) {
	if options.Project.ProjectID == "" {
		return nil, errors.Errorf("invalid project: %s", options.Project)
	}

	filter := c.buildSearchFilter(options)
	var (
		pageToken = ""
		ret       []*openv1alpha1resource.Record
	)

	for {
		req := connect.NewRequest(&openv1alpha1service.SearchRecordsRequest{
			Parent:    options.Project.String(),
			PageSize:  constants.MaxPageSize,
			PageToken: pageToken,
			OrderBy:   options.OrderBy,
		})

		if filter != "" {
			req.Msg.QueryFilter = &openv1alpha1service.SearchRecordsRequest_Filter{
				Filter: filter,
			}
		}

		res, err := c.recordServiceClient.SearchRecords(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to search records: %w", err)
		}

		ret = append(ret, res.Msg.Records...)

		isEmpty := len(res.Msg.Records) == 0
		isLastPage := isEmpty || len(res.Msg.Records) < constants.MaxPageSize || res.Msg.NextPageToken == ""
		if isLastPage {
			break
		}
		pageToken = res.Msg.NextPageToken
	}

	return ret, nil
}

func (c *recordClient) SearchWithPageToken(ctx context.Context, options *SearchRecordsOptions) (*SearchRecordsResult, error) {
	if options.Project.ProjectID == "" {
		return nil, errors.Errorf("invalid project: %s", options.Project)
	}

	filter := c.buildSearchFilter(options)

	req := connect.NewRequest(&openv1alpha1service.SearchRecordsRequest{
		Parent:    options.Project.String(),
		PageSize:  options.PageSize,
		PageToken: options.PageToken,
		OrderBy:   options.OrderBy,
	})

	if filter != "" {
		req.Msg.QueryFilter = &openv1alpha1service.SearchRecordsRequest_Filter{
			Filter: filter,
		}
	}

	res, err := c.recordServiceClient.SearchRecords(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search records: %w", err)
	}

	return &SearchRecordsResult{
		Records:       res.Msg.Records,
		NextPageToken: res.Msg.NextPageToken,
		TotalSize:     res.Msg.TotalSize,
	}, nil
}

// buildSearchFilter builds the filter string for the SearchRecords API using AIP-160 syntax.
func (c *recordClient) buildSearchFilter(opts *SearchRecordsOptions) string {
	var filters []string
	if !opts.IncludeArchive {
		filters = append(filters, "isArchived = false")
	}
	if len(opts.Titles) > 0 {
		quotedTitles := lo.Map(opts.Titles, func(title string, _ int) string {
			return fmt.Sprintf(`"%s"`, title)
		})
		filters = append(filters, fmt.Sprintf("title in [%s]", strings.Join(quotedTitles, ", ")))
	}
	if len(opts.Labels) > 0 {
		labelIDs, err := c.transformLabelsToIDs(context.TODO(), opts.Project.String(), opts.Labels)
		if err != nil {
			fmt.Printf("Warning: failed to lookup labels: %v\n", err)
		} else if len(labelIDs) > 0 {
			quotedIDs := lo.Map(labelIDs, func(id string, _ int) string {
				return fmt.Sprintf(`"%s"`, id)
			})
			filters = append(filters, fmt.Sprintf("relatedLabels.id in [%s]", strings.Join(quotedIDs, ", ")))
		}
	}
	return strings.Join(filters, " AND ")
}

func (c *recordClient) GenerateRecordThumbnailUploadUrl(ctx context.Context, recordName *name.Record) (string, error) {
	req := connect.NewRequest(&openv1alpha1service.GenerateRecordThumbnailUploadUrlRequest{
		Record: recordName.String(),
	})
	resp, err := c.recordServiceClient.GenerateRecordThumbnailUploadUrl(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Msg.PreSignedUri, nil
}

func (c *recordClient) RecordId2Name(ctx context.Context, recordIdOrName string, projectName *name.Project) (*name.Record, error) {
	recordName, err := name.NewRecord(recordIdOrName)
	if err == nil {
		return recordName, nil
	}

	recordName = &name.Record{
		ProjectID: projectName.ProjectID,
		RecordID:  recordIdOrName,
	}

	if _, err := c.Get(ctx, recordName); err != nil {
		return nil, errors.Wrapf(err, "unable to get record: %s", recordName.String())
	}

	return recordName, nil
}

// transformLabelsToIDs converts label display names to just IDs
func (c *recordClient) transformLabelsToIDs(ctx context.Context, parent string, labelNames []string) ([]string, error) {
	if len(labelNames) == 0 {
		return nil, nil
	}

	// Create a map to track label display name to ID mapping
	labelNameIdMap := make(map[string]string)
	for _, labelName := range labelNames {
		labelNameIdMap[labelName] = ""
	}

	// List labels from platform server using batch lookup
	req := connect.NewRequest(&openv1alpha1service.ListLabelsRequest{
		Parent:   parent,
		Filter:   fmt.Sprintf("display_name=[%s]", strings.Join(labelNames, `,`)),
		PageSize: constants.MaxPageSize,
	})
	labelResp, err := c.labelServiceClient.ListLabels(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}

	// Extract label IDs from the response
	for _, label := range labelResp.Msg.Labels {
		labelNameArr := strings.Split(label.Name, `/`)
		labelId := labelNameArr[len(labelNameArr)-1]
		labelNameIdMap[label.DisplayName] = labelId
	}

	// Check for missing labels
	var missedLabelNames []string
	for labelName, labelId := range labelNameIdMap {
		if labelId == "" {
			missedLabelNames = append(missedLabelNames, labelName)
		}
	}
	if len(missedLabelNames) > 0 {
		return nil, fmt.Errorf("labels not found: %v", strings.Join(missedLabelNames, ","))
	}

	// Return just the IDs in the same order as input
	var ret []string
	for _, labelName := range labelNames {
		labelId := labelNameIdMap[labelName]
		ret = append(ret, labelId)
	}

	return ret, nil
}
