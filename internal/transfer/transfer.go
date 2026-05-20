package transfer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	tg "ai-transfer-tg-to-vk/internal/telegram"
	vk "ai-transfer-tg-to-vk/internal/vk"
	"ai-transfer-tg-to-vk/internal/config"
	"ai-transfer-tg-to-vk/internal/media"
	"ai-transfer-tg-to-vk/internal/transformer"
	"go.uber.org/zap"
)

// Transfer orchestrates the complete transfer process.
type Transfer struct {
	config         *config.Config
	logger         *zap.Logger
	telegramClient tg.Client
	vkClient       vk.Client
	mediaStorage   *media.StorageManager
	downloader     *media.Downloader
	transformer    *transformer.PostTransformer
	checkpoint     *CheckpointManager
	progress       *ProgressTracker
}

// NewTransfer creates a new Transfer instance with all required components.
func NewTransfer(cfg *config.Config, logger *zap.Logger) (*Transfer, error) {
	// Create temporary directory for media files
	tempDir := cfg.Media.TempDir
	if tempDir == "" {
		tempDir = filepath.Join(os.TempDir(), "tg-vk-transfer")
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Initialize Telegram client
	tgConfig := tg.Config{
		BotToken:         cfg.Telegram.BotToken,
		APIURL:           cfg.Telegram.APIURL,
		Timeout:          time.Duration(cfg.Telegram.Timeout),
		Debug:            cfg.Telegram.Debug,
		RequestsPerSecond: cfg.Telegram.RequestsPerSecond,
		MaxRetries:       cfg.Telegram.MaxRetries,
	}
	telegramClient, err := tg.NewBotAPIClient(tgConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram client: %w", err)
	}

	// Initialize VK client
	vkConfig := vk.Config{
		AccessToken:      cfg.VK.AccessToken,
		APIURL:           cfg.VK.APIURL,
		APIVersion:       cfg.VK.APIVersion,
		Timeout:          time.Duration(cfg.VK.Timeout),
		Language:         cfg.VK.Language,
		RequestsPerSecond: cfg.VK.RequestsPerSecond,
	}
	vkClient, err := vk.NewClient(vkConfig)
	if err != nil {
		telegramClient.Close()
		return nil, fmt.Errorf("failed to create VK client: %w", err)
	}

	// Initialize media storage with default values
	// Use 24h max age and 1GB max size as defaults
	maxAge := 24 * time.Hour
	var maxSize int64 = 1 * 1024 * 1024 * 1024 // 1GB
	storage, err := media.NewStorageManager(tempDir, maxAge, maxSize)
	if err != nil {
		telegramClient.Close()
		vkClient.Close()
		return nil, fmt.Errorf("failed to create media storage: %w", err)
	}

	// Initialize downloader
	downloader := media.NewDownloader(tempDir, cfg.Media.DownloadRetries, 
		cfg.Media.MaxConcurrentDownloads, storage)

	// Initialize transformer
	transformerConfig := transformer.Config{
		PreserveDates:    cfg.Transfer.PreserveDates,
		FromGroup:        cfg.VK.FromGroup,
		AddSourceLink:    cfg.Transfer.AddSourceLink,
		SourceLinkFormat: cfg.Transfer.SourceLinkFormat,
		IncludeForwardInfo: cfg.Transfer.IncludeForwardInfo,
		ForwardFormat:    cfg.Transfer.ForwardFormat,
		MaxPostsPerBatch: cfg.Transfer.MaxPostsPerBatch,
		SkipFailedPosts:  cfg.Transfer.SkipFailedPosts,
	}
	postTransformer := transformer.NewPostTransformer(transformerConfig)

	// Initialize checkpoint manager
	checkpoint := NewCheckpointManager(filepath.Join(tempDir, "checkpoint.json"), logger.Named("checkpoint"))

	// Initialize progress tracker
	progress := NewProgressTracker(logger.Named("progress"))

	return &Transfer{
		config:         cfg,
		logger:         logger.Named("transfer"),
		telegramClient: telegramClient,
		vkClient:       vkClient,
		mediaStorage:   storage,
		downloader:     downloader,
		transformer:    postTransformer,
		checkpoint:     checkpoint,
		progress:       progress,
	}, nil
}

// Run executes the complete transfer process.
func (t *Transfer) Run(ctx context.Context) error {
	t.logger.Info("Starting transfer process",
		zap.String("telegram_channel", t.config.Telegram.ChannelID),
		zap.Int("vk_group", t.config.VK.GroupID),
		zap.Bool("dry_run", t.config.Transfer.DryRun),
	)

	// Load checkpoint if exists
	checkpointState := t.checkpoint.Load()

	// Fetch posts from Telegram
	t.logger.Info("Fetching posts from Telegram channel")
	posts, err := t.fetchPosts(ctx, checkpointState)
	if err != nil {
		return fmt.Errorf("failed to fetch posts: %w", err)
	}

	t.logger.Info("Fetched posts from Telegram",
		zap.Int("total_posts", len(posts)),
		zap.Int("posts_to_process", len(posts)-checkpointState.LastProcessedIndex-1),
	)

	// Start progress tracking
	t.progress.Start(len(posts))

	// Process posts
	if err := t.processPosts(ctx, posts, checkpointState); err != nil {
		return fmt.Errorf("failed to process posts: %w", err)
	}

	t.logger.Info("Transfer completed successfully")
	return nil
}

// fetchPosts retrieves posts from Telegram channel.
func (t *Transfer) fetchPosts(ctx context.Context, checkpoint *CheckpointState) ([]tg.Post, error) {
	opts := tg.GetPostsOptions{
		Limit:    t.config.Telegram.BatchSize,
		StartDate: time.Time{},
		EndDate:   time.Time{},
		OnlyMedia: false,
	}

	// Apply date filters if configured
	if t.config.Telegram.StartDate != nil {
		opts.StartDate = *t.config.Telegram.StartDate
	}
	if t.config.Telegram.EndDate != nil {
		opts.EndDate = *t.config.Telegram.EndDate
	}

	// If we have a checkpoint with offset ID, use it
	if checkpoint.LastProcessedID > 0 {
		opts.OffsetID = checkpoint.LastProcessedID
	}

	posts, err := t.telegramClient.GetPosts(ctx, t.config.Telegram.ChannelID, opts)
	if err != nil {
		return nil, fmt.Errorf("telegram get posts failed: %w", err)
	}

	return posts, nil
}

// processPosts processes each post: download media, transform, and post to VK.
func (t *Transfer) processPosts(ctx context.Context, posts []tg.Post, checkpoint *CheckpointState) error {
	startIndex := checkpoint.LastProcessedIndex + 1
	if startIndex >= len(posts) {
		t.logger.Info("All posts already processed according to checkpoint")
		return nil
	}

	for i := startIndex; i < len(posts); i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		post := posts[i]
		t.logger.Info("Processing post",
			zap.Int("index", i),
			zap.Int64("post_id", post.ID),
			zap.Time("date", post.Date),
		)

		t.progress.UpdatePostProcessed(post.ID, "processing")

		if err := t.processSinglePost(ctx, post, i); err != nil {
			if t.config.Transfer.SkipFailedPosts {
				t.logger.Error("Failed to process post, skipping", zap.Error(err))
				t.progress.UpdatePostFailed(post.ID, err.Error())
				continue
			}
			return fmt.Errorf("failed to process post %d: %w", post.ID, err)
		}

		// Update checkpoint
		checkpoint.LastProcessedIndex = i
		checkpoint.LastProcessedID = post.ID
		checkpoint.TotalPosts = len(posts)
		if err := t.checkpoint.Save(checkpoint); err != nil {
			t.logger.Warn("Failed to save checkpoint", zap.Error(err))
		}

		t.progress.UpdatePostProcessed(post.ID, "completed")

		// Respect rate limiting between posts
		if t.config.VK.PostDelay > 0 {
			delay := time.Duration(t.config.VK.PostDelay)
			t.logger.Debug("Waiting between posts", zap.Duration("delay", delay))
			time.Sleep(delay)
		}
	}

	return nil
}

