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

package upload_utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	openv1alpha1resource "buf.build/gen/go/coscene-io/coscene-openapi/protocolbuffers/go/coscene/openapi/dataplatform/v1alpha1/resources"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/coscene-io/cocli/internal/constants"
	"github.com/coscene-io/cocli/internal/fs"
	"github.com/coscene-io/cocli/internal/name"
	"github.com/coscene-io/cocli/internal/utils"
	"github.com/coscene-io/cocli/pkg/cmd_utils"
	"github.com/getsentry/sentry-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/muesli/reflow/wordwrap"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
)

const (
	userTagRecordIdKey     = "X-COS-RECORD-ID"
	userTagProjectIdKey    = "X-COS-PROJECT-ID"
	mutipartUploadInfoKey  = "STORE-KEY-MUTIPART-UPLOAD-INFO"
	maxSinglePutObjectSize = 1024 * 1024 * 1024 * 500 // 500GiB
	defaultWindowSize      = 1024 * 1024 * 1024       // 1GiB
	processBatchSize       = 20
)

var (
	spinnerFrames = []string{"⣾", "⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽"}
)

// UploadStatusEnum is used to keep track of the state of a file upload
type UploadStatusEnum int

const (
	// Unprocessed is used to indicate that the file has not been processed yet
	Unprocessed UploadStatusEnum = iota

	// CalculatingSha256 is used to indicate that the file sha256 is being calculated
	CalculatingSha256

	// PreviouslyUploaded is used to indicate that the file has been uploaded before
	PreviouslyUploaded

	// WaitingForUpload is used to indicate that the file is waiting to be uploaded
	WaitingForUpload

	// UploadInProgress is used to indicate that the file upload is in progress
	UploadInProgress

	// UploadCompleted is used to indicate that the file upload has completed
	UploadCompleted

	// MultipartCompletionInProgress is used to indicate that the multipart upload completion is in progress
	MultipartCompletionInProgress

	// UploadFailed is used to indicate that the file upload has failed
	UploadFailed
)

// FileInfo contains the path, size and sha256 of a file.
type FileInfo struct {
	Path       string // Local absolute path
	RemotePath string // Remote destination path (for display)
	Size       int64
	Sha256     string
	Uploaded   int64
	Status     UploadStatusEnum
}

// UploadInfo contains the information needed to upload a file or a file part (multipart upload).
type UploadInfo struct {
	Path       string
	Bucket     string
	Key        string
	Tags       map[string]string
	FileReader *os.File

	// Upload result infos
	Result minio.ObjectPart
	Err    error

	// Multipart info
	UploadId        string
	PartId          int
	TotalPartsCount int
	ReadOffset      int64
	ReadSize        int64
	DB              *UploadDB
}

// MultipartCheckpointInfo contains the information needed to resume a multipart upload.
type MultipartCheckpointInfo struct {
	UploadId     string               `json:"upload_id"`
	UploadedSize int64                `json:"uploaded_size"`
	Parts        []minio.CompletePart `json:"parts"`
}

// IncUploadedMsg is used to send incremental uploaded size to the progress update goroutine
type IncUploadedMsg struct {
	Path        string
	UploadedInc int64
}

// UploadManager is a manager for uploading files through minio client.
type UploadManager struct {
	// client and opts
	opts    *UploadManagerOpts
	apiOpts *ApiOpts
	client  *minio.Client

	// file status related
	fileInfos  map[string]*FileInfo
	fileList   []string
	uploadWg   sync.WaitGroup
	progressCh chan IncUploadedMsg

	// Monitor related
	windowWidth int
	spinnerIdx  int
	manualQuit  bool
	monitor     *tea.Program
	noTTY       bool

	// other
	errs    map[string]error
	isDebug bool
}

// parentContext abstracts record/project parent for upload.
type parentContext struct {
	parentString      string
	buildResourceName func(relativePath string) string
}

// UploadParent is a public abstraction for upload destination (record or project).
type UploadParent interface {
	ParentString() string
	BuildResourceName(relativePath string) string
}

// RecordParent implements UploadParent for record-level uploads.
type RecordParent struct{ R *name.Record }

func NewRecordParent(r *name.Record) RecordParent { return RecordParent{R: r} }

func (rp RecordParent) ParentString() string { return rp.R.String() }
func (rp RecordParent) BuildResourceName(relativePath string) string {
	return name.File{ProjectID: rp.R.ProjectID, RecordID: rp.R.RecordID, Filename: relativePath}.String()
}

// ProjectParent implements UploadParent for project-level uploads.
type ProjectParent struct{ P *name.Project }

func NewProjectParent(p *name.Project) ProjectParent { return ProjectParent{P: p} }

func (pp ProjectParent) ParentString() string { return pp.P.String() }
func (pp ProjectParent) BuildResourceName(relativePath string) string {
	return name.ProjectFile{ProjectID: pp.P.ProjectID, Filename: relativePath}.String()
}

// removed old helpers; prefer UploadParent and newParentContextFrom

