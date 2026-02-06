# Drift Package - Test Coverage Improvements

## Current Coverage: 39.4%

### Priority 1: Critical Missing Tests

#### 1. Detector.ScanDirectory() - 0% coverage
**Why Critical:** This is the core functionality that scans directories for .tf files
**Test Cases Needed:**
- [ ] Test scanning directory with multiple .tf files
- [ ] Test scanning nested directory structure
- [ ] Test handling of hidden directories (should skip)
- [ ] Test handling of non-.tf files (should skip)
- [ ] Test error handling for non-existent path
- [ ] Test preferring versions.tf over other .tf files in same directory

#### 2. HCL Parsing Functions - 0% coverage
**Functions:** parseWithHCL(), extractTerraformBlock(), extractProvidersBlock()
**Why Critical:** Core version extraction logic
**Test Cases Needed:**
- [ ] Test parsing valid terraform block with required_version
- [ ] Test parsing required_providers block
- [ ] Test handling malformed HCL
- [ ] Test handling missing terraform block
- [ ] Test handling missing required_providers
- [ ] Test extracting multiple provider versions

#### 3. Formatter Output Formats - 0% coverage
**Functions:** formatJSON(), formatCSV(), formatStatusLipgloss()
**Why Important:** User-facing output needs to be correct
**Test Cases Needed:**
- [ ] Test JSON output format
- [ ] Test CSV output format with headers
- [ ] Test CSV output with drift records
- [ ] Test status formatting for all DriftStatus values
- [ ] Test status formatting with/without color

### Priority 2: Improve Partial Coverage

#### 4. formatTable() - 31% coverage
**Test Cases Needed:**
- [ ] Test table output with drift records
- [ ] Test table with no drift
- [ ] Test table with errors
- [ ] Test table with long file paths (truncation)
- [ ] Test table with provider drifts

#### 5. HasCriticalDrift() - 0% coverage
**Test Cases Needed:**
- [ ] Test returns true when major drift exists
- [ ] Test returns false when only minor drift
- [ ] Test returns false when no drift

### Priority 3: CMD Package Tests

#### 6. cmd/drift.go - No tests
**Test Cases Needed:**
- [ ] Test path validation (exists, is directory)
- [ ] Test absolute path conversion
- [ ] Test with various --path inputs (., ./, relative, absolute)
- [ ] Test with --format flag (table, json, csv)
- [ ] Test with --no-color flag
- [ ] Test exit codes (0, 1, 2)

## Recommended Actions

1. **Add integration tests for Detector:**
   Create test fixtures with actual .tf files to test ScanDirectory() end-to-end

2. **Add unit tests for HCL parsing:**
   Use string literals with HCL content to test parsing functions

3. **Add formatter output tests:**
   Compare actual output with expected strings for JSON/CSV formats

4. **Add cmd tests:**
   Use test fixtures and table-driven tests for various scenarios

## Target Coverage: 70%+

With the above tests, we should reach 70%+ coverage, which is good for a CLI tool.
