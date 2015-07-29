// Copyright 2015 Google Inc. All Rights Reserved.
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

package keyserver

// validate performs correctness checking on each v2 message type.

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	v2pb "github.com/google/e2e-key-server/proto/v2"
)

// Maximum period of time to allow between CreationTime and server time.
const MaxClockDrift = 5 * time.Minute
const PGPAppID = "pgp"

var requiredScopes = []string{"https://www.googleapis.com/auth/userinfo.email"}

// validateEmail compares the given email against the one provided by GAIA
func (s *Server) validateEmail(ctx context.Context, email string) error {
	if err := s.a.CheckScopes(ctx, requiredScopes...); err != nil {
		return err
	}
	verifiedEmail, err := s.a.GetAuthenticatedEmail(ctx, requiredScopes...)
	if err != nil {
		return err
	}

	if verifiedEmail != email {
		return grpc.Errorf(codes.PermissionDenied, "wrong user")
	}
	return nil
}

// validateKey verifies
// - Key is valid for its format.
// - AppId is present.
// - Creation time is present and current.
func (s *Server) validateKey(userID string, key *v2pb.SignedKey_Key) (*Fingerprint, error) {
	if key == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Missing Key")
	}
	var fingerprint *Fingerprint
	if key.AppId == PGPAppID {
		var err error
		pgpUserID := fmt.Sprintf("<%v>", userID)
		fingerprint, err = validatePGP(pgpUserID, bytes.NewBuffer(key.Key))
		if err != nil {
			return nil, err
		}
	}
	if key.AppId == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Missing AppId")
	}
	// Validate CreationTime
	ct := key.GetCreationTime()
	if ct == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Missing CreationTime")
	}
	ct.Nanos = 0 // Quash nano seconds.
	if drift := time.Since(time.Unix(ct.Seconds, 0)); drift > MaxClockDrift {
		return nil, grpc.Errorf(codes.InvalidArgument, "CreationTime %v off", drift)
	}
	return fingerprint, nil
}

// validateSignedKey verifies
// - SignedKey is not nil.
// - Key is present and valid.
// - Signatures are valid.
// fills KeyId with the correct value.
func (s *Server) validateSignedKey(userID string, signedKey *v2pb.SignedKey) error {
	if signedKey == nil {
		return grpc.Errorf(codes.InvalidArgument, "Missing SignedKey")
	}
	// First validate interior fields and set KeyId.
	fingerprint, err := s.validateKey(userID, signedKey.GetKey())
	if err != nil {
		return err
	}
	signedKey.KeyId = hex.EncodeToString(fingerprint[:])
	return nil
}

func (s *Server) validateCreateKeyRequest(ctx context.Context, in *v2pb.CreateKeyRequest) error {
	// Validate proper authentication.
	if err := s.validateEmail(ctx, in.UserId); err != nil {
		return err
	}

	if err := s.validateSignedKey(in.UserId, in.GetSignedKey()); err != nil {
		return err
	}
	return nil
}

func (s *Server) validateUpdateKeyRequest(ctx context.Context, in *v2pb.UpdateKeyRequest) error {
	// Validate proper authentication.
	if err := s.validateEmail(ctx, in.UserId); err != nil {
		return err
	}

	if err := s.validateSignedKey(in.UserId, in.GetSignedKey()); err != nil {
		return err
	}
	return nil
}