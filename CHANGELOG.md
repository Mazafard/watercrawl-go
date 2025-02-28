# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1-alpha] - 2025-02-28

### Fixed
- Implemented proper Server-Sent Events (SSE) parsing to handle the API's event stream format
- Fixed JSON response handling in `DownloadCrawlRequest` to support both object and array responses
- Resolved "no result received" errors that occurred with certain API responses
- Enhanced error handling throughout the SDK to prevent resource leaks
- Improved response body closing to ensure all HTTP connections are properly released
- Added better error propagation and contextual error messages
- Fixed race conditions in asynchronous event processing

### Added
- Support for "state" events in addition to "result" events
- Fallback mechanism to use state data when no result events are received
- Better progress tracking and reporting
- Enhanced debugging output with detailed request and response logging
- More comprehensive error messages from API responses
- Improved context support for better timeout and cancellation handling
- Additional test cases for error conditions and edge cases

### Changed
- Modified `MonitorCrawlRequest` to use `bufio.Reader` for proper line-by-line reading
- Updated `ScrapeURL` to handle various event types (state, progress, result, error)
- Improved `processResponse` to extract more useful error information
- Enhanced test suite with more comprehensive test cases
- Updated error handling in all test code for better diagnostics
- Improved logging to provide more context in error situations

### Security
- Added proper resource cleanup to prevent potential memory leaks
- Improved timeout handling for network operations
- Enhanced error checking for all I/O operations
- Added safeguards against unclosed resources

## [0.1.0] - 2024-03-14

### Added
- Initial release of the WaterCrawl Go SDK
- Basic client implementation with all core API endpoints
- Support for creating and managing crawl requests
- Support for monitoring crawl progress
- Support for downloading crawl results
- Comprehensive documentation and examples
- MIT License
- Contributing guidelines