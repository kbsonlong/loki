package v1

import (
	"hash"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/grafana/loki/pkg/util/encoding"
)

type BoundsCheck uint8

const (
	Before BoundsCheck = iota
	Overlap
	After
)

type FingerprintBounds struct {
	Min, Max model.Fingerprint
}

func NewBounds(min, max model.Fingerprint) FingerprintBounds {
	return FingerprintBounds{Min: min, Max: max}
}

func (b FingerprintBounds) Hash(h hash.Hash32) error {
	var enc encoding.Encbuf
	enc.PutBE64(uint64(b.Min))
	enc.PutBE64(uint64(b.Max))
	_, err := h.Write(enc.Get())
	return errors.Wrap(err, "writing OwnershipRange")
}

func (b FingerprintBounds) String() string {
	return b.Min.String() + "-" + b.Max.String()
}

// Cmp returns the fingerprint's position relative to the bounds
func (b FingerprintBounds) Cmp(fp model.Fingerprint) BoundsCheck {
	if fp < b.Min {
		return Before
	} else if fp > b.Max {
		return After
	}
	return Overlap
}

func (b FingerprintBounds) Overlaps(target FingerprintBounds) bool {
	return b.Cmp(target.Min) != After && b.Cmp(target.Max) != Before
}

// Slice returns a new fingerprint bounds clipped to the target bounds or nil if there is no overlap
func (b FingerprintBounds) Slice(min, max model.Fingerprint) *FingerprintBounds {
	return b.Intersection(FingerprintBounds{Min: min, Max: max})
}

// Returns whether the fingerprint is fully within the target bounds
func (b FingerprintBounds) Within(target FingerprintBounds) bool {
	return b.Min >= target.Min && b.Max <= target.Max
}

// Intersection returns the intersection of the two bounds
func (b FingerprintBounds) Intersection(target FingerprintBounds) *FingerprintBounds {
	if !b.Overlaps(target) {
		return nil
	}

	return &FingerprintBounds{
		Min: max(b.Min, target.Min),
		Max: min(b.Max, target.Max),
	}
}

// Union returns the union of the two bounds
func (b FingerprintBounds) Union(target FingerprintBounds) (res []FingerprintBounds) {
	if !b.Overlaps(target) {
		if b.Cmp(target.Min) == Before {
			return []FingerprintBounds{target, b}
		}
		return []FingerprintBounds{b, target}
	}

	return []FingerprintBounds{
		{
			Min: min(b.Min, target.Min),
			Max: max(b.Max, target.Max),
		},
	}
}

// Unless returns the subspace of itself which does not intersect with the target bounds
func (b FingerprintBounds) Unless(target FingerprintBounds) (res []FingerprintBounds) {
	if !b.Overlaps(target) {
		return []FingerprintBounds{b}
	}

	if b == target {
		return nil
	}

	if b.Min < target.Min {
		res = append(res, FingerprintBounds{Min: b.Min, Max: min(b.Max, target.Min-1)})
	}
	if target.Max < b.Max {
		res = append(res, FingerprintBounds{Min: max(b.Min, target.Max+1), Max: b.Max})
	}
	return res
}

// unused, but illustrative
type BoundedIter[V any] struct {
	Iterator[V]
	cmp func(V) BoundsCheck
}

func (bi *BoundedIter[V]) Next() bool {
	for bi.Iterator.Next() {
		switch bi.cmp(bi.Iterator.At()) {
		case Before:
			continue
		case After:
			return false
		default:
			return true
		}
	}
	return false
}

func NewBoundedIter[V any](itr Iterator[V], cmp func(V) BoundsCheck) *BoundedIter[V] {
	return &BoundedIter[V]{Iterator: itr, cmp: cmp}
}
