// Copyright (c) 2025 Index Exchange Inc.
//
// This file is part of the Agentic RTB Framework reference implementation.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package handlers implements mutation handlers for different ARTF intents
package handlers

import (
	"context"
	"log"

	pb "github.com/iabtechlab/agentic-rtb-framework/pkg/pb/artf"
	openrtb "github.com/iabtechlab/agentic-rtb-framework/pkg/pb/openrtb"
)

// MutationHandlers contains all registered mutation handlers
type MutationHandlers struct {
	// Add any configuration or dependencies here
}

// NewMutationHandlers creates a new handlers instance
func NewMutationHandlers() *MutationHandlers {
	return &MutationHandlers{}
}

// IsIntentApplicable checks if an intent is in the applicable intents list.
// If applicableIntents is nil or empty, all intents are applicable.
func IsIntentApplicable(intent pb.Intent, applicableIntents []pb.Intent) bool {
	if len(applicableIntents) == 0 {
		return true
	}
	for _, ai := range applicableIntents {
		if ai == intent {
			return true
		}
	}
	return false
}

// ProcessSegments analyzes the bid request and returns segment activation mutations.
// Respects applicableIntents filtering - if empty, all intents are applicable.
func (h *MutationHandlers) ProcessSegments(ctx context.Context, req *openrtb.BidRequest, applicableIntents []pb.Intent) ([]*pb.Mutation, error) {
	if req == nil {
		return nil, nil
	}

	// Check if ACTIVATE_SEGMENTS intent is applicable
	if !IsIntentApplicable(pb.Intent_ACTIVATE_SEGMENTS, applicableIntents) {
		return nil, nil
	}

	var mutations []*pb.Mutation

	// Example: Activate segments based on user data
	// In a real implementation, this would call your ML model or segment service
	user := req.GetUser()
	if user != nil {
		// Example segment activation based on user attributes
		segments := determineUserSegments(user)
		if len(segments) > 0 {
			mutation := &pb.Mutation{
				Intent: pb.Intent_ACTIVATE_SEGMENTS.Enum(),
				Op:     pb.Operation_OPERATION_ADD.Enum(),
				Path:   stringPtr("/user/data/segment"),
				Value: &pb.Mutation_Ids{
					Ids: &pb.IDsPayload{
						Id: segments,
					},
				},
			}
			mutations = append(mutations, mutation)
			log.Printf("Activating %d segments for user", len(segments))
		}
	}

	return mutations, nil
}

// ProcessDeals analyzes the bid request and returns deal-related mutations.
// Respects applicableIntents filtering for ACTIVATE_DEALS, SUPPRESS_DEALS, and ADJUST_DEAL_FLOOR.
func (h *MutationHandlers) ProcessDeals(ctx context.Context, req *openrtb.BidRequest, applicableIntents []pb.Intent) ([]*pb.Mutation, error) {
	if req == nil {
		return nil, nil
	}

	var mutations []*pb.Mutation

	activateDealsApplicable := IsIntentApplicable(pb.Intent_ACTIVATE_DEALS, applicableIntents)
	adjustFloorApplicable := IsIntentApplicable(pb.Intent_ADJUST_DEAL_FLOOR, applicableIntents)

	// Process each impression
	for _, imp := range req.GetImp() {
		impID := imp.GetId()

		// Example: Activate deals based on impression characteristics
		if activateDealsApplicable {
			dealsToActivate := determineDealActivations(imp)
			if len(dealsToActivate) > 0 {
				mutation := &pb.Mutation{
					Intent: pb.Intent_ACTIVATE_DEALS.Enum(),
					Op:     pb.Operation_OPERATION_ADD.Enum(),
					Path:   stringPtr("/imp/" + impID),
					Value: &pb.Mutation_Ids{
						Ids: &pb.IDsPayload{
							Id: dealsToActivate,
						},
					},
				}
				mutations = append(mutations, mutation)
				log.Printf("Activating %d deals for impression %s", len(dealsToActivate), impID)
			}
		}

		// Example: Adjust deal floors
		if adjustFloorApplicable {
			if floorAdjustment := calculateDealFloorAdjustment(imp); floorAdjustment != nil {
				mutation := &pb.Mutation{
					Intent: pb.Intent_ADJUST_DEAL_FLOOR.Enum(),
					Op:     pb.Operation_OPERATION_REPLACE.Enum(),
					Path:   stringPtr("/imp/" + impID + "/pmp/deals"),
					Value: &pb.Mutation_AdjustDeal{
						AdjustDeal: floorAdjustment,
					},
				}
				mutations = append(mutations, mutation)
			}
		}
	}

	return mutations, nil
}

