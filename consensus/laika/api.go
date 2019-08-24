package laika

// API exposes laika related methods for the RPC interface.
type API struct {
	laika *Laika
}

// GetHashrate returns the current hashrate
func (api *API) GetHashrate() uint64 {
	return uint64(api.laika.Hashrate())
}
