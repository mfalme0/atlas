// Package trie implements a trie (prefix tree) for string operations.
//
// Operations:
//
//	Insert:    O(m) where m is the key length
//	Search:    O(m)
//	Delete:    O(m)
//	PrefixSearch: O(p + k) where p is prefix length, k is number of matches
//
// Use cases: autocomplete, spell checking, IP routing tables,
// dictionary implementations, string prefix matching.
package trie

// TrieNode is a node in the trie.
type TrieNode struct {
	Children map[rune]*TrieNode
	IsEnd    bool
	Value    interface{}
}

// Trie is a prefix tree for string storage and retrieval.
type Trie struct {
	Root *TrieNode
	size int
}

// New creates a new empty trie.
func New() *Trie {
	return &Trie{
		Root: &TrieNode{Children: make(map[rune]*TrieNode)},
	}
}

// Insert adds a word to the trie. O(m).
func (t *Trie) Insert(word string) {
	node := t.Root
	for _, ch := range word {
		if _, ok := node.Children[ch]; !ok {
			node.Children[ch] = &TrieNode{Children: make(map[rune]*TrieNode)}
		}
		node = node.Children[ch]
	}
	if !node.IsEnd {
		t.size++
	}
	node.IsEnd = true
}

// InsertWithValue adds a word with an associated value. O(m).
func (t *Trie) InsertWithValue(word string, value interface{}) {
	node := t.Root
	for _, ch := range word {
		if _, ok := node.Children[ch]; !ok {
			node.Children[ch] = &TrieNode{Children: make(map[rune]*TrieNode)}
		}
		node = node.Children[ch]
	}
	if !node.IsEnd {
		t.size++
	}
	node.IsEnd = true
	node.Value = value
}

// Search returns true if the exact word exists in the trie. O(m).
func (t *Trie) Search(word string) bool {
	node := t.Root
	for _, ch := range word {
		if _, ok := node.Children[ch]; !ok {
			return false
		}
		node = node.Children[ch]
	}
	return node.IsEnd
}

// StartsWith returns true if any word in the trie starts with the given prefix. O(m).
func (t *Trie) StartsWith(prefix string) bool {
	node := t.Root
	for _, ch := range prefix {
		if _, ok := node.Children[ch]; !ok {
			return false
		}
		node = node.Children[ch]
	}
	return true
}

// Delete removes a word from the trie. O(m).
func (t *Trie) Delete(word string) bool {
	found, _ := t.delete(t.Root, word, 0)
	if found {
		t.size--
	}
	return found
}

func (t *Trie) delete(node *TrieNode, word string, depth int) (bool, bool) {
	if node == nil {
		return false, false
	}

	if depth == len(word) {
		if !node.IsEnd {
			return false, false
		}
		node.IsEnd = false
		node.Value = nil
		return true, len(node.Children) == 0
	}

	ch := rune(word[depth])
	child, ok := node.Children[ch]
	if !ok {
		return false, false
	}

	found, shouldDeleteChild := t.delete(child, word, depth+1)

	if shouldDeleteChild {
		delete(node.Children, ch)
		return found, !node.IsEnd && len(node.Children) == 0
	}

	return found, false
}

// WordsWithPrefix returns all words that start with the given prefix. O(p + k).
func (t *Trie) WordsWithPrefix(prefix string) []string {
	node := t.Root
	for _, ch := range prefix {
		if _, ok := node.Children[ch]; !ok {
			return nil
		}
		node = node.Children[ch]
	}

	var results []string
	t.collectWords(node, []rune(prefix), &results)
	return results
}

func (t *Trie) collectWords(node *TrieNode, current []rune, results *[]string) {
	if node.IsEnd {
		*results = append(*results, string(current))
	}
	for ch, child := range node.Children {
		t.collectWords(child, append(current, ch), results)
	}
}

// Size returns the number of words in the trie. O(1).
func (t *Trie) Size() int {
	return t.size
}

// IsEmpty returns true if the trie has no words. O(1).
func (t *Trie) IsEmpty() bool {
	return t.size == 0
}

// GetAll returns all words in the trie. O(n).
func (t *Trie) GetAll() []string {
	var results []string
	t.collectWords(t.Root, nil, &results)
	return results
}
