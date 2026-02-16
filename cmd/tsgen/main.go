package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/tkrajina/typescriptify-golang-structs/typescriptify"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/channel_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_channel_role"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_target_type"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_type"
	"github.com/wickedv43/TAM-backend/internal/common/constants/transaction_status"
)

func main() {
	outputDir := "./shared/types"
	outputFile := filepath.Join(outputDir, "types.ts")

	_ = os.MkdirAll(outputDir, 0755)

	converter := typescriptify.New().
		WithInterface(true).
		WithBackupDir("").
		ManageType(time.Time{}, typescriptify.TypeOptions{
			TSType:      "string",
			TSTransform: "__VALUE__",
		}).
		ManageType(uuid.UUID{}, typescriptify.TypeOptions{
			TSType:      "string",
			TSTransform: "__VALUE__",
		}).
		ManageType(int64(0), typescriptify.TypeOptions{
			TSType:      "string",
			TSTransform: "__VALUE__.toString()",
		}).

		// --- Enums ---
		AddEnum(deal_type.DealTypeValues()).
		AddEnum(deal_status.DealStatusValues()).
		AddEnum(deal_target_type.DealTargetTypeValues()).
		AddEnum(customer_status.CustomerStatusValues()).
		AddEnum(channel_status.ChannelStatusValues()).
		AddEnum(customer_channel_role.CustomerChannelRoleValues()).
		AddEnum(transaction_status.TransactionStatusValues()).

		// --- API Models ---
		// Auth
		Add(common.AuthRequest{}).
		Add(common.AuthResponse{}).
		// Brief
		Add(common.BriefResponse{}).
		// Channel
		Add(common.AddChannelRequest{}).
		Add(common.ChannelResponse{}).
		Add(common.ChannelResponseWithoutStats{}).
		Add(common.GetChannelRequest{}).
		Add(common.GetChannelResponse{}).
		Add(common.ChannelPrices{}).
		Add(common.ProvideChannelDataRequest{}).
		Add(common.PageOptions{}).
		Add(common.ListChannelsRequest{}).
		Add(common.ListChannelsResponse{}).
		Add(common.ChannelStats{}).
		Add(common.ChannelGraphs{}).
		Add(common.Graph{}).
		Add(common.LanguageStat{}).
		Add(common.PostsStats{}).
		Add(common.BroadcastStats{}).
		Add(common.BroadcastValue{}).
		Add(common.BroadcastPeriod{}).
		// Customer
		Add(common.UpdateMeRequest{}).
		Add(common.CustomerResponse{}).
		Add(common.WithdrawRequest{}).
		Add(common.WithdrawResponse{}).
		// Deal
		Add(common.DealMessageRequest{}).
		Add(common.SendDealDataRequest{}).
		Add(common.OfferDealRequest{}).
		Add(common.MakeDealRequest{}).
		Add(common.ApproveAndProposeTimeRequest{}).
		Add(common.UpdateDealRequest{}).
		Add(common.DealResponse{}).
		Add(common.DealRequest{}).
		Add(common.GetMyDealsResponse{}).
		Add(common.ListDealsRequest{}).
		Add(common.ListDealsResponse{}).
		Add(common.DealWithChannelResponse{}).
		Add(common.ListDealsWithChannelResponse{})

	err := converter.ConvertToFile(outputFile)
	if err != nil {
		panic(err.Error())
	}
}