// processSinglePost handles a single Telegram post.
func (t *Transfer) processSinglePost(ctx context.Context, tgPost tg.Post, index int) error {
	// Download media files if needed
	var downloadedFiles []*media.MediaFile
	if t.config.Transfer.IncludeMedia && len(tgPost.Media) > 0 {
		files, err := t.downloadMedia(ctx, tgPost.Media)
		if err != nil {
			return fmt.Errorf("failed to download media: %w", err)
		}
		downloadedFiles = files
		defer func() {
			// Clean up downloaded files after processing
			if !t.config.Media.KeepFiles {
				for _, file := range downloadedFiles {
					os.Remove(file.Path)
				}
			}
		}()
	}

	// Transform Telegram post to VK post
	vkPost, warnings, err := t.transformer.TransformPost(tgPost)
	if err != nil {
		return fmt.Errorf("failed to transform post: %w", err)
	}
	for _, warning := range warnings {
		t.logger.Warn("Transformation warning", zap.String("warning", warning))
	}

	// Upload media to VK if we have downloaded files
	if len(downloadedFiles) > 0 {
		attachments, err := t.uploadMedia(ctx, downloadedFiles, tgPost.Media)
		if err != nil {
			return fmt.Errorf("failed to upload media: %w", err)
		}
		vkPost.Attachments = append(vkPost.Attachments, attachments...)
	}

	// Post to VK (or dry run)
	if t.config.Transfer.DryRun {
		t.logger.Info("Dry run: would post to VK",
			zap.String("text_preview", previewText(vkPost.Text)),
			zap.Int("attachments", len(vkPost.Attachments)),
		)
		return nil
	}

	result, err := t.vkClient.Post(ctx, *vkPost)
	if err != nil {
		return fmt.Errorf("failed to post to VK: %w", err)
	}

	t.logger.Info("Successfully posted to VK",
		zap.Int("post_id", result.PostID),
	)

	return nil
}

