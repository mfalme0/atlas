package trie

import "testing"

func TestInsertSearch(t *testing.T) {
	trie := New()
	words := []string{"apple", "app", "application", "bat", "ball"}
	for _, w := range words {
		trie.Insert(w)
	}

	for _, w := range words {
		if !trie.Search(w) {
			t.Errorf("expected to find '%s'", w)
		}
	}
	if trie.Search("ap") {
		t.Error("expected false for 'ap' (not a complete word)")
	}
	if trie.Search("xyz") {
		t.Error("expected false for 'xyz'")
	}
}

func TestStartsWith(t *testing.T) {
	trie := New()
	trie.Insert("apple")
	trie.Insert("application")

	if !trie.StartsWith("app") {
		t.Error("expected true for prefix 'app'")
	}
	if !trie.StartsWith("appl") {
		t.Error("expected true for prefix 'appl'")
	}
	if trie.StartsWith("b") {
		t.Error("expected false for prefix 'b'")
	}
}

func TestDelete(t *testing.T) {
	trie := New()
	trie.Insert("apple")
	trie.Insert("app")
	trie.Insert("application")

	if !trie.Delete("apple") {
		t.Error("expected true for deleting 'apple'")
	}
	if trie.Search("apple") {
		t.Error("expected false after deleting 'apple'")
	}
	if !trie.Search("app") {
		t.Error("expected 'app' to still exist")
	}
	if !trie.Search("application") {
		t.Error("expected 'application' to still exist")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	trie := New()
	trie.Insert("hello")
	if trie.Delete("world") {
		t.Error("expected false for deleting non-existent word")
	}
}

func TestWordsWithPrefix(t *testing.T) {
	trie := New()
	trie.Insert("cat")
	trie.Insert("car")
	trie.Insert("card")
	trie.Insert("care")
	trie.Insert("dog")

	words := trie.WordsWithPrefix("car")
	if len(words) != 3 {
		t.Errorf("expected 3 words with prefix 'car', got %d: %v", len(words), words)
	}
}

func TestWordsWithPrefixEmpty(t *testing.T) {
	trie := New()
	words := trie.WordsWithPrefix("xyz")
	if len(words) != 0 {
		t.Errorf("expected 0 words, got %d", len(words))
	}
}

func TestSize(t *testing.T) {
	trie := New()
	if trie.Size() != 0 {
		t.Errorf("expected size 0, got %d", trie.Size())
	}
	trie.Insert("a")
	trie.Insert("b")
	trie.Insert("a")
	if trie.Size() != 2 {
		t.Errorf("expected size 2 (no duplicates), got %d", trie.Size())
	}
}

func TestIsEmpty(t *testing.T) {
	trie := New()
	if !trie.IsEmpty() {
		t.Error("expected empty")
	}
	trie.Insert("x")
	if trie.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestGetAll(t *testing.T) {
	trie := New()
	trie.Insert("a")
	trie.Insert("b")
	trie.Insert("c")

	all := trie.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 words, got %d", len(all))
	}
}

func TestInsertWithValue(t *testing.T) {
	trie := New()
	trie.InsertWithValue("hello", 42)

	if !trie.Search("hello") {
		t.Error("expected to find 'hello'")
	}
}

func TestSingleChar(t *testing.T) {
	trie := New()
	trie.Insert("a")
	trie.Insert("b")

	if !trie.Search("a") {
		t.Error("expected to find 'a'")
	}
	if !trie.StartsWith("a") {
		t.Error("expected prefix 'a'")
	}
}

func BenchmarkTrieInsert(b *testing.B) {
	trie := New()
	words := []string{"apple", "application", "banana", "bat", "ball", "cat", "car"}
	for i := 0; i < b.N; i++ {
		trie.Insert(words[i%len(words)]+string(rune('0'+i%10)))
	}
}

func BenchmarkTrieSearch(b *testing.B) {
	trie := New()
	for i := 0; i < 10000; i++ {
		trie.Insert("word" + string(rune(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Search("word" + string(rune(i%10000)))
	}
}

func BenchmarkTriePrefixSearch(b *testing.B) {
	trie := New()
	for i := 0; i < 10000; i++ {
		trie.Insert("test" + string(rune(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.WordsWithPrefix("test")
	}
}
