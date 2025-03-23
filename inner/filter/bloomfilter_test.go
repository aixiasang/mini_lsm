package filter

import (
	"fmt"
	"testing"
)

// TestBloomFilterInterface ensures that BloomFilter implements the Filter interface
func TestBloomFilterInterface(t *testing.T) {
	var _ Filter = (*BloomFilter)(nil)
}

// TestBloomFilterAdd tests the Add and Contains methods
func TestBloomFilterAdd(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Test keys
	testKeys := [][]byte{
		[]byte("key1"),
		[]byte("key2"),
		[]byte("key3"),
		[]byte("this is a longer key"),
	}

	// Add keys to the filter
	for _, key := range testKeys {
		bf.Add(key)
	}

	// Test Contains on added keys (should return true)
	for _, key := range testKeys {
		if !bf.Contains(key) {
			t.Errorf("BloomFilter should contain key %s but returned false", key)
		}
	}

	// Test Contains on keys not added (might have false positives but generally should be false)
	nonExistingKeys := [][]byte{
		[]byte("nonexistent1"),
		[]byte("nonexistent2"),
		[]byte("this key was never added"),
	}

	falsePositives := 0
	for _, key := range nonExistingKeys {
		if bf.Contains(key) {
			falsePositives++
		}
	}

	// Check that false positive rate is reasonable
	falsePositiveRate := float64(falsePositives) / float64(len(nonExistingKeys))
	t.Logf("False positive rate: %f", falsePositiveRate)

	// Check the calculated false positive rate
	calculatedRate := bf.(*BloomFilter).FalsePositiveRate()
	t.Logf("Calculated false positive rate: %f", calculatedRate)
}

// TestBloomFilterSaveLoad tests the Save and Load methods
func TestBloomFilterSaveLoad(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Add some keys
	testKeys := [][]byte{
		[]byte("save1"),
		[]byte("save2"),
		[]byte("save3"),
	}

	for _, key := range testKeys {
		bf.Add(key)
	}

	// Save the filter to bytes
	data := bf.Save()
	if len(data) == 0 {
		t.Fatalf("Failed to save bloom filter: returned empty data")
	}

	// Test saving/loading with a specific test key
	testKey := []byte("save-test-key")
	bf.Add(testKey)
	if !bf.Contains(testKey) {
		t.Errorf("Filter should contain test key after adding")
	}

	// Load the filter into a new instance (should not contain the test key since we serialized before adding it)
	loadedBF := &BloomFilter{}
	err := loadedBF.Load(data)
	if err != nil {
		t.Fatalf("Failed to load bloom filter: %v", err)
	}

	// Check that the loaded filter contains the original keys
	for _, key := range testKeys {
		if !loadedBF.Contains(key) {
			t.Errorf("Loaded BloomFilter should contain key %s but returned false", key)
		}
	}

	// The test key shouldn't be in the loaded filter because it was added after serialization
	if loadedBF.Contains(testKey) {
		t.Errorf("Loaded filter should not contain test key that was added after serialization")
	}
}

// TestBloomFilterWithParams tests the NewBloomFilterWithParams constructor
func TestBloomFilterWithParams(t *testing.T) {
	// Create a filter with an expected 1000 items and 1% false positive rate
	bf := NewBloomFilterWithParams(1000, 0.01)

	// Type assertion to access BloomFilter specific methods
	bloomFilter := bf.(*BloomFilter)

	// The parameters should be optimized for 1000 items and 0.01 false positive rate
	t.Logf("Optimized size (m): %d", bloomFilter.m)
	t.Logf("Optimized hash functions (k): %d", bloomFilter.k)

	// Add 1000 items
	for i := 0; i < 1000; i++ {
		key := []byte(string(rune(i)))
		bf.Add(key)
	}

	// Check the false positive rate - should be close to 0.01
	calculatedRate := bloomFilter.FalsePositiveRate()
	t.Logf("Calculated false positive rate: %f", calculatedRate)

	// Verify it's reasonably close to our target
	if calculatedRate > 0.02 {
		t.Errorf("False positive rate too high: %f (should be close to 0.01)", calculatedRate)
	}
}

