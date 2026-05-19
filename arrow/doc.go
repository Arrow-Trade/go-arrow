// Package arrow provides a Go client for Arrow Trade APIs.
//
// It includes:
//   - authentication helpers (request-token auth and AutoLogin with TOTP)
//   - order/trade endpoints (place, modify, cancel, books)
//   - user/portfolio endpoints (profile, holdings, positions, limits)
//   - market endpoints (quotes, option chain, instruments, candles)
//   - WebSocket streams (order updates; token market data on ds.arrow.trade; HFT market data on socket.arrow.trade — use NewStreams / NewStreamsWithHFT / NewStreamsOrderOnly as needed)
//
// Basic usage:
//
//	client := arrow.NewClient(appID, appSecret)
//	client.SetDebug(true) // optional
//	err := client.AutoLogin(userID, password, totpSecret)
//
// API docs:
//   - https://docs.arrow.trade/
package arrow
