package arrow

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

// Limits is the /user/limits payload: segment allocations plus account margin summary.
// Field values are strings in live API responses (e.g. "usableMargin", "netPnl").
type Limits struct {
	Data struct {
		Allocations []map[string]any `json:"allocations"`
		Margin      map[string]any     `json:"margin"`
	} `json:"data"`
	Status string `json:"status"`
}

// GetLimits fetches trading limits and margin for the authenticated user.
func (c *Client) GetLimits() (*Limits, error) {
	endpoint := "/user/limits"

	resp, err := c.request(endpoint, "GET", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch trading limits")
		return nil, err
	}

	var result Limits
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse trading limits response")
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("failed to retrieve trading limits")
	}

	c.debugf("Trading limits retrieved successfully", nil)
	return &result, nil
}
