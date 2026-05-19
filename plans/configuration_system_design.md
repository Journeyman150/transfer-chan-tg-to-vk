# Configuration System Design

## Overview
Flexible configuration system supporting YAML files, environment variables, and command-line flags with validation and defaults. The system is designed for **minimal user configuration** - most parameters have sensible defaults, and users only need to provide essential credentials and identifiers.

## Design Philosophy: Minimal Configuration Required

### Essential Configuration (User Must Provide)
1. **Telegram Bot Token** - From @BotFather
2. **Telegram Channel ID** - @username or numeric ID
3. **VK Access Token** - With required permissions
4. **VK Group ID** - Target group for posting

### Everything Else Has Sensible Defaults
- Rate limits optimized for each API
- Media processing with safe defaults
- Error handling with automatic retries
- Logging at appropriate levels
- Performance settings for typical use cases

## Minimal Configuration Example

```yaml
# Absolute minimum configuration - just 4 values needed!
telegram:
  bot_token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
  channel_id: "@my_channel"

vk:
  access_token: "vk1.a.abcdef1234567890"
  group_id: 123456789
```

That's it! With just these 4 values, the application will:
- Transfer all posts with media
- Use optimal rate limiting for both APIs
- Handle errors automatically
- Log progress to console
- Use safe defaults for all other parameters

## Configuration Structure with Smart Defaults

### Telegram Configuration
```yaml
telegram:
  # REQUIRED: Bot token from @BotFather
  bot_token: ""
  
  # REQUIRED: Channel identifier
  channel_id: ""
  
  # Optional: Most users won't need to change these
  api_url: "https://api.telegram.org"  # Default works for everyone
  timeout: "30s"                       # Safe timeout
  requests_per_second: 20              # Under Telegram's 30/sec limit
  max_retries: 3                       # Reasonable retry count
  include_media: true                  # Most users want media
  max_file_size: "100MB"               # Reasonable limit
```

### VK Configuration
```yaml
vk:
  # REQUIRED: Access token
  access_token: ""
  
  # REQUIRED: Group ID
  group_id: 0
  
  # Optional: Sensible defaults
  api_version: "5.199"                 # Latest stable API
  requests_per_second: 3.0             # VK's actual limit
  post_delay: "3s"                     # Safe spacing between posts
  from_group: true                     # Usually posting as group
  signed: true                         # Show group signature
```

### Transfer Configuration
```yaml
transfer:
  # Safety first: dry-run by default
  dry_run: true                        # Prevents accidental posting!
  
  # Sensible content handling
  include_text: true
  include_media: true
  preserve_formatting: true
  skip_duplicates: true                # Avoid reposting
  
  # Error handling defaults
  skip_failed_posts: true              # Continue on errors
  max_errors_before_stop: 10           # Stop if too many errors
```

### Media Processing (Automatic Optimization)
```yaml
media:
  # Automatic temp management
  temp_dir: "./temp"                   # Created automatically
  keep_files: false                    # Clean up after transfer
  
  # Safe concurrent operations
  max_concurrent_downloads: 3          # Won't overload Telegram
  max_concurrent_uploads: 2            # Respects VK limits
  
  # Automatic media optimization
  photos:
    max_width: 1920                    # Full HD is enough
    quality: 85                        # Good quality/size balance
  videos:
    max_size: "2GB"                    # VK's limit
    convert_gif_to_mp4: true           # Better compatibility
```

## Default Values Strategy

### Why These Defaults?
1. **Safety First**: `dry_run: true` prevents accidental posting
2. **API Respect**: Rate limits set below actual API limits
3. **User Experience**: Continue on errors, skip duplicates
4. **Performance**: Concurrent operations balanced for stability
5. **Quality**: Media optimized without noticeable quality loss

### No Configuration Needed For:
- Logging levels (info is perfect)
- HTTP timeouts (30-60s works for most)
- Retry logic (3 attempts with backoff)
- Checkpoint system (enabled by default)
- Memory limits (automatic based on system)

## Configuration Loading Strategy

### Priority Order (Highest to Lowest)
1. Command-line flags (for one-off overrides)
2. Environment variables (for secrets in production)
3. Configuration file (for project-specific settings)
4. **Smart defaults** (carefully chosen for most users)