// ProcessBidShading analyzes bid responses and returns bid adjustment mutations.
// Respects applicableIntents filtering for BID_SHADE intent.
func (h *MutationHandlers) ProcessBidShading(ctx context.Context, req *openrtb.BidRequest, resp *openrtb.BidResponse, applicableIntents []pb.Intent) ([]*pb.Mutation, error) {
	if req == nil || resp == nil {
		return nil, nil
	}

	// Check if BID_SHADE intent is applicable
	if !IsIntentApplicable(pb.Intent_BID_SHADE, applicableIntents) {
		return nil, nil
	}

	var mutations []*pb.Mutation

	// Process each seatbid
	for _, seatbid := range resp.GetSeatbid() {
		for _, bid := range seatbid.GetBid() {
			// Calculate optimal bid price using bid shading logic
			shadedPrice := calculateShadedBidPrice(req, bid)
			if shadedPrice != nil && *shadedPrice != bid.GetPrice() {
				mutation := &pb.Mutation{
					Intent: pb.Intent_BID_SHADE.Enum(),
					Op:     pb.Operation_OPERATION_REPLACE.Enum(),
					Path:   stringPtr("/seatbid/" + seatbid.GetSeat() + "/bid/" + bid.GetId()),
					Value: &pb.Mutation_AdjustBid{
						AdjustBid: &pb.AdjustBidPayload{
							Price: shadedPrice,
						},
					},
				}
				mutations = append(mutations, mutation)
				log.Printf("Bid shading: adjusted bid %s from %.4f to %.4f",
					bid.GetId(), bid.GetPrice(), *shadedPrice)
			}
		}
	}

	return mutations, nil
}

// ProcessContentData analyzes bid request and returns content ID mutations.
// Respects applicableIntents filtering for ADD_CIDS intent.
// NOTE: ADD_CIDS intent requires protobuf regeneration to be fully supported.
// This is a placeholder that returns nil until protos are regenerated.
func (h *MutationHandlers) ProcessContentData(ctx context.Context, req *openrtb.BidRequest, applicableIntents []pb.Intent) ([]*pb.Mutation, error) {
	// ADD_CIDS (Intent 8) is defined in proto but not yet in generated Go code.
	// After running `make bindings`, this can be fully implemented.
	// For now, return nil to allow compilation.
	return nil, nil
}

// determineUserSegments analyzes user data and returns applicable segment IDs
func determineUserSegments(user *openrtb.BidRequest_User) []string {
	var segments []string

	// Example logic - in production this would use ML models or segment services
	// Check existing user data for segment hints
	for _, data := range user.GetData() {
		for _, seg := range data.GetSegment() {
			if seg.GetId() != "" {
				// Re-activate or enrich existing segments
				segments = append(segments, seg.GetId())
			}
		}
	}

	// Example: Add demographic segments based on user attributes
	if user.GetYob() > 0 {
		age := 2024 - int(user.GetYob())
		if age >= 18 && age <= 24 {
			segments = append(segments, "demo-18-24")
		} else if age >= 25 && age <= 34 {
			segments = append(segments, "demo-25-34")
		} else if age >= 35 && age <= 44 {
			segments = append(segments, "demo-35-44")
		}
	}

	return segments
}

// determineDealActivations returns deal IDs to activate for an impression
func determineDealActivations(imp *openrtb.BidRequest_Imp) []string {
	var deals []string

	// Example logic - check impression characteristics
	bidfloor := imp.GetBidfloor()
	if bidfloor >= 5.0 {
		deals = append(deals, "premium-deal-001")
	}

	// Check if video impression for video-specific deals
	if imp.GetVideo() != nil {
		deals = append(deals, "video-deal-001")
	}

	// Check if native impression
	if imp.GetNative() != nil {
		deals = append(deals, "native-deal-001")
	}

	return deals
}

// calculateDealFloorAdjustment calculates floor adjustments for deals
func calculateDealFloorAdjustment(imp *openrtb.BidRequest_Imp) *pb.AdjustDealPayload {
	pmp := imp.GetPmp()
	if pmp == nil || len(pmp.GetDeals()) == 0 {
		return nil
	}

	// Example: Adjust floor based on time of day, inventory quality, etc.
	// In production, this would use sophisticated pricing models
	currentFloor := imp.GetBidfloor()
	if currentFloor > 0 {
		// Example: 10% floor adjustment
		adjustedFloor := currentFloor * 1.1
		return &pb.AdjustDealPayload{
			Bidfloor: &adjustedFloor,
		}
	}

	return nil
}

// calculateShadedBidPrice calculates the optimal shaded bid price
func calculateShadedBidPrice(req *openrtb.BidRequest, bid *openrtb.BidResponse_Bid) *float64 {
	originalPrice := bid.GetPrice()
	if originalPrice <= 0 {
		return nil
	}

	// Example bid shading logic
	// In production, this would use ML models trained on win rate data
	// to find the optimal price point

	// Simple example: shade by 5-15% based on bid floor
	var shadePercent float64
	for _, imp := range req.GetImp() {
		if imp.GetId() == bid.GetImpid() {
			bidfloor := imp.GetBidfloor()
			if bidfloor > 0 {
				// More aggressive shading when far above floor
				margin := originalPrice - bidfloor
				if margin > bidfloor*0.5 {
					shadePercent = 0.15 // 15% shade
				} else if margin > bidfloor*0.2 {
					shadePercent = 0.10 // 10% shade
				} else {
					shadePercent = 0.05 // 5% shade
				}
			}
			break
		}
	}

	if shadePercent > 0 {
		shadedPrice := originalPrice * (1 - shadePercent)
		return &shadedPrice
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}