func newParentContextFrom(up UploadParent) parentContext {
	return parentContext{
		parentString:      up.ParentString(),
		buildResourceName: func(relativePath string) string { return up.BuildResourceName(relativePath) },
	}
}

func NewUploadManagerFromConfig(proj *name.Project, timeout time.Duration, apiOpts *ApiOpts, opts *UploadManagerOpts) (*UploadManager, error) {
	if err := opts.Valid(); err != nil {
		return nil, errors.Wrap(err, "invalid multipart options")
	}
	generateSecurityTokenRes, err := apiOpts.GenerateSecurityToken(context.Background(), proj.String())
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate security token")
	}
	mc, err := minio.New(generateSecurityTokenRes.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(generateSecurityTokenRes.GetAccessKeyId(), generateSecurityTokenRes.GetAccessKeySecret(), generateSecurityTokenRes.GetSessionToken()),
		Secure:    true,
		Region:    "",
		Transport: cmd_utils.NewTransport(timeout),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create minio client")
	}

	// Determine if we should use interactive mode
	useInteractive := opts.ShouldUseInteractiveMode()

	um := &UploadManager{
		opts:       opts,
		apiOpts:    apiOpts,
		client:     mc,
		isDebug:    log.GetLevel() == log.DebugLevel,
		fileInfos:  make(map[string]*FileInfo),
		fileList:   []string{},
		progressCh: make(chan IncUploadedMsg, 5000), // buffer the channel to avoid blocking
		errs:       make(map[string]error),
		noTTY:      !useInteractive,
	}

	// Log the mode detection
	if !opts.NoTTY && !opts.TTY && IsHeadlessEnvironment() {
		log.Info("Detected headless environment, automatically using non-interactive mode")
	}

	// Only create tea.Program if we're using interactive mode
	if useInteractive {
		um.monitor = tea.NewProgram(um)
	}

	return um, nil
}

// Run is used to start the upload process.
func (um *UploadManager) Run(ctx context.Context, parent UploadParent, fileOpts *FileOpts) error {
	if err := fileOpts.Valid(); err != nil {
		return err
	}

	if um.noTTY {
		// Non-interactive mode - use simple logging
		log.Info("Starting upload in non-interactive mode...")
	} else {
		// Start the status monitor
		um.goWithSentry("upload status monitor", func(_ *sentry.Hub) {
			_, err := um.monitor.Run()
			if err != nil {
				log.Fatalf("Error running upload status monitor: %v", err)
			}
			if um.manualQuit {
				log.Fatalf("Upload quit manually")
			}
		})
	}

	// Start the progress monitor
	go func() {
		for {
			msg := <-um.progressCh
			um.fileInfos[msg.Path].Uploaded += msg.UploadedInc
		}
	}()

	// Send an empty message to wait for the monitor to start (only if we have monitor)
	if !um.noTTY {
		um.monitor.Send(struct{}{})
	}

	// Only enable trace logging if the log level is set to trace
	if log.GetLevel() == log.TraceLevel {
		um.client.TraceOn(log.StandardLogger().WriterLevel(log.TraceLevel))
	}

	var filesToUpload []string
	for _, path := range fileOpts.GetPaths() {
		filesToUpload = append(filesToUpload, fs.FindFiles(path, fileOpts.Recursive, fileOpts.IncludeHidden)...)
	}
	um.uploadWg.Add(len(filesToUpload) + len(fileOpts.AdditionalUploads))

	fileToUploadUrls := um.findAllUploadUrlsGeneric(filesToUpload, newParentContextFrom(parent), fileOpts.RelDir(), fileOpts.Prefix)
	for f, v := range fileOpts.AdditionalUploads {
		fileToUploadUrls[f] = v
		um.addFile(f)
		checksum, size, err := fs.CalSha256AndSize(f)
		if err != nil {
			um.addErr(f, errors.Wrapf(err, "unable to calculate sha256 for file"))
			continue
		}
		um.fileInfos[f].Size = size
		um.fileInfos[f].Sha256 = checksum
	}

	// Declare a channel that sends the upload infos for each file to be uploaded
	uploadInfos := make(chan UploadInfo)
	// Declare a channel that sends the next upload info to be processed
	uploadCh := make(chan UploadInfo)
	// Declare a channel that receives the result of the upload
	uploadResultCh := make(chan UploadInfo)

	// Start the upload workers
	for i := 0; i < um.opts.Threads; i++ {
		um.goWithSentry(fmt.Sprintf("upload worker %d", i), func(_ *sentry.Hub) {
			defer func() {
				um.debugF("Worker %d stopped", i)
			}()

			for uploadInfo := range uploadCh {
				um.debugF("Worker %d received upload task with path: %s, part id: %d", i, uploadInfo.Path, uploadInfo.PartId)
				if uploadInfo.UploadId == "" {
					uploadInfo.Err = um.consumeSingleUploadInfo(ctx, uploadInfo)
				} else {
					uploadInfo.Result, uploadInfo.Err = um.consumeMultipartUploadInfo(ctx, uploadInfo)
				}

				uploadResultCh <- uploadInfo
			}
		})
	}

	// Start the producer and scheduler
	um.goWithSentry("upload producer", func(_ *sentry.Hub) {
		um.produceUploadInfos(ctx, fileToUploadUrls, uploadInfos)
	})
	um.goWithSentry("upload scheduler", func(_ *sentry.Hub) {
		um.scheduleUploads(ctx, uploadInfos, uploadCh, uploadResultCh, um.opts.Threads)
	})

	// Start non-interactive progress reporter if in no-TTY mode
	if um.noTTY {
		um.goWithSentry("non-interactive progress reporter", func(_ *sentry.Hub) {
			um.nonInteractiveProgressReporter()
		})
	}

	um.uploadWg.Wait()

	// Print final summary in non-interactive mode
	if um.noTTY {
		var totalFiles, completedFiles, failedFiles, skippedFiles int
		for _, fileInfo := range um.fileInfos {
			totalFiles++
			switch fileInfo.Status {
			case UploadCompleted:
				completedFiles++
			case UploadFailed:
				failedFiles++
			case PreviouslyUploaded:
				skippedFiles++
			}
		}

		log.Infof("Upload completed! Total: %d | Success: %d | Failed: %d | Skipped: %d",
			totalFiles, completedFiles, failedFiles, skippedFiles)

		if failedFiles > 0 {
			log.Warn("Some files failed to upload. Check the error messages above.")
		}
	}

	um.stopMonitorAndWait()

	return nil
}