### Environment Variables for Production
```bash
# Just 4 environment variables needed for production
export TG2VK_TELEGRAM_BOT_TOKEN="123456:ABC..."
export TG2VK_TELEGRAM_CHANNEL_ID="@my_channel"
export TG2VK_VK_ACCESS_TOKEN="vk1.a.abc..."
export TG2VK_VK_GROUP_ID="123456789"

# Run with zero config files
./transfer
```

## Advanced Configuration (Optional)

### Only Change If You Know What You're Doing
```yaml
# These sections exist but most users should not touch them
performance:
  # Already optimized
  max_goroutines: 100
  worker_pool_size: 10

error_handling:
  # Already handles errors well
  circuit_breaker:
    failure_threshold: 5
    reset_timeout: "60s"

monitoring:
  # Disabled by default to keep it simple
  enabled: false
```

## Configuration Validation

### Helpful Error Messages
If required values are missing:
```
Error: Configuration validation failed:
- telegram.bot_token is required (get it from @BotFather)
- telegram.channel_id is required (use @username for public channels)
- vk.access_token is required (create at https://vk.com/dev)
- vk.group_id is required (numeric ID of your VK group)

Tip: Create a minimal config.yaml with just these 4 values!
```

### Automatic Fixes
- Creates `./temp` directory if it doesn't exist
- Creates `./logs` directory for log files
- Validates token formats before using them
- Suggests corrections for common mistakes

## Example Workflows

### First-Time User
1. Get Telegram bot token from @BotFather
2. Get VK access token from VK Dev
3. Create `config.yaml` with 4 values
4. Run with `dry_run: true` (default) to test
5. Set `dry_run: false` when ready

### Production Deployment
1. Set 4 environment variables
2. No config file needed
3. Run application
4. Check logs for progress

### Advanced User
1. Start with minimal config
2. Run and monitor performance
3. Tune only specific parameters if needed
4. Most defaults will work fine

## Implementation

### Default Values in Code
```go
func setDefaults(v *viper.Viper) {
	// Safety first
	v.SetDefault("transfer.dry_run", true)
	
	// API respect
	v.SetDefault("telegram.requests_per_second", 20)  // Under 30 limit
	v.SetDefault("vk.requests_per_second", 3.0)       // VK's actual limit
	
	// User experience
	v.SetDefault("transfer.skip_duplicates", true)
	v.SetDefault("transfer.skip_failed_posts", true)
	v.SetDefault("transfer.max_errors_before_stop", 10)
	
	// Media optimization
	v.SetDefault("media.photos.max_width", 1920)
	v.SetDefault("media.photos.quality", 85)
	v.SetDefault("media.videos.convert_gif_to_mp4", true)
	
	// Performance
	v.SetDefault("media.max_concurrent_downloads", 3)
	v.SetDefault("media.max_concurrent_uploads", 2)
	
	// Everything else has sensible defaults
	// Users don't need to configure these
}
```

### Configuration Generator
```bash
# Generates config with just the 4 required fields
./transfer --generate-config

# Creates config.yaml with comments explaining each field
# Only shows essential options, hides advanced ones
```

## Best Practices for Minimal Configuration

1. **Require Only Essentials**: 4 values to start
2. **Safe Defaults**: Prevent accidents, respect APIs
3. **Progressive Disclosure**: Advanced options available but hidden
4. **Helpful Errors**: Tell users exactly what's missing and how to fix it
5. **Environment Ready**: Production deployment with just env vars

## Testing Configuration

### Test Cases
1. **Minimal Config**: Just 4 values should work
2. **Empty Config**: Should fail with helpful error
3. **Environment Override**: Env vars should override defaults
4. **Dry Run Safety**: Default should prevent posting
5. **Auto-Creation**: Temp dirs should be created automatically

## Conclusion

The configuration system is designed to "just work" with minimal input while providing escape hatches for advanced users. By focusing on sensible defaults and requiring only essential credentials, we reduce setup time and prevent configuration errors.

**User Experience Goal**: Get from zero to transferring content in under 5 minutes with just 4 pieces of information.