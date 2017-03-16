// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package mutator defines the operations to transform mutations into changes in
// the map as well as operations to write and read mutations to and from the
// database.
package mutator

import (
	"context"
	"errors"

	"github.com/google/keytransparency/core/transaction"
)

var (
	// MaxMutationSize represent the maximum allowed mutation size in bytes.
	MaxMutationSize = 16 * 1024
	// ErrReplay occurs when two mutations acting on the same entry & epoch
	// occur.
	ErrReplay = errors.New("mutation replay")
	// ErrSize occurs when the mutation size is larger than the allowed upper
	// bound.
	ErrSize = errors.New("mutation is too large")
	// ErrPreviousHash occurs when the mutation the hash of the previous
	// entry provided in the mutation does not match the previous entry
	// itself.
	ErrPreviousHash = errors.New("previous entry hash does not match the hash provided in the mutation")
	// ErrMissingKey occurs when a mutation does not have authorized keys.
	ErrMissingKey = errors.New("missing authorized key(s)")
	// ErrInvalidSig occurs when either the current or previous update entry
	// signature verification fails.
	ErrInvalidSig = errors.New("invalid signature")
)

// Mutator verifies mutations and transforms values in the map.
type Mutator interface {
	// CheckMutation verifies that this is a valid mutation for this item.
	CheckMutation(value, mutation []byte) error
	// Mutate applies mutation to value
	Mutate(value, mutation []byte) ([]byte, error)
}

// Mutation reads and writes mutations to the database.
type Mutation interface {
	// Read reads all mutations for a specific given mapID, epoch, and index.
	Read(ctx context.Context, txn transaction.Txn, epoch int64, index []byte) ([][]byte, error)
	// Write saves the mutation in the database.
	Write(ctx context.Context, txn transaction.Txn, epoch int64, index, mutation []byte) error
}
