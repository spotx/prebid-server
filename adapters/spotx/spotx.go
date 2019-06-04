package spotx

import (
	"context"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

type SpotxAdapter struct {
	http *adapters.HTTPAdapter
	URI string
	
}

// Name must be identical to the BidderName.
func (a *SpotxAdapter) Name() string {
	return openrtb_ext.BidderSpotx.String()
}

// Determines whether this adapter should get callouts if there is not a synched user ID.
func (a *SpotxAdapter) SkipNoCookies() bool {
	return true
}

// Call produces bids which should be considered, given the auction params.
//
// In practice, implementations almost always make one call to an external server here.
// However, that is not a requirement for satisfying this interface.
//
// An error here will cause all bids to be ignored. If the error was caused by bad user input,
// this should return a BadInputError. If it was caused by bad server behavior
// (e.g. 500, unexpected response format, etc), this should return a BadServerResponseError.
func (a *SpotxAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {

}


// MakeRequests makes the HTTP requests which should be made to fetch bids.
//
// Bidder implementations can assume that the incoming BidRequest has:
//
//   1. Only {Imp.Type, Platform} combinations which are valid, as defined by the static/bidder-info.{bidder}.yaml file.
//   2. Imp.Ext of the form {"bidder": params}, where "params" has been validated against the static/bidder-params/{bidder}.json JSON Schema.
//
// nil return values are acceptable, but nil elements *inside* those slices are not.
//
// The errors should contain a list of errors which explain why this bidder's bids will be
// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
//
// If the error is caused by bad user input, return an errortypes.BadInput.
func (a *SpotxAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	
}

// MakeBids unpacks the server's response into Bids.
//
// The bids can be nil (for no bids), but should not contain nil elements.
//
// The errors should contain a list of errors which explain why this bidder's bids will be
// "subpar" in some way. For example: the server response didn't have the expected format.
//
// If the error was caused by bad user input, return a errortypes.BadInput.
// If the error was caused by a bad server response, return a errortypes.BadServerResponse
func (a *SpotxAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	
}

func NewAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *SpotxAdapter {
	return NewBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

func NewBidder(client *http.Client, endpoint string) *SpotxAdapter {
	a := &adapters.HTTPAdapter{Client: client}


	return &SpotxAdapter{
		http:           a,
		URI:            endpoint,
	}
}
