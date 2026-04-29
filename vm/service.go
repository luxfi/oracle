// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/rpc/v2"
	grjson "github.com/gorilla/rpc/v2/json"

	"github.com/luxfi/ids"
)

// Service provides RPC access to the OracleVM
type Service struct {
	vm *VM
}

// NewService creates a new OracleVM service
func NewService(vm *VM) http.Handler {
	server := rpc.NewServer()
	server.RegisterCodec(grjson.NewCodec(), "application/json")
	server.RegisterCodec(grjson.NewCodec(), "application/json;charset=UTF-8")
	server.RegisterService(&Service{vm: vm}, "oracle")
	return server
}

// RegisterFeedArgs are arguments for RegisterFeed
type RegisterFeedArgs struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Sources     []string          `json:"sources"`
	UpdateFreq  string            `json:"updateFreq"`
	Operators   []string          `json:"operators"`
	Metadata    map[string]string `json:"metadata"`
}

// RegisterFeedReply is the reply for RegisterFeed
type RegisterFeedReply struct {
	FeedID string `json:"feedId"`
}

// RegisterFeed registers a new oracle feed
func (s *Service) RegisterFeed(r *http.Request, args *RegisterFeedArgs, reply *RegisterFeedReply) error {
	// Generate feed ID
	feedBytes, _ := json.Marshal(args)
	feedID := ids.ID{}
	copy(feedID[:], feedBytes[:32])

	feed := &Feed{
		ID:          feedID,
		Name:        args.Name,
		Description: args.Description,
		Sources:     args.Sources,
		Metadata:    args.Metadata,
	}

	if err := s.vm.RegisterFeed(feed); err != nil {
		return err
	}

	reply.FeedID = feedID.String()
	return nil
}

// GetFeedArgs are arguments for GetFeed
type GetFeedArgs struct {
	FeedID string `json:"feedId"`
}

// GetFeedReply is the reply for GetFeed
type GetFeedReply struct {
	Feed *Feed `json:"feed"`
}

// GetFeed returns a feed by ID
func (s *Service) GetFeed(r *http.Request, args *GetFeedArgs, reply *GetFeedReply) error {
	feedID, err := ids.FromString(args.FeedID)
	if err != nil {
		return err
	}

	feed, err := s.vm.GetFeed(feedID)
	if err != nil {
		return err
	}

	reply.Feed = feed
	return nil
}

// GetValueArgs are arguments for GetValue
type GetValueArgs struct {
	FeedID string `json:"feedId"`
}

// GetValueReply is the reply for GetValue
type GetValueReply struct {
	Value *AggregatedValue `json:"value"`
}

// GetValue returns the latest value for a feed
func (s *Service) GetValue(r *http.Request, args *GetValueArgs, reply *GetValueReply) error {
	feedID, err := ids.FromString(args.FeedID)
	if err != nil {
		return err
	}

	value, err := s.vm.GetLatestValue(feedID)
	if err != nil {
		return err
	}

	reply.Value = value
	return nil
}

// SubmitObservationArgs are arguments for SubmitObservation
type SubmitObservationArgs struct {
	FeedID    string `json:"feedId"`
	Value     []byte `json:"value"`
	Signature []byte `json:"signature"`
}

// SubmitObservationReply is the reply for SubmitObservation
type SubmitObservationReply struct {
	Success bool `json:"success"`
}

// SubmitObservation submits an observation
func (s *Service) SubmitObservation(r *http.Request, args *SubmitObservationArgs, reply *SubmitObservationReply) error {
	feedID, err := ids.FromString(args.FeedID)
	if err != nil {
		return err
	}

	obs := &Observation{
		FeedID:    feedID,
		Value:     args.Value,
		Signature: args.Signature,
	}

	if err := s.vm.SubmitObservation(obs); err != nil {
		return err
	}

	reply.Success = true
	return nil
}

// GetAttestationArgs are arguments for GetAttestation
type GetAttestationArgs struct {
	FeedID string `json:"feedId"`
	Epoch  uint64 `json:"epoch"`
}

// GetAttestationReply is the reply for GetAttestation
type GetAttestationReply struct {
	Attestation []byte `json:"attestation"`
}

// GetAttestation returns an oracle attestation
func (s *Service) GetAttestation(r *http.Request, args *GetAttestationArgs, reply *GetAttestationReply) error {
	feedID, err := ids.FromString(args.FeedID)
	if err != nil {
		return err
	}

	att, err := s.vm.CreateAttestation(feedID, args.Epoch)
	if err != nil {
		return err
	}

	reply.Attestation = att.Bytes()
	return nil
}

// HealthArgs are arguments for Health
type HealthArgs struct{}

// HealthReply is the reply for Health
type HealthReply struct {
	Healthy bool `json:"healthy"`
	Feeds   int  `json:"feeds"`
}

// Health returns health status
func (s *Service) Health(r *http.Request, args *HealthArgs, reply *HealthReply) error {
	health, err := s.vm.HealthCheck(context.Background())
	if err != nil {
		return err
	}

	reply.Healthy = health.Healthy
	reply.Feeds = len(s.vm.feeds)
	return nil
}