// RunToProject uploads files directly under a project (no record) using project-level file APIs.
// RunToProject has been removed; use Run with UploadParent instead.

// goWithSentry starts a goroutine with sentry error publishing.
// Also stops the monitor and waits for it to finish if an error occurs.
func (um *UploadManager) goWithSentry(routineName string, fn func(*sentry.Hub)) {
	utils.SentryRunOptions{
		RoutineName: routineName,
		OnErrorFn:   um.stopMonitorAndWait,
	}.Run(fn)
}

// stopMonitorAndWait stops the monitor and waits for it to finish.
func (um *UploadManager) stopMonitorAndWait() {
	if !um.noTTY && um.monitor != nil {
		um.monitor.Quit()
		um.monitor.Wait()
	}

	um.printErrs()
	if um.manualQuit {
		log.Fatalf("Upload quit manually")
	}
}

// produceUploadInfos is a producer of upload infos for each file to be uploaded.
func (um *UploadManager) produceUploadInfos(ctx context.Context, fileToUploadUrls map[string]string, uploadInfos chan UploadInfo) {
	defer close(uploadInfos)
	for _, fileAbsolutePath := range um.fileList {
		uploadUrl, ok := fileToUploadUrls[fileAbsolutePath]
		if !ok {
			continue
		}

		bucket, key, tags, err := um.parseUrl(uploadUrl)
		if err != nil {
			um.addErr(fileAbsolutePath, errors.Wrapf(err, "unable to parse upload url"))
			continue
		}

		fileInfo := um.fileInfos[fileAbsolutePath]

		if fileInfo.Size <= int64(um.opts.partSizeUint64) {
			fileReader, err := os.Open(fileAbsolutePath)
			if err != nil {
				um.addErr(fileAbsolutePath, errors.Wrapf(err, "unable to open file"))
				continue
			}

			uploadInfos <- UploadInfo{
				Path:       fileAbsolutePath,
				Bucket:     bucket,
				Key:        key,
				Tags:       tags,
				FileReader: fileReader,
			}
		} else {
			multipartUploadInfo, err := um.produceMultipartUploadInfos(ctx, fileAbsolutePath, bucket, key, tags)
			if err != nil {
				um.addErr(fileAbsolutePath, errors.Wrapf(err, "unable to produce multipart upload infos"))
				continue
			}
			for idx, info := range multipartUploadInfo {
				uploadInfos <- info
				if idx == 0 {
					um.fileInfos[fileAbsolutePath].Status = UploadInProgress
				}
			}
		}
	}
}

