// Package tree implements binary search tree and AVL tree.
//
// BST:
//
//	Insert:  O(h) where h is tree height
//	Search:  O(h)
//	Delete:  O(h)
//	Best:    O(log n) average
//	Worst:   O(n) for degenerate input
//
// AVL Tree (self-balancing BST):
//
//	Insert:  O(log n) guaranteed
//	Search:  O(log n) guaranteed
//	Delete:  O(log n) guaranteed
//	Uses rotations to maintain balance factor in [-1, 1].
//
// Use cases: sorted data, range queries, ordered maps, database indexing.
package tree

// BSTNode is a node in a binary search tree.
type BSTNode[T any] struct {
	Value T
	Left  *BSTNode[T]
	Right *BSTNode[T]
}

// BST is a binary search tree.
type BST[T any] struct {
	Root *BSTNode[T]
	cmp  func(a, b T) int
	size int
}

// NewBST creates a new binary search tree.
func NewBST[T any](cmp func(a, b T) int) *BST[T] {
	return &BST[T]{cmp: cmp}
}

// Insert adds a value to the BST. O(h).
func (t *BST[T]) Insert(value T) {
	var inserted bool
	t.Root, inserted = t.insert(t.Root, value)
	if inserted {
		t.size++
	}
}

func (t *BST[T]) insert(node *BSTNode[T], value T) (*BSTNode[T], bool) {
	if node == nil {
		return &BSTNode[T]{Value: value}, true
	}
	cmp := t.cmp(value, node.Value)
	if cmp < 0 {
		left, inserted := t.insert(node.Left, value)
		node.Left = left
		return node, inserted
	} else if cmp > 0 {
		right, inserted := t.insert(node.Right, value)
		node.Right = right
		return node, inserted
	}
	return node, false
}

// Search returns true if the value exists. O(h).
func (t *BST[T]) Search(value T) bool {
	return t.search(t.Root, value)
}

func (t *BST[T]) search(node *BSTNode[T], value T) bool {
	if node == nil {
		return false
	}
	cmp := t.cmp(value, node.Value)
	if cmp == 0 {
		return true
	} else if cmp < 0 {
		return t.search(node.Left, value)
	}
	return t.search(node.Right, value)
}

// Inorder returns the values in sorted order. O(n).
func (t *BST[T]) Inorder() []T {
	result := make([]T, 0, t.size)
	t.inorder(t.Root, &result)
	return result
}

func (t *BST[T]) inorder(node *BSTNode[T], result *[]T) {
	if node == nil {
		return
	}
	t.inorder(node.Left, result)
	*result = append(*result, node.Value)
	t.inorder(node.Right, result)
}

// Size returns the number of nodes. O(1).
func (t *BST[T]) Size() int {
	return t.size
}

// Height returns the height of the tree. O(n).
func (t *BST[T]) Height() int {
	return t.height(t.Root)
}

func (t *BST[T]) height(node *BSTNode[T]) int {
	if node == nil {
		return 0
	}
	l := t.height(node.Left)
	r := t.height(node.Right)
	if l > r {
		return l + 1
	}
	return r + 1
}

// AVLNode is a node in an AVL tree.
type AVLNode[T any] struct {
	Value  T
	Left   *AVLNode[T]
	Right  *AVLNode[T]
	Height int
}

// AVLTree is a self-balancing binary search tree.
type AVLTree[T any] struct {
	Root *AVLNode[T]
	cmp  func(a, b T) int
	size int
}

// NewAVL creates a new AVL tree.
func NewAVL[T any](cmp func(a, b T) int) *AVLTree[T] {
	return &AVLTree[T]{cmp: cmp}
}

func (t *AVLTree[T]) nodeHeight(node *AVLNode[T]) int {
	if node == nil {
		return 0
	}
	return node.Height
}

func (t *AVLTree[T]) balanceFactor(node *AVLNode[T]) int {
	if node == nil {
		return 0
	}
	return t.nodeHeight(node.Left) - t.nodeHeight(node.Right)
}

