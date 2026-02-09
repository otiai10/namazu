// Package p2pquake provides a WebSocket client for P2P地震情報 API.
//
// The client connects to the real-time earthquake information API and receives
// JMA (Japan Meteorological Agency) earthquake data via WebSocket.
//
// Features:
//   - Automatic reconnection every 9 minutes (before 10-minute forced disconnect)
//   - Exponential backoff retry on connection errors
//   - Message deduplication using LRU cache (keeps last 1000 IDs)
//   - Filters for code 551 (JMAQuake) events only
//   - Thread-safe operations with mutex protection
//
// Example usage:
//
//	client := p2pquake.NewClient("wss://api.p2pquake.net/v2/ws")
//	ctx := context.Background()
//
//	if err := client.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	for event := range client.Events() {
//	    quake := event.(*p2pquake.JMAQuake)
//	    fmt.Printf("Earthquake: %s, Severity: %d\n",
//	        quake.Earthquake.Hypocenter.Name,
//	        event.GetSeverity())
//	}
package p2pquake