// TestBloomFilterReset tests the Reset method
func TestBloomFilterReset(t *testing.T) {
	bf := NewBloomFilter(1000, 3)

	// Add some keys
	testKeys := [][]byte{
		[]byte("reset1"),
		[]byte("reset2"),
		[]byte("reset3"),
	}

	for _, key := range testKeys {
		bf.Add(key)
	}

	// Check that keys are in the filter
	for _, key := range testKeys {
		if !bf.Contains(key) {
			t.Errorf("BloomFilter should contain key %s but returned false", key)
		}
	}

	// Reset the filter
	bf.Reset()

	// Check that filter was cleared - need type assertion to access n
	if bf.(*BloomFilter).n != 0 {
		t.Errorf("After reset, element count should be 0, got %d", bf.(*BloomFilter).n)
	}

	// Keys should no longer be in the filter
	for _, key := range testKeys {
		if bf.Contains(key) {
			t.Errorf("After reset, BloomFilter should not contain key %s but returned true", key)
		}
	}
}

// TestBloomFilterResetEnhanced provides a more comprehensive test of the Reset method
func TestBloomFilterResetEnhanced(t *testing.T) {
	// Create a bloom filter with a specific size for better testing
	m := uint64(64 * 10) // 10 words of 64 bits
	k := uint(3)
	bf := NewBloomFilter(m, k)
	bloomFilter := bf.(*BloomFilter) // Type assertion for access to internal fields

	// Add a significant number of keys to set many bits
	keyCount := 1000
	for i := 0; i < keyCount; i++ {
		key := []byte(fmt.Sprintf("test-key-%d", i))
		bf.Add(key)
	}

	// Verify some properties before reset
	if bloomFilter.n != uint64(keyCount) {
		t.Errorf("Expected element count to be %d, got %d", keyCount, bloomFilter.n)
	}

	// Check that at least some bits are set
	bitsSet := uint64(0)
	for _, word := range bloomFilter.bits {
		// Count bits set in each word using bit manipulation
		w := word
		for w > 0 {
			bitsSet++
			w &= w - 1 // Clear the least significant bit set
		}
	}
	if bitsSet == 0 {
		t.Error("Expected some bits to be set before reset, but none were")
	}
	t.Logf("Bits set before reset: %d", bitsSet)

	// Store some keys to verify they exist before reset and don't exist after
	verificationKeys := [][]byte{
		[]byte("test-key-10"),
		[]byte("test-key-200"),
		[]byte("test-key-500"),
		[]byte("test-key-999"),
	}

	// Verify keys exist before reset
	for _, key := range verificationKeys {
		if !bf.Contains(key) {
			t.Errorf("Key %s should exist before reset", key)
		}
	}

	// Reset the filter
	bf.Reset()

	// Verify element count is reset
	if bloomFilter.n != 0 {
		t.Errorf("After reset, element count should be 0, got %d", bloomFilter.n)
	}

	// Verify all bits are cleared
	for i, word := range bloomFilter.bits {
		if word != 0 {
			t.Errorf("After reset, all bits should be 0, but word %d has value %d", i, word)
		}
	}

	// Verify keys no longer exist after reset
	for _, key := range verificationKeys {
		if bf.Contains(key) {
			t.Errorf("After reset, key %s should not exist", key)
		}
	}

	// Verify we can still add and find keys after reset
	newKey := []byte("new-key-after-reset")
	bf.Add(newKey)
	if !bf.Contains(newKey) {
		t.Error("Should be able to add and find keys after reset")
	}
	if bloomFilter.n != 1 {
		t.Errorf("After adding one key post-reset, element count should be 1, got %d", bloomFilter.n)
	}
}