func (um *UploadManager) produceMultipartUploadInfos(ctx context.Context, fileAbsolutePath string, bucket string, key string, tags map[string]string) (uploadInfos []UploadInfo, err error) {
	fileInfo := um.fileInfos[fileAbsolutePath]

	// Check for largest object size allowed.
	if fileInfo.Size > int64(maxSinglePutObjectSize) {
		return nil, errors.Errorf("Your proposed upload size '%d' exceeds the maximum allowed object size '%d' for single PUT operation.", fileInfo.Size, maxSinglePutObjectSize)
	}

	// Create uploader directory if not exists
	if err = os.MkdirAll(constants.DefaultUploaderDirPath, 0755); err != nil {
		return nil, errors.Wrap(err, "Create uploader directory failed")
	}

	// Create uploader db
	// Prefer record id if present; otherwise fallback to project id to uniquely key uploads per destination
	recordOrProjectId := tags[userTagRecordIdKey]
	if recordOrProjectId == "" {
		recordOrProjectId = tags[userTagProjectIdKey]
	}
	db, err := NewUploadDB(fileAbsolutePath, recordOrProjectId, fileInfo.Sha256, um.opts.partSizeUint64)
	if err != nil {
		return nil, errors.Wrap(err, "Create uploader db failed")
	}

	c := minio.Core{Client: um.client}
	// ----------------- Start fetching previous upload info from db -----------------
	// Fetch upload id. If not found, initiate a new multipart upload.
	var checkpoint MultipartCheckpointInfo
	if err = db.Get(mutipartUploadInfoKey, &checkpoint); err != nil {
		um.debugF("Get checkpoint failed: %v", err)
		checkpoint = MultipartCheckpointInfo{}
	}

	// Fetch upload id. If not found, initiate a new multipart upload.
	if checkpoint.UploadId != "" {
		um.debugF("Upload id: %s is found in db", checkpoint.UploadId)

		// Check if the upload id is still valid
		result, err := c.ListObjectParts(ctx, bucket, key, checkpoint.UploadId, 0, 2000)
		if err != nil || len(result.ObjectParts) == 0 {
			um.debugF("List object parts by: %s failed: %v", checkpoint.UploadId, err)
			checkpoint.UploadId = ""
			if err = db.Reset(); err != nil {
				return nil, errors.Wrap(err, "Reset db failed")
			}
		} else {
			um.debugF("Upload id: %s is still valid", checkpoint.UploadId)
		}
	}

	if checkpoint.UploadId == "" {
		// first reset checkpoint
		checkpoint = MultipartCheckpointInfo{}
		checkpoint.UploadId, err = c.NewMultipartUpload(ctx, bucket, key, minio.PutObjectOptions{
			UserTags: tags,
			PartSize: um.opts.partSizeUint64,
		})
		if err != nil {
			return nil, errors.Wrap(err, "New multipart upload failed")
		}
	}

	partNumbers := lo.Map(checkpoint.Parts, func(p minio.CompletePart, _ int) int {
		return p.PartNumber
	})
	sort.Ints(partNumbers)
	um.debugF("Get upload id: %s", checkpoint.UploadId)
	um.debugF("Get uploaded size: %d", checkpoint.UploadedSize)
	um.debugF("Get uploaded parts: %v", partNumbers)

	// ----------------- End fetching previous upload info from db -----------------
	// Calculate the optimal parts info for a given size.
	totalPartsCount, partSize, lastPartSize, err := minio.OptimalPartInfo(fileInfo.Size, um.opts.partSizeUint64)
	if err != nil {
		return nil, errors.Wrap(err, "Optimal part info failed")
	}
	um.debugF("Total part: %v, part size: %v, last part size: %v", totalPartsCount, partSize, lastPartSize)

	// Get reader of the file to be uploaded.
	fileReader, err := os.Open(fileAbsolutePath)
	if err != nil {
		return nil, errors.Wrap(err, "Open file failed")
	}

	// Compute remaining parts to upload.
	for partId := 1; partId <= totalPartsCount; partId++ {
		if slices.Contains(partNumbers, partId) {
			continue
		}

		readSize := partSize
		if partId == totalPartsCount {
			readSize = lastPartSize
		}
		uploadInfos = append(uploadInfos, UploadInfo{
			Path:            fileAbsolutePath,
			Bucket:          bucket,
			Key:             key,
			Tags:            tags,
			UploadId:        checkpoint.UploadId,
			PartId:          partId,
			TotalPartsCount: totalPartsCount,
			ReadOffset:      int64(partId-1) * partSize,
			ReadSize:        readSize,
			FileReader:      fileReader,
			DB:              db,
		})
	}

	um.fileInfos[fileAbsolutePath].Uploaded = checkpoint.UploadedSize
	return uploadInfos, nil
}

func (um *UploadManager) scheduleUploads(ctx context.Context, uploadInfos <-chan UploadInfo, uploadCh chan<- UploadInfo, uploadResultCh <-chan UploadInfo, numThread int) {
	uploadInfoInProgress := make([]UploadInfo, 0)
	var previousUploadInfo *UploadInfo

	for {
		for i := len(uploadInfoInProgress) + 1; i <= numThread; i++ {
			// If there is a previous upload info, try to upload it first.
			if previousUploadInfo != nil {
				if um.fileInfos[previousUploadInfo.Path].Status == UploadFailed {
					// Skip if some other part of the same file has failed.
					previousUploadInfo = nil
					i--
					continue
				}

				if um.canUpload(*previousUploadInfo, uploadInfoInProgress) {
					uploadCh <- *previousUploadInfo
					uploadInfoInProgress = append(uploadInfoInProgress, *previousUploadInfo)
					previousUploadInfo = nil
					continue
				}

				break
			}
			if uploadInfo, ok := <-uploadInfos; ok {
				if um.fileInfos[uploadInfo.Path].Status == UploadFailed {
					// Skip if some other part of the same file has failed
					i--
					continue
				}
				if um.canUpload(uploadInfo, uploadInfoInProgress) {
					uploadCh <- uploadInfo
					uploadInfoInProgress = append(uploadInfoInProgress, uploadInfo)
				} else {
					previousUploadInfo = &uploadInfo
					break
				}
			}
		}

		result := <-uploadResultCh

		uploadInfoInProgress = lo.Filter(uploadInfoInProgress, func(info UploadInfo, _ int) bool {
			return info.Path != result.Path || info.PartId != result.PartId
		})

		if um.fileInfos[result.Path].Status == UploadFailed {
			// Skip if some other part of the same file has failed.
			continue
		}

		if err := um.handleUploadResult(result); err != nil {
			um.debugF("Handle upload result failed: %v", err)

			// todo: retry, abort remaining parts on error, etc.
			um.addErr(result.Path, err)
		}
	}
}

