package spotx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)
// 79391
var (
	ortbVersionMap = map[string]string{
		"" : "2.3",
		"2.3": "2.3",
		"2.5": "2.5",
	}
	skipTrue = int8(1)
)

func parseParam(paramsInput json.RawMessage) (config openrtb_ext.ExtImpSpotX, err error) {
	if err = json.Unmarshal(paramsInput, &config); err != nil {
		return
	}

	if config.ChannelID == 0 {
		return config, errors.New("invalid channel id")
	}

	if v, ok := ortbVersionMap[config.ORTBVersion]; ok {
		config.ORTBVersion = v
	} else  {
		return config, errors.New("unsupported Open RTB version")
	}

	return
}

// getMediaTypeForImp figures out which media type this bid is for.
func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Audio != nil {
				mediaType = openrtb_ext.BidTypeAudio
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}

func translateProtocols(protos []int8) []openrtb.Protocol {
	length := len(protos)
	result := make([]openrtb.Protocol, length, length)

	for i, p := range protos {
		result[i] = openrtb.Protocol(p)
	}

	return result
}

func getPlaybackMethods(methods ...int8) []openrtb.PlaybackMethod {
	length := len(methods)
	result := make([]openrtb.PlaybackMethod, length, length)

	for i, p := range methods {
		result[i] = openrtb.PlaybackMethod(p)
	}

	return result
}

func kvpToExt(items []openrtb_ext.ExtImpSpotXKeyVal) json.RawMessage {
	result := map[string][]string{

	}
	for _, kvp := range items {
		result[kvp.Key] = kvp.Values
	}
	data, err := json.Marshal(map[string]map[string][]string{"custom": result})
	if err != nil {
		return nil
	}
	return data
}

type SpotxAdapter struct {
	http *adapters.HTTPAdapter
	URI string
	
}

// Name must be identical to the BidderName.
func (a *SpotxAdapter) Name() string {
	return string(openrtb_ext.BidderSpotx)
}

// Determines whether this adapter should get callouts if there is not a synched user ID.
func (a *SpotxAdapter) SkipNoCookies() bool {
	return false
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
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	ortbReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), supportedMediaTypes)

	if err != nil {
		return nil, err
	}

	var param openrtb_ext.ExtImpSpotX

	for i, unit := range bidder.AdUnits {
		param, err = parseParam(unit.Params)
		if err != nil {
			return nil, err
		}
		ortbReq.BAdv = param.BlackList.Advertiser
		ortbReq.BCat = param.BlackList.Category

		ortbReq.WLang = param.WhiteList.Language

		if len(param.WhiteList.Seat) > 0 {
			ortbReq.WSeat = param.WhiteList.Seat
		} else if len(param.BlackList.Seat) > 0 {
			ortbReq.BSeat = param.BlackList.Seat
		}

		ortbReq.Imp[i].ID = unit.BidID
		ortbReq.Imp[i].BidFloor = param.PriceFloor
		ortbReq.Imp[i].BidFloorCur = param.Currency

		if param.Boxing.Valid && param.Boxing.Bool {
			ortbReq.Imp[1].Video.BoxingAllowed = int8(1)
		}

		if len(param.KVP) > 0 {
			ortbReq.Imp[i].Ext = kvpToExt(param.KVP)
		}
	}
	uri := fmt.Sprintf("%s/%s/%d", a.URI, param.ORTBVersion, param.ChannelID)

	reqJSON, err := json.Marshal(ortbReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: uri,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	resp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = resp.StatusCode

	if resp.StatusCode == 204 {
		return nil, nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	responseBody := string(body)

	if resp.StatusCode == http.StatusBadRequest {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", resp.StatusCode, responseBody),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", resp.StatusCode, responseBody),
		}
	}

	if req.IsDebug {
		debug.ResponseBody = responseBody
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, err
	}

	bids := make(pbs.PBSBidSlice, 0)
	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
			}

			mediaType := getMediaTypeForImp(bid.ImpID, ortbReq.Imp)
			pbid.CreativeMediaType = string(mediaType)

			bids = append(bids, &pbid)
		}
	}

	return bids, nil
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

	return nil, nil
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

	return nil, nil
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
