package tree

import "testing"

func intCmp(a, b int) int { return a - b }

func TestBSTInsertSearch(t *testing.T) {
	bst := NewBST[int](intCmp)
	values := []int{5, 3, 7, 1, 4, 6, 8}
	for _, v := range values {
		bst.Insert(v)
	}

	for _, v := range values {
		if !bst.Search(v) {
			t.Errorf("expected to find %d", v)
		}
	}
	if bst.Search(99) {
		t.Error("expected false for 99")
	}
}

func TestBSTInorder(t *testing.T) {
	bst := NewBST[int](intCmp)
	for _, v := range []int{5, 3, 7, 1, 4} {
		bst.Insert(v)
	}

	sorted := bst.Inorder()
	expected := []int{1, 3, 4, 5, 7}
	if len(sorted) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(sorted))
	}
	for i, v := range sorted {
		if v != expected[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestBSTSize(t *testing.T) {
	bst := NewBST[int](intCmp)
	if bst.Size() != 0 {
		t.Errorf("expected size 0, got %d", bst.Size())
	}
	bst.Insert(1)
	bst.Insert(2)
	if bst.Size() != 2 {
		t.Errorf("expected size 2, got %d", bst.Size())
	}
}

func TestBSTHeight(t *testing.T) {
	bst := NewBST[int](intCmp)
	for _, v := range []int{1, 2, 3, 4, 5} {
		bst.Insert(v)
	}
	h := bst.Height()
	if h != 5 {
		t.Errorf("expected height 5 for degenerate tree, got %d", h)
	}
}

func TestBSTDegenerate(t *testing.T) {
	bst := NewBST[int](intCmp)
	for i := 0; i < 100; i++ {
		bst.Insert(i)
	}
	h := bst.Height()
	if h != 100 {
		t.Errorf("expected height 100 for degenerate tree, got %d", h)
	}
	sorted := bst.Inorder()
	for i := 0; i < 100; i++ {
		if sorted[i] != i {
			t.Errorf("expected %d at index %d, got %d", i, i, sorted[i])
		}
	}
}

func TestAVLInsertSearch(t *testing.T) {
	avl := NewAVL[int](intCmp)
	values := []int{5, 3, 7, 1, 4, 6, 8}
	for _, v := range values {
		avl.Insert(v)
	}

	for _, v := range values {
		if !avl.Search(v) {
			t.Errorf("expected to find %d", v)
		}
	}
	if avl.Search(99) {
		t.Error("expected false for 99")
	}
}

func TestAVLInorder(t *testing.T) {
	avl := NewAVL[int](intCmp)
	for _, v := range []int{5, 3, 7, 1, 4} {
		avl.Insert(v)
	}

	sorted := avl.Inorder()
	expected := []int{1, 3, 4, 5, 7}
	for i, v := range sorted {
		if v != expected[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestAVLIsBalanced(t *testing.T) {
	avl := NewAVL[int](intCmp)
	for i := 0; i < 1000; i++ {
		avl.Insert(i)
	}

	if !avl.IsBalanced() {
		t.Error("AVL tree should always be balanced")
	}
}

func TestAVLHeight(t *testing.T) {
	avl := NewAVL[int](intCmp)
	for i := 0; i < 1000; i++ {
		avl.Insert(i)
	}

	h := avl.Height()
	if h > 20 {
		t.Errorf("AVL height for 1000 nodes should be ~10-15, got %d", h)
	}
}

func TestAVLRepeatedInsert(t *testing.T) {
	avl := NewAVL[int](intCmp)
	for i := 0; i < 100; i++ {
		avl.Insert(50)
	}
	if avl.Size() != 1 {
		t.Errorf("expected size 1 for duplicate inserts, got %d", avl.Size())
	}
}

func TestAVLBalancedAfterSorted(t *testing.T) {
	avl := NewAVL[int](intCmp)
	for i := 0; i < 100; i++ {
		avl.Insert(i)
	}
	if !avl.IsBalanced() {
		t.Error("AVL should be balanced even with sorted input")
	}
	if avl.Height() > 15 {
		t.Errorf("expected height < 15, got %d", avl.Height())
	}
}

func BenchmarkBSTInsert(b *testing.B) {
	bst := NewBST[int](intCmp)
	for i := 0; i < b.N; i++ {
		bst.Insert(i)
	}
}

func BenchmarkBSTSearch(b *testing.B) {
	bst := NewBST[int](intCmp)
	for i := 0; i < 10000; i++ {
		bst.Insert(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bst.Search(i % 10000)
	}
}

func BenchmarkAVLInsert(b *testing.B) {
	avl := NewAVL[int](intCmp)
	for i := 0; i < b.N; i++ {
		avl.Insert(i)
	}
}

func BenchmarkAVLSearch(b *testing.B) {
	avl := NewAVL[int](intCmp)
	for i := 0; i < 10000; i++ {
		avl.Insert(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		avl.Search(i % 10000)
	}
}