func (um *UploadManager) handleUploadResult(result UploadInfo) error {
	// On error result
	if result.Err != nil {
		return result.Err
	}

	// On success single upload
	if result.UploadId == "" {
		um.fileInfos[result.Path].Status = UploadCompleted
		um.uploadWg.Done()
		if um.noTTY {
			displayPath := um.fileInfos[result.Path].RemotePath
			if displayPath == "" {
				displayPath = result.Path
			}
			log.Infof("Completed upload: %s", displayPath)
		}
		return nil
	}

	// On success multipart upload
	var checkpoint MultipartCheckpointInfo
	if err := result.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(multipartUploadsBucket))
		value := bucket.Get([]byte(mutipartUploadInfoKey))

		if value == nil {
			checkpoint = MultipartCheckpointInfo{
				UploadId: result.UploadId,
			}
		} else {
			if err := json.Unmarshal(value, &checkpoint); err != nil {
				return errors.Wrap(err, "unmarshal checkpoint info")
			}
		}

		checkpoint.UploadId = result.UploadId
		checkpoint.UploadedSize += result.ReadSize
		checkpoint.Parts = append(checkpoint.Parts, minio.CompletePart{
			PartNumber:     result.Result.PartNumber,
			ETag:           result.Result.ETag,
			ChecksumCRC32:  result.Result.ChecksumCRC32,
			ChecksumCRC32C: result.Result.ChecksumCRC32C,
			ChecksumSHA1:   result.Result.ChecksumSHA1,
			ChecksumSHA256: result.Result.ChecksumSHA256,
		})
		checkpointBytes, err := json.Marshal(checkpoint)
		if err != nil {
			return errors.Wrap(err, "marshal checkpoint info")
		}

		return bucket.Put([]byte(mutipartUploadInfoKey), checkpointBytes)
	}); err != nil {
		return errors.Wrapf(err, "update checkpoint info failed for %s", result.Path)
	}

	// Check if multipart upload is completed
	if len(checkpoint.Parts) == result.TotalPartsCount {
		defer func(DB *UploadDB) {
			if err := DB.Delete(); err != nil {
				um.debugF("Delete db failed: %v", err)
			}
		}(result.DB)
		defer func(FileReader *os.File) {
			if err := FileReader.Close(); err != nil {
				um.debugF("Close file failed: %v", err)
			}
		}(result.FileReader)

		um.fileInfos[result.Path].Status = MultipartCompletionInProgress

		slices.SortFunc(checkpoint.Parts, func(i, j minio.CompletePart) int {
			return i.PartNumber - j.PartNumber
		})

		opts := minio.PutObjectOptions{
			UserTags: result.Tags,
		}
		if opts.ContentType = mime.TypeByExtension(filepath.Ext(result.Path)); opts.ContentType == "" {
			opts.ContentType = "application/octet-stream"
		}

		_, err := minio.Core{Client: um.client}.CompleteMultipartUpload(context.Background(), result.Bucket, result.Key, result.UploadId, checkpoint.Parts, opts)
		if err != nil {
			return errors.Wrap(err, "complete multipart upload failed")
		}

		um.fileInfos[result.Path].Status = UploadCompleted
		um.uploadWg.Done()
		if um.noTTY {
			displayPath := um.fileInfos[result.Path].RemotePath
			if displayPath == "" {
				displayPath = result.Path
			}
			log.Infof("Completed multipart upload: %s", displayPath)
		}
	}

	return nil
}

