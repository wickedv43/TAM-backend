package common

import (
	"time"

	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
)

// DealStatusDurations defines how long a deal can remain in each status
// Terminal statuses (CANCELED, EXPIRED, TERMS_VIOLATED, COMPLETED) have 0 duration as they don't expire
var DealStatusDurations = map[deal_status.DealStatus]time.Duration{
	deal_status.DRAFT:                  24 * time.Hour,      // 1 day to complete draft
	deal_status.PENDING:                24 * time.Hour,      // 1 day to accept/reject
	deal_status.DISCUSSION:             48 * time.Hour,      // 2 days for discussion
	deal_status.IN_PROGRESS:            7 * 24 * time.Hour,  // 7 days to complete work
	deal_status.AWAITING_APPROVAL:      48 * time.Hour,      // 2 days for approval
	deal_status.AWAITING_APPROVAL_TIME: 24 * time.Hour,      // 1 day
	deal_status.AWAITING_PUBLICATION:   30 * 24 * time.Hour, // 30 day to publish
	deal_status.PUBLISHED:              30 * 24 * time.Hour, // 30 days after publication
	deal_status.CANCELED:               0,                   // Terminal status
	deal_status.EXPIRED:                0,                   // Terminal status
	deal_status.TERMS_VIOLATED:         0,                   // Terminal status
	deal_status.COMPLETED:              0,                   // Terminal status
}

// GetDealStatusDuration returns the duration for a given deal status
// Returns 0 if status has no defined duration
func GetDealStatusDuration(status deal_status.DealStatus) time.Duration {
	return DealStatusDurations[status]
}