func (t *AVLTree[T]) updateHeight(node *AVLNode[T]) {
	lh := t.nodeHeight(node.Left)
	rh := t.nodeHeight(node.Right)
	if lh > rh {
		node.Height = lh + 1
	} else {
		node.Height = rh + 1
	}
}

// rightRotate performs a right rotation.
//
//	    y               x
//	   / \             / \
//	  x   C   =>    A   y
// / \                 / \
// A   B              B   C
func (t *AVLTree[T]) rightRotate(y *AVLNode[T]) *AVLNode[T] {
	x := y.Left
	b := x.Right
	x.Right = y
	y.Left = b
	t.updateHeight(y)
	t.updateHeight(x)
	return x
}

// leftRotate performs a left rotation.
//
//	  x               y
// / \             / \
// A   y     =>    x   C
//   / \         / \
//  B   C       A   B
func (t *AVLTree[T]) leftRotate(x *AVLNode[T]) *AVLNode[T] {
	y := x.Right
	b := y.Left
	y.Left = x
	x.Right = b
	t.updateHeight(x)
	t.updateHeight(y)
	return y
}

// Insert adds a value and rebalances. O(log n) guaranteed.
func (t *AVLTree[T]) Insert(value T) {
	var inserted bool
	t.Root, inserted = t.insert(t.Root, value)
	if inserted {
		t.size++
	}
}

func (t *AVLTree[T]) insert(node *AVLNode[T], value T) (*AVLNode[T], bool) {
	if node == nil {
		return &AVLNode[T]{Value: value, Height: 1}, true
	}

	cmp := t.cmp(value, node.Value)
	if cmp < 0 {
		var inserted bool
		node.Left, inserted = t.insert(node.Left, value)
		if inserted {
			t.updateHeight(node)
			return t.balance(node), true
		}
		return node, false
	} else if cmp > 0 {
		var inserted bool
		node.Right, inserted = t.insert(node.Right, value)
		if inserted {
			t.updateHeight(node)
			return t.balance(node), true
		}
		return node, false
	}

	return node, false
}

func (t *AVLTree[T]) balance(node *AVLNode[T]) *AVLNode[T] {
	bf := t.balanceFactor(node)

	if bf > 1 {
		if t.balanceFactor(node.Left) < 0 {
			node.Left = t.leftRotate(node.Left)
		}
		return t.rightRotate(node)
	}

	if bf < -1 {
		if t.balanceFactor(node.Right) > 0 {
			node.Right = t.rightRotate(node.Right)
		}
		return t.leftRotate(node)
	}

	return node
}

// Search returns true if the value exists. O(log n) guaranteed.
func (t *AVLTree[T]) Search(value T) bool {
	return t.search(t.Root, value)
}

func (t *AVLTree[T]) search(node *AVLNode[T], value T) bool {
	if node == nil {
		return false
	}
	cmp := t.cmp(value, node.Value)
	if cmp == 0 {
		return true
	} else if cmp < 0 {
		return t.search(node.Left, value)
	}
	return t.search(node.Right, value)
}

// Inorder returns the values in sorted order. O(n).
func (t *AVLTree[T]) Inorder() []T {
	result := make([]T, 0, t.size)
	t.inorder(t.Root, &result)
	return result
}

func (t *AVLTree[T]) inorder(node *AVLNode[T], result *[]T) {
	if node == nil {
		return
	}
	t.inorder(node.Left, result)
	*result = append(*result, node.Value)
	t.inorder(node.Right, result)
}

// Size returns the number of nodes. O(1).
func (t *AVLTree[T]) Size() int {
	return t.size
}

// Height returns the height of the tree. O(1).
func (t *AVLTree[T]) Height() int {
	return t.nodeHeight(t.Root)
}

// IsBalanced returns true if the tree is balanced. O(n).
func (t *AVLTree[T]) IsBalanced() bool {
	return t.isBalanced(t.Root)
}

func (t *AVLTree[T]) isBalanced(node *AVLNode[T]) bool {
	if node == nil {
		return true
	}
	bf := t.balanceFactor(node)
	if bf < -1 || bf > 1 {
		return false
	}
	return t.isBalanced(node.Left) && t.isBalanced(node.Right)
}