// canUpload checks if the upload candidate is allowed to upload.
// It basically checks if the upload candidate is a multipart upload and if the part id is within the window size
// of the least in progress upload part id.
func (um *UploadManager) canUpload(uploadCandidate UploadInfo, uploadInfoInProgress []UploadInfo) bool {
	leastUploadingPartId := lo.Min(lo.FilterMap(uploadInfoInProgress, func(info UploadInfo, _ int) (int, bool) {
		if info.Path == uploadCandidate.Path {
			return info.PartId, true
		} else {
			return 0, false
		}
	}))

	if leastUploadingPartId == 0 {
		// Case 1: Single upload, no upload with the same path is in progress.
		// Case 2: Multipart upload, no other parts are in progress.
		// For both cases, we can upload directly.
		return true
	}

	windowSize := defaultWindowSize
	if windowSize < int(um.opts.partSizeUint64) {
		windowSize = int(um.opts.partSizeUint64)
	}
	threshold := leastUploadingPartId + windowSize/int(um.opts.partSizeUint64)

	return uploadCandidate.PartId <= threshold
}

func (um *UploadManager) consumeSingleUploadInfo(ctx context.Context, uploadInfo UploadInfo) error {
	defer func(FileReader *os.File) {
		if err := FileReader.Close(); err != nil {
			um.debugF("Close file failed: %v", err)
		}
	}(uploadInfo.FileReader)

	um.fileInfos[uploadInfo.Path].Status = UploadInProgress
	progressReader := &uploadProgressReader{
		File:       uploadInfo.FileReader,
		fileInfo:   um.fileInfos[uploadInfo.Path],
		progressCh: um.progressCh,
	}

	_, err := minio.Core{Client: um.client}.PutObject(
		ctx, uploadInfo.Bucket, uploadInfo.Key, progressReader, um.fileInfos[uploadInfo.Path].Size, "",
		um.fileInfos[uploadInfo.Path].Sha256, minio.PutObjectOptions{
			UserTags:         uploadInfo.Tags,
			DisableMultipart: true,
		})

	return err
}

func (um *UploadManager) consumeMultipartUploadInfo(ctx context.Context, uploadInfo UploadInfo) (minio.ObjectPart, error) {
	sectionReader := &uploadProgressSectionReader{
		SectionReader: io.NewSectionReader(uploadInfo.FileReader, uploadInfo.ReadOffset, uploadInfo.ReadSize),
		fileInfo:      um.fileInfos[uploadInfo.Path],
		progressCh:    um.progressCh,
	}
	um.debugF("Uploading part %d of %s", uploadInfo.PartId, uploadInfo.Path)

	objPart, err := minio.Core{Client: um.client}.PutObjectPart(ctx, uploadInfo.Bucket, uploadInfo.Key, uploadInfo.UploadId, uploadInfo.PartId, sectionReader, uploadInfo.ReadSize, minio.PutObjectPartOptions{})
	if err != nil {
		um.debugF("Put object part %d of %s failed: %v", uploadInfo.PartId, uploadInfo.Path, err)
	} else {
		um.debugF("Put object part %d of %s succeeded", uploadInfo.PartId, uploadInfo.Path)
	}
	return objPart, err
}

// parseUrl parses the upload url to get the bucket, key and tags.
func (um *UploadManager) parseUrl(uploadUrl string) (string, string, map[string]string, error) {
	parsedUrl, err := url.Parse(uploadUrl)
	if err != nil {
		return "", "", nil, errors.Wrap(err, "parse upload url failed")
	}

	// Parse tags
	tagsMap, err := url.ParseQuery(parsedUrl.Query().Get("X-Amz-Tagging"))
	if err != nil {
		return "", "", nil, errors.Wrap(err, "parse tags failed")
	}
	tags := lo.MapValues(tagsMap, func(value []string, _ string) string {
		if len(value) == 0 {
			return ""
		}
		return value[0]
	})

	// Parse bucket and key
	pathParts := strings.SplitN(parsedUrl.Path, "/", 3)
	bucket := pathParts[1]
	key := pathParts[2]
	return bucket, key, tags, nil
}

// addFile adds a file to the upload manager.
func (um *UploadManager) addFile(path string) {
	um.fileList = append(um.fileList, path)
	um.fileInfos[path] = &FileInfo{
		Path: path,
	}
}

// debugF is used to print debug messages.
// cannot use logrus here because tea.Program overtakes the log output.
func (um *UploadManager) debugF(format string, args ...interface{}) {
	if um.isDebug {
		msg := fmt.Sprintf(format, args...)
		if um.noTTY {
			// In non-interactive mode, use logrus
			log.Debug(msg)
		} else if um.monitor != nil {
			// In interactive mode, use tea.Program
			debugMsg := wordwrap.String(fmt.Sprintf("DEBUG: %s", msg), um.windowWidth)
			um.monitor.Println(debugMsg)
		}
	}
}

// addErr adds an error to the manager.
func (um *UploadManager) addErr(path string, err error) {
	um.debugF("Upload %s failed with: %v", path, err)
	um.fileInfos[path].Status = UploadFailed
	um.errs[path] = err
	um.uploadWg.Done()
	if um.noTTY {
		log.Errorf("Upload failed for %s: %v", path, err)
	}
}

// printErrs prints all errors.
func (um *UploadManager) printErrs() {
	if len(um.errs) > 0 {
		fmt.Printf("\n%d files failed to upload\n", len(um.errs))
		for kPath, vErr := range um.errs {
			fmt.Printf("Upload %v failed with: \n%v\n\n", kPath, vErr)
		}
		return
	}
}

