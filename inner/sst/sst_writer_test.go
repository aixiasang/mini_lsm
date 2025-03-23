package sst

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/aixiasang/lsm/inner/config"
)

func TestSSTWriter(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sst_writer_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom config for testing with smaller block size
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	// Use a larger block size so we don't immediately rotate
	conf.BlockSize = 5

	// Initialize a new SSTWriter
	sstFile := filepath.Join(tempDir, "test.sst")
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTWriter: %v", err)
	}

	// Add some key-value pairs
	testData := map[string]string{
		"key1":   "value1",
		"key2":   "value2",
		"key3":   "value3",
		"key100": "value100",
		"key200": "value200",
		"key300": "value300",
	}

	for k, v := range testData {
		err := writer.Add([]byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Failed to add key-value pair: %v", err)
		}
	}

	// Force a rotation to create at least one index entry
	if err := writer.mustRotateDataBlock(); err != nil {
		t.Fatalf("Failed to force data block rotation: %v", err)
	}

	// Check if blocks were created
	if len(writer.index) == 0 {
		t.Error("Expected at least one index entry after rotation, but none found")
	}

	// Add more data to ensure multiple blocks
	for i := 0; i < 20; i++ {
		key := []byte(string(rune('a' + i)))
		value := []byte(string(rune('A' + i)))
		err := writer.Add(key, value)
		if err != nil {
			t.Fatalf("Failed to add additional key-value pair: %v", err)
		}
	}

	// Force rotation again
	if err := writer.mustRotateDataBlock(); err != nil {
		t.Fatalf("Failed to force data block rotation: %v", err)
	}

	// Verify that multiple blocks were created
	if len(writer.index) < 2 {
		t.Errorf("Expected multiple index entries after adding data, got %d", len(writer.index))
	}

	// Test the flush method
	err = writer.Flush()
	if err != nil {
		t.Fatalf("Failed to flush writer: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Verify the SST file was created and has content
	fileInfo, err := os.Stat(sstFile)
	if err != nil {
		t.Fatalf("Failed to stat SST file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Error("SST file is empty after flush")
	}

	// Read the file and verify its structure
	content, err := os.ReadFile(sstFile)
	if err != nil {
		t.Fatalf("Failed to read SST file: %v", err)
	}

	// The footer should have 3 uint32 values (data length, index length, filter length)
	if len(content) < 12 {
		t.Fatalf("SST file content too small: %d bytes", len(content))
	}

	// Read the footer values
	footer := content[len(content)-12:]
	dataLength := binary.BigEndian.Uint32(footer[0:4])
	indexLength := binary.BigEndian.Uint32(footer[4:8])
	filterLength := binary.BigEndian.Uint32(footer[8:12])

	// Verify that the lengths make sense
	if dataLength == 0 || indexLength == 0 || filterLength == 0 {
		t.Errorf("Invalid footer values: dataLength=%d, indexLength=%d, filterLength=%d",
			dataLength, indexLength, filterLength)
	}

	t.Logf("SST file analysis: size=%d, dataLength=%d, indexLength=%d, filterLength=%d",
		fileInfo.Size(), dataLength, indexLength, filterLength)
}

// TestSSTWriterBloomFilter tests that the bloom filter is being used correctly
func TestSSTWriterBloomFilter(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sst_writer_bloom_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom config for testing with smaller block size
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 3 // Small block size to test filter persistence

	// Initialize a new SSTWriter
	sstFile := filepath.Join(tempDir, "bloom_test.sst")
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTWriter: %v", err)
	}

	// Add keys to the bloom filter directly
	testKeys := [][]byte{
		[]byte("filter_key1"),
		[]byte("filter_key2"),
		[]byte("filter_key3"),
	}

	// Add keys to writer
	for _, key := range testKeys {
		if err := writer.Add(key, []byte("value")); err != nil {
			t.Fatalf("Failed to add key to writer: %v", err)
		}
	}

	// Force a block rotation to capture the current filter
	err = writer.mustRotateDataBlock()
	if err != nil {
		t.Fatalf("Failed to rotate data block: %v", err)
	}

	// Check if we have filter data
	if len(writer.mapFilter) == 0 {
		t.Fatal("Expected at least one filter entry after rotation, but none found")
	}

	// Find the filter for the first block length
	var filterData []byte
	var found bool
	for _, fData := range writer.mapFilter {
		filterData = fData
		found = true
		break
	}

	if !found || len(filterData) == 0 {
		t.Fatal("Expected filter data for a block, but none found")
	}

	// Verify filter contents
	// Load the filter data into a new filter
	loadedFilter := conf.FilterConstructor(1024, 3)
	if err := loadedFilter.Load(filterData); err != nil {
		t.Fatalf("Failed to load filter data: %v", err)
	}

	// Test that the filter contains the test keys
	for _, key := range testKeys {
		if !loadedFilter.Contains(key) {
			t.Errorf("Filter should contain key %s, but it doesn't", key)
		}
	}

	// Test that the filter is properly reset after rotation
	// Add a new key to the current block's filter
	newKey := []byte("new_test_key")
	writer.filter.Add(newKey)

	// This key should be in the current filter but not in the saved one
	if !writer.filter.Contains(newKey) {
		t.Error("Current filter should contain new key, but it doesn't")
	}

	if loadedFilter.Contains(newKey) {
		t.Error("Saved filter should not contain new key added after rotation")
	}

	// Flush all data and close
	if err := writer.Flush(); err != nil {
		t.Fatalf("Failed to flush writer: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Read the file to verify content
	content, err := os.ReadFile(sstFile)
	if err != nil {
		t.Fatalf("Failed to read SST file: %v", err)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(sstFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	// The footer should have 3 uint32 values (data length, index length, filter length)
	if len(content) < 12 {
		t.Fatalf("SST file content too small: %d bytes", len(content))
	}

	// Read the footer values
	footer := content[len(content)-12:]
	dataLength := binary.BigEndian.Uint32(footer[0:4])
	indexLength := binary.BigEndian.Uint32(footer[4:8])
	filterLength := binary.BigEndian.Uint32(footer[8:12])

	// Verify that the lengths make sense
	if dataLength == 0 || indexLength == 0 || filterLength == 0 {
		t.Errorf("Invalid footer values: dataLength=%d, indexLength=%d, filterLength=%d",
			dataLength, indexLength, filterLength)
	}

	t.Logf("SST file analysis: size=%d, dataLength=%d, indexLength=%d, filterLength=%d",
		fileInfo.Size(), dataLength, indexLength, filterLength)
}

// Test that block rotation works properly when the block size is reached
func TestSSTWriterBlockRotation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sst_writer_rotation_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom config for testing with smaller block size
	conf := config.DefaultConfig()
	conf.DataDir = tempDir
	conf.BlockSize = 2 // Very small block size to force rotations

	// Initialize a new SSTWriter
	sstFile := filepath.Join(tempDir, "rotation_test.sst")
	writer, err := NewSSTWriter(conf, sstFile)
	if err != nil {
		t.Fatalf("Failed to create SSTWriter: %v", err)
	}

	// Add exactly 3 entries (should not trigger rotation yet)
	for i := 0; i < 3; i++ {
		key := []byte(string(rune('a' + i)))
		value := []byte(string(rune('A' + i)))
		if err := writer.Add(key, value); err != nil {
			t.Fatalf("Failed to add key-value pair: %v", err)
		}
	}

	// Force the rotation
	if err := writer.mustRotateDataBlock(); err != nil {
		t.Fatalf("Failed to force rotation: %v", err)
	}

	// Now should have 1 index entry after forced rotation
	if len(writer.index) != 1 {
		t.Errorf("Expected 1 index entry after forced rotation, got %d", len(writer.index))
	}

	// Add several more entries
	for i := 0; i < 9; i++ {
		key := []byte(string(rune('A' + i)))
		value := []byte(string(rune('a' + i)))
		if err := writer.Add(key, value); err != nil {
			t.Fatalf("Failed to add additional key-value pair: %v", err)
		}
	}

	// Force final rotation to ensure all data is captured
	if err := writer.mustRotateDataBlock(); err != nil {
		t.Fatalf("Failed to force final rotation: %v", err)
	}

	// Should have multiple index entries now (initial + 3 more sets of 3 entries)
	expectedIndices := 1 + (9 / 3)
	if len(writer.index) != expectedIndices {
		t.Errorf("Expected %d index entries after rotations, got %d",
			expectedIndices, len(writer.index))
	}

	// Flush to file
	if err := writer.Flush(); err != nil {
		t.Fatalf("Failed to flush writer: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Verify the contents with a file reader
	content, err := os.ReadFile(sstFile)
	if err != nil {
		t.Fatalf("Failed to read SST file: %v", err)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(sstFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	// The footer should have 3 uint32 values (data length, index length, filter length)
	if len(content) < 12 {
		t.Fatalf("SST file content too small: %d bytes", len(content))
	}

	// Read the footer values
	footer := content[len(content)-12:]
	dataLength := binary.BigEndian.Uint32(footer[0:4])
	indexLength := binary.BigEndian.Uint32(footer[4:8])
	filterLength := binary.BigEndian.Uint32(footer[8:12])

	// Verify that the lengths make sense
	if dataLength == 0 || indexLength == 0 || filterLength == 0 {
		t.Errorf("Invalid footer values: dataLength=%d, indexLength=%d, filterLength=%d",
			dataLength, indexLength, filterLength)
	}

	t.Logf("SST file analysis: size=%d, dataLength=%d, indexLength=%d, filterLength=%d",
		fileInfo.Size(), dataLength, indexLength, filterLength)
}