// downloadMedia downloads all media files from a Telegram post.
func (t *Transfer) downloadMedia(ctx context.Context, mediaItems []tg.Media) ([]*media.MediaFile, error) {
	var wg sync.WaitGroup
	errors := make(chan error, len(mediaItems))
	fileResults := make(chan *media.MediaFile, len(mediaItems))

	for _, m := range mediaItems {
		wg.Add(1)
		go func(mediaItem tg.Media) {
			defer wg.Done()
			
			// Create download request - convert Telegram MediaType to media.MediaType
			req := media.DownloadRequest{
				FileID: mediaItem.FileID,
				Type:   media.MediaType(mediaItem.Type),
			}
			
			file, err := t.downloader.Download(ctx, req)
			if err != nil {
				errors <- fmt.Errorf("failed to download media %s: %w", mediaItem.FileID, err)
				return
			}
			fileResults <- file
		}(m)
	}

	wg.Wait()
	close(errors)
	close(fileResults)

	// Collect errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	// Collect results
	var files []*media.MediaFile
	for file := range fileResults {
		files = append(files, file)
	}

	if len(errs) > 0 {
		return files, fmt.Errorf("download errors: %v", errs)
	}

	return files, nil
}

// uploadMedia uploads downloaded files to VK.
func (t *Transfer) uploadMedia(ctx context.Context, mediaFiles []*media.MediaFile, mediaItems []tg.Media) ([]vk.Attachment, error) {
	attachments := make([]vk.Attachment, 0, len(mediaFiles))

	for i, mediaFile := range mediaFiles {
		mediaType := mediaItems[i].Type
		var attachment *vk.Attachment
		var err error

		switch mediaType {
		case tg.MediaTypePhoto:
			attachment, err = t.vkClient.UploadPhoto(ctx, mediaFile.Path, t.config.VK.GroupID)
		case tg.MediaTypeVideo:
			attachment, err = t.vkClient.UploadVideo(ctx, mediaFile.Path, "", "", t.config.VK.GroupID)
		case tg.MediaTypeDocument:
			attachment, err = t.vkClient.UploadDocument(ctx, mediaFile.Path, "", t.config.VK.GroupID)
		default:
			t.logger.Warn("Unsupported media type, skipping", 
				zap.String("type", string(mediaType)))
			continue
		}

		if err != nil {
			return attachments, fmt.Errorf("failed to upload %s: %w", mediaType, err)
		}

		attachments = append(attachments, *attachment)
	}

	return attachments, nil
}

// previewText returns a short preview of text for logging.
func previewText(text string) string {
	if len(text) > 100 {
		return text[:100] + "..."
	}
	return text
}

// Close releases all resources.
func (t *Transfer) Close() error {
	var errs []error

	if err := t.telegramClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("telegram client close: %w", err))
	}

	if err := t.vkClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("vk client close: %w", err))
	}

	if err := t.mediaStorage.Cleanup(); err != nil {
		errs = append(errs, fmt.Errorf("media storage cleanup: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	return nil
}