// removed old compatibility wrapper: findAllUploadUrls

// removed old compatibility wrapper: findAllUploadUrlsProject

// findAllUploadUrlsGeneric prepares upload URLs for either record or project parent using parentContext.
func (um *UploadManager) findAllUploadUrlsGeneric(filesToUpload []string, pCtx parentContext, relativeDir string, targetPrefix string) map[string]string {
	ret := make(map[string]string)
	var files []*openv1alpha1resource.File
	resourceToRel := make(map[string]string)

	if um.noTTY && len(filesToUpload) > 0 {
		log.Infof("Processing %d files for upload...", len(filesToUpload))
	}

	for _, f := range filesToUpload {
		um.addFile(f)
		um.fileInfos[f].Status = CalculatingSha256
		checksum, size, err := fs.CalSha256AndSize(f)
		if err != nil {
			um.addErr(f, errors.Wrapf(err, "unable to calculate sha256 for file"))
			continue
		}
		um.fileInfos[f].Size = size
		um.fileInfos[f].Sha256 = checksum

		relativePath, err := filepath.Rel(relativeDir, f)
		if err != nil {
			um.addErr(f, errors.Wrapf(err, "unable to get relative path"))
			continue
		}

		// Apply target prefix if specified
		remotePath := relativePath
		if targetPrefix != "" {
			remotePath = filepath.Join(targetPrefix, relativePath)
		}

		// Existence check
		resourceName := pCtx.buildResourceName(remotePath)
		getFileRes, err := um.apiOpts.GetFile(context.TODO(), resourceName)
		if err == nil && getFileRes.Sha256 == checksum && getFileRes.Size == size {
			um.fileInfos[f].Status = PreviouslyUploaded
			um.uploadWg.Done()
			if um.noTTY {
				log.Infof("File %s already uploaded, skipping", relativePath)
			}
			continue
		}

		um.fileInfos[f].Status = WaitingForUpload
		um.fileInfos[f].RemotePath = remotePath // Set remote path for display
		files = append(files, &openv1alpha1resource.File{
			Name:     resourceName,
			Filename: remotePath,
			Sha256:   checksum,
			Size:     size,
		})
		resourceToRel[resourceName] = relativePath

		if len(files) == processBatchSize {
			um.debugF("Generating upload urls for %d files", len(files))
			res, err := um.apiOpts.GenerateFileUploadUrls(context.TODO(), pCtx.parentString, files)
			if err != nil {
				for _, file := range files {
					um.addErr(filepath.Join(relativeDir, file.Filename), errors.Wrapf(err, "unable to generate upload urls"))
				}
				continue
			}
			for k, v := range res {
				if rel, ok := resourceToRel[k]; ok {
					ret[filepath.Join(relativeDir, rel)] = v
				}
			}
			files = nil
		}
	}

	if len(files) > 0 {
		um.debugF("Generating upload urls for %d files", len(files))
		res, err := um.apiOpts.GenerateFileUploadUrls(context.TODO(), pCtx.parentString, files)
		if err != nil {
			for _, file := range files {
				um.addErr(filepath.Join(relativeDir, file.Filename), errors.Wrapf(err, "unable to generate upload urls"))
			}
		}
		for k, v := range res {
			if rel, ok := resourceToRel[k]; ok {
				ret[filepath.Join(relativeDir, rel)] = v
			}
		}
	}

	return ret
}

// calculateUploadProgress is used to calculate the progress of a file upload
func (um *UploadManager) calculateUploadProgress(name string) float64 {
	status := um.fileInfos[name]
	if status.Size == 0 {
		return 100
	}
	return float64(status.Uploaded) * 100 / float64(status.Size)
}

func (um *UploadManager) Init() tea.Cmd {
	return tick()
}

func (um *UploadManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		um.windowWidth = msg.Width
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEscape, tea.KeyCtrlD:
			um.manualQuit = true
			return um, tea.Quit
		}
	case TickMsg:
		um.spinnerIdx = (um.spinnerIdx + 1) % len(spinnerFrames)
		return um, tick()
	}
	return um, nil
}

func (um *UploadManager) View() string {
	s := "Upload Status:\n"
	skipCount := 0
	successCount := 0
	spinnerFrame := spinnerFrames[um.spinnerIdx]
	for _, k := range um.fileList {
		// Use remote path for display if available, fallback to local path
		displayPath := um.fileInfos[k].RemotePath
		if displayPath == "" {
			displayPath = k
		}

		statusStrLen := um.windowWidth - len(displayPath) - 1
		switch um.fileInfos[k].Status {
		case Unprocessed:
			s += fmt.Sprintf("%s:%*s\n", displayPath, statusStrLen, "Preparing for upload"+spinnerFrame)
		case CalculatingSha256:
			s += fmt.Sprintf("%s:%*s\n", displayPath, statusStrLen, "Calculating sha256"+spinnerFrame)
		case PreviouslyUploaded:
			s += fmt.Sprintf("%s:%*s\n", displayPath, statusStrLen, "Previously uploaded, skipping")
			skipCount++
		case WaitingForUpload:
			s += fmt.Sprintf("%s:%*s\n", displayPath, statusStrLen, "Waiting for upload")
		case UploadCompleted:
			s += fmt.Sprintf("%s:%*s\n", displayPath, statusStrLen, "Upload completed")
			successCount++
		case MultipartCompletionInProgress:
			s += fmt.Sprintf("%s:%*s\n", displayPath, statusStrLen, "Completing multipart upload"+spinnerFrame)
		case UploadFailed:
			s += fmt.Sprintf("%s:%*s\n", displayPath, statusStrLen, "Upload failed")
		case UploadInProgress:
			progress := um.calculateUploadProgress(k)
			barWidth := max(um.windowWidth-len(k)-12, 10)                       // Adjust for label and percentage, make sure it is at least 10
			progressCount := min(int(progress*float64(barWidth)/100), barWidth) // min used to prevent float rounding errors
			emptyBar := strings.Repeat("-", barWidth-progressCount)
			progressBar := strings.Repeat("█", progressCount)
			s += fmt.Sprintf("%s: [%s%s] %*.2f%%\n", displayPath, progressBar, emptyBar, 6, progress)
		}
	}

	// Add summary of all file status
	s += "\n"
	s += fmt.Sprintf("Total: %d, Skipped: %d, Success: %d", len(um.fileList), skipCount, successCount)
	if successCount+skipCount < len(um.fileList) {
		s += fmt.Sprintf(", Remaining: %d", len(um.fileList)-successCount-skipCount)
	}
	s += "\n"
	s = wordwrap.String(s, um.windowWidth)
	return s
}

// TickMsg is a message that is sent to the update function every 0.5 second.
type TickMsg time.Time

// tick is a command that sends a TickMsg every 0.5 second.
func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// uploadProgressReader is a reader that sends progress updates to a channel.
type uploadProgressReader struct {
	*os.File
	fileInfo   *FileInfo
	progressCh chan IncUploadedMsg
}

func (r *uploadProgressReader) Read(b []byte) (int, error) {
	n, err := r.File.Read(b)
	if err != nil && err != io.EOF {
		r.progressCh <- IncUploadedMsg{
			Path:        r.fileInfo.Path,
			UploadedInc: -r.fileInfo.Uploaded,
		}
	} else {
		r.progressCh <- IncUploadedMsg{
			Path:        r.fileInfo.Path,
			UploadedInc: int64(n),
		}
	}
	return n, err
}

// uploadProgressSectionReader is a SectionReader that also sends progress updates to a channel.
type uploadProgressSectionReader struct {
	*io.SectionReader
	fileInfo   *FileInfo
	uploaded   int64
	progressCh chan IncUploadedMsg
}

func (r *uploadProgressSectionReader) Read(b []byte) (int, error) {
	n, err := r.SectionReader.Read(b)
	if err != nil && err != io.EOF {
		r.progressCh <- IncUploadedMsg{
			Path:        r.fileInfo.Path,
			UploadedInc: -r.uploaded,
		}
		r.uploaded = 0
	} else {
		r.progressCh <- IncUploadedMsg{
			Path:        r.fileInfo.Path,
			UploadedInc: int64(n),
		}
		r.uploaded += int64(n)
	}
	return n, err
}

// nonInteractiveProgressReporter reports progress in non-interactive mode
func (um *UploadManager) nonInteractiveProgressReporter() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			um.reportNonInteractiveProgress()
		default:
			// Check if all uploads are done
			allDone := true
			for _, fileInfo := range um.fileInfos {
				if fileInfo.Status != UploadCompleted &&
					fileInfo.Status != PreviouslyUploaded &&
					fileInfo.Status != UploadFailed {
					allDone = false
					break
				}
			}
			if allDone {
				um.reportNonInteractiveProgress() // Final report
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// reportNonInteractiveProgress logs the current upload progress
func (um *UploadManager) reportNonInteractiveProgress() {
	var totalFiles, completedFiles, failedFiles, skippedFiles int
	var totalBytes, uploadedBytes int64

	for _, fileInfo := range um.fileInfos {
		totalFiles++
		totalBytes += fileInfo.Size
		uploadedBytes += fileInfo.Uploaded

		switch fileInfo.Status {
		case UploadCompleted:
			completedFiles++
		case UploadFailed:
			failedFiles++
		case PreviouslyUploaded:
			skippedFiles++
		}
	}

	progress := float64(0)
	if totalBytes > 0 {
		progress = float64(uploadedBytes) * 100 / float64(totalBytes)
	}

	log.Infof("Upload Progress: %.1f%% | Total: %d | Completed: %d | Failed: %d | Skipped: %d",
		progress, totalFiles, completedFiles, failedFiles, skippedFiles)
}